// 模块 10：Demo 画布种子接口。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/internal/middleware"
	"github.com/nodal/nodal-server/internal/response"
	"github.com/nodal/nodal-server/internal/service"
)

type DemoHandler struct {
	demo *service.DemoService
}

func NewDemoHandler(demo *service.DemoService) *DemoHandler {
	return &DemoHandler{demo: demo}
}

// Seed POST /api/demo/seed
// 幂等：该身份已有画布时返回 created=0，不覆盖用户内容。
func (h *DemoHandler) Seed(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	result, err := h.demo.Seed(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, result)
}
