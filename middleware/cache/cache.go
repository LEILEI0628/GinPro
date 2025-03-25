package cachex

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"time"
)

// ErrKeyNotExist 表示键不存在的错误
var ErrKeyNotExist = redis.Nil

// KeyFunc 定义生成缓存键的函数
type KeyFunc[K comparable] func(id K) string

// Cache 泛型缓存结构，支持任意值类型和可比较的键类型
type Cache[T any, K comparable] struct { // T：泛型（值类型），K：可比较类型（键类型，生成键函数参数）
	// A用到了B，B应当是接口；A用到了B，B应当是A的字段；A用到了B，A绝对不初始化B，而是通过外部注入
	// 传单机或cluster Redis都可以
	// 对外隐藏内部的实现
	client     redis.Cmdable // Redis客户端
	expiration time.Duration // 缓存过期时间
	keyFunc    KeyFunc[K]    // 键生成函数
}

// NewCache 创建新的通用缓存实例
func NewCache[T any, K comparable](
	client redis.Cmdable,
	expiration time.Duration,
	keyFunc KeyFunc[K],
) *Cache[T, K] {
	// 不要在这里初始化！（传入配置或从系统获取都不要）
	return &Cache[T, K]{
		client:     client,
		expiration: expiration,
		keyFunc:    keyFunc,
	}
}

// Get 从缓存获取值
func (cache *Cache[T, K]) Get(context context.Context, id K) (T, error) {
	var value T
	key := cache.keyFunc(id)
	data, err := cache.client.Get(context, key).Bytes()

	if err != nil {
		return value, err // 返回包含ErrKeyNotExist（数据不存在），也可能为其他错误（偶发或崩溃）
	}

	err = json.Unmarshal(data, &value)
	//if err != nil {
	//	return value, err
	//}
	return value, nil // 此处为简写：有err则user一定为空
}

// Set 将值存入缓存
func (cache *Cache[T, K]) Set(context context.Context, id K, value T) error {
	key := cache.keyFunc(id)
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cache.client.Set(context, key, data, cache.expiration).Err()
}
