// Package terminal 提供本机伪终端会话：启动用户默认 shell、双向字节流、窗口尺寸变更、
// 停止时回收整棵进程树，以及退出状态。
//
// 职责边界：
//   - 只做伪终端会话本身：启动、读写、尺寸、停止、退出状态；
//   - 不做行缓冲、不做字符集转换，写入与读回都是原始字节；
//   - 工作目录由调用方给出（会话工作目录），本包不自行推测；
//   - 不承载界面与命令历史，也不执行任何会话内命令（`!` 命令由服务端执行）。
//
// 生命周期约定：一个会话一个实例，由 Registry 保证。重复启动同一会话被明确拒绝
// （ErrAlreadyRunning），不返回已有实例；上一实例已退出时允许替换。
//
// 第三方依赖：仅本包使用 github.com/creack/pty，且只用于伪终端的创建、尺寸设置与设备读写。
// 该例外的设计与理由见 docs/domains/A.md 的伪终端方案附录与 docs/domains/D9-terminal.md。
package terminal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultCols 是未指定列数时使用的列数。
	DefaultCols uint16 = 80
	// DefaultRows 是未指定行数时使用的行数。
	DefaultRows uint16 = 24
	// DefaultTerm 是 TERM 缺省值；伪终端内的程序依赖它选择终端能力。
	DefaultTerm = "xterm-256color"
	// DefaultGracePeriod 是停止时先发 SIGHUP、等待 shell 自行结束的宽限期。
	DefaultGracePeriod = 500 * time.Millisecond
	// defaultStopTimeout 是强杀进程组后等待 shell 退出的上限。
	defaultStopTimeout = 2 * time.Second
	// defaultSweepTimeout 是清扫同一会话内残留进程的上限。
	defaultSweepTimeout = 3 * time.Second
)

var (
	// ErrUnsupportedPlatform 表示当前平台没有伪终端实现。
	ErrUnsupportedPlatform = errors.New("当前平台不支持伪终端")
	// ErrEmptySessionID 表示调用方没有给出会话标识。
	ErrEmptySessionID = errors.New("会话标识为空")
	// ErrEmptyWorkingDir 表示调用方没有给出工作目录。
	ErrEmptyWorkingDir = errors.New("工作目录为空")
	// ErrWorkingDirNotExist 表示工作目录不存在或不可访问。
	ErrWorkingDirNotExist = errors.New("工作目录不存在或不可访问")
	// ErrWorkingDirNotDir 表示工作目录存在但不是目录。
	ErrWorkingDirNotDir = errors.New("工作目录不是目录")
	// ErrShellNotFound 表示找不到可执行的 shell。
	ErrShellNotFound = errors.New("找不到可用的 shell")
	// ErrInvalidSize 表示行列数不合法。
	ErrInvalidSize = errors.New("终端行列数不合法")
	// ErrTerminalClosed 表示终端已停止，设备不可再用。
	ErrTerminalClosed = errors.New("终端已关闭")
	// ErrAlreadyRunning 表示该会话已有运行中的终端实例。
	ErrAlreadyRunning = errors.New("该会话已有运行中的终端")
	// ErrNoTerminal 表示该会话没有登记过终端实例。
	ErrNoTerminal = errors.New("该会话没有终端实例")
	// ErrStopTimeout 表示停止时 shell 没有在期限内退出。
	ErrStopTimeout = errors.New("停止终端超时")
	// ErrResidualProcesses 表示清扫期限已到，同一会话内仍有存活进程。
	ErrResidualProcesses = errors.New("停止后仍有残留进程")
)

// Options 是启动终端的可选参数；零值即默认行为。
type Options struct {
	// Shell 是 shell 可执行文件路径或名字；为空时取环境变量 SHELL，
	// 再为空则用平台缺省（macOS 为 /bin/zsh，其余平台为 /bin/sh）。
	Shell string
	// ShellArgs 是传给 shell 的参数，默认不传。
	// shell 的 stdin 是伪终端，因此无需 -i 也会进入交互模式。
	ShellArgs []string
	// Env 是子进程环境；为空时继承当前进程环境。
	// TERM 缺失时补 DefaultTerm（除非 Term 被置空字符串）。
	Env []string
	// Cols 是初始列数；0 表示 DefaultCols。
	Cols uint16
	// Rows 是初始行数；0 表示 DefaultRows。
	Rows uint16
	// Term 是 TERM 缺省值；为空表示 DefaultTerm。
	Term string
	// GracePeriod 是停止时等待 shell 自行结束的宽限期；0 表示 DefaultGracePeriod。
	GracePeriod time.Duration
}

// ExitStatus 描述 shell 进程的退出结果。
type ExitStatus struct {
	// Code 是退出码；被信号终止时为 -1。
	Code int
	// Signal 是导致终止的信号编号；正常退出为 0。
	Signal int
	// Signaled 表示进程是否被信号终止。
	Signaled bool
}

// String 返回可读的退出结果。
func (s ExitStatus) String() string {
	if s.Signaled {
		return fmt.Sprintf("被信号 %d 终止", s.Signal)
	}
	return fmt.Sprintf("退出码 %d", s.Code)
}

// Terminal 是一个运行在伪终端上的 shell 会话。
//
// 并发约定：Write 与 Resize 可以并发调用；Read 只应由一个 goroutine 调用，
// 多个并发读会让输出字节在多个调用方之间交错。Stop 可以与上述任一方法并发调用。
type Terminal struct {
	sessionID string
	dir       string
	shell     string
	args      []string
	env       []string
	cols      uint16
	rows      uint16
	grace     time.Duration

	cmd  *exec.Cmd
	pid  int
	pgid int
	tty  string

	mu     sync.Mutex
	ptmx   *os.File
	closed bool
	exited bool
	status ExitStatus

	// done 在 shell 退出并回收后关闭。
	done chan struct{}

	writeMu sync.Mutex

	stopOnce sync.Once
	stopErr  error
}

// Start 在 dir 下用伪终端启动 shell。dir 必须是已存在的目录，本包不推测工作目录。
func Start(sessionID, dir string, opts Options) (*Terminal, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, ErrEmptySessionID
	}
	if !platformSupported() {
		return nil, ErrUnsupportedPlatform
	}
	workDir, err := resolveWorkingDir(dir)
	if err != nil {
		return nil, err
	}
	shell, err := resolveShell(opts.Shell, os.Getenv, defaultShell())
	if err != nil {
		return nil, err
	}
	cols, rows := normalizeSize(opts.Cols, opts.Rows)
	grace := opts.GracePeriod
	if grace <= 0 {
		grace = DefaultGracePeriod
	}
	term := opts.Term
	if term == "" {
		term = DefaultTerm
	}
	env := buildEnv(opts.Env, term)
	args := append([]string(nil), opts.ShellArgs...)

	cmd := exec.Command(shell, args...)
	cmd.Dir = workDir
	cmd.Env = env

	ptmx, err := startPTY(cmd, cols, rows)
	if err != nil {
		return nil, err
	}

	t := &Terminal{
		sessionID: sessionID,
		dir:       workDir,
		shell:     shell,
		args:      args,
		env:       env,
		cols:      cols,
		rows:      rows,
		grace:     grace,
		cmd:       cmd,
		pid:       cmd.Process.Pid,
		// 伪终端以独立会话启动，shell 既是会话首进程也是进程组首进程。
		pgid: cmd.Process.Pid,
		ptmx: ptmx,
		done: make(chan struct{}),
	}
	// 控制终端名用来在停止时找到整个会话的进程（作业控制会让子作业自成一进程组）。
	t.tty = waitForTTY(cmd.Process.Pid)
	go t.waitExit()
	return t, nil
}

// waitExit 回收 shell 进程并记录退出状态；自然的 exit 与 Stop 走的都是这条路径。
func (t *Terminal) waitExit() {
	_ = t.cmd.Wait()
	status := decodeStatus(t.cmd.ProcessState)
	t.mu.Lock()
	t.status = status
	t.exited = true
	t.mu.Unlock()
	close(t.done)
}

// SessionID 返回调用方登记的会话标识。
func (t *Terminal) SessionID() string { return t.sessionID }

// Dir 返回 shell 的工作目录（绝对路径）。
func (t *Terminal) Dir() string { return t.dir }

// Shell 返回实际启动的 shell 可执行文件路径。
func (t *Terminal) Shell() string { return t.shell }

// Args 返回传给 shell 的参数副本。
func (t *Terminal) Args() []string { return append([]string(nil), t.args...) }

// PID 返回 shell 进程号（同时是进程组标识）。
func (t *Terminal) PID() int { return t.pid }

// TTY 返回 shell 的控制终端名（如 ttys009），供诊断使用；无法取得时为空。
func (t *Terminal) TTY() string { return t.tty }

// Size 返回最近一次设置的行列数。
func (t *Terminal) Size() (cols, rows uint16) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cols, t.rows
}

// Alive 表示 shell 尚未退出且终端设备仍可用。
func (t *Terminal) Alive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ptmx != nil && !t.closed && !t.exited
}

// Read 读取终端输出。读到的就是设备上的原始字节，不做任何转换。
// 终端停止或 shell 退出后返回 io.EOF。
func (t *Terminal) Read(p []byte) (int, error) {
	f, err := t.device()
	if err != nil {
		return 0, io.EOF
	}
	n, err := f.Read(p)
	if err != nil {
		if isDeviceGone(err) {
			return n, io.EOF
		}
		return n, fmt.Errorf("读取终端输出: %w", err)
	}
	return n, nil
}

// Write 把字节写入终端输入，不做行缓冲与编码转换。
// 终端停止后返回 ErrTerminalClosed。
func (t *Terminal) Write(p []byte) (int, error) {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	f, err := t.device()
	if err != nil {
		return 0, err
	}
	n, err := f.Write(p)
	if err != nil {
		if isDeviceGone(err) {
			return n, ErrTerminalClosed
		}
		return n, fmt.Errorf("向终端写入: %w", err)
	}
	return n, nil
}

// Resize 变更新建窗口的行列数。内核会给终端前台进程组发送 SIGWINCH。
func (t *Terminal) Resize(cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return ErrInvalidSize
	}
	f, err := t.device()
	if err != nil {
		return err
	}
	if err := setTTYSize(f, cols, rows); err != nil {
		return fmt.Errorf("变更终端尺寸: %w", err)
	}
	t.mu.Lock()
	t.cols, t.rows = cols, rows
	t.mu.Unlock()
	return nil
}

// ExitStatus 返回已记录的退出状态；第二个返回值表示是否已经有结果。
func (t *Terminal) ExitStatus() (ExitStatus, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status, t.exited
}

// Wait 阻塞到 shell 退出（自然退出或被 Stop 终止），返回退出状态。
func (t *Terminal) Wait() ExitStatus {
	<-t.done
	status, _ := t.ExitStatus()
	return status
}

// Stop 停止终端并回收进程树，可重复调用。
//
// 顺序：在 shell 还活着时按控制终端取进程快照（shell 一退出这些关联就消失）；
// 向进程组发 SIGHUP 并等宽限期；未退出则强杀进程组；强杀快照里的进程
// （作业控制会让子作业落在自己的进程组里，只杀父进程或父进程组都不够）；
// 关闭主设备挂断终端；等待 shell 退出；最后核对快照里的进程都已消失。
func (t *Terminal) Stop() error {
	t.stopOnce.Do(func() { t.stopErr = t.shutdown() })
	return t.stopErr
}

func (t *Terminal) shutdown() error {
	// 1. 快照必须最先取：控制终端关联在 shell 退出后即消失，之后再找就找不全。
	snapshot, snapErr := ttyMembers(t.tty)

	// 2. 先礼后兵：SIGHUP 让交互 shell 自行结束，通常也带走它的作业。
	signalGroup(t.pgid, sigHUP)
	graceful := t.waitDone(t.grace)
	if !graceful {
		// 3. 宽限期内没退出：强杀整个进程组，而不是只杀父进程。
		signalGroup(t.pgid, sigKILL)
		// shell 还没退出，说明期间的作业仍在控制终端下，补一次快照。
		if extra, err := ttyMembers(t.tty); err == nil {
			snapshot = mergePIDs(snapshot, extra)
		}
	}
	// 4. 强杀快照里的每个进程，覆盖自成进程组或已脱离父进程的后台作业。
	killPIDs(snapshot, sigKILL)

	// 5. 关闭主设备：终端挂断，前台作业随之结束，阻塞中的读返回。
	t.closeDevice()
	var err error
	if snapErr != nil {
		err = errors.Join(err, snapErr)
	}
	if !t.waitDone(defaultStopTimeout) {
		err = errors.Join(err, fmt.Errorf("%w: 会话 %s", ErrStopTimeout, t.sessionID))
	}
	// 6. 核对快照里的进程都已消失；仍活着说明回收不完整，如实上报。
	if werr := waitGone(snapshot, defaultSweepTimeout); werr != nil {
		err = errors.Join(err, werr)
	}
	return err
}

// mergePIDs 合并两组进程号并去重。
func mergePIDs(a, b []int) []int {
	seen := make(map[int]struct{}, len(a)+len(b))
	out := make([]int, 0, len(a)+len(b))
	for _, group := range [][]int{a, b} {
		for _, pid := range group {
			if _, ok := seen[pid]; ok {
				continue
			}
			seen[pid] = struct{}{}
			out = append(out, pid)
		}
	}
	return out
}

// device 取出可用的终端设备；停止后返回 ErrTerminalClosed。
func (t *Terminal) device() (*os.File, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ptmx == nil || t.closed {
		return nil, ErrTerminalClosed
	}
	return t.ptmx, nil
}

// closeDevice 关闭主设备并标记终端已关闭；重复调用无副作用。
func (t *Terminal) closeDevice() {
	t.mu.Lock()
	f := t.ptmx
	t.ptmx = nil
	t.closed = true
	t.mu.Unlock()
	if f != nil {
		_ = f.Close()
	}
}

// waitForTTY 取得 shell 的控制终端名；刚启动时 ps 可能还看不到，短暂重试。
func waitForTTY(pid int) string {
	for i := 0; i < 10; i++ {
		if tty := processTTY(pid); tty != "" {
			return tty
		}
		time.Sleep(5 * time.Millisecond)
	}
	return ""
}

// waitDone 等待 shell 退出，最多等 d；返回是否已退出。
func (t *Terminal) waitDone(d time.Duration) bool {
	if d <= 0 {
		select {
		case <-t.done:
			return true
		default:
			return false
		}
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-t.done:
		return true
	case <-timer.C:
		return false
	}
}

// resolveWorkingDir 校验并规范化工作目录；空目录是调用方错误，不猜测。
func resolveWorkingDir(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", ErrEmptyWorkingDir
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("规范化工作目录: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrWorkingDirNotExist, abs)
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrWorkingDirNotExist, abs)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrWorkingDirNotDir, abs)
	}
	// 解析符号链接：shell 的 pwd 报的是物理路径，按同一路径登记才能直接比对。
	return real, nil
}

// resolveShell 依次取显式指定、环境变量 SHELL、平台缺省值，并校验可执行。
func resolveShell(explicit string, getenv func(string) string, fallback string) (string, error) {
	candidate := strings.TrimSpace(explicit)
	if candidate == "" && getenv != nil {
		candidate = strings.TrimSpace(getenv("SHELL"))
	}
	if candidate == "" {
		candidate = fallback
	}
	path, err := exec.LookPath(candidate)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrShellNotFound, candidate)
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("%w: %s", ErrShellNotFound, path)
	}
	return path, nil
}

// defaultShell 返回本平台缺省的 shell。
func defaultShell() string {
	if runtime.GOOS == "darwin" {
		return "/bin/zsh"
	}
	return "/bin/sh"
}

// buildEnv 组合子进程环境；base 为空时继承当前进程环境，TERM 缺失时补上。
func buildEnv(base []string, term string) []string {
	env := base
	if env == nil {
		env = os.Environ()
	}
	out := make([]string, 0, len(env)+1)
	hasTerm := false
	for _, kv := range env {
		if strings.HasPrefix(kv, "TERM=") {
			hasTerm = true
		}
		out = append(out, kv)
	}
	if !hasTerm && term != "" {
		out = append(out, "TERM="+term)
	}
	return out
}

// normalizeSize 补齐零值行列数。
func normalizeSize(cols, rows uint16) (uint16, uint16) {
	if cols == 0 {
		cols = DefaultCols
	}
	if rows == 0 {
		rows = DefaultRows
	}
	return cols, rows
}
