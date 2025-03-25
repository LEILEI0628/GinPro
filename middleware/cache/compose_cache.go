package cachex

import (
	"context"
	"errors"
	"time"
)

// TwoLevelCache 二级缓存组合结构
// T: 缓存值类型，K: 可比较的键类型
type TwoLevelCache[T any, K comparable] struct {
	local   *LocalCache[T, K] // 本地内存缓存（一级缓存）
	remote  *RedisCache[T, K] // Redis远程缓存（二级缓存）
	timeout time.Duration     // 远程操作超时时间
}

// NewTwoLevelCache 创建二级缓存实例
// local: 本地缓存实现（如LRU/LFU）
// remote: Redis缓存实例
// timeout: 远程操作超时时间（推荐500ms-1s）
func NewTwoLevelCache[T any, K comparable](
	local *LocalCache[T, K],
	remote *RedisCache[T, K],
	timeout time.Duration,
) *TwoLevelCache[T, K] {
	return &TwoLevelCache[T, K]{
		local:   local,
		remote:  remote,
		timeout: timeout,
	}
}

// Get 二级缓存读取策略：
// 1. 优先读取本地缓存
// 2. 本地未命中则查询远程缓存
// 3. 远程命中后回填本地缓存
// 4. 双重未命中返回ErrKeyNotExist
func (c *TwoLevelCache[T, K]) Get(ctx context.Context, id K) (T, error) {
	// 第一步：尝试本地缓存
	val, err := c.local.Get(ctx, id)
	if err == nil {
		return val, nil
	}

	// 第二步：查询远程缓存（带超时控制）
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	remoteVal, err := c.remote.Get(ctx, id)
	if err != nil {
		var zero T
		if errors.Is(err, ErrKeyNotExist) {
			return zero, ErrKeyNotExist // 透传未命中错误
		}
		return zero, err // 返回其他查询错误
	}

	// 第三步：回填本地缓存
	err = c.local.Set(ctx, id, remoteVal)
	if err != nil {
		// 记录日志回填出错
	}
	return remoteVal, nil
}

// Set 二级缓存写入策略：
// 1. 同步更新本地缓存
// 2. 异步更新远程缓存（最终一致）
// 3. 快速返回不等待远程操作
func (c *TwoLevelCache[T, K]) Set(ctx context.Context, id K, value T) error {
	// 同步更新本地缓存
	err := c.local.Set(ctx, id, value)
	if err != nil {
		// 记录日志同步出错
	}

	// 异步更新远程缓存（非阻塞）
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()

		// 忽略错误处理（添加日志记录）
		_ = c.remote.Set(ctx, id, value)
	}()

	return nil
}

// Delete 组合缓存删除策略：
// 1. 同步删除本地缓存（立即生效）
// 2. 异步删除远程缓存（最终一致）
// 3. 快速返回不等待远程操作
func (c *TwoLevelCache[T, K]) Delete(ctx context.Context, id K) error {
	// 同步删除本地缓存
	err := c.local.Delete(ctx, id)
	if err != nil {
		// 删除本地缓存失败
	}

	// 异步删除远程缓存（非阻塞）
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()

		// 直接调用Delete方法删除远程键
		if err := c.remote.Delete(ctx, id); err != nil {
			// 添加日志记录
			// log.Printf("远程缓存删除失败 key=%v: %v", id, err)
		}
	}()

	return nil
}
