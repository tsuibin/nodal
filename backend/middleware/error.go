package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nodal/nodal-server/response"
)

// RequestID 为每个请求生成/透传 X-Request-ID，便于把前端报错和日志对上。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// AccessLog 输出结构化访问日志。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "err", c.Errors.String())
		}
		if c.Writer.Status() >= http.StatusInternalServerError {
			slog.Error("request", attrs...)
			return
		}
		slog.Info("request", attrs...)
	}
}

// ErrorHandler 是统一错误出口：handler 只负责 c.Error(err)，这里生成响应。
// 必须放在中间件链的最后（c.Next() 之后执行），保证所有错误都已收集。
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}
		if c.Writer.Written() {
			// 已经有响应体（例如静态文件），不再覆盖。
			return
		}
		response.Fail(c, c.Errors.Last().Err)
	}
}
