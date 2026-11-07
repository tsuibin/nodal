package replsandbox

import (
	"fmt"
	"os"
	"os/exec"
)

// FirecrackerRuntime 是 Firecracker microVM 的接入点。
//
// 本仓库不内置 kernel/rootfs 镜像，也不假设宿主机一定安装了 firecracker；
// 生产部署通过一个 launcher 可执行文件完成 VM 启动与终端桥接：
//
//  1. launcher 读取下面的环境变量，启动 Firecracker microVM；
//  2. guest 内运行 ttyd（或等价 PTY 服务），监听 vsock；
//  3. launcher 用 socat/vsock 把 guest 终端接到自己的 stdin/stdout；
//  4. 平台侧把 launcher 的 stdio 挂到 PTY，WebSocket 协议与 local 运行时一致。
//
// 这样 Firecracker 的镜像、内核、vsock CID 分配等部署细节不侵入业务代码，
// 也便于在没有 KVM 的开发机上切换到 local 运行时。
type FirecrackerRuntime struct {
	Launcher string
}

func (r *FirecrackerRuntime) Name() string { return "firecracker" }

func (r *FirecrackerRuntime) Command(spec Spec) (*exec.Cmd, error) {
	info, err := os.Stat(r.Launcher)
	if err != nil {
		return nil, fmt.Errorf("Firecracker launcher 不可用: %w", err)
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return nil, fmt.Errorf("Firecracker launcher 不是可执行文件: %s", r.Launcher)
	}

	cmd := exec.Command(r.Launcher)
	cmd.Dir = "/"
	// 只透传 launcher 必需信息，避免把服务端 JWT/数据库密钥带进沙箱侧进程。
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"LANG=C.UTF-8",
		"NODAL_REPL_SESSION_ID=" + spec.SessionID,
		"NODAL_REPL_LANGUAGE=" + string(spec.Language),
		fmt.Sprintf("NODAL_REPL_MEMORY_MB=%d", spec.MemoryMB),
		fmt.Sprintf("NODAL_REPL_TIMEOUT_SEC=%d", spec.Timeout),
		fmt.Sprintf("NODAL_REPL_DISK_MB=%d", spec.DiskMB),
	}
	// pty.StartWithSize 负责 Setsid/Setctty；不要在这里设置 Setpgid。
	return cmd, nil
}
