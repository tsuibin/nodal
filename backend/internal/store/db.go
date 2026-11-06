// Package store 封装 GORM 数据库连接。
// 模块 0 只做初始化：连接池、日志、健康探测；具体模型与迁移在模块 1 落地。
package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Options 数据库连接参数。
type Options struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	Verbose         bool // 开发环境打印慢 SQL
}

// DefaultOptions 返回适合本地开发的默认值。
func DefaultOptions(dsn string, verbose bool) Options {
	return Options{
		DSN:             dsn,
		MaxOpenConns:    20,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		Verbose:         verbose,
	}
}

// Open 建立连接并返回 *gorm.DB。
// 调用方需负责在退出时调用 Close。
func Open(opt Options) (*gorm.DB, error) {
	if opt.DSN == "" {
		return nil, fmt.Errorf("数据库 DSN 为空")
	}

	logLevel := logger.Warn
	if opt.Verbose {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(opt.DSN), &gorm.Config{
		Logger: logger.New(
			log.New(log.Writer(), "[gorm] ", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logLevel,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		// 时间统一按 UTC 存取，展示层再按用户时区渲染
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("连接 PostgreSQL 失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 *sql.DB 失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(opt.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opt.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(opt.ConnMaxLifetime)

	// 启动即探活，避免带着坏连接跑起来
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("数据库 Ping 失败: %w", err)
	}

	return db, nil
}

// Close 关闭底层连接池。
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
