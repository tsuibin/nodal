package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/nodal/nodal-server/apperr"
	"github.com/nodal/nodal-server/llm"
	"github.com/nodal/nodal-server/model"
	"github.com/nodal/nodal-server/token"
)

// AgentService 管理 Agent 模板（name / system_prompt / model_id）。
// 模板归属于注册用户，按 (owner_key, name) 唯一。
type AgentService struct {
	db     *gorm.DB
	router *llm.Router
}

func NewAgentService(db *gorm.DB, router *llm.Router) *AgentService {
	return &AgentService{db: db, router: router}
}

type CreateAgentInput struct {
	Name         string `json:"name" binding:"required,max=80"`
	Description  string `json:"description" binding:"omitempty,max=300"`
	SystemPrompt string `json:"system_prompt" binding:"required,max=8000"`
	ModelID      string `json:"model_id" binding:"required,max=80"`
}

type UpdateAgentInput struct {
	Name         *string `json:"name" binding:"omitempty,max=80"`
	Description  *string `json:"description" binding:"omitempty,max=300"`
	SystemPrompt *string `json:"system_prompt" binding:"omitempty,max=8000"`
	ModelID      *string `json:"model_id" binding:"omitempty,max=80"`
}

// AgentView 是返回给前端的模板视图。
type AgentView struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	ModelID      string    `json:"model_id"`
	ModelLabel   string    `json:"model_label"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AgentList struct {
	Items []AgentView `json:"items"`
	Total int64       `json:"total"`
}

func (s *AgentService) Create(ctx context.Context, id *token.Identity, in CreateAgentInput) (*AgentView, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法创建 Agent，请先登录")
	}

	name, err := normalizeAgentName(in.Name)
	if err != nil {
		return nil, err
	}
	prompt := strings.TrimSpace(in.SystemPrompt)
	if prompt == "" {
		return nil, apperr.New(apperr.CodeInvalidParam, "system_prompt 不能为空")
	}
	if utf8.RuneCountInString(prompt) > 8000 {
		return nil, apperr.New(apperr.CodeInvalidParam, "system_prompt 最多 8000 字")
	}
	modelID, err := s.validateModel(in.ModelID)
	if err != nil {
		return nil, err
	}

	agent := &model.AgentTemplate{
		OwnerKey:     id.OwnerKey(),
		Name:         name,
		Description:  strings.TrimSpace(in.Description),
		SystemPrompt: prompt,
		ModelID:      modelID,
	}
	if err := s.db.WithContext(ctx).Create(agent).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperr.New(apperr.CodeAgentNameTaken, "")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "创建 Agent 失败", err)
	}

	slog.Info("agent created", "agent_id", agent.ID, "owner_key", agent.OwnerKey, "model_id", agent.ModelID)
	return s.view(agent), nil
}

func (s *AgentService) List(ctx context.Context, id *token.Identity) (*AgentList, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法查看 Agent，请先登录")
	}

	var agents []model.AgentTemplate
	if err := s.db.WithContext(ctx).
		Where("owner_key = ?", id.OwnerKey()).
		Order("updated_at DESC").Order("id DESC").
		Find(&agents).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "查询 Agent 列表失败", err)
	}

	items := make([]AgentView, 0, len(agents))
	for i := range agents {
		items = append(items, *s.view(&agents[i]))
	}
	return &AgentList{Items: items, Total: int64(len(items))}, nil
}

func (s *AgentService) Get(ctx context.Context, id *token.Identity, agentID uint) (*AgentView, error) {
	agent, err := s.findOwned(ctx, id, agentID)
	if err != nil {
		return nil, err
	}
	return s.view(agent), nil
}

func (s *AgentService) Update(ctx context.Context, id *token.Identity, agentID uint, in UpdateAgentInput) (*AgentView, error) {
	agent, err := s.findOwned(ctx, id, agentID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if in.Name != nil {
		name, err := normalizeAgentName(*in.Name)
		if err != nil {
			return nil, err
		}
		updates["name"] = name
	}
	if in.Description != nil {
		updates["description"] = strings.TrimSpace(*in.Description)
	}
	if in.SystemPrompt != nil {
		prompt := strings.TrimSpace(*in.SystemPrompt)
		if prompt == "" {
			return nil, apperr.New(apperr.CodeInvalidParam, "system_prompt 不能为空")
		}
		updates["system_prompt"] = prompt
	}
	if in.ModelID != nil {
		modelID, err := s.validateModel(*in.ModelID)
		if err != nil {
			return nil, err
		}
		updates["model_id"] = modelID
	}

	if len(updates) == 0 {
		return s.view(agent), nil
	}
	if err := s.db.WithContext(ctx).Model(agent).Updates(updates).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperr.New(apperr.CodeAgentNameTaken, "")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "保存 Agent 失败", err)
	}

	updated, err := s.findOwned(ctx, id, agentID)
	if err != nil {
		return nil, err
	}
	return s.view(updated), nil
}

func (s *AgentService) Delete(ctx context.Context, id *token.Identity, agentID uint) error {
	agent, err := s.findOwned(ctx, id, agentID)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(agent).Error; err != nil {
		return apperr.Wrap(apperr.CodeInternal, "删除 Agent 失败", err)
	}
	slog.Info("agent deleted", "agent_id", agentID, "owner_key", agent.OwnerKey)
	return nil
}

// findOwned 只按「当前用户 + id」查询，别人的 Agent 对当前用户等价于不存在。
func (s *AgentService) findOwned(ctx context.Context, id *token.Identity, agentID uint) (*model.AgentTemplate, error) {
	if id == nil || !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法访问 Agent，请先登录")
	}
	if agentID == 0 {
		return nil, apperr.New(apperr.CodeInvalidParam, "Agent ID 不合法")
	}
	var agent model.AgentTemplate
	err := s.db.WithContext(ctx).Where("id = ? AND owner_key = ?", agentID, id.OwnerKey()).First(&agent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.New(apperr.CodeAgentNotFound, "")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "查询 Agent 失败", err)
	}
	return &agent, nil
}

func (s *AgentService) validateModel(modelID string) (string, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return "", apperr.New(apperr.CodeInvalidParam, "model_id 不能为空")
	}
	if _, _, err := s.router.Resolve(modelID); err != nil {
		return "", err
	}
	return modelID, nil
}

func (s *AgentService) view(a *model.AgentTemplate) *AgentView {
	label := a.ModelID
	if spec, _, err := s.router.Resolve(a.ModelID); err == nil {
		label = spec.Label
	}
	return &AgentView{
		ID:           a.ID,
		Name:         a.Name,
		Description:  a.Description,
		SystemPrompt: a.SystemPrompt,
		ModelID:      a.ModelID,
		ModelLabel:   label,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

func normalizeAgentName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperr.New(apperr.CodeInvalidParam, "Agent 名称不能为空")
	}
	if utf8.RuneCountInString(name) > 80 {
		return "", apperr.New(apperr.CodeInvalidParam, "Agent 名称最多 80 字")
	}
	return name, nil
}
