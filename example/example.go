package main

import (
	"log"
	"time"

	"github.com/huzhongqing/gocache"
)

var cache gocache.Cache

func main() {
	store := gocache.NewSyncMap()
	cacheImpl := gocache.NewMemCacheWithConfig(store, gocache.DefaultMemConfig())

	// 如果存在大量 TTL key 建议
	go cacheImpl.AutoCleanExpireKey(5 * time.Minute)

	cache = cacheImpl

	// 如果之前保存过文件，建议
	err := cache.LoadFromDisk()
	if err != nil {
		log.Println("LoadFromDisk", err)
		return
	}

	cache.Set("Set", "1") // 如果capacity==-1， 则不用处理 error

	v, ok := cache.Get("Set")
	if ok {
		log.Println("key: set, value: " + v.(string))
	}

	// 退出时，可以选择落盘
	err = cache.WriteToDisk()
	if err != nil {
		log.Println("WriteToDisk", err)
		return
	}

	cache.Close()
}
