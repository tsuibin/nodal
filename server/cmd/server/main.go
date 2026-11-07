// Nodal 后端入口：装载配置 → 连接数据库 → 迁移 → 装配依赖 → 启动 HTTP 服务。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nodal/nodal-server/internal/config"
	"github.com/nodal/nodal-server/internal/db"
	"github.com/nodal/nodal-server/internal/embed"
	"github.com/nodal/nodal-server/internal/handler"
	"github.com/nodal/nodal-server/internal/llm"
	"github.com/nodal/nodal-server/internal/middleware"
	"github.com/nodal/nodal-server/internal/repl"
	"github.com/nodal/nodal-server/internal/router"
	"github.com/nodal/nodal-server/internal/secretbox"
	"github.com/nodal/nodal-server/internal/service"
	"github.com/nodal/nodal-server/internal/token"
	"github.com/nodal/nodal-server/internal/vector"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}

	gdb, err := db.Open(cfg)
	if err != nil {
		slog.Error("open database failed", "error", err)
		os.Exit(1)
	}
	if err := db.Migrate(gdb, cfg); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}

	tokens, err := token.NewManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.UserTokenTTL, cfg.GuestTokenTTL)
	if err != nil {
		slog.Error("init token manager failed", "error", err)
		os.Exit(1)
	}

	authService := service.NewAuthService(gdb, tokens, cfg.GuestEnabled, cfg.GuestTokenTTL)
	authHandler := handler.NewAuthHandler(authService)
	canvasService := service.NewCanvasService(gdb)
	canvasHandler := handler.NewCanvasHandler(canvasService)

	llmRouter := llm.NewRouter(cfg.LLMTimeout)
	keyBox, err := secretbox.New(cfg.EncKey)
	if err != nil {
		slog.Error("init secretbox failed", "error", err)
		os.Exit(1)
	}
	agentService := service.NewAgentService(gdb, llmRouter)
	credentialService := service.NewCredentialService(gdb, keyBox, llmRouter)

	// 模块 5：知识库 / RAG
	embedders := embed.NewRegistry(cfg.EmbedDim)
	vecStore := vector.New(gdb, cfg.DBDriver, cfg.EmbedDim)
	knowledgeService, err := service.NewKnowledgeService(gdb, vecStore, embedders, credentialService, cfg)
	if err != nil {
		slog.Error("init knowledge service failed", "error", err)
		os.Exit(1)
	}
	ragService := service.NewRAGService(gdb, knowledgeService, vecStore, cfg)

	// 模块 10：Demo 画布（注册时种，资源库空列表时也可调用 /api/demo/seed 兜底）
	demoService := service.NewDemoService(gdb)
	demoHandler := handler.NewDemoHandler(demoService)

	// 模块 6：广场 / 点赞评论 / 分享链接
	squareService := service.NewSquareService(gdb)
	shareService := service.NewShareService(gdb, canvasService)

	// 模块 7：REPL 沙箱。默认 local（用户命名空间 + PTY），生产可切换 firecracker launcher。
	var replRuntime repl.Runtime
	if cfg.ReplEnabled {
		replRuntime, err = repl.NewRuntime(cfg.ReplRuntime, cfg.ReplFirecrackerLauncher)
		if err != nil {
			slog.Error("init repl runtime failed", "error", err)
			os.Exit(1)
		}
	}
	replManager := repl.NewManager(repl.Config{
		Enabled:      cfg.ReplEnabled,
		Runtime:      replRuntime,
		Timeout:      cfg.ReplTimeout,
		MemoryMB:     cfg.ReplMemoryMB,
		MaxSessions:  cfg.ReplMaxSessions,
		MaxPerUser:   cfg.ReplMaxPerUser,
		DiskMB:       cfg.ReplDiskMB,
		MaxCodeBytes: 16 * 1024,
	})
	replHandler := handler.NewReplHandler(replManager, tokens, cfg.CORSOrigins)

	inferService := service.NewInferService(gdb, llmRouter, credentialService, agentService, ragService, cfg.InferMaxNodes)

	agentHandler := handler.NewAgentHandler(agentService)
	modelHandler := handler.NewModelHandler(llmRouter, credentialService)
	inferHandler := handler.NewInferHandler(inferService, cfg.LLMTimeout)
	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeService, ragService)
	ragHandler := handler.NewRAGHandler(ragService)
	squareHandler := handler.NewSquareHandler(squareService)
	shareHandler := handler.NewShareHandler(shareService)

	authenticator := middleware.NewAuthenticator(tokens, cfg.GuestEnabled)

	engine := router.New(router.Deps{
		Config:        cfg,
		Auth:          authHandler,
		Canvas:        canvasHandler,
		Agent:         agentHandler,
		Model:         modelHandler,
		Infer:         inferHandler,
		Knowledge:     knowledgeHandler,
		RAG:           ragHandler,
		Square:        squareHandler,
		Share:         shareHandler,
		Repl:          replHandler,
		Demo:          demoHandler,
		Authenticator: authenticator,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 后台清理过期游客会话（不影响请求链路）。
	stopJanitor := startGuestJanitor(authService, 30*time.Minute)

	go func() {
		slog.Info("nodal server started", "addr", cfg.HTTPAddr, "env", cfg.Env, "db", cfg.DBDriver)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down")

	stopJanitor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	replManager.CloseAll()
	slog.Info("server stopped")
}

func startGuestJanitor(svc *service.AuthService, interval time.Duration) func() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := svc.PurgeExpiredGuests(ctx, 500); err != nil {
					slog.Warn("purge guest sessions failed", "error", err)
				} else if n > 0 {
					slog.Info("purged guest sessions", "count", n)
				}
			}
		}
	}()
	return cancel
}
