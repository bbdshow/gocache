package main

import (
	"log"
	"time"

	"github.com/bbdshow/gocache"
)

var cache gocache.Cache

type User struct {
	Name string
	Age  int
}
type Address struct {
	RoomNo string
	Addr   string
}

func main() {
	store := gocache.NewSyncMap()
	cacheImpl := gocache.NewMemCacheWithConfig(store, gocache.Config{
		LimitSize: -1,
		Filename:  "/test/example.gob",
	})
	// 如果要保存到文件，一定要注册自定义结构体
	cacheImpl.GobRegister(User{}, Address{})

	// 如果存在大量 TTL key 建议
	cacheImpl.AutoCleanExpireKey(5 * time.Minute)

	cache = cacheImpl

	// 如果之前保存过文件，建议
	err := cache.LoadFromDisk()
	if err != nil {
		log.Println("LoadFromDisk", err)
		return
	}

	_ = cache.Set("User", User{Name: "goCache", Age: 1}) // 如果LimitSize = -1， 则不用处理 error

	v, ok := cache.Get("User")
	if ok {
		log.Println("key: set, value: " + v.(User).Name)
	}

	_ = cache.Set("&Address", &Address{RoomNo: "888", Addr: "CQ"}) // 如果LimitSize = -1， 则不用处理 error

	v, ok = cache.Get("&Address")
	if ok {
		log.Println("key: set, value: " + v.(*Address).RoomNo)
	}

	//当进程或者任务退出时，可以选择落盘, 如果数据量过大，不建议落盘，因为编码等问题，会申请 2*缓存容量，可能会导致OOM
	err = cache.WriteToDisk()
	if err != nil {
		log.Println("WriteToDisk", err)
		return
	}

	cacheNew := gocache.NewMemCacheWithConfig(store, gocache.Config{
		LimitSize: -1,
		Filename:  "/test/example.gob",
	})

	err = cache.LoadFromDisk()
	if err != nil {
		log.Println("LoadFromDisk", err)
		return
	}
	log.Println("size: ", cacheNew.Size())

	v, ok = cache.Get("User")
	if ok {
		log.Println("LoadFromDisk key: User, value: " + v.(User).Name)
	}

	v, ok = cache.Get("&Address")
	if ok {
		log.Println("LoadFromDisk key: &Address, value: " + v.(*Address).RoomNo)
	}

	cache.Close()
}
