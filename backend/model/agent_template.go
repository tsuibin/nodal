package model

import "time"

// AgentTemplate 是可复用的推演 Agent 模板：一段系统提示词 + 绑定的模型。
// 归属于用户（owner_key = u:<uid>），按 (owner, name) 唯一。
// 模板是轻量配置，删除采用物理删除，避免软删除与唯一索引互相干扰。
type AgentTemplate struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerKey     string    `gorm:"size:80;not null;uniqueIndex:uk_agents_owner_name,priority:1" json:"-"`
	Name         string    `gorm:"size:80;not null;uniqueIndex:uk_agents_owner_name,priority:2" json:"name"`
	Description  string    `gorm:"size:300;not null;default:''" json:"description"`
	SystemPrompt string    `gorm:"type:text;not null" json:"system_prompt"`
	ModelID      string    `gorm:"size:80;not null" json:"model_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (AgentTemplate) TableName() string { return "agent_templates" }
