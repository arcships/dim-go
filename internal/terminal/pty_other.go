//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !dragonfly

package terminal

import (
	"errors"
	"os"
	"os/exec"
	"time"
)

// 本文件只保证非伪终端平台（含 Windows）能编译并给出明确失败，不提供替代实现。
// 详见 docs/domains/D9-terminal.md 的跨平台一节。

const (
	// sigHUP 在无伪终端平台没有意义，仅为共享代码可编译。
	sigHUP = 0
	// sigKILL 同上。
	sigKILL = 0
)

// platformSupported 表示当前平台没有伪终端实现。
func platformSupported() bool { return false }

// startPTY 在无伪终端平台直接失败。
func startPTY(*exec.Cmd, uint16, uint16) (*os.File, error) {
	return nil, ErrUnsupportedPlatform
}

// setTTYSize 在无伪终端平台直接失败。
func setTTYSize(*os.File, uint16, uint16) error {
	return ErrUnsupportedPlatform
}

// signalGroup 在无伪终端平台没有进程组可回收。
func signalGroup(int, int) {}

// isDeviceGone 判断错误是否表示设备已关闭。
func isDeviceGone(err error) bool {
	return errors.Is(err, os.ErrClosed)
}

// decodeStatus 在无伪终端平台给出与 os.ProcessState 一致的退出码。
func decodeStatus(ps *os.ProcessState) ExitStatus {
	if ps == nil {
		return ExitStatus{Code: -1}
	}
	return ExitStatus{Code: ps.ExitCode()}
}

// killPIDs 在无伪终端平台没有进程可回收。
func killPIDs([]int, int) {}

// ttyMembers 在无伪终端平台没有控制终端可查。
func ttyMembers(string) ([]int, error) { return nil, nil }

// alivePIDs 在无伪终端平台没有进程可查。
func alivePIDs([]int) ([]int, error) { return nil, nil }

// waitGone 在无伪终端平台无需等待。
func waitGone([]int, time.Duration) error { return nil }

// processTTY 在无伪终端平台没有控制终端可查。
func processTTY(int) string { return "" }
