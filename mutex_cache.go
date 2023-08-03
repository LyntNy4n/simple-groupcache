package simplegroupcache

// cache 模块负责提供对lru模块的并发控制

import (
	cachestrategy "simple-groupcache/cache-strategy"
	"simple-groupcache/cache-strategy/arc"
	"simple-groupcache/cache-strategy/lfu"
	"simple-groupcache/cache-strategy/lru"
	"sync"
)

// 这样设计可以进行mutexCache和算法的分离，比如我现在实现了lfu缓存模块
// 只需替换mutexCache成员即可
type mutexCache struct {
	mu       sync.Mutex
	cache    cachestrategy.CacheStrategy
	capacity int64 // 缓存最大容量
}

func newCache(capacity int64, cacheStrategy string) *mutexCache {
	var cache cachestrategy.CacheStrategy
	switch cacheStrategy {
	case "lru":
		cache = lru.New(capacity, nil)
	case "lfu":
		cache = lfu.New(capacity, nil)
	case "arc":
		cache = arc.New(capacity, nil)
	default:
		cache = lru.New(capacity, nil)
	}
	return &mutexCache{
		cache:    cache,
		capacity: capacity,
	}
}

func (c *mutexCache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Add(key, value)
}

func (c *mutexCache) get(key string) (ByteView, bool) {
	if c.cache == nil {
		return ByteView{}, false
	}
	// 注意：Get操作需要修改lru中的双向链表，需要使用互斥锁。
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.cache.Get(key); ok {
		return v.(ByteView), true
	}
	return ByteView{}, false
}
