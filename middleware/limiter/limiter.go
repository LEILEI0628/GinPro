package limiter

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type Limiter interface {
	// Limit 是否触发限流
	// bool：true触发限流，error：限流器本身有无错误
	Limit(ctx context.Context, key string) (bool, error)
}

type KeyType string

const (
	IP  KeyType = "ip"
	UID KeyType = "uid"
)

type Builder struct {
	prefix  string  // 前缀
	keyType KeyType // 限流key类型
	limiter Limiter
}

func NewBuilder(l Limiter) *Builder {
	return &Builder{
		prefix:  "ip-limiter",
		keyType: IP,
		limiter: l,
	}
}

func (b *Builder) Prefix(prefix string) *Builder {
	b.prefix = prefix
	return b
}

func (b *Builder) KeyType(keyType KeyType) *Builder {
	b.keyType = keyType
	return b
}

func (b *Builder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var key string
		switch b.keyType {
		//case UID:
		//	key = ctx.GetString("UID")
		default:
			// 默认使用ip限流器
			key = ctx.ClientIP()
		}
		limited, err := b.limiter.Limit(ctx, fmt.Sprintf("%s:%s", b.prefix, key))
		if err != nil {
			log.Println(err)
			// Redis出错
			// 保守做法：因为借助Redis限流，所以Redis崩溃后为了防止系统崩溃直接限流（下游处理能力较差时）
			ctx.AbortWithStatus(http.StatusInternalServerError)
			// 激进做法：虽然Redis崩溃了，但为了尽量服务正常的用户，所以不限流（可用性要求很高或下游服务处理能力很强时）
			// ctx.Next()
			return
		}
		if limited {
			log.Println(err)
			ctx.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		ctx.Next()
	}
}
