package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/token"
)

// GuestTokenHeader 是服务端即时签发游客 token 时回写的响应头。
// 前端拿到后应与用户 token 一样持久化，后续请求带在 Authorization 里，
// 这样同一游客的临时画布不会每次刷新都换身份。
const GuestTokenHeader = "X-Guest-Token"

// Authenticator 负责全局 JWT 鉴权。
//
// GuestEnabled 打开时，未携带 token 的匿名请求不会直接被拒，而是被分配一个
// 「临时游客身份」（role=guest，无 uid），用于临时画布等免注册场景。
type Authenticator struct {
	tokens       *token.Manager
	guestEnabled bool
}

func NewAuthenticator(tokens *token.Manager, guestEnabled bool) *Authenticator {
	return &Authenticator{tokens: tokens, guestEnabled: guestEnabled}
}

// Global 是挂在所有业务路由组上的全局鉴权中间件。
// 新增业务接口只要挂在受保护分组下，就自动获得鉴权，无需逐个添加。
func (a *Authenticator) Global() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.GetHeader("Authorization"))

		// 1) 没有凭证：游客可用则即时发放临时身份，否则 401。
		if raw == "" {
			if !a.guestEnabled {
				c.Error(apperr.New(apperr.CodeUnauthorized, ""))
				c.Abort()
				return
			}

			guestID := token.NewID()
			sessionID := token.NewID()
			tk, err := a.tokens.IssueGuest(guestID, sessionID)
			if err != nil {
				c.Error(err)
				c.Abort()
				return
			}

			SetIdentity(c, &token.Identity{
				Role:      token.RoleGuest,
				GuestID:   guestID,
				SessionID: sessionID,
				ExpiresAt: tk.ExpiresAt,
			})
			// 让客户端可以握住这个游客身份；浏览器需在 CORS 中 expose 该头。
			c.Header(GuestTokenHeader, tk.Value)
			c.Next()
			return
		}

		// 2) 有凭证：解析成功即放行（user / guest 都算已认证身份）。
		id, err := a.tokens.Parse(raw)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		SetIdentity(c, id)
		c.Next()
	}
}

// RequireUser 在全局鉴权之后使用，只允许注册用户通过。
// 游客访问时返回 1003，前端据此弹出「登录后继续」。
func (a *Authenticator) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := IdentityFrom(c)
		if !ok {
			c.Error(errUnauthorized())
			c.Abort()
			return
		}
		if !id.IsUser() {
			c.Error(apperr.New(apperr.CodeForbidden, "该功能需要登录后使用"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// Optional 用于极少数公开但希望「有 token 就识别身份」的接口（如埋点）。
// 解析失败时按匿名处理，不阻断请求。
func (a *Authenticator) Optional() gin.HandlerFunc {
	return func(c *gin.Context) {
		if raw := bearerToken(c.GetHeader("Authorization")); raw != "" {
			if id, err := a.tokens.Parse(raw); err == nil {
				SetIdentity(c, id)
			}
		}
		c.Next()
	}
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func errUnauthorized() error { return apperr.New(apperr.CodeUnauthorized, "") }
