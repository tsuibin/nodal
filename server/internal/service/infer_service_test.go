package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nodal/nodal-server/internal/llm"
	"github.com/nodal/nodal-server/internal/model"
)

type recordedEvent struct {
	Name    string
	Payload map[string]any
}

type fakeEmitter struct {
	events []recordedEvent
}

func (f *fakeEmitter) Emit(name string, payload any) error {
	var m map[string]any
	raw, _ := json.Marshal(payload)
	_ = json.Unmarshal(raw, &m)
	f.events = append(f.events, recordedEvent{Name: name, Payload: m})
	return nil
}

func (f *fakeEmitter) names() []string {
	out := make([]string, 0, len(f.events))
	for _, e := range f.events {
		out = append(out, e.Name)
	}
	return out
}

// 核心约束：AI 只能新增——delete/update 被忽略、撞 id 自动改名、已有节点原样不动。
func TestInferStateOnlyAdds(t *testing.T) {
	emit := &fakeEmitter{}
	st := newInferState(&RunPlan{
		RunID:        "run",
		MaxNodes:     2,
		ExistingIDs:  map[string]struct{}{"n1": {}, "n2": {}},
		ExistingEdge: map[string]struct{}{"e1": {}},
		LayoutX:      500,
		LayoutY:      0,
		Agent:        &model.AgentTemplate{ID: 7},
		Spec:         llm.Spec{ID: "mock/basic"},
	}, emit)

	lines := []string{
		`{"type":"node","node":{"id":"n1","data":{"label":"冒名节点"}}}`,       // 撞已有 id -> n1-2
		`{"type":"delete","node_id":"n1"}`,                                 // 必须忽略
		`{"type":"update","node":{"id":"n1","data":{"label":"改掉"}}}`,       // 必须忽略（不能当新增）
		`{"type":"edge","edge":{"id":"e1","source":"n1-2","target":"n1"}}`, // 撞已有边 id -> e1-2
		`{"type":"edge","edge":{"id":"late-e","source":"late1","target":"n1"}}`,
		`{"type":"node","node":{"id":"late1"}}`, // 补上后，前面挂起的边应被补发
		`{"type":"node","node":{"id":"extra"}}`, // 超过 MaxNodes=4 后忽略
	}
	for _, line := range lines {
		if err := st.handleOp(json.RawMessage(line)); err != nil {
			t.Fatalf("handleOp(%s): %v", line, err)
		}
	}

	t.Logf("nodeCount=%d edgeCount=%d skipped=%d", st.nodeCount, st.edgeCount, st.skipped)
	t.Logf("events: %+v", emit.events)
	if st.nodeCount != 2 {
		t.Fatalf("nodeCount = %d, want 2（第三个节点应被上限截断）", st.nodeCount)
	}
	if st.edgeCount != 2 {
		t.Fatalf("edgeCount = %d, want 2", st.edgeCount)
	}
	if st.skipped != 3 { // delete + update + 超出上限的节点
		t.Fatalf("skipped = %d, want 3", st.skipped)
	}

	// 收集实际下发的节点 id，确认没有 n1 本身，也没有任何“修改”
	ids := map[string]bool{}
	edgeIDs := map[string]bool{}
	for _, e := range emit.events {
		if e.Name == "node" {
			node, _ := e.Payload["node"].(map[string]any)
			ids[node["id"].(string)] = true
		}
		if e.Name == "edge" {
			edge, _ := e.Payload["edge"].(map[string]any)
			edgeIDs[edge["id"].(string)] = true
		}
	}
	if ids["n1"] || !ids["n1-2"] || !ids["late1"] {
		t.Fatalf("节点 id 处理不符合预期: %v", ids)
	}
	t.Logf("events: %+v", emit.events)
	if edgeIDs["e1"] || !edgeIDs["e1-2"] || !edgeIDs["late-e"] {
		t.Fatalf("边 id 处理不符合预期: %v", edgeIDs)
	}
	// 新增节点撞已有 id 时，模型对原 id 的引用必须仍然指向已有节点（不能变成自环）
	for _, e := range emit.events {
		if e.Name != "edge" {
			continue
		}
		edge, _ := e.Payload["edge"].(map[string]any)
		if edge["id"] == "e1-2" {
			if edge["source"] == edge["target"] {
				t.Fatalf("边 e1-2 退化成自环：%v", edge)
			}
		}
	}

	// 同一画布同时只允许一次推演
	svc := &InferService{running: make(map[uint]struct{})}
	if !svc.acquire(9) {
		t.Fatalf("首次 acquire 应成功")
	}
	if svc.acquire(9) {
		t.Fatalf("同一画布不应允许第二次推演")
	}
	svc.release(9)
	if !svc.acquire(9) {
		t.Fatalf("release 后应可再次 acquire")
	}
	for _, n := range emit.names() {
		if n != "node" && n != "edge" && n != "warning" {
			t.Fatalf("出现意外事件 %q", n)
		}
	}
}

func TestJSONLParserTolerant(t *testing.T) {
	var ops []string
	p := &jsonlParser{onOp: func(raw json.RawMessage) error {
		ops = append(ops, strings.TrimSpace(string(raw)))
		return nil
	}}

	stream := "```json\n" +
		"{\"type\":\"node\",\"node\":{\"id\":\"a\"}}\n" +
		"{\"type\":\"node\",\n  \"node\":{\"id\":\"b\"}}\n" + // 跨行 JSON
		"```\n" +
		"[{\"type\":\"node\",\"node\":{\"id\":\"c\"}},{\"type\":\"edge\",\"edge\":{\"id\":\"d\"}}]\n"

	// 故意按不规则切片喂入，模拟 token 流
	for i := 0; i < len(stream); i += 7 {
		end := i + 7
		if end > len(stream) {
			end = len(stream)
		}
		if err := p.Feed(stream[i:end]); err != nil {
			t.Fatalf("feed: %v", err)
		}
	}
	if err := p.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(ops) != 4 {
		t.Fatalf("解析到 %d 条操作，期望 4: %v", len(ops), ops)
	}
}
