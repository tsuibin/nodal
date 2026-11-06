package auth

import (
	"testing"
	"time"
)

func TestSignAndParse(t *testing.T) {
	secret := "test-secret-32-bytes-minimum-xxxxx"
	token, err := Sign(secret, "u_1", "a@b.c", time.Hour)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	claims, err := Parse(secret, token)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if claims.UserID != "u_1" || claims.Email != "a@b.c" {
		t.Fatalf("载荷不符: %+v", claims)
	}
}

func TestParseExpired(t *testing.T) {
	secret := "test-secret-32-bytes-minimum-xxxxx"
	token, err := Sign(secret, "u_1", "", -time.Minute)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	if _, err := Parse(secret, token); err != ErrExpiredToken {
		t.Fatalf("期望过期错误，实际: %v", err)
	}
}

func TestParseWrongSecret(t *testing.T) {
	token, err := Sign("secret-a-32-bytes-minimum-xxxxxxxx", "u_1", "", time.Hour)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	if _, err := Parse("secret-b-32-bytes-minimum-yyyyyyyy", token); err != ErrInvalidToken {
		t.Fatalf("期望签名无效，实际: %v", err)
	}
}

func TestPasswordHash(t *testing.T) {
	hash, err := HashPassword("nodal123456")
	if err != nil {
		t.Fatalf("哈希失败: %v", err)
	}
	if !VerifyPassword(hash, "nodal123456") {
		t.Fatal("正确密码校验失败")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Fatal("错误密码不应通过")
	}
}
