package model

import "time"

// ModelCredential 是用户自有的模型 API Key。
//
// APIKeyEnc 只存密文（AES-256-GCM，AAD 绑定 owner+provider），
// KeyHint 只保留尾号用于界面展示，任何接口都不返回完整明文。
type ModelCredential struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerKey  string    `gorm:"size:80;not null;uniqueIndex:uk_credentials_owner_provider,priority:1" json:"-"`
	Provider  string    `gorm:"size:32;not null;uniqueIndex:uk_credentials_owner_provider,priority:2" json:"provider"`
	APIKeyEnc string    `gorm:"type:text;not null" json:"-"`
	KeyHint   string    `gorm:"size:24;not null;default:''" json:"key_hint"`
	BaseURL   string    `gorm:"size:255;not null;default:''" json:"base_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ModelCredential) TableName() string { return "model_credentials" }
