package gocache

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"strings"
	"sync"
	"time"
)

/*
	内存实现cache

Notice:
	1. 使用 sync.Map 实现，最大限度优化读的性能，特别是在 cpu > 4核的情况下，因为没有锁的竞争，读取优势很明显
	但是内存消耗严重(约等于2倍)，空间换时间。而且写入性能没有 rwMutex map 性能好
	2. 当存在大量的 TTLKey 时，不建议总缓存Key超过百万，在清理TTLKey时会占用锁，会有一定的写入延迟
	3. 不建议缓存内容过大时，执行写入磁盘操作，写入时会进行编码，会申请内容的一倍内存，如果内存不够，会导致OOM
*/

var (
	ErrKeysOverLimitSize = errors.New("keys over limit size")
)

type Config struct {
	// 缓存容量， -1 - 不限制
	LimitSize int64
	// 保存文件位置, 不设置，默认当前执行路径
	Filename string
}

func NewRWMapCache() *MemCache {
	return NewMemCache(NewRWMap())
}

func NewRWMapCacheWithConfig(config Config) *MemCache {
	return NewMemCacheWithConfig(NewRWMap(), config)
}

func NewSyncMapCache() *MemCache {
	return NewMemCache(NewSyncMap())
}

func NewSyncMapCacheWithConfig(config Config) *MemCache {
	return NewMemCacheWithConfig(NewSyncMap(), config)
}

func NewMemCache(store Store) *MemCache {
	return NewMemCacheWithConfig(store, Config{
		LimitSize: -1,
	})
}

func NewMemCacheWithConfig(store Store, config Config) *MemCache {
	mem := MemCache{
		limitSize: config.LimitSize,

		store: store,

		disk: NewDisk(config.Filename),

		exit: make(chan int, 1),
	}

	return &mem
}

/*
	MemCache
*/
type MemCache struct {
	// Key  limit cap, default -1 not limit
	limitSize int64

	// 存储所有数据 key , value - expireValue
	store Store

	// 写入磁盘
	disk *Disk

	once sync.Once

	exit chan int
}

type expireValue struct {
	Value  interface{}
	Expire int64 // expire time /sec  -1 never expire
}

func (ev *expireValue) ttl(ttl int64) {
	if ttl <= -1 {
		ev.Expire = -1
		return
	}
	ev.Expire = time.Now().Unix() + ttl
}

func (ev expireValue) surplusSec(nowSec int64) int64 {
	if ev.Expire == -1 {
		return ev.Expire
	}
	expire := ev.Expire - nowSec
	// 存在这种可能
	if expire < 0 {
		expire = 0
	}

	return expire
}

// true - expired
// nowSec Avoid frequent timing
func (ev expireValue) isExpire(nowSec int64) bool {
	if ev.Expire == -1 {
		return false
	}
	return nowSec >= ev.Expire
}

func (mem *MemCache) Get(key string) (value interface{}, ok bool) {
	v, ok := mem.getValue(key)
	if !ok {
		return nil, ok
	}
	return v.Value, ok
}

func (mem *MemCache) GetWithExpire(key string) (value interface{}, ttl int64, ok bool) {
	v, ok := mem.getValue(key)
	if !ok {
		return nil, 0, ok
	}

	if v.Expire == -1 {
		return v.Value, v.Expire, ok
	}

	return v.Value, v.surplusSec(time.Now().Unix()), ok
}

func (mem *MemCache) Set(key string, value interface{}) error {
	return mem.SetWithExpire(key, value, -1)
}

// SetWithExpire  ttl - 过期时间秒级别， -1 永久有效
func (mem *MemCache) SetWithExpire(key string, value interface{}, ttl int64) error {

	if mem.limitSize >= 0 {
		if mem.Size() >= mem.limitSize {
			return ErrKeysOverLimitSize
		}
	}

	mem.setValue(key, value, ttl)

	return nil
}

func (mem *MemCache) Delete(key string) {
	mem.store.Delete(key)
}

type iKeys struct {
	keys   []string
	length int64
}

func (k *iKeys) Size() int64 {
	return k.length
}

func (k *iKeys) Value() []string {
	return k.keys
}

func (mem *MemCache) Keys(prefix string) Keys {
	keys := iKeys{
		keys: make([]string, 0, mem.store.Size()),
	}
	nowSec := time.Now().Unix()
	mem.store.Range(func(k string, v interface{}) bool {
		if !v.(expireValue).isExpire(nowSec) {
			if len(prefix) != 0 && !strings.HasPrefix(k, prefix) {
				return true
			}
			keys.keys = append(keys.keys, k)
			keys.length++
		}
		return true
	})
	return &keys
}

// Size
func (mem *MemCache) Size() int64 {
	return mem.store.Size()
}

// FlushAll 清空所有数据
func (mem *MemCache) FlushAll() {
	mem.store.Flush()
}

// Close
func (mem *MemCache) Close() { mem.exit <- 1 }

func (mem *MemCache) getValue(key string) (expireValue, bool) {
	v, ok := mem.store.Load(key)
	if !ok {
		return expireValue{}, ok
	}

	ev := v.(expireValue)
	if ev.isExpire(time.Now().Unix()) {
		mem.store.Delete(key)
		return expireValue{}, false
	}

	return ev, true
}

func (mem *MemCache) setValue(key string, value interface{}, ttl int64) {
	if ttl == 0 {
		return
	}
	ev := expireValue{
		Value: value,
	}
	ev.ttl(ttl)

	// LoadOrStore 为了准确计数当前容量
	mem.store.LoadOrStore(key, ev)
}

// AutoCleanExpireKey 自动在一定时间内清理过期 key
// 当设置了大量的 expire key 且通常只读取一次的情况下再建议使用。
// interval 建议设置大一点，否则可能影响写入性能，建议设置 5-10 minute
func (mem *MemCache) AutoCleanExpireKey(interval time.Duration) {
	mem.once.Do(func() {
		go func() {
			ticker := time.NewTicker(interval)
			for {
				select {
				case <-mem.exit:
					ticker.Stop()
					return
				case <-ticker.C:
					mem.expireClean()
				}
			}
		}()
	})
}

// 在超量后，才执行此函数
func (mem *MemCache) expireClean() int {
	keys := make([]string, 0)
	nowSec := time.Now().Unix()
	mem.store.Range(func(k string, v interface{}) bool {
		if v.(expireValue).isExpire(nowSec) {
			keys = append(keys, k)
		}
		return true
	})

	for _, key := range keys {
		mem.store.Delete(key)
	}
	// 删除数量
	return len(keys)
}

// GobRegister 注册自定义结构
func (mem *MemCache) GobRegister(v ...interface{}) {
	for _, vv := range v {
		if vv != nil {
			gob.Register(vv)
		}
	}
}

// WriteToDisk 缓存内容写入磁盘，当缓存内容比较大时，不建议写入磁盘，比较耗费时间
func (mem *MemCache) WriteToDisk() error {

	data := bytes.Buffer{}
	enc := gob.NewEncoder(&data)

	nowSec := time.Now().Unix()
	values := make(map[string]expireValue, mem.Size())
	mem.store.Range(func(k string, v interface{}) bool {
		value := v.(expireValue)
		if !value.isExpire(nowSec) {
			values[k] = value
		}
		return true
	})
	log.Printf("WriteToDisk: to save the %d keys,in progress JSON encoding\n", len(values))
	err := enc.Encode(values)
	if err != nil {
		return err
	}
	log.Printf("WriteToDisk: the encoded data size is %.2fMB\n", float64(data.Len())/1024/1024)
	return mem.disk.WriteToFile(data.Bytes())
}

// LoadFromDisk 从磁盘中读取缓存内容，过过滤掉已经过期的内容
func (mem *MemCache) LoadFromDisk() error {
	values := make(map[string]expireValue, 0)

	data, err := mem.disk.ReadFromFile()
	if err != nil {
		return err
	}
	log.Printf("LoadFromDisk: the encoded data size is %.2fMB\n", float64(len(data))/1024/1024)

	if len(data) == 0 {
		return nil
	}

	store := bytes.Buffer{}
	store.Write(data)
	dec := gob.NewDecoder(&store)
	err = dec.Decode(&values)
	if err != nil {
		return err
	}

	nowSec := time.Now().Unix()

	for k, v := range values {
		if !v.isExpire(nowSec) {
			// 没有过期再写入
			mem.setValue(k, v.Value, v.surplusSec(nowSec))
		}
	}

	return nil
}
