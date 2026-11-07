//go:build !windows

package replsandbox

import (
	"os/exec"
	"syscall"
)

// terminateGroup 向整个进程组发 SIGKILL。
//
// local 运行时通过 PTY 的 Setsid 建立了独立进程组，所以负 pid 能一次收掉
// 解释器及其子进程，避免只剩孤儿子进程占着会话名额。
func terminateGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
