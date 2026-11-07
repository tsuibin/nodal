//go:build windows

package replsandbox

import "os/exec"

// terminateGroup 在 Windows 上没有进程组信号语义，只能杀单个进程；
// 调用方随后还会调用 cmd.Process.Kill() 兜底。
//
// 说明：Windows 上本地演示请设 NODAL_REPL_ENABLED=false —— REPL 的隔离依赖
// Linux 的 unshare / 用户命名空间 / tmpfs，本来就不在 Windows 的支持范围内。
func terminateGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
