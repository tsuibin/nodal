package model

import (
	"time"

	"gorm.io/datatypes"
)

// 关系类型（PRD §5.2），连线颜色跟随关系。
const (
	RelationDerives   = "derives"   // 推导自 · #4060FF
	RelationRefutes   = "refutes"   // 反例质疑 · #FF6B6B
	RelationSupports  = "supports"  // 补充佐证 · #36C9A5
	RelationParallel  = "parallel"  // 并行假设 · #FFC857
	RelationOptimizes = "optimizes" // 优化分支 · #4060FF 60%
)

// CanvasEdge 对应 canvas_edges 表。
// 主键 (canvas_id, edge_id)；source/target 通过复合外键指向 canvas_nodes 保证完整性。
type CanvasEdge struct {
	CanvasID string `gorm:"column:canvas_id;type:uuid;primaryKey" json:"canvas_id"`
	EdgeID   string `gorm:"column:edge_id;type:varchar(64);primaryKey" json:"id"`

	SourceNodeID string `gorm:"column:source_node_id;type:varchar(64);not null;index:idx_edges_source" json:"source"`
	TargetNodeID string `gorm:"column:target_node_id;type:varchar(64);not null;index:idx_edges_target" json:"target"`

	Relation string         `gorm:"column:relation;type:varchar(16);not null;default:'derives';index:idx_edges_relation" json:"relation"`
	Label    string         `gorm:"column:label;type:varchar(32);not null;default:''" json:"label"`
	DataJSON datatypes.JSON `gorm:"column:data_json;type:jsonb;not null;default:'{}'" json:"data,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`

	Canvas *Canvas `gorm:"foreignKey:CanvasID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (CanvasEdge) TableName() string { return "canvas_edges" }
