package terminal

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const (
	// waitTimeout 是单步断言的等待上限。
	waitTimeout = 8 * time.Second
)

// testEnv 返回隔离了用户 shell 配置文件的环境：ZDOTDIR 指向空目录，
// 并去掉 TERM，让本包补上缺省值（TERM 缺省注入由此得到验证）。
func testEnv(t *testing.T) []string {
	t.Helper()
	zdot := t.TempDir()
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "ZDOTDIR=") || strings.HasPrefix(kv, "TERM=") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, "ZDOTDIR="+zdot)
}

// startTerm 启动终端并在测试结束时停止它。
func startTerm(t *testing.T, sessionID, dir string, opts Options) *Terminal {
	t.Helper()
	if opts.Env == nil {
		opts.Env = testEnv(t)
	}
	term, err := Start(sessionID, dir, opts)
	if err != nil {
		t.Fatalf("启动终端: %v", err)
	}
	t.Cleanup(func() {
		if err := term.Stop(); err != nil {
			t.Errorf("清理终端: %v", err)
		}
	})
	return term
}

// capture 收集终端输出，模拟一个持续读取的界面。
type capture struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// startCapture 启动后台读循环；终端停止后读循环自行退出。
func startCapture(t *testing.T, term *Terminal) *capture {
	t.Helper()
	c := &capture{}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := term.Read(buf)
			if n > 0 {
				c.mu.Lock()
				c.buf.Write(buf[:n])
				c.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	return c
}

func (c *capture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

// WaitFor 等待 want 出现在输出里，返回当时的全部输出。
func (c *capture) WaitFor(t *testing.T, want string) string {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for {
		out := c.String()
		if strings.Contains(out, want) {
			return out
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待输出 %q 超时，已收到 %q", want, truncate(out, 600))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitForLine 等待输出中出现一整行等于 want 的内容。
func (c *capture) waitForLine(t *testing.T, want string) string {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for {
		out := c.String()
		for _, line := range strings.Split(out, "\n") {
			if strings.TrimSpace(line) == want {
				return out
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待输出行 %q 超时，已收到 %q", want, truncate(out, 600))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitForInt 等待正则的第一个数字捕获组出现，返回该数字。
func (c *capture) waitForInt(t *testing.T, pattern string) int {
	t.Helper()
	re := regexp.MustCompile(pattern)
	deadline := time.Now().Add(waitTimeout)
	for {
		out := c.String()
		if m := re.FindStringSubmatch(out); m != nil {
			value, err := strconv.Atoi(m[1])
			if err != nil {
				t.Fatalf("解析 %q 的数字: %v", m[0], err)
			}
			return value
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待 %q 超时，已收到 %q", pattern, truncate(out, 600))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// run 写入一条命令并等待 want 出现。
// want 不得是命令文本的子串，否则会命中终端回显而不是命令输出。
func run(t *testing.T, term *Terminal, c *capture, cmd, want string) {
	t.Helper()
	if strings.Contains(cmd, want) {
		t.Fatalf("测试标记 %q 出现在命令文本里，无法区分回显与输出", want)
	}
	if _, err := term.Write([]byte(cmd + "\r")); err != nil {
		t.Fatalf("写入命令 %q: %v", cmd, err)
	}
	c.WaitFor(t, want)
}

// waitStatus 等待 shell 退出，超时即失败。
func waitStatus(t *testing.T, term *Terminal) ExitStatus {
	t.Helper()
	ch := make(chan ExitStatus, 1)
	go func() { ch <- term.Wait() }()
	select {
	case status := <-ch:
		return status
	case <-time.After(waitTimeout):
		t.Fatalf("等待 shell 退出超时")
		return ExitStatus{}
	}
}

// aliveProcess 报告进程是否仍在运行；僵尸进程不算存活。
func aliveProcess(t *testing.T, pid int) bool {
	t.Helper()
	out, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		// ps 找不到该进程即已退出。
		return false
	}
	state := strings.TrimSpace(string(out))
	return state != "" && !strings.HasPrefix(state, "Z")
}

// containsPID 报告进程号是否在列表里。
func containsPID(pids []int, want int) bool {
	for _, pid := range pids {
		if pid == want {
			return true
		}
	}
	return false
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}

// TestPwdEqualsSessionDir 断言本机 shell 的 pwd 等于会话工作目录。
func TestPwdEqualsSessionDir(t *testing.T) {
	base := t.TempDir()
	cases := []struct{ name, sub string }{
		{name: "普通目录", sub: "work"},
		{name: "含空格与中文", sub: "会话 目录"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(base, tc.sub)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("建会话目录: %v", err)
			}
			term := startTerm(t, "sess-pwd", dir, Options{})
			c := startCapture(t, term)

			// shell 的 pwd 报物理路径，终端登记的工作目录也是解析符号链接后的路径。
			real, err := filepath.EvalSymlinks(dir)
			if err != nil {
				t.Fatalf("解析会话目录: %v", err)
			}
			if term.Dir() != real {
				t.Fatalf("终端工作目录 = %q，期望 %q", term.Dir(), real)
			}

			if _, err := term.Write([]byte("pwd\r")); err != nil {
				t.Fatalf("写入 pwd: %v", err)
			}
			c.waitForLine(t, real)
		})
	}
}

// TestStopReapsProcessGroupTree 断言停止后整个终端会话都没有残留，
// 包括自成进程组的后台作业与已脱离父进程的后台作业。
func TestStopReapsProcessGroupTree(t *testing.T) {
	dir := t.TempDir()
	term := startTerm(t, "sess-stop", dir, Options{})
	c := startCapture(t, term)

	// 一个后台作业（自成一进程组）、一个脱离父 shell 的后台作业，以及一个前台作业。
	cmd := "sleep 300 & echo BG=$!; ( sleep 300 & echo SUB=$! ); sleep 300"
	if _, err := term.Write([]byte(cmd + "\r")); err != nil {
		t.Fatalf("写入命令: %v", err)
	}
	bg := c.waitForInt(t, `BG=(\d+)`)
	sub := c.waitForInt(t, `SUB=(\d+)`)

	for _, pid := range []int{bg, sub} {
		if !aliveProcess(t, pid) {
			t.Fatalf("进程 %d 没有按预期启动", pid)
		}
	}

	// shell 还活着时，快照里应能找到这两个进程与 shell 自己，说明清扫目标是对的。
	if term.TTY() == "" {
		t.Fatalf("没有取得 shell 的控制终端名")
	}
	snapshot, err := ttyMembers(term.TTY())
	if err != nil {
		t.Fatalf("枚举终端进程: %v", err)
	}
	for _, want := range []int{term.PID(), bg, sub} {
		if !containsPID(snapshot, want) {
			t.Fatalf("控制终端快照 %v 里缺少进程 %d", snapshot, want)
		}
	}

	if err := term.Stop(); err != nil {
		t.Fatalf("停止终端: %v", err)
	}
	status := waitStatus(t, term)
	if !status.Signaled {
		t.Logf("shell 退出状态为 %s（非信号终止也算退出）", status)
	}

	for _, pid := range []int{bg, sub} {
		if aliveProcess(t, pid) {
			t.Fatalf("停止后进程 %d 仍然存活", pid)
		}
	}
}

// TestResizeWindowSize 断言运行中变更行列数生效且不崩。
func TestResizeWindowSize(t *testing.T) {
	dir := t.TempDir()
	term := startTerm(t, "sess-size", dir, Options{Cols: 100, Rows: 30})
	c := startCapture(t, term)

	run(t, term, c, "stty size", "30 100")

	if cols, rows := term.Size(); cols != 100 || rows != 30 {
		t.Fatalf("初始尺寸 = %dx%d，期望 100x30", cols, rows)
	}

	// 连续变更到极小与极大，确认既不崩也能继续工作。
	for _, size := range []struct {
		cols, rows uint16
	}{{120, 40}, {20, 5}, {400, 200}} {
		if err := term.Resize(size.cols, size.rows); err != nil {
			t.Fatalf("变更尺寸到 %dx%d: %v", size.cols, size.rows, err)
		}
	}
	if err := term.Resize(132, 43); err != nil {
		t.Fatalf("变更尺寸: %v", err)
	}
	if cols, rows := term.Size(); cols != 132 || rows != 43 {
		t.Fatalf("变更后尺寸 = %dx%d，期望 132x43", cols, rows)
	}
	run(t, term, c, "stty size", "43 132")

	// 尺寸变更后 shell 仍可用。
	run(t, term, c, "printf 'AL%s\\n' IVE", "ALIVE")

	if err := term.Resize(0, 24); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("零列数应返回 ErrInvalidSize，实际 %v", err)
	}
	if err := term.Resize(80, 0); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("零行数应返回 ErrInvalidSize，实际 %v", err)
	}
}

// TestUTF8MultibyteRoundTrip 断言多字节输入按原始字节双向传递：命令行输出与
// 交给交互程序的输入都不出现替换字符，且字节序列一致。
func TestUTF8MultibyteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	term := startTerm(t, "sess-utf8", dir, Options{})
	c := startCapture(t, term)

	// 输出方向：命令里写的是 %s，输出里才是实际字符，避免命中回显。
	run(t, term, c, "printf '中文-%s\\n' OK", "中文-OK")

	// 输入方向：交互程序按原始字节收到的内容再原样写出。
	if _, err := term.Write([]byte("head -c 6; printf '\\nE%sD\\n' N\r")); err != nil {
		t.Fatalf("写入命令: %v", err)
	}
	c.WaitFor(t, "head -c 6")
	if _, err := term.Write([]byte("汉字")); err != nil {
		t.Fatalf("写入多字节输入: %v", err)
	}
	if _, err := term.Write([]byte("\r")); err != nil {
		t.Fatalf("写入回车: %v", err)
	}
	out := c.WaitFor(t, "END")

	if got := strings.Count(out, "汉字"); got < 2 {
		t.Fatalf("多字节输入只出现 %d 次（应至少包括回显与程序输出各一次），输出 %q", got, truncate(out, 600))
	}
	if strings.Contains(out, "\uFFFD") {
		t.Fatalf("输出出现替换字符，说明字节被改写: %q", truncate(out, 600))
	}
}

// TestExitStatusForExitCommand 断言能取到 shell 的退出码。
func TestExitStatusForExitCommand(t *testing.T) {
	dir := t.TempDir()
	term := startTerm(t, "sess-exit", dir, Options{})
	c := startCapture(t, term)
	run(t, term, c, "printf 'R%s\\n' EADY", "READY")

	if _, err := term.Write([]byte("exit 7\r")); err != nil {
		t.Fatalf("写入 exit: %v", err)
	}
	status := waitStatus(t, term)
	if status.Signaled {
		t.Fatalf("期望正常退出，实际 %s", status)
	}
	if status.Code != 7 {
		t.Fatalf("退出码 = %d，期望 7", status.Code)
	}
	if term.Alive() {
		t.Fatalf("shell 已退出但终端仍报告存活")
	}
	// 自然退出后仍可停止，作为收尾。
	if err := term.Stop(); err != nil {
		t.Fatalf("停止已退出的终端: %v", err)
	}
}

// TestStopUnblocksAndIsIdempotent 断言停止后读写与尺寸变更都有明确结果，且 Stop 可重复调用。
func TestStopUnblocksAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	term := startTerm(t, "sess-stop-api", dir, Options{})
	c := startCapture(t, term)
	run(t, term, c, "printf 'R%s\\n' EADY", "READY")

	if err := term.Stop(); err != nil {
		t.Fatalf("停止终端: %v", err)
	}
	waitStatus(t, term)

	if _, err := term.Write([]byte("echo x\r")); !errors.Is(err, ErrTerminalClosed) {
		t.Fatalf("停止后写入应返回 ErrTerminalClosed，实际 %v", err)
	}
	if err := term.Resize(80, 24); !errors.Is(err, ErrTerminalClosed) {
		t.Fatalf("停止后变更尺寸应返回 ErrTerminalClosed，实际 %v", err)
	}

	var wg sync.WaitGroup
	errs := make([]error, 3)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = term.Stop()
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("第 %d 次重复停止返回错误: %v", i, err)
		}
	}
}

// TestRegistryRejectsDuplicateStart 断言一个会话只有一份终端：重复启动被明确拒绝，
// 上一实例退出后可以重新启动。
func TestRegistryRejectsDuplicateStart(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry()
	t.Cleanup(func() {
		if err := r.StopAll(); err != nil {
			t.Errorf("停止全部终端: %v", err)
		}
	})

	first, err := r.Start("s1", dir, Options{Env: testEnv(t)})
	if err != nil {
		t.Fatalf("首次启动: %v", err)
	}
	if _, err := r.Start("s1", dir, Options{Env: testEnv(t)}); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("重复启动应返回 ErrAlreadyRunning，实际 %v", err)
	}
	if _, err := r.Start("s2", dir, Options{Env: testEnv(t)}); err != nil {
		t.Fatalf("另一个会话启动: %v", err)
	}
	if n := r.Len(); n != 2 {
		t.Fatalf("登记表实例数 = %d，期望 2", n)
	}

	if err := r.Stop("s1"); err != nil {
		t.Fatalf("停止 s1: %v", err)
	}
	if got, ok := r.Get("s1"); !ok || got != first {
		t.Fatalf("停止后应能取回原实例以读取退出状态")
	}
	if got, ok := r.Get("s1"); ok && got.Alive() {
		t.Fatalf("停止后实例仍报告存活")
	}

	// 上一实例已退出，允许重新启动。
	again, err := r.Start("s1", dir, Options{Env: testEnv(t)})
	if err != nil {
		t.Fatalf("实例退出后重启: %v", err)
	}
	if again == first {
		t.Fatalf("重启应得到新实例")
	}
	if err := r.Stop("missing"); !errors.Is(err, ErrNoTerminal) {
		t.Fatalf("未知会话停止应返回 ErrNoTerminal，实际 %v", err)
	}
	if err := r.Forget("s2"); err != nil {
		t.Fatalf("移除 s2: %v", err)
	}
	if n := r.Len(); n != 1 {
		t.Fatalf("移除后登记表实例数 = %d，期望 1", n)
	}
}

// TestStartRejectsInvalidInput 断言失败前置条件在启动前就被拒绝。
func TestStartRejectsInvalidInput(t *testing.T) {
	base := t.TempDir()
	file := filepath.Join(base, "a-file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("建文件: %v", err)
	}

	cases := []struct {
		name      string
		sessionID string
		dir       string
		opts      Options
		wantErr   error
	}{
		{name: "空会话标识", sessionID: "", dir: base, wantErr: ErrEmptySessionID},
		{name: "空工作目录", sessionID: "s", dir: "", wantErr: ErrEmptyWorkingDir},
		{name: "工作目录不存在", sessionID: "s", dir: filepath.Join(base, "缺失"), wantErr: ErrWorkingDirNotExist},
		{name: "工作目录是文件", sessionID: "s", dir: file, wantErr: ErrWorkingDirNotDir},
		{name: "shell 不存在", sessionID: "s", dir: base, opts: Options{Shell: filepath.Join(base, "no-shell")}, wantErr: ErrShellNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.opts.Env = testEnv(t)
			if _, err := Start(tc.sessionID, tc.dir, tc.opts); !errors.Is(err, tc.wantErr) {
				t.Fatalf("错误 = %v，期望 %v", err, tc.wantErr)
			}
		})
	}
}

// TestShellArgsArePassed 断言参数原样传给 shell。
func TestShellArgsArePassed(t *testing.T) {
	dir := t.TempDir()
	term := startTerm(t, "sess-args", dir, Options{
		Shell:     "/bin/sh",
		ShellArgs: []string{"-c", "printf 'ARG%s\\n' S-OK; sleep 0.3"},
	})
	c := startCapture(t, term)
	c.WaitFor(t, "ARGS-OK")

	status := waitStatus(t, term)
	if status.Code != 0 {
		t.Fatalf("退出码 = %d，期望 0", status.Code)
	}
	if got := term.Args(); len(got) != 2 {
		t.Fatalf("参数 = %v，期望两个", got)
	}
}

// TestDefaultTermIsInjected 断言 TERM 缺失时补上缺省值，伪终端里的程序能看到。
func TestDefaultTermIsInjected(t *testing.T) {
	dir := t.TempDir()
	term := startTerm(t, "sess-term", dir, Options{})
	c := startCapture(t, term)
	run(t, term, c, "printf 'T%s\\n' \"$TERM\"", DefaultTerm)
}

// TestResolveShell 覆盖 shell 解析顺序。
func TestResolveShell(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "myshell")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("建 shell: %v", err)
	}
	notExecutable := filepath.Join(dir, "not-exec")
	if err := os.WriteFile(notExecutable, []byte("x"), 0o644); err != nil {
		t.Fatalf("建文件: %v", err)
	}

	cases := []struct {
		name     string
		explicit string
		env      map[string]string
		fallback string
		want     string
		wantErr  bool
	}{
		{name: "显式优先于环境", explicit: executable, env: map[string]string{"SHELL": "/bin/sh"}, fallback: "/bin/sh", want: executable},
		{name: "用环境变量 SHELL", env: map[string]string{"SHELL": "/bin/sh"}, fallback: "/bin/zsh", want: "/bin/sh"},
		{name: "回退到平台缺省", env: map[string]string{}, fallback: "/bin/sh", want: "/bin/sh"},
		{name: "显式不存在", explicit: filepath.Join(dir, "no-shell"), fallback: "/bin/sh", wantErr: true},
		{name: "不可执行", explicit: notExecutable, fallback: "/bin/sh", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string { return tc.env[key] }
			got, err := resolveShell(tc.explicit, getenv, tc.fallback)
			if tc.wantErr {
				if !errors.Is(err, ErrShellNotFound) {
					t.Fatalf("错误 = %v，期望 ErrShellNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("解析 shell: %v", err)
			}
			if got != tc.want {
				t.Fatalf("shell = %q，期望 %q", got, tc.want)
			}
		})
	}
}

// TestBuildEnv 覆盖 TERM 注入规则。
func TestBuildEnv(t *testing.T) {
	contains := func(env []string, key, value string) bool {
		for _, kv := range env {
			if kv == key+"="+value {
				return true
			}
		}
		return false
	}
	hasKey := func(env []string, key string) bool {
		for _, kv := range env {
			if strings.HasPrefix(kv, key+"=") {
				return true
			}
		}
		return false
	}

	t.Run("继承当前环境", func(t *testing.T) {
		env := buildEnv(nil, DefaultTerm)
		if len(env) == 0 {
			t.Fatalf("期望继承当前进程环境")
		}
	})
	t.Run("已有 TERM 时保留", func(t *testing.T) {
		env := buildEnv([]string{"TERM=vt100"}, DefaultTerm)
		if !contains(env, "TERM", "vt100") || contains(env, "TERM", DefaultTerm) {
			t.Fatalf("环境 = %v，期望保留调用方的 TERM", env)
		}
	})
	t.Run("缺失时补缺省 TERM", func(t *testing.T) {
		env := buildEnv([]string{"A=1"}, DefaultTerm)
		if !contains(env, "TERM", DefaultTerm) {
			t.Fatalf("环境 = %v，期望补上 TERM", env)
		}
	})
	t.Run("Term 为空时不补", func(t *testing.T) {
		env := buildEnv([]string{"A=1"}, "")
		if hasKey(env, "TERM") {
			t.Fatalf("环境 = %v，不应出现 TERM", env)
		}
	})
}

// TestExitStatusString 覆盖退出状态的可读输出。
func TestExitStatusString(t *testing.T) {
	if got := (ExitStatus{Code: 3}).String(); !strings.Contains(got, "3") {
		t.Fatalf("退出码文本 = %q", got)
	}
	if got := (ExitStatus{Code: -1, Signal: int(syscall.SIGKILL), Signaled: true}).String(); !strings.Contains(got, strconv.Itoa(int(syscall.SIGKILL))) {
		t.Fatalf("信号文本 = %q", got)
	}
}
