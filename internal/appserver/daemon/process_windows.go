//go:build windows

package daemon

import (
	"os/exec"
	"syscall"
)

// configureDetached 在 Windows 上隐藏子进程窗口；脱离作业对象的完整处理随
// 命名管道传输一起作为后续项。
func configureDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
