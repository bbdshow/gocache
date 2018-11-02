package gocache

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// 利用 sync.map 达到读取性能相对更高，sync.map 并不太适应大量写入的缓存操作, 且因为计数使用了 LoadOrStrore
// sync.map 在空间上并不占优势。如果存在频繁写入建议使用 RwMutex map  github.com/patrickmn/go-cache

// 在 4 核 机器上 锁竞争不明显， 所以 RwMutex map 在性能上更占优势，但是当 cpu 核数 往上时， 锁竞争变大， sync.map 的优势就体现出来了。
// 性能测试 引用 https://medium.com/@deckarep/the-new-kid-in-town-gos-sync-map-de24a6bf7c2c

// Options 配置选项说明
type Options struct {
	// 超过容量限制自动清除 keys 方式
	OverSizeClearMode cleanMode
	// keys 容量限制，通过此预估内存使用量
	MaxSize int64
	// 定时清除过期的 key, 未设置，则不清除
	CleanInterval time.Duration

	// 自动保存缓存内容到文件
	AutoSave bool
	// 保存的内容类型
	SaveType saveMode
	// 保存到文件名， 绝对路径 eg: /tmp/cache.back
	Filename string
}

type (
	saveMode  int
	cleanMode int
)

// 超过 maxSize 清除策略
const (
	// 超出容量报错
	NoEvictionMode cleanMode = iota
	// 随机删除 keys 的某一 key
	AllKeysRandomMode
	// 随机删除 设置过期时间中keys de某一 key
	VolatileRandomMode
)

// 存储规则
const (
	// 保存所有的 keys value
	SaveAllKeysMode saveMode = iota
	// 只保存设置过期时间的 keys value
	SaveExpireKeysMode
	// 只保存永久 keys value
	SaveNoExpireKeysMode
)

const (
	fileMode0666 = os.FileMode(0666)
)

// Cache Cache
type Cache struct {
	data    sync.Map
	expired sync.Map

	autoSave bool
	saveType saveMode
	filename string

	cleanInterval time.Duration
	hitCount      int64

	overSizeClearMode cleanMode
	// map存储的容量 利用此容量 估算限制 cache 使用内存的大小， 当不设置此值，可以忽略 set 的 error
	maxSize int64
	size    int64

	stop chan bool
}

type ivalue struct {
	Expire int64
	Value  interface{}
}

// Set 设置 key value 用 LoadOrStore 是方便计数 考虑缓存是 读多写少才用 LoadOrStore
func (c *Cache) Set(key interface{}, value interface{}) error {
	if c.maxSize > 0 && atomic.LoadInt64(&c.size) >= c.maxSize {
		if err := c.overSize(); err != nil {
			return err
		}
		if atomic.LoadInt64(&c.size) >= c.maxSize {
			return errors.New("keys over cache size")
		}
	}

	v := ivalue{Value: value}
	loadV, load := c.data.LoadOrStore(key, v)
	if !load {
		atomic.AddInt64(&c.size, 1)
	} else {
		// 从 ttl key 变成永久 key
		vold := loadV.(ivalue)
		if vold.Expire > 0 {
			c.expired.Delete(key)
		}

		c.data.Store(key, value)
	}

	return nil
}

// SetExpire 设置 key value 用 LoadOrStore 是方便计数 考虑缓存是 读多写少才用 LoadOrStore
func (c *Cache) SetExpire(key interface{}, value interface{}, expire time.Duration) error {
	if c.maxSize > 0 && atomic.LoadInt64(&c.size) >= c.maxSize {
		if err := c.overSize(); err != nil {
			return err
		}
		if atomic.LoadInt64(&c.size) >= c.maxSize {
			return errors.New("keys over cache size")
		}
	}

	v := ivalue{Value: value, Expire: time.Now().Add(expire).UnixNano()}
	_, load := c.data.LoadOrStore(key, v)
	if !load {
		atomic.AddInt64(&c.size, 1)
	} else {
		c.data.Store(key, v)
	}
	c.expired.Store(key, v.Expire)

	return nil
}

// Get 获取 value 同时检测是否 过期
func (c *Cache) Get(key interface{}) (value interface{}, ok bool) {
	v, ok := c.data.Load(key)
	if !ok {
		return nil, false
	}

	vv := v.(ivalue)
	if vv.expired() {

		c.data.Delete(key)
		atomic.AddInt64(&c.size, -1)

		if vv.Expire > 0 {
			c.expired.Delete(key)
		}
		return value, false
	}

	return vv.Value, true
}

// GetWithExpire 返回 key 剩余时间
func (c *Cache) GetWithExpire(key interface{}) (interface{}, time.Duration, bool) {
	v, ok := c.data.Load(key)
	if !ok {
		return nil, 0, false
	}

	vv := v.(ivalue)
	if vv.expired() {
		c.data.Delete(key)
		atomic.AddInt64(&c.size, -1)

		if vv.Expire > 0 {
			c.expired.Delete(key)
		}
		return nil, 0, false
	}
	return vv.Value, time.Unix(0, vv.Expire).Sub(time.Now()), true
}

// Range 遍历缓存  f return false = range break
func (c *Cache) Range(f func(key, value interface{}) bool) {
	c.data.Range(func(k, v interface{}) bool {
		vv := v.(ivalue)
		return f(k, vv.Value)
	})
}

// Delete Del
func (c *Cache) Delete(key interface{}) {
	_, ok := c.data.Load(key)
	if ok {
		c.data.Delete(key)
		atomic.AddInt64(&c.size, -1)
	}
}

// Add 不存在添加, 存在则报错
func (c *Cache) Add(key interface{}, value interface{}, expire time.Duration) error {
	if c.maxSize > 0 && atomic.LoadInt64(&c.size) >= c.maxSize {
		if err := c.overSize(); err != nil {
			return err
		}
	}

	v := ivalue{Value: value}
	if expire.Nanoseconds() > 0 {
		v.Expire = time.Now().Add(expire).UnixNano()
	}
	_, load := c.data.LoadOrStore(key, v)
	if !load { // 不存在
		atomic.AddInt64(&c.size, 1)
		if expire.Nanoseconds() > 0 {
			c.expired.Store(key, v.Expire)
		}
		return nil
	}

	return errors.New("key existing")
}

// Size 数据长度
func (c *Cache) Size() int64 {
	return atomic.LoadInt64(&c.size)
}

// Flush 清除全部缓存
func (c *Cache) Flush() {
	c.data = sync.Map{}
	c.expired = sync.Map{}
	c.size = 0
}

// Close 保存缓存到磁盘和释放资源
func (c *Cache) Close() error {
	if c.autoSave {
		if err := c.SaveDisk(c.filename, c.saveType); err != nil {
			return err
		}
	}
	c.Flush()
	c.stop <- true

	return nil
}

func (c *Cache) overSize() error {
	switch c.overSizeClearMode {
	case NoEvictionMode:
		return errors.New("keys over cache size")
	case VolatileRandomMode:
		c.expired.Range(func(k, v interface{}) bool {
			c.Delete(k)
			c.expired.Delete(k)
			// 只删除一个
			return false
		})

	case AllKeysRandomMode:
		c.data.Range(func(k, v interface{}) bool {
			c.Delete(k)
			c.expired.Delete(k)
			return false
		})
	}

	return nil
}

func (c *Cache) deleteExpire() {
	c.expired.Range(func(k, v interface{}) bool {
		if expired(v.(int64)) {
			// 自动删除的时候， 判断当前 key 是否为 ttl
			v, ok := c.data.Load(k)
			if ok {
				vv := v.(ivalue)
				if vv.Expire > 0 {
					c.Delete(k)
				}
			}
			c.expired.Delete(k)
		}
		return true
	})
}

func (c *Cache) expireClean() {
	ticker := time.NewTicker(c.cleanInterval)
	for {
		select {
		case <-ticker.C:
			c.deleteExpire()
		case <-c.stop:
			ticker.Stop()
			return
		}
	}
}

// LoadDisk 从磁盘加载缓存到内存
func (c *Cache) LoadDisk(filename string) error {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	if !filenameExists(filename) {
		return nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	idata := make(map[interface{}]ivalue, c.maxSize)
	if err := dec.Decode(&idata); err != nil {
		return err
	}

	for k, v := range idata {
		if !v.expired() {
			c.data.Store(k, v)
			atomic.AddInt64(&c.size, 1)
		}
	}

	return nil
}

// SaveDisk 从内存把缓存保存在磁盘
func (c *Cache) SaveDisk(filename string, mode saveMode) error {

	filename, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	if filenameExists(filename) {
		if err := os.Remove(filename); err != nil {
			return err
		}
	} else {
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, fileMode0666); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(c.filename, os.O_RDWR|os.O_CREATE, fileMode0666)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := gob.NewEncoder(file)

	defer func() {
		if x := recover(); x != nil {
			err = errors.New("Error registering ivalue types with Gob library")
		}
	}()

	idata := make(map[interface{}]ivalue, atomic.LoadInt64(&c.size))
	switch mode {
	case SaveNoExpireKeysMode:
		c.expired.Range(func(k, v interface{}) bool {
			c.data.Delete(k)
			return true
		})

		c.data.Range(func(k, v interface{}) bool {
			gob.Register(v.(ivalue))
			idata[k] = v.(ivalue)
			return true
		})
	case SaveExpireKeysMode:
		c.data.Range(func(k, v interface{}) bool {
			vv := v.(ivalue)
			if vv.Expire > 0 && !vv.expired() {
				gob.Register(vv)
				idata[k] = vv
			}
			return true
		})
	case SaveAllKeysMode:
		c.data.Range(func(k, v interface{}) bool {
			gob.Register(v.(ivalue))
			idata[k] = v.(ivalue)
			return true
		})
	}

	return enc.Encode(&idata)
}

func (i *ivalue) expired() bool {
	return expired(i.Expire)
}

func expired(expire int64) bool {
	if expire <= 0 {
		return false
	}
	return time.Now().UnixNano() > expire
}

func filenameExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}

// NewCache NewCache
func NewCache(opt Options) (*Cache, error) {
	c := Cache{
		data:              sync.Map{},
		expired:           sync.Map{},
		overSizeClearMode: opt.OverSizeClearMode,
		maxSize:           opt.MaxSize,
		cleanInterval:     opt.CleanInterval,

		autoSave: opt.AutoSave,
		saveType: opt.SaveType,
		filename: opt.Filename,

		stop: make(chan bool, 1),
	}

	if c.autoSave {
		if err := c.LoadDisk(c.filename); err != nil {
			return nil, err
		}
	}

	if c.cleanInterval.Nanoseconds() > 0 {
		go c.expireClean()
	}

	return &c, nil
}
