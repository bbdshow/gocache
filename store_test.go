package gocache

import "testing"

func TestRWMap(t *testing.T) {
	rwMap := NewRWMap()

	rwMap.Store("store", "1")

	v, ok := rwMap.Load("store")
	if !ok {
		t.Fatal("store key not exists")
		return
	}
	if v.(string) != "1" {
		t.Fatal("store value not equal")
		return
	}

	ac, exists := rwMap.LoadOrStore("store", "2")
	if !exists {
		t.Fatal("store should exists")
		return
	}

	if ac.(string) != "1" {
		t.Fatal("store value should = 1")
		return
	}

	ac, exists = rwMap.LoadOrStore("restore", "2")
	if exists {
		t.Fatal("store should not exists")
		return
	}

	if ac.(string) != "2" {
		t.Fatal("store value should = 2")
		return
	}

	rwMap.Range(func(k string, v interface{}) bool {
		switch k {
		case "store", "restore":
		default:
			t.Fatal("key error ", k)
			return false
		}

		return true
	})
	count := 0
	rwMap.Range(func(k string, v interface{}) bool {
		count++
		return false
	})

	if count != 1 {
		t.Fatal("range break error")
	}

	rwMap.Delete("store")

	_, ok = rwMap.Load("store")
	if ok {
		t.Fatal("store key should not exists")
		return
	}
}
