package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/internal/middleware"
	"github.com/nodal/nodal-server/internal/response"
	"github.com/nodal/nodal-server/internal/service"
)

// ShareHandler 处理分享链接的创建、停用、解析与 Fork。
type ShareHandler struct {
	shares *service.ShareService
}

func NewShareHandler(shares *service.ShareService) *ShareHandler {
	return &ShareHandler{shares: shares}
}

// Create POST /api/canvas/:id/share-link
func (h *ShareHandler) Create(c *gin.Context) {
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}
	var in service.CreateShareLinkInput
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&in); err != nil {
			c.Error(bindError(err))
			return
		}
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	view, err := h.shares.Create(c.Request.Context(), id, canvasID, in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// List GET /api/canvas/:id/share-links
func (h *ShareHandler) List(c *gin.Context) {
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	links, err := h.shares.List(c.Request.Context(), id, canvasID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"items": links, "total": len(links)})
}

// Revoke DELETE /api/canvas/:id/share-links/:linkId
func (h *ShareHandler) Revoke(c *gin.Context) {
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}
	linkID, err := parseUintParam(c, "linkId", "分享链接 ID 不合法")
	if err != nil {
		c.Error(err)
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	if err := h.shares.Revoke(c.Request.Context(), id, canvasID, linkID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"id": linkID, "enabled": false})
}

// Resolve GET /api/share/:token —— 公开接口，校验 token 并返回只读画布。
func (h *ShareHandler) Resolve(c *gin.Context) {
	view, err := h.shares.Resolve(c.Request.Context(), c.Param("token"))
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// Fork POST /api/share/:token/fork —— 需要 fork 权限；游客也可以分叉到自己的临时身份。
func (h *ShareHandler) Fork(c *gin.Context) {
	var in service.ForkCanvasInput
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&in); err != nil {
			c.Error(bindError(err))
			return
		}
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	view, err := h.shares.Fork(c.Request.Context(), id, c.Param("token"), in.Title)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}
