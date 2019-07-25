package gocache

type Cache interface {
	Get(key string) (value interface{}, exists bool)
	GetWithExpire(key string) (value interface{}, expire int64, exists bool)

	Set(key string, value interface{}) error
	SetWithExpire(key string, value interface{}, expire int64) error

	//Keys() Keys

	Delete(key string)

	// 删除所有 key
	FlushAll() error

	Close() error
}

//type Keys interface {
//	Size() int64
//	Value() []string
//}
