// Package llm 是模型适配器层：对上暴露统一的 Stream 接口，对下适配不同厂商，
// 并由 Router 按 model_id 路由到具体 client。
package llm

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/nodal/nodal-server/internal/apperr"
)

// Message 是对话消息。
type Message struct {
	Role    string `json:"role"` // system / user / assistant
	Content string `json:"content"`
}

// Request 是统一的模型调用请求。
type Request struct {
	Model       string
	Messages    []Message
	Temperature float64
	MaxTokens   int
}

// Credential 是调用某个 provider 所需的最小凭证；BaseURL 为空时用 Spec 默认值。
type Credential struct {
	Provider string
	APIKey   string
	BaseURL  string
}

// Client 是一个 provider 适配器。Stream 逐段回调增量文本，回调返回错误应立即中止。
//
// 之所以用回调而不是 channel：调用方（SSE handler）在同一 goroutine 内边收边写，
// 天然背压，也不会因客户端断开而泄漏 goroutine。
type Client interface {
	Name() string
	Stream(ctx context.Context, cred Credential, req Request, onDelta func(string) error) error
}

// Spec 描述一个可选模型。model_id 形如 "openai/gpt-4o-mini"。
type Spec struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Provider       string `json:"provider"`
	UpstreamModel  string `json:"upstream_model"`
	DefaultBaseURL string `json:"-"`
	MaxTokens      int    `json:"max_tokens"`
	RequiresKey    bool   `json:"requires_key"`
}

// Router 保存模型目录与 provider 适配器，负责按 model_id 路由。
type Router struct {
	specs   map[string]Spec
	order   []string
	clients map[string]Client
	timeout time.Duration
}

func NewRouter(timeout time.Duration) *Router {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	r := &Router{
		specs:   make(map[string]Spec),
		clients: make(map[string]Client),
		timeout: timeout,
	}

	httpClient := &http.Client{Timeout: timeout}
	r.registerClient("openai", &openAIClient{http: httpClient, name: "openai"})
	r.registerClient("deepseek", &openAIClient{http: httpClient, name: "deepseek"})
	r.registerClient("moonshot", &openAIClient{http: httpClient, name: "moonshot"})
	r.registerClient("zhipu", &openAIClient{http: httpClient, name: "zhipu"})
	r.registerClient("qwen", &openAIClient{http: httpClient, name: "qwen"})
	r.registerClient("anthropic", &anthropicClient{http: httpClient})
	r.registerClient("mock", &mockClient{})

	for _, spec := range builtinCatalog() {
		r.specs[spec.ID] = spec
		r.order = append(r.order, spec.ID)
	}
	sort.SliceStable(r.order, func(i, j int) bool {
		return r.specs[r.order[i]].Label < r.specs[r.order[j]].Label
	})
	return r
}

func (r *Router) registerClient(provider string, c Client) {
	r.clients[provider] = c
}

// List 返回全部可选模型，供前端下拉框和凭证配置页使用。
func (r *Router) List() []Spec {
	out := make([]Spec, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.specs[id])
	}
	return out
}

// Providers 返回支持保存 API Key 的 provider 列表（mock 不需要 key）。
func (r *Router) Providers() []string {
	var out []string
	for _, spec := range r.specs {
		if !spec.RequiresKey {
			continue
		}
		out = append(out, spec.Provider)
	}
	sort.Strings(out)
	return dedupe(out)
}

// SupportsProvider 判断 provider 是否存在。
func (r *Router) SupportsProvider(provider string) bool {
	_, ok := r.clients[strings.ToLower(provider)]
	return ok
}

// RequiresKey 判断 provider 是否必须配置 API Key。
func (r *Router) RequiresKey(provider string) bool {
	for _, spec := range r.specs {
		if spec.Provider == provider {
			return spec.RequiresKey
		}
	}
	return false
}

// Resolve 按 model_id 找到模型定义与对应 client。
func (r *Router) Resolve(modelID string) (Spec, Client, error) {
	id := strings.TrimSpace(modelID)
	spec, ok := r.specs[id]
	if !ok {
		return Spec{}, nil, apperr.Newf(apperr.CodeModelNotSupport, "不支持的模型 %q", modelID)
	}
	client, ok := r.clients[spec.Provider]
	if !ok {
		return Spec{}, nil, apperr.Newf(apperr.CodeInternal, "模型适配器缺失: %s", spec.Provider)
	}
	return spec, client, nil
}

// DefaultBaseURL 返回某个 provider 的默认接入地址。
func (r *Router) DefaultBaseURL(provider string) string {
	for _, spec := range r.specs {
		if spec.Provider == provider && spec.DefaultBaseURL != "" {
			return spec.DefaultBaseURL
		}
	}
	return ""
}

// Stream 是统一调用入口：补全 BaseURL 后转发给具体适配器。
func (r *Router) Stream(ctx context.Context, spec Spec, client Client, cred Credential, req Request, onDelta func(string) error) error {
	if cred.BaseURL == "" {
		cred.BaseURL = spec.DefaultBaseURL
	}
	cred.Provider = spec.Provider
	if req.MaxTokens <= 0 {
		req.MaxTokens = spec.MaxTokens
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 4096
	}
	if req.Model == "" {
		req.Model = spec.UpstreamModel
	}

	return client.Stream(ctx, cred, req, onDelta)
}

func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
