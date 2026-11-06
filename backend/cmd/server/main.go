// Nodal 后端入口。
// 模块 0：加载配置 → （可选）连接数据库 → 启动 HTTP 服务 → 优雅退出。
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/nodal/nodal/backend/internal/config"
	"github.com/nodal/nodal/backend/internal/httpapi"
	"github.com/nodal/nodal/backend/internal/store"
)

func main() {
	// 本地开发时尝试读取仓库根目录 .env；容器内通常只给环境变量
	_ = godotenv.Load("../.env", ".env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 数据库：配了 DSN 就连，没配也能起（前端联调、CI 冒烟不需要库）
	var db *gorm.DB
	if cfg.DatabaseURL != "" {
		db, err = store.Open(store.DefaultOptions(cfg.DatabaseURL, !cfg.IsProduction()))
		if err != nil {
			log.Fatalf("数据库初始化失败: %v", err)
		}
		defer func() {
			if cerr := store.Close(db); cerr != nil {
				log.Printf("关闭数据库连接失败: %v", cerr)
			}
		}()
		log.Println("数据库已连接")
	} else {
		log.Println("未配置 DATABASE_URL，跳过数据库连接（仅模块 0 允许）")
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(cfg, db),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Nodal backend 启动于 %s (env=%s)", cfg.HTTPAddr, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP 服务异常退出: %v", err)
		}
	}()

	// 等待中断信号，最多 10s 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("优雅关闭失败: %v", err)
	}
	log.Println("Nodal backend 已退出")
}
