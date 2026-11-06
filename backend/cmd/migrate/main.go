// Nodal 数据库迁移 CLI。
//
// 用法：
//
//	go run ./cmd/migrate --mode=sql        # 按 migrate/*.up.sql 建库（推荐，可审计）
//	go run ./cmd/migrate --mode=auto       # 用 GORM AutoMigrate 建库（本地快速迭代）
//	go run ./cmd/migrate --mode=sql --down # 回滚最新一版
package main

import (
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/nodal/nodal/backend/internal/config"
	"github.com/nodal/nodal/backend/internal/store"
	"github.com/nodal/nodal/backend/migrate"
)

func main() {
	mode := flag.String("mode", "sql", "迁移模式：sql（执行 SQL 文件）或 auto（GORM AutoMigrate）")
	down := flag.Bool("down", false, "回滚最新一版（仅 sql 模式）")
	version := flag.String("version", "", "指定回滚版本号，如 000001")
	flag.Parse()

	_ = godotenv.Load("../.env", ".env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL 为空，无法执行迁移（示例：postgres://nodal:nodal_dev_pwd@localhost:5432/nodal?sslmode=disable）")
	}

	db, err := store.Open(store.DefaultOptions(cfg.DatabaseURL, true))
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer func() { _ = store.Close(db) }()

	switch {
	case *down:
		if err := migrate.RollbackSQL(db, *version); err != nil {
			log.Fatalf("回滚失败: %v", err)
		}
	case *mode == "auto":
		if err := migrate.AutoMigrate(db); err != nil {
			log.Fatalf("AutoMigrate 失败: %v", err)
		}
		log.Println("AutoMigrate 完成，内置 Agent 模板已就绪")
	default:
		if err := migrate.ApplySQL(db); err != nil {
			log.Fatalf("SQL 迁移失败: %v", err)
		}
		log.Println("SQL 迁移完成")
	}

	os.Exit(0)
}
