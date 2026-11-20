package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 终端页的「状态 × 元素 × 交互」表（docs/domains/U3-interaction.md §1.2）。
//
// 每一行是一个互斥状态，每个状态都分别断言三处：标签条（入口在不在、能不能点、
// 原因逐字）、画布（说明行 / 输出行 / 结束提示 / 关闭提示）、提示行（在不在），
// 外加右键菜单五项里哪些必须禁用。断言只读界面（Access / Texts / Menu），不读
// 实现里的中间变量——表说的就是用户能看见的东西。

// termWant 是一个状态在界面上的期望。
type termWant struct {
	// notices 是画布上必须逐字出现的说明行（空态、启动中、结束提示、关闭提示）。
	notices []string
	// lines 是画布上必须出现的输出行（退出后保留最后输出）。
	lines []string
	// absent 是画布与提示行上必须不出现的文案。
	absent []string
	// prompt 表示提示行（`→ 目录` + 输入框）在不在。
	prompt bool
	// disabled 是标签条入口 → 禁用原因；"" 表示该入口可点。
	disabled map[string]string
	// menuDisabled 是右键菜单里必须禁用的项。
	menuDisabled []string
	// paste 表示右键「粘贴」可点（点下去真的写进提示行）。
	paste bool
}

// 标签条与菜单里的入口名（断言与表 1.2.3–1.2.4 用同一套名字）。
const (
	termNewLabel    = "新建终端"
	termCloseLabel  = "关闭终端"
	termCloseProc   = "关闭进程"
	termClearLabel  = "清屏"
	termCopyLabel   = "复制输出"
	termInputLabel  = "终端输入"
	termMenuCopy    = "复制选中内容"
	termMenuPaste   = "粘贴"
	termMenuClear   = "清屏"
	termMenuNew     = "新建终端"
	termMenuClose   = "关闭进程"
	termMenuAllFive = 5
)

// termStateApp 打开右栏终端页，并渲染一帧。
func termStateApp(t *testing.T, src *fakeSource, intents Intents) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	app.rightOpen, app.rightTab = true, tabTerminal
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	return app, tt
}

// termEventual 等一个后台意图的结果落到视图上（失败行、停止完成等）。
func termEventual(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("%s：等不到", what)
}

// blockingStartIntents 让启动请求停在半路：用来把画面钉在「启动中」这一帧。
type blockingStartIntents struct {
	*terminalIntents
	release chan struct{}
}

func (b *blockingStartIntents) StartTerminal(context.Context) error {
	<-b.release
	return nil
}

// failingStopIntents 已经在本包里（panel_gap_test.go）：停止报错、页签保留。

// TestTerminalStateTable 是 §1.2.3 的主表：每个状态至少断言标签条、画布、
// 提示行三处，外加 §1.2.4 的右键菜单。
func TestTerminalStateTable(t *testing.T) {
	cases := []struct {
		name  string
		build func(t *testing.T) (*App, *gui.Tester)
		want  termWant
	}{
		{
			name: "1 没有会话",
			build: func(t *testing.T) (*App, *gui.Tester) {
				src := liveTerminalSource("", TerminalSnapshot{})
				src.hasSess, src.session = false, SessionSnapshot{}
				return termStateApp(t, src, newTerminalIntents())
			},
			want: termWant{
				notices: []string{noteNoSessionID},
				absent:  []string{termNoCwdText, termNoEntryText, termEmptyText, termNotStartedText},
				prompt:  false,
				disabled: map[string]string{
					termNewLabel:   noteNoSessionID,
					termCloseLabel: "",
					termCloseProc:  termNoProcessNote,
					termClearLabel: termEmptyText,
					termCopyLabel:  termEmptyText,
				},
				menuDisabled: []string{termMenuCopy, termMenuPaste, termMenuClear, termMenuNew, termMenuClose},
			},
		},
		{
			name: "2 会话没有工作目录",
			build: func(t *testing.T) (*App, *gui.Tester) {
				return termStateApp(t, liveTerminalSource("", TerminalSnapshot{}), newTerminalIntents())
			},
			want: termWant{
				notices: []string{termNoCwdText},
				absent:  []string{noteNoSessionID, termNoEntryText, termEmptyText, termNotStartedText},
				prompt:  false,
				disabled: map[string]string{
					termNewLabel:   termNoCwdText,
					termCloseLabel: "",
					termCloseProc:  termNoProcessNote,
					termClearLabel: termEmptyText,
					termCopyLabel:  termEmptyText,
				},
				menuDisabled: []string{termMenuCopy, termMenuPaste, termMenuClear, termMenuNew, termMenuClose},
			},
		},
		{
			// 没有写入口时，即便适配器手里还留着一份「运行中」的旧快照，也不画
			// 一个看起来能输入的终端（全局前置先于进程事实）。
			name: "3 没有终端写入口",
			build: func(t *testing.T) (*App, *gui.Tester) {
				src := liveTerminalSource("/work/project", TerminalSnapshot{Active: true, Running: true})
				return termStateApp(t, src, nil)
			},
			want: termWant{
				notices: []string{termNoEntryText},
				absent:  []string{noteNoSessionID, termNoCwdText, termEmptyText, termNotStartedText, "$ echo ok"},
				prompt:  false,
				disabled: map[string]string{
					termNewLabel:   termNoEntryText,
					termCloseLabel: "",
					termCloseProc:  termNoProcessNote,
					termClearLabel: termEmptyText,
					termCopyLabel:  termEmptyText,
				},
				menuDisabled: []string{termMenuCopy, termMenuPaste, termMenuClear, termMenuNew, termMenuClose},
			},
		},
		{
			name: "4 还没启动",
			build: func(t *testing.T) (*App, *gui.Tester) {
				return termStateApp(t, liveTerminalSource("/work/project", TerminalSnapshot{}), newTerminalIntents())
			},
			want: termWant{
				notices: []string{termNotStartedText},
				absent:  []string{noteNoSessionID, termNoCwdText, termNoEntryText, termEmptyText},
				prompt:  false,
				disabled: map[string]string{
					termNewLabel:   "",
					termCloseLabel: "",
					termCloseProc:  termNoProcessNote,
					termClearLabel: termEmptyText,
					termCopyLabel:  termEmptyText,
				},
				menuDisabled: []string{termMenuCopy, termMenuPaste, termMenuClear, termMenuClose},
			},
		},
		{
			name: "5 启动中",
			build: func(t *testing.T) (*App, *gui.Tester) {
				intents := &blockingStartIntents{
					terminalIntents: newTerminalIntents(),
					release:         make(chan struct{}),
				}
				app, tt := termStateApp(t, liveTerminalSource("/work/project", TerminalSnapshot{}), intents)
				if err := tt.Click(termNewLabel); err != nil {
					t.Fatalf("点不到新建终端：%v", err)
				}
				tt.Frame()
				t.Cleanup(func() {
					close(intents.release)
					termEventual(t, "启动请求收尾", func() bool {
						tt.Frame()
						return !app.termStarting
					})
				})
				return app, tt
			},
			want: termWant{
				notices: []string{termStartingText},
				absent:  []string{termNotStartedText, termEmptyText},
				prompt:  false,
				disabled: map[string]string{
					termNewLabel:   termStartingNote,
					termCloseLabel: termStartingCloseNote,
					termCloseProc:  termStartingNote,
					termClearLabel: termEmptyText,
					termCopyLabel:  termEmptyText,
				},
				menuDisabled: []string{termMenuCopy, termMenuPaste, termMenuClear, termMenuNew, termMenuClose},
			},
		},
		{
			name: "6 运行中",
			build: func(t *testing.T) (*App, *gui.Tester) {
				src := liveTerminalSource("/work/project", TerminalSnapshot{
					Active: true, Running: true,
					Lines: []string{"$ echo ok", "ok"},
				})
				return termStateApp(t, src, newTerminalIntents())
			},
			want: termWant{
				notices: []string{},
				lines:   []string{"$ echo ok", "ok"},
				absent:  []string{termEmptyText, termNotStartedText, termNoEntryText, termNoCwdText, noteNoSessionID, termClosedNotice, termReadFailedText},
				prompt:  true,
				disabled: map[string]string{
					termNewLabel:   termRunningNote,
					termCloseLabel: "",
					termCloseProc:  "",
					termClearLabel: "",
					termCopyLabel:  "",
				},
				menuDisabled: []string{termMenuCopy, termMenuNew},
				paste:        true,
			},
		},
		{
			name: "7 进程自行退出",
			build: func(t *testing.T) (*App, *gui.Tester) {
				src := liveTerminalSource("/work/project", TerminalSnapshot{
					Active: true, Exited: true, ExitKnown: true, ExitCode: 3,
					Lines: []string{"$ exit 3", "bye"},
				})
				return termStateApp(t, src, newTerminalIntents())
			},
			want: termWant{
				notices: []string{"［进程已退出，退出码 3］"},
				lines:   []string{"$ exit 3", "bye"},
				absent:  []string{termClosedNotice, termReadFailedText, termNotStartedText},
				prompt:  false,
				disabled: map[string]string{
					termNewLabel:   "",
					termCloseLabel: "",
					termCloseProc:  termExitedCloseNote,
					termClearLabel: "",
					termCopyLabel:  "",
				},
				menuDisabled: []string{termMenuCopy, termMenuPaste},
			},
		},
		{
			name: "8 用户关闭了进程",
			build: func(t *testing.T) (*App, *gui.Tester) {
				intents := newTerminalIntents()
				src := liveTerminalSource("/work/project", TerminalSnapshot{
					Active: true, Running: true, Lines: []string{"$ echo ok", "ok"},
				})
				app, tt := termStateApp(t, src, intents)
				if err := tt.Click(termCloseProc); err != nil {
					t.Fatalf("点不到关闭进程：%v", err)
				}
				waitIntent(t, intents.stopped, "关闭进程")
				// 真实适配器停在成功后是「已结束 + 最后输出仍在缓冲里」。
				src.terminal = TerminalSnapshot{
					Active: true, Exited: true, ExitKnown: true, ExitCode: 0,
					Lines: []string{"$ echo ok", "ok"},
				}
				tt.Frame()
				return app, tt
			},
			want: termWant{
				notices: []string{termClosedNotice},
				absent:  []string{"［进程已退出，退出码 0］", termExitedNotice, termReadFailedText},
				prompt:  false,
				disabled: map[string]string{
					termNewLabel:   "",
					termCloseLabel: "",
					termCloseProc:  termClosedNote,
					termClearLabel: termEmptyText,
					termCopyLabel:  termEmptyText,
				},
				menuDisabled: []string{termMenuCopy, termMenuPaste, termMenuClear, termMenuClose},
			},
		},
		{
			// 9 与 6 逐格相同：停止报错不改变画面，只多一行全局错误；页签不消失。
			name: "9 关闭失败",
			build: func(t *testing.T) (*App, *gui.Tester) {
				intents := &failingStopIntents{terminalIntents: newTerminalIntents()}
				src := liveTerminalSource("/work/project", TerminalSnapshot{
					Active: true, Running: true, Lines: []string{"$ echo ok", "ok"},
				})
				app, tt := termStateApp(t, src, intents)
				if err := tt.Click(termCloseProc); err != nil {
					t.Fatalf("点不到关闭进程：%v", err)
				}
				termEventual(t, "停止失败的原因", func() bool { return app.intentError() != "" })
				tt.Frame()
				return app, tt
			},
			want: termWant{
				notices: []string{"关闭终端进程：进程没有响应"},
				lines:   []string{"$ echo ok", "ok"},
				absent:  []string{termClosedNotice, termReadFailedText},
				prompt:  true,
				disabled: map[string]string{
					termNewLabel:   termRunningNote,
					termCloseLabel: "",
					termCloseProc:  "",
					termClearLabel: "",
					termCopyLabel:  "",
				},
				menuDisabled: []string{termMenuCopy, termMenuNew},
				paste:        true,
			},
		},
		{
			name: "10 读不到输出",
			build: func(t *testing.T) (*App, *gui.Tester) {
				src := liveTerminalSource("/work/project", TerminalSnapshot{
					Active: true, Running: true, ReadFailed: true,
					Lines: []string{"$ make watch", "building…"},
				})
				return termStateApp(t, src, newTerminalIntents())
			},
			want: termWant{
				notices: []string{termReadFailedText},
				lines:   []string{"$ make watch", "building…"},
				absent:  []string{termClosedNotice, termExitedNotice, termNotStartedText},
				prompt:  false,
				disabled: map[string]string{
					termNewLabel:   termRunningNote,
					termCloseLabel: "",
					termCloseProc:  "",
					termClearLabel: "",
					termCopyLabel:  "",
				},
				menuDisabled: []string{termMenuCopy, termMenuPaste, termMenuNew},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, tt := tc.build(t)
			checkTermWant(t, app, tt, tc.want)
		})
	}
}

// TestTerminalLastTabClosed 覆盖状态 11：关掉最后一个页签后终端页整页收起，
// 右栏回文件页，不再画提示行（不在表内 = 整页不渲染）。
func TestTerminalLastTabClosed(t *testing.T) {
	intents := newTerminalIntents()
	src := liveTerminalSource("/work/project", TerminalSnapshot{Active: true, Running: true})
	app, tt := termStateApp(t, src, intents)

	if err := tt.Click(termCloseLabel); err != nil {
		t.Fatalf("点不到关闭终端：%v", err)
	}
	waitIntent(t, intents.stopped, "关闭终端")
	src.terminal = TerminalSnapshot{Active: true, Exited: true}
	tt.Frame()

	if app.rightTab != tabFiles {
		t.Fatalf("关掉最后一个页签后应回文件页：tab=%d", app.rightTab)
	}
	if _, ok := tt.Access(termInputLabel); ok {
		t.Fatal("终端页收起后仍画着提示行")
	}
	if tt.HasText(termClosedNotice) {
		t.Fatalf("整页收起后仍画着关闭提示：%v", tt.Texts())
	}
}

// TestTerminalStartFailureBackToNotStarted 覆盖表外的一条：启动失败回到 4，
// 原因进全局错误行，不留下半启动的画面；「新建终端」重新可点。
func TestTerminalStartFailureBackToNotStarted(t *testing.T) {
	intents := &failStartIntents{terminalIntents: newTerminalIntents()}
	src := liveTerminalSource("/work/project", TerminalSnapshot{})
	app, tt := termStateApp(t, src, intents)

	if err := tt.Click(termNewLabel); err != nil {
		t.Fatalf("点不到新建终端：%v", err)
	}
	termEventual(t, "启动失败的原因", func() bool { return app.intentError() != "" })
	tt.Frame()

	if app.termStarting {
		t.Fatal("启动失败后仍停在「启动中」")
	}
	if !tt.HasText(termNotStartedText) {
		t.Fatalf("启动失败后画布没有回到「还没启动」：%v", tt.Texts())
	}
	if _, ok := tt.Access(termInputLabel); ok {
		t.Fatal("启动失败后仍画着提示行（没有 pty 却看起来能输入）")
	}
	info, ok := tt.Access(termNewLabel)
	if !ok || info.Disabled {
		t.Fatalf("启动失败后「新建终端」仍不可点：%+v", info)
	}
}

// failStartIntents 只让启动失败，其余方法留在嵌入的实现上。
type failStartIntents struct{ *terminalIntents }

func (f *failStartIntents) StartTerminal(context.Context) error {
	return errors.New("没有可用的 shell")
}

// termButtonTip 是入口悬停提示的前半句：清屏还带快捷键，其余就是名字。
func termButtonTip(name string) string {
	if name == termClearLabel {
		return "清屏 · Cmd-K"
	}
	return name
}

// checkTermWant 按表断言一个状态的三处与菜单。
func checkTermWant(t *testing.T, app *App, tt *gui.Tester, want termWant) {
	t.Helper()

	// 标签条：页签在（读屏名 = 「终端标签：」+ 标题）。
	if _, ok := tt.Access("终端标签：" + app.termTitle()); !ok {
		t.Fatalf("标签条没有页签 %q：%v", "终端标签："+app.termTitle(), tt.Texts())
	}
	for name, reason := range want.disabled {
		info, ok := tt.Access(name)
		if !ok {
			t.Fatalf("标签条没有入口 %q：%v", name, tt.Texts())
		}
		if reason == "" {
			if info.Disabled {
				t.Fatalf("「%s」应当可点，实际禁用（说明 %q）", name, info.Description)
			}
			continue
		}
		if !info.Disabled {
			t.Fatalf("「%s」应当禁用，实际可点", name)
		}
		if wantTip := termTip(termButtonTip(name), reason); info.Description != wantTip {
			t.Fatalf("「%s」的悬停提示 = %q，想要 %q", name, info.Description, wantTip)
		}
	}

	// 画布：说明行、输出行、必须不出现的文案。
	for _, text := range want.notices {
		if !tt.HasText(text) {
			t.Fatalf("画布没有说明行 %q：%v", text, tt.Texts())
		}
	}
	for _, text := range want.lines {
		if !tt.HasText(text) {
			t.Fatalf("画布没有输出行 %q：%v", text, tt.Texts())
		}
	}
	for _, text := range want.absent {
		if tt.HasText(text) {
			t.Fatalf("画布/提示行不该出现 %q：%v", text, tt.Texts())
		}
	}

	// 提示行：只在运行中（及关闭失败的运行中）出现。
	if _, ok := tt.Access(termInputLabel); ok != want.prompt {
		t.Fatalf("提示行在场 = %v，想要 %v：%v", ok, want.prompt, tt.Texts())
	}

	// 右键菜单：五项齐全；表里说禁用的项必须点不动（点它只会得到 disabled）。
	tt.RightClickAt(WindowWidth-100, WindowHeight/2)
	items := tt.Menu()
	wantItems := []string{termMenuCopy, termMenuPaste, "-", termMenuClear, termMenuNew, "-", termMenuClose}
	if strings.Join(items, "|") != strings.Join(wantItems, "|") {
		t.Fatalf("右键菜单 = %v，想要 %v", items, wantItems)
	}
	for _, label := range want.menuDisabled {
		err := tt.ChooseMenuItem(label)
		if err == nil || !strings.Contains(err.Error(), "disabled") {
			t.Fatalf("菜单项「%s」应当禁用，ChooseMenuItem = %v", label, err)
		}
	}
	tt.CloseMenu()

	// 粘贴只有在运行中才可点：点下去真的写进提示行。
	if want.paste {
		tt.SetClipboard("echo hi")
		tt.RightClickAt(WindowWidth-100, WindowHeight/2)
		if err := tt.ChooseMenuItem(termMenuPaste); err != nil {
			t.Fatalf("运行中「粘贴」应当可点：%v", err)
		}
		if app.termInput != "echo hi" {
			t.Fatalf("粘贴后提示行 = %q，想要 %q", app.termInput, "echo hi")
		}
	}
}
