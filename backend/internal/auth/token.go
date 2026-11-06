// Package auth 提供 JWT 签发/解析与密码哈希。
// 模块 0 只放基础能力；注册/登录接口与鉴权中间件在模块 2 落地。
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost 12：兼顾安全与登录耗时（约 30-50ms）。
const bcryptCost = 12

// Claims 是 Nodal 的 JWT 载荷。
// 只放非敏感标识，昵称邮箱等由接口按需查库，避免令牌过大与信息泄露。
type Claims struct {
	UserID string `json:"uid"`
	Email  string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

var (
	// ErrInvalidToken 令牌无效（签名错误、结构错误）。
	ErrInvalidToken = errors.New("无效的令牌")
	// ErrExpiredToken 令牌已过期。
	ErrExpiredToken = errors.New("令牌已过期")
)

// Sign 以 secret 签发 HS256 令牌，ttl 为有效期。
func Sign(secret, userID, email string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("JWT secret 为空")
	}
	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "nodal",
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Parse 校验签名与有效期并返回 Claims。
func Parse(secret, token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名算法: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithIssuer("nodal"), jwt.WithLeeway(30*time.Second))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// HashPassword 生成 bcrypt 哈希。
func HashPassword(plain string) (string, error) {
	if len(plain) < 8 {
		return "", fmt.Errorf("密码长度至少 8 位")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword 校验明文与哈希是否匹配。
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
