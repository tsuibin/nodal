package model

import "time"

// EmbeddingDim 是向量维度，必须与 .env 的 EMBEDDING_DIM 一致（默认 1536，OpenAI text-embedding-3-small）。
const EmbeddingDim = 1536

// KnowledgeChunk 对应 knowledge_chunks 表，embedding 列类型为 vector(1536)。
// 说明：embedding 由迁移脚本创建（AutoMigrate 路径在 migrate 包后置 SQL 补列），
// 这里用显式 type 标签让 GORM 在自动迁移时也建出 vector(1536)。
type KnowledgeChunk struct {
	ID         string `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	FileID     string `gorm:"column:file_id;type:uuid;not null;uniqueIndex:uk_chunks_file_index,priority:1;index:idx_chunks_file" json:"file_id"`
	ChunkIndex int    `gorm:"column:chunk_index;type:integer;not null;uniqueIndex:uk_chunks_file_index,priority:2" json:"chunk_index"`
	Content    string `gorm:"column:content;type:text;not null" json:"content"`
	TokenCount int    `gorm:"column:token_count;type:integer;not null;default:0" json:"token_count"`
	Embedding  Vector `gorm:"column:embedding;type:vector(1536)" json:"-"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`

	File *KnowledgeFile `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }
