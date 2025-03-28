package limiter

import (
	"context"
	_ "embed"
	"github.com/redis/go-redis/v9"
	"time"
)

//go:embed redis_slide_window.lua
var luaScript string

// RedisSlidingWindowLimiter 基于Redis的滑动窗口算法限流器
type RedisSlidingWindowLimiter struct {
	cmd      redis.Cmdable
	interval time.Duration // 窗口大小
	rate     int           // 阈值（interval内允许rate个请求）
}

func NewRedisSlidingWindowLimiter(cmd redis.Cmdable, interval time.Duration, rate int) *RedisSlidingWindowLimiter {
	return &RedisSlidingWindowLimiter{
		cmd:      cmd,
		interval: interval,
		rate:     rate,
	}
}

func (limiter *RedisSlidingWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {
	return limiter.cmd.Eval(ctx, luaScript,
		[]string{key}, limiter.interval.Milliseconds(), limiter.rate, time.Now().UnixMilli()).Bool()
}
