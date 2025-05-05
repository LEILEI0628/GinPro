package cachex

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"sync"
	"time"
)

// MultiLevelCache 多级缓存实现
type MultiLevelCache[T any, K comparable] struct {
	localCache   *LocalCache[T, K]                   // 本地缓存
	redisClient  redis.Cmdable                       // Redis客户端
	loadFunc     func(context.Context, K) (T, error) // 数据加载函数
	singleFlight singleflight.Group                  // 防止缓存击穿
	keyToString  func(K) string                      // Key转换函数

	mu           sync.RWMutex
	redisEnabled bool   // Redis可用状态
	config       Config // 配置参数
}

type LocalCacheV1[T any, K comparable] interface {
	Get(key K) (T, bool)
	Set(key K, value T)
	Delete(key K)
}

type Config struct {
	LocalCacheSize        int
	RedisTTL              time.Duration
	DegradeThreshold      int           // 降级阈值（错误次数）
	RecoveryCheckInterval time.Duration // 状态检查间隔
}

// NewMultiLevelCache 创建多级缓存
func NewMultiLevelCache[T any, K comparable](
	redisClient redis.Cmdable,
	loadFunc func(context.Context, K) (T, error),
	keyToString func(K) string,
	config Config,
) *MultiLevelCache[T, K] {
	lc := NewLocalCache[T, K](config.LocalCacheSize, LRU)

	mlc := &MultiLevelCache[T, K]{
		localCache:   lc,
		redisClient:  redisClient,
		loadFunc:     loadFunc,
		keyToString:  keyToString,
		redisEnabled: true,
		config:       config,
	}

	go mlc.healthCheck()
	return mlc
}

// Get 实现缓存获取逻辑
func (c *MultiLevelCache[T, K]) Get(ctx context.Context, key K) (T, error) {
	var zero T

	// 1. 尝试本地缓存
	if val, err := c.localCache.Get(ctx, key); err == nil {
		return val, nil
	}

	// 2. 尝试Redis（如果未降级）
	if c.isRedisEnabled() {
		redisKey := c.keyToString(key)
		val, err := c.redisClient.Get(ctx, redisKey).Result()
		if err == nil {
			// 反序列化并更新本地缓存
			var parsedVal T
			err = json.Unmarshal([]byte(val), &parsedVal)
			if err == nil {
				err = c.localCache.Set(ctx, key, parsedVal)
				if err != nil {
					// 更新本地缓存失败
					// TODO 记录日志
				}
				return parsedVal, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			// Redis错误计数
			// TODO 记录日志
		}
	}

	// 3. 使用SingleFlight加载数据
	keyStr := c.keyToString(key)
	result, err, _ := c.singleFlight.Do(keyStr, func() (interface{}, error) {
		// 调用加载函数获取数据
		val, err := c.loadFunc(ctx, key)
		if err != nil {
			return zero, err
		}

		// 回填缓存
		c.Set(ctx, key, val)
		return val, nil
	})

	if err != nil {
		return zero, err
	}
	return result.(T), nil
}

// Set 更新缓存
func (c *MultiLevelCache[T, K]) Set(ctx context.Context, key K, value T) error {
	// 1. 更新本地缓存
	c.localCache.Set(ctx, key, value)

	// 2. 异步更新Redis（如果可用）
	if c.isRedisEnabled() {
		go func() {
			redisKey := c.keyToString(key)
			serialized, _ := json.Marshal(value)
			_, err := c.redisClient.Set(ctx, redisKey, serialized, c.config.RedisTTL).Result()
			if err != nil {
				// TODO 记录日志
			}
		}()
	}
	return nil
}

// Delete 删除缓存
func (c *MultiLevelCache[T, K]) Delete(ctx context.Context, key K) error {
	// 1. 删除本地缓存
	c.localCache.Delete(ctx, key)

	// 2. 异步删除Redis
	if c.isRedisEnabled() {
		go func() {
			redisKey := c.keyToString(key)
			c.redisClient.Del(ctx, redisKey)
		}()
	}
	return nil
}

// 状态检查相关方法
func (c *MultiLevelCache[T, K]) isRedisEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.redisEnabled
}

func (c *MultiLevelCache[T, K]) healthCheck() {
	ticker := time.NewTicker(c.config.RecoveryCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		if _, err := c.redisClient.Ping(context.Background()).Result(); err == nil {
			c.mu.Lock()
			c.redisEnabled = true
			c.mu.Unlock()
		}
	}
}
