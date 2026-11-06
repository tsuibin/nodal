package model

import "time"

// CanvasLike 对应 canvas_likes 表，复合主键 (canvas_id, user_id) 天然去重。
type CanvasLike struct {
	CanvasID  string    `gorm:"column:canvas_id;type:uuid;primaryKey" json:"canvas_id"`
	UserID    string    `gorm:"column:user_id;type:uuid;primaryKey;index:idx_likes_user" json:"user_id"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`

	Canvas *Canvas `gorm:"foreignKey:CanvasID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	User   *User   `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (CanvasLike) TableName() string { return "canvas_likes" }
