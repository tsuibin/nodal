package migrate

import (
	"strings"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	script := `
-- 这是注释
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE demo (
  id uuid PRIMARY KEY
);
`
	stmts := splitStatements(script)
	if len(stmts) != 2 {
		t.Fatalf("期望 2 条语句，实际 %d: %#v", len(stmts), stmts)
	}
	if !strings.HasPrefix(stmts[0], "CREATE EXTENSION") {
		t.Fatalf("第一条语句不符: %s", stmts[0])
	}
	if !strings.Contains(stmts[1], "CREATE TABLE demo") {
		t.Fatalf("第二条语句不符: %s", stmts[1])
	}
}

func TestListSQLFilesOrdered(t *testing.T) {
	files, err := listSQLFiles()
	if err != nil {
		t.Fatalf("读取迁移目录失败: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("migrate 目录下没有任何 up.sql")
	}
	if files[0].Version != "000001" || files[0].Name != "init" {
		t.Fatalf("首个迁移不符: %+v", files[0])
	}
	for i := 1; i < len(files); i++ {
		if files[i-1].Version >= files[i].Version {
			t.Fatalf("版本未升序: %s >= %s", files[i-1].Version, files[i].Version)
		}
	}
}

func TestModelsCoverAllTables(t *testing.T) {
	want := []string{
		"users", "canvases", "agent_templates", "canvas_nodes", "canvas_edges",
		"knowledge_files", "knowledge_chunks", "canvas_likes", "canvas_comments",
	}
	got := tablenames(t, Models())
	if len(got) != len(want) {
		t.Fatalf("模型数量不符: 期望 %d 实际 %d", len(want), len(got))
	}
	for _, w := range want {
		if !got[w] {
			t.Fatalf("缺少表模型: %s", w)
		}
	}
}
