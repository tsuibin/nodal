package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// openAIClient 适配 OpenAI 的 /chat/completions 流式协议。
// deepseek / moonshot / zhipu / qwen（compatible-mode）都使用同一协议，
// 因此复用这个 client，只靠 BaseURL + upstream model 区分。
type openAIClient struct {
	http *http.Client
	name string
}

func (c *openAIClient) Name() string { return c.name }

type openAIChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (c *openAIClient) Stream(ctx context.Context, cred Credential, req Request, onDelta func(string) error) error {
	if strings.TrimSpace(cred.APIKey) == "" {
		return fmt.Errorf("%s: 缺少 API Key", c.name)
	}
	base := strings.TrimRight(strings.TrimSpace(cred.BaseURL), "/")
	if base == "" {
		return fmt.Errorf("%s: 缺少 BaseURL", c.name)
	}

	payload := map[string]any{
		"model":       req.Model,
		"messages":    req.Messages,
		"stream":      true,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: marshal request: %w", c.name, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: build request: %w", c.name, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+cred.APIKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%s: request failed: %w", c.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s: upstream status %d: %s", c.name, resp.StatusCode, truncate(string(snippet), 500))
	}

	err = scanSSE(resp.Body, func(event, data string) error {
		data = strings.TrimSpace(data)
		if data == "" {
			return nil
		}
		if data == "[DONE]" {
			return errStreamDone
		}

		var chunk openAIChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 个别厂商会插入非 JSON 心跳，忽略而不是中断整条流
			return nil
		}
		if chunk.Error != nil {
			return fmt.Errorf("%s: %s", c.name, chunk.Error.Message)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			if err := onDelta(choice.Delta.Content); err != nil {
				return err
			}
		}
		return nil
	})

	if errors.Is(err, errStreamDone) {
		return nil
	}
	return err
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
