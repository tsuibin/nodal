package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/model"
	"github.com/nodal/nodal-server/internal/token"
)

// CanvasService 处理画布的 CRUD、权限与分叉。
//
// 权限模型：
//   - 读：所有者，或 share_mode=link/public、is_public=true 时任何人可读；
//   - 写（更新/删除/改分享设置）：只有所有者（游客也可以拥有自己的临时画布）；
//   - 分叉：任何「可读」的人都可以，分叉出来归分叉者所有且默认私有。
type CanvasService struct {
	db *gorm.DB
}

func NewCanvasService(db *gorm.DB) *CanvasService {
	return &CanvasService{db: db}
}

// ---------- 入参 ----------

type CreateCanvasInput struct {
	Title       string          `json:"title" binding:"omitempty,max=120"`
	Description string          `json:"description" binding:"omitempty,max=500"`
	Nodes       json.RawMessage `json:"nodes"`
	Edges       json.RawMessage `json:"edges"`
	Viewport    json.RawMessage `json:"viewport"`
	Tags        []string        `json:"tags" binding:"omitempty,max=10,dive,max=20"`
	IsPublic    bool            `json:"is_public"`
	ShareMode   string          `json:"share_mode" binding:"omitempty,max=16"`
}

// UpdateCanvasInput 支持部分更新；nodes/edges 一旦提交即整图替换（必须成对出现）。
type UpdateCanvasInput struct {
	Title       *string         `json:"title" binding:"omitempty,max=120"`
	Description *string         `json:"description" binding:"omitempty,max=500"`
	Nodes       json.RawMessage `json:"nodes"`
	Edges       json.RawMessage `json:"edges"`
	Viewport    json.RawMessage `json:"viewport"`
	Tags        *[]string       `json:"tags" binding:"omitempty,max=10,dive,max=20"`
	IsPublic    *bool           `json:"is_public"`
	ShareMode   *string         `json:"share_mode" binding:"omitempty,max=16"`
}

type ForkCanvasInput struct {
	Title string `json:"title" binding:"omitempty,max=120"`
}

type ListCanvasQuery struct {
	Scope    string // mine（默认）| public
	Keyword  string
	Tag      string
	Page     int
	PageSize int
}

// ---------- 出参 ----------

type CanvasOwnerView struct {
	IsGuest bool  `json:"is_guest"`
	UserID  *uint `json:"user_id,omitempty"`
}

type CanvasPermissions struct {
	IsOwner   bool `json:"is_owner"`
	CanEdit   bool `json:"can_edit"`
	CanDelete bool `json:"can_delete"`
	CanFork   bool `json:"can_fork"`
	CanShare  bool `json:"can_share"`
}

// CanvasView 是画布详情，前端可直接喂给 React Flow：
//
//	reactFlowInstance.setNodes(view.nodes)
//	reactFlowInstance.setEdges(view.edges)
//	reactFlowInstance.setViewport(view.viewport)
type CanvasView struct {
	ID           uint              `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Nodes        json.RawMessage   `json:"nodes"`
	Edges        json.RawMessage   `json:"edges"`
	Viewport     json.RawMessage   `json:"viewport"`
	Tags         []string          `json:"tags"`
	IsPublic     bool              `json:"is_public"`
	ShareMode    string            `json:"share_mode"`
	ForkedFrom   *uint             `json:"forked_from,omitempty"`
	NodeCount    int               `json:"node_count"`
	EdgeCount    int               `json:"edge_count"`
	LikeCount    int               `json:"like_count"`
	CommentCount int               `json:"comment_count"`
	PublishedAt  *time.Time        `json:"published_at,omitempty"`
	Owner        CanvasOwnerView   `json:"owner"`
	Permissions  CanvasPermissions `json:"permissions"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// CanvasSummary 是列表项，不含 nodes/edges，避免列表接口传输过大。
type CanvasSummary struct {
	ID           uint              `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Tags         []string          `json:"tags"`
	IsPublic     bool              `json:"is_public"`
	ShareMode    string            `json:"share_mode"`
	ForkedFrom   *uint             `json:"forked_from,omitempty"`
	NodeCount    int               `json:"node_count"`
	EdgeCount    int               `json:"edge_count"`
	LikeCount    int               `json:"like_count"`
	CommentCount int               `json:"comment_count"`
	PublishedAt  *time.Time        `json:"published_at,omitempty"`
	Owner        CanvasOwnerView   `json:"owner"`
	Permissions  CanvasPermissions `json:"permissions"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type CanvasList struct {
	Items    []CanvasSummary `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	HasMore  bool            `json:"has_more"`
}

// ---------- Create ----------

func (s *CanvasService) Create(ctx context.Context, id *token.Identity, in CreateCanvasInput) (*CanvasView, error) {
	if id == nil || id.OwnerKey() == "" {
		return nil, apperr.New(apperr.CodeUnauthorized, "")
	}

	doc, err := normalizeDocument(in.Nodes, in.Edges, in.Viewport)
	if err != nil {
		return nil, err
	}
	tags, err := normalizeTags(in.Tags)
	if err != nil {
		return nil, err
	}
	isPublic, shareMode, err := normalizeVisibility(in.IsPublic, in.ShareMode)
	if err != nil {
		return nil, err
	}

	canvas := &model.Canvas{
		OwnerKey:    id.OwnerKey(),
		Title:       trimTitle(in.Title),
		Description: strings.TrimSpace(in.Description),
		Nodes:       datatypes.JSON(doc.Nodes),
		Edges:       datatypes.JSON(doc.Edges),
		Viewport:    datatypes.JSON(doc.Viewport),
		Tags:        encodeTags(tags),
		IsPublic:    isPublic,
		ShareMode:   shareMode,
		NodeCount:   doc.NodeCount,
		EdgeCount:   doc.EdgeCount,
	}

	if err := s.db.WithContext(ctx).Create(canvas).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "创建画布失败", err)
	}

	slog.Info("canvas created", "canvas_id", canvas.ID, "owner_key", canvas.OwnerKey,
		"nodes", canvas.NodeCount, "edges", canvas.EdgeCount, "share_mode", canvas.ShareMode)
	return canvasView(canvas, id), nil
}

// ---------- Get ----------

func (s *CanvasService) Get(ctx context.Context, id *token.Identity, canvasID uint) (*CanvasView, error) {
	canvas, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	if !canReadCanvas(canvas, id) {
		return nil, apperr.New(apperr.CodeForbidden, "无权访问该画布")
	}
	return canvasView(canvas, id), nil
}

// ---------- Update ----------

func (s *CanvasService) Update(ctx context.Context, id *token.Identity, canvasID uint, in UpdateCanvasInput) (*CanvasView, error) {
	canvas, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	if !canvas.IsOwner(id.OwnerKey()) {
		return nil, apperr.New(apperr.CodeForbidden, "只有画布所有者可以编辑")
	}

	updates := map[string]any{}

	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return nil, apperr.New(apperr.CodeInvalidParam, "标题不能为空")
		}
		updates["title"] = trimTitle(title)
	}
	if in.Description != nil {
		updates["description"] = strings.TrimSpace(*in.Description)
	}
	if in.Tags != nil {
		tags, err := normalizeTags(*in.Tags)
		if err != nil {
			return nil, err
		}
		updates["tags"] = encodeTags(tags)
	}

	// nodes / edges 必须成对提交：单改一半会让连线指向已删除的节点。
	if (in.Nodes == nil) != (in.Edges == nil) {
		return nil, apperr.New(apperr.CodeInvalidParam, "更新画布图形时需同时提交 nodes 与 edges")
	}
	if in.Nodes != nil {
		viewport := in.Viewport
		if viewport == nil {
			viewport = json.RawMessage(canvas.Viewport)
		}
		doc, err := normalizeDocument(in.Nodes, in.Edges, viewport)
		if err != nil {
			return nil, err
		}
		updates["nodes"] = datatypes.JSON(doc.Nodes)
		updates["edges"] = datatypes.JSON(doc.Edges)
		updates["viewport"] = datatypes.JSON(doc.Viewport)
		updates["node_count"] = doc.NodeCount
		updates["edge_count"] = doc.EdgeCount
	} else if in.Viewport != nil {
		viewport, err := normalizeViewport(in.Viewport)
		if err != nil {
			return nil, err
		}
		updates["viewport"] = datatypes.JSON(viewport)
	}

	// 分享设置：任一项变化都重新归一化，避免 is_public 与 share_mode 打架。
	if in.IsPublic != nil || in.ShareMode != nil {
		isPublic := canvas.IsPublic
		shareMode := canvas.ShareMode

		if in.IsPublic != nil {
			isPublic = *in.IsPublic
			// 只取消发布、不动模式时，public 必须降级，否则会被 normalize 推回公开
			if !isPublic && in.ShareMode == nil && shareMode == model.ShareModePublic {
				shareMode = model.ShareModePrivate
			}
		}
		if in.ShareMode != nil {
			shareMode = *in.ShareMode
			// 只改模式且未显式传 is_public 时，以模式为准推导是否公开
			if in.IsPublic == nil {
				isPublic = false
			}
		}

		normalizedPublic, normalizedMode, err := normalizeVisibility(isPublic, shareMode)
		if err != nil {
			return nil, err
		}
		updates["is_public"] = normalizedPublic
		updates["share_mode"] = normalizedMode
	}

	if len(updates) == 0 {
		return canvasView(canvas, id), nil
	}

	if err := s.db.WithContext(ctx).Model(canvas).Updates(updates).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "保存画布失败", err)
	}

	updated, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	slog.Info("canvas updated", "canvas_id", canvasID, "fields", len(updates))
	return canvasView(updated, id), nil
}

// ---------- Delete ----------

func (s *CanvasService) Delete(ctx context.Context, id *token.Identity, canvasID uint) error {
	canvas, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return err
	}
	if !canvas.IsOwner(id.OwnerKey()) {
		return apperr.New(apperr.CodeForbidden, "只有画布所有者可以删除")
	}
	if err := s.db.WithContext(ctx).Delete(canvas).Error; err != nil {
		return apperr.Wrap(apperr.CodeInternal, "删除画布失败", err)
	}
	slog.Info("canvas deleted", "canvas_id", canvasID, "owner_key", canvas.OwnerKey)
	return nil
}

// ---------- Fork ----------

func (s *CanvasService) Fork(ctx context.Context, id *token.Identity, canvasID uint, in ForkCanvasInput) (*CanvasView, error) {
	source, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	if !canReadCanvas(source, id) {
		return nil, apperr.New(apperr.CodeForbidden, "无权访问该画布")
	}

	title := trimTitle(in.Title)
	if strings.TrimSpace(in.Title) == "" {
		title = trimTitle(source.Title + " 副本")
	}
	return s.forkFrom(ctx, id, source, title)
}

// forkFrom 被画布 Fork 与分享链接 Fork 共用：调用方已完成权限校验。
func (s *CanvasService) forkFrom(ctx context.Context, id *token.Identity, source *model.Canvas, title string) (*CanvasView, error) {
	canvas := &model.Canvas{
		OwnerKey:    id.OwnerKey(),
		Title:       title,
		Description: source.Description,
		Nodes:       source.Nodes,
		Edges:       source.Edges,
		Viewport:    source.Viewport,
		Tags:        source.Tags,
		IsPublic:    false,
		ShareMode:   model.ShareModePrivate,
		ForkedFrom:  &source.ID,
		NodeCount:   source.NodeCount,
		EdgeCount:   source.EdgeCount,
	}
	if err := s.db.WithContext(ctx).Create(canvas).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "分叉画布失败", err)
	}

	slog.Info("canvas forked", "source_id", source.ID, "new_id", canvas.ID, "owner_key", canvas.OwnerKey)
	return canvasView(canvas, id), nil
}

// ---------- List ----------

func (s *CanvasService) List(ctx context.Context, id *token.Identity, q ListCanvasQuery) (*CanvasList, error) {
	scope := strings.ToLower(strings.TrimSpace(q.Scope))
	switch scope {
	case "", "mine":
		scope = "mine"
	case "public":
	default:
		return nil, apperr.New(apperr.CodeInvalidParam, "scope 只能是 mine / public")
	}

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}

	filters := func(db *gorm.DB) *gorm.DB {
		if scope == "public" {
			db = db.Where("is_public = ?", true)
		} else {
			db = db.Where("owner_key = ?", id.OwnerKey())
		}
		if kw := strings.TrimSpace(q.Keyword); kw != "" {
			db = db.Where("title LIKE ?", "%"+kw+"%")
		}
		if tag := strings.TrimSpace(q.Tag); tag != "" {
			// tags 是 JSON 数组，这里按完整字符串匹配，避免 "go" 命中 "golang"
			db = db.Where("tags LIKE ?", "%\""+tag+"\"%")
		}
		return db
	}

	base := s.db.WithContext(ctx).Model(&model.Canvas{})

	var total int64
	if err := filters(base.Session(&gorm.Session{})).Count(&total).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "统计画布失败", err)
	}

	var canvases []model.Canvas
	err := filters(base.Session(&gorm.Session{})).
		Order("updated_at DESC").Order("id DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&canvases).Error
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "查询画布列表失败", err)
	}

	items := make([]CanvasSummary, 0, len(canvases))
	for i := range canvases {
		items = append(items, canvasSummary(&canvases[i], id))
	}

	return &CanvasList{
		Items:    items,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
		HasMore:  int64(q.Page*q.PageSize) < total,
	}, nil
}

// ---------- helpers ----------

func (s *CanvasService) findCanvas(ctx context.Context, canvasID uint) (*model.Canvas, error) {
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
	return &canvas, nil
}

func canReadCanvas(c *model.Canvas, id *token.Identity) bool {
	if id == nil {
		return c.ReadableByOthers()
	}
	return c.IsOwner(id.OwnerKey()) || c.ReadableByOthers()
}

func canvasView(c *model.Canvas, id *token.Identity) *CanvasView {
	return &CanvasView{
		ID:           c.ID,
		Title:        c.Title,
		Description:  c.Description,
		Nodes:        rawOrEmptyArray(c.Nodes),
		Edges:        rawOrEmptyArray(c.Edges),
		Viewport:     rawOrDefaultViewport(c.Viewport),
		Tags:         decodeTags(c.Tags),
		IsPublic:     c.IsPublic,
		ShareMode:    c.ShareMode,
		ForkedFrom:   c.ForkedFrom,
		NodeCount:    c.NodeCount,
		EdgeCount:    c.EdgeCount,
		LikeCount:    c.LikeCount,
		CommentCount: c.CommentCount,
		PublishedAt:  c.PublishedAt,
		Owner:        ownerView(c.OwnerKey),
		Permissions:  permissionsOf(c, id),
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func canvasSummary(c *model.Canvas, id *token.Identity) CanvasSummary {
	return CanvasSummary{
		ID:           c.ID,
		Title:        c.Title,
		Description:  c.Description,
		Tags:         decodeTags(c.Tags),
		IsPublic:     c.IsPublic,
		ShareMode:    c.ShareMode,
		ForkedFrom:   c.ForkedFrom,
		NodeCount:    c.NodeCount,
		EdgeCount:    c.EdgeCount,
		LikeCount:    c.LikeCount,
		CommentCount: c.CommentCount,
		PublishedAt:  c.PublishedAt,
		Owner:        ownerView(c.OwnerKey),
		Permissions:  permissionsOf(c, id),
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func permissionsOf(c *model.Canvas, id *token.Identity) CanvasPermissions {
	isOwner := id != nil && c.IsOwner(id.OwnerKey())
	canRead := canReadCanvas(c, id)
	return CanvasPermissions{
		IsOwner:   isOwner,
		CanEdit:   isOwner,
		CanDelete: isOwner,
		CanFork:   canRead,
		CanShare:  isOwner,
	}
}

func ownerView(ownerKey string) CanvasOwnerView {
	if strings.HasPrefix(ownerKey, "u:") {
		if uid64, err := strconv.ParseUint(ownerKey[2:], 10, 64); err == nil && uid64 > 0 {
			uid := uint(uid64)
			return CanvasOwnerView{UserID: &uid}
		}
	}
	return CanvasOwnerView{IsGuest: true}
}

func rawOrEmptyArray(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("[]")
	}
	return json.RawMessage(raw)
}

func rawOrDefaultViewport(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{"x":0,"y":0,"zoom":1}`)
	}
	return json.RawMessage(raw)
}
