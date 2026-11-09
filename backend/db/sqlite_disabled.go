//go:build nodal_nosqlite

// 生产精简构建：不链接 modernc.org/libc 的 SQLite 实现。
// 适用于 LoongArch 等尚无 modernc.org/libc 平台支持的目标。
//
package db

import (
	"fmt"

	"gorm.io/gorm"
)

func sqliteDialector(dsn string) (gorm.Dialector, error) {
	return nil, fmt.Errorf("当前构建未包含 sqlite 驱动")
}
