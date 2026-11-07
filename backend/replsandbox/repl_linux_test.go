package replsandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodal/nodal-server/token"
)

func localAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("unshare"); err != nil {
		t.Skip("unshare 不可用")
	}
	if err := exec.Command("unshare", "--user", "--map-root-user", "--net", "--mount", "--pid", "--fork", "true").Run(); err != nil {
		t.Skipf("当前环境不支持用户命名空间: %v", err)
	}
}

func newTestManager(timeout time.Duration) *Manager {
	return NewManager(Config{
		Enabled:      true,
		Runtime:      &LocalRuntime{},
		Timeout:      timeout,
		MemoryMB:     256,
		MaxSessions:  4,
		MaxPerUser:   2,
		DiskMB:       16,
		MaxCodeBytes: 16 * 1024,
	})
}

type readResult struct {
	text string
	err  error
}

func readUntil(s *Session, marker string, timeout time.Duration) readResult {
	out := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 2048)
		var b strings.Builder
		for {
			n, err := s.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
				if strings.Contains(b.String(), marker) {
					out <- readResult{text: b.String()}
					return
				}
			}
			if err != nil {
				out <- readResult{text: b.String(), err: err}
				return
			}
		}
	}()
	select {
	case r := <-out:
		return r
	case <-time.After(timeout):
		return readResult{err: os.ErrDeadlineExceeded}
	}
}

func TestLocalRuntimePythonAndIsolation(t *testing.T) {
	localAvailable(t)
	mgr := newTestManager(30 * time.Second)
	defer mgr.CloseAll()

	owner := &token.Identity{Role: token.RoleUser, UserID: 42}
	sess, err := mgr.Create(owner, LanguagePython, "print(6*7)")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer sess.Close("test")
	if err := sess.MarkAttached(); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if err := sess.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := sess.SeedInput(sess.SeedCode); err != nil {
		t.Fatalf("seed: %v", err)
	}

	r := readUntil(sess, "42", 15*time.Second)
	if r.err != nil || !strings.Contains(r.text, "42") {
		t.Fatalf("seed output = %q, err=%v", r.text, r.err)
	}

	// 网络必须不可达
	if err := sess.Write([]byte("import socket\nsocket.create_connection(('1.1.1.1', 80), 1)\n\n")); err != nil {
		t.Fatalf("write network probe: %v", err)
	}
	r = readUntil(sess, "unreachable", 10*time.Second)
	if !strings.Contains(strings.ToLower(r.text), "unreachable") {
		t.Fatalf("网络隔离未生效: %q err=%v", r.text, r.err)
	}

	// 写文件后关闭会话：宿主机不得留下文件
	hostPath := filepath.Join(os.TempDir(), "nodal-repl-persist-"+token.NewID())
	if err := sess.Write([]byte("open('" + strings.ReplaceAll(hostPath, "'", "") + "', 'w').write('x')\n\n")); err != nil {
		t.Fatalf("write persistence probe: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	sess.Close("test")
	if _, err := os.Stat(hostPath); err == nil {
		_ = os.Remove(hostPath)
		t.Fatalf("文件穿透到宿主机: %s", hostPath)
	}

	sess2, err := mgr.Create(owner, LanguageShell, "echo shell-ok")
	if err != nil {
		t.Fatalf("create shell: %v", err)
	}
	defer sess2.Close("test")
	if err := sess2.MarkAttached(); err != nil {
		t.Fatalf("attach shell: %v", err)
	}
	if err := sess2.Start(); err != nil {
		t.Fatalf("start shell: %v", err)
	}
	// bash 交互模式初始化时会丢弃过早输入；handler 会等首个提示符，这里等价等待。
	time.Sleep(300 * time.Millisecond)
	if err := sess2.SeedInput(sess2.SeedCode); err != nil {
		t.Fatalf("seed shell: %v", err)
	}
	r = readUntil(sess2, "shell-ok", 10*time.Second)
	if !strings.Contains(r.text, "shell-ok") {
		t.Fatalf("shell output = %q err=%v", r.text, r.err)
	}
}

func TestManagerSessionLimitAndTimeout(t *testing.T) {
	localAvailable(t)
	mgr := newTestManager(5 * time.Second)
	owner := &token.Identity{Role: token.RoleUser, UserID: 7}

	s1, err := mgr.Create(owner, LanguageShell, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s1.Close("test")
	s2, err := mgr.Create(owner, LanguageShell, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close("test")

	if _, err := mgr.Create(owner, LanguageShell, ""); err == nil {
		t.Fatalf("超出单用户并发上限应失败")
	}

	select {
	case <-s1.Done():
		t.Fatalf("会话提前结束")
	case <-time.After(1 * time.Second):
	}
	select {
	case <-s1.Done():
	case <-time.After(5 * time.Second):
		t.Fatalf("会话未按 5s 超时")
	}
	if s1.Reason() != "timeout" {
		t.Fatalf("close reason = %s", s1.Reason())
	}

	// 释放名额后可以再创建
	if _, err := mgr.Create(owner, LanguageShell, ""); err != nil {
		t.Fatalf("释放后创建失败: %v", err)
	}
}

func TestGuestCannotCreate(t *testing.T) {
	mgr := newTestManager(30 * time.Second)
	if _, err := mgr.Create(&token.Identity{Role: token.RoleGuest, GuestID: "g"}, LanguagePython, ""); err == nil {
		t.Fatalf("游客不应创建 REPL")
	}
}
