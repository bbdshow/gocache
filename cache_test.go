package gocache

import (
	"fmt"
	"os"
	"testing"
	"time"
)

var cache *Cache
var dir, _ = os.Getwd()
var opt = Options{
	MaxSize:       1000,
	CleanInterval: time.Second * 3,
	AutoSave:      false,
	SaveType:      SaveAllKeysMode,
	Filename:      dir + "/cache.back",
}

func init() {

	var err error

	cache, err = NewCache(opt)
	if err != nil {
		fmt.Println("new cache ", err.Error())
		os.Exit(1)
	}
	fmt.Println("cache init")
}
func TestLoadData(t *testing.T) {
	if filenameExists(dir + "/cache.back") {
		v, ok := cache.Get("test")
		if ok {
			if !opt.AutoSave {
				t.Fail()
			} else if v.(string) != "123" {
				t.Fatal(v.(string))
			}
		}
	}
}
func TestGetAndSet(t *testing.T) {
	cache.Set("test", "123")

	v, ok := cache.Get("test")
	if !ok {
		t.Fail()
	}
	if v.(string) != "123" {
		t.Fatal(v.(string))
	}
}

func TestSaveDisk(t *testing.T) {
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExpireClean(t *testing.T) {
	cache.SetExpire("expire", "expire", time.Second*1)
	v, ok := cache.Get("expire")
	if !ok {
		t.Fail()
	}

	if v.(string) != "expire" {
		t.Fatal(v.(string))
	}
	size := cache.Size()

	time.Sleep(time.Second * 4)
	v, ok = cache.Get("expire")
	if ok {
		t.Fail()
	}
	if cache.Size()+1 != size {
		t.Fail()
	}
}
