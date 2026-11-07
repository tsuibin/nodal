package service

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/nodal/nodal-server/apperr"
	"github.com/nodal/nodal-server/config"
	"github.com/nodal/nodal-server/model"
	"github.com/nodal/nodal-server/token"
	"github.com/nodal/nodal-server/vector"
)

// RAGService 负责「画布临时挂载知识库」与「向量召回」。
//
// 挂载关系只存在进程内存里，不落库：重启即清空，符合「画布临时关联，不持久绑定」。
// 多实例部署需要换成 Redis/共享存储。
type RAGService struct {
	db        *gorm.DB
	knowledge *KnowledgeService
	store     vector.Store
	topK      int
	maxFiles  int

	mu     sync.RWMutex
	mounts map[uint]*knowledgeMount
}

type knowledgeMount struct {
	OwnerKey  string
	FileIDs   []uint
	UpdatedAt time.Time
}

func NewRAGService(db *gorm.DB, knowledge *KnowledgeService, store vector.Store, cfg *config.Config) *RAGService {
	return &RAGService{
		db:        db,
		knowledge: knowledge,
		store:     store,
		topK:      cfg.RAGTopK,
		maxFiles:  cfg.MountMaxFiles,
		mounts:    make(map[uint]*knowledgeMount),
	}
}

type MountedFile struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	ChunkCount int    `json:"chunk_count"`
	Status     string `json:"status"`
}

type MountView struct {
	CanvasID  uint          `json:"canvas_id"`
	FileIDs   []uint        `json:"file_ids"`
	Files     []MountedFile `json:"files"`
	UpdatedAt *time.Time    `json:"updated_at,omitempty"`
}

type RetrievedChunk struct {
	FileID     uint    `json:"file_id"`
	FileName   string  `json:"file_name"`
	ChunkIndex int     `json:"chunk_index"`
	Content    string  `json:"content"`
	Preview    string  `json:"preview"`
	Score      float64 `json:"score"`
}

type RetrieveResult struct {
	Query   string           `json:"query"`
	TopK    int              `json:"top_k"`
	Backend string           `json:"backend"`
	Items   []RetrievedChunk `json:"items"`
}

// Mount 用 file_ids 覆盖式设置画布的临时知识库；传空数组等于解除挂载。
func (s *RAGService) Mount(ctx context.Context, id *token.Identity, canvasID uint, fileIDs []uint) (*MountView, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法挂载知识库，请先登录")
	}
	canvas, err := s.findOwnedCanvas(ctx, id.OwnerKey(), canvasID)
	if err != nil {
		return nil, err
	}

	unique := make([]uint, 0, len(fileIDs))
	seen := make(map[uint]struct{}, len(fileIDs))
	for _, fid := range fileIDs {
		if _, dup := seen[fid]; dup {
			continue
		}
		seen[fid] = struct{}{}
		unique = append(unique, fid)
	}
	if len(unique) > s.maxFiles {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "最多同时挂载 %d 个知识库文件", s.maxFiles)
	}

	files, err := s.knowledge.FilesOwned(ctx, id.OwnerKey(), unique)
	if err != nil {
		return nil, err
	}
	for _, fid := range unique {
		if _, ok := files[fid]; !ok {
			return nil, apperr.Newf(apperr.CodeKnowledgeNotFound, "知识库文件 %d 不存在", fid)
		}
	}

	if len(unique) == 0 {
		s.mu.Lock()
		delete(s.mounts, canvas.ID)
		s.mu.Unlock()
		return &MountView{CanvasID: canvas.ID, FileIDs: []uint{}, Files: []MountedFile{}}, nil
	}

	now := time.Now().UTC()
	s.mu.Lock()
	s.mounts[canvas.ID] = &knowledgeMount{
		OwnerKey:  id.OwnerKey(),
		FileIDs:   unique,
		UpdatedAt: now,
	}
	s.mu.Unlock()

	slog.Info("knowledge mounted", "canvas_id", canvas.ID, "owner_key", id.OwnerKey(), "files", len(unique))
	return s.viewOfMount(canvas.ID, unique, &now, files), nil
}

// CurrentMount 返回当前挂载（没有挂载时返回空列表，不报错）。
func (s *RAGService) CurrentMount(ctx context.Context, id *token.Identity, canvasID uint) (*MountView, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法查看知识库挂载，请先登录")
	}
	canvas, err := s.findOwnedCanvas(ctx, id.OwnerKey(), canvasID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	mount := s.mounts[canvas.ID]
	var fileIDs []uint
	var updated *time.Time
	if mount != nil && mount.OwnerKey == id.OwnerKey() {
		fileIDs = append([]uint(nil), mount.FileIDs...)
		t := mount.UpdatedAt
		updated = &t
	}
	s.mu.RUnlock()

	files, err := s.knowledge.FilesOwned(ctx, id.OwnerKey(), fileIDs)
	if err != nil {
		return nil, err
	}
	view := s.viewOfMount(canvas.ID, fileIDs, updated, files)
	return view, nil
}

// Unmount 解除画布挂载。
func (s *RAGService) Unmount(ctx context.Context, id *token.Identity, canvasID uint) (*MountView, error) {
	return s.Mount(ctx, id, canvasID, nil)
}

// Retrieve 是 /api/rag/retrieve 的实现：只检索当前画布已挂载的文件。
func (s *RAGService) Retrieve(ctx context.Context, id *token.Identity, canvasID uint, query string, topK int) (*RetrieveResult, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法使用 RAG 检索，请先登录")
	}
	if _, err := s.findOwnedCanvas(ctx, id.OwnerKey(), canvasID); err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, apperr.New(apperr.CodeInvalidParam, "query 不能为空")
	}
	if len([]rune(query)) > 2000 {
		return nil, apperr.New(apperr.CodeInvalidParam, "query 最多 2000 字")
	}
	if topK <= 0 {
		topK = s.topK
	}
	if topK > 20 {
		topK = 20
	}

	items, err := s.retrieveHits(ctx, id.OwnerKey(), canvasID, query, topK)
	if err != nil {
		return nil, err
	}
	return &RetrieveResult{Query: query, TopK: topK, Backend: s.store.Backend(), Items: items}, nil
}

// RetrieveForInfer 供推演编排调用：已在外层校验过身份与画布所有权。
// 未挂载知识库时返回 4004，调用方按「无知识库」降级处理。
func (s *RAGService) RetrieveForInfer(ctx context.Context, ownerKey string, canvasID uint, query string, topK int) ([]RetrievedChunk, error) {
	if topK <= 0 {
		topK = s.topK
	}
	if topK > 20 {
		topK = 20
	}
	return s.retrieveHits(ctx, ownerKey, canvasID, query, topK)
}

func (s *RAGService) retrieveHits(ctx context.Context, ownerKey string, canvasID uint, query string, topK int) ([]RetrievedChunk, error) {
	s.mu.RLock()
	mount := s.mounts[canvasID]
	var fileIDs []uint
	if mount != nil && mount.OwnerKey == ownerKey {
		fileIDs = append([]uint(nil), mount.FileIDs...)
	}
	s.mu.RUnlock()

	if len(fileIDs) == 0 {
		return nil, apperr.New(apperr.CodeKnowledgeNotMounted, "")
	}

	queryVec, err := s.knowledge.EmbedText(ctx, ownerKey, query)
	if err != nil {
		return nil, err
	}

	hits, err := s.store.Search(ctx, ownerKey, fileIDs, queryVec, topK)
	if err != nil {
		slog.Warn("rag search failed", "canvas_id", canvasID, "owner_key", ownerKey, "error", err)
		return nil, apperr.Wrap(apperr.CodeEmbeddingFailed, "向量检索失败", err)
	}

	files, err := s.knowledge.FilesOwned(ctx, ownerKey, fileIDs)
	if err != nil {
		return nil, err
	}

	items := make([]RetrievedChunk, 0, len(hits))
	for _, hit := range hits {
		name := ""
		if f, ok := files[hit.FileID]; ok {
			name = f.Name
		}
		items = append(items, RetrievedChunk{
			FileID:     hit.FileID,
			FileName:   name,
			ChunkIndex: hit.ChunkIndex,
			Content:    hit.Content,
			Preview:    trimForPreview(hit.Content, 120),
			Score:      hit.Score,
		})
	}
	// 统一按分数倒序（pgvector 的 ORDER BY 已排好，JSON 实现也排过，这里再兜底）
	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	return items, nil
}

func (s *RAGService) findOwnedCanvas(ctx context.Context, ownerKey string, canvasID uint) (*model.Canvas, error) {
	if canvasID == 0 {
		return nil, apperr.New(apperr.CodeInvalidParam, "画布 ID 不合法")
	}
	var canvas model.Canvas
	err := s.db.WithContext(ctx).First(&canvas, canvasID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.New(apperr.CodeNotFound, "画布不存在或已删除")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "查询画布失败", err)
	}
	if !canvas.IsOwner(ownerKey) {
		return nil, apperr.New(apperr.CodeForbidden, "只有画布所有者可以管理知识库挂载")
	}
	return &canvas, nil
}

func (s *RAGService) viewOfMount(canvasID uint, fileIDs []uint, updated *time.Time, files map[uint]*model.KnowledgeFile) *MountView {
	view := &MountView{
		CanvasID:  canvasID,
		FileIDs:   fileIDs,
		Files:     make([]MountedFile, 0, len(fileIDs)),
		UpdatedAt: updated,
	}
	if view.FileIDs == nil {
		view.FileIDs = []uint{}
	}
	for _, fid := range fileIDs {
		f, ok := files[fid]
		if !ok {
			continue
		}
		view.Files = append(view.Files, MountedFile{
			ID:         f.ID,
			Name:       f.Name,
			ChunkCount: f.ChunkCount,
			Status:     f.Status,
		})
	}
	return view
}
