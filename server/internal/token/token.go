// Package token 负责 JWT 的签发与校验。
//
// 身份分两类：
//   - 注册用户：role=user，携带 uid；
//   - 游客：role=guest，携带随机 gid，没有 uid，用于临时画布。
package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nodal/nodal-server/internal/apperr"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

// Claims 是写入 token 的私有声明。
type Claims struct {
	Role    Role   `json:"role"`
	UserID  uint   `json:"uid,omitempty"`
	GuestID string `json:"gid,omitempty"`
	// SessionID 预留给后续的登出/黑名单与多端登录管理。
	SessionID string `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

// Token 是签发结果。
type Token struct {
	Value     string
	ExpiresAt time.Time
}

// Identity 是鉴权中间件解析后写入 context 的身份。
type Identity struct {
	Role      Role
	UserID    uint
	GuestID   string
	SessionID string
	ExpiresAt time.Time
}

func (i *Identity) IsUser() bool  { return i != nil && i.Role == RoleUser && i.UserID > 0 }
func (i *Identity) IsGuest() bool { return i != nil && i.Role == RoleGuest && i.GuestID != "" }

// OwnerKey 返回业务对象的归属键，画布等表统一使用它记录 owner，避免用户/游客两套字段。
func (i *Identity) OwnerKey() string {
	switch {
	case i.IsUser():
		return fmt.Sprintf("u:%d", i.UserID)
	case i.IsGuest():
		return "g:" + i.GuestID
	default:
		return ""
	}
}

// Manager 是并发安全的 token 管理器（无内部可变状态）。
type Manager struct {
	secret   []byte
	issuer   string
	userTTL  time.Duration
	guestTTL time.Duration
	now      func() time.Time
}

func NewManager(secret, issuer string, userTTL, guestTTL time.Duration) (*Manager, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT secret 长度不足 16 位")
	}
	return &Manager{
		secret:   []byte(secret),
		issuer:   issuer,
		userTTL:  userTTL,
		guestTTL: guestTTL,
		now:      func() time.Time { return time.Now().UTC() },
	}, nil
}

// IssueUser 为注册用户签发 token。
func (m *Manager) IssueUser(userID uint, sessionID string) (*Token, error) {
	if userID == 0 {
		return nil, apperr.Newf(apperr.CodeInternal, "签发 token 失败：uid 为空")
	}
	return m.issue(Claims{Role: RoleUser, UserID: userID, SessionID: sessionID}, m.userTTL)
}

// IssueGuest 为游客签发 token；guestID 必须由调用方生成并持久化会话。
func (m *Manager) IssueGuest(guestID, sessionID string) (*Token, error) {
	if guestID == "" {
		return nil, apperr.Newf(apperr.CodeInternal, "签发游客 token 失败：gid 为空")
	}
	return m.issue(Claims{Role: RoleGuest, GuestID: guestID, SessionID: sessionID}, m.guestTTL)
}

func (m *Manager) issue(claims Claims, ttl time.Duration) (*Token, error) {
	now := m.now()
	exp := now.Add(ttl)
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    m.issuer,
		Subject:   subjectOf(claims),
		ID:        NewID(),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)), // 容忍轻微时钟漂移
		ExpiresAt: jwt.NewNumericDate(exp),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "签发 token 失败", err)
	}
	return &Token{Value: signed, ExpiresAt: exp}, nil
}

// Parse 校验 token 并返回身份；过期与非法 token 分别映射到不同业务码。
func (m *Manager) Parse(raw string) (*Identity, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	},
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if isExpired(err) {
			return nil, apperr.Wrap(apperr.CodeTokenExpired, "", err)
		}
		return nil, apperr.Wrap(apperr.CodeUnauthorized, "登录凭证无效，请重新登录", err)
	}
	if !parsed.Valid {
		return nil, apperr.New(apperr.CodeUnauthorized, "登录凭证无效，请重新登录")
	}

	id := &Identity{
		Role:      claims.Role,
		UserID:    claims.UserID,
		GuestID:   claims.GuestID,
		SessionID: claims.SessionID,
	}
	if claims.ExpiresAt != nil {
		id.ExpiresAt = claims.ExpiresAt.Time
	}

	switch {
	case id.IsUser():
		return id, nil
	case id.IsGuest():
		return id, nil
	default:
		return nil, apperr.New(apperr.CodeUnauthorized, "登录凭证无效，请重新登录")
	}
}

func subjectOf(c Claims) string {
	if c.Role == RoleUser {
		return fmt.Sprintf("user:%d", c.UserID)
	}
	return "guest:" + c.GuestID
}

func isExpired(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired) || strings.Contains(err.Error(), "token is expired")
}

// NewID 生成 32 字节随机 ID（base64url，无填充），用于游客 ID / 会话 ID。
func NewID() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		// rand.Read 失败在实践中几乎不可能发生；真发生时应快速失败而不是返回弱随机。
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
