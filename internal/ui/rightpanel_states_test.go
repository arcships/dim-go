package ui

import (
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 右栏七面板的状态表测试：面板 × 状态各至少一条断言（在场 / 不在场 / 禁用 /
// 文案）。表在 docs/domains/U3-interaction.md §13，这里是它的可执行副本。
//
// 三条硬约束逐格核对：
//   - 空态不等于错误态：读到空给空态，读不到给错误行 + 重试，两种文案不共用；
//   - 读取中不等于空：首读画骨架，读完才决定空还是有内容；
//   - 断线时本机面板仍可用：文件 / 改动照常读本机磁盘与 git，只是横幅说明连接。
//
// 终端由终端线负责，这里只登记接口（§13.5）：断线仍能画 pty 输出、页内不画
// 连接横幅。

// stateCase 是状态表里的一格。
type stateCase struct {
	name string
	tab  int
	// src 为 nil 表示 -demo（没有 Source，只有样例）。
	src func() *fakePanelSource
	// open 摆上额外的视图状态（筛选词、子标签、侧聊草稿等）。
	open func(*App)
	// want 是必须在场的文字（逐字标签或读屏名）。
	want []string
	// absent 是不该在场的文字。
	absent []string
	// disabled / enabled 是按读屏名核对的入口可用性。
	disabled []string
	enabled  []string
}

// runStateCases 跑一组状态格。
func runStateCases(t *testing.T, cases []stateCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var app *App
			if tc.src != nil {
				app = NewApp(Options{State: StateDefault, Source: tc.src()})
			} else {
				app = NewApp(Options{State: StateDefault})
			}
			app.rightOpen, app.rightTab = true, tc.tab
			if tc.open != nil {
				tc.open(app)
			}
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			tt.Frame()
			for _, text := range tc.want {
				if !showsText(tt, text) {
					t.Fatalf("缺少 %q\n屏上文字：%v", text, tt.Texts())
				}
			}
			for _, text := range tc.absent {
				if showsText(tt, text) {
					t.Fatalf("不该出现 %q\n屏上文字：%v", text, tt.Texts())
				}
			}
			for _, name := range tc.disabled {
				info, ok := tt.Access(name)
				if !ok {
					t.Fatalf("找不到入口 %q（要核对禁用态）", name)
				}
				if !info.Disabled {
					t.Fatalf("入口 %q 应当置灰：%+v", name, info)
				}
				if strings.TrimSpace(info.Description) == "" {
					t.Fatalf("入口 %q 置灰却没有写原因", name)
				}
			}
			for _, name := range tc.enabled {
				info, ok := tt.Access(name)
				if !ok {
					t.Fatalf("找不到入口 %q（要核对可用态）", name)
				}
				if info.Disabled {
					t.Fatalf("入口 %q 不该置灰：%+v", name, info)
				}
			}
		})
	}
}

// showsText 是 Find（整名相等）与 HasText（包含）的合取：表里的格子既可能是
// 一个元素的读屏名，也可能是一句更长的文案。
func showsText(tt *gui.Tester, text string) bool {
	if _, ok := tt.Find(text); ok {
		return true
	}
	return tt.HasText(text)
}

// wiredNoSession 是没有打开会话的源（连接在线）。
func wiredNoSession() *fakePanelSource {
	src := newFakePanelSource("/work")
	src.hasSess = false
	src.session = SessionSnapshot{}
	return src
}

// wiredDisconnected 是断线但仍有最后一份会话快照的源。
func wiredDisconnected(cwd string) *fakePanelSource {
	src := newFakePanelSource(cwd)
	src.conn = Connection{Phase: "disconnected", Reason: "连接被网络中断"}
	return src
}

// ---- 13.1 文件 ----

func TestFilesPanelStateTable(t *testing.T) {
	runStateCases(t, []stateCase{
		{
			name:     "未打开会话",
			tab:      tabFiles,
			src:      wiredNoSession,
			want:     []string{"未打开会话", "打开一个会话后，这里显示它的工作目录。"},
			absent:   []string{rpEmptyDirTitle, "正在加载文件树"},
			disabled: []string{"筛选文件"},
		},
		{
			name: "会话没有工作目录",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("")
				return src
			},
			want:     []string{"没有工作区", "这个会话没有工作目录，文件与改动都读不到。"},
			disabled: []string{"筛选文件"},
		},
		{
			name: "首次读取中",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = FileTreeSnapshot{Phase: PanelReadLoading}
				return src
			},
			want:   []string{"正在加载文件树"},
			absent: []string{rpEmptyDirTitle, "没有未提交的改动"},
		},
		{
			name: "读取失败给错误行与重试",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = FileTreeSnapshot{Phase: PanelReadFailed, Error: "读取目录: permission denied"}
				return src
			},
			want:   []string{"无法加载文件树。", "读取目录: permission denied", "重试读取文件树"},
			absent: []string{rpEmptyDirTitle, "无匹配文件"},
		},
		{
			name: "读到空目录是空态不是错误",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = readyTree()
				return src
			},
			want:   []string{rpEmptyDirTitle, rpEmptyDirDetail},
			absent: []string{"无法加载文件树。", "无匹配文件"},
		},
		{
			name: "有内容",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = readyTree(
					FileEntrySnapshot{Name: "docs", Dir: true},
					FileEntrySnapshot{Name: "main.go"},
				)
				return src
			},
			want:   []string{"目录，docs，已收起", "文件，main.go"},
			absent: []string{rpEmptyDirTitle, "无法加载文件树。"},
		},
		{
			name: "搜索读取中",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = readyTree(FileEntrySnapshot{Name: "main.go"})
				src.search = FileSearchSnapshot{Phase: PanelReadLoading}
				return src
			},
			open:   func(a *App) { a.rp().filter = "main" },
			want:   []string{"正在搜索整个工作区…"},
			absent: []string{"无匹配文件"},
		},
		{
			name: "搜索失败给重试",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = readyTree(FileEntrySnapshot{Name: "main.go"})
				src.search = FileSearchSnapshot{Phase: PanelReadFailed, Error: "扫描失败"}
				return src
			},
			open:     func(a *App) { a.rp().filter = "main" },
			want:     []string{"文件搜索失败。", "扫描失败"},
			enabled:  []string{"重试搜索"},
			absent:   []string{"无匹配文件"},
			disabled: nil,
		},
		{
			name: "搜索部分失败给提示",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = readyTree(FileEntrySnapshot{Name: "main.go"})
				src.search = FileSearchSnapshot{Phase: PanelReadReady, Limit: 200, Denied: true,
					Hits: []FileHitSnapshot{{Path: "main.go", Line: 1}}}
				return src
			},
			open: func(a *App) { a.rp().filter = "main" },
			want: []string{"部分文件夹无法搜索，结果可能不完整。"},
		},
		{
			name: "搜索截断给上限与成因",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = readyTree(FileEntrySnapshot{Name: "main.go"})
				src.search = FileSearchSnapshot{Phase: PanelReadReady, Limit: rpSearchLimit,
					Truncated: true, StopReason: "max-files",
					Hits: []FileHitSnapshot{{Path: "main.go", Line: 1}}}
				return src
			},
			open: func(a *App) { a.rp().filter = "main" },
			want: []string{"仅显示前 200 个匹配项：已扫到文件数上限，结果可能不完整"},
		},
		{
			name: "搜索无命中是空态",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = readyTree(FileEntrySnapshot{Name: "main.go"})
				src.search = FileSearchSnapshot{Phase: PanelReadReady, Limit: 200}
				return src
			},
			open:   func(a *App) { a.rp().filter = "token" },
			want:   []string{"无匹配文件", "没有找到包含“token”的文件"},
			absent: []string{"文件搜索失败。", rpEmptyDirTitle},
		},
		{
			name: "断线仍读本机",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := wiredDisconnected("/work")
				src.trees[""] = readyTree(FileEntrySnapshot{Name: "main.go"})
				return src
			},
			want: []string{"连接已断开：连接被网络中断", "重新连接", "文件，main.go"},
			// 断线不等于没有工作目录：本机读数照常，不能退回「未打开会话」。
			absent: []string{"未打开会话", "没有工作区"},
		},
	})
}

// TestFilesRootRetryRereads 断言根级「重试」真的带刷新重读，不是只换文案。
func TestFilesRootRetryRereads(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = FileTreeSnapshot{Phase: PanelReadFailed, Error: "读取目录: permission denied"}
	_, tt := rightWiredTester(t, tabFiles, src)
	if err := tt.Click("重试读取文件树"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	if len(src.treeRefreshed) == 0 || src.treeRefreshed[len(src.treeRefreshed)-1] != "" {
		t.Fatalf("重试没有带刷新重读根目录：%v", src.treeRefreshed)
	}
}

// ---- 13.2 改动 ----

func TestChangesPanelStateTable(t *testing.T) {
	runStateCases(t, []stateCase{
		{
			name:     "未打开会话",
			tab:      tabChanges,
			src:      wiredNoSession,
			want:     []string{"未打开会话"},
			absent:   []string{"没有未提交的改动", "Git 改动不可用"},
			disabled: []string{"筛选文件", "刷新"},
		},
		{
			name: "会话没有工作目录",
			tab:  tabChanges,
			src: func() *fakePanelSource {
				return newFakePanelSource("")
			},
			want:     []string{"没有工作区", "这个会话没有工作目录，文件与改动都读不到。"},
			absent:   []string{"没有未提交的改动"},
			disabled: []string{"筛选文件"},
		},
		{
			name: "首次读取中",
			tab:  tabChanges,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.changes = ChangeSet{Phase: PanelReadLoading}
				return src
			},
			want:   []string{"正在加载文件树"},
			absent: []string{"没有未提交的改动", "Git 改动不可用"},
		},
		{
			name: "读取失败给错误行与重试",
			tab:  tabChanges,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.changes = ChangeSet{Phase: PanelReadFailed, Error: "git status 失败"}
				return src
			},
			want:   []string{"Git 改动不可用", "git status 失败", "重试读取改动"},
			absent: []string{"没有未提交的改动"},
		},
		{
			name: "明细不可用是能力缺口不是空态",
			tab:  tabChanges,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main",
					Dirty: true, ChangedFileCount: 2, Unavailable: "读不到按文件与暂存态的条目"}
				return src
			},
			want:   []string{"改动明细不可用", "读不到按文件与暂存态的条目；Git 读到 2 个改动文件。"},
			absent: []string{"没有未提交的改动", "Git 改动不可用"},
		},
		{
			name: "读到空是空态",
			tab:  tabChanges,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main"}
				return src
			},
			want:   []string{"没有未提交的改动", "工作区是干净的"},
			absent: []string{"Git 改动不可用", "改动明细不可用"},
		},
		{
			name: "有内容",
			tab:  tabChanges,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main",
					Dirty: true, ChangedFileCount: 1,
					Files: []ChangeFileSnapshot{{Path: "main.go", Status: "M", StatKnown: true,
						Added: 3, Removed: 1}}}
				return src
			},
			want: []string{"文件，main.go，新增 3 行，删除 1 行", "1 个文件改动", "+3", "-1"},
		},
		{
			name: "权限受限给出权限条",
			tab:  tabChanges,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main",
					Restricted: true, Unavailable: "读不到条目"}
				return src
			},
			want: []string{"没有写入权限，部分操作不可用", "重试读取文件树"},
		},
		{
			name: "断线仍读本机 git",
			tab:  tabChanges,
			src: func() *fakePanelSource {
				src := wiredDisconnected("/work")
				src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main"}
				return src
			},
			want:   []string{"连接已断开：连接被网络中断", "重新连接", "没有未提交的改动"},
			absent: []string{"未打开会话", "没有工作区"},
		},
	})
}

// TestChangesRetryRereads 断言改动页的「重试」真的带刷新重读。
func TestChangesRetryRereads(t *testing.T) {
	src := newFakePanelSource("/work")
	src.changes = ChangeSet{Phase: PanelReadFailed, Error: "git status 失败"}
	_, tt := rightWiredTester(t, tabChanges, src)
	before := src.changeCalls
	if err := tt.Click("重试读取改动"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	if src.changeCalls <= before {
		t.Fatalf("重试没有重读改动：前 %d 次，后 %d 次", before, src.changeCalls)
	}
	// 一帧里可能画多次：只看刷新请求出现过没有，别拿最后一帧的请求当结论。
	if !src.changeRefreshed {
		t.Fatal("重试没有带刷新请求")
	}
}

// ---- 13.3 后台任务 ----

func TestTasksPanelStateTable(t *testing.T) {
	runStateCases(t, []stateCase{
		{
			name:   "未打开会话",
			tab:    tabTasks,
			src:    wiredNoSession,
			want:   []string{"未打开会话", "打开一个会话后，这里显示它的后台进程。"},
			absent: []string{"暂无运行进程"},
		},
		{
			name:   "未打开会话（唤醒子标签）",
			tab:    tabTasks,
			src:    wiredNoSession,
			open:   func(a *App) { a.panels.taskKind = 1 },
			want:   []string{"未打开会话", "打开一个会话后，这里显示它的唤醒记录。"},
			absent: []string{"暂无待唤醒事项"},
		},
		{
			name: "读到空是空态",
			tab:  tabTasks,
			src: func() *fakePanelSource {
				return newFakePanelSource("/work")
			},
			want:   []string{"暂无运行进程"},
			absent: []string{"未打开会话", "连接已断开"},
		},
		{
			name: "有内容",
			tab:  tabTasks,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.session.Tasks = []TaskSnapshot{{ID: "task-1", Description: "go test ./..."}}
				return src
			},
			want:    []string{"后台任务：go test ./... · 运行中"},
			enabled: []string{"取消任务"},
		},
		{
			name: "断线时列表仍在但停止禁用",
			tab:  tabTasks,
			src: func() *fakePanelSource {
				src := wiredDisconnected("/work")
				src.session.Tasks = []TaskSnapshot{{ID: "task-1", Description: "go test ./..."}}
				return src
			},
			want:     []string{"连接已断开：连接被网络中断", "后台任务：go test ./... · 运行中"},
			disabled: []string{"取消任务"},
		},
		{
			name: "唤醒读到空是空态",
			tab:  tabTasks,
			src: func() *fakePanelSource {
				return newFakePanelSource("/work")
			},
			open:   func(a *App) { a.panels.taskKind = 1 },
			want:   []string{"暂无待唤醒事项"},
			absent: []string{"读不到唤醒记录"},
		},
		{
			name: "唤醒读不到是错误态",
			tab:  tabTasks,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.wakeups = WakeupsSnapshot{ReadError: "连接中断"}
				return src
			},
			open:   func(a *App) { a.panels.taskKind = 1 },
			want:   []string{"读不到唤醒记录", "读取失败：连接中断"},
			absent: []string{"暂无待唤醒事项"},
		},
		{
			name: "唤醒能力缺失不装成没有",
			tab:  tabTasks,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.wakeups = WakeupsSnapshot{CapabilityMissing: true}
				return src
			},
			open:   func(a *App) { a.panels.taskKind = 1 },
			want:   []string{"读不到唤醒记录", "暂时读不到定时任务"},
			absent: []string{"暂无待唤醒事项"},
		},
		{
			name: "唤醒有内容",
			tab:  tabTasks,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.wakeups = WakeupsSnapshot{Items: []WakeupSnapshot{{ID: "w1", Prompt: "夜间构建"}}}
				return src
			},
			open:   func(a *App) { a.panels.taskKind = 1 },
			want:   []string{"唤醒：夜间构建"},
			absent: []string{"读不到唤醒记录", "暂无待唤醒事项"},
		},
	})
}

// TestTaskDetailShowsOutputReason 断言连着 Source 时输出框写「读不到输出」，
// 不写「还没有输出」——后者是一句关于事实的话，这里并不知道。
func TestTaskDetailShowsOutputReason(t *testing.T) {
	src := newFakePanelSource("/work")
	src.session.Tasks = []TaskSnapshot{{ID: "task-1", Description: "go test ./..."}}
	app, tt := rightWiredTester(t, tabTasks, src)
	app.panels.taskSel = 0
	tt.Frame()
	if !tt.HasText(noteTaskOutput) {
		t.Fatalf("输出框没有写读不到：%v", tt.Texts())
	}
	if tt.HasText(termEmptyText) {
		t.Fatalf("输出框仍写「%s」：%v", termEmptyText, tt.Texts())
	}
	// 打开日志没有通道：入口置灰并写原因（K-27）。
	info, ok := tt.Access("打开日志")
	if !ok {
		t.Fatal("详情没有打开日志入口")
	}
	if !info.Disabled || !strings.Contains(info.Description, noteTaskLog) {
		t.Fatalf("打开日志应当置灰并写原因：%+v", info)
	}
}

// ---- 13.4 目标 ----

func TestGoalPanelStateTable(t *testing.T) {
	runStateCases(t, []stateCase{
		{
			name:   "未打开会话",
			tab:    tabGoal,
			src:    wiredNoSession,
			want:   []string{"未打开会话", "打开一个会话后，这里显示它的目标。"},
			absent: []string{"还没有设置目标"},
		},
		{
			name:   "没有目标",
			tab:    tabGoal,
			src:    func() *fakePanelSource { return newFakePanelSource("/work") },
			want:   []string{"还没有设置目标", "在输入区切到 Goal 模式即可设置目标。"},
			absent: []string{"把发送路径改成排队"},
		},
		{
			name: "有目标与状态",
			tab:  tabGoal,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.session.Goal, src.session.GoalStatus = "把测试补齐", "active"
				return src
			},
			want:   []string{"把测试补齐", "进行中", "目标状态：进行中"},
			absent: []string{"还没有设置目标", "把发送路径改成排队"},
		},
		{
			name: "暂停的目标",
			tab:  tabGoal,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.session.Goal, src.session.GoalStatus = "把测试补齐", "paused"
				return src
			},
			want: []string{"把测试补齐", "已暂停"},
		},
		{
			name: "没有状态事实时不写状态",
			tab:  tabGoal,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.session.Goal = "把测试补齐"
				return src
			},
			want:   []string{"把测试补齐"},
			absent: []string{"进行中", "已暂停", "已完成", "已阻塞"},
		},
		{
			name: "断线时目标照常可读",
			tab:  tabGoal,
			src: func() *fakePanelSource {
				src := wiredDisconnected("/work")
				src.session.Goal, src.session.GoalStatus = "把测试补齐", "active"
				return src
			},
			want: []string{"把测试补齐", "进行中"},
		},
	})
}

// TestGoalPanelDemoKeepsSample 断言只有 -demo 才画样例目标：接上 Source 后
// 样例一个字都不出现。
func TestGoalPanelDemoKeepsSample(t *testing.T) {
	app, tt := panelTester(t, tabGoal)
	tt.Frame()
	if app.wired() {
		t.Fatal("样例用例不该带 Source")
	}
	if !tt.HasText("把发送路径改成排队") {
		t.Fatalf("-demo 的目标面板丢了样例：%v", tt.Texts())
	}
}

// ---- 13.6 侧聊 ----

func TestSideChatPanelStateTable(t *testing.T) {
	runStateCases(t, []stateCase{
		{
			name:     "未打开会话",
			tab:      tabSideChat,
			src:      wiredNoSession,
			want:     []string{"未打开会话", "打开一个会话后，这里显示它的侧边对话。"},
			absent:   []string{sideChatEmptyThread},
			disabled: []string{"侧边对话输入"},
		},
		{
			name:     "连接到但没有会话消息",
			tab:      tabSideChat,
			src:      func() *fakePanelSource { return newFakePanelSource("/work") },
			want:     []string{sideChatEmptyThread, noteSideChatFacts},
			disabled: []string{"侧边对话输入"},
		},
		{
			name:     "从某条消息之前开始的空态",
			tab:      tabSideChat,
			src:      func() *fakePanelSource { return newFakePanelSource("/work") },
			open:     func(a *App) { a.panels.sideChatFromMessage = true },
			want:     []string{sideChatEmptyBefore},
			absent:   []string{sideChatEmptyThread},
			disabled: []string{"侧边对话输入"},
		},
		{
			name:   "发送中给等待形态",
			tab:    tabSideChat,
			src:    func() *fakePanelSource { return newFakePanelSource("/work") },
			open:   func(a *App) { a.panels.sideChatSending = true },
			want:   []string{"正在等待回复"},
			absent: []string{sideChatEmptyThread, "正在加载"},
		},
		{
			name:   "已结束给状态条",
			tab:    tabSideChat,
			src:    func() *fakePanelSource { return newFakePanelSource("/work") },
			open:   func(a *App) { a.panels.sideChatEnded = true },
			want:   []string{sideChatClosedNotice},
			absent: []string{"结束分支"},
		},
		{
			name: "读取失败给横幅但不给假重试",
			tab:  tabSideChat,
			src:  func() *fakePanelSource { return newFakePanelSource("/work") },
			open: func(a *App) { a.panels.sideChatError = "读取消息失败" },
			want: []string{"读取消息失败"},
			// 连着 Source 没有重读侧聊的通道：重试入口不出现。
			absent:   []string{sideChatRetry},
			disabled: []string{"侧边对话输入"},
		},
		{
			name:   "断线不把通道缺口说成断线",
			tab:    tabSideChat,
			src:    func() *fakePanelSource { return wiredDisconnected("/work") },
			want:   []string{sideChatEmptyThread, noteSideChatFacts, noteSideChatWrite},
			absent: []string{"连接已断开，暂时不能操作"},
		},
	})
}

// TestSideChatDemoEchoStaysLocal 断言 -demo 的本地回声仍只发生在本端：发送后
// 内容出现在只读时间线里，输入框清空；连着 Source 时不发生回声。
func TestSideChatDemoEchoStaysLocal(t *testing.T) {
	app, tt := panelTester(t, tabSideChat)
	app.panels.sideChatDraft = "问一个旁支问题"
	app.sendSideChat()
	tt.Frame()
	if len(app.panels.sideChatSent) != 1 || app.panels.sideChatDraft != "" {
		t.Fatalf("本地回声没有发生：sent=%v draft=%q", app.panels.sideChatSent, app.panels.sideChatDraft)
	}
	if !showsText(tt, "问一个旁支问题") {
		t.Fatalf("回声没有出现在只读时间线里：%v", tt.Texts())
	}

	src := newFakePanelSource("/work")
	app2 := NewApp(Options{State: StateDefault, Source: src})
	app2.rightOpen, app2.rightTab = true, tabSideChat
	app2.panels.sideChatDraft = "问一个旁支问题"
	app2.sendSideChat()
	if len(app2.panels.sideChatSent) != 0 || app2.panels.sideChatDraft == "" {
		t.Fatalf("连着 Source 时不应回声：sent=%v draft=%q",
			app2.panels.sideChatSent, app2.panels.sideChatDraft)
	}
}

// ---- 13.7 子代理 ----

func TestSubagentPanelStateTable(t *testing.T) {
	runStateCases(t, []stateCase{
		{
			name:   "未打开会话",
			tab:    tabSubagent,
			src:    wiredNoSession,
			want:   []string{"还没有打开会话", noteSubagentNoSession},
			absent: []string{"未打开子代理会话"},
		},
		{
			name:   "有会话但没有子代理事实",
			tab:    tabSubagent,
			src:    func() *fakePanelSource { return newFakePanelSource("/work") },
			want:   []string{"未打开子代理会话", noteSubagentFacts},
			absent: []string{"正在加载"},
		},
		{
			name: "读取失败给错误行与禁用的重试",
			tab:  tabSubagent,
			src:  func() *fakePanelSource { return newFakePanelSource("/work") },
			open: func(a *App) {
				a.panels.subagentError = "子代理流读取失败"
				a.panels.subagentFailed = true
			},
			want:     []string{"子代理流读取失败", noteSubagentFacts},
			disabled: []string{"重试"},
		},
		{
			name:   "读取失败但原文为空仍画错误行",
			tab:    tabSubagent,
			src:    func() *fakePanelSource { return newFakePanelSource("/work") },
			open:   func(a *App) { a.panels.subagentFailed = true },
			want:   []string{"加载失败", noteSubagentFacts},
			absent: []string{"未打开子代理会话"},
		},
	})
}

// TestSubagentDemoRetryClearsError 断言 -demo 里「重试」只清样例错误，且入口
// 可用（连着 Source 时同一入口置灰写原因，见上表）。
func TestSubagentDemoRetryClearsError(t *testing.T) {
	app, tt := panelTester(t, tabSubagent)
	app.panels.subagentError = "样例失败"
	app.panels.subagentFailed = true
	tt.Frame()
	if err := tt.Click("重试"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	if app.panels.subagentError != "" || app.panels.subagentFailed {
		t.Fatalf("重试没有清掉错误：%q %v", app.panels.subagentError, app.panels.subagentFailed)
	}
}

// ---- 13.5 终端（只登记接口，实现归终端线） ----

// TestTerminalPanelIsLocal 断言终端页不依赖控制连接：断线时 pty 输出照常画，
// 页内不出现连接横幅（横幅只属于走连接的读数面）。
func TestTerminalPanelIsLocal(t *testing.T) {
	src := liveTerminalSource("/work/project", TerminalSnapshot{
		Active: true, Running: true, Lines: []string{"$ go test ./...", "ok"},
	})
	src.conn = Connection{Phase: "disconnected", Reason: "连接被网络中断"}
	app, tt := terminalTester(t, src, newTerminalIntents())
	for _, want := range []string{"$ go test ./...", "ok"} {
		if !showsText(tt, want) {
			t.Fatalf("断线时丢了本机终端输出 %q：%v", want, tt.Texts())
		}
	}
	// 终端页自己不是连接读数面：这一页上不出现断线横幅（窗口层的提示另算）。
	if tt.HasText("连接已断开，暂时不能操作") {
		t.Fatalf("终端页出现了连接横幅：%v", tt.Texts())
	}
	// 断线也不影响本机入口：清屏与复制输出（纯本机动作）照常可点。
	for _, name := range []string{"清屏", "复制输出"} {
		info, ok := tt.Access(name)
		if !ok {
			t.Fatalf("工具条没有 %q", name)
		}
		if info.Disabled {
			t.Fatalf("断线时 %q 被禁用了（本机 pty 不依赖连接）：%+v", name, info)
		}
	}
	_ = app
}

// TestTerminalNoSessionDisablesStart 断言没有会话时「新建终端」置灰并写原因：
// 不猜一个工作目录，也不留一个点了没反应的入口。
func TestTerminalNoSessionDisablesStart(t *testing.T) {
	src := liveTerminalSource("", TerminalSnapshot{})
	src.hasSess = false
	src.session = SessionSnapshot{}
	_, tt := terminalTester(t, src, newTerminalIntents())
	info, ok := tt.Access("新建终端")
	if !ok {
		t.Fatal("工具条没有新建终端")
	}
	if !info.Disabled {
		t.Fatalf("没有会话时新建终端应当置灰：%+v", info)
	}
	if !strings.Contains(info.Description, noteNoSessionID) {
		t.Fatalf("新建终端置灰却没写原因：%+v", info)
	}
}
