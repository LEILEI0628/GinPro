package loggerx

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/atomic"
	"io"
	"time"
)

type Builder struct {
	allowReqBody  *atomic.Bool
	reqLen        *atomic.Int64
	allowRespBody *atomic.Bool
	loggerFunc    func(ctx context.Context, al *AccessLog)
}

func NewBuilder(fn func(ctx context.Context, al *AccessLog)) *Builder {
	return &Builder{allowReqBody: atomic.NewBool(false), reqLen: atomic.NewInt64(1024), allowRespBody: atomic.NewBool(false), loggerFunc: fn}
}

func (b *Builder) AllowReqBody(flag bool, maxLen int64) *Builder {
	b.allowReqBody.Store(flag)
	b.reqLen.Store(maxLen)
	return b
}

func (b *Builder) AllowRespBody(flag bool) *Builder {
	b.allowRespBody.Store(flag)
	return b
}

func (b *Builder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		al := &AccessLog{
			Method: ctx.Request.Method,
			Url:    ctx.Request.URL.String(),
		}
		if b.allowReqBody.Load() && ctx.Request.Body != nil {
			body, _ := io.ReadAll(ctx.Request.Body)
			//body, _ := ctx.GetRawData()
			ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
			if int64(len(body)) > b.reqLen.Load() {
				body = body[:b.reqLen.Load()]
			}
			al.ReqBody = string(body)
		}

		if b.allowRespBody.Load() {
			ctx.Writer = responseWriter{
				ResponseWriter: ctx.Writer,
				al:             al,
			}
		}

		defer func() {
			al.Duration = time.Since(start).String()
			b.loggerFunc(ctx, al)
		}()

		// 执行到业务逻辑
		ctx.Next()
	}
}

type responseWriter struct {
	al *AccessLog
	gin.ResponseWriter
}

func (w responseWriter) WriteHeader(statusCode int) {
	w.al.Status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w responseWriter) Write(data []byte) (n int, err error) {
	w.al.RespBody = string(data)
	return w.ResponseWriter.Write(data)
}

func (w responseWriter) WriteString(data string) (n int, err error) {
	w.al.RespBody = data
	return w.ResponseWriter.WriteString(data)
}

type AccessLog struct {
	Method   string
	Url      string
	Duration string
	ReqBody  string
	RespBody string
	Status   int
}
