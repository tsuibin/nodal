package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// sseWriter 封装 text/event-stream 的写帧与 flush。
// 所有写操作加锁，允许心跳 goroutine 与主流程并发写同一连接而不撕裂帧。
type sseWriter struct {
	c  *gin.Context
	mu sync.Mutex
}

func newSSEWriter(c *gin.Context) *sseWriter {
	return &sseWriter{c: c}
}

// Prepare 写 SSE 响应头并立即 flush，之后只能发事件、不能再返回 JSON。
func (w *sseWriter) Prepare() error {
	header := w.c.Writer.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-cache, no-transform")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no") // 关闭 nginx 等反代的缓冲

	w.c.Writer.WriteHeader(http.StatusOK)
	if _, err := w.c.Writer.Write([]byte(": connected\n\n")); err != nil {
		return err
	}
	w.flush()
	return nil
}

// Emit 发送一个具名事件：event: node\ndata: {...}\n\n
func (w *sseWriter) Emit(event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sse: marshal payload: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.c.Writer.Write([]byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data))); err != nil {
		return err
	}
	w.flush()
	return nil
}

// Comment 发送注释帧（心跳），前端会忽略，但能保活连接。
func (w *sseWriter) Comment(text string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.c.Writer.Write([]byte(": " + text + "\n\n")); err != nil {
		return err
	}
	w.flush()
	return nil
}

// StartHeartbeat 每 interval 发一次心跳；返回的函数用于停止并等待 goroutine 退出。
func (w *sseWriter) StartHeartbeat(interval time.Duration) func() {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_ = w.Comment("ping")
			}
		}
	}()

	return func() {
		close(done)
		wg.Wait()
	}
}

func (w *sseWriter) flush() {
	if f, ok := w.c.Writer.(http.Flusher); ok {
		f.Flush()
	}
}
