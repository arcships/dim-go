//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package terminal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

var (
	// sigHUP 用于停止终端时先请求 shell 自行结束。
	sigHUP = syscall.SIGHUP
	// sigKILL 用于宽限期后强杀整棵进程树。
	sigKILL = syscall.SIGKILL
)

// platformSupported 表示当前平台有伪终端实现。
func platformSupported() bool { return true }

// startPTY 创建伪终端并以它作为 shell 的控制终端启动进程。
// pty.StartWithSize 会让子进程成为新会话的首进程，并设置控制终端。
func startPTY(cmd *exec.Cmd, cols, rows uint16) (*os.File, error) {
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("启动伪终端: %w", err)
	}
	return f, nil
}

// setTTYSize 设置伪终端行列数；内核据此向前台进程组发送 SIGWINCH。
func setTTYSize(f *os.File, cols, rows uint16) error {
	return pty.Setsize(f, &pty.Winsize{Cols: cols, Rows: rows})
}

// signalGroup 向进程组发送信号；组已空时忽略。
func signalGroup(pgid int, sig syscall.Signal) {
	if pgid <= 0 {
		return
	}
	_ = syscall.Kill(-pgid, sig)
}

// killPIDs 逐个发送信号；进程已不存在时忽略。
func killPIDs(pids []int, sig syscall.Signal) {
	self := os.Getpid()
	for _, pid := range pids {
		if pid <= 0 || pid == self {
			continue
		}
		_ = syscall.Kill(pid, sig)
	}
}

// isDeviceGone 判断错误是否表示终端设备已经挂断或关闭。
func isDeviceGone(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, os.ErrClosed) ||
		errors.Is(err, syscall.EIO) ||
		errors.Is(err, syscall.EBADF)
}

// decodeStatus 从进程状态解出退出码或终止信号。
func decodeStatus(ps *os.ProcessState) ExitStatus {
	if ps == nil {
		return ExitStatus{Code: -1}
	}
	if ws, ok := ps.Sys().(syscall.WaitStatus); ok {
		if ws.Signaled() {
			return ExitStatus{Code: -1, Signal: int(ws.Signal()), Signaled: true}
		}
		return ExitStatus{Code: ws.ExitStatus()}
	}
	return ExitStatus{Code: ps.ExitCode()}
}

// processTTY 返回进程的控制终端名（如 ttys009）；没有控制终端或查询失败时返回空。
func processTTY(pid int) string {
	if pid <= 0 {
		return ""
	}
	out, err := exec.Command("ps", "-o", "tty=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	tty := strings.TrimSpace(string(out))
	if tty == "" || tty == "??" || tty == "-" {
		return ""
	}
	return tty
}

// ttyMembers 列出以 tty 为控制终端的存活进程号；僵尸进程不算存活（已无资源，等待被回收）。
//
// 为什么按控制终端找，而不是按会话或进程组：交互 shell 打开作业控制后，每个作业
// 自成一个进程组；后台作业还可能脱离父进程，父进程链与进程组都找不全。同一会话的
// 进程共享同一个控制终端，而一个终端同时只能属于一个会话，所以终端名是会话的可靠标记。
// 注意：shell 一退出，这些进程的控制终端关联就消失（ps 显示 ??），因此必须在
// shell 存活时取快照，并把快照用在随后的强杀与核对上。
func ttyMembers(tty string) ([]int, error) {
	if tty == "" {
		return nil, nil
	}
	out, err := exec.Command("ps", "-A", "-o", "pid=,tty=,stat=").Output()
	if err != nil {
		return nil, fmt.Errorf("枚举终端进程: %w", err)
	}
	self := os.Getpid()
	pids := make([]int, 0, 8)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || fields[1] != tty || pid == self {
			continue
		}
		if strings.HasPrefix(fields[2], "Z") {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

// alivePIDs 返回给定进程号中仍在运行的（僵尸不算存活）。
func alivePIDs(pids []int) ([]int, error) {
	if len(pids) == 0 {
		return nil, nil
	}
	list := make([]string, 0, len(pids))
	for _, pid := range pids {
		list = append(list, strconv.Itoa(pid))
	}
	out, err := exec.Command("ps", "-o", "pid=,stat=", "-p", strings.Join(list, ",")).Output()
	if err != nil {
		// 给定的进程都不存在时 ps 返回非零退出码，此时按没有存活处理。
		return nil, nil
	}
	alive := make([]int, 0, len(pids))
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || strings.HasPrefix(fields[1], "Z") {
			continue
		}
		alive = append(alive, pid)
	}
	return alive, nil
}

// waitGone 等待给定进程全部消失，超时报告残留。
func waitGone(pids []int, deadline time.Duration) error {
	if len(pids) == 0 {
		return nil
	}
	end := time.Now().Add(deadline)
	for {
		alive, err := alivePIDs(pids)
		if err != nil {
			return err
		}
		if len(alive) == 0 {
			return nil
		}
		if !time.Now().Before(end) {
			return fmt.Errorf("%w: 进程 %v", ErrResidualProcesses, alive)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
