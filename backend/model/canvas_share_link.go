package model

import "time"

// 分享链接权限。
const (
	SharePermissionRead = "read" // 只能查看
	SharePermissionFork = "fork" // 可查看，也可 Fork 到自己的账号
)

// CanvasShareLink 是画布的一条分享链接。
//
// 与 canvas.share_mode 的区别：share_mode 控制「拿到画布 ID 的人能否读」，
// 分享链接是带随机 token 的能力凭证，额外区分 read / fork 权限，可单独停用与过期。
type CanvasShareLink struct {
	ID         uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	CanvasID   uint       `gorm:"not null;index:idx_canvas_share_links_canvas" json:"canvas_id"`
	Token      string     `gorm:"size:64;not null;uniqueIndex:uk_canvas_share_links_token" json:"token"`
	Permission string     `gorm:"size:16;not null;default:'read'" json:"permission"`
	CreatedBy  string     `gorm:"size:80;not null" json:"-"`
	Enabled    bool       `gorm:"not null;default:true" json:"enabled"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (CanvasShareLink) TableName() string { return "canvas_share_links" }

func (l *CanvasShareLink) IsExpired(now time.Time) bool {
	return l.ExpiresAt != nil && !l.ExpiresAt.After(now)
}
