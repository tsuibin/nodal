package replsandbox

import (
	"log/slog"
	"sync"
	"time"

	"github.com/nodal/nodal-server/apperr"
	"github.com/nodal/nodal-server/token"
)

const (
	defaultDiskMB    = 32
	maxCodeBytes     = 16 * 1024
	defaultTimeout   = 30 * time.Second
	defaultMemoryMB  = 256
	defaultMaxGlobal = 20
	defaultMaxUser   = 2
)

// Config 是 REPL 沙箱的运行参数。
type Config struct {
	Enabled      bool
	Runtime      Runtime
	Timeout      time.Duration
	MemoryMB     int
	MaxSessions  int
	MaxPerUser   int
	DiskMB       int
	MaxCodeBytes int
}

// Manager 管理所有活跃会话。会话只存在于内存，进程退出/超时后立即释放，
// 不落库，符合「禁止文件持久化」的语义。
type Manager struct {
	cfg Config

	mu       sync.Mutex
	sessions map[string]*Session
	perUser  map[uint]int
}

func NewManager(cfg Config) *Manager {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MemoryMB <= 0 {
		cfg.MemoryMB = defaultMemoryMB
	}
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = defaultMaxGlobal
	}
	if cfg.MaxPerUser <= 0 {
		cfg.MaxPerUser = defaultMaxUser
	}
	if cfg.DiskMB <= 0 {
		cfg.DiskMB = defaultDiskMB
	}
	if cfg.MaxCodeBytes <= 0 {
		cfg.MaxCodeBytes = maxCodeBytes
	}
	return &Manager{
		cfg:      cfg,
		sessions: make(map[string]*Session),
		perUser:  make(map[uint]int),
	}
}

func (m *Manager) RuntimeName() string {
	if m.cfg.Runtime == nil {
		return "none"
	}
	return m.cfg.Runtime.Name()
}

func (m *Manager) Timeout() time.Duration { return m.cfg.Timeout }
func (m *Manager) MemoryMB() int          { return m.cfg.MemoryMB }

// Create 登记一个会话。此时不启动进程，等 WebSocket attach 后再 Start。
func (m *Manager) Create(id *token.Identity, language Language, code string) (*Session, error) {
	if !m.cfg.Enabled || m.cfg.Runtime == nil {
		return nil, apperr.New(apperr.CodeReplUnavailable, "REPL 沙箱未启用")
	}
	if !id.IsUser() {
		return nil, apperr.New(apperr.CodeForbidden, "REPL 仅对注册用户开放")
	}
	if !language.Valid() {
		return nil, apperr.New(apperr.CodeReplLanguage, "只支持 python 或 shell")
	}
	if len(code) > m.cfg.MaxCodeBytes {
		return nil, apperr.New(apperr.CodeReplCodeTooLarge, "代码过长，最多 16KB")
	}

	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= m.cfg.MaxSessions {
		return nil, apperr.New(apperr.CodeReplTooMany, "当前执行任务较多，请稍后再试")
	}
	if m.perUser[id.UserID] >= m.cfg.MaxPerUser {
		return nil, apperr.New(apperr.CodeReplTooMany, "你同时运行的 REPL 会话过多，请先关闭旧会话")
	}

	s := &Session{
		ID:        token.NewID(),
		UserID:    id.UserID,
		OwnerKey:  id.OwnerKey(),
		Language:  language,
		CreatedAt: now,
		ExpiresAt: now.Add(m.cfg.Timeout),
		MemoryMB:  m.cfg.MemoryMB,
		DiskMB:    m.cfg.DiskMB,
		Timeout:   m.cfg.Timeout,
		SeedCode:  code,
		runtime:   m.cfg.Runtime,
		done:      make(chan struct{}),
		state:     stateCreated,
	}
	s.onClose = m.release
	m.sessions[s.ID] = s
	m.perUser[id.UserID]++

	// 从创建时刻开始计时：即使客户端不连接，也不会长期占用槽位。
	s.timer = time.AfterFunc(m.cfg.Timeout, func() {
		s.Close("timeout")
	})
	slog.Info("repl session created", "session_id", s.ID, "user_id", s.UserID, "language", s.Language, "runtime", m.RuntimeName())
	return s, nil
}

func (m *Manager) release(s *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[s.ID]; !ok {
		return
	}
	delete(m.sessions, s.ID)
	if n := m.perUser[s.UserID]; n <= 1 {
		delete(m.perUser, s.UserID)
	} else {
		m.perUser[s.UserID] = n - 1
	}
}

// Get 校验归属与有效期，返回可 attach 的会话。
func (m *Manager) Get(sessionID string, userID uint) (*Session, error) {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	m.mu.Unlock()
	if !ok || s.UserID != userID {
		return nil, apperr.New(apperr.CodeReplNotFound, "REPL 会话不存在或已结束")
	}
	if s.Expired() {
		s.Close("timeout")
		return nil, apperr.New(apperr.CodeReplNotFound, "REPL 会话已超时，请重新创建")
	}
	return s, nil
}

// CloseAll 在服务退出时回收所有 PTY 与子进程。
func (m *Manager) CloseAll() {
	m.mu.Lock()
	list := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		list = append(list, s)
	}
	m.mu.Unlock()
	for _, s := range list {
		s.Close("server-shutdown")
	}
}
