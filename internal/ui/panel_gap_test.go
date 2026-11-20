package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 右栏面板补齐项：终端退出码、关闭终端标签、文件页目录错误与单目录重试、
// 改动页 diff 的横向滚动与截断提示、任务「正在停止」、侧聊与子代理的空态原因。
// 条目编号见 docs/domains/requirements/requirements-panels.md §1–§7。

// TestTerminalExitCodeNotice 断言进程自行退出后画布给带真实退出码的提示行
// （T-11 / R-TM-087 / R-TM-088），代码 0 也照写。
func TestTerminalExitCodeNotice(t *testing.T) {
	src := liveTerminalSource("/work/project", TerminalSnapshot{
		Active: true, Exited: true, ExitKnown: true, ExitCode: 0,
		Lines: []string{"$ go test ./...", "ok"},
	})
	_, tt := terminalTester(t, src, newTerminalIntents())

	const want = "［进程已退出，代码 0］"
	if !tt.HasText(want) {
		t.Fatalf("画布没有退出提示行 %q：%v", want, tt.Texts())
	}
	// 最后一屏内容保留（T-08-065）。
	if !tt.HasText("ok") {
		t.Fatalf("退出后丢了最后一屏内容：%v", tt.Texts())
	}
	// 主动关闭与自行退出互斥：没有关闭提示。
	if tt.HasText(termClosedNotice) {
		t.Fatalf("自行退出时同时画了「进程已关闭」：%v", tt.Texts())
	}
}

// TestTerminalExitSignalNotice 断言被信号终止时写信号号，不写空的「代码 」
// （R-TM-089）。
func TestTerminalExitSignalNotice(t *testing.T) {
	src := liveTerminalSource("/work/project", TerminalSnapshot{
		Active: true, Exited: true, ExitKnown: true, ExitCode: -1,
		Signaled: true, Signal: 15,
	})
	_, tt := terminalTester(t, src, newTerminalIntents())
	if !tt.HasText("［进程已退出，信号 15］") {
		t.Fatalf("没有按信号写提示行：%v", tt.Texts())
	}
	for _, text := range tt.Texts() {
		if strings.Contains(text, "代码 ") {
			t.Fatalf("被信号终止时写了空的代码段：%q", text)
		}
	}
}

// TestTerminalExitUnknownNotice 断言读不到退出状态时只写「进程已退出」
// （R-TM-093），不拿零值冒充代码 0。
func TestTerminalExitUnknownNotice(t *testing.T) {
	src := liveTerminalSource("/work/project", TerminalSnapshot{Active: true, Exited: true})
	_, tt := terminalTester(t, src, newTerminalIntents())
	if !tt.HasText(termExitedNotice) {
		t.Fatalf("没有写通用退出提示：%v", tt.Texts())
	}
	if tt.HasText("［进程已退出，代码 0］") {
		t.Fatalf("退出状态未知却写了代码 0：%v", tt.Texts())
	}
}

// TestTerminalExitNoticeUnit 直接验提示行的三种写法。
func TestTerminalExitNoticeUnit(t *testing.T) {
	cases := []struct {
		name string
		exit termExit
		want string
	}{
		{"代码", termExit{known: true, code: 2}, "［进程已退出，代码 2］"},
		{"零码", termExit{known: true, code: 0}, "［进程已退出，代码 0］"},
		{"信号", termExit{known: true, signaled: true, signal: 9}, "［进程已退出，信号 9］"},
		{"信号无号", termExit{known: true, signaled: true}, termExitedNotice},
		{"未知", termExit{}, termExitedNotice},
	}
	for _, c := range cases {
		if got := termExitNotice(c.exit); got != c.want {
			t.Fatalf("%s：提示行 = %q，想要 %q", c.name, got, c.want)
		}
	}
}

// TestCloseTerminalClosesTab 断言「关闭终端」收掉当前标签（T-06 / R-TM-042），
// 多个标签时只少一个，不是切回文件页后标签还留着。
func TestCloseTerminalClosesTab(t *testing.T) {
	app, tt := panelTester(t, tabTerminal)
	tt.Frame()
	if err := tt.Click("新建终端"); err != nil {
		t.Fatalf("点不到新建终端：%v", err)
	}
	tt.Frame()
	if got := app.termLayerCount(); got != 2 {
		t.Fatalf("新建后应有 2 层，实际 %d", got)
	}
	if err := tt.Click("关闭终端"); err != nil {
		t.Fatalf("点不到关闭终端：%v", err)
	}
	tt.Frame()
	if got := app.termLayerCount(); got != 1 {
		t.Fatalf("关闭终端后应剩 1 层，实际 %d", got)
	}
	if app.rightTab != tabTerminal {
		t.Fatalf("还有标签时不该离开终端页：tab=%d", app.rightTab)
	}
	// 再关掉最后一个：回到文件页，终端页回到「未启动」，不留后台还在跑的层。
	if err := tt.Click("关闭终端"); err != nil {
		t.Fatalf("点不到关闭终端（最后一个）：%v", err)
	}
	tt.Frame()
	if app.rightTab != tabFiles {
		t.Fatalf("关掉最后一个终端后应回文件页：tab=%d", app.rightTab)
	}
	if got := app.termLayerCount(); got != 1 || app.termTabs[0].state != termIdle {
		t.Fatalf("关掉最后一个终端后应回到未启动：%+v", app.termTabs)
	}
}

// TestFilesDirErrorNamesDir 断言目录错误行的读屏名带目录路径（R-RF-215），
// 用户听得出来是哪一格失败。
func TestFilesDirErrorNamesDir(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree(FileEntrySnapshot{Name: "src", Dir: true})
	src.trees["src"] = FileTreeSnapshot{Phase: PanelReadFailed, Error: "没有读取权限"}
	_, tt := rightWiredTester(t, tabFiles, src)
	tt.Frame()
	if err := tt.Click("目录，src，已收起"); err != nil {
		t.Fatalf("点不到目录行：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Access("无法加载文件夹：src"); !ok {
		t.Fatalf("错误行没有带目录名：%v", tt.Texts())
	}
	if !tt.HasText("没有读取权限") {
		t.Fatalf("错误行没写原因：%v", tt.Texts())
	}
}

// TestFilesDirRetryOnlyThatDir 断言子目录的重试只重读该目录（带 Refresh），
// 不把整棵树推倒重来（F-26 / R-RF-219）。
func TestFilesDirRetryOnlyThatDir(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree(FileEntrySnapshot{Name: "src", Dir: true})
	src.trees["src"] = FileTreeSnapshot{Phase: PanelReadFailed, Error: "没有读取权限"}
	app, tt := rightWiredTester(t, tabFiles, src)
	tt.Frame()
	if err := tt.Click("目录，src，已收起"); err != nil {
		t.Fatalf("点不到目录行：%v", err)
	}
	tt.Frame()
	if err := tt.Click("重试读取文件夹：src"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	refreshed := false
	for _, dir := range src.treeRefreshed {
		if dir == "src" {
			refreshed = true
		}
	}
	if !refreshed {
		t.Fatalf("重试没有带 Refresh 重读该目录：%v", src.treeRefreshed)
	}
	for _, dir := range src.treeRefreshed {
		if dir != "src" {
			t.Fatalf("子目录重试把别的目录也刷新了：%v", src.treeRefreshed)
		}
	}
	if app.right.reload != 0 {
		t.Fatalf("子目录重试不该触发整树刷新（reload=%d）", app.right.reload)
	}
}

// failingStopIntents 让停止终端失败：用来验关闭失败时标签不消失。
type failingStopIntents struct{ *terminalIntents }

func (f *failingStopIntents) StopTerminal(context.Context) error {
	return errors.New("进程没有响应")
}

// TestCloseTerminalLiveWaitsForStop 断言连着 pty 时「关闭终端」先停进程，停止
// 成功后才收标签（T-06 / R-TM-048：停止失败时标签保留）。
func TestCloseTerminalLiveWaitsForStop(t *testing.T) {
	intents := newTerminalIntents()
	src := liveTerminalSource("/work/project", TerminalSnapshot{Active: true, Running: true})
	app, tt := terminalTester(t, src, intents)
	tt.Frame()
	if err := tt.Click("关闭终端"); err != nil {
		t.Fatalf("点不到关闭终端：%v", err)
	}
	waitIntent(t, intents.stopped, "关闭终端")
	tt.Frame()
	if app.rightTab != tabFiles {
		t.Fatalf("停止成功后应回文件页：tab=%d", app.rightTab)
	}

	// 停止失败：标签保留，不假装已关。
	bad := &failingStopIntents{terminalIntents: newTerminalIntents()}
	src2 := liveTerminalSource("/work/project", TerminalSnapshot{Active: true, Running: true})
	app2, tt2 := terminalTester(t, src2, bad)
	tt2.Frame()
	if err := tt2.Click("关闭终端"); err != nil {
		t.Fatalf("点不到关闭终端（失败路径）：%v", err)
	}
	tt2.Frame()
	tt2.Frame()
	if app2.rightTab != tabTerminal {
		t.Fatalf("停止失败却离开了终端页：tab=%d", app2.rightTab)
	}
	if got := app2.termLayerCount(); got != 1 {
		t.Fatalf("停止失败后标签应保留：%d 层", got)
	}
}

// TestFilesSearchRetryReturnsResults 断言样例里搜索失败后点「重试」当帧回到结果，
// 不停在没有返回点的「正在搜索整个工作区…」上（F-06 / R-RF-046）。
func TestFilesSearchRetryReturnsResults(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	app.right.filter = "composer"
	app.right.searchFailed = true
	tt.Frame()
	if !tt.HasText("文件搜索失败。") {
		t.Fatalf("失败态没有错误行：%v", tt.Texts())
	}
	if err := tt.Click("重试搜索"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	if tt.HasText("正在搜索整个工作区…") {
		t.Fatalf("重试后停在假等待上：%v", tt.Texts())
	}
	if app.right.searchFailed {
		t.Fatal("重试后失败位没清")
	}
	if !tt.HasText("composer.go") {
		t.Fatalf("重试后没有出匹配结果：%v", tt.Texts())
	}
}

// TestChangesDiffTruncatedOnTop 断言行内 diff 的「已截断」在内容之上
// （D-49-457），长行照原样保留、不折行（D-48-447）。
func TestChangesDiffTruncatedOnTop(t *testing.T) {
	app, tt := rightTester(t, tabChanges)
	// 给样例改动换一份超过上限的 diff：61 行，首行是长行（不该折行）。
	diff := []string{"+\t" + strings.Repeat("x", 400)}
	for i := 1; i < rpDiffMaxLines+1; i++ {
		diff = append(diff, fmt.Sprintf("-\tline %03d", i))
	}
	for _, change := range app.right.changes {
		if change.path == "internal/ui/composer.go" {
			change.diff = diff
		}
	}
	if len(app.right.changes) == 0 || app.right.changes[0].diff == nil {
		t.Fatalf("样例改动没有准备出来：%+v", app.right.changes)
	}
	tt.Frame()
	// 展开第一行的 diff。
	if err := tt.Click("文件，internal/ui/composer.go，新增 2 行，删除 1 行"); err != nil {
		t.Fatalf("点不到改动行：%v", err)
	}
	tt.Frame()
	note, ok := tt.Find("已截断")
	if !ok {
		t.Fatalf("长 diff 没有截断提示：%v", tt.Texts())
	}
	first, ok := tt.Find(diff[0])
	if !ok {
		t.Fatalf("diff 没有画出内容行（长行被折行或吞掉了）：%v", tt.Texts())
	}
	if note.Y > first.Y {
		t.Fatalf("截断提示应在内容之上：note.y=%.1f first.y=%.1f", note.Y, first.Y)
	}
	// 超出上限的行不画。
	if _, ok := tt.Find(diff[rpDiffMaxLines]); ok {
		t.Fatalf("超出上限的行仍画出来了")
	}
}

// TestChangesLiveDiffShowsReason 断言连着真实读数时展开行给「读不到改动内容」
// 的原因，而不是一个空白块（D-47-443）。
func TestChangesLiveDiffShowsReason(t *testing.T) {
	src := newFakePanelSource("/work")
	src.changes = ChangeSet{Phase: PanelReadReady, Files: []ChangeFileSnapshot{
		{Path: "internal/app/app.go", Status: "M", StatKnown: true, Added: 3, Removed: 1},
	}}
	_, tt := rightWiredTester(t, tabChanges, src)
	tt.Frame()
	row := "文件，internal/app/app.go，新增 3 行，删除 1 行"
	if err := tt.Click(row); err != nil {
		t.Fatalf("点不到改动行：%v", err)
	}
	tt.Frame()
	if !tt.HasText(noteDiffContent) {
		t.Fatalf("展开行没有写 diff 读不到的原因：%v", tt.Texts())
	}
}

// TestTaskStoppingState 断言已发停止、还没结束的任务写「正在停止」，不写
// 「已完成」（K-15）：服务端还没有结束事实。
func TestTaskStoppingState(t *testing.T) {
	src := wiredSource()
	src.session = SessionSnapshot{ID: "s1", Title: "真实会话",
		Tasks: []TaskSnapshot{{ID: "t1", Description: "npm run dev", Stopping: true}}}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.rightOpen, app.rightTab = true, tabTasks
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	if !tt.HasText("正在停止") {
		t.Fatalf("停止中的任务没有写「正在停止」：%v", tt.Texts())
	}
	if tt.HasText("已完成") {
		t.Fatalf("还没结束就写成了「已完成」：%v", tt.Texts())
	}
	if note := taskStateLine(task{kind: taskProcess, state: taskStopping}); note != "正在停止" {
		t.Fatalf("详情状态行 = %q，想要「正在停止」", note)
	}
}

// TestSideChatDisabledReasons 断言输入区不可用的三种原因分开写（S-11）。
func TestSideChatDisabledReasons(t *testing.T) {
	demo := NewApp(Options{State: StateDefault})
	if got := demo.sideChatDisabledNote(); got != "" {
		t.Fatalf("演示态可输入时给了原因 %q", got)
	}
	demo.panels.sideChatEnded = true
	if got := demo.sideChatDisabledNote(); got != sideChatEndedNote {
		t.Fatalf("已结束的原因 = %q，想要 %q", got, sideChatEndedNote)
	}
	demo.panels.sideChatEnded, demo.panels.sideChatSending = false, true
	if got := demo.sideChatDisabledNote(); got != sideChatSendingNote {
		t.Fatalf("发送中的原因 = %q，想要 %q", got, sideChatSendingNote)
	}
	live := NewApp(Options{State: StateDefault, Source: wiredSource()})
	if got := live.sideChatDisabledNote(); got != noteSideChatWrite {
		t.Fatalf("连着读数时的原因 = %q，想要 %q", got, noteSideChatWrite)
	}
}

// TestSideChatEmptyShowsReason 断言连着读数、内容区为空时给原因（S-08 的异常
// 分支），而不是只有一句看不出为什么的空态。
func TestSideChatEmptyShowsReason(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	app.rightOpen, app.rightTab = true, tabSideChat
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	if !tt.HasText(sideChatEmptyThread) {
		t.Fatalf("空态标题不是 S-08 的逐字文案：%v", tt.Texts())
	}
	if !tt.HasText(noteSideChatFacts) {
		t.Fatalf("空态没写原因：%v", tt.Texts())
	}
}

// TestSubagentEmptyReasons 断言子代理空态按「有没有会话」分开写原因（G-04）：
// 没打开会话与协议缺子代理会话流不是同一件事。
func TestSubagentEmptyReasons(t *testing.T) {
	// 有会话：协议没有子代理会话流。
	withSession := NewApp(Options{State: StateDefault, Source: wiredSource()})
	withSession.rightOpen, withSession.rightTab = true, tabSubagent
	tt := gui.NewTester(withSession.View, WindowWidth, WindowHeight)
	tt.Frame()
	if !tt.HasText(subagentNoSession) || !tt.HasText(noteSubagentFacts) {
		t.Fatalf("有会话时的空态不对：%v", tt.Texts())
	}

	// 没有会话：先要打开会话。快照没有会话才算「没打开」——只清选中名不够。
	noSessionSrc := wiredSource()
	noSessionSrc.hasSess = false
	noSessionSrc.session = SessionSnapshot{}
	noSession := NewApp(Options{State: StateDefault, Source: noSessionSrc})
	noSession.selectedSession = ""
	noSession.rightOpen, noSession.rightTab = true, tabSubagent
	tt2 := gui.NewTester(noSession.View, WindowWidth, WindowHeight)
	tt2.Frame()
	if !tt2.HasText(subagentNoOpenSession) {
		t.Fatalf("没打开会话时的空态标题不对：%v", tt2.Texts())
	}
	if !tt2.HasText(noteSubagentNoSession) {
		t.Fatalf("没打开会话时没写原因：%v", tt2.Texts())
	}
	if tt2.HasText(noteSubagentFacts) {
		t.Fatalf("没打开会话时却写成了协议缺能力：%v", tt2.Texts())
	}
}
