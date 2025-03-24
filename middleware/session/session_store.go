package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

// StorageType 受限的存储类型枚举
type StorageType string

const (
	Redis    StorageType = "redis"    // Redis存储
	Cookie   StorageType = "cookie"   // Cookie存储
	MemStore StorageType = "memstore" // 内存存储
)

// Config 会话存储配置
type Config struct {
	StorageType StorageType // 使用受限类型
	AuthKey     []byte      // 认证密钥（推荐32/64字节）
	EncryptKey  []byte      // 加密密钥（推荐32/64字节）
	RedisOpts   RedisOpts   // Redis专有配置
}

// RedisOpts Redis配置参数
type RedisOpts struct {
	MaxIdle  int    // 最大空闲连接数
	Network  string // 网络类型（通常为tcp）
	Addr     string // Redis地址（格式：host:port）
	Password string // Redis密码
}

// SessionStore 创建会话中间件
func SessionStore(cfg Config) gin.HandlerFunc {
	var store sessions.Store
	var err error

	switch cfg.StorageType {
	case Redis:
		// 第一个参数是最大空闲连接数量，第二个参数是连接方式，第三四个参数是连接信息和密码，第五六个是key
		store, err = redis.NewStore(
			cfg.RedisOpts.MaxIdle,
			cfg.RedisOpts.Network,
			cfg.RedisOpts.Addr,
			cfg.RedisOpts.Password,
			cfg.AuthKey,
			cfg.EncryptKey,
		)
	case Cookie:
		// cookie和memstore的NewStore()第一个参数为authentication key，第二个参数为encryption key，推荐32or64位
		store = cookie.NewStore(cfg.AuthKey, cfg.EncryptKey)
	case MemStore:
		store = memstore.NewStore(cfg.AuthKey, cfg.EncryptKey)
	default:
		panic("invalid storage type: " + string(cfg.StorageType))
	}

	if err != nil {
		panic("session store init failed: " + err.Error())
	}

	return sessions.Sessions("rb_session", store)
}
