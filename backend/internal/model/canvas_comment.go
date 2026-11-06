package model

import "time"

// CanvasComment 对应 canvas_comments 表。
// node_id 可空：为空时是整图评论，非空时是锚定到某节点的行内评论（P2 F-15）。
// 不使用软删除：删除评论直接物理删除，保证广场展示口径简单一致。
type CanvasComment struct {
	ID       string  `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CanvasID string  `gorm:"column:canvas_id;type:uuid;not null;index:idx_comments_canvas" json:"canvas_id"`
	UserID   string  `gorm:"column:user_id;type:uuid;not null;index:idx_comments_user" json:"user_id"`
	NodeID   *string `gorm:"column:node_id;type:varchar(64)" json:"node_id,omitempty"`
	Content  string  `gorm:"column:content;type:text;not null" json:"content"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`

	Canvas *Canvas `gorm:"foreignKey:CanvasID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	User   *User   `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (CanvasComment) TableName() string { return "canvas_comments" }
