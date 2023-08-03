package cachestrategy

// Lengthable 接口指明对象可以获取自身占有内存空间大小 以字节为单位
type Lengthable interface {
	Len() int
}

// OnEvicted 当key-value被淘汰时 执行的处理函数
type OnEvicted func(key string, value Lengthable)

type CacheStrategy interface {
	Get(key string) (value Lengthable, ok bool)
	Add(key string, value Lengthable)
	Len() int64
}
