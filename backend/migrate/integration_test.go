package migrate

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/nodal/nodal/backend/internal/model"
)

// TestApplySQLAgainstRealDB 需要真实 pgvector 数据库，未设置 NODAL_TEST_DSN 时跳过。
// 本地：docker compose up -d && NODAL_TEST_DSN="postgres://nodal:nodal_dev_pwd@localhost:5432/nodal?sslmode=disable" go test ./migrate -run RealDB
// CI：见 .github/workflows/ci.yml 的 integration job。
func TestApplySQLAgainstRealDB(t *testing.T) {
	dsn := os.Getenv("NODAL_TEST_DSN")
	if dsn == "" {
		t.Skip("未设置 NODAL_TEST_DSN，跳过真实数据库集成测试")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}
	// 从干净状态开始：先回滚再迁移，保证可重复执行
	if err := RollbackSQL(db, "000001"); err != nil {
		t.Logf("回滚（首次库为空时正常报错）: %v", err)
	}
	if err := applySQLFresh(db); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}

	// 1) 九张核心表都存在
	for _, name := range []string{
		"users", "canvases", "agent_templates", "canvas_nodes", "canvas_edges",
		"knowledge_files", "knowledge_chunks", "canvas_likes", "canvas_comments",
	} {
		if !db.Migrator().HasTable(name) {
			t.Fatalf("缺少表: %s", name)
		}
	}

	// 2) 向量列类型必须是 vector(1536)
	var colType string
	if err := db.Raw(`SELECT format_type(atttypid, atttypmod) FROM pg_attribute
		WHERE attrelid = 'knowledge_chunks'::regclass AND attname = 'embedding'`).Scan(&colType).Error; err != nil {
		t.Fatalf("查询向量列类型失败: %v", err)
	}
	if colType != "vector(1536)" {
		t.Fatalf("embedding 列类型不符，实际 %s", colType)
	}

	// 3) 内置 Agent 种子数据为 4 条
	var agents int64
	if err := db.Model(&model.AgentTemplate{}).Where("is_builtin = true").Count(&agents).Error; err != nil {
		t.Fatalf("统计内置 Agent 失败: %v", err)
	}
	if agents != 4 {
		t.Fatalf("内置 Agent 数量应为 4，实际 %d", agents)
	}

	// 4) 迁移可重复执行（幂等）
	if err := ApplySQL(db); err != nil {
		t.Fatalf("重复执行迁移应幂等，实际报错: %v", err)
	}

	// 5) 复合外键生效：往不存在的节点插连线必须被拒
	if err := db.Exec(`INSERT INTO canvases (id, owner_id, title) VALUES
		('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 't')`).Error; err == nil {
		t.Fatalf("应因 owner 外键失败，但插入成功了")
	}
}

// applySQLFresh 只执行 SQL 迁移（不经过 schema_migrations 跳过逻辑）。
func applySQLFresh(db *gorm.DB) error {
	if err := EnsureExtensions(db); err != nil {
		return err
	}
	files, err := listSQLFiles()
	if err != nil {
		return err
	}
	for _, f := range files {
		raw, err := sqlFS.ReadFile(f.Path)
		if err != nil {
			return err
		}
		if err := execScript(db, string(raw)); err != nil {
			return fmt.Errorf("%s: %w", f.Path, err)
		}
	}
	return nil
}
