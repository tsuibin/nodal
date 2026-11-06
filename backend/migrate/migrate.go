// Package migrate 提供两种建库路径：
//
//  1. SQL 文件模式（线上 / 可审计）：按序执行 migrate/*.up.sql，并记录到 schema_migrations。
//  2. AutoMigrate 模式（本地 / 快速迭代）：由 GORM 依据 model 建表，再补齐 SQL 无法声明的部分
//     （复合外键、部分唯一索引、HNSW 向量索引）。
//
// 两条路径最终必须收敛到同一份结构；schema_test 用 SQL 文件结构做基准。
package migrate

import (
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/nodal/nodal/backend/internal/model"
)

//go:embed *.sql
var sqlFS embed.FS

// Models 返回全部需要建表的模型，顺序即 AutoMigrate 顺序（被引用方在前）。
func Models() []any {
	return []any{
		&model.User{},
		&model.Canvas{},
		&model.AgentTemplate{},
		&model.CanvasNode{},
		&model.CanvasEdge{},
		&model.KnowledgeFile{},
		&model.KnowledgeChunk{},
		&model.CanvasLike{},
		&model.CanvasComment{},
	}
}

// EnsureExtensions 开启 pgvector 与 pgcrypto。
func EnsureExtensions(db *gorm.DB) error {
	for _, stmt := range []string{
		"CREATE EXTENSION IF NOT EXISTS pgcrypto",
		"CREATE EXTENSION IF NOT EXISTS vector",
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("开启扩展失败 (%s): %w", stmt, err)
		}
	}
	return nil
}

// postAutoSQL 是 AutoMigrate 无法表达、但 SQL 迁移里有等价声明的结构。
// 全部写成可重复执行的语句（IF NOT EXISTS / DO $$ 捕获重复）。
var postAutoSQL = []string{
	// 知识库切片：GORM 会按 type 标签建出 vector(1536)，这里兜底补列
	`ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS embedding vector(1536)`,
	// 连线复合外键：保证两端节点属于同一画布
	`DO $$ BEGIN
	   ALTER TABLE canvas_edges ADD CONSTRAINT fk_edges_source
	     FOREIGN KEY (canvas_id, source_node_id) REFERENCES canvas_nodes (canvas_id, node_id) ON DELETE CASCADE;
	 EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	`DO $$ BEGIN
	   ALTER TABLE canvas_edges ADD CONSTRAINT fk_edges_target
	     FOREIGN KEY (canvas_id, target_node_id) REFERENCES canvas_nodes (canvas_id, node_id) ON DELETE CASCADE;
	 EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	// 内置 Agent 模板按名字唯一，保证种子数据可重复执行
	`CREATE UNIQUE INDEX IF NOT EXISTS uk_agents_builtin_name ON agent_templates (name) WHERE is_builtin = true`,
	// 广场列表部分索引
	`CREATE INDEX IF NOT EXISTS idx_canvases_public ON canvases (is_public, updated_at DESC) WHERE is_public = true`,
	// HNSW 向量索引
	`CREATE INDEX IF NOT EXISTS idx_chunks_embedding ON knowledge_chunks USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64)`,
}

// AutoMigrate 走 GORM 建表路径（本地开发 / 测试库）。
func AutoMigrate(db *gorm.DB) error {
	if err := EnsureExtensions(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(Models()...); err != nil {
		return fmt.Errorf("AutoMigrate 失败: %w", err)
	}
	for _, stmt := range postAutoSQL {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("后置结构补齐失败: %w", err)
		}
	}
	return SeedBuiltinAgents(db)
}

// SQLFile 描述一个迁移文件。
type SQLFile struct {
	Version string // 000001
	Name    string // init
	Path    string // 000001_init.up.sql
}

// listSQLFiles 列出全部 up 迁移，按版本号升序。
func listSQLFiles() ([]SQLFile, error) {
	entries, err := sqlFS.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var files []SQLFile
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		parts := strings.SplitN(strings.TrimSuffix(name, ".up.sql"), "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("迁移文件名不合规（应为 <version>_<name>.up.sql）: %s", name)
		}
		files = append(files, SQLFile{Version: parts[0], Name: parts[1], Path: name})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Version < files[j].Version })
	return files, nil
}

// ApplySQL 按序执行 up 迁移，已执行过的版本自动跳过。
func ApplySQL(db *gorm.DB) error {
	if err := EnsureExtensions(db); err != nil {
		return err
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    varchar(64) PRIMARY KEY,
		name       varchar(128) NOT NULL DEFAULT '',
		applied_at timestamptz NOT NULL DEFAULT now()
	)`).Error; err != nil {
		return fmt.Errorf("创建 schema_migrations 失败: %w", err)
	}

	files, err := listSQLFiles()
	if err != nil {
		return err
	}

	for _, f := range files {
		var count int64
		if err := db.Raw("SELECT count(1) FROM schema_migrations WHERE version = ?", f.Version).
			Scan(&count).Error; err != nil {
			return fmt.Errorf("查询迁移状态失败: %w", err)
		}
		if count > 0 {
			log.Printf("[migrate] 跳过已执行版本 %s_%s", f.Version, f.Name)
			continue
		}

		raw, err := sqlFS.ReadFile(f.Path)
		if err != nil {
			return err
		}

		log.Printf("[migrate] 执行 %s", f.Path)
		if err := execScript(db, string(raw)); err != nil {
			return fmt.Errorf("%s 执行失败: %w", f.Path, err)
		}
		if err := db.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", f.Version, f.Name).
			Error; err != nil {
			return fmt.Errorf("记录迁移版本失败: %w", err)
		}
	}
	return nil
}

// RollbackSQL 按版本倒序执行 down 迁移（仅回滚指定版本，默认最新一版）。
func RollbackSQL(db *gorm.DB, version string) error {
	files, err := listSQLFiles()
	if err != nil {
		return err
	}
	if version == "" && len(files) > 0 {
		version = files[len(files)-1].Version
	}

	for _, f := range files {
		if f.Version != version {
			continue
		}
		raw, err := sqlFS.ReadFile(f.Version + "_" + f.Name + ".down.sql")
		if err != nil {
			return fmt.Errorf("读取 down 迁移失败: %w", err)
		}
		if err := execScript(db, string(raw)); err != nil {
			return fmt.Errorf("回滚 %s 失败: %w", f.Version, err)
		}
		if err := db.Exec("DELETE FROM schema_migrations WHERE version = ?", f.Version).Error; err != nil {
			return err
		}
		log.Printf("[migrate] 已回滚版本 %s_%s", f.Version, f.Name)
		return nil
	}
	return fmt.Errorf("未找到版本 %s 的迁移", version)
}

// execScript 逐条执行 SQL 脚本。
// pgx 的扩展协议不支持一次提交多条语句，因此按分号切分；注释行整体丢弃。
func execScript(db *gorm.DB, script string) error {
	for _, stmt := range splitStatements(script) {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("语句执行失败: %w\nSQL: %s", err, truncate(stmt, 200))
		}
	}
	return nil
}

// splitStatements 按分号切分 SQL，跳过空语句与纯注释块。
func splitStatements(script string) []string {
	var out []string
	var buf strings.Builder

	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}

	for _, chunk := range strings.Split(buf.String(), ";") {
		stmt := strings.TrimSpace(chunk)
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
