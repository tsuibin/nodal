package model

import "time"

// 知识库文件处理状态。
const (
	KFilePending  = "pending"
	KFileIndexing = "indexing"
	KFileReady    = "ready"
	KFileFailed   = "failed"
)

// KnowledgeFile 对应 knowledge_files 表（PRD §7.1）。
// 文件本身落对象存储 / 本地 UPLOAD_DIR，库里只存元数据与处理状态。
type KnowledgeFile struct {
	ID          string `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OwnerID     string `gorm:"column:owner_id;type:uuid;not null;index:idx_kfiles_owner" json:"owner_id"`
	Filename    string `gorm:"column:filename;type:varchar(255);not null" json:"filename"`
	Mime        string `gorm:"column:mime;type:varchar(64);not null;default:''" json:"mime"`
	SizeBytes   int64  `gorm:"column:size_bytes;type:bigint;not null;default:0" json:"size_bytes"`
	StoragePath string `gorm:"column:storage_path;type:text;not null;default:''" json:"storage_path"`
	Status      string `gorm:"column:status;type:varchar(16);not null;default:'pending';index:idx_kfiles_status" json:"status"`
	ChunkCount  int    `gorm:"column:chunk_count;type:integer;not null;default:0" json:"chunk_count"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`

	Owner *User `gorm:"foreignKey:OwnerID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (KnowledgeFile) TableName() string { return "knowledge_files" }
