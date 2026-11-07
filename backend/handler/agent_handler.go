package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/middleware"
	"github.com/nodal/nodal-server/response"
	"github.com/nodal/nodal-server/service"
)

// AgentHandler 是 Agent 模板 CRUD 的 HTTP 适配层（仅注册用户可用）。
type AgentHandler struct {
	agents *service.AgentService
}

func NewAgentHandler(agents *service.AgentService) *AgentHandler {
	return &AgentHandler{agents: agents}
}

// Create POST /api/agent
func (h *AgentHandler) Create(c *gin.Context) {
	var in service.CreateAgentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	view, err := h.agents.Create(c.Request.Context(), id, in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// List GET /api/agent/list
func (h *AgentHandler) List(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	list, err := h.agents.List(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, list)
}

// Get GET /api/agent/:id
func (h *AgentHandler) Get(c *gin.Context) {
	agentID, err := parseUintParam(c, "id", "Agent ID 不合法")
	if err != nil {
		c.Error(err)
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	view, err := h.agents.Get(c.Request.Context(), id, agentID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// Update PUT /api/agent/:id
func (h *AgentHandler) Update(c *gin.Context) {
	agentID, err := parseUintParam(c, "id", "Agent ID 不合法")
	if err != nil {
		c.Error(err)
		return
	}

	var in service.UpdateAgentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	view, err := h.agents.Update(c.Request.Context(), id, agentID, in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// Delete DELETE /api/agent/:id
func (h *AgentHandler) Delete(c *gin.Context) {
	agentID, err := parseUintParam(c, "id", "Agent ID 不合法")
	if err != nil {
		c.Error(err)
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	if err := h.agents.Delete(c.Request.Context(), id, agentID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"id": agentID, "deleted": true})
}
