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
	1. 使用 sync.Map 实现，最大限度优化读的性能，特别是在 cpu > 4核的情况下，因为没有锁的竞争，读取优势很明显
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

type MemCacheConfig struct {

	// 缓存容量， -1 - 不限制
	Capacity int64
	// 超出缓存容量，处理方式
	OverCapacityProcessMode int

	// 保存文件位置, 不设置，默认当前执行路径
	Filename string
}

func NewRWMapCache() *MemCacheImpl {
	return NewMemCache(NewRWMap())
}

func NewRWMapCacheWithConfig(config MemCacheConfig) *MemCacheImpl {
	return NewMemCacheWithConfig(NewRWMap(), config)
}

func NewSyncMapCache() *MemCacheImpl {
	return NewMemCache(NewSyncMap())
}

func NewSyncMapCacheWithConfig(config MemCacheConfig) *MemCacheImpl {
	return NewMemCacheWithConfig(NewSyncMap(), config)
}

func NewMemCache(store Store) *MemCacheImpl {
	return NewMemCacheWithConfig(store, DefaultMemConfig())
}

func NewMemCacheWithConfig(store Store, config MemCacheConfig) *MemCacheImpl {
	if config.OverCapacityProcessMode <= -1 {
		config.OverCapacityProcessMode = -1
	}

	mem := MemCacheImpl{
		config:          &config,
		store:           store,
		currentCapacity: 0,
		disk:            NewDisk(config.Filename),
		closed:          0,
	}

	return &mem
}

func DefaultMemConfig() MemCacheConfig {
	return MemCacheConfig{
		Capacity:                defaultCapacity,
		OverCapacityProcessMode: defaultOverCapacityProcessMode,
	}
}

/*
	MemCacheImpl
*/
type MemCacheImpl struct {
	config *MemCacheConfig

	// 存储所有数据 key , value - expireValue
	store Store

	// 当前存储容量
	currentCapacity int64

	// 写入磁盘
	disk *Disk

	once   sync.Once
	closed int32
}

type expireValue struct {
	Value  interface{}
	Expire int64 // 纳秒级别
}

func (ev *expireValue) expire(expire int64) {
	if expire <= -1 {
		ev.Expire = -1
		return
	}

	ev.Expire = time.Now().Add(time.Duration(expire) * time.Millisecond).UnixNano()
}

func (ev expireValue) validExpire() int64 {
	if ev.Expire == -1 {
		return ev.Expire
	}
	expire := time.Unix(0, ev.Expire).Sub(time.Now()).Nanoseconds() / time.Millisecond.Nanoseconds()
	// 存在这种可能
	if expire < 0 {
		expire = 0
	}

	return expire
}

// true - expired
func (ev expireValue) isExpire() bool {
	if ev.Expire == -1 {
		return false
	}
	return time.Now().UnixNano() >= ev.Expire
}

func (mem *MemCacheImpl) Get(key string) (value interface{}, ok bool) {
	value, _, ok = mem.GetWithExpire(key)
	return value, ok
}

func (mem *MemCacheImpl) GetWithExpire(key string) (value interface{}, expire int64, ok bool) {
	if mem.isClosed() {
		return
	}

	v, ok := mem.getValue(key)
	if !ok {
		return nil, 0, ok
	}

	if v.Expire == -1 {
		return v.Value, v.Expire, ok
	}

	return v.Value, v.validExpire(), ok
}

func (mem *MemCacheImpl) Set(key string, value interface{}) error {
	return mem.SetWithExpire(key, value, -1)
}

// SetWithExpire  expire - 过期时间毫秒级别， -1 永久有效
func (mem *MemCacheImpl) SetWithExpire(key string, value interface{}, expire int64) error {
	if mem.isClosed() {
		return nil
	}

	err := mem.overCapacity()
	if err != nil {
		return err
	}

	mem.setValue(key, value, expire)

	return nil
}

func (mem *MemCacheImpl) Delete(key string) {
	if mem.isClosed() {
		return
	}

	ok := mem.store.Exists(key)
	if !ok {
		return
	}

	mem.deleteValue(key, true)
}

type ikeys struct {
	keys   []string
	length int64
}

func (k *ikeys) Size() int64 {
	return k.length
}

func (k *ikeys) Value() []string {
	return k.keys
}

func (mem *MemCacheImpl) Keys(prefix string) Keys {
	keys := ikeys{
		keys: make([]string, 0, mem.currentCapacity),
	}
	if prefix == "" {
		mem.store.Range(func(k string, v interface{}) bool {
			if !v.(expireValue).isExpire() {
				keys.keys = append(keys.keys, k)
				keys.length++
			}
			return true
		})
	} else {
		mem.store.Range(func(k string, v interface{}) bool {
			if strings.HasPrefix(k, prefix) {
				if !v.(expireValue).isExpire() {
					keys.keys = append(keys.keys, k)
					keys.length++
				}
			}
			return true
		})
	}

	return &keys
}

func (mem *MemCacheImpl) Size() int64 {
	return atomic.LoadInt64(&mem.currentCapacity)
}

// FlushAll 清空所有数据，先关闭所有查询/写入
func (mem *MemCacheImpl) FlushAll() {
	atomic.StoreInt32(&mem.closed, 1)

	mem.store.Flush()

	mem.currentCapacity = 0
	atomic.StoreInt32(&mem.closed, 0)
}

// Close 开启写入磁盘，则写入文件
func (mem *MemCacheImpl) Close() {
	atomic.StoreInt32(&mem.closed, 1)

	mem.store = nil
}

func (mem *MemCacheImpl) getValue(key string) (expireValue, bool) {
	v, ok := mem.store.Load(key)
	if !ok {
		return expireValue{}, ok
	}

	ev := v.(expireValue)
	if ev.isExpire() {
		mem.deleteValue(key, true)
		return expireValue{}, false
	}

	return ev, true
}

func (mem *MemCacheImpl) setValue(key string, value interface{}, expire int64) {
	if expire == 0 {
		return
	}

	ev := expireValue{
		Value: value,
	}
	ev.expire(expire)

	// LoadOrStore 为了准确计数当前容量
	_, exists := mem.store.LoadOrStore(key, ev)
	if !exists {
		atomic.AddInt64(&mem.currentCapacity, 1)
	}
}

func (mem *MemCacheImpl) deleteValue(key string, sub bool) {
	mem.store.Delete(key)
	if sub {
		atomic.AddInt64(&mem.currentCapacity, -1)
	}
}

func (mem *MemCacheImpl) isClosed() bool {
	closed := atomic.LoadInt32(&mem.closed)
	if closed == 1 {
		return true
	}
	return false
}

func (mem *MemCacheImpl) overCapacity() error {
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

func (mem *MemCacheImpl) randomDeleteOne() {
	mem.store.Range(func(k string, v interface{}) bool {
		mem.deleteValue(k, true)
		return false
	})
}

// 删除已经过期的 expire key， 如果一个就没有删除，则随机删除一个有效的 expire key
// 如果 int=0 则证明没有删除任何 key
func (mem *MemCacheImpl) randomDeleteExpireKey() int {
	c := mem.expireClean()
	if c > 0 {
		return c
	}

	mem.store.Range(func(k string, v interface{}) bool {
		if v.(expireValue).Expire != -1 {
			mem.deleteValue(k, true)
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
func (mem *MemCacheImpl) AutoCleanExpireKey(interval time.Duration) {
	mem.once.Do(func() {
		ticker := time.NewTicker(interval)
		for {
			if mem.isClosed() {
				ticker.Stop()
				return
			}
			select {
			case <-ticker.C:
				mem.expireClean()
			}
		}
	})
}

// 在超量后，才执行此函数
func (mem *MemCacheImpl) expireClean() int {
	if mem.isClosed() {
		return 0
	}

	keys := make([]string, 0)
	mem.store.Range(func(k string, v interface{}) bool {
		if v.(expireValue).isExpire() {
			keys = append(keys, k)
		}
		return true
	})

	for _, key := range keys {
		mem.deleteValue(key, true)
	}

	// 删除数量
	return len(keys)
}

func (mem *MemCacheImpl) WriteToDisk() error {
	datas := make(map[string]expireValue, mem.currentCapacity)
	mem.store.Range(func(k string, v interface{}) bool {
		data := v.(expireValue)
		if !data.isExpire() {
			datas[k] = data
		}
		return true
	})

	byt, err := json.Marshal(datas)
	if err != nil {
		return err
	}
	return mem.disk.WriteToFile(byt)
}

func (mem *MemCacheImpl) LoadFromDisk() error {
	datas := make(map[string]expireValue, 0)

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

	for k, expireValue := range datas {
		if !expireValue.isExpire() {
			// 没有过期再写入
			mem.setValue(k, expireValue.Value, expireValue.validExpire())
		}
	}

	return nil
}
