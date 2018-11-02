package gocache

import (
	"os"
	"testing"
	"time"
)

var dir, _ = os.Getwd()
var opt = Options{
	MaxSize:       1000,
	CleanInterval: time.Millisecond * 500,
	AutoSave:      false,
	SaveType:      SaveAllKeysMode,
	Filename:      dir + "/cache.back",
}
var cache, _ = NewCache(opt)

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

func TestCacheFunc(t *testing.T) {
	opt.CleanInterval = time.Millisecond * 500
	opt.MaxSize = 5
	opt.OverSizeClearMode = NoEvictionMode
	c, _ := NewCache(opt)
	c.Flush()
	c.Set("t1", 1)
	c.Set("t2", 2)
	c.Set("t3", 3)
	c.Set("t4", 4)
	c.SetExpire("t5", 5, time.Second)
	if err := c.SetExpire("t6", 6, time.Second); err == nil {
		t.Fatal("size limit fatal")
	}

	if c.Size() != 5 {
		t.Fatal("size loss")
	}

	time.Sleep(time.Second * 2)
	if c.Size() != 4 {
		t.Fatal("expired failure", c.Size())
	}

	if err := c.Add("t5", 5, time.Second*3); err != nil {
		t.Fatal(err)
	}

	if err := c.Add("t5", 5, 0); err == nil {
		t.Fatal("add failure")
	}

	c.Range(func(k, v interface{}) bool {
		if k.(string) == "t2" {
			return false
		}
		return true
	})

	v, e, ok := c.GetWithExpire("t5")
	if !ok {
		t.Fatal("Add err not found")
	}
	if v.(int) != 5 || e.Nanoseconds() <= 0 {
		t.Fail()
	}

	opt.MaxSize = 2
	opt.OverSizeClearMode = VolatileRandomMode
	c, _ = NewCache(opt)
	c.Flush()
	if err := c.Set("t1", 1); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("t2", 1); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("t3", 1); err == nil {
		t.Fatal("t3 error")
	}
	c.Delete("t2")
	if err := c.SetExpire("t2", 1, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("t3", 1); err != nil {
		t.Fatal("t3 error")
	}

	if c.Size() > 2 {
		t.Fatal("size err", c.Size())
	}

	opt.MaxSize = 2
	opt.OverSizeClearMode = AllKeysRandomMode
	c, _ = NewCache(opt)
	c.Flush()
	if err := c.Set("t1", 1); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("t2", 1); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("t3", 1); err != nil {
		t.Fatal(err)
	}
	if c.Size() > 2 {
		t.Fatal("size err", c.Size())
	}
}
