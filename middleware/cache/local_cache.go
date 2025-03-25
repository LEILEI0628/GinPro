package cachex

import (
	"container/list"
	"context"
	"sync"
)

type PolicyType string

const (
	LRU PolicyType = "lru"
	LFU PolicyType = "lfu"
)

// EvictionPolicy 缓存淘汰策略接口
type EvictionPolicy[K comparable, T any] interface {
	// Add 添加新条目到策略跟踪
	Add(key K, value T)

	// Access 记录条目被访问
	Access(key K)

	// Evict 执行淘汰并返回被淘汰的键
	// 返回值：被淘汰的键和是否存在有效淘汰
	Evict() (K, bool)

	// Remove 移除指定条目
	Remove(key K)
}

// LocalCache 本地通用缓存实现
// 示例：创建LRU缓存：lruCache := NewLocalCache[string, int](1000, LRU)
// 创建LFU缓存：lfuCache := NewLocalCache[string, int](500, LFU)
type LocalCache[T any, K comparable] struct {
	mu       sync.RWMutex         // 读写锁保证并发安全
	store    map[K]T              // 实际数据存储
	policy   EvictionPolicy[K, T] // 淘汰策略实现
	capacity int                  // 最大容量限制
}

// NewLocalCache 创建支持不同淘汰策略的本地缓存
// capacity: 缓存容量
// policyType: 策略类型 "lru" 或 "lfu"
func NewLocalCache[T any, K comparable](capacity int, policyType PolicyType) *LocalCache[T, K] {
	if capacity <= 0 {
		capacity = 1000 // 设置默认容量防止无效值
	}

	c := &LocalCache[T, K]{
		store:    make(map[K]T),
		capacity: capacity,
	}

	// 根据策略类型初始化对应的淘汰策略
	switch policyType {
	case LFU:
		c.policy = newLFUPolicy[K, T]()
	default: // 默认为LRU策略
		c.policy = newLRUPolicy[K, T](capacity)
	}

	return c
}

// Get 获取缓存值
// 返回值：缓存值和是否存在标记
// 示例：val, ok := lruCache.Get(1)
func (cache *LocalCache[T, K]) Get(ctx context.Context, key K) (T, error) {
	cache.mu.RLock() // 读锁保护并发读取
	defer cache.mu.RUnlock()

	value, exists := cache.store[key]
	if exists {
		cache.policy.Access(key) // 记录访问事件
		return value, nil
	}
	return value, ErrKeyNotExist
}

// Set 设置缓存值
// 当缓存达到容量限制时触发淘汰策略
// 示例：lruCache.Get(1,"A")
func (cache *LocalCache[T, K]) Set(ctx context.Context, key K, value T) error {
	cache.mu.Lock() // 写锁保证互斥访问
	defer cache.mu.Unlock()

	if _, exists := cache.store[key]; exists {
		// 更新现有值
		cache.store[key] = value
		cache.policy.Access(key) // 记录访问
		return nil
	}

	// 执行淘汰检查
	if len(cache.store) >= cache.capacity {
		if evictedKey, ok := cache.policy.Evict(); ok { // 触发淘汰策略
			delete(cache.store, evictedKey)
		}
	}
	// 添加新条目
	cache.store[key] = value
	cache.policy.Add(key, value)
	return nil
}

// Delete 显式删除缓存项
// 用于主动失效缓存内容
func (cache *LocalCache[T, K]) Delete(ctx context.Context, key K) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	delete(cache.store, key)
	cache.policy.Remove(key)
	return nil
}

// LRU策略实现

// lruItem LRU缓存条目结构
type lruItem[K comparable, T any] struct {
	key   K
	value T
}

// lruPolicy LRU淘汰策略实现
type lruPolicy[K comparable, T any] struct {
	capacity int                 // 最大容量
	ll       *list.List          // 双向链表维护访问顺序
	elements map[K]*list.Element // 键到链表元素的映射
}

// newLRUPolicy 创建LRU策略实例
func newLRUPolicy[K comparable, T any](capacity int) EvictionPolicy[K, T] {
	return &lruPolicy[K, T]{
		capacity: capacity,
		ll:       list.New(), // 初始化链表
		elements: make(map[K]*list.Element),
	}
}

func (p *lruPolicy[K, T]) Add(key K, value T) {
	if elem, exists := p.elements[key]; exists {
		// 已存在则更新值并移动到链表前端
		p.ll.MoveToFront(elem)
		elem.Value.(*lruItem[K, T]).value = value
		return
	}

	// 创建新条目并添加到链表前端
	newElem := p.ll.PushFront(&lruItem[K, T]{key, value})
	p.elements[key] = newElem
}

// Access 处理访问事件
func (p *lruPolicy[K, T]) Access(key K) {
	if elem, exists := p.elements[key]; exists {
		p.ll.MoveToFront(elem) // 移动到前端表示最近使用
	}
}

// Evict 执行淘汰操作
func (p *lruPolicy[K, T]) Evict() (K, bool) {
	if p.ll.Len() == 0 {
		var zero K
		return zero, false
	}

	// 淘汰链表末尾元素（最近最少使用）
	oldest := p.ll.Back()
	if oldest != nil {
		key := oldest.Value.(*lruItem[K, T]).key
		p.ll.Remove(oldest)
		delete(p.elements, key)
		return key, true
	}
	var zero K
	return zero, false
}

// Remove 移除指定条目
func (p *lruPolicy[K, T]) Remove(key K) {
	if elem, exists := p.elements[key]; exists {
		p.ll.Remove(elem)
		delete(p.elements, key)
	}
}

// LFU策略实现
// lfuItem LFU缓存条目结构
type lfuItem[K comparable, T any] struct {
	key       K
	value     T
	frequency int // 访问频率计数器
}

// lfuPolicy LFU淘汰策略实现
type lfuPolicy[K comparable, T any] struct {
	minFreq  int                 // 当前最小频率
	elements map[K]*list.Element // 键到元素的映射
	freqMap  map[int]*list.List  // 频率到链表的映射
}

// newLFUPolicy 创建LFU策略实例
func newLFUPolicy[K comparable, T any]() EvictionPolicy[K, T] {
	return &lfuPolicy[K, T]{
		elements: make(map[K]*list.Element),
		freqMap:  make(map[int]*list.List),
	}
}

// Add 添加或更新条目
func (p *lfuPolicy[K, T]) Add(key K, value T) {
	if elem, exists := p.elements[key]; exists {
		// 已存在则更新值并增加频率
		item := elem.Value.(*lfuItem[K, T])
		item.value = value
		p.increment(elem)
		return
	}

	// 新建条目（初始频率为1）
	newItem := &lfuItem[K, T]{
		key:       key,
		value:     value,
		frequency: 1,
	}

	// 初始化频率链表
	if p.freqMap[1] == nil {
		p.freqMap[1] = list.New()
	}
	newElem := p.freqMap[1].PushFront(newItem)
	p.elements[key] = newElem
	p.minFreq = 1 // 重置最小频率
}

// Access 处理访问事件
func (p *lfuPolicy[K, T]) Access(key K) {
	if elem, exists := p.elements[key]; exists {
		p.increment(elem) // 增加访问频率
	}
}

// Evict 执行淘汰操作
func (p *lfuPolicy[K, T]) Evict() (K, bool) {
	minList := p.freqMap[p.minFreq]
	if minList == nil || minList.Len() == 0 {
		var zero K
		return zero, false
	}

	// 淘汰最小频率链表的最后一个元素
	back := minList.Back()
	if back != nil {
		key := back.Value.(*lfuItem[K, T]).key
		minList.Remove(back)
		delete(p.elements, key)

		// 清理空链表
		if minList.Len() == 0 {
			delete(p.freqMap, p.minFreq)
		}
		return key, true
	}
	var zero K
	return zero, false
}

// Remove 移除指定条目
func (p *lfuPolicy[K, T]) Remove(key K) {
	if elem, exists := p.elements[key]; exists {
		item := elem.Value.(*lfuItem[K, T])
		// 从频率链表中移除
		p.freqMap[item.frequency].Remove(elem)
		// 清理空链表
		if p.freqMap[item.frequency].Len() == 0 {
			delete(p.freqMap, item.frequency)
		}
		delete(p.elements, key)
	}
}

// increment 增加条目访问频率
func (p *lfuPolicy[K, T]) increment(elem *list.Element) {
	item := elem.Value.(*lfuItem[K, T])

	// 从原频率链表移除
	oldList := p.freqMap[item.frequency]
	oldList.Remove(elem)

	// 清理空链表
	if oldList.Len() == 0 {
		delete(p.freqMap, item.frequency)
		// 更新最小频率（如果淘汰的是当前最小频率的最后一个元素）
		if p.minFreq == item.frequency {
			p.minFreq++
		}
	}

	// 提升频率并插入新链表
	item.frequency++
	if p.freqMap[item.frequency] == nil {
		p.freqMap[item.frequency] = list.New()
	}
	newList := p.freqMap[item.frequency]
	newElem := newList.PushFront(item)
	p.elements[item.key] = newElem
}
