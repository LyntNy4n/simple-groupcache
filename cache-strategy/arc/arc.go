package arc

import (
	cachestrategy "simple-groupcache/cache-strategy"
	"simple-groupcache/cache-strategy/lfu"
	"simple-groupcache/cache-strategy/lru"
)

type Cache struct {
	maxByte  int64
	part     int64 // lru和ghostLru占用字节数的偏好值, 越大表示更倾向于保留初次访问的数据
	lru      *lru.Cache
	ghostLru *lru.Cache
	lfu      *lfu.Cache
	ghostLfu *lfu.Cache

	callback cachestrategy.OnEvicted
}

var _ cachestrategy.CacheStrategy = (*Cache)(nil)

func New(maxByte int64, callback cachestrategy.OnEvicted) *Cache {
	return &Cache{
		maxByte:  maxByte,
		part:     0,
		lru:      lru.New(maxByte, nil),
		ghostLru: lru.New(maxByte, nil),
		lfu:      lfu.New(maxByte, nil),
		ghostLfu: lfu.New(maxByte, nil),
		callback: callback,
	}
}

// Len 返回当前缓存元素个数
func (c *Cache) Len() int64 {
	return c.lru.Len() + c.lfu.Len()
}

// Get 从缓存获取对应key的value
func (c *Cache) Get(key string) (value cachestrategy.Lengthable, ok bool) {
	if v, ok := c.lru.Get(key); ok {
		c.lru.Remove(key)
		c.lfu.Add(key, v)
		return v, ok
	}
	if v, ok := c.lfu.Get(key); ok {
		return v, ok
	}

	return
}

// Add 向缓存添加/更新一枚key-value
func (c *Cache) Add(key string, value cachestrategy.Lengthable) {
	// 如果lru有, 则移动到lfu
	if c.lru.Contains(key) {
		c.lru.Remove(key)
		c.lfu.Add(key, value)
		return
	}
	// 如果lfu有, 则更新lfu
	if c.lfu.Contains(key) {
		c.lfu.Add(key, value)
		return
	}
	// 看看ghostLru和ghostLfu有没有
	if c.ghostLru.Contains(key) {
		kvSize := int64(len(key)) + int64(value.Len())
		var delta int64 = kvSize
		if c.ghostLru.CurrByte < c.ghostLfu.CurrByte {
			delta = (c.ghostLfu.Len() / c.ghostLru.Len()) * kvSize
		}
		if c.part+delta < c.maxByte {
			c.part += delta
		} else {
			c.part = c.maxByte
		}

		if c.lru.CurrByte+c.lfu.CurrByte >= c.maxByte {
			c.replace(key)
		}

		c.ghostLru.Remove(key)
		c.lfu.Add(key, value)
		return
	}

	if c.ghostLfu.Contains(key) {
		kvSize := int64(len(key)) + int64(value.Len())
		var delta int64 = kvSize
		if c.ghostLru.CurrByte > c.ghostLfu.CurrByte {
			delta = (c.ghostLru.Len() / c.ghostLfu.Len()) * kvSize
		}
		if delta >= c.part {
			c.part = 0
		} else {
			c.part -= delta
		}

		if c.lru.CurrByte+c.lfu.CurrByte >= c.maxByte {
			c.replace(key)
		}

		c.ghostLfu.Remove(key)
		c.lfu.Add(key, value)
		return
	}

	// 四个缓存都没有,是全新数据
	// 先看看是否容量满了
	if c.lru.CurrByte+c.lfu.CurrByte >= c.maxByte {
		c.replace(key)
	}
	// 然后看是否需要调整
	if c.ghostLru.CurrByte > c.maxByte-c.part {
		c.ghostLru.Evict()
		if c.callback != nil {
			c.callback(key, value)
		}
	}
	if c.ghostLfu.CurrByte > c.part {
		c.ghostLfu.Evict()
		if c.callback != nil {
			c.callback(key, value)
		}
	}
	// 最后添加
	c.lru.Add(key, value)
}

func (c *Cache) replace(key string) {
	if c.lru.Len() > 0 && (c.lru.CurrByte > c.part || (c.ghostLfu.Contains(key) && c.lru.CurrByte == c.part)) {
		k, v := c.lru.Evict()
		c.ghostLru.Add(k, v)
	} else {
		k, v := c.lfu.Evict()
		c.ghostLfu.Add(k, v)
	}
}
