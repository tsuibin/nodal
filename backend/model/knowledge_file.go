package model

import "time"

// KnowledgeFile 是用户知识库里的一个原始文件。
// owner_key 与画布/Agent 一致（u:<uid>），游客不能拥有知识库。
type KnowledgeFile struct {
	ID         uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerKey   string `gorm:"size:80;not null;index:idx_knowledge_files_owner" json:"-"`
	Name       string `gorm:"size:160;not null" json:"name"`
	Ext        string `gorm:"size:16;not null" json:"ext"`
	MIME       string `gorm:"size:96;not null;default:''" json:"mime"`
	Size       int64  `gorm:"not null;default:0" json:"size"`
	StoredName string `gorm:"size:64;not null" json:"-"`
	// Status: processing / ready / failed
	Status     string    `gorm:"size:16;not null;default:'processing'" json:"status"`
	ChunkCount int       `gorm:"not null;default:0" json:"chunk_count"`
	ErrorMsg   string    `gorm:"size:300;not null;default:''" json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (KnowledgeFile) TableName() string { return "knowledge_files" }

const (
	KnowledgeStatusProcessing = "processing"
	KnowledgeStatusReady      = "ready"
	KnowledgeStatusFailed     = "failed"
)
