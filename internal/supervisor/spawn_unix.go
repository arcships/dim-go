//go:build unix

package supervisor

import "syscall"

// detachedSysProcAttr puts the daemon in its own session so it survives a
// terminal hangup of the client process.
func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
