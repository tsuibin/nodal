package model

import "time"

// KnowledgeChunk 是一个文本切片及其向量。
//
// 开发环境（sqlite/mysql）把向量存在 EmbeddingJSON（JSON 数组），检索时在 Go 里算余弦；
// Postgres 环境由 vector 包额外创建 embedding vector(N) 列，用 pgvector 的 <=> 检索。
type KnowledgeChunk struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerKey      string    `gorm:"size:80;not null;index:idx_knowledge_chunks_owner" json:"-"`
	FileID        uint      `gorm:"not null;index:idx_knowledge_chunks_file" json:"file_id"`
	ChunkIndex    int       `gorm:"not null;default:0" json:"chunk_index"`
	Content       string    `gorm:"type:text;not null" json:"content"`
	EmbeddingJSON string    `gorm:"type:text;not null;default:''" json:"-"`
	CreatedAt     time.Time `json:"created_at"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }
