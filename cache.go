package gocache

type Cache interface {
	Get(key string) (value interface{}, exists bool)                      //
	GetWithExpire(key string) (value interface{}, ttl int64, exists bool) // 返回值和剩余时间
	Set(key string, value interface{}) error                              //
	SetWithExpire(key string, value interface{}, ttl int64) error         // ttl 秒级别
	Keys(prefix string) Keys                                              // prefix - 前缀查询，"" 查询所有， 只返回当前有效的key
	Delete(key string)                                                    //
	Size() int64                                                          // 当前存储的数据量
	FlushAll()                                                            // 删除所有 key
	GobRegister(v ...interface{})                                         // 注册自定义结构体
	WriteToDisk() error                                                   // 写入数据到磁盘， 如果存在自定义结构类型，在使用时 一定要先注册结构
	LoadFromDisk() error                                                  // 从磁盘加载数据
	Close()                                                               //
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
	Size() int64
	Flush()
}
