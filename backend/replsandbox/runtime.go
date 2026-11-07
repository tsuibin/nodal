// Package repl 提供代码节点的隔离执行环境。
//
// 运行时是可替换的：
//   - local：Linux 用户命名空间 + 网络/挂载/PID 隔离 + PTY，本机可直接跑；
//   - firecracker：通过外部 launcher 启动 Firecracker microVM（guest 内跑 ttyd），
//     launcher 的 stdio 就是终端流，平台侧只负责 WebSocket 桥接与生命周期。
//
// 两条路径共用同一个 Runtime 接口，因此会话管理、超时、内存限制、
// WebSocket 协议都不需要感知底层是 microVM 还是命名空间。
package replsandbox

import (
	"fmt"
	"os/exec"
)

// Language 是支持的解释器。
type Language string

const (
	LanguagePython Language = "python"
	LanguageShell  Language = "shell"
)

func (l Language) Valid() bool {
	return l == LanguagePython || l == LanguageShell
}

// Spec 是启动一个终端所需的全部参数。
type Spec struct {
	SessionID string
	Language  Language
	MemoryMB  int
	Timeout   int // 秒
	DiskMB    int
}

// Runtime 负责把 Spec 变成一个挂在 PTY 上的进程。
// 返回的 cmd 的 stdin/stdout/stderr 会由调用方接到伪终端。
type Runtime interface {
	Name() string
	Command(spec Spec) (*exec.Cmd, error)
}

// NewRuntime 按配置选择运行时。
func NewRuntime(name, firecrackerLauncher string) (Runtime, error) {
	switch name {
	case "", "local":
		return &LocalRuntime{}, nil
	case "firecracker":
		if firecrackerLauncher == "" {
			return nil, fmt.Errorf("NODAL_REPL_FIRECRACKER_LAUNCHER 未配置")
		}
		return &FirecrackerRuntime{Launcher: firecrackerLauncher}, nil
	default:
		return nil, fmt.Errorf("不支持的 NODAL_REPL_RUNTIME: %q（可选 local/firecracker）", name)
	}
}
