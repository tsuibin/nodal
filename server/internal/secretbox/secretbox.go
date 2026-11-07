// Package secretbox 用 AES-256-GCM 加密用户的模型 API Key。
//
// 存储格式：v1:<base64url(nonce || ciphertext || tag)>
// AAD 绑定 owner_key + provider，防止把 A 用户的密文挪到 B 用户或别的 provider 上解密。
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const versionPrefix = "v1:"

type Box struct {
	aead cipher.AEAD
}

func New(key []byte) (*Box, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("secretbox: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secretbox: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secretbox: new gcm: %w", err)
	}
	return &Box{aead: aead}, nil
}

// Encrypt 加密明文；aad 用于绑定上下文（如 "u:1|openai"）。
func (b *Box) Encrypt(plaintext, aad string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secretbox: nonce: %w", err)
	}

	sealed := b.aead.Seal(nil, nonce, []byte(plaintext), []byte(aad))
	payload := make([]byte, 0, len(nonce)+len(sealed))
	payload = append(payload, nonce...)
	payload = append(payload, sealed...)

	return versionPrefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

// Decrypt 解密；aad 必须与加密时一致。
func (b *Box) Decrypt(encoded, aad string) (string, error) {
	if !strings.HasPrefix(encoded, versionPrefix) {
		return "", errors.New("secretbox: unsupported ciphertext version")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(encoded, versionPrefix))
	if err != nil {
		return "", fmt.Errorf("secretbox: decode: %w", err)
	}
	nonceSize := b.aead.NonceSize()
	if len(raw) < nonceSize+b.aead.Overhead() {
		return "", errors.New("secretbox: ciphertext too short")
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plain, err := b.aead.Open(nil, nonce, ciphertext, []byte(aad))
	if err != nil {
		return "", fmt.Errorf("secretbox: open: %w", err)
	}
	return string(plain), nil
}

// AAD 生成加密附加数据，确保密文与「归属 + provider」绑定。
func AAD(ownerKey, provider string) string {
	return ownerKey + "|" + provider
}
