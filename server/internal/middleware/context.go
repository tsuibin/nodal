// Package middleware 提供鉴权、错误收敛与上下文工具。
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/internal/token"
)

const ctxIdentityKey = "nodal.identity"

// SetIdentity 把身份写入请求上下文（仅鉴权中间件内部使用）。
func SetIdentity(c *gin.Context, id *token.Identity) {
	c.Set(ctxIdentityKey, id)
}

// IdentityFrom 读取当前请求身份；公开路由上可能为空。
func IdentityFrom(c *gin.Context) (*token.Identity, bool) {
	v, ok := c.Get(ctxIdentityKey)
	if !ok {
		return nil, false
	}
	id, ok := v.(*token.Identity)
	return id, ok && id != nil
}

// MustIdentity 取身份，取不到时返回未认证错误，供 handler 兜底使用。
func MustIdentity(c *gin.Context) (*token.Identity, error) {
	id, ok := IdentityFrom(c)
	if !ok {
		return nil, errUnauthorized()
	}
	return id, nil
}
