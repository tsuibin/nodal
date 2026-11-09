//go:build !nodal_nosqlite

// 默认构建保留 SQLite 驱动，供本地开发/单机演示使用。
//
package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func sqliteDialector(dsn string) (gorm.Dialector, error) {
	// "file:" 形式由 sqlite 驱动自己解析路径，这里不替它建目录；
	// :memory: 也不需要落盘。
	if !strings.HasPrefix(dsn, "file:") && dsn != ":memory:" {
		dir := filepath.Dir(dsn)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("创建 sqlite 目录失败: %w", err)
			}
		}
	}
	return sqlite.Open(dsn), nil
}
