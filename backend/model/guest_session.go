package model

import "time"

// GuestSession 是游客的临时会话记录。
//
// 游客没有 UserID：画布等业务对象的归属靠 OwnerKey（"g:<guest_id>"），
// 会话过期后由清理任务回收，数据可整体转为只读/删除，不影响注册用户。
type GuestSession struct {
	ID         string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	ClientIP   string    `gorm:"size:64;not null;default:''" json:"client_ip"`
	UserAgent  string    `gorm:"size:255;not null;default:''" json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `gorm:"index:idx_guest_sessions_expires_at" json:"expires_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

func (GuestSession) TableName() string { return "guest_sessions" }

// Expired 判断会话在 now 时刻是否已过期。
func (g *GuestSession) Expired(now time.Time) bool { return !g.ExpiresAt.After(now) }
