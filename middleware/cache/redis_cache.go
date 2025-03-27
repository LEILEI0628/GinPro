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

// RedisCache 泛型缓存结构，支持任意值类型和可比较的键类型
type RedisCache[T any, K comparable] struct { // T：泛型（值类型），K：可比较类型（键类型，生成键函数参数）
	// A用到了B，B应当是接口（保证面向接口变成）；A用到了B，B应当是A的字段（规避缺乏扩展性的包变量，包方法）；
	//A用到了B，A绝对不初始化B，而是通过外部注入（保持依赖注入（DI）和依赖翻转（IOC））
	// 传单机或cluster Redis都可以
	// 对外隐藏内部的实现
	client     redis.Cmdable // Redis客户端
	expiration time.Duration // 缓存过期时间
	keyFunc    KeyFunc[K]    // 键生成函数
}

// NewRedisCache 创建新的通用缓存实例
func NewRedisCache[T any, K comparable](
	client redis.Cmdable,
	expiration time.Duration,
	keyFunc KeyFunc[K],
) *RedisCache[T, K] {
	// 不要在这里初始化！（传入配置或从系统获取都不要）
	// RedisCache只是用到了client，无需也不应关注client如何初始化或实现
	return &RedisCache[T, K]{
		client:     client,
		expiration: expiration,
		keyFunc:    keyFunc,
	}
}

// Get 从缓存获取值
func (cache *RedisCache[T, K]) Get(ctx context.Context, id K) (T, error) {
	var value T
	key := cache.keyFunc(id)
	data, err := cache.client.Get(ctx, key).Bytes()

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
func (cache *RedisCache[T, K]) Set(ctx context.Context, id K, value T) error {
	key := cache.keyFunc(id)
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cache.client.Set(ctx, key, data, cache.expiration).Err()
}

// Delete 删除Redis缓存项
// ctx: 上下文
// id: 缓存键
// 返回值: 删除操作错误（包含ErrKeyNotExist）
func (c *RedisCache[T, K]) Delete(ctx context.Context, id K) error {
	// 直接调用Redis DEL命令删除键
	// 键不存在返回redis.Nil错误，需根据业务需求处理
	return c.client.Del(ctx, c.keyFunc(id)).Err()
}
