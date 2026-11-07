package handler

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/middleware"
	"github.com/nodal/nodal-server/internal/service"
)

// InferHandler 处理 POST /api/canvas/:id/run-infer（SSE 流式推演）。
type InferHandler struct {
	infer   *service.InferService
	timeout time.Duration
}

func NewInferHandler(infer *service.InferService, timeout time.Duration) *InferHandler {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &InferHandler{infer: infer, timeout: timeout}
}

// Run 推演接口。
//
// 响应是 text/event-stream，事件序列：
//
//	start   {run_id, canvas, agent, model}
//	node    {node, index}      ← 前端 setNodes(prev => [...prev, node])
//	edge    {edge, index}      ← 前端 setEdges(prev => [...prev, edge])
//	warning {message, skipped}  ← 被忽略的非新增指令/非法输出
//	done    {node_count, edge_count, skipped}
//	error   {code, message}
//
// 注意：本接口不写库，只产出增量；SSE 中断不会影响已有画布。
func (h *InferHandler) Run(c *gin.Context) {
	canvasID, err := parseCanvasID(c)
	if err != nil {
		c.Error(err)
		return
	}

	var in service.RunInferInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	// Prepare 完成鉴权、Agent/模型/凭证校验；失败时还能返回正常 JSON 错误。
	plan, err := h.infer.Prepare(c.Request.Context(), id, canvasID, in)
	if err != nil {
		c.Error(err)
		return
	}
	defer plan.Release()

	ctx := c.Request.Context()
	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}

	w := newSSEWriter(c)
	if err := w.Prepare(); err != nil {
		c.Error(err)
		return
	}
	stopHeartbeat := w.StartHeartbeat(15 * time.Second)
	defer stopHeartbeat()

	result, err := h.infer.Stream(ctx, plan, w)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			// 客户端断开，没必要也不能再写
			return
		case errors.Is(err, context.DeadlineExceeded):
			_ = w.Emit("error", gin.H{"code": int(apperr.CodeModelCallFailed), "message": "推演超时，请稍后重试"})
		default:
			appErr := apperr.From(err)
			_ = w.Emit("error", gin.H{"code": int(appErr.Code), "message": appErr.Message})
		}
		return
	}

	_ = w.Emit("done", result)
}
