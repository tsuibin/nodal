package replsandbox

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"

	"github.com/nodal/nodal-server/apperr"
)

const (
	stateCreated = "created"
	stateRunning = "running"
	stateClosed  = "closed"
)

// Session 是一次 REPL 会话。
//
// 会话在 Create 时登记并开始计时，在第一次 WebSocket attach 时才真正拉起进程，
// 这样创建接口可以立即返回，而 seed 代码的输出不会在连接建立前丢失。
type Session struct {
	ID        string
	UserID    uint
	OwnerKey  string
	Language  Language
	CreatedAt time.Time
	ExpiresAt time.Time
	MemoryMB  int
	DiskMB    int
	Timeout   time.Duration
	SeedCode  string

	runtime Runtime
	onClose func(*Session)

	mu        sync.Mutex
	state     string
	attached  bool
	cmd       *exec.Cmd
	pty       *os.File
	reason    string
	exitCode  int
	done      chan struct{}
	closeOnce sync.Once
	timer     *time.Timer
}

func (s *Session) Done() <-chan struct{} { return s.done }

// MarkAttached 保证一个会话只能被一个 WebSocket 连接接管。
func (s *Session) MarkAttached() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == stateClosed {
		return apperr.New(apperr.CodeReplNotFound, "REPL 会话已结束，请重新创建")
	}
	if s.attached {
		return apperr.New(apperr.CodeReplAttached, "该 REPL 会话已有一个终端连接")
	}
	s.attached = true
	return nil
}

// Start 拉起隔离进程并接到 PTY。只允许调用一次。
func (s *Session) Start() error {
	s.mu.Lock()
	if s.state == stateClosed {
		s.mu.Unlock()
		return apperr.New(apperr.CodeReplNotFound, "REPL 会话已结束，请重新创建")
	}
	if s.state == stateRunning {
		s.mu.Unlock()
		return apperr.New(apperr.CodeReplAttached, "REPL 会话已启动")
	}

	cmd, err := s.runtime.Command(Spec{
		SessionID: s.ID,
		Language:  s.Language,
		MemoryMB:  s.MemoryMB,
		Timeout:   int(s.Timeout / time.Second),
		DiskMB:    s.DiskMB,
	})
	if err != nil {
		s.state = stateClosed
		s.mu.Unlock()
		s.Close("start-failed")
		return apperr.New(apperr.CodeReplUnavailable, "沙箱启动失败："+err.Error())
	}

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 100, Rows: 30})
	if err != nil {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		s.state = stateClosed
		s.mu.Unlock()
		s.Close("start-failed")
		return apperr.New(apperr.CodeReplUnavailable, "终端初始化失败："+err.Error())
	}

	s.cmd = cmd
	s.pty = ptmx
	s.state = stateRunning
	s.mu.Unlock()
	return nil
}

// Read 读取终端输出。进程退出或超时被杀时返回错误。
func (s *Session) Read(p []byte) (int, error) {
	s.mu.Lock()
	ptmx := s.pty
	s.mu.Unlock()
	if ptmx == nil {
		return 0, fmt.Errorf("repl: 终端尚未启动")
	}
	return ptmx.Read(p)
}

// Write 把用户输入写进终端。
func (s *Session) Write(data []byte) error {
	s.mu.Lock()
	ptmx := s.pty
	closed := s.state == stateClosed
	s.mu.Unlock()
	if closed || ptmx == nil {
		return apperr.New(apperr.CodeReplNotFound, "REPL 会话已结束")
	}
	_, err := ptmx.Write(data)
	return err
}

// Resize 同步终端窗口大小。
func (s *Session) Resize(cols, rows uint16) {
	s.mu.Lock()
	ptmx := s.pty
	s.mu.Unlock()
	if ptmx == nil || cols == 0 || rows == 0 {
		return
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Cols: cols, Rows: rows})
}

// Wait 回收子进程并返回退出码。只应由输出泵调用一次。
func (s *Session) Wait() int {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()
	if cmd == nil {
		return -1
	}
	err := cmd.Wait()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
				if ws.Signaled() {
					code = 128 + int(ws.Signal())
				} else {
					code = ws.ExitStatus()
				}
			} else {
				code = -1
			}
		} else {
			code = -1
		}
	}
	s.mu.Lock()
	s.exitCode = code
	s.mu.Unlock()
	return code
}

func (s *Session) ExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCode
}

func (s *Session) Reason() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reason == "" {
		return "exit"
	}
	return s.reason
}

func (s *Session) Expired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Close 是幂等的：关闭 PTY、杀掉整个进程组、释放会话名额。
func (s *Session) Close(reason string) {
	s.closeOnce.Do(func() {
		if s.timer != nil {
			s.timer.Stop()
		}

		s.mu.Lock()
		if s.reason == "" {
			s.reason = reason
		}
		s.state = stateClosed
		ptmx := s.pty
		cmd := s.cmd
		s.mu.Unlock()

		if ptmx != nil {
			_ = ptmx.Close()
		}
		if cmd != nil && cmd.Process != nil {
			// 先收整个进程组（子进程在独立进程组里），再兜底杀单个进程。
			// 进程组信号的实现按平台分文件，Windows 上没有这套语义。
			terminateGroup(cmd)
			_ = cmd.Process.Kill()
		}
		slog.Info("repl session closed",
			"session_id", s.ID,
			"user_id", s.UserID,
			"reason", s.reason,
			"duration_ms", time.Since(s.CreatedAt).Milliseconds(),
		)
		close(s.done)
		if s.onClose != nil {
			s.onClose(s)
		}
	})
}

// SeedInput 把创建时传入的代码变成终端输入。
// Python 的复合语句需要空行才会执行，这里补一个换行。
func (s *Session) SeedInput(code string) error {
	if code == "" {
		return nil
	}
	input := code
	if !strings.HasSuffix(input, "\n") {
		input += "\n"
	}
	// Python 的 if/for/def 等复合语句需要再一个空行才会执行。
	if s.Language == LanguagePython && !strings.HasSuffix(input, "\n\n") {
		input += "\n"
	}
	return s.Write([]byte(input))
}
