package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type Builder struct {
	ignorePaths []string
}

func NewBuilder() *Builder {
	return &Builder{}
}

// IgnorePaths 要忽略的路径
func (builder *Builder) IgnorePaths(path string) *Builder {
	// 中间方法
	builder.ignorePaths = append(builder.ignorePaths, path)
	return builder
}

// Build 终结方法（使用Session进行校验）
func (builder *Builder) Build(maxAgeSec int, leftTime time.Duration) gin.HandlerFunc {
	return func(context *gin.Context) {
		for _, path := range builder.ignorePaths {
			if context.Request.URL.Path == path {
				return // 无需登录校验
			}
		}

		session := sessions.Default(context)

		id := session.Get("userId")
		if id == nil {
			context.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// TODO 封装插件
		// 实现定时刷新token操作
		now := time.Now().UnixMilli()
		updateTime := session.Get("updateTime")
		session.Set("userId", id)
		err := session.Save()
		if err != nil {
			context.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if updateTime == nil {
			// 还没刷新过
			session.Set("updateTime", now)
			session.Options(sessions.Options{
				MaxAge: maxAgeSec,
			})
			err := session.Save()
			if err != nil {
				context.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			return
		}
		updateTimeVal, ok := updateTime.(int64)
		if !ok {
			context.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if now-updateTimeVal > leftTime.Milliseconds() { // 一分钟刷新一次
			session.Set("updateTime", now)
			session.Options(sessions.Options{
				MaxAge: maxAgeSec,
			})
			err := session.Save()
			if err != nil {
				context.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			return
		}

	}
}
