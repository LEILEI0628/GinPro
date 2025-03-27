package cachex

import "context"

// Cache 通用缓存接口
// TODO 日志记录及错误返回处理
type Cache[T any, K comparable] interface {
	Get(ctx context.Context, key K) (T, error)
	Set(ctx context.Context, key K, value T) error
	Delete(ctx context.Context, key K) error
}
