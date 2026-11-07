package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/nodal/nodal-server/apperr"
	"github.com/nodal/nodal-server/config"
	"github.com/nodal/nodal-server/embed"
	"github.com/nodal/nodal-server/model"
	"github.com/nodal/nodal-server/token"
	"github.com/nodal/nodal-server/vector"
)

// KnowledgeService 负责私有知识库：上传 → 解析 → 切片 → 向量化 → 落库。
// 文件按 owner_key 隔离；游客没有知识库。
type KnowledgeService struct {
	db         *gorm.DB
	store      vector.Store
	embedders  *embed.Registry
	creds      *CredentialService
	storageDir string
	maxBytes   int64
	chunkRunes int
	chunkOlap  int
	embedModel string
	embedDim   int
}

func NewKnowledgeService(db *gorm.DB, store vector.Store, embedders *embed.Registry,
	creds *CredentialService, cfg *config.Config) (*KnowledgeService, error) {

	if err := os.MkdirAll(cfg.StorageDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建上传目录 %s 失败: %w", cfg.StorageDir, err)
	}
	return &KnowledgeService{
		db:         db,
		store:      store,
		embedders:  embedders,
		creds:      creds,
		storageDir: cfg.StorageDir,
		maxBytes:   cfg.UploadMaxBytes,
		chunkRunes: cfg.ChunkRunes,
		chunkOlap:  cfg.ChunkOverlap,
		embedModel: cfg.EmbedModel,
		embedDim:   cfg.EmbedDim,
	}, nil
}

func (s *KnowledgeService) MaxBytes() int64 { return s.maxBytes }

type KnowledgeFileView struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Ext        string    `json:"ext"`
	MIME       string    `json:"mime"`
	Size       int64     `json:"size"`
	Status     string    `json:"status"`
	ChunkCount int       `json:"chunk_count"`
	ErrorMsg   string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type KnowledgeList struct {
	Items []KnowledgeFileView `json:"items"`
	Total int64               `json:"total"`
}

// Upload 保存文件并同步完成切片/向量化。任何一步失败都会清理文件与半成品数据。
func (s *KnowledgeService) Upload(ctx context.Context, id *token.Identity, name string, size int64, mime string, src io.Reader) (*KnowledgeFileView, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法使用知识库，请先登录")
	}

	name = sanitizeFileName(name)
	ext := normalizeExt(name)
	if ext == "" {
		return nil, apperr.New(apperr.CodeUnsupportedFile, "缺少文件扩展名")
	}
	extMIME, ok := supportedExt(ext)
	if !ok {
		return nil, apperr.Newf(apperr.CodeUnsupportedFile, "不支持的文件类型 %s（支持 txt/md/csv/json/pdf/docx 及常见代码文件）", ext)
	}
	if mime == "" {
		mime = extMIME
	}
	if size <= 0 {
		return nil, apperr.New(apperr.CodeInvalidParam, "文件为空")
	}
	if size > s.maxBytes {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "文件超过大小上限（%d MB）", s.maxBytes>>20)
	}

	storedName := token.NewID() + ext
	dstPath := filepath.Join(s.storageDir, storedName)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "保存文件失败", err)
	}
	written, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(dstPath)
		return nil, apperr.Wrap(apperr.CodeInternal, "写入文件失败", errors.Join(copyErr, closeErr))
	}
	if written == 0 {
		_ = os.Remove(dstPath)
		return nil, apperr.New(apperr.CodeInvalidParam, "文件为空")
	}

	ownerKey := id.OwnerKey()
	record := &model.KnowledgeFile{
		OwnerKey:   ownerKey,
		Name:       name,
		Ext:        ext,
		MIME:       mime,
		Size:       written,
		StoredName: storedName,
		Status:     model.KnowledgeStatusProcessing,
	}
	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		_ = os.Remove(dstPath)
		return nil, apperr.Wrap(apperr.CodeInternal, "创建知识库记录失败", err)
	}

	cleanup := func() {
		_ = s.store.DeleteFile(context.Background(), ownerKey, record.ID)
		_ = s.db.WithContext(context.Background()).Delete(record).Error
		_ = os.Remove(dstPath)
	}

	text, err := extractText(dstPath, ext)
	if err != nil {
		cleanup()
		return nil, err
	}
	text = cleanKnowledgeText(text)
	if len([]rune(text)) < 20 {
		cleanup()
		return nil, apperr.New(apperr.CodeFileParseFailed, "文件里没有足够可用的文本")
	}

	chunks, truncated := splitKnowledgeChunks(text, s.chunkRunes, s.chunkOlap)
	if len(chunks) == 0 {
		cleanup()
		return nil, apperr.New(apperr.CodeFileParseFailed, "文本切片为空")
	}

	inputs, err := s.embedChunks(ctx, ownerKey, chunks)
	if err != nil {
		cleanup()
		return nil, err
	}
	if err := s.store.ReplaceFile(ctx, ownerKey, record.ID, inputs); err != nil {
		cleanup()
		return nil, apperr.Wrap(apperr.CodeInternal, "写入向量失败", err)
	}

	updates := map[string]any{
		"status":      model.KnowledgeStatusReady,
		"chunk_count": len(inputs),
		"error_msg":   "",
	}
	if truncated {
		updates["error_msg"] = fmt.Sprintf("文本过长，仅索引前 %d 个切片", maxChunksPerFile)
	}
	if err := s.db.WithContext(ctx).Model(record).Updates(updates).Error; err != nil {
		cleanup()
		return nil, apperr.Wrap(apperr.CodeInternal, "更新知识库状态失败", err)
	}

	slog.Info("knowledge file indexed", "file_id", record.ID, "owner_key", ownerKey,
		"chunks", len(inputs), "backend", s.store.Backend())

	record.Status = model.KnowledgeStatusReady
	record.ChunkCount = len(inputs)
	if truncated {
		record.ErrorMsg = updates["error_msg"].(string)
	}
	return viewOfKnowledgeFile(record), nil
}

func (s *KnowledgeService) List(ctx context.Context, id *token.Identity) (*KnowledgeList, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法使用知识库，请先登录")
	}

	var rows []model.KnowledgeFile
	if err := s.db.WithContext(ctx).
		Where("owner_key = ?", id.OwnerKey()).
		Order("created_at DESC").Order("id DESC").
		Find(&rows).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "查询知识库失败", err)
	}

	items := make([]KnowledgeFileView, 0, len(rows))
	for i := range rows {
		items = append(items, *viewOfKnowledgeFile(&rows[i]))
	}
	return &KnowledgeList{Items: items, Total: int64(len(items))}, nil
}

func (s *KnowledgeService) Delete(ctx context.Context, id *token.Identity, fileID uint) error {
	if id == nil || !id.IsUser() {
		return apperr.New(apperr.CodeForbidden, "游客无法使用知识库，请先登录")
	}

	record, err := s.findOwned(ctx, id.OwnerKey(), fileID)
	if err != nil {
		return err
	}

	if err := s.store.DeleteFile(ctx, record.OwnerKey, record.ID); err != nil {
		return apperr.Wrap(apperr.CodeInternal, "删除向量失败", err)
	}
	if err := s.db.WithContext(ctx).Delete(record).Error; err != nil {
		return apperr.Wrap(apperr.CodeInternal, "删除知识库文件失败", err)
	}
	_ = os.Remove(filepath.Join(s.storageDir, record.StoredName))
	slog.Info("knowledge file deleted", "file_id", fileID, "owner_key", record.OwnerKey)
	return nil
}

func (s *KnowledgeService) findOwned(ctx context.Context, ownerKey string, fileID uint) (*model.KnowledgeFile, error) {
	if fileID == 0 {
		return nil, apperr.New(apperr.CodeInvalidParam, "文件 ID 不合法")
	}
	var record model.KnowledgeFile
	err := s.db.WithContext(ctx).Where("id = ? AND owner_key = ?", fileID, ownerKey).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.New(apperr.CodeKnowledgeNotFound, "")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "查询知识库文件失败", err)
	}
	return &record, nil
}

// FilesOwned 返回当前用户的指定文件（用于挂载校验），缺失的文件不会出现在结果里。
func (s *KnowledgeService) FilesOwned(ctx context.Context, ownerKey string, fileIDs []uint) (map[uint]*model.KnowledgeFile, error) {
	out := make(map[uint]*model.KnowledgeFile, len(fileIDs))
	if len(fileIDs) == 0 {
		return out, nil
	}
	var rows []model.KnowledgeFile
	if err := s.db.WithContext(ctx).
		Where("owner_key = ? AND id IN ?", ownerKey, fileIDs).
		Find(&rows).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "查询知识库文件失败", err)
	}
	for i := range rows {
		out[rows[i].ID] = &rows[i]
	}
	return out, nil
}

func (s *KnowledgeService) embedChunks(ctx context.Context, ownerKey string, chunks []string) ([]vector.ChunkInput, error) {
	inputs := make([]vector.ChunkInput, 0, len(chunks))
	const batch = 32
	for start := 0; start < len(chunks); start += batch {
		end := start + batch
		if end > len(chunks) {
			end = len(chunks)
		}
		vecs, err := s.embedTexts(ctx, ownerKey, chunks[start:end])
		if err != nil {
			return nil, err
		}
		for i, vec := range vecs {
			inputs = append(inputs, vector.ChunkInput{
				ChunkIndex: start + i,
				Content:    chunks[start+i],
				Vector:     vec,
			})
		}
	}
	return inputs, nil
}

// EmbedText 供 RAG 检索使用：把查询文本向量化。
func (s *KnowledgeService) EmbedText(ctx context.Context, ownerKey, text string) ([]float32, error) {
	vecs, err := s.embedTexts(ctx, ownerKey, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (s *KnowledgeService) embedTexts(ctx context.Context, ownerKey string, texts []string) ([][]float32, error) {
	e, err := s.embedders.Resolve(s.embedModel)
	if err != nil {
		return nil, apperr.New(apperr.CodeInternal, err.Error())
	}

	cred := embed.Credential{Provider: e.Provider()}
	if e.RequiresKey() {
		c, err := s.creds.CredentialFor(ctx, ownerKey, e.Provider())
		if err != nil {
			return nil, err
		}
		cred.APIKey = c.APIKey
		cred.BaseURL = c.BaseURL
	}

	vecs, err := e.Embed(ctx, cred, texts)
	if err != nil {
		slog.Warn("embedding failed", "owner_key", ownerKey, "model", s.embedModel, "error", err)
		return nil, apperr.Wrap(apperr.CodeEmbeddingFailed, "", err)
	}
	if len(vecs) != len(texts) {
		return nil, apperr.New(apperr.CodeEmbeddingFailed, "向量服务返回数量与文本不一致")
	}
	for _, vec := range vecs {
		if len(vec) != s.embedDim {
			return nil, apperr.Newf(apperr.CodeEmbeddingFailed,
				"向量维度 %d 与配置 %d 不一致", len(vec), s.embedDim)
		}
	}
	return vecs, nil
}

func viewOfKnowledgeFile(f *model.KnowledgeFile) *KnowledgeFileView {
	return &KnowledgeFileView{
		ID:         f.ID,
		Name:       f.Name,
		Ext:        f.Ext,
		MIME:       f.MIME,
		Size:       f.Size,
		Status:     f.Status,
		ChunkCount: f.ChunkCount,
		ErrorMsg:   f.ErrorMsg,
		CreatedAt:  f.CreatedAt,
	}
}

func trimForPreview(s string, max int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
