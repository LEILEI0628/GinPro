package limiter

import "context"

type Limiter interface {
	// Limit 是否触发限流
	// bool：true触发限流，error：限流器本身有无错误
	Limit(ctx context.Context, key string) (bool, error)
}
