package llm

import (
	"context"
	"time"
)

// mockClient 是本地模拟模型：不需要 API Key，按固定 JSONL 输出 3 个节点 + 2 条边，
// 每条之间 sleep 一小段时间，用于开发/演示/端到端测试流式增量。
type mockClient struct{}

func (c *mockClient) Name() string { return "mock" }

func (c *mockClient) Stream(ctx context.Context, _ Credential, _ Request, onDelta func(string) error) error {
	lines := []string{
		`{"type":"node","node":{"id":"mock-n1","type":"default","position":{"x":0,"y":0},"data":{"label":"拆解目标（模拟）"}}}`,
		`{"type":"node","node":{"id":"mock-n2","type":"default","position":{"x":0,"y":0},"data":{"label":"执行步骤（模拟）"}}}`,
		`{"type":"node","node":{"id":"mock-n3","type":"default","position":{"x":0,"y":0},"data":{"label":"验收标准（模拟）"}}}`,
		`{"type":"edge","edge":{"id":"mock-e1","source":"mock-n1","target":"mock-n2","animated":true}}`,
		`{"type":"edge","edge":{"id":"mock-e2","source":"mock-n2","target":"mock-n3"}}`,
	}

	for _, line := range lines {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Millisecond):
		}
		if err := onDelta(line + "\n"); err != nil {
			return err
		}
	}
	return nil
}
