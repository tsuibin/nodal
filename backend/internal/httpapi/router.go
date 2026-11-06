// Package httpapi 组装 Gin 路由。
// 模块 0 仅暴露健康检查；后续模块在此注册 /api/auth、/api/canvas 等路由组。
package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal/backend/internal/config"
)

// NewRouter 构建引擎与路由表。
func NewRouter(cfg *config.Config) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 健康检查：供前端启动时探测与容器探针使用
	r.GET("/api/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": cfg.Version,
			"env":     cfg.AppEnv,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	api := r.Group("/api")
	{
		// 模块 2 起在此挂载：api.POST("/auth/register", ...) 等
		_ = api
	}

	return r
}
