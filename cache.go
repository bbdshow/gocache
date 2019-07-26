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
	FlushAll() error

	Close() error
}

type Keys interface {
	Size() int64
	Value() []string
}
