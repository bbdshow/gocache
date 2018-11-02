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
	CleanInterval: time.Millisecond * 500,
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

func TestExpireClean(t *testing.T) {
	cache.SetExpire("expire", "expire", time.Millisecond*200)
	v, ok := cache.Get("expire")
	if !ok {
		t.Fail()
	}

	if v.(string) != "expire" {
		t.Fatal(v.(string))
	}
	size := cache.Size()

	time.Sleep(time.Millisecond * 800)
	v, ok = cache.Get("expire")
	if ok {
		t.Fail()
	}
	if cache.Size()+1 != size {
		t.Fail()
	}
}

func TestSaveDisk(t *testing.T) {
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadData(t *testing.T) {
	c, err := NewCache(opt)
	if err != nil {
		t.Fatal(err)
	}

	v, ok := c.Get("test")
	if ok {
		if !opt.AutoSave {
			t.Fail()
		} else if v.(string) != "123" {
			t.Fatal(v.(string))
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	opt.SaveType = SaveAllKeysMode
	c, err := NewCache(opt)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Set("SaveAllKeysMode", "SaveAllKeysMode"); err != nil {
		t.Fatal(err)
	}
	c.SaveDisk(opt.Filename, opt.SaveType)

	opt.AutoSave = false
	c.Close()

	// 重置缓存
	c, err = NewCache(opt)
	if err := c.LoadDisk(opt.Filename); err != nil {
		t.Fatal(err)
	}

	v, ok := c.Get("SaveAllKeysMode")
	if !ok || v.(string) != "SaveAllKeysMode" {
		t.Fatal("load data loss")
	}
}

func TestAutoSaveAndLoad(t *testing.T) {
	opt.SaveType = SaveAllKeysMode
	opt.AutoSave = true

	c, err := NewCache(opt)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Set("autoSaveAllKeysMode", "SaveAllKeysMode"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetExpire("SaveAllKeysMode", "SaveAllKeysMode", time.Minute); err != nil {
		t.Fatal(err)
	}
	// 模拟程序正常退出 close
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}

	// 重置缓存, 自动加载已经保存的文件
	c, err = NewCache(opt)

	v, ok := c.Get("autoSaveAllKeysMode")
	if !ok || v.(string) != "SaveAllKeysMode" {
		t.Fatal("load data loss")
	}

	v, ok = c.Get("SaveAllKeysMode")
	if !ok || v.(string) != "SaveAllKeysMode" {
		t.Fatal("load data loss")
	}
}

func TestAutoSaveAndLoadMode(t *testing.T) {
	opt.SaveType = SaveExpireKeysMode
	opt.AutoSave = true

	c, err := NewCache(opt)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Set("testSaveMode", "SaveAllKeysMode"); err != nil {
		t.Fatal(err)
	}

	if err := c.SetExpire("SaveExpireKeysMode", "SaveExpireKeysMode", time.Minute); err != nil {
		t.Fatal(err)
	}

	// 模拟程序正常退出 close
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}

	// 重置缓存, 自动加载已经保存的文件
	c, err = NewCache(opt)

	v, ok := c.Get("SaveExpireKeysMode")
	if !ok || v.(string) != "SaveExpireKeysMode" {
		t.Fatal("SaveExpireKeysMode load data loss")
	}

	v, ok = c.Get("testSaveMode")
	if ok {
		t.Fatal("save data mode err")
	}

	// 保存永久 key
	opt.SaveType = SaveNoExpireKeysMode

	c, err = NewCache(opt)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Set("testSaveMode", "SaveAllKeysMode"); err != nil {
		t.Fatal(err)
	}

	if err := c.SetExpire("SaveExpireKeysMode", "SaveExpireKeysMode", time.Minute); err != nil {
		t.Fatal(err)
	}

	// 模拟程序正常退出 close
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}

	// 重置缓存, 自动加载已经保存的文件
	c, err = NewCache(opt)

	v, ok = c.Get("testSaveMode")
	if !ok || v.(string) != "SaveAllKeysMode" {
		t.Fatal("SaveAllKeysMode load data loss")
	}

	v, ok = c.Get("SaveExpireKeysMode")
	if ok {
		t.Fatal("save data mode err")
	}
}
