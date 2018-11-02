# gocache

### 特性

1. 采用 sync.Map 在多核，大量读取，锁竞争多的情况下存在优势
2. 支持容量控制，容量控制采用 maxSize 预估 keys 容量实现， 容量达到限制后由多种策略控制内存使用量
3. 支持重启缓存内容 选择性 自动保存／加载，简单防止因重启导致的缓存击穿
4. 缓存定时过期清除，单独存储 过期时间 keys ，防止定时清除时锁定主存储太久。

##### 注版本要求 golang 1.9^

### 为什么选择 sync.Map 

利用 sync.map 达到读取性能相对更高，sync.Map 并不太适应大量写入的缓存操作, 且因为计数使用了 LoadOrStrore 对 key 计数。
sync.Map 在空间上并不占优势。如果存在频繁写入建议使用 RwMutex map  github.com/patrickmn/go-cache

在 4 核 机器上 锁竞争不明显， 所以 RwMutex map 在性能上更占优势，但是当 cpu 核数 往上时， 锁竞争变大， sync.Map 的优势就体现出来了。
性能测试 引用 https://medium.com/@deckarep/the-new-kid-in-town-gos-sync-map-de24a6bf7c2c


### Usage

``` go

package main

import (
	"fmt"
	"log"
	"os"
    "time"
    "github.com/hopingtop/gocache"
)

var dir, _ = os.Getwd()
var opt = gocache.Options{
	MaxSize:       1000,
	CleanInterval: time.Millisecond * 500,
	AutoSave:      true,
	SaveType:      gocache.SaveAllKeysMode,
	Filename:      dir + "/cache.back",
}

func main() {
	cache, err := gocache.NewCache(opt)
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

```

``` go
// Options 配置选项说明
type Options struct {
	// 超过容量限制自动清除 keys 方式
	OverSizeClearMode cleanMode
	// keys 容量限制，通过此预估内存使用量
	MaxSize int64
	// 定时清除过期的 key, 未设置，则不清除
	CleanInterval time.Duration

	// 自动保存缓存内容到文件
	AutoSave bool
	// 保存的内容类型
	SaveType saveMode
	// 保存到文件名， 绝对路径 eg: /tmp/cache.back
	Filename string
}
```


