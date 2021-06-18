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

	err := cache.SetWithExpire("set", "1", 3)
	if err != nil {
		t.Fatal(err)
		return
	}

	_, ttl, ok := cache.GetWithExpire("set")
	if !ok {
		t.Fatal("not exists")
		return
	}

	if ttl < 2 {
		t.Fatal("expire error", ttl)
		return
	}

	time.Sleep(3 * time.Second)

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
			err := cache.SetWithExpire(fmt.Sprintf("Expire-%d", i), i, 1)
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
	time.Sleep(3 * time.Second)

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
	config := Config{
		LimitSize: 5,
	}
	cache := NewSyncMapCacheWithConfig(config)

	for i := 0; i < 6; i++ {
		err := cache.Set(fmt.Sprintf("%d", i), i)
		if err != nil && i != 5 {
			t.Fatal("0-4 error")
			return
		} else if err != nil && i == 5 {
			if err != ErrKeysOverLimitSize {
				t.Fatal("over capacity ", err.Error())
				return
			}
		}
	}
}

func TestMemCacheImpl_AutoCleanExpireKey(t *testing.T) {
	cache := NewSyncMapCache()

	cache.AutoCleanExpireKey(10 * time.Millisecond)

	err := cache.Set("set", "1")
	if err != nil {
		t.Fatal(err)
		return
	}

	err = cache.SetWithExpire("SetWithExpire", "2", 2)
	if err != nil {
		t.Fatal(err)
		return
	}

	size := cache.Size()
	if size != 2 {
		t.Fatal("size err")
		return
	}

	time.Sleep(3 * time.Second)

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

// 暂时关闭，过不了 Travis 文件权限

//func TestMemCacheImpl_SaveAndLoad(t *testing.T) {
//	value := iType{Value: "1"}
//	cache := NewSyncMapCacheWithConfig(Config{
//		LimitSize: -1,
//		Filename:  "./test/cache.gob",
//	})
//
//	cache.GobRegister(iType{}, &iiType{})
//
//	err := cache.Set("Set", value)
//	if err != nil {
//		t.Fatal(err)
//		return
//	}
//
//	err = cache.SetWithExpire("SetWithExpire", "2", 1)
//	if err != nil {
//		t.Fatal(err)
//		return
//	}
//	value1 := &iiType{Number: 2}
//	err = cache.Set("Set1", value1)
//	if err != nil {
//		t.Fatal(err)
//		return
//	}
//
//	time.Sleep(2 * time.Second)
//
//	err = cache.WriteToDisk()
//	if err != nil {
//		t.Fatal(err)
//		return
//	}
//
//	cache.Delete("Set")
//	cache.Delete("Set1")
//
//	_, ok := cache.Get("Set1")
//	if ok {
//		t.Fatal("Set1 should delete")
//	}
//
//	// 加载保存的数据
//	err = cache.LoadFromDisk()
//	if err != nil {
//		t.Fatal(err)
//		return
//	}
//	v, ok := cache.Get("Set")
//	if !ok {
//		t.Fatal("write disk error")
//		return
//	}
//
//	if v.(iType).Value != "1" {
//		t.Fatal("write disk value not equal")
//		return
//	}
//
//	v1, ok := cache.Get("Set1")
//	if !ok {
//		t.Fatal("write disk error")
//		return
//	}
//
//	if v1.(*iiType).Number != 2 {
//		t.Fatal("write disk iiType value not equal")
//		return
//	}
//
//	v, ok = cache.Get("SetWithExpire")
//	if ok {
//		t.Fatal("SetWithExpire should not exists")
//		return
//	}
//
//	// 删除文件
//	//err = os.Remove(cache.disk.Filename)
//	//if err != nil {
//	//	t.Fatal("remove file ", err.Error())
//	//	return
//	//}
//}
