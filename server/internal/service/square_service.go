package service

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/model"
	"github.com/nodal/nodal-server/internal/token"
)

// SquareService 负责公共广场：发布/下架、广场列表、点赞、评论。
//
// 约定：
//   - 发布/下架：仅画布所有者，且必须是注册用户（游客画布不能进广场）；
//   - 广场列表：公开可读，带 token 时额外返回「我是否点过赞」；
//   - 点赞/评论：仅注册用户，且只能对「已发布」的画布操作。
type SquareService struct {
	db *gorm.DB
}

func NewSquareService(db *gorm.DB) *SquareService {
	return &SquareService{db: db}
}

// ---------- 入参/出参 ----------

type SquareQuery struct {
	Tag      string
	Keyword  string
	Sort     string // latest（默认）| hot
	Page     int
	PageSize int
}

type SquareAuthor struct {
	UserID    *uint  `json:"user_id,omitempty"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
	IsGuest   bool   `json:"is_guest"`
}

type SquareItem struct {
	ID           uint              `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Tags         []string          `json:"tags"`
	ForkedFrom   *uint             `json:"forked_from,omitempty"`
	NodeCount    int               `json:"node_count"`
	EdgeCount    int               `json:"edge_count"`
	LikeCount    int               `json:"like_count"`
	CommentCount int               `json:"comment_count"`
	Liked        bool              `json:"liked"`
	Author       SquareAuthor      `json:"author"`
	Permissions  CanvasPermissions `json:"permissions"`
	PublishedAt  *time.Time        `json:"published_at,omitempty"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type SquareList struct {
	Items    []SquareItem `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	HasMore  bool         `json:"has_more"`
}

type LikeResult struct {
	CanvasID  uint `json:"canvas_id"`
	Liked     bool `json:"liked"`
	LikeCount int  `json:"like_count"`
}

type CommentAuthor struct {
	UserID    uint   `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type CommentView struct {
	ID        uint          `json:"id"`
	CanvasID  uint          `json:"canvas_id"`
	Content   string        `json:"content"`
	Author    CommentAuthor `json:"author"`
	CreatedAt time.Time     `json:"created_at"`
	Mine      bool          `json:"mine"`
}

type CommentList struct {
	Items    []CommentView `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	HasMore  bool          `json:"has_more"`
}

// ---------- 发布 / 下架 ----------

func (s *SquareService) Publish(ctx context.Context, id *token.Identity, canvasID uint) (*CanvasView, error) {
	canvas, err := s.findOwnedCanvas(ctx, id, canvasID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	updates := map[string]any{
		"is_public":    true,
		"share_mode":   model.ShareModePublic,
		"published_at": now,
	}
	if err := s.db.WithContext(ctx).Model(canvas).Updates(updates).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "发布画布失败", err)
	}

	updated, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	slog.Info("canvas published", "canvas_id", canvasID, "owner_key", canvas.OwnerKey)
	return canvasView(updated, id), nil
}

func (s *SquareService) Unpublish(ctx context.Context, id *token.Identity, canvasID uint) (*CanvasView, error) {
	canvas, err := s.findOwnedCanvas(ctx, id, canvasID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{
		"is_public":    false,
		"share_mode":   model.ShareModePrivate,
		"published_at": nil,
	}
	if err := s.db.WithContext(ctx).Model(canvas).Updates(updates).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "下架画布失败", err)
	}

	updated, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	slog.Info("canvas unpublished", "canvas_id", canvasID, "owner_key", canvas.OwnerKey)
	return canvasView(updated, id), nil
}

// ---------- 广场列表 ----------

func (s *SquareService) List(ctx context.Context, id *token.Identity, q SquareQuery) (*SquareList, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}

	sortMode := strings.ToLower(strings.TrimSpace(q.Sort))
	switch sortMode {
	case "", "latest":
		sortMode = "latest"
	case "hot":
	default:
		return nil, apperr.New(apperr.CodeInvalidParam, "sort 只能是 latest / hot")
	}

	base := s.db.WithContext(ctx).Model(&model.Canvas{}).
		Where("is_public = ? AND published_at IS NOT NULL", true)
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		base = base.Where("title LIKE ?", "%"+kw+"%")
	}
	if tag := strings.TrimSpace(q.Tag); tag != "" {
		base = base.Where("tags LIKE ?", "%\""+tag+"\"%")
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "统计广场画布失败", err)
	}

	var rows []model.Canvas
	db := base.Session(&gorm.Session{})
	if sortMode == "hot" {
		db = db.Order("like_count DESC").Order("comment_count DESC")
	}
	db = db.Order("published_at DESC").Order("id DESC")

	if err := db.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "查询广场列表失败", err)
	}

	authorMap := s.authorsOf(ctx, rows)
	likedMap := s.likedMap(ctx, id, rows)

	items := make([]SquareItem, 0, len(rows))
	for i := range rows {
		c := &rows[i]
		items = append(items, SquareItem{
			ID:           c.ID,
			Title:        c.Title,
			Description:  c.Description,
			Tags:         decodeTags(c.Tags),
			ForkedFrom:   c.ForkedFrom,
			NodeCount:    c.NodeCount,
			EdgeCount:    c.EdgeCount,
			LikeCount:    c.LikeCount,
			CommentCount: c.CommentCount,
			Liked:        likedMap[c.ID],
			Author:       authorMap[c.OwnerKey],
			Permissions:  permissionsOf(c, id),
			PublishedAt:  c.PublishedAt,
			UpdatedAt:    c.UpdatedAt,
		})
	}

	return &SquareList{
		Items:    items,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
		HasMore:  int64(q.Page*q.PageSize) < total,
	}, nil
}

// ---------- 点赞 ----------

// Like 切换点赞状态：已赞则取消，未赞则点赞。
func (s *SquareService) Like(ctx context.Context, id *token.Identity, canvasID uint) (*LikeResult, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "点赞需要登录后使用")
	}
	canvas, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	if err := requirePublished(canvas); err != nil {
		return nil, err
	}

	userKey := id.OwnerKey()
	liked := false
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var like model.CanvasLike
		err := tx.Where("canvas_id = ? AND user_key = ?", canvasID, userKey).First(&like).Error
		switch {
		case err == nil:
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Canvas{}).Where("id = ?", canvasID).
				UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error; err != nil {
				return err
			}
			return nil
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(&model.CanvasLike{CanvasID: canvasID, UserKey: userKey}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Canvas{}).Where("id = ?", canvasID).
				UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
				return err
			}
			liked = true
			return nil
		default:
			return err
		}
	})
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "操作点赞失败", err)
	}

	updated, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	return &LikeResult{CanvasID: canvasID, Liked: liked, LikeCount: updated.LikeCount}, nil
}

// ---------- 评论 ----------

func (s *SquareService) CreateComment(ctx context.Context, id *token.Identity, canvasID uint, content string) (*CommentView, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "评论需要登录后使用")
	}
	canvas, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	if err := requirePublished(canvas); err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, apperr.New(apperr.CodeInvalidParam, "评论内容不能为空")
	}
	if len([]rune(content)) > 500 {
		return nil, apperr.New(apperr.CodeInvalidParam, "评论最多 500 字")
	}

	comment := &model.CanvasComment{
		CanvasID: canvasID,
		UserKey:  id.OwnerKey(),
		UserID:   id.UserID,
		Content:  content,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		return tx.Model(&model.Canvas{}).Where("id = ?", canvasID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "发表评论失败", err)
	}

	author := s.authorOf(ctx, comment.UserID)
	slog.Info("comment created", "canvas_id", canvasID, "comment_id", comment.ID, "user_id", id.UserID)
	return &CommentView{
		ID:        comment.ID,
		CanvasID:  comment.CanvasID,
		Content:   comment.Content,
		Author:    author,
		CreatedAt: comment.CreatedAt,
		Mine:      true,
	}, nil
}

func (s *SquareService) ListComments(ctx context.Context, id *token.Identity, canvasID uint, page, pageSize int) (*CommentList, error) {
	canvas, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	if !canReadCanvas(canvas, id) {
		return nil, apperr.New(apperr.CodeForbidden, "无权查看该画布的评论")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	if err := s.db.WithContext(ctx).Model(&model.CanvasComment{}).
		Where("canvas_id = ?", canvasID).Count(&total).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "统计评论失败", err)
	}

	var rows []model.CanvasComment
	if err := s.db.WithContext(ctx).
		Where("canvas_id = ?", canvasID).
		Order("created_at DESC").Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "查询评论失败", err)
	}

	userIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}
	users := s.usersByID(ctx, userIDs)

	items := make([]CommentView, 0, len(rows))
	for _, row := range rows {
		author := CommentAuthor{UserID: row.UserID, Nickname: fallbackNickname(row.UserID)}
		if user, ok := users[row.UserID]; ok {
			author.Nickname = displayNickname(user)
			author.AvatarURL = user.AvatarURL
		}
		items = append(items, CommentView{
			ID:        row.ID,
			CanvasID:  row.CanvasID,
			Content:   row.Content,
			Author:    author,
			CreatedAt: row.CreatedAt,
			Mine:      id != nil && id.IsUser() && row.UserID == id.UserID,
		})
	}

	return &CommentList{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  int64(page*pageSize) < total,
	}, nil
}

// DeleteComment 允许评论作者或画布所有者删除。
func (s *SquareService) DeleteComment(ctx context.Context, id *token.Identity, canvasID, commentID uint) error {
	if id == nil || id.OwnerKey() == "" {
		return apperr.New(apperr.CodeUnauthorized, "")
	}
	canvas, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return err
	}
	var comment model.CanvasComment
	err = s.db.WithContext(ctx).Where("id = ? AND canvas_id = ?", commentID, canvasID).First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.CodeCommentNotFound, "")
		}
		return apperr.Wrap(apperr.CodeInternal, "查询评论失败", err)
	}

	allowed := comment.UserKey == id.OwnerKey() || canvas.IsOwner(id.OwnerKey())
	if !allowed {
		return apperr.New(apperr.CodeForbidden, "只能删除自己的评论")
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&comment).Error; err != nil {
			return err
		}
		return tx.Model(&model.Canvas{}).Where("id = ?", canvasID).
			UpdateColumn("comment_count", gorm.Expr("CASE WHEN comment_count > 0 THEN comment_count - 1 ELSE 0 END")).Error
	})
	if err != nil {
		return apperr.Wrap(apperr.CodeInternal, "删除评论失败", err)
	}
	return nil
}

// ---------- helpers ----------

func (s *SquareService) findCanvas(ctx context.Context, canvasID uint) (*model.Canvas, error) {
	var canvas model.Canvas
	if canvasID == 0 {
		return nil, apperr.New(apperr.CodeInvalidParam, "画布 ID 不合法")
	}
	if err := s.db.WithContext(ctx).First(&canvas, canvasID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.New(apperr.CodeNotFound, "画布不存在或已删除")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "查询画布失败", err)
	}
	return &canvas, nil
}

func (s *SquareService) findOwnedCanvas(ctx context.Context, id *token.Identity, canvasID uint) (*model.Canvas, error) {
	if id == nil || id.OwnerKey() == "" {
		return nil, apperr.New(apperr.CodeUnauthorized, "")
	}
	if !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "发布到广场需要登录账号")
	}
	canvas, err := s.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	if !canvas.IsOwner(id.OwnerKey()) {
		return nil, apperr.New(apperr.CodeForbidden, "只有画布所有者可以发布或下架")
	}
	return canvas, nil
}

func requirePublished(c *model.Canvas) error {
	if !c.IsPublic || c.PublishedAt == nil {
		return apperr.New(apperr.CodeCanvasNotPublished, "")
	}
	return nil
}

func (s *SquareService) authorsOf(ctx context.Context, rows []model.Canvas) map[string]SquareAuthor {
	out := make(map[string]SquareAuthor, len(rows))
	seen := make(map[uint]struct{})
	for _, row := range rows {
		if uid, ok := userIDFromOwnerKey(row.OwnerKey); ok {
			seen[uid] = struct{}{}
		}
	}
	users := s.usersByID(ctx, keysOf(seen))

	for _, row := range rows {
		uid, ok := userIDFromOwnerKey(row.OwnerKey)
		if !ok {
			out[row.OwnerKey] = SquareAuthor{Nickname: "游客", IsGuest: true}
			continue
		}
		author := SquareAuthor{UserID: &uid, Nickname: fallbackNickname(uid)}
		if user, exists := users[uid]; exists {
			author.Nickname = displayNickname(user)
			author.AvatarURL = user.AvatarURL
		}
		out[row.OwnerKey] = author
	}
	return out
}

func (s *SquareService) likedMap(ctx context.Context, id *token.Identity, rows []model.Canvas) map[uint]bool {
	out := make(map[uint]bool, len(rows))
	if id == nil || !id.IsUser() || len(rows) == 0 {
		return out
	}
	canvasIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		canvasIDs = append(canvasIDs, row.ID)
	}
	var likes []model.CanvasLike
	if err := s.db.WithContext(ctx).
		Where("user_key = ? AND canvas_id IN ?", id.OwnerKey(), canvasIDs).
		Find(&likes).Error; err != nil {
		return out
	}
	for _, like := range likes {
		out[like.CanvasID] = true
	}
	return out
}

func (s *SquareService) usersByID(ctx context.Context, ids []uint) map[uint]*model.User {
	out := make(map[uint]*model.User, len(ids))
	if len(ids) == 0 {
		return out
	}
	var users []model.User
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return out
	}
	for i := range users {
		out[users[i].ID] = &users[i]
	}
	return out
}

func (s *SquareService) authorOf(ctx context.Context, userID uint) CommentAuthor {
	author := CommentAuthor{UserID: userID, Nickname: fallbackNickname(userID)}
	if user, ok := s.usersByID(ctx, []uint{userID})[userID]; ok {
		author.Nickname = displayNickname(user)
		author.AvatarURL = user.AvatarURL
	}
	return author
}

func displayNickname(u *model.User) string {
	if strings.TrimSpace(u.Nickname) != "" {
		return u.Nickname
	}
	return fallbackNickname(u.ID)
}

func fallbackNickname(uid uint) string {
	return "用户#" + strconv.FormatUint(uint64(uid), 10)
}

func userIDFromOwnerKey(ownerKey string) (uint, bool) {
	if !strings.HasPrefix(ownerKey, "u:") {
		return 0, false
	}
	v, err := strconv.ParseUint(ownerKey[2:], 10, 64)
	if err != nil || v == 0 {
		return 0, false
	}
	return uint(v), true
}

func keysOf(set map[uint]struct{}) []uint {
	out := make([]uint, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
