package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/middleware"
	"github.com/nodal/nodal-server/response"
	"github.com/nodal/nodal-server/service"
	"github.com/nodal/nodal-server/token"
)

// SquareHandler 处理发布/广场列表/点赞/评论。
type SquareHandler struct {
	square *service.SquareService
}

func NewSquareHandler(square *service.SquareService) *SquareHandler {
	return &SquareHandler{square: square}
}

// Publish POST /api/canvas/:id/publish
func (h *SquareHandler) Publish(c *gin.Context) {
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
	view, err := h.square.Publish(c.Request.Context(), id, canvasID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// Unpublish DELETE /api/canvas/:id/publish
func (h *SquareHandler) Unpublish(c *gin.Context) {
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
	view, err := h.square.Unpublish(c.Request.Context(), id, canvasID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// List GET /api/square/list?tag=&keyword=&sort=latest|hot&page=1&page_size=20
// 公开接口：带 token 时识别身份，用于返回「我是否点过赞」。
func (h *SquareHandler) List(c *gin.Context) {
	id := optionalIdentity(c)
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	list, err := h.square.List(c.Request.Context(), id, service.SquareQuery{
		Tag:      c.Query("tag"),
		Keyword:  c.Query("keyword"),
		Sort:     c.Query("sort"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, list)
}

// Like POST /api/canvas/:id/like —— 切换点赞状态
func (h *SquareHandler) Like(c *gin.Context) {
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
	result, err := h.square.Like(c.Request.Context(), id, canvasID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}

type createCommentInput struct {
	Content string `json:"content" binding:"required,max=500"`
}

// CreateComment POST /api/canvas/:id/comment
func (h *SquareHandler) CreateComment(c *gin.Context) {
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}
	var in createCommentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	comment, err := h.square.CreateComment(c.Request.Context(), id, canvasID, in.Content)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, comment)
}

// ListComments GET /api/canvas/:id/comments?page=1&page_size=20
func (h *SquareHandler) ListComments(c *gin.Context) {
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	id := optionalIdentity(c)

	list, err := h.square.ListComments(c.Request.Context(), id, canvasID, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, list)
}

// DeleteComment DELETE /api/canvas/:id/comments/:commentId
func (h *SquareHandler) DeleteComment(c *gin.Context) {
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}
	commentID, err := parseUintParam(c, "commentId", "评论 ID 不合法")
	if err != nil {
		c.Error(err)
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	if err := h.square.DeleteComment(c.Request.Context(), id, canvasID, commentID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"id": commentID, "deleted": true})
}

func optionalIdentity(c *gin.Context) *token.Identity {
	if id, ok := middleware.IdentityFrom(c); ok {
		return id
	}
	return nil
}
