package model

import (
	"time"

	"gorm.io/gorm"
)

// CanvasComment 是广场画布下的一条评论。
// 只有注册用户可以评论；删除采用软删除，comment_count 同步递减。
type CanvasComment struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CanvasID  uint           `gorm:"not null;index:idx_canvas_comments_canvas" json:"canvas_id"`
	UserKey   string         `gorm:"size:80;not null;index:idx_canvas_comments_user" json:"-"`
	UserID    uint           `gorm:"not null;index:idx_canvas_comments_uid" json:"user_id"`
	Content   string         `gorm:"size:1000;not null" json:"content"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index:idx_canvas_comments_deleted_at" json:"-"`
}

func (CanvasComment) TableName() string { return "canvas_comments" }
