package main

import (
	"log"
	"time"

	"github.com/bbdshow/gocache"
)

var cache gocache.Cache

type Test struct {
	Value string
}

func main() {
	store := gocache.NewSyncMap()
	cacheImpl := gocache.NewMemCacheWithConfig(store, gocache.DefaultMemConfig())

	// 如果存在大量 TTL key 建议
	go cacheImpl.AutoCleanExpireKey(5 * time.Minute)

	cache = cacheImpl

	// 如果存储自定义结构体，又保存文件则要注册结构体
	cache.GobRegisterCustomType(Test{})

	// 如果之前保存过文件，建议
	err := cache.LoadFromDisk()
	if err != nil {
		log.Println("LoadFromDisk", err)
		return
	}

	cache.Set("Set", Test{Value: "1"}) // 如果capacity==-1， 则不用处理 error

	v, ok := cache.Get("Set")
	if ok {
		log.Println("key: set, value: " + v.(Test).Value)
	}

	// 退出时，可以选择落盘
	err = cache.WriteToDisk()
	if err != nil {
		log.Println("WriteToDisk", err)
		return
	}

	cache.Delete("Set")

	log.Println("size: ", cache.Size())

	err = cache.LoadFromDisk()
	if err != nil {
		log.Println("LoadFromDisk", err)
		return
	}

	v, ok = cache.Get("Set")
	if ok {
		log.Println("LoadFromDisk key: set, value: " + v.(Test).Value)
	}

	cache.Close()
}
