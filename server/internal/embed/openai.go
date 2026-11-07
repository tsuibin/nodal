package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const openAIEmbedModel = "text-embedding-3-small"

type openAIEmbedder struct {
	dim    int
	client *http.Client
}

func newOpenAIEmbedder(dim int) *openAIEmbedder {
	return &openAIEmbedder{
		dim:    dim,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (o *openAIEmbedder) ID() string        { return "openai/text-embedding-3-small" }
func (o *openAIEmbedder) Provider() string  { return "openai" }
func (o *openAIEmbedder) RequiresKey() bool { return true }

func (o *openAIEmbedder) Embed(ctx context.Context, cred Credential, texts []string) ([][]float32, error) {
	if strings.TrimSpace(cred.APIKey) == "" {
		return nil, fmt.Errorf("openai embeddings: 缺少 API Key")
	}
	base := strings.TrimRight(strings.TrimSpace(cred.BaseURL), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}

	payload := map[string]any{
		"model":           openAIEmbedModel,
		"input":           texts,
		"encoding_format": "float",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cred.APIKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("openai embeddings: status %d: %s", resp.StatusCode, truncate(string(snippet), 400))
	}

	var result struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openai embeddings: decode: %w", err)
	}
	if len(result.Data) != len(texts) {
		return nil, fmt.Errorf("openai embeddings: 返回 %d 条向量，期望 %d", len(result.Data), len(texts))
	}

	out := make([][]float32, len(texts))
	for _, item := range result.Data {
		if item.Index < 0 || item.Index >= len(texts) {
			return nil, fmt.Errorf("openai embeddings: 非法 index %d", item.Index)
		}
		if len(item.Embedding) != o.dim {
			return nil, fmt.Errorf("openai embeddings: 向量维度 %d 与配置 %d 不一致", len(item.Embedding), o.dim)
		}
		v := make([]float32, o.dim)
		for i, f := range item.Embedding {
			v[i] = float32(f)
		}
		out[item.Index] = v
	}
	return out, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
