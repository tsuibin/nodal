package model

import "time"

// User 对应 users 表。见 docs/PRD.md §7.1。
type User struct {
	ID           string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string    `gorm:"column:email;type:varchar(255);not null;uniqueIndex:uk_users_email" json:"email"`
	PasswordHash string    `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	Nickname     string    `gorm:"column:nickname;type:varchar(64);not null;default:''" json:"nickname"`
	AvatarURL    string    `gorm:"column:avatar_url;type:text" json:"avatar_url"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

// TableName 固定表名，避免 GORM 复数化带来的不确定性。
func (User) TableName() string { return "users" }
