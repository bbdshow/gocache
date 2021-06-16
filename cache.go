package gocache

type Cache interface {
	Get(key string) (value interface{}, exists bool)
	GetWithExpire(key string) (value interface{}, expire int64, exists bool)

	Set(key string, value interface{}) error
	SetWithExpire(key string, value interface{}, expire int64) error

	// prefix - 前缀查询，"" 查询所有， 只返回当前有效的key
	Keys(prefix string) Keys

	Delete(key string)

	// 当前存储的数据量
	Size() int64

	// 删除所有 key
	FlushAll()

	// 写入数据到磁盘
	WriteToDisk() error

	// 从磁盘加载数据
	LoadFromDisk() error

	Close()
}

type Keys interface {
	Size() int64
	Value() []string
}

type Store interface {
	Load(key string) (value interface{}, ok bool)
	Store(key string, value interface{})
	Delete(key string)
	LoadOrStore(key string, value interface{}) (actual interface{}, loaded bool)
	Exists(key string) bool
	Range(f func(k string, v interface{}) bool)
	Flush()
}
