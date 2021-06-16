package gocache

import (
	"fmt"
	"testing"
	"time"
)

func TestMemCacheImpl_SetAndGet(t *testing.T) {
	cache := NewSyncMapCache()

	err := cache.Set("set", "1")
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

func TestMemCacheImpl_SetAndGetExpire(t *testing.T) {
	cache := NewSyncMapCache()

	err := cache.SetWithExpire("set", "1", 15)
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

func TestMemCacheImpl_Delete(t *testing.T) {
	cache := NewSyncMapCache()

	err := cache.Set("set", "1")
	if err != nil {
		t.Fatal(err)
		return
	}

	cache.Delete("set")

	_, ok := cache.Get("set")
	if ok {
		t.Fatal("delete error")
		return
	}
}

func TestMemCacheImpl_Keys(t *testing.T) {
	cache := NewSyncMapCache()

	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			err := cache.Set(fmt.Sprintf("%d", i), i)
			if err != nil {
				t.Fatal("ExpireKeyRandomCleanProcessMode error")
				return
			}
		} else {
			err := cache.SetWithExpire(fmt.Sprintf("Expire-%d", i), i, 20)
			if err != nil {
				t.Fatal("ExpireKeyRandomCleanProcessMode error")
				return
			}
		}
	}

	keys := cache.Keys("Expire")
	if keys.Size() != 5 {
		t.Fatal("keys prefix error ", keys.Size())
		return
	}
	for _, key := range keys.Value() {
		switch key {
		case "Expire-1", "Expire-3", "Expire-5", "Expire-7", "Expire-9":
		default:
			t.Fatal("prefix keys.Value()")
			return
		}
	}
	// 删除掉过期的key
	time.Sleep(30 * time.Millisecond)

	keys = cache.Keys("")
	if keys.Size() != 5 {
		t.Fatal("keys error ", keys.Size())
		return
	}
	for _, key := range keys.Value() {
		switch key {
		case "0", "2", "4", "6", "8":
		default:
			t.Fatal("---- keys.Value()")
			return
		}
	}
}

func TestMemCacheImpl_FlushAll(t *testing.T) {
	cache := NewSyncMapCache()

	err := cache.Set("set", "1")
	if err != nil {
		t.Fatal(err)
		return
	}

	cache.FlushAll()

	_, ok := cache.Get("set")
	if ok {
		t.Fatal("flush all error")
		return
	}
}

func TestMemCacheImpl_Capacity(t *testing.T) {
	config := MemCacheConfig{
		Capacity:                5,
		OverCapacityProcessMode: ErrProcessMode,
	}
	cache := NewSyncMapCacheWithConfig(config)

	for i := 0; i < 6; i++ {
		err := cache.Set(fmt.Sprintf("%d", i), i)
		if err != nil && i != 5 {
			t.Fatal("0-4 error")
			return
		} else if err != nil && i == 5 {
			if err != ErrOverCapacity {
				t.Fatal("over capacity ", err.Error())
				return
			}
		}
	}
	// #----- RandomCleanProcessMode
	config.OverCapacityProcessMode = RandomCleanProcessMode

	cache1 := NewSyncMapCacheWithConfig(config)

	for i := 0; i < 10; i++ {
		err := cache1.Set(fmt.Sprintf("%d", i), i)
		if err != nil {
			t.Fatal(err)
			return
		}
		if i > 5 {
			size := cache1.Size()
			if size != 5 {
				t.Fatal("size error", size)
				return
			}
		}
	}

	// #-----

	config.OverCapacityProcessMode = ExpireKeyRandomCleanProcessMode
	cache2 := NewSyncMapCacheWithConfig(config)

	for i := 0; i < 10; i++ {
		if i < 6 {
			err := cache2.Set(fmt.Sprintf("%d", i), i)
			if i == 5 && err == nil {
				t.Fatal("should capacity error")
				return
			}
		}
	}

	cache2.FlushAll()

	for i := 1; i <= 10; i++ {
		if i%2 == 0 {
			err := cache2.Set(fmt.Sprintf("%d", i), i)
			if err != nil {
				t.Fatal("ExpireKeyRandomCleanProcessMode error")
				return
			}
		} else {
			err := cache2.SetWithExpire(fmt.Sprintf("SetWithExpire-%d", i), i, 20)
			if err != nil {
				t.Fatal("ExpireKeyRandomCleanProcessMode error")
				return
			}
		}
	}
}

func TestMemCacheImpl_AutoCleanExpireKey(t *testing.T) {
	cache := NewSyncMapCache()

	go cache.AutoCleanExpireKey(10 * time.Millisecond)

	err := cache.Set("set", "1")
	if err != nil {
		t.Fatal(err)
		return
	}

	err = cache.SetWithExpire("SetWithExpire", "2", 10)
	if err != nil {
		t.Fatal(err)
		return
	}

	size := cache.Size()
	if size != 2 {
		t.Fatal("size err")
		return
	}

	time.Sleep(50 * time.Millisecond)

	size = cache.Size()
	if size != 1 {
		t.Fatal("AutoCleanExpireKey err")
		return
	}

}

type iType struct {
	Value string
}

type iiType struct {
	Number int
}

func TestMemCacheImpl_SaveAndLoad(t *testing.T) {
	value := iType{Value: "1"}
	cache := NewSyncMapCache()
	err := cache.Set("Set", value)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = cache.SetWithExpire("SetWithExpire", "2", 5)
	if err != nil {
		t.Fatal(err)
		return
	}
	cache.GobRegisterCustomType(iType{})
	cache.GobRegisterCustomType(iiType{})
	time.Sleep(10 * time.Millisecond)

	err = cache.WriteToDisk()
	if err != nil {
		t.Fatal(err)
		return
	}

	cache.Delete("Set")

	// 加载保存的数据
	err = cache.LoadFromDisk()
	if err != nil {
		t.Fatal(err)
		return
	}
	v, ok := cache.Get("Set")
	if !ok {
		t.Fatal("write disk error")
		return
	}

	if v.(iType).Value != "1" {
		t.Fatal("write disk value not equal")
		return
	}

	v, ok = cache.Get("SetWithExpire")
	if ok {
		t.Fatal("SetWithExpire should not exists")
		return
	}

	// 删除文件
	//err = os.Remove(cache.disk.Filename)
	//if err != nil {
	//	t.Fatal("remove file ", err.Error())
	//	return
	//}
}
