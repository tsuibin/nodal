package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/apperr"
	"github.com/nodal/nodal-server/middleware"
	"github.com/nodal/nodal-server/response"
	"github.com/nodal/nodal-server/service"
)

// CanvasHandler 是画布 CRUD 的 HTTP 适配层。
type CanvasHandler struct {
	canvases *service.CanvasService
}

func NewCanvasHandler(canvases *service.CanvasService) *CanvasHandler {
	return &CanvasHandler{canvases: canvases}
}

// Create POST /api/canvas
func (h *CanvasHandler) Create(c *gin.Context) {
	limitCanvasBody(c)

	var in service.CreateCanvasInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	view, err := h.canvases.Create(c.Request.Context(), id, in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// List GET /api/canvas/list?scope=mine|public&keyword=&tag=&page=1&page_size=20
func (h *CanvasHandler) List(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	list, err := h.canvases.List(c.Request.Context(), id, service.ListCanvasQuery{
		Scope:    c.Query("scope"),
		Keyword:  c.Query("keyword"),
		Tag:      c.Query("tag"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, list)
}

// Get GET /api/canvas/:id
func (h *CanvasHandler) Get(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}

	view, err := h.canvases.Get(c.Request.Context(), id, canvasID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// Update PUT /api/canvas/:id —— nodes/edges 为整图替换，需成对提交。
func (h *CanvasHandler) Update(c *gin.Context) {
	limitCanvasBody(c)

	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}

	var in service.UpdateCanvasInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	view, err := h.canvases.Update(c.Request.Context(), id, canvasID, in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// Delete DELETE /api/canvas/:id
func (h *CanvasHandler) Delete(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}

	if err := h.canvases.Delete(c.Request.Context(), id, canvasID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"id": canvasID, "deleted": true})
}

// Fork POST /api/canvas/:id/fork
func (h *CanvasHandler) Fork(c *gin.Context) {
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}

	var in service.ForkCanvasInput
	// body 可选：不传标题时自动用「原名 副本」
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

	view, err := h.canvases.Fork(c.Request.Context(), id, canvasID, in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

func parseCanvasID(c *gin.Context) (uint, error) {
	return parseUintParam(c, "id", "画布 ID 不合法")
}

func parseUintParam(c *gin.Context, name, message string) (uint, error) {
	raw := c.Param(name)
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || v == 0 {
		return 0, apperr.New(apperr.CodeInvalidParam, message)
	}
	return uint(v), nil
}

// limitCanvasBody 在解析前限制请求体大小，避免超大 JSON 先被读进内存。
func limitCanvasBody(c *gin.Context) {
	const maxBody = service.MaxCanvasBytes + (1 << 20) // 留 1MB 给标题/标签等元数据
	if c.Request.Body != nil {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
	}
}
