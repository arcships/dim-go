package ui

import (
	"context"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 右栏终端的 pty 接线：输出、输入、启动与停止都走 Source / Intents，不留在
// 视图里自说自话。真实 pty 的读写在 internal/uiwire 与 internal/terminal 验，
// 这里用一个注入源证明界面确实按事实与写入口办事。

// terminalIntents 记录三个终端动作；其余方法留在嵌入的 nil 接口上，被调到会
// panic，说明测试点到了一个没接的动作。
type terminalIntents struct {
	Intents
	started chan struct{}
	stopped chan struct{}
	written chan string
}

func (t *terminalIntents) StartTerminal(context.Context) error {
	t.started <- struct{}{}
	return nil
}

func (t *terminalIntents) StopTerminal(context.Context) error {
	t.stopped <- struct{}{}
	return nil
}

func (t *terminalIntents) WriteTerminal(_ context.Context, data string) error {
	t.written <- data
	return nil
}

// SetPreferences 吞掉视图状态的落盘：打开右栏 / 切页签会触发一次，与终端无关。
func (t *terminalIntents) SetPreferences(context.Context, PreferencesSnapshot) error { return nil }

// OpenSession 吞掉自动打开会话：右栏打开时视图会把选中的会话报给集成层。
func (t *terminalIntents) OpenSession(context.Context, string) error { return nil }

// newTerminalIntents 建一份写入口记录（缓冲开够，测试里不阻塞）。
func newTerminalIntents() *terminalIntents {
	return &terminalIntents{
		started: make(chan struct{}, 4),
		stopped: make(chan struct{}, 4),
		written: make(chan string, 8),
	}
}

// liveTerminalSource 造一个连着事实、有会话工作目录的源。
func liveTerminalSource(cwd string, terminal TerminalSnapshot) *fakeSource {
	src := wiredSource()
	src.session = SessionSnapshot{ID: "s1", Title: "真实会话", Cwd: cwd}
	src.terminal = terminal
	return src
}

// terminalTester 打开右栏终端页。
func terminalTester(t *testing.T, src *fakeSource, intents Intents) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	app.rightOpen, app.rightTab = true, tabTerminal
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// waitIntent 等一次写入口调用。
func waitIntent[T any](t *testing.T, ch chan T, what string) T {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(2 * time.Second):
		t.Fatalf("%s 没有走到写入口", what)
		var zero T
		return zero
	}
}

// TestTerminalLiveShowsSnapshotLines 断言画布画的是 pty 的输出行，不是样例。
func TestTerminalLiveShowsSnapshotLines(t *testing.T) {
	src := liveTerminalSource("/work/project", TerminalSnapshot{
		Active: true, Running: true,
		Lines: []string{"$ echo ok", "ok", "/work/project"},
	})
	app, tt := terminalTester(t, src, newTerminalIntents())

	for _, want := range []string{"$ echo ok", "ok", "/work/project"} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("画布没有画 pty 的输出行 %q：%v", want, tt.Texts())
		}
	}
	// 样例里的命令与目录名一个都不该出现。
	for _, sample := range []string{"npm run build", "go test ./internal/ui", "ui-preview.html"} {
		if _, ok := tt.Find(sample); ok {
			t.Fatalf("连着 pty 时仍画了样例输出 %q", sample)
		}
	}
	// 提示行的目录取会话工作目录的末段。
	if got := app.termTitle(); got != "project" {
		t.Fatalf("终端标题 = %q，想要会话工作目录的末段 project", got)
	}
	if _, ok := tt.Access("终端 · 运行中"); !ok {
		t.Fatalf("工具条没有按 pty 状态标运行中：%v", tt.Texts())
	}
}

// TestTerminalLiveStartAndStop 断言「新建终端」与「关闭进程」走写入口。
func TestTerminalLiveStartAndStop(t *testing.T) {
	intents := newTerminalIntents()
	src := liveTerminalSource("/work/project", TerminalSnapshot{})
	_, tt := terminalTester(t, src, intents)

	if err := tt.Click("新建终端"); err != nil {
		t.Fatalf("点不到新建终端：%v", err)
	}
	waitIntent(t, intents.started, "新建终端")
	// 启动之后快照里是「运行中」（真实适配器从 pty 读到的就是这个）。
	src.terminal = TerminalSnapshot{Active: true, Running: true}
	tt.Frame()

	intents.stopped = make(chan struct{}, 4)
	if err := tt.Click("关闭进程"); err != nil {
		t.Fatalf("点不到关闭进程：%v", err)
	}
	waitIntent(t, intents.stopped, "关闭进程")
}

// TestTerminalLiveSubmitWritesInput 断言提示行里的内容按回车写回 pty（行尾补 CR）。
func TestTerminalLiveSubmitWritesInput(t *testing.T) {
	intents := newTerminalIntents()
	src := liveTerminalSource("/work/project", TerminalSnapshot{Active: true, Running: true})
	_, tt := terminalTester(t, src, intents)

	if err := tt.Click("终端输入"); err != nil {
		t.Fatalf("点不到终端输入：%v", err)
	}
	tt.Type("echo ok")
	tt.Frame()
	tt.Key(0, gui.KeyEnter)
	got := waitIntent(t, intents.written, "终端输入")
	if got != "echo ok\r" {
		t.Fatalf("写回 pty 的字节 = %q，想要 %q", got, "echo ok\r")
	}
}

// TestTerminalLiveClearKeepsBuffer 断言「清屏」只把可见基线移走：pty 的输出不
// 因为清屏而丢，之后新到的行照常显示。
func TestTerminalLiveClearKeepsBuffer(t *testing.T) {
	src := liveTerminalSource("/work/project", TerminalSnapshot{
		Active: true, Running: true,
		Lines: []string{"$ first", "first"},
	})
	app, tt := terminalTester(t, src, newTerminalIntents())

	if err := tt.Click("清屏"); err != nil {
		t.Fatalf("点不到清屏：%v", err)
	}
	tt.Frame()
	if len(app.termLines) != 0 {
		t.Fatalf("清屏后仍有 %d 行", len(app.termLines))
	}
	// 新输出到达后照常上屏。
	src.terminal.Lines = append(src.terminal.Lines, "$ second", "second")
	tt.Frame()
	if _, ok := tt.Find("second"); !ok {
		t.Fatalf("清屏后新到的输出没有上屏：%v", tt.Texts())
	}
	if _, ok := tt.Find("first"); ok {
		t.Fatal("清屏前的输出又回来了")
	}
}

// TestTerminalLiveNoCwdDisabled 断言没有工作目录时「新建终端」禁用并写原因，
// 画布给空态而不是样例。
func TestTerminalLiveNoCwdDisabled(t *testing.T) {
	src := liveTerminalSource("", TerminalSnapshot{})
	_, tt := terminalTester(t, src, newTerminalIntents())

	info, ok := tt.Access("新建终端")
	if !ok {
		t.Fatalf("工具条没有新建终端：%v", tt.Texts())
	}
	if !info.Disabled {
		t.Fatal("没有工作目录时「新建终端」仍可点")
	}
	if _, ok := tt.Find(termNoCwdText); !ok {
		t.Fatalf("没有工作目录时画布没有写原因 %q：%v", termNoCwdText, tt.Texts())
	}
	for _, sample := range []string{"$ go test ./internal/ui -run TestSender", "npm run build"} {
		if _, ok := tt.Find(sample); ok {
			t.Fatalf("没有工作目录时仍画了样例输出 %q", sample)
		}
	}
}

// TestTerminalLiveNoSessionDisabled 断言没有打开的会话时「新建终端」禁用并写原因。
func TestTerminalLiveNoSessionDisabled(t *testing.T) {
	src := liveTerminalSource("/work/project", TerminalSnapshot{})
	src.hasSess = false
	src.session = SessionSnapshot{}
	_, tt := terminalTester(t, src, newTerminalIntents())

	info, ok := tt.Access("新建终端")
	if !ok {
		t.Fatalf("工具条没有新建终端：%v", tt.Texts())
	}
	if !info.Disabled {
		t.Fatal("没有会话时「新建终端」仍可点")
	}
	if _, ok := tt.Find(noteNoSessionID); !ok {
		t.Fatalf("没有会话时画布没有写原因 %q：%v", noteNoSessionID, tt.Texts())
	}
}
