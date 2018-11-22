package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hopingtop/gocache"
)

var dir, _ = os.Getwd()
var cfg = gocache.Config{
	MaxSize:       1000,
	CleanInterval: time.Millisecond * 500,
	AutoSave:      true,
	SaveType:      gocache.SaveAllKeysMode,
	Filename:      dir + "/cache.back",
}

func main() {
	cache, err := gocache.NewCache(cfg)
	if err != nil {
		log.Panic(err.Error())
	}

	//如果设置了 容量，请处理 error
	cache.Set("test", "1")
	cache.SetExpire("testExpire", "2", time.Minute)

	v, ok := cache.Get("test")
	fmt.Println(v.(string), ok)

	v, t, ok := cache.GetWithExpire("testExpire")
	fmt.Println(v.(string), t.Seconds(), ok)

	// 释放资源， 如果 autoSave 开启，则保存内存内容到文件， NewCache 自动加载
	if err := cache.Close(); err != nil {
		// 如果保存文件存在 error
		log.Panic(err.Error())
	}
}
