package model

import "time"

// CanvasLike 是一条点赞记录。
// (canvas_id, user_key) 唯一，保证同一用户对同一画布只有一条记录；取消点赞即物理删除。
type CanvasLike struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CanvasID  uint      `gorm:"not null;uniqueIndex:uk_canvas_likes_canvas_user,priority:1" json:"canvas_id"`
	UserKey   string    `gorm:"size:80;not null;uniqueIndex:uk_canvas_likes_canvas_user,priority:2" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

func (CanvasLike) TableName() string { return "canvas_likes" }
