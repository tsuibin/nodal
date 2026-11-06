package model

import (
	"time"

	"gorm.io/datatypes"
)

// 节点类型（PRD §5.1）。
const (
	NodeTypeText    = "text"
	NodeTypeCode    = "code"
	NodeTypeFormula = "formula"
	NodeTypeFile    = "file"
	NodeTypeImage   = "image"
)

// 节点来源（PRD 规则 8：必须区分用户与 Agent 生成）。
const (
	NodeSourceUser  = "user"
	NodeSourceAgent = "agent"
)

// 节点状态：idle 静止 / streaming 流式生成中 / changed 前提已变更 / error 生成失败。
const (
	NodeStatusIdle      = "idle"
	NodeStatusStreaming = "streaming"
	NodeStatusChanged   = "changed"
	NodeStatusError     = "error"
)

// CanvasNode 对应 canvas_nodes 表。
// 主键为 (canvas_id, node_id)：node_id 沿用前端 ReactFlow 的字符串 id。
type CanvasNode struct {
	CanvasID string `gorm:"column:canvas_id;type:uuid;primaryKey" json:"canvas_id"`
	NodeID   string `gorm:"column:node_id;type:varchar(64);primaryKey" json:"node_id"`

	NodeType string  `gorm:"column:node_type;type:varchar(16);not null;index:idx_nodes_type" json:"type"`
	PosX     float64 `gorm:"column:position_x;type:double precision;not null;default:0" json:"-"`
	PosY     float64 `gorm:"column:position_y;type:double precision;not null;default:0" json:"-"`

	// DataJSON 保存节点内容与样式（PRD §7.2 data 字段）
	DataJSON datatypes.JSON `gorm:"column:data_json;type:jsonb;not null;default:'{}'" json:"data"`

	Source  string  `gorm:"column:source;type:varchar(8);not null;default:'user';index:idx_nodes_source" json:"source"`
	AgentID *string `gorm:"column:agent_id;type:uuid" json:"agent_id,omitempty"`
	Status  string  `gorm:"column:status;type:varchar(16);not null;default:'idle'" json:"status"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`

	Canvas *Canvas `gorm:"foreignKey:CanvasID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (CanvasNode) TableName() string { return "canvas_nodes" }
