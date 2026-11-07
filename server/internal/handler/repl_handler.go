package handler

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/middleware"
	"github.com/nodal/nodal-server/internal/repl"
	"github.com/nodal/nodal-server/internal/response"
	"github.com/nodal/nodal-server/internal/token"
)

// ReplHandler 对代码节点提供「创建会话 + WebSocket 终端流」。
//
// 浏览器 WebSocket 无法自定义 Authorization 头，所以 WS 入口额外支持
// ?token=<jwt>；访问日志只记录 URL.Path，不会把查询参数写进日志。
type ReplHandler struct {
	manager  *repl.Manager
	tokens   *token.Manager
	upgrader websocket.Upgrader
}

func NewReplHandler(manager *repl.Manager, tokens *token.Manager, origins []string) *ReplHandler {
	h := &ReplHandler{manager: manager, tokens: tokens}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return originAllowed(r, origins)
		},
	}
	return h
}

func originAllowed(r *http.Request, allowed []string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true // 非浏览器客户端 / 同源代理
	}
	if u, err := url.Parse(origin); err == nil && strings.EqualFold(u.Host, r.Host) {
		return true // 同源页面直接连 WS
	}
	for _, item := range allowed {
		if item == "*" || strings.EqualFold(item, origin) {
			return true
		}
	}
	return false
}

type createReplInput struct {
	Language string `json:"language" binding:"required"`
	Code     string `json:"code"`
}

type createReplView struct {
	SessionID      string    `json:"session_id"`
	Language       string    `json:"language"`
	WSPath         string    `json:"ws_path"`
	TimeoutSeconds int       `json:"timeout_seconds"`
	MemoryMB       int       `json:"memory_mb"`
	ExpiresAt      time.Time `json:"expires_at"`
	Runtime        string    `json:"runtime"`
}

// Create 创建隔离会话。返回 ws_path，前端带 ?token= 连接。
func (h *ReplHandler) Create(c *gin.Context) {
	id, ok := middleware.IdentityFrom(c)
	if !ok || !id.IsUser() {
		c.Error(apperr.New(apperr.CodeForbidden, "REPL 仅对注册用户开放"))
		return
	}

	// 代码上限由 service 判定（7006）；这里只挡住异常大的请求体。
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	var in createReplInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}

	sess, err := h.manager.Create(id, repl.Language(in.Language), in.Code)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, createReplView{
		SessionID:      sess.ID,
		Language:       string(sess.Language),
		WSPath:         "/api/repl/ws/" + sess.ID,
		TimeoutSeconds: int(sess.Timeout / time.Second),
		MemoryMB:       sess.MemoryMB,
		ExpiresAt:      sess.ExpiresAt,
		Runtime:        h.manager.RuntimeName(),
	})
}

type replClientMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type replServerMessage struct {
	Type           string     `json:"type"`
	SessionID      string     `json:"session_id,omitempty"`
	Language       string     `json:"language,omitempty"`
	TimeoutSeconds int        `json:"timeout_seconds,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	Data           string     `json:"data,omitempty"`
	Code           int        `json:"code,omitempty"`
	Reason         string     `json:"reason,omitempty"`
	ErrorCode      int        `json:"error_code,omitempty"`
	Message        string     `json:"message,omitempty"`
}

// Stream 把 WebSocket 与 PTY 双向桥接。
func (h *ReplHandler) Stream(c *gin.Context) {
	id, err := h.authenticateWS(c)
	if err != nil {
		c.Error(err)
		return
	}

	sess, err := h.manager.Get(c.Param("sessionId"), id.UserID)
	if err != nil {
		c.Error(err)
		return
	}
	if err := sess.MarkAttached(); err != nil {
		c.Error(err)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		sess.Close("upgrade-failed")
		return // Upgrade 已写回 HTTP 错误，不能再走 JSON 出口
	}
	defer conn.Close()
	conn.SetReadLimit(16 << 10)

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.pump(conn, sess)
	}()

	// 读循环是唯一的 WebSocket reader；写只发生在 pump 里，避免并发写帧。
	for {
		var msg replClientMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		switch msg.Type {
		case "input":
			if err := sess.Write([]byte(msg.Data)); err != nil {
				sess.Close("write-failed")
				break
			}
		case "resize":
			sess.Resize(msg.Cols, msg.Rows)
		case "close":
			sess.Close("client-close")
		default:
			// 未知消息忽略，保持终端连接可用。
		}
		if sess.Reason() == "client-close" || sess.Reason() == "write-failed" {
			break
		}
	}

	sess.Close("client-close")
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func (h *ReplHandler) authenticateWS(c *gin.Context) (*token.Identity, error) {
	raw := strings.TrimSpace(c.Query("token"))
	if raw == "" {
		if header := strings.TrimSpace(c.GetHeader("Authorization")); strings.HasPrefix(strings.ToLower(header), "bearer ") {
			raw = strings.TrimSpace(header[len("Bearer "):])
		}
	}
	if raw == "" {
		return nil, apperr.New(apperr.CodeUnauthorized, "请先登录")
	}
	id, err := h.tokens.Parse(raw)
	if err != nil {
		return nil, err
	}
	if !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "REPL 仅对注册用户开放")
	}
	return id, nil
}

func (h *ReplHandler) pump(conn *websocket.Conn, sess *repl.Session) {
	defer conn.Close() // 让读循环立即退出，避免超时后 handler 卡在 ReadJSON
	write := func(msg replServerMessage) error {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteJSON(msg)
	}

	if err := sess.Start(); err != nil {
		code := apperr.From(err).Code
		_ = write(replServerMessage{Type: "error", ErrorCode: int(code), Message: err.Error()})
		sess.Close("start-failed")
		return
	}

	ready := replServerMessage{
		Type:           "ready",
		SessionID:      sess.ID,
		Language:       string(sess.Language),
		TimeoutSeconds: int(sess.Timeout / time.Second),
		ExpiresAt:      &sess.ExpiresAt,
	}
	if err := write(ready); err != nil {
		sess.Close("write-failed")
		return
	}

	// PTY 读取放到独立 goroutine：主循环还要处理「等首个提示符再写 seed 代码」。
	// bash 在交互模式初始化时会丢弃过早写入的输入，所以不能 Start 后立刻灌代码。
	type chunk struct {
		data []byte
		err  error
	}
	ch := make(chan chunk, 128)
	go func() {
		defer close(ch)
		buf := make([]byte, 4096)
		for {
			n, err := sess.Read(buf)
			if n > 0 {
				payload := make([]byte, n)
				copy(payload, buf[:n])
				ch <- chunk{data: payload}
			}
			if err != nil {
				ch <- chunk{err: err}
				return
			}
		}
	}()

	seedPending := sess.SeedCode != ""
	var seedC <-chan time.Time
	var seedTimer *time.Timer
	if seedPending {
		seedTimer = time.NewTimer(250 * time.Millisecond)
		seedC = seedTimer.C
		defer seedTimer.Stop()
	}
	seed := func() {
		if !seedPending {
			return
		}
		seedPending = false
		if err := sess.SeedInput(sess.SeedCode); err != nil {
			sess.Close("write-failed")
		}
	}

	for {
		select {
		case c, ok := <-ch:
			if !ok {
				goto done
			}
			if len(c.data) > 0 {
				if err := write(replServerMessage{Type: "output", Data: string(c.data)}); err != nil {
					sess.Close("write-failed")
					goto done
				}
				seed()
			}
			if c.err != nil {
				goto done
			}
		case <-seedC:
			seed()
		}
		if sess.Reason() == "write-failed" {
			goto done
		}
	}

done:
	if seedTimer != nil {
		seedTimer.Stop()
	}
	code := sess.Wait()
	sess.Close("exit") // 自然退出也要立刻释放并发名额；已超时/断开时 reason 不会被覆盖
	_ = write(replServerMessage{Type: "exit", Code: code, Reason: sess.Reason()})
}
