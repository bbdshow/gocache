package gocache

import (
	"testing"
	"time"
)

func TestMemSyncMapCache_SetAndGet(t *testing.T) {
	cache, err := NewMemSyncMapCache()
	if err != nil {
		t.Fatal(err)
		return
	}

	err = cache.Set("set", "1")
	if err != nil {
		t.Fatal(err)
		return
	}

	v, ok := cache.Get("set")
	if !ok {
		t.Fatal("not exists")
		return
	}

	if v.(string) != "1" {
		t.Fatal("value equal")
		return
	}
}

func TestMemSyncMapCache_SetAndGetExpire(t *testing.T) {
	cache, err := NewMemSyncMapCache()
	if err != nil {
		t.Fatal(err)
		return
	}

	err = cache.SetWithExpire("set", "1", 15)
	if err != nil {
		t.Fatal(err)
		return
	}

	_, e, ok := cache.GetWithExpire("set")
	if !ok {
		t.Fatal("not exists")
		return
	}

	if e < 10 {
		t.Fatal("expire error", e)
		return
	}

	time.Sleep(20 * time.Millisecond)

	_, ok = cache.Get("set")
	if ok {
		t.Fatal("key should expired")
	}

}
