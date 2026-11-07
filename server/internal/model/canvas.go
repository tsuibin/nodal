package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// 画布分享模式。
//
//   - private：仅所有者可读写；
//   - link    ：拿到 id/链接的任何人可只读，不出现在公开列表；
//   - public  ：任何人可只读，并出现在公开画布列表。
const (
	ShareModePrivate = "private"
	ShareModeLink    = "link"
	ShareModePublic  = "public"
)

// Canvas 是一张画布。
//
// Nodes / Edges / Viewport 直接存储 React Flow 的序列化结果（JSON 原样透传，
// 不丢字段）；Tags 存字符串数组。
type Canvas struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerKey    string `gorm:"size:80;not null;index:idx_canvases_owner" json:"owner_key"`
	Title       string `gorm:"size:120;not null" json:"title"`
	Description string `gorm:"size:500;not null;default:''" json:"description"`

	// 画布文档（与 React Flow 结构一致）
	Nodes    datatypes.JSON `gorm:"type:json;not null" json:"nodes"`
	Edges    datatypes.JSON `gorm:"type:json;not null" json:"edges"`
	Viewport datatypes.JSON `gorm:"type:json;not null" json:"viewport"`

	Tags      datatypes.JSON `gorm:"type:json;not null" json:"tags"`
	IsPublic  bool           `gorm:"not null;default:false;index:idx_canvases_public" json:"is_public"`
	ShareMode string         `gorm:"size:16;not null;default:'private';index:idx_canvases_share_mode" json:"share_mode"`

	// 分叉来源；被分叉的画布删除后仍保留 ID（用于溯源，不做外键级联）。
	ForkedFrom *uint `gorm:"index:idx_canvases_forked_from" json:"forked_from,omitempty"`

	NodeCount int `gorm:"not null;default:0" json:"node_count"`
	EdgeCount int `gorm:"not null;default:0" json:"edge_count"`

	// 广场相关：发布后进入 /square/list；点赞/评论为冗余计数，便于列表排序。
	PublishedAt  *time.Time `gorm:"index:idx_canvases_published_at" json:"published_at,omitempty"`
	LikeCount    int        `gorm:"not null;default:0" json:"like_count"`
	CommentCount int        `gorm:"not null;default:0" json:"comment_count"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index:idx_canvases_deleted_at" json:"-"`
}

func (Canvas) TableName() string { return "canvases" }

// IsOwner 判断某个归属键是否是这张画布的所有者。
func (c *Canvas) IsOwner(ownerKey string) bool {
	return ownerKey != "" && c.OwnerKey == ownerKey
}

// ReadableByOthers 判断非所有者是否可只读访问。
func (c *Canvas) ReadableByOthers() bool {
	return c.IsPublic || c.ShareMode == ShareModeLink || c.ShareMode == ShareModePublic
}
