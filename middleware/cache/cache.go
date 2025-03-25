package cachex

import "context"

// Cache 通用缓存接口
// TODO 日志记录及错误返回处理
type Cache[T any, K comparable] interface {
	Get(context context.Context, key K) (T, error)
	Set(context context.Context, key K, value T) error
	Delete(context context.Context, key K) error
}
