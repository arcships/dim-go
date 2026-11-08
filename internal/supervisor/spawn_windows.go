//go:build windows

package supervisor

import "syscall"

// createNoWindow keeps the daemon from opening a console window. The Windows
// daemon is not detached; it is stopped through daemon.shutdown.
const createNoWindow = 0x08000000

func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
