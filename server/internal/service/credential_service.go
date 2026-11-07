package service

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/llm"
	"github.com/nodal/nodal-server/internal/model"
	"github.com/nodal/nodal-server/internal/secretbox"
	"github.com/nodal/nodal-server/internal/token"
)

// CredentialService 管理用户自有的模型 API Key：加密落库、脱敏展示、按 provider 取用。
type CredentialService struct {
	db     *gorm.DB
	box    *secretbox.Box
	router *llm.Router
}

func NewCredentialService(db *gorm.DB, box *secretbox.Box, router *llm.Router) *CredentialService {
	return &CredentialService{db: db, box: box, router: router}
}

type SaveCredentialInput struct {
	APIKey  string `json:"api_key" binding:"required,min=8,max=4096"`
	BaseURL string `json:"base_url" binding:"omitempty,max=255,url"`
}

// CredentialView 永远不包含完整明文，只给尾号提示与配置状态。
type CredentialView struct {
	Provider   string     `json:"provider"`
	Label      string     `json:"label"`
	Configured bool       `json:"configured"`
	KeyHint    string     `json:"key_hint"`
	BaseURL    string     `json:"base_url"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

type CredentialList struct {
	Items []CredentialView `json:"items"`
}

func (s *CredentialService) List(ctx context.Context, id *token.Identity) (*CredentialList, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法配置模型，请先登录")
	}

	var rows []model.ModelCredential
	if err := s.db.WithContext(ctx).Where("owner_key = ?", id.OwnerKey()).Find(&rows).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "查询模型凭证失败", err)
	}

	byProvider := make(map[string]model.ModelCredential, len(rows))
	for _, row := range rows {
		byProvider[row.Provider] = row
	}

	providers := s.router.Providers()
	sort.Strings(providers)
	items := make([]CredentialView, 0, len(providers))
	for _, provider := range providers {
		view := CredentialView{Provider: provider, Label: providerLabel(provider)}
		if row, ok := byProvider[provider]; ok {
			updated := row.UpdatedAt
			view.Configured = true
			view.KeyHint = row.KeyHint
			view.BaseURL = row.BaseURL
			view.UpdatedAt = &updated
		}
		items = append(items, view)
	}
	return &CredentialList{Items: items}, nil
}

func (s *CredentialService) Save(ctx context.Context, id *token.Identity, provider string, in SaveCredentialInput) (*CredentialView, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法配置模型，请先登录")
	}

	provider = strings.ToLower(strings.TrimSpace(provider))
	if !s.router.SupportsProvider(provider) {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "不支持的 provider %q", provider)
	}
	if !s.router.RequiresKey(provider) {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "provider %s 无需配置 API Key", provider)
	}

	apiKey := strings.TrimSpace(in.APIKey)
	if len(apiKey) < 8 {
		return nil, apperr.New(apperr.CodeInvalidParam, "API Key 太短，请检查是否粘贴完整")
	}

	encrypted, err := s.box.Encrypt(apiKey, secretbox.AAD(id.OwnerKey(), provider))
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "API Key 加密失败", err)
	}

	row := model.ModelCredential{
		OwnerKey:  id.OwnerKey(),
		Provider:  provider,
		APIKeyEnc: encrypted,
		KeyHint:   keyHint(apiKey),
		BaseURL:   strings.TrimRight(strings.TrimSpace(in.BaseURL), "/"),
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "owner_key"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"api_key_enc", "key_hint", "base_url", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "保存模型凭证失败", err)
	}

	slog.Info("model credential saved", "owner_key", id.OwnerKey(), "provider", provider, "key_hint", row.KeyHint)

	updated := row.UpdatedAt
	if updated.IsZero() {
		updated = time.Now().UTC()
	}
	return &CredentialView{
		Provider:   provider,
		Label:      providerLabel(provider),
		Configured: true,
		KeyHint:    row.KeyHint,
		BaseURL:    row.BaseURL,
		UpdatedAt:  &updated,
	}, nil
}

func (s *CredentialService) Delete(ctx context.Context, id *token.Identity, provider string) error {
	if id == nil || !id.IsUser() {
		return apperr.New(apperr.CodeForbidden, "游客无法配置模型，请先登录")
	}

	provider = strings.ToLower(strings.TrimSpace(provider))
	res := s.db.WithContext(ctx).
		Where("owner_key = ? AND provider = ?", id.OwnerKey(), provider).
		Delete(&model.ModelCredential{})
	if res.Error != nil {
		return apperr.Wrap(apperr.CodeInternal, "删除模型凭证失败", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperr.Newf(apperr.CodeNotFound, "未配置 provider %s 的 API Key", provider)
	}
	slog.Info("model credential deleted", "owner_key", id.OwnerKey(), "provider", provider)
	return nil
}

// CredentialFor 取得调用某个 provider 所需的凭证；mock 等免密钥 provider 返回空凭证。
func (s *CredentialService) CredentialFor(ctx context.Context, ownerKey, provider string) (llm.Credential, error) {
	defBase := s.router.DefaultBaseURL(provider)
	if !s.router.RequiresKey(provider) {
		return llm.Credential{Provider: provider, BaseURL: defBase}, nil
	}

	var row model.ModelCredential
	err := s.db.WithContext(ctx).
		Where("owner_key = ? AND provider = ?", ownerKey, provider).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return llm.Credential{}, apperr.Newf(apperr.CodeModelKeyMissing,
				"该模型需要先配置 %s 的 API Key", providerLabel(provider))
		}
		return llm.Credential{}, apperr.Wrap(apperr.CodeInternal, "查询模型凭证失败", err)
	}

	plain, err := s.box.Decrypt(row.APIKeyEnc, secretbox.AAD(ownerKey, provider))
	if err != nil {
		slog.Error("decrypt model credential failed", "owner_key", ownerKey, "provider", provider, "error", err)
		return llm.Credential{}, apperr.New(apperr.CodeInternal, "模型凭证解密失败，请重新保存 API Key")
	}

	base := row.BaseURL
	if base == "" {
		base = defBase
	}
	return llm.Credential{Provider: provider, APIKey: plain, BaseURL: base}, nil
}

// ConfiguredFor 用于模型目录展示：该 provider 是否已配置。
func (s *CredentialService) ConfiguredFor(ctx context.Context, ownerKey, provider string) bool {
	if !s.router.RequiresKey(provider) {
		return true
	}
	var count int64
	err := s.db.WithContext(ctx).Model(&model.ModelCredential{}).
		Where("owner_key = ? AND provider = ?", ownerKey, provider).
		Count(&count).Error
	return err == nil && count > 0
}

func keyHint(key string) string {
	if len(key) <= 4 {
		return "…"
	}
	last4 := key[len(key)-4:]
	if len(key) <= 8 {
		return "…" + last4
	}
	return "…" + last4
}

func providerLabel(provider string) string {
	switch provider {
	case "openai":
		return "OpenAI"
	case "deepseek":
		return "DeepSeek"
	case "moonshot":
		return "Moonshot"
	case "zhipu":
		return "智谱 AI"
	case "qwen":
		return "通义千问"
	case "anthropic":
		return "Anthropic"
	default:
		return provider
	}
}
