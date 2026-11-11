//go:build windows

package host

import (
	"os/exec"
	"syscall"
)

const (
	// detachedProcess 让新进程不继承控制台。
	detachedProcess = 0x00000008
	// createNewProcessGroup 让新进程独立成组。
	createNewProcessGroup = 0x00000200
)

// configureDetached 让安装器/新版本脱离当前进程组，父进程退出后继续运行。
func configureDetached(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewProcessGroup | detachedProcess,
	}
}
