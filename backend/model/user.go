// Package model 定义 GORM 数据模型。
package model

import (
	"time"

	"gorm.io/gorm"
)

// 用户状态。
const (
	UserStatusActive   int8 = 1
	UserStatusDisabled int8 = 0
)

// User 是注册用户。
// 注意：PasswordHash 带 json:"-"，任何情况下都不会被序列化出去。
type User struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string         `gorm:"size:190;not null;uniqueIndex:uk_users_email" json:"email"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Nickname     string         `gorm:"size:32;not null;default:''" json:"nickname"`
	AvatarURL    string         `gorm:"size:512;not null;default:''" json:"avatar_url"`
	Status       int8           `gorm:"not null;default:1;index:idx_users_status" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index:idx_users_deleted_at" json:"-"`
}

func (User) TableName() string { return "users" }

// IsActive 判断账号是否可用。
func (u *User) IsActive() bool { return u.Status == UserStatusActive }
