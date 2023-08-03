package lfu

import (
	"container/list"
	cachestrategy "simple-groupcache/cache-strategy"
)

// Node 定义双向链表节点所存储的对象
// 其实key和freq可以不用存储在Node中，但是为了一些情况下查找方便，就存了
type Node struct {
	key   string
	value cachestrategy.Lengthable
	freq  int
}

// Cache 是LFU算法实现的缓存
type Cache struct {
	MaxByte  int64                    // Cache 最大容量(Byte)
	CurrByte int64                    // Cache 当前容量(Byte)
	kvMap    map[string]*list.Element // key对应的双向链表节点
	freqMap  map[int]*list.List       // 频率对应的双向链表,链头表示最近使用
	minFreq  int                      // 最小频率

	callback cachestrategy.OnEvicted // 淘汰回调
}

var _ cachestrategy.CacheStrategy = (*Cache)(nil)

// New 创建指定最大容量的LFU缓存
// 当maxBytes为0时，代表cache无内存限制，无限存放
func New(maxBytes int64, callback cachestrategy.OnEvicted) *Cache {
	return &Cache{
		MaxByte:  maxBytes,
		kvMap:    make(map[string]*list.Element),
		freqMap:  make(map[int]*list.List),
		callback: callback,
	}
}

// Len 返回当前缓存元素个数
func (c *Cache) Len() int64 {
	return int64(len(c.kvMap))
}

func (c *Cache) Contains(key string) bool {
	if _, ok := c.kvMap[key]; ok {
		return true
	}
	return false
}

// Get 从缓存获取对应key的value
// ok 指明查询结果 false代表查无此key
func (c *Cache) Get(key string) (value cachestrategy.Lengthable, ok bool) {
	if elem, ok := c.kvMap[key]; ok {
		c.updateFreq(elem)
		node := elem.Value.(*Node)
		return node.value, true
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

	if elem, ok := c.kvMap[key]; ok {
		node := elem.Value.(*Node)
		c.CurrByte += int64(value.Len()) - int64(node.value.Len())
		node.value = value
		c.updateFreq(elem)
	} else {
		// 新增缓存key值
		node := &Node{key: key, value: value, freq: 1}
		// 如果freq为1的链表不存在，创建该链表
		if _, ok := c.freqMap[1]; !ok {
			c.freqMap[1] = list.New()
		}
		elem := c.freqMap[1].PushFront(node)
		c.kvMap[key] = elem
		// 更新minFreq
		c.minFreq = 1
		// 更新写入字节
		c.CurrByte += kvSize
	}
}

// Evict 淘汰一枚最低频率的缓存,如果次数相同,则淘汰最早的数据
func (c *Cache) Evict() (string, cachestrategy.Lengthable) {
	if len(c.kvMap) == 0 {
		return "", nil
	}
	// 获取最低频率的链表
	lowFreqList := c.freqMap[c.minFreq]
	// 获取最低频率链表的最后一个节点
	node := lowFreqList.Back().Value.(*Node)
	// 删除最低频率链表的最后一个节点
	lowFreqList.Remove(lowFreqList.Back())
	// 删除kvMap中对应的key
	delete(c.kvMap, node.key)
	// 更新写入字节
	c.CurrByte -= int64(len(node.key)) + int64(node.value.Len())
	// 如果最低频率链表为空，删除该链表
	if lowFreqList.Len() == 0 {
		delete(c.freqMap, c.minFreq)
	}
	// 执行淘汰回调
	if c.callback != nil {
		c.callback(node.key, node.value)
	}
	return node.key, node.value
}

func (c *Cache) updateFreq(elem *list.Element) {
	node := elem.Value.(*Node)
	// 将节点从原来的freq对应的链表中删除
	c.freqMap[node.freq].Remove(elem)
	// 更新节点的freq
	node.freq++
	// 如果新的freq对应的链表不存在，创建该链表
	if _, ok := c.freqMap[node.freq]; !ok {
		c.freqMap[node.freq] = list.New()
	}
	// 将节点插入到新的freq对应的链表中
	c.freqMap[node.freq].PushFront(elem)
	// 如果原来的freq对应的链表为空，删除该链表
	if c.freqMap[node.freq].Len() == 0 {
		delete(c.freqMap, node.freq)
	}
	// 更新minFreq
	if c.minFreq == node.freq-1 {
		c.minFreq++
	}
}

func (c *Cache) Remove(key string) {
	if elem, ok := c.kvMap[key]; ok {
		node := elem.Value.(*Node)
		c.freqMap[node.freq].Remove(elem)
		delete(c.kvMap, node.key)
		c.CurrByte -= int64(len(node.key)) + int64(node.value.Len())
		if c.freqMap[node.freq].Len() == 0 {
			delete(c.freqMap, node.freq)
			if len(c.freqMap) == 0 {
				c.minFreq = 0
				return
			}
			// 更新minFreq
			newFreq := node.freq + 1
			for {
				if _, ok := c.freqMap[newFreq]; ok {
					c.minFreq = newFreq
					break
				}
				newFreq++
			}
		}
	}
}
