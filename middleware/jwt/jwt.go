package jwtx

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

// Option 定义配置选项类型
type Option func(*Builder)

// Builder 使用结构体封装配置参数
type Builder struct {
	ignorePaths     map[string]string // 使用map提升查找性能
	verificationKey string
	expiresTime     time.Duration
	leftTime        time.Duration
}

// NewBuilder 默认/自定义配置
func NewBuilder(opts ...Option) *Builder {
	builder := &Builder{
		ignorePaths: make(map[string]string), // 初始化ignorePathsMap
		expiresTime: 24 * time.Hour,          // 默认过期时间
		leftTime:    1 * time.Hour,           // 默认续约时间
	}

	for _, opt := range opts {
		opt(builder)
	}
	return builder
}

// WithVerificationKey 校验key（Option配置函数）
func WithVerificationKey(key string) Option {
	return func(b *Builder) {
		b.verificationKey = key
	}
}

// WithExpiresTime 过期时间（Option配置函数）
func WithExpiresTime(d time.Duration) Option {
	return func(b *Builder) {
		b.expiresTime = d
	}
}

// WithLeftTime 刷新剩余时间（Option配置函数）
func WithLeftTime(d time.Duration) Option {
	return func(b *Builder) {
		b.leftTime = d
	}
}

// UserClaims 用户JWT Claims
type UserClaims struct {
	jwt.RegisteredClaims // 组合RegisteredClaims可以更简洁的实现Claims接口
	// 下列是自定义字段
	UID       int64
	Ssid      string
	UserAgent string
}

// IgnorePaths 要忽略的路径（中间方法）
func (builder *Builder) IgnorePaths(path string) *Builder {
	// 中间方法
	// 注：方法接收器使用值接收器时每次调用方法都会创建一个副本，当进行取地址操作时可以实现功能，
	// 返回的是新副本的指针，但原实例并未更改，这也就造成了资源浪费，因此强烈建议使用指针接收器
	builder.ignorePaths[path] = ""
	return builder
}

// Build 终结方法
func (builder *Builder) Build() gin.HandlerFunc {
	if builder.verificationKey == "" {
		panic("verification key is required")
	}

	return func(ctx *gin.Context) {
		// 忽略路径检查
		if _, exists := builder.ignorePaths[ctx.Request.URL.Path]; exists {
			ctx.Next()
			return
		}

		// JWT验证流程
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			// 还未登录
			// TODO 记录日志"Token extraction failed"
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// token格式错误
			// TODO 记录日志"Token extraction failed"
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]
		claims := &UserClaims{}
		// ParseWithClaims方法的claims参数一定要传指针，方法会对claims进行修改
		token, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(builder.verificationKey), nil
			})

		if err != nil || !token.Valid || token == nil || claims.UID == 0 { // 过期Valid为false
			// token检验错误
			// TODO 记录日志"Token validation failed"
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 自定义校验
		if claims.UserAgent != ctx.Request.UserAgent() {
			// 严重的安全问题
			// TODO 记录日志/监控"Custom validation failed"
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}
		// 通过校验
		// Token续约逻辑（还剩leftTime时）
		if time.Until(claims.ExpiresAt.Time) < builder.leftTime {
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(builder.expiresTime)) // expiresTime后过期
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			newToken, err := token.SignedString([]byte(builder.verificationKey)) // 重新生成token
			if err != nil {
				// 无需中断程序运行
				// TODO 记录日志"Token refresh failed"
			} else {
				ctx.Header("x-refresh-token", newToken)
			}
		}

		// 设置上下文
		ctx.Set("claims", claims)
		ctx.Next()
	}
}
