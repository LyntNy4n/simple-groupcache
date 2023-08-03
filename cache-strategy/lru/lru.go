package lru

// lru 包实现了使用最近最久未使用使用算法的缓存功能
// 用于cache内存不足情况下 移除相应缓存记录
// Warning: lru包不提供并发一致机制

import (
	"container/list"
	cachestrategy "simple-groupcache/cache-strategy"
)

// Node 定义双向链表节点所存储的对象
// 在链表中仍保存每个值对应的 key 的好处在于，淘汰队首节点时，需要用 key 从字典中删除对应的映射
type Node struct {
	key   string
	value cachestrategy.Lengthable
}

// Cache 是LRU算法实现的缓存
// 参考Leetcode使用哈希表+双向链表实现LRU
type Cache struct {
	MaxByte          int64 // Cache 最大容量(Byte)
	CurrByte         int64 // Cache 当前容量(Byte)
	hashmap          map[string]*list.Element
	doublyLinkedList *list.List // 链头表示最近使用

	callback cachestrategy.OnEvicted // 淘汰回调
}

// 确保Cache实现了CacheStrategy接口
var _ cachestrategy.CacheStrategy = (*Cache)(nil)

// New 创建指定最大容量的LRU缓存
// 当maxBytes为0时，代表cache无内存限制，无限存放
func New(maxBytes int64, callback cachestrategy.OnEvicted) *Cache {
	return &Cache{
		MaxByte:          maxBytes,
		hashmap:          make(map[string]*list.Element),
		doublyLinkedList: list.New(),
		callback:         callback,
	}
}

// Len 返回当前缓存元素个数
func (c *Cache) Len() int64 {
	return int64(c.doublyLinkedList.Len())
}

// Contains,看看key存不存在(不访问)
func (c *Cache) Contains(key string) bool {
	if _, ok := c.hashmap[key]; ok {
		return true
	}
	return false
}

// Get 从缓存获取对应key的value
// ok 指明查询结果 false代表查无此key
func (c *Cache) Get(key string) (value cachestrategy.Lengthable, ok bool) {
	if elem, ok := c.hashmap[key]; ok {
		c.doublyLinkedList.MoveToFront(elem)
		entry := elem.Value.(*Node)
		return entry.value, true
	}
	return
}

// Add 向缓存添加/更新一枚key-value
func (c *Cache) Add(key string, value cachestrategy.Lengthable) {
	kvSize := int64(len(key)) + int64(value.Len())
	// cache 容量检查
	for c.MaxByte != 0 && c.CurrByte+kvSize > c.MaxByte {
		c.Evict()
	}
	if elem, ok := c.hashmap[key]; ok {
		// 更新缓存key值
		c.doublyLinkedList.MoveToFront(elem)
		oldEntry := elem.Value.(*Node)
		// 先更新写入字节 再更新
		c.CurrByte += int64(value.Len()) - int64(oldEntry.value.Len())
		oldEntry.value = value
	} else {
		// 新增缓存key
		elem := c.doublyLinkedList.PushFront(&Node{key: key, value: value})
		c.hashmap[key] = elem
		c.CurrByte += kvSize
	}
}

// Evict 淘汰一枚最近最不常用缓存
func (c *Cache) Evict() (string, cachestrategy.Lengthable) {
	tailElem := c.doublyLinkedList.Back()
	if tailElem != nil {
		entry := tailElem.Value.(*Node)
		k, v := entry.key, entry.value
		delete(c.hashmap, k)                         // 移除映射
		c.doublyLinkedList.Remove(tailElem)          // 移除缓存
		c.CurrByte -= int64(len(k)) + int64(v.Len()) // 更新占用内存情况
		// 移除后的善后处理
		if c.callback != nil {
			c.callback(k, v)
		}
		return k, v
	}
	return "", nil
}

// Remove 删除特定key的数据
func (c *Cache) Remove(key string) {
	if elem, ok := c.hashmap[key]; ok {
		entry := elem.Value.(*Node)
		delete(c.hashmap, key)
		c.doublyLinkedList.Remove(elem)
		c.CurrByte -= int64(len(key)) + int64(entry.value.Len())
	}
}
