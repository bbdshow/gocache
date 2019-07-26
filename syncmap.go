package gocache

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/*
	内存实现cache

Notice:
	使用 sync.Map 实现，最大限度优化读的性能，特别是在 cpu > 4核的情况下，因为没有锁的竞争，读取优势很明显
	但是内存消耗严重(约等于2倍)，空间换时间。而且写入性能没有 rwMutex map 性能好
*/

const (
	ErrProcessMode int = iota
	RandomCleanProcessMode
	ExpireKeyRandomCleanProcessMode

	defaultCapacity                = int64(-1)
	defaultOverCapacityProcessMode = ErrProcessMode
)

var (
	ErrOverCapacity = errors.New("over capacity")
)

type MemSyncMapCacheConfig struct {

	// 缓存容量， -1 - 不限制
	Capacity int64
	// 超出缓存容量，处理方式
	OverCapacityProcessMode int

	// 当退出时，保存数据到文件， 加载时，加载文件数据到内存
	WriteDisk bool
	// 保存文件位置
	Filename string
}

type MemSyncMapCache struct {
	config *MemSyncMapCacheConfig

	// 存储所有数据 key , value - mValue
	storeMap sync.Map

	// 当前存储容量
	currentCapacity int64

	disk *Disk

	once   sync.Once
	closed int32
}

func NewMemSyncMapCache() (*MemSyncMapCache, error) {
	return NewMemSyncMapCacheWithConfig(DefaultMemSyncMapConfig())
}

func NewMemSyncMapCacheWithConfig(config MemSyncMapCacheConfig) (*MemSyncMapCache, error) {
	if config.OverCapacityProcessMode <= -1 {
		config.OverCapacityProcessMode = -1
	}

	mem := MemSyncMapCache{
		config:          &config,
		storeMap:        sync.Map{},
		currentCapacity: 0,
		disk:            NewDisk(config.Filename),
		closed:          0,
	}

	if config.WriteDisk {
		if err := mem.readFromDisk(); err != nil {
			return nil, err
		}
	}

	return &mem, nil
}

func DefaultMemSyncMapConfig() MemSyncMapCacheConfig {
	return MemSyncMapCacheConfig{
		Capacity:                defaultCapacity,
		OverCapacityProcessMode: defaultOverCapacityProcessMode,
		WriteDisk:               false,
		Filename:                "",
	}
}

type mValue struct {
	Value  interface{}
	Expire int64 // 纳秒级别
}

func (v *mValue) expire(expire int64) {
	if expire <= -1 {
		v.Expire = -1
		return
	}

	v.Expire = time.Now().Add(time.Duration(expire) * time.Millisecond).UnixNano()
}

func (v mValue) validExpire() int64 {
	if v.Expire == -1 {
		return v.Expire
	}
	expire := time.Unix(0, v.Expire).Sub(time.Now()).Nanoseconds() / time.Millisecond.Nanoseconds()
	// 存在这种可能
	if expire < 0 {
		expire = 0
	}

	return expire
}

// true - expired
func (v mValue) isExpire() bool {
	if v.Expire == -1 {
		return false
	}
	return time.Now().UnixNano() >= v.Expire
}

func (mem *MemSyncMapCache) Get(key string) (value interface{}, ok bool) {
	value, _, ok = mem.GetWithExpire(key)
	return value, ok
}

func (mem *MemSyncMapCache) GetWithExpire(key string) (value interface{}, expire int64, ok bool) {
	if mem.isClosed() {
		return
	}

	v, ok := mem.getMValue(key)
	if !ok {
		return nil, 0, ok
	}

	if v.Expire == -1 {
		return v.Value, v.Expire, ok
	}

	return v.Value, v.validExpire(), ok
}

func (mem *MemSyncMapCache) Set(key string, value interface{}) error {
	return mem.SetWithExpire(key, value, -1)
}

// SetWithExpire  expire - 过期时间毫秒级别， -1 永久有效
func (mem *MemSyncMapCache) SetWithExpire(key string, value interface{}, expire int64) error {
	if mem.isClosed() {
		return nil
	}

	err := mem.overCapacity()
	if err != nil {
		return err
	}

	mem.setMValue(key, value, expire)

	return nil
}

func (mem *MemSyncMapCache) Delete(key string) {
	if mem.isClosed() {
		return
	}

	_, ok := mem.storeMap.Load(key)
	if !ok {
		return
	}

	mem.deleteMValue(key, true)
}

type CacheKeys struct {
	keys   []string
	length int64
}

func (k *CacheKeys) Size() int64 {
	return k.length
}

func (k *CacheKeys) Value() []string {
	return k.keys
}

func (mem *MemSyncMapCache) Keys(prefix string) Keys {
	keys := CacheKeys{
		keys: make([]string, 0, mem.currentCapacity),
	}
	if prefix == "" {
		mem.storeMap.Range(func(k, v interface{}) bool {
			if !v.(mValue).isExpire() {
				keys.keys = append(keys.keys, k.(string))
				keys.length++
			}
			return true
		})
	} else {
		mem.storeMap.Range(func(k, v interface{}) bool {
			if strings.HasPrefix(k.(string), prefix) {
				if !v.(mValue).isExpire() {
					keys.keys = append(keys.keys, k.(string))
					keys.length++
				}
			}
			return true
		})
	}

	return &keys
}

func (mem *MemSyncMapCache) Size() int64 {
	return atomic.LoadInt64(&mem.currentCapacity)
}

// FlushAll 清空所有数据，先关闭所有查询/写入
func (mem *MemSyncMapCache) FlushAll() error {
	atomic.StoreInt32(&mem.closed, 1)
	mem.storeMap = sync.Map{}
	mem.currentCapacity = 0
	atomic.StoreInt32(&mem.closed, 0)
	return nil
}

// Close 开启写入磁盘，则写入文件
func (mem *MemSyncMapCache) Close() error {
	atomic.StoreInt32(&mem.closed, 1)

	if mem.config.WriteDisk {
		err := mem.writeToDisk()
		if err != nil {
			return err
		}
	}

	mem.storeMap = sync.Map{}

	return nil
}

func (mem *MemSyncMapCache) getMValue(key string) (mValue, bool) {
	v, ok := mem.storeMap.Load(key)
	if !ok {
		return mValue{}, ok
	}

	mv := v.(mValue)
	if mv.isExpire() {
		mem.deleteMValue(key, true)
		return mValue{}, false
	}

	return mv, true
}

func (mem *MemSyncMapCache) setMValue(key string, value interface{}, expire int64) {
	if expire == 0 {
		return
	}

	mv := mValue{
		Value: value,
	}
	mv.expire(expire)

	// LoadOrStore 为了准确计数当前容量
	_, exists := mem.storeMap.LoadOrStore(key, mv)
	if !exists {
		atomic.AddInt64(&mem.currentCapacity, 1)
	}
}

func (mem *MemSyncMapCache) deleteMValue(key string, sub bool) {
	mem.storeMap.Delete(key)
	if sub {
		atomic.AddInt64(&mem.currentCapacity, -1)
	}
}

func (mem *MemSyncMapCache) isClosed() bool {
	closed := atomic.LoadInt32(&mem.closed)
	if closed == 1 {
		return true
	}
	return false
}

func (mem *MemSyncMapCache) overCapacity() error {
	if mem.config.Capacity == -1 {
		return nil
	}
	// 开区间，保持 mem.config.Capacity 容量
	if atomic.LoadInt64(&mem.currentCapacity) < mem.config.Capacity {
		return nil
	}

	// 超出容量
	switch mem.config.OverCapacityProcessMode {
	case RandomCleanProcessMode:
		mem.randomDeleteOne()
		return nil
	case ExpireKeyRandomCleanProcessMode:
		c := mem.randomDeleteExpireKey()
		if c > 0 {
			return nil
		}
		// 没有删除任何key
	}

	return ErrOverCapacity
}

// # -----------------------------------------------------

func (mem *MemSyncMapCache) randomDeleteOne() {
	mem.storeMap.Range(func(k, v interface{}) bool {
		mem.deleteMValue(k.(string), true)
		return false
	})
}

// 删除已经过期的 expire key， 如果一个就没有删除，则随机删除一个有效的 expire key
// 如果 int=0 则证明没有删除任何 key
func (mem *MemSyncMapCache) randomDeleteExpireKey() int {
	c := mem.expireClean()
	if c > 0 {
		return c
	}

	mem.storeMap.Range(func(k, v interface{}) bool {
		if v.(mValue).Expire != -1 {
			mem.deleteMValue(k.(string), true)
			c++
			return false
		}
		return true
	})

	return c
}

// AutoCleanExpireKey 自动在一定时间内清理过期key
// 当设置了大量的 expire key 且通常只读取一次的情况下再建议使用。
// interval 建议设置大一点，否则可能影响读性能，建议设置 5 minute
// go AutoCleanExpireKey(5 * time.Minute)
func (mem *MemSyncMapCache) AutoCleanExpireKey(interval time.Duration) {
	mem.once.Do(func() {
		timer := time.NewTimer(interval)
		for {
			if mem.isClosed() {
				timer.Stop()
				return
			}
			select {
			case <-timer.C:
				mem.expireClean()
			}
		}
	})
}

// 在超量后，才执行此函数
func (mem *MemSyncMapCache) expireClean() int {
	if mem.isClosed() {
		return 0
	}

	keys := make([]string, 0)
	mem.storeMap.Range(func(k, v interface{}) bool {
		if v.(mValue).isExpire() {
			keys = append(keys, k.(string))
		}
		return true
	})

	for _, key := range keys {
		mem.deleteMValue(key, true)
	}

	// 删除数量
	return len(keys)
}

func (mem *MemSyncMapCache) writeToDisk() error {
	datas := make(map[string]mValue, mem.currentCapacity)
	mem.storeMap.Range(func(k, v interface{}) bool {
		datas[k.(string)] = v.(mValue)
		return true
	})

	byt, err := json.Marshal(datas)
	if err != nil {
		return err
	}
	return mem.disk.WriteToFile(byt)
}

func (mem *MemSyncMapCache) readFromDisk() error {
	datas := make(map[string]mValue, 0)

	byt, err := mem.disk.ReadFromFile()
	if err != nil {
		return err
	}

	if len(byt) == 0 {
		return nil
	}

	err = json.Unmarshal(byt, &datas)
	if err != nil {
		return err
	}

	for k, mValue := range datas {
		if !mValue.isExpire() {
			// 没有过期再写入
			mem.setMValue(k, mValue.Value, mValue.validExpire())
		}
	}

	return nil
}
