package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/apperr"
	"github.com/nodal/nodal-server/llm"
	"github.com/nodal/nodal-server/middleware"
	"github.com/nodal/nodal-server/response"
	"github.com/nodal/nodal-server/service"
)

// ModelHandler 提供模型目录与用户 API Key 管理。
type ModelHandler struct {
	router *llm.Router
	creds  *service.CredentialService
}

func NewModelHandler(router *llm.Router, creds *service.CredentialService) *ModelHandler {
	return &ModelHandler{router: router, creds: creds}
}

// CatalogItem 是模型下拉框所需的全部信息。
type CatalogItem struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Provider    string `json:"provider"`
	MaxTokens   int    `json:"max_tokens"`
	RequiresKey bool   `json:"requires_key"`
	Configured  bool   `json:"configured"`
}

// Catalog GET /api/model/catalog
func (h *ModelHandler) Catalog(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	specs := h.router.List()
	items := make([]CatalogItem, 0, len(specs))
	for _, spec := range specs {
		items = append(items, CatalogItem{
			ID:          spec.ID,
			Label:       spec.Label,
			Provider:    spec.Provider,
			MaxTokens:   spec.MaxTokens,
			RequiresKey: spec.RequiresKey,
			Configured:  h.creds.ConfiguredFor(c.Request.Context(), id.OwnerKey(), spec.Provider),
		})
	}
	response.OK(c, gin.H{"items": items})
}

// ListCredentials GET /api/model/credentials
func (h *ModelHandler) ListCredentials(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	list, err := h.creds.List(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, list)
}

// SaveCredential PUT /api/model/credentials/:provider
func (h *ModelHandler) SaveCredential(c *gin.Context) {
	provider := c.Param("provider")
	var in service.SaveCredentialInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	view, err := h.creds.Save(c.Request.Context(), id, provider, in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// DeleteCredential DELETE /api/model/credentials/:provider
func (h *ModelHandler) DeleteCredential(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.Error(apperr.New(apperr.CodeInvalidParam, "provider 不能为空"))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	if err := h.creds.Delete(c.Request.Context(), id, provider); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"provider": provider, "deleted": true})
}
