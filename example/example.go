package main

import (
	"log"

	"github.com/huzhongqing/gocache"
)

var cache gocache.Cache

func main() {
	// 可以自定义配置 MemSyncMapCacheConfig{}
	memc, err := gocache.NewMemSyncMapCache()
	if err != nil {
		log.Println("NewMemSyncMapCache", err)
		return
	}
	cache = memc

	cache.Set("Set", "1") // 如果capacity==-1， 则不用处理 error

	v, ok := cache.Get("Set")
	if ok {
		log.Println("key: set, value: " + v.(string))
	}

	err = cache.Close()
	if err != nil {
		log.Println("Close", err)
		return
	}
}
