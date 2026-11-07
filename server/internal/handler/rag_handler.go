package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/internal/middleware"
	"github.com/nodal/nodal-server/internal/response"
	"github.com/nodal/nodal-server/internal/service"
)

// RAGHandler 处理向量检索。
type RAGHandler struct {
	rag *service.RAGService
}

func NewRAGHandler(rag *service.RAGService) *RAGHandler {
	return &RAGHandler{rag: rag}
}

type retrieveInput struct {
	CanvasID uint   `json:"canvas_id" binding:"required"`
	Query    string `json:"query" binding:"required,max=2000"`
	TopK     int    `json:"top_k" binding:"omitempty,min=1,max=20"`
}

// Retrieve POST /api/rag/retrieve
//
// 只检索当前画布「已挂载」的知识库文件；未挂载返回 4004。
func (h *RAGHandler) Retrieve(c *gin.Context) {
	var in retrieveInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	result, err := h.rag.Retrieve(c.Request.Context(), id, in.CanvasID, in.Query, in.TopK)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}
