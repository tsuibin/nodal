// Package router 组装 HTTP 路由与中间件。
package router

import (
	"log/slog"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/config"
	"github.com/nodal/nodal-server/internal/handler"
	"github.com/nodal/nodal-server/internal/middleware"
	"github.com/nodal/nodal-server/internal/response"
)

// Deps 是路由层依赖的组件。
type Deps struct {
	Config        *config.Config
	Auth          *handler.AuthHandler
	Canvas        *handler.CanvasHandler
	Agent         *handler.AgentHandler
	Model         *handler.ModelHandler
	Infer         *handler.InferHandler
	Knowledge     *handler.KnowledgeHandler
	RAG           *handler.RAGHandler
	Square        *handler.SquareHandler
	Share         *handler.ShareHandler
	Repl          *handler.ReplHandler
	Authenticator *middleware.Authenticator
}

// New 构建 gin.Engine。
//
// 路由分三组：
//
//	公开     /healthz、/api/auth/register、/api/auth/login、/api/auth/guest
//	全局鉴权 /api/** —— 用户与游客都可访问（游客由中间件即时分配临时身份）
//	仅用户   /api/auth/profile 等 —— 在全局鉴权基础上再加 RequireUser
func New(d Deps) *gin.Engine {
	if d.Config.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     d.Config.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID", middleware.GuestTokenHeader},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// 统一错误出口必须在最后注册，保证 c.Next() 之后能收拢所有 c.Error。
	r.Use(middleware.ErrorHandler())

	r.NoRoute(func(c *gin.Context) {
		c.Error(apperr.New(apperr.CodeNotFound, "接口不存在"))
	})
	r.NoMethod(func(c *gin.Context) {
		c.Error(apperr.New(apperr.CodeNotFound, "请求方法不支持"))
	})

	r.GET("/healthz", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	authHandler := d.Auth
	canvasHandler := d.Canvas
	agentHandler := d.Agent
	modelHandler := d.Model
	inferHandler := d.Infer
	knowledgeHandler := d.Knowledge
	ragHandler := d.RAG
	squareHandler := d.Square
	shareHandler := d.Share
	replHandler := d.Repl

	api := r.Group("/api")

	// ---- 公开：不经过身份解析 ----
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/guest", authHandler.Guest)

	// 公共广场与分享链接解析：允许匿名访问；广场带 token 时识别身份（liked 状态）。
	api.GET("/square/list", d.Authenticator.Optional(), squareHandler.List)
	// 公开画布的评论允许匿名读取；私人画布仍由 service 校验可读性。
	api.GET("/canvas/:id/comments", d.Authenticator.Optional(), squareHandler.ListComments)
	api.GET("/share/:token", shareHandler.Resolve)
	// WS 入口自行校验 ?token= / Authorization（浏览器 WS 不能带自定义头）。
	api.GET("/repl/ws/:sessionId", replHandler.Stream)

	// ---- 全局鉴权：任何新增业务接口挂在这里即自动受保护 ----
	authed := api.Group("")
	authed.Use(d.Authenticator.Global())
	{
		authed.GET("/auth/me", authHandler.Me)

		// 仅注册用户
		userOnly := authed.Group("")
		userOnly.Use(d.Authenticator.RequireUser())
		{
			userOnly.PUT("/auth/profile", authHandler.UpdateProfile)

			// Agent 模板 / 模型目录 / API Key：都要求注册用户
			userOnly.POST("/agent", agentHandler.Create)
			userOnly.GET("/agent/list", agentHandler.List)
			userOnly.GET("/agent/:id", agentHandler.Get)
			userOnly.PUT("/agent/:id", agentHandler.Update)
			userOnly.DELETE("/agent/:id", agentHandler.Delete)

			userOnly.GET("/model/catalog", modelHandler.Catalog)
			userOnly.GET("/model/credentials", modelHandler.ListCredentials)
			userOnly.PUT("/model/credentials/:provider", modelHandler.SaveCredential)
			userOnly.DELETE("/model/credentials/:provider", modelHandler.DeleteCredential)

			// SSE 推演（游客不可用：模型调用需要用户凭证）
			userOnly.POST("/canvas/:id/run-infer", inferHandler.Run)

			// 模块 5：私有知识库 + 画布临时挂载 + RAG 检索
			userOnly.POST("/knowledge/upload", knowledgeHandler.Upload)
			userOnly.GET("/knowledge/list", knowledgeHandler.List)
			userOnly.DELETE("/knowledge/:id", knowledgeHandler.Delete)
			userOnly.POST("/canvas/:id/mount-knowledge", knowledgeHandler.Mount)
			userOnly.GET("/canvas/:id/mount-knowledge", knowledgeHandler.CurrentMount)
			userOnly.DELETE("/canvas/:id/mount-knowledge", knowledgeHandler.Unmount)
			userOnly.POST("/rag/retrieve", ragHandler.Retrieve)

			// 模块 7：REPL 沙箱（仅注册用户；WS 入口在公开分组自行解析 token）
			userOnly.POST("/repl/create", replHandler.Create)

			// 模块 6：发布/广场互动/分享链接（广场列表与分享解析在公开分组里）
			userOnly.POST("/canvas/:id/publish", squareHandler.Publish)
			userOnly.DELETE("/canvas/:id/publish", squareHandler.Unpublish)
			userOnly.POST("/canvas/:id/like", squareHandler.Like)
			userOnly.POST("/canvas/:id/comment", squareHandler.CreateComment)
			userOnly.DELETE("/canvas/:id/comments/:commentId", squareHandler.DeleteComment)
			userOnly.POST("/canvas/:id/share-link", shareHandler.Create)
			userOnly.GET("/canvas/:id/share-links", shareHandler.List)
			userOnly.DELETE("/canvas/:id/share-links/:linkId", shareHandler.Revoke)
		}

		// 画布：游客也能创建/读取自己的临时画布；写权限由 service 按 owner_key 校验。
		authed.POST("/canvas", canvasHandler.Create)
		authed.GET("/canvas/list", canvasHandler.List)
		authed.GET("/canvas/:id", canvasHandler.Get)
		authed.PUT("/canvas/:id", canvasHandler.Update)
		authed.DELETE("/canvas/:id", canvasHandler.Delete)
		authed.POST("/canvas/:id/fork", canvasHandler.Fork)
		authed.POST("/share/:token/fork", shareHandler.Fork)
	}

	slog.Info("routes registered",
		"healthz", "/healthz",
		"api", "/api",
		"guest_enabled", d.Config.GuestEnabled,
	)
	return r
}
