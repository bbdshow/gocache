package gocache

import (
	"sync"
	"sync/atomic"
)

// SyncMap 并发Map 当数据竞争大时，多核CPU时，使用比RwMap性能好，缺点空间占用会多点
type SyncMap struct {
	store sync.Map
	// about the value, Store func, I don't know if it exists
	size int64
}

func NewSyncMap() *SyncMap {
	s := SyncMap{
		store: sync.Map{},
	}
	return &s
}

func (s *SyncMap) Load(key string) (value interface{}, ok bool) {
	value, ok = s.store.Load(key)
	return
}

func (s *SyncMap) Store(key string, value interface{}) {
	s.store.Store(key, value)
}

func (s *SyncMap) Delete(key string) {
	_, loaded := s.store.LoadAndDelete(key)
	if loaded {
		atomic.AddInt64(&s.size, -1)
	}
}

func (s *SyncMap) LoadOrStore(key string, value interface{}) (actual interface{}, loaded bool) {
	v, loaded := s.store.LoadOrStore(key, value)
	if !loaded {
		atomic.AddInt64(&s.size, 1)
	}
	return v, loaded
}

func (s *SyncMap) Exists(key string) bool {
	_, ok := s.Load(key)
	return ok
}

func (s *SyncMap) Range(fn func(k string, v interface{}) bool) {
	s.store.Range(func(key, value interface{}) bool {
		return fn(key.(string), value)
	})
}

func (s *SyncMap) Flush() {
	s.store = sync.Map{}
	atomic.StoreInt64(&s.size, 0)
}

func (s *SyncMap) Size() int64 {
	size := atomic.LoadInt64(&s.size)
	if size < 0 {
		return 0
	}
	return size
}

// RWMap 读写Map 当数据竞争不强，或读取多时。使用节省空间更快
type RWMap struct {
	rwMutex sync.RWMutex
	store   map[string]interface{}
}

func NewRWMap() *RWMap {
	s := RWMap{
		rwMutex: sync.RWMutex{},
		store:   make(map[string]interface{}),
	}
	return &s
}

func (s *RWMap) Load(key string) (value interface{}, ok bool) {
	s.rwMutex.RLock()
	value, ok = s.store[key]
	s.rwMutex.RUnlock()

	return value, ok
}

func (s *RWMap) Store(key string, value interface{}) {
	s.rwMutex.Lock()
	s.store[key] = value
	s.rwMutex.Unlock()
}

func (s *RWMap) Delete(key string) {
	s.rwMutex.Lock()
	delete(s.store, key)
	s.rwMutex.Unlock()
}

func (s *RWMap) LoadOrStore(key string, value interface{}) (actual interface{}, loaded bool) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	v, ok := s.store[key]
	if ok {
		loaded = ok
		actual = v
	}

	s.store[key] = value

	if !loaded {
		actual = value
	}

	return actual, loaded
}

func (s *RWMap) Exists(key string) bool {
	_, ok := s.Load(key)
	return ok
}

func (s *RWMap) Range(f func(k string, v interface{}) bool) {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	for k, v := range s.store {
		if !f(k, v) {
			break
		}
	}
}

func (s *RWMap) Flush() {
	s.rwMutex.Lock()
	s.store = make(map[string]interface{})
	s.rwMutex.Unlock()
}

func (s *RWMap) Size() int64 {
	s.rwMutex.Lock()
	size := len(s.store)
	s.rwMutex.Unlock()
	return int64(size)
}
