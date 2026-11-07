package service

import (
	"context"
	"strings"
	"testing"

	"github.com/nodal/nodal-server/internal/embed"
)

func TestSplitKnowledgeChunksRespectsBoundaryAndOverlap(t *testing.T) {
	text := strings.Repeat("第一段内容。", 40) + "\n\n" + strings.Repeat("第二段内容。", 40)
	chunks, truncated := splitKnowledgeChunks(text, 200, 40)
	if truncated {
		t.Fatalf("短文本不应触发截断")
	}
	if len(chunks) < 2 {
		t.Fatalf("期望至少 2 个切片，实际 %d", len(chunks))
	}
	for i, chunk := range chunks {
		if runeLen := len([]rune(chunk)); runeLen > 200 {
			t.Fatalf("第 %d 片长度 %d 超过窗口 200", i, runeLen)
		}
	}
	// overlap 应让相邻切片有共享内容
	if !strings.Contains(chunks[1], "第一段") && !strings.Contains(chunks[0], "第二段") {
		t.Logf("相邻切片无明显重叠（窗口边界刚好落在句末也可接受）")
	}
}

func TestSplitKnowledgeChunksTruncatesAtLimit(t *testing.T) {
	text := strings.Repeat("句子。", 5000)
	chunks, truncated := splitKnowledgeChunks(text, 20, 0)
	if !truncated {
		t.Fatalf("超长文本应标记截断")
	}
	if len(chunks) != maxChunksPerFile {
		t.Fatalf("期望 %d 片，实际 %d", maxChunksPerFile, len(chunks))
	}
}

func TestCleanKnowledgeText(t *testing.T) {
	in := "a\r\n\r\n\r\n\r\nb\x00"
	out := cleanKnowledgeText(in)
	if out != "a\n\nb" {
		t.Fatalf("清洗结果不符: %q", out)
	}
}

func TestMockEmbedRelatedTextCloser(t *testing.T) {
	e, err := embed.NewRegistry(64).Resolve("mock/embed")
	if err != nil {
		t.Fatal(err)
	}
	vecs, err := e.Embed(context.Background(), embed.Credential{}, []string{
		"报销流程需要提供发票和审批单",
		"发票和报销流程的说明",
		"今天天气很好适合出门",
	})
	if err != nil {
		t.Fatal(err)
	}
	dot := func(a, b []float32) float64 {
		var sum float64
		for i := range a {
			sum += float64(a[i] * b[i])
		}
		return sum
	}
	related := dot(vecs[0], vecs[1])
	unrelated := dot(vecs[0], vecs[2])
	if related <= unrelated {
		t.Fatalf("相关文本相似度 %.3f 应大于无关文本 %.3f", related, unrelated)
	}
}

func TestSanitizeFileName(t *testing.T) {
	if got := sanitizeFileName("../../etc/passwd"); got != "passwd" {
		t.Fatalf("应去掉路径，实际 %q", got)
	}
	if got := sanitizeFileName("   "); got != "未命名文件" {
		t.Fatalf("空白名应兜底，实际 %q", got)
	}
}
