// Package vector 提供知识库切片的向量存储与相似度检索。
//
// 生产（postgres）使用 pgvector：embedding 列为 vector(N)，用余弦距离 `<=>` 检索；
// 开发（sqlite/mysql）没有 pgvector，退化为 embedding_json + Go 内余弦计算，
// 接口与语义保持一致，便于本地跑通全链路。
package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/nodal/nodal-server/model"
)

// ChunkInput 是待写入的一个切片。
type ChunkInput struct {
	ChunkIndex int
	Content    string
	Vector     []float32
}

// Hit 是一次检索命中的片段。
type Hit struct {
	FileID     uint    `json:"file_id"`
	ChunkIndex int     `json:"chunk_index"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
}

// Store 是向量存储的统一接口。
type Store interface {
	Backend() string
	Dim() int
	ReplaceFile(ctx context.Context, ownerKey string, fileID uint, chunks []ChunkInput) error
	Search(ctx context.Context, ownerKey string, fileIDs []uint, query []float32, topK int) ([]Hit, error)
	DeleteFile(ctx context.Context, ownerKey string, fileID uint) error
}

// New 按数据库驱动选择实现。
func New(db *gorm.DB, driver string, dim int) Store {
	if strings.ToLower(driver) == "postgres" {
		return &pgStore{db: db, dim: dim}
	}
	return &jsonStore{db: db, dim: dim}
}

// EnsurePGVectorSchema 在 Postgres 上启用 pgvector 并补齐向量列/索引。
// 必须在 AutoMigrate 之后调用（表已存在）。
func EnsurePGVectorSchema(db *gorm.DB, dim int) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return fmt.Errorf("启用 pgvector 扩展失败（请确认数据库已安装 pgvector）: %w", err)
	}
	if err := db.Exec(fmt.Sprintf(
		"ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS embedding vector(%d)", dim,
	)).Error; err != nil {
		return fmt.Errorf("创建 embedding vector 列失败: %w", err)
	}

	indexSQL := "CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding " +
		"ON knowledge_chunks USING hnsw (embedding vector_cosine_ops)"
	if err := db.Exec(indexSQL).Error; err != nil {
		// HNSW 在部分老版本 pgvector 上不可用；没有索引仍能正确检索，只是慢。
		slog.Warn("create hnsw index failed, fallback to sequential scan", "error", err)
	}
	slog.Info("pgvector schema ready", "dim", dim)
	return nil
}

// ---------- JSON 兜底实现 ----------

type jsonStore struct {
	db  *gorm.DB
	dim int
}

func (s *jsonStore) Backend() string { return "json" }
func (s *jsonStore) Dim() int        { return s.dim }

func (s *jsonStore) ReplaceFile(ctx context.Context, ownerKey string, fileID uint, chunks []ChunkInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("owner_key = ? AND file_id = ?", ownerKey, fileID).
			Delete(&model.KnowledgeChunk{}).Error; err != nil {
			return err
		}
		if len(chunks) == 0 {
			return nil
		}

		const batch = 100
		for start := 0; start < len(chunks); start += batch {
			end := start + batch
			if end > len(chunks) {
				end = len(chunks)
			}
			rows := make([]model.KnowledgeChunk, 0, end-start)
			for _, chunk := range chunks[start:end] {
				raw, err := json.Marshal(chunk.Vector)
				if err != nil {
					return err
				}
				rows = append(rows, model.KnowledgeChunk{
					OwnerKey:      ownerKey,
					FileID:        fileID,
					ChunkIndex:    chunk.ChunkIndex,
					Content:       chunk.Content,
					EmbeddingJSON: string(raw),
				})
			}
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *jsonStore) Search(ctx context.Context, ownerKey string, fileIDs []uint, query []float32, topK int) ([]Hit, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}

	var rows []model.KnowledgeChunk
	if err := s.db.WithContext(ctx).
		Where("owner_key = ? AND file_id IN ?", ownerKey, fileIDs).
		Limit(10000).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	hits := make([]Hit, 0, len(rows))
	for _, row := range rows {
		var vec []float32
		if err := json.Unmarshal([]byte(row.EmbeddingJSON), &vec); err != nil || len(vec) != len(query) {
			continue
		}
		score := cosine(query, vec)
		hits = append(hits, Hit{
			FileID:     row.FileID,
			ChunkIndex: row.ChunkIndex,
			Content:    row.Content,
			Score:      score,
		})
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

func (s *jsonStore) DeleteFile(ctx context.Context, ownerKey string, fileID uint) error {
	return s.db.WithContext(ctx).
		Where("owner_key = ? AND file_id = ?", ownerKey, fileID).
		Delete(&model.KnowledgeChunk{}).Error
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// ---------- pgvector 实现 ----------

type pgStore struct {
	db  *gorm.DB
	dim int
}

func (s *pgStore) Backend() string { return "pgvector" }
func (s *pgStore) Dim() int        { return s.dim }

func (s *pgStore) ReplaceFile(ctx context.Context, ownerKey string, fileID uint, chunks []ChunkInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("owner_key = ? AND file_id = ?", ownerKey, fileID).
			Delete(&model.KnowledgeChunk{}).Error; err != nil {
			return err
		}

		for _, chunk := range chunks {
			if len(chunk.Vector) != s.dim {
				return fmt.Errorf("向量维度 %d 与配置 %d 不一致", len(chunk.Vector), s.dim)
			}
			// embedding_json 留空：Postgres 下真正的向量在 embedding 列里。
			if err := tx.Exec(`
INSERT INTO knowledge_chunks (owner_key, file_id, chunk_index, content, embedding_json, embedding, created_at)
VALUES (?, ?, ?, ?, '', ?::vector, NOW())`,
				ownerKey, fileID, chunk.ChunkIndex, chunk.Content, vectorLiteral(chunk.Vector),
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *pgStore) Search(ctx context.Context, ownerKey string, fileIDs []uint, query []float32, topK int) ([]Hit, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}
	if len(query) != s.dim {
		return nil, fmt.Errorf("查询向量维度 %d 与配置 %d 不一致", len(query), s.dim)
	}

	type row struct {
		FileID     uint
		ChunkIndex int
		Content    string
		Score      float64
	}
	var rows []row

	sql := `
SELECT file_id, chunk_index, content, 1 - (embedding <=> ?::vector) AS score
FROM knowledge_chunks
WHERE owner_key = ? AND file_id IN ?
ORDER BY embedding <=> ?::vector
LIMIT ?`
	literal := vectorLiteral(query)
	if err := s.db.WithContext(ctx).Raw(sql, literal, ownerKey, fileIDs, literal, topK).Scan(&rows).Error; err != nil {
		return nil, err
	}

	hits := make([]Hit, 0, len(rows))
	for _, r := range rows {
		hits = append(hits, Hit{FileID: r.FileID, ChunkIndex: r.ChunkIndex, Content: r.Content, Score: r.Score})
	}
	return hits, nil
}

func (s *pgStore) DeleteFile(ctx context.Context, ownerKey string, fileID uint) error {
	return s.db.WithContext(ctx).
		Where("owner_key = ? AND file_id = ?", ownerKey, fileID).
		Delete(&model.KnowledgeChunk{}).Error
}

func vectorLiteral(vec []float32) string {
	var b strings.Builder
	b.Grow(len(vec) * 8)
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
