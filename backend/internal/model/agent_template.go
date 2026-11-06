package model

import "time"

// Agent 角色（PRD §4 F-08 四个内置 Agent）。
const (
	RoleExplorer   = "explorer"   // 探索者 · 蓝 · 发散
	RoleVerifier   = "verifier"   // 校验者 · 红 · 挑漏洞
	RoleSummarizer = "summarizer" // 总结者 · 绿 · 收拢
	RoleBuilder    = "builder"    // 构建者 · 黄 · 出产物
)

// AgentTemplate 对应 agent_templates 表。
// owner_id 为 null 表示系统内置模板（is_builtin=true）。
type AgentTemplate struct {
	ID           string  `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OwnerID      *string `gorm:"column:owner_id;type:uuid;index:idx_agents_owner" json:"owner_id,omitempty"`
	Name         string  `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Role         string  `gorm:"column:role;type:varchar(32);not null;index:idx_agents_role" json:"role"`
	Description  string  `gorm:"column:description;type:varchar(200);not null;default:''" json:"description"`
	SystemPrompt string  `gorm:"column:system_prompt;type:text;not null;default:''" json:"system_prompt"`
	ModelID      string  `gorm:"column:model_id;type:varchar(64);not null;default:''" json:"model_id"`
	IsBuiltin    bool    `gorm:"column:is_builtin;type:boolean;not null;default:false;index:idx_agents_builtin" json:"is_builtin"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`

	Owner *User `gorm:"foreignKey:OwnerID;references:ID;constraint:OnDelete:SET NULL" json:"-"`
}

func (AgentTemplate) TableName() string { return "agent_templates" }
