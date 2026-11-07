// Package response 统一 HTTP 响应结构，保证所有接口（成功/失败）格式一致：
//
//	{ "code": 0, "message": "ok", "data": {...}, "request_id": "..." }
package response

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/apperr"
)

// Envelope 是所有接口的统一响应体。
type Envelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// OK 输出成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{
		Code:      int(apperr.CodeOK),
		Message:   "ok",
		Data:      data,
		RequestID: requestID(c),
	})
}

// Fail 输出失败响应；未知错误一律折叠为 5000，避免内部信息外泄。
func Fail(c *gin.Context, err error) {
	appErr := apperr.From(err)
	if appErr.Status() >= http.StatusInternalServerError {
		slog.Error("request failed",
			"request_id", requestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"error", err,
		)
		appErr = apperr.New(apperr.CodeInternal, "")
	}

	c.AbortWithStatusJSON(appErr.Status(), Envelope{
		Code:      int(appErr.Code),
		Message:   appErr.Message,
		Details:   appErr.Details,
		RequestID: requestID(c),
	})
}

func requestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
