package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/nodal/nodal-server/apperr"
	"github.com/nodal/nodal-server/model"
	"github.com/nodal/nodal-server/token"
)

// ShareService 负责「带权限的分享链接」。
//
// 与 canvas.share_mode 的关系：
//   - share_mode=link/public 是「按画布 ID 可读」的粗粒度开关；
//   - 分享链接是随机 token 的能力凭证，额外区分 read / fork，可单独停用与设置过期，
//     并且即使画布是 private，只要链接有效仍可访问。
type ShareService struct {
	db       *gorm.DB
	canvases *CanvasService
}

func NewShareService(db *gorm.DB, canvases *CanvasService) *ShareService {
	return &ShareService{db: db, canvases: canvases}
}

type CreateShareLinkInput struct {
	Permission   string `json:"permission" binding:"omitempty,oneof=read fork"`
	ExpiresHours int    `json:"expires_hours" binding:"omitempty,min=1,max=720"`
}

type ShareLinkView struct {
	ID         uint       `json:"id"`
	CanvasID   uint       `json:"canvas_id"`
	Token      string     `json:"token"`
	Permission string     `json:"permission"`
	Enabled    bool       `json:"enabled"`
	Expired    bool       `json:"expired"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	// 前端路由约定；实际域名由前端拼接
	SharePath string `json:"share_path"`
}

// ShareView 是「分享链接解析」的结果：拿到的人可据此只读渲染，或在 fork 权限下分叉。
type ShareView struct {
	Token      string      `json:"token"`
	Permission string      `json:"permission"`
	CanRead    bool        `json:"can_read"`
	CanFork    bool        `json:"can_fork"`
	ExpiresAt  *time.Time  `json:"expires_at,omitempty"`
	Canvas     *CanvasView `json:"canvas"`
}

// Create 创建（或复用）一条分享链接，仅画布所有者可调用。
// 同一画布 + 同一权限下，若已有有效链接则直接复用，避免不断生成新链接让旧链接失效。
func (s *ShareService) Create(ctx context.Context, id *token.Identity, canvasID uint, in CreateShareLinkInput) (*ShareLinkView, error) {
	canvas, err := s.canvases.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	if id == nil || !canvas.IsOwner(id.OwnerKey()) {
		return nil, apperr.New(apperr.CodeForbidden, "只有画布所有者可以创建分享链接")
	}

	permission := strings.TrimSpace(in.Permission)
	if permission == "" {
		permission = model.SharePermissionRead
	}
	if permission != model.SharePermissionRead && permission != model.SharePermissionFork {
		return nil, apperr.New(apperr.CodeInvalidParam, "permission 只能是 read / fork")
	}

	now := time.Now().UTC()
	var existing model.CanvasShareLink
	err = s.db.WithContext(ctx).
		Where("canvas_id = ? AND permission = ? AND enabled = ?", canvasID, permission, true).
		Where("expires_at IS NULL OR expires_at > ?", now).
		Order("id ASC").First(&existing).Error
	if err == nil {
		return shareLinkView(&existing, now), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Wrap(apperr.CodeInternal, "查询分享链接失败", err)
	}

	var expiresAt *time.Time
	if in.ExpiresHours > 0 {
		t := now.Add(time.Duration(in.ExpiresHours) * time.Hour)
		expiresAt = &t
	}

	link := &model.CanvasShareLink{
		CanvasID:   canvasID,
		Permission: permission,
		CreatedBy:  id.OwnerKey(),
		Enabled:    true,
		ExpiresAt:  expiresAt,
	}

	for attempt := 0; attempt < 3; attempt++ {
		link.Token = token.NewID()
		if err := s.db.WithContext(ctx).Create(link).Error; err == nil {
			slog.Info("share link created", "canvas_id", canvasID, "permission", permission, "link_id", link.ID)
			return shareLinkView(link, now), nil
		}
	}
	return nil, apperr.New(apperr.CodeInternal, "生成分享链接失败，请重试")
}

// List 列出画布的全部分享链接（含已停用/过期），用于分享设置面板。
func (s *ShareService) List(ctx context.Context, id *token.Identity, canvasID uint) ([]ShareLinkView, error) {
	canvas, err := s.canvases.findCanvas(ctx, canvasID)
	if err != nil {
		return nil, err
	}
	if id == nil || !canvas.IsOwner(id.OwnerKey()) {
		return nil, apperr.New(apperr.CodeForbidden, "只有画布所有者可以查看分享链接")
	}

	var links []model.CanvasShareLink
	if err := s.db.WithContext(ctx).Where("canvas_id = ?", canvasID).
		Order("id DESC").Find(&links).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "查询分享链接失败", err)
	}

	now := time.Now().UTC()
	out := make([]ShareLinkView, 0, len(links))
	for i := range links {
		out = append(out, *shareLinkView(&links[i], now))
	}
	return out, nil
}

// Revoke 停用一条分享链接（软停用，保留记录便于追溯）。
func (s *ShareService) Revoke(ctx context.Context, id *token.Identity, canvasID, linkID uint) error {
	canvas, err := s.canvases.findCanvas(ctx, canvasID)
	if err != nil {
		return err
	}
	if id == nil || !canvas.IsOwner(id.OwnerKey()) {
		return apperr.New(apperr.CodeForbidden, "只有画布所有者可以停用分享链接")
	}

	var link model.CanvasShareLink
	err = s.db.WithContext(ctx).Where("id = ? AND canvas_id = ?", linkID, canvasID).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.CodeShareInvalid, "")
		}
		return apperr.Wrap(apperr.CodeInternal, "查询分享链接失败", err)
	}
	if err := s.db.WithContext(ctx).Model(&link).Update("enabled", false).Error; err != nil {
		return apperr.Wrap(apperr.CodeInternal, "停用分享链接失败", err)
	}
	return nil
}

// Resolve 校验分享 token 并返回画布内容；公开接口，不需要身份。
func (s *ShareService) Resolve(ctx context.Context, rawToken string) (*ShareView, error) {
	link, err := s.resolveLink(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	canvas, err := s.canvases.findCanvas(ctx, link.CanvasID)
	if err != nil {
		// 画布已删除时按「链接失效」处理，避免泄露删除状态
		return nil, apperr.New(apperr.CodeShareInvalid, "")
	}

	return &ShareView{
		Token:      link.Token,
		Permission: link.Permission,
		CanRead:    true,
		CanFork:    link.Permission == model.SharePermissionFork,
		ExpiresAt:  link.ExpiresAt,
		Canvas:     canvasView(canvas, nil),
	}, nil
}

// Fork 通过分享链接分叉画布：必须是 fork 权限；登录用户与游客都可以分叉。
func (s *ShareService) Fork(ctx context.Context, id *token.Identity, rawToken, title string) (*CanvasView, error) {
	link, err := s.resolveLink(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	if link.Permission != model.SharePermissionFork {
		return nil, apperr.New(apperr.CodeShareForbidden, "")
	}
	if id == nil || id.OwnerKey() == "" {
		return nil, apperr.New(apperr.CodeUnauthorized, "")
	}

	source, err := s.canvases.findCanvas(ctx, link.CanvasID)
	if err != nil {
		return nil, apperr.New(apperr.CodeShareInvalid, "")
	}

	title = trimTitle(title)
	if title == "" {
		title = trimTitle(source.Title + " 副本")
	}
	slog.Info("canvas forked from share link", "source_id", source.ID, "token", link.Token, "owner_key", id.OwnerKey())
	return s.canvases.forkFrom(ctx, id, source, title)
}

func (s *ShareService) resolveLink(ctx context.Context, rawToken string) (*model.CanvasShareLink, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" || len(rawToken) > 64 {
		return nil, apperr.New(apperr.CodeShareInvalid, "")
	}

	var link model.CanvasShareLink
	err := s.db.WithContext(ctx).Where("token = ?", rawToken).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.New(apperr.CodeShareInvalid, "")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "查询分享链接失败", err)
	}
	if !link.Enabled {
		return nil, apperr.New(apperr.CodeShareInvalid, "")
	}
	if link.IsExpired(time.Now().UTC()) {
		return nil, apperr.New(apperr.CodeShareExpired, "")
	}
	return &link, nil
}

func shareLinkView(link *model.CanvasShareLink, now time.Time) *ShareLinkView {
	return &ShareLinkView{
		ID:         link.ID,
		CanvasID:   link.CanvasID,
		Token:      link.Token,
		Permission: link.Permission,
		Enabled:    link.Enabled,
		Expired:    link.IsExpired(now),
		ExpiresAt:  link.ExpiresAt,
		CreatedAt:  link.CreatedAt,
		SharePath:  "/share/" + link.Token,
	}
}
