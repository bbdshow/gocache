package gocache

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Options Options
type Options struct {
	MaxSize       int64
	CleanInterval time.Duration

	SaveDisk bool
	SaveType int
	Filename string
}

// 超过 maxSize 清除策略
const (
	NoEviction int = iota
	AllKeysLru
	AllKeysRandom
	VolatileRandom
)

// 存储规则
const (
	SaveAllKeys int = iota
	SaveExpireKeys
	SaveNoExpireKeys
)
const (
	fileMode0666 = os.FileMode(0666)
)

// Cache Cache
type Cache struct {
	data    sync.Map
	expired sync.Map

	enableSaveDisk bool
	saveType       int
	filename       string

	cleanInterval time.Duration
	hitCount      int64
	maxSize       int64
	size          int64

	stop chan bool
}

type ivalue struct {
	Expire int64
	Value  interface{}
}
type iexpire struct {
}

// Set 设置 key value 用 LoadOrStore 是方便计数 考虑缓存是 读多写少才用 LoadOrStore
func (c *Cache) Set(key interface{}, value interface{}) {
	if c.maxSize > 0 && c.size >= c.maxSize {

	}
	v := ivalue{Value: value}
	_, load := c.data.LoadOrStore(key, v)
	if !load {
		atomic.AddInt64(&c.size, 1)
	}
}

// SetExpire 设置 key value 用 LoadOrStore 是方便计数 考虑缓存是 读多写少才用 LoadOrStore
func (c *Cache) SetExpire(key interface{}, value interface{}, expire time.Duration) {
	v := ivalue{Value: value, Expire: time.Now().Add(expire).UnixNano()}
	_, load := c.data.LoadOrStore(key, v)
	if !load {
		atomic.AddInt64(&c.size, 1)
	}
	c.expired.Store(key, v.Expire)
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

// Del Del
func (c *Cache) Del(key interface{}) {
	_, ok := c.data.Load(key)
	if ok {
		c.data.Delete(key)
		atomic.AddInt64(&c.size, -1)
	}
}

// Add 不存在添加
func (c *Cache) Add(key interface{}, value interface{}) error {
	return nil
}

func (c *Cache) AddExpire(key interface{}, value interface{}, expire time.Duration) error {

	return nil
}

// Replace 存在修改
func (c *Cache) Replace(key interface{}, value interface{}) error {
	return nil
}

// ReplaceExpire 存在修改
func (c *Cache) ReplaceExpire(key interface{}, value interface{}, expire time.Duration) error {
	return nil
}

// Size 数据长度
func (c *Cache) Size() int64 {
	return c.size
}

func (c *Cache) IncOrDecInt64(key string, i int64) int64 {
	var v int64

	return v
}
func (c *Cache) IncOrDecFloat64(key string, f float64) float64 {
	var v float64

	return v
}

// Flush 清除全部缓存
func (c *Cache) Flush() {
	c.data = sync.Map{}
	c.expired = sync.Map{}
}

// Close 保存缓存到磁盘和释放资源
func (c *Cache) Close() error {
	if c.enableSaveDisk {
		if err := c.saveDisk(); err != nil {
			return err
		}
	}

	c.Flush()

	c.stop <- true

	return nil
}

func (c *Cache) deleteExpire() {
	c.expired.Range(func(k, v interface{}) bool {
		if expired(v.(int64)) {
			c.Del(k)
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

func (c *Cache) loadDisk() error {
	filename, err := filepath.Abs(c.filename)
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

func (c *Cache) saveDisk() error {

	filename, err := filepath.Abs(c.filename)
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
			err = fmt.Errorf("Error registering ivalue types with Gob library")
		}
	}()

	idata := make(map[interface{}]ivalue, c.size)
	switch c.saveType {
	case SaveNoExpireKeys:
		c.expired.Range(func(k, v interface{}) bool {
			c.data.Delete(k)
			return true
		})

		c.data.Range(func(k, v interface{}) bool {
			gob.Register(v.(ivalue))
			idata[k] = v.(ivalue)
			return true
		})
	case SaveExpireKeys:
		c.data.Range(func(k, v interface{}) bool {
			vv := v.(ivalue)
			if vv.Expire > 0 && !vv.expired() {
				gob.Register(vv)
				idata[k] = vv
			}
			return true
		})
	case SaveAllKeys:
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
		data:          sync.Map{},
		expired:       sync.Map{},
		maxSize:       opt.MaxSize,
		cleanInterval: opt.CleanInterval,

		enableSaveDisk: opt.SaveDisk,
		saveType:       opt.SaveType,
		filename:       opt.Filename,

		stop: make(chan bool, 1),
	}

	if c.enableSaveDisk {
		if err := c.loadDisk(); err != nil {
			return nil, err
		}
	}

	if c.cleanInterval.Nanoseconds() > 0 {
		go c.expireClean()
	}

	return &c, nil
}
