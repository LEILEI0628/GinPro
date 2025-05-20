package ginx

import (
	loggerx "github.com/LEILEI0628/GinPro/middleware/logger"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
)

var L loggerx.Logger // 可以使用包变量的形式实现

func WrapToken[claims jwt.Claims](fn func(ctx *gin.Context, uc *claims) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 可以通过传入钩子函数before来在这里进行一些业务操作
		val, ok := ctx.Get("claims")
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c, ok := val.(*claims)
		if !ok {
			// 可以监控这里
			L.Error("claims断言失败",
				loggerx.String("path", ctx.Request.URL.Path),
				// 命中的路由
				loggerx.String("route", ctx.FullPath()))
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 下面业务逻辑可能要操作ctx和读取HTTP header，因此选择传入业务处理方法
		res, err := fn(ctx, c)
		if err != nil {
			// 处理error，记录日志
			L.Error("处理业务逻辑出错",
				loggerx.String("path", ctx.Request.URL.Path),
				// 命中的路由
				loggerx.String("route", ctx.FullPath()),
				loggerx.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
		// 可以通过传入钩子函数after来在这里进行一些业务操作
	}
}

func WrapBodyAndToken[Req any, claims jwt.Claims](fn func(ctx *gin.Context, req Req, uc *claims) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.Bind(&req); err != nil {
			return
		}

		val, ok := ctx.Get("claims")
		if !ok {
			// 未登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claim, ok := val.(*claims)
		if !ok {
			// 可以监控这里
			L.Error("claims断言失败",
				loggerx.String("path", ctx.Request.URL.Path),
				// 命中的路由
				loggerx.String("route", ctx.FullPath()))
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// 下面业务逻辑可能要操作ctx和读取HTTP header，因此选择传入业务处理方法
		res, err := fn(ctx, req, claim)
		if err != nil {
			// 处理error，记录日志
			L.Error("处理业务逻辑出错",
				loggerx.String("path", ctx.Request.URL.Path),
				// 命中的路由
				loggerx.String("route", ctx.FullPath()),
				loggerx.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

func WrapBody[T any](fn func(ctx *gin.Context, req T) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req T
		if err := ctx.Bind(&req); err != nil {
			return
		}

		// 下面业务逻辑可能要操作ctx和读取HTTP header，因此选择传入业务处理方法
		res, err := fn(ctx, req)
		if err != nil {
			// 处理error，记录日志
			L.Error("处理业务逻辑出错",
				loggerx.String("path", ctx.Request.URL.Path),
				// 命中的路由
				loggerx.String("route", ctx.FullPath()),
				loggerx.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}
