// Package db 负责建立 GORM 连接与迁移。
package db

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/nodal/nodal-server/config"
	"github.com/nodal/nodal-server/model"
	"github.com/nodal/nodal-server/vector"
)

// Open 按配置建立数据库连接。sqlite 会自动创建所在目录。
func Open(cfg *config.Config) (*gorm.DB, error) {
	dialector, err := dialectorOf(cfg)
	if err != nil {
		return nil, err
	}

	logLevel := logger.Warn
	if cfg.DBDebug {
		logLevel = logger.Info
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		// 把唯一键冲突等驱动错误翻译成 gorm.ErrDuplicatedKey，便于服务层精确判断。
		TranslateError: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("获取连接池失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return gdb, nil
}

// Migrate 执行表结构迁移。
func Migrate(gdb *gorm.DB, cfg *config.Config) error {
	if err := gdb.AutoMigrate(
		&model.User{},
		&model.GuestSession{},
		&model.Canvas{},
		&model.AgentTemplate{},
		&model.ModelCredential{},
		&model.KnowledgeFile{},
		&model.KnowledgeChunk{},
		&model.CanvasLike{},
		&model.CanvasComment{},
		&model.CanvasShareLink{},
	); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 生产用 pgvector：建扩展、加 vector(N) 列与 HNSW 索引。
	if cfg.DBDriver == "postgres" {
		if err := vector.EnsurePGVectorSchema(gdb, cfg.EmbedDim); err != nil {
			return fmt.Errorf("pgvector 初始化失败: %w", err)
		}
	}
	slog.Info("database migrated")
	return nil
}

func dialectorOf(cfg *config.Config) (gorm.Dialector, error) {
	switch cfg.DBDriver {
	case "sqlite":
		return sqliteDialector(cfg.DBDSN)
	case "mysql":
		return mysql.Open(cfg.DBDSN), nil
	case "postgres":
		return postgres.Open(cfg.DBDSN), nil
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.DBDriver)
	}
}
