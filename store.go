package gocache

import "sync"

type SyncMap struct {
	store sync.Map
}

func NewSyncMap() *SyncMap {
	sm := SyncMap{
		store: sync.Map{},
	}
	return &sm
}

func (s *SyncMap) Load(key string) (value interface{}, ok bool) {
	value, ok = s.store.Load(key)
	return
}

func (s *SyncMap) Store(key string, value interface{}) {
	s.store.Store(key, value)
}

func (s *SyncMap) Delete(key string) {
	s.store.Delete(key)
}

func (s *SyncMap) LoadOrStore(key string, value interface{}) (actual interface{}, loaded bool) {
	return s.store.LoadOrStore(key, value)
}

func (s *SyncMap) Exists(key string) bool {
	_, ok := s.Load(key)
	return ok
}

func (s *SyncMap) Range(f func(k string, v interface{}) bool) {
	s.store.Range(func(key, value interface{}) bool {
		return f(key.(string), value)
	})
}

func (s *SyncMap) Flush() {
	s.store = sync.Map{}
}

// -------

type RWMap struct {
	rwMutex sync.RWMutex
	store   map[string]interface{}
}

func NewRWMap() *RWMap {
	rw := RWMap{
		rwMutex: sync.RWMutex{},
		store:   make(map[string]interface{}),
	}
	return &rw
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
