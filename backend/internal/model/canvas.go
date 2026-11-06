package model

import (
	"time"

	"gorm.io/datatypes"
)

// 画布分享模式（PRD §7.1 share_mode）。
const (
	ShareModePrivate = "private" // 仅自己可见
	ShareModeRead    = "read"    // 只读分享
	ShareModeFork    = "fork"    // 可 Fork
)

// Canvas 对应 canvases 表。
// content_json 是画布全量快照（nodes + edges + viewport），结构见 PRD §7.2；
// canvas_nodes / canvas_edges 是它的关系化投影，用于按节点检索、评论锚定与节点级 Fork。
type Canvas struct {
	ID          string         `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OwnerID     string         `gorm:"column:owner_id;type:uuid;not null;index:idx_canvases_owner" json:"owner_id"`
	Title       string         `gorm:"column:title;type:varchar(200);not null;default:'未命名画布'" json:"title"`
	ContentJSON datatypes.JSON `gorm:"column:content_json;type:jsonb;not null;default:'{\"nodes\":[],\"edges\":[],\"viewport\":{\"x\":0,\"y\":0,\"zoom\":1}}'" json:"content_json"`
	ShareMode   string         `gorm:"column:share_mode;type:varchar(16);not null;default:'private';index:idx_canvases_share" json:"share_mode"`
	IsPublic    bool           `gorm:"column:is_public;type:boolean;not null;default:false;index:idx_canvases_public" json:"is_public"`
	// ForkedFrom 记录分叉来源画布；ForkedFromNode 记录节点级 Fork 锚点（null 表示整图 Fork）。
	ForkedFrom     *string `gorm:"column:forked_from;type:uuid" json:"forked_from,omitempty"`
	ForkedFromNode *string `gorm:"column:forked_from_node;type:varchar(64)" json:"forked_from_node,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`

	// 关联（AutoMigrate 建外键用；查询时不自动 Preload）
	Owner *User `gorm:"foreignKey:OwnerID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (Canvas) TableName() string { return "canvases" }
