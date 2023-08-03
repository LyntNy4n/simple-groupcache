package singlefilght

import (
	"sync"
)

// singlefilght 为节点提供缓存击穿的保护
// 当cache并发访问peer获取缓存时 如果peer未缓存该值
// 则会向db发送大量的请求获取 造成db的压力骤增
// 因此 将所有由key产生的请求抽象成flight
// 这个flight只会起飞一次(single) 这样就可以缓解击穿的可能性
// flight载有我们要的缓存数据 称为packet

type packet struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Flight struct {
	mu     sync.Mutex
	flight map[string]*packet
}

// Fly 负责key航班的飞行 fn是获取packet的方法
func (f *Flight) Fly(key string, fn func() (interface{}, error)) (interface{}, error) {
	f.mu.Lock()
	// 结构未初始化
	if f.flight == nil {
		f.flight = make(map[string]*packet)
	}
	// 航班已起飞(已缓存该key的数据) 则等待
	if p, ok := f.flight[key]; ok {
		f.mu.Unlock()
		p.wg.Wait() // 等待航班完成
		return p.val, p.err
	}

	// 航班未起飞(未缓存该key的数据) 则创建packet
	p := new(packet)
	p.wg.Add(1)
	f.flight[key] = p
	f.mu.Unlock()
	// 创建packet后,航班起飞(获取数据)
	p.val, p.err = fn()
	p.wg.Done() // 航班完成

	f.mu.Lock()
	delete(f.flight, key)
	f.mu.Unlock()
	return p.val, p.err
}
