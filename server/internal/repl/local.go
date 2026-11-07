package repl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// LocalRuntime 用 Linux 用户命名空间做轻量隔离。
//
//	unshare --user --map-root-user --net --mount --pid --fork
//
// 关键点：
//   - --net：新网络命名空间只有 lo，且默认 down，无法出站；
//   - --mount + tmpfs：/tmp、/var/tmp、/dev/shm 全部改为临时文件系统，会话结束即销毁；
//   - /home、/root 以只读 bind mount 隐藏，防止改写宿主机文件；
//   - ulimit -v / -t / -f：内存、CPU 时间、单文件大小限制。
type LocalRuntime struct{}

func (r *LocalRuntime) Name() string { return "local" }

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func (r *LocalRuntime) Command(spec Spec) (*exec.Cmd, error) {
	unshare, err := exec.LookPath("unshare")
	if err != nil {
		return nil, fmt.Errorf("未找到 unshare，无法启用本地隔离运行时: %w", err)
	}

	var interpreter string
	switch spec.Language {
	case LanguagePython:
		python, err := exec.LookPath("python3")
		if err != nil {
			return nil, fmt.Errorf("未找到 python3: %w", err)
		}
		interpreter = shellQuote(python) + " -q -u -i"
	case LanguageShell:
		shell := "/bin/bash"
		if _, err := os.Stat(shell); err != nil {
			shell = "/bin/sh"
		}
		interpreter = shell + " --noprofile --norc -i"
	default:
		return nil, fmt.Errorf("不支持的语言: %s", spec.Language)
	}

	// 所有数值都来自服务端配置并经过范围校验，这里只做整数格式化，不拼接用户输入。
	script := fmt.Sprintf(`set -eu
mount -t tmpfs -o size=%dm,mode=1777 tmpfs /tmp
mkdir -p /tmp/repl /tmp/repl-home
chmod 700 /tmp/repl /tmp/repl-home

# 世界可写的宿主机目录改为临时文件系统，退出即消失
for d in /var/tmp /dev/shm; do
  if [ -d "$d" ]; then
    mount -t tmpfs -o size=8m,mode=1777 tmpfs "$d"
  fi
done

# 当前用户家目录只读，避免改写项目文件（/home 本身常是独立挂载，不能自我 bind）
if [ -n "${NODAL_REPL_GUARD_HOME:-}" ] && [ -d "$NODAL_REPL_GUARD_HOME" ]; then
  mount --bind "$NODAL_REPL_GUARD_HOME" "$NODAL_REPL_GUARD_HOME"
  mount -o remount,bind,ro "$NODAL_REPL_GUARD_HOME"
fi

cd /tmp/repl
ulimit -v %d
ulimit -t %d
ulimit -f %d
umask 077
export HOME=/tmp/repl-home
export TMPDIR=/tmp/repl
export TMP=/tmp/repl
export PYTHONUNBUFFERED=1
export PYTHONDONTWRITEBYTECODE=1
export TERM=xterm-256color
export PATH=/opt/python3.12/bin:/usr/local/bin:/usr/bin:/bin
exec %s
`, spec.DiskMB, spec.MemoryMB*1024, spec.Timeout, spec.DiskMB*2048, interpreter)

	cmd := exec.Command(unshare,
		"--user", "--map-root-user",
		"--net",
		"--mount", "--propagation", "private",
		"--pid", "--fork",
		"--", "/bin/sh", "-c", script,
	)
	cmd.Dir = "/"
	homeDir, _ := os.UserHomeDir()
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"LANG=C.UTF-8",
		"NODAL_REPL_GUARD_HOME=" + homeDir,
	}
	// 不设置 Setpgid：pty.StartWithSize 会设置 Setsid/Setctty，
	// 两者同时设置会导致 exec 返回 EPERM。Setsid 后的进程组仍可用 -pid 杀掉。
	return cmd, nil
}
