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

// anthropicClient 适配 Anthropic Messages API 的流式协议。
// 与 OpenAI 的差异：system 是顶层字段、鉴权用 x-api-key、增量事件是 content_block_delta。
type anthropicClient struct {
	http *http.Client
}

func (c *anthropicClient) Name() string { return "anthropic" }

func (c *anthropicClient) Stream(ctx context.Context, cred Credential, req Request, onDelta func(string) error) error {
	if strings.TrimSpace(cred.APIKey) == "" {
		return fmt.Errorf("anthropic: 缺少 API Key")
	}
	base := strings.TrimRight(strings.TrimSpace(cred.BaseURL), "/")
	if base == "" {
		return fmt.Errorf("anthropic: 缺少 BaseURL")
	}

	var system string
	var messages []Message
	for _, m := range req.Messages {
		if m.Role == "system" {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}
		messages = append(messages, m)
	}
	if len(messages) == 0 {
		return fmt.Errorf("anthropic: 至少需要一条 user 消息")
	}

	payload := map[string]any{
		"model":       req.Model,
		"max_tokens":  req.MaxTokens,
		"messages":    messages,
		"stream":      true,
		"temperature": req.Temperature,
	}
	if system != "" {
		payload["system"] = system
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", cred.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("anthropic: upstream status %d: %s", resp.StatusCode, truncate(string(snippet), 500))
	}

	err = scanSSE(resp.Body, func(event, data string) error {
		data = strings.TrimSpace(data)
		if data == "" {
			return nil
		}

		var envelope struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Error *struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &envelope); err != nil {
			return nil
		}

		switch envelope.Type {
		case "content_block_delta":
			if envelope.Delta.Text != "" {
				return onDelta(envelope.Delta.Text)
			}
		case "message_stop":
			return errStreamDone
		case "error":
			msg := "unknown error"
			if envelope.Error != nil && envelope.Error.Message != "" {
				msg = envelope.Error.Message
			}
			return fmt.Errorf("anthropic: %s", msg)
		}
		return nil
	})

	if errors.Is(err, errStreamDone) {
		return nil
	}
	return err
}
