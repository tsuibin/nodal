// Package embed 提供文本向量化（Embedding）适配层。
//
// 与 llm 的 Chat 适配器分开：Embedding 走的是另一个端点（/embeddings），
// 且必须一次性批量返回，不能流式。当前支持：
//   - mock/embed                      本地确定性向量（免 Key，开发/测试用）
//   - openai/text-embedding-3-small   OpenAI 兼容 /embeddings
package embed

import (
	"context"
	"fmt"
	"strings"
)

// Credential 是调用向量接口所需的鉴权信息。
type Credential struct {
	Provider string
	APIKey   string
	BaseURL  string
}

// Embedder 是一个向量模型。
type Embedder interface {
	// ID 是模型目录里的 id，例如 mock/embed。
	ID() string
	// Provider 决定用哪个 provider 的 API Key。
	Provider() string
	RequiresKey() bool
	// Embed 返回与 texts 一一对应的向量，每个向量长度必须等于 Registry.Dim()。
	Embed(ctx context.Context, cred Credential, texts []string) ([][]float32, error)
}

// Registry 是向量模型注册表。
type Registry struct {
	dim    int
	models map[string]Embedder
	order  []string
}

func NewRegistry(dim int) *Registry {
	r := &Registry{dim: dim, models: make(map[string]Embedder)}
	r.register(&mockEmbedder{dim: dim})
	r.register(newOpenAIEmbedder(dim))
	return r
}

func (r *Registry) register(e Embedder) {
	r.models[e.ID()] = e
	r.order = append(r.order, e.ID())
}

func (r *Registry) Dim() int { return r.dim }

func (r *Registry) Resolve(id string) (Embedder, error) {
	id = strings.TrimSpace(id)
	e, ok := r.models[id]
	if !ok {
		return nil, fmt.Errorf("不支持的向量模型 %q（可选 %s）", id, strings.Join(r.order, " / "))
	}
	return e, nil
}

func (r *Registry) IDs() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}
