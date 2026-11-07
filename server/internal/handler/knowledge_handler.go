package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/middleware"
	"github.com/nodal/nodal-server/internal/response"
	"github.com/nodal/nodal-server/internal/service"
)

// KnowledgeHandler 提供私有知识库上传/列表/删除，以及画布临时挂载。
type KnowledgeHandler struct {
	knowledge *service.KnowledgeService
	rag       *service.RAGService
}

func NewKnowledgeHandler(knowledge *service.KnowledgeService, rag *service.RAGService) *KnowledgeHandler {
	return &KnowledgeHandler{knowledge: knowledge, rag: rag}
}

// Upload POST /api/knowledge/upload（multipart/form-data，字段名 file）
func (h *KnowledgeHandler) Upload(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	// 多留 1MB 给 multipart 边界/其它字段；真正的文件大小在 service 里二次校验
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.knowledge.MaxBytes()+(1<<20))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.Error(bindError(err))
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.Error(apperr.Wrap(apperr.CodeInternal, "读取上传文件失败", err))
		return
	}
	defer file.Close()

	view, err := h.knowledge.Upload(
		c.Request.Context(),
		id,
		fileHeader.Filename,
		fileHeader.Size,
		fileHeader.Header.Get("Content-Type"),
		file,
	)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// List GET /api/knowledge/list
func (h *KnowledgeHandler) List(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	list, err := h.knowledge.List(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, list)
}

// Delete DELETE /api/knowledge/:id
func (h *KnowledgeHandler) Delete(c *gin.Context) {
	fileID, err := parseUintParam(c, "id", "文件 ID 不合法")
	if err != nil {
		c.Error(err)
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	if err := h.knowledge.Delete(c.Request.Context(), id, fileID); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"id": fileID, "deleted": true})
}

type mountKnowledgeInput struct {
	FileIDs []uint `json:"file_ids"`
}

// Mount POST /api/canvas/:id/mount-knowledge
// 覆盖式设置：传空数组等于解除挂载；挂载关系只存内存，不落库。
func (h *KnowledgeHandler) Mount(c *gin.Context) {
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}

	var in mountKnowledgeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	view, err := h.rag.Mount(c.Request.Context(), id, canvasID, in.FileIDs)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// CurrentMount GET /api/canvas/:id/mount-knowledge
func (h *KnowledgeHandler) CurrentMount(c *gin.Context) {
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

	view, err := h.rag.CurrentMount(c.Request.Context(), id, canvasID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}

// Unmount DELETE /api/canvas/:id/mount-knowledge
func (h *KnowledgeHandler) Unmount(c *gin.Context) {
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

	view, err := h.rag.Unmount(c.Request.Context(), id, canvasID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, view)
}
