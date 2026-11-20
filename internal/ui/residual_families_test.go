package ui

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件补三族残留的行为测试：改动面板差异列表（D-16–D-27、D-80–D-97）、
// 任务详情（K-08–K-36）与左栏残留（LB-02–LB-09、LB-58–LB-65、LB-105/LB-106）。
//
// 断言口径：文案逐字、在场 / 不在场、禁用与原因（Description）、剪贴板内容；
// 像素与坐标由截图（docs/domains/screens）负责，这里判结构与状态。

// ---- 左栏：栏头品牌位与更新入口 ----

// TestSidebarBrandOnlyOffMacOS 核对品牌图形与产品名只在非 macOS 出现：macOS 上
// 那一段是窗口控件的位置（LB-01），不画品牌（LB-02、LB-03）。
func TestSidebarBrandOnlyOffMacOS(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if tt.HasText(sidebarProductName) {
		t.Fatalf("macOS 栏头不该画产品名：%v", tt.Texts())
	}

	old := sidebarChromeIsMacOS
	sidebarChromeIsMacOS = func() bool { return false }
	t.Cleanup(func() { sidebarChromeIsMacOS = old })
	tt.Frame()
	if !tt.HasText(sidebarProductName) {
		t.Fatal("非 macOS 栏头没有产品名")
	}
	if _, ok := tt.Find("品牌图形"); !ok {
		t.Fatal("非 macOS 栏头没有品牌图形")
	}
}

// sidebarUpdateSource 给栏头一份更新读数（其余事实走 fakeSource）。
type sidebarUpdateSource struct {
	*fakeSource
	snapshot UpdateSnapshot
}

func (s *sidebarUpdateSource) Update() UpdateSnapshot { return s.snapshot }

// TestSidebarUpdateButtonPhases 核对更新按钮按相位换文案、禁用与动作（LB-04–LB-08），
// 以及浮层里的状态 · 版本 · 错误（LB-09）。
func TestSidebarUpdateButtonPhases(t *testing.T) {
	cases := []struct {
		name      string
		snapshot  UpdateSnapshot
		label     string
		disabled  bool
		danger    bool
		tipParts  []string
		install   bool
		downloads bool
	}{
		{
			name:      "有新版本/可下载",
			snapshot:  UpdateSnapshot{Configured: true, Phase: "available", AvailableVersion: "1.4.0"},
			label:     "更新",
			tipParts:  []string{"有新版本可用", "v1.4.0"},
			downloads: true,
		},
		{
			name:     "下载中/百分比禁用",
			snapshot: UpdateSnapshot{Configured: true, Phase: "downloading", Percent: 42, AvailableVersion: "1.4.0"},
			label:    "42%",
			disabled: true,
			tipParts: []string{"下载中 42%", "v1.4.0"},
		},
		{
			name:     "正在准备/禁用",
			snapshot: UpdateSnapshot{Configured: true, Phase: "preparing", AvailableVersion: "1.4.0"},
			label:    "正在准备…",
			disabled: true,
			tipParts: []string{"正在准备", "v1.4.0"},
		},
		{
			name:     "已下载/无安装能力置灰并写原因",
			snapshot: UpdateSnapshot{Configured: true, Phase: "downloaded", AvailableVersion: "1.4.0"},
			label:    "重启安装",
			disabled: true,
			tipParts: []string{"已下载，待重启", "v1.4.0", NoteUpdateInstall},
		},
		{
			name: "已下载/有安装能力可点",
			snapshot: UpdateSnapshot{Configured: true, Phase: "downloaded", AvailableVersion: "1.4.0",
				Installable: true},
			label:    "重启安装",
			install:  true,
			tipParts: []string{"已下载，待重启", "v1.4.0"},
		},
		{
			name: "下载失败/重试更新用警示色",
			snapshot: UpdateSnapshot{Configured: true, Phase: "available", AvailableVersion: "1.4.0",
				Err: "网络不可用"},
			label:     "重试更新",
			danger:    true,
			downloads: true,
			tipParts:  []string{"更新失败", "v1.4.0", "网络不可用"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := &sidebarUpdateSource{fakeSource: &fakeSource{conn: Connection{Phase: "connected"}}, snapshot: c.snapshot}
			intents := newUpdateTestIntents()
			app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			info, ok := tt.Access(c.label)
			if !ok {
				t.Fatalf("栏头没有更新按钮 %q：%v", c.label, tt.Texts())
			}
			if info.Disabled != c.disabled {
				t.Fatalf("%q 的禁用态 = %v，想要 %v", c.label, info.Disabled, c.disabled)
			}
			for _, part := range c.tipParts {
				if !strings.Contains(info.Description, part) {
					t.Fatalf("%q 的浮层 %q 缺少 %q", c.label, info.Description, part)
				}
			}
			if c.disabled {
				return
			}
			if err := tt.Click(c.label); err != nil {
				t.Fatalf("点不到 %q：%v", c.label, err)
			}
			switch {
			case c.install:
				if !waitSignal(intents.installed) {
					t.Fatal("点「重启安装」没有发出安装动作")
				}
			case c.downloads:
				if !waitSignal(intents.downloaded) {
					t.Fatal("点更新没有发出下载动作")
				}
			}
		})
	}
}

// TestSidebarUpdateHiddenWhenUpToDate 核对已是最新时不挂更新按钮（「已是最新」
// 属于设置 › 关于，栏头不重复）。
func TestSidebarUpdateHiddenWhenUpToDate(t *testing.T) {
	src := &sidebarUpdateSource{fakeSource: &fakeSource{}, snapshot: UpdateSnapshot{Configured: true, Phase: "up_to_date"}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	for _, label := range []string{"更新", "重试更新", "重启安装"} {
		if _, ok := tt.Access(label); ok {
			t.Fatalf("已是最新版本时栏头不该有 %q", label)
		}
	}
}

// waitSignal 等一个动作信号（意图在后台协程里跑，点完不能立刻断言）。
func waitSignal(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	case <-time.After(time.Second):
		return false
	}
}

// ---- 左栏：分页与拖入 ----

// sidebarPagedApp 造一个分组视图、带 n 个附加工作区的左栏。
func sidebarPagedApp(t *testing.T, n int) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: StateDefault})
	app.workspaces = nil
	for i := 1; i <= n; i++ {
		app.workspaces = append(app.workspaces, t.TempDir()+"/ws"+strconv.Itoa(i))
	}
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// TestSidebarGroupPager：分组视图按页展示工作区，「显示更多 / 上一页」翻页
// （LB-58 / LB-59）。
func TestSidebarGroupPager(t *testing.T) {
	app, tt := sidebarPagedApp(t, sidebarGroupPageSize+2)
	if _, ok := tt.Find("显示更多"); !ok {
		t.Fatalf("分组多于一页时没有「显示更多」：%v", tt.Texts())
	}
	if _, ok := tt.Find("上一页"); ok {
		t.Fatal("第一页不该有「上一页」")
	}
	if err := tt.Click("显示更多"); err != nil {
		t.Fatalf("点不到「显示更多」：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("上一页"); !ok {
		t.Fatal("翻到第二页后没有「上一页」")
	}
	if app.sidebarView.groupPage != 1 {
		t.Fatalf("当前页 = %d，想要 1", app.sidebarView.groupPage)
	}
	if err := tt.Click("上一页"); err != nil {
		t.Fatalf("点不到「上一页」：%v", err)
	}
	tt.Frame()
	if app.sidebarView.groupPage != 0 {
		t.Fatalf("上一页后当前页 = %d，想要 0", app.sidebarView.groupPage)
	}
}

// TestSidebarFlatPager：平铺视图按批展示，尾部给「显示更多 / 显示更少」
// （LB-64 / LB-65）。
func TestSidebarFlatPager(t *testing.T) {
	names := make([]string, 0, sidebarFlatPageSize+5)
	for i := 0; i < sidebarFlatPageSize+5; i++ {
		names = append(names, "会话-"+strconv.Itoa(i))
	}
	src := &fakeSource{conn: Connection{Phase: "connected"}, names: names}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.sidebarView.flat = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("显示更多"); !ok {
		t.Fatalf("平铺视图还有未展示项时没有「显示更多」：%v", tt.Texts())
	}
	if err := tt.Click("显示更多"); err != nil {
		t.Fatalf("点不到「显示更多」：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("显示更少"); !ok {
		t.Fatal("全部展示后没有「显示更少」")
	}
	// 列表比窗口长：先把列表滚到底，再点尾部分页。
	tt.Wheel(120, 300, 0, 2000)
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	if err := tt.Click("显示更少"); err != nil {
		t.Fatalf("点不到「显示更少」：%v", err)
	}
	tt.Frame()
	if app.sidebarView.flatLimit != sidebarFlatPageSize {
		t.Fatalf("收起后展示上限 = %d，想要 %d", app.sidebarView.flatLimit, sidebarFlatPageSize)
	}
}

// TestSidebarDragHint：拖入时提示只有一个、主副两行逐字（LB-105 / LB-106）。
func TestSidebarDragHint(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	app.sidebarView.dragHint = true
	tt.Frame()
	if !tt.HasText("松开以添加工作区") {
		t.Fatalf("拖入时没有主提示：%v", tt.Texts())
	}
	if !tt.HasText("仅限文件夹 · 同时新建会话") {
		t.Fatal("拖入时没有副提示")
	}
	app.sidebarView.dragHint = false
	tt.Frame()
	if tt.HasText("松开以添加工作区") {
		t.Fatal("拖出后提示没有消失")
	}
}

// TestSidebarDropFolderAddsWorkspace：拖入文件夹登记成工作区并新建会话
// （LB-105 / LB-106）；拖入文件给失败原因，不留半截提示。
func TestSidebarDropFolderAddsWorkspace(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	dir := t.TempDir()
	at, ok := tt.Find("会话")
	if !ok {
		t.Fatal("找不到列表头")
	}
	if !tt.DropFiles(at.X+at.W/2, at.Y+at.H/2, dir) {
		t.Fatal("列表区没有接住拖入的文件夹")
	}
	tt.Frame()
	found := false
	for _, path := range app.workspaces {
		if strings.HasPrefix(path, dir) {
			found = true
		}
	}
	if !found {
		t.Fatalf("拖入的文件夹没有登记成工作区：%v", app.workspaces)
	}
	if app.state != StateEmpty {
		t.Fatalf("拖入后没有新建会话（状态 = %v）", app.state)
	}

	app2 := NewApp(Options{State: StateDefault})
	tt2 := gui.NewTester(app2.View, WindowWidth, WindowHeight)
	file := dir + "/note.txt"
	if err := os.WriteFile(file, []byte("note"), 0o600); err != nil {
		t.Fatalf("准备文件：%v", err)
	}
	at2, _ := tt2.Find("会话")
	tt2.DropFiles(at2.X+at2.W/2, at2.Y+at2.H/2, file)
	tt2.Frame()
	if !tt2.HasText("只能拖入文件夹（文件不会被登记成工作区）") {
		t.Fatalf("拖入文件后没有失败原因：%v", tt2.Texts())
	}
}

// TestSidebarGroupDragReorder：把工作区拖到列表最前 / 最后（LB-60）。
func TestSidebarGroupDragReorder(t *testing.T) {
	app, tt := sidebarPagedApp(t, 3)
	// 分组按会话数与活跃时间排，样例里最后两组是没有会话的本端目录：
	// 取一个目标组的行，把它拖到列表最前的落点线上。
	source, ok := tt.Find("ws1")
	if !ok {
		t.Fatalf("找不到工作区行 ws1：%v", tt.Texts())
	}
	line, ok := tt.Find("工作区拖拽落点：列表开头")
	if !ok {
		t.Fatal("没有列表开头的落点线")
	}
	tt.Press(source.X+source.W/2, source.Y+source.H/2)
	tt.Move(line.X+line.W/2, line.Y+line.H/2)
	tt.Frame()
	tt.Release(line.X+line.W/2, line.Y+line.H/2)
	tt.Frame()
	if len(app.sidebarView.groupOrder) == 0 || !strings.HasSuffix(app.sidebarView.groupOrder[0], "ws1") {
		t.Fatalf("拖到最前后顺序 = %v，想要 ws1 在最前", app.sidebarView.groupOrder)
	}
}

// ---- 任务面板：行与详情 ----

// taskTester 造一个打开在任务面板、带指定任务样例的界面。
func taskTester(t *testing.T, items []task) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: StateDefault})
	app.tasks = items
	app.rightOpen, app.rightTab = true, tabTasks
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// TestTaskRowBranchBadgeAndTime 核对行上的分支徽标（K-11）与任务时间（K-12）：
// main 分支不画徽标。
func TestTaskRowBranchBadgeAndTime(t *testing.T) {
	app, tt := taskTester(t, []task{
		{id: "t1", kind: taskProcess, name: "go test ./...", state: taskRunning,
			branch: "feature/panel", startedAt: "21:24", command: "go test ./..."},
	})
	if !tt.HasText("feature/panel") {
		t.Fatalf("行上没有分支徽标：%v", tt.Texts())
	}
	if !tt.HasText("21:24") {
		t.Fatal("行上没有任务时间")
	}
	app.tasks = append(app.tasks, task{id: "t2", kind: taskProcess, name: "npm run build", state: taskDone, branch: "main", startedAt: "21:10"})
	tt.Frame()
	times := 0
	for _, text := range tt.Texts() {
		if text == "main" {
			times++
		}
	}
	if times != 0 {
		t.Fatal("main 分支不该画徽标")
	}
}

// TestTaskStateGlyphs 核对五种状态各有图标（K-05–K-09、K-20）。
func TestTaskStateGlyphs(t *testing.T) {
	app, tt := taskTester(t, []task{
		{id: "r", name: "运行中任务", state: taskRunning},
		{id: "d", name: "完成任务", state: taskDone},
		{id: "f", name: "失败任务", state: taskFailed},
		{id: "s", name: "停止任务", state: taskStopped},
		{id: "u", name: "未知任务", state: taskNoStatus},
	})
	if _, ok := tt.Find("图标：task"); !ok {
		t.Fatalf("运行中的行首没有状态图标：%v", tt.Texts())
	}
	for _, state := range []taskState{taskDone, taskFailed, taskStopped, taskNoStatus} {
		if !tt.HasText("状态：" + taskStateText(state)) {
			t.Fatalf("状态 %q 没有行首图标：%v", taskStateText(state), tt.Texts())
		}
	}
	_ = app
}

// TestTaskDetailStatusAndMeta：详情头状态行按「状态 · 时刻」拼（K-21–K-24），
// 元信息行各段按有无拼接（K-28），命令行带 $ 前缀（K-29）。
func TestTaskDetailStatusAndMeta(t *testing.T) {
	app, tt := taskTester(t, []task{{
		id: "t1", kind: taskProcess, name: "go test ./...", state: taskRunning,
		toolName: "exec", pid: "48213", startedAt: "21:24", command: "go test ./...",
		startAt: time.Now().Add(-5 * time.Second),
	}})
	if err := tt.Click("go test ./..."); err != nil {
		t.Fatalf("点不到任务行：%v", err)
	}
	tt.Frame()
	if !tt.HasText("运行中 · ") {
		t.Fatalf("详情头没有「状态 · 时长」：%v", tt.Texts())
	}
	if !tt.HasText("exec · 21:24 · pid 48213") {
		t.Fatalf("元信息行不对：%v", tt.Texts())
	}
	if _, ok := tt.Find("$ go test ./..."); !ok {
		t.Fatal("详情没有命令行")
	}
	_ = app
}

// TestTaskDetailCopyFeedback：复制命令进剪贴板，按钮两秒内显示「已复制」
// （K-25 / K-26）。
func TestTaskDetailCopyFeedback(t *testing.T) {
	app, tt := taskTester(t, []task{{
		id: "t1", kind: taskProcess, name: "go test ./...", state: taskRunning,
		command: "go test ./...",
	}})
	if err := tt.Click("go test ./..."); err != nil {
		t.Fatalf("点不到任务行：%v", err)
	}
	tt.Frame()
	if err := tt.Click("复制命令"); err != nil {
		t.Fatalf("点不到复制命令：%v", err)
	}
	waitFrame(t, tt, "剪贴板拿到命令", func() bool { return tt.Clipboard() == "go test ./..." })
	if !tt.HasText("已复制") {
		t.Fatalf("复制后没有「已复制」反馈：%v", tt.Texts())
	}
	// 两秒窗口外回到复制入口：把时间戳挪到窗口前，等价于计时器到点。
	app.panels.taskCopiedAt = time.Now().Add(-taskCopiedFeedback - time.Second)
	tt.Frame()
	if tt.HasText("已复制") {
		t.Fatal("两秒后仍显示「已复制」")
	}
	if _, ok := tt.Access("复制命令"); !ok {
		t.Fatal("两秒后没有回到复制入口")
	}
}

// TestTaskOpenLogDisabled：打开日志要宿主打开文件的通道，入口保留但置灰，
// 原因写在提示里（K-27）。
func TestTaskOpenLogDisabled(t *testing.T) {
	_, tt := taskTester(t, []task{{
		id: "t1", kind: taskProcess, name: "go test ./...", state: taskRunning,
		command: "go test ./...",
	}})
	if err := tt.Click("go test ./..."); err != nil {
		t.Fatalf("点不到任务行：%v", err)
	}
	tt.Frame()
	info, ok := tt.Access("打开日志")
	if !ok {
		t.Fatal("详情没有打开日志入口")
	}
	if !info.Disabled {
		t.Fatal("没有宿主打开文件的通道时「打开日志」应置灰")
	}
	if !strings.Contains(info.Description, noteTaskLog) {
		t.Fatalf("置灰原因没有写进提示：%q", info.Description)
	}
}

// TestTaskOutputTruncatedAndReconnect：输出截断给「预览已截断」（K-32），读取
// 失败给「终端操作失败」与「重新连接」（K-33 / K-34）。
func TestTaskOutputTruncatedAndReconnect(t *testing.T) {
	app, tt := taskTester(t, []task{{
		id: "t1", kind: taskProcess, name: "go test ./...", state: taskRunning,
		command: "go test ./...", logTruncated: true,
	}})
	if err := tt.Click("go test ./..."); err != nil {
		t.Fatalf("点不到任务行：%v", err)
	}
	tt.Frame()
	if !tt.HasText("预览已截断") {
		t.Fatalf("截断态没有标签：%v", tt.Texts())
	}

	app.tasks[0].logTruncated = false
	app.tasks[0].logFailed = true
	tt.Frame()
	if !tt.HasText("终端操作失败") {
		t.Fatalf("失败态没有提示行：%v", tt.Texts())
	}
	if err := tt.Click("重新连接"); err != nil {
		t.Fatalf("点不到重新连接：%v", err)
	}
	tt.Frame()
	if app.tasks[0].logFailed {
		t.Fatal("重连后失败标记没有清掉")
	}
	if tt.HasText("终端操作失败") {
		t.Fatal("重连后仍显示失败提示")
	}
}

// TestTaskWakeupsFromSource：唤醒列表与详情取 Wakeups() 的读数（K-35 / K-36）。
func TestTaskWakeupsFromSource(t *testing.T) {
	src := &fakeSource{
		conn:    Connection{Phase: "connected"},
		session: SessionSnapshot{ID: "s1", Title: "唤醒", Branch: "main"},
		hasSess: true,
		wakeups: WakeupsSnapshot{Items: []WakeupSnapshot{
			{ID: "w1", TriggerAt: time.Date(2026, 11, 20, 9, 30, 0, 0, time.Local), Prompt: "把夜间构建的结果发出来"},
			{ID: "w2", Prompt: "没有触发时刻的记录"},
		}},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.rightOpen, app.rightTab, app.panels.taskKind = true, tabTasks, 1
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText("把夜间构建的结果发出来") {
		t.Fatalf("唤醒列表没有提示正文：%v", tt.Texts())
	}
	if !tt.HasText("09:30") {
		t.Fatalf("唤醒列表没有触发时刻：%v", tt.Texts())
	}
	if err := tt.Click("把夜间构建的结果发出来"); err != nil {
		t.Fatalf("点不到唤醒行：%v", err)
	}
	tt.Frame()
	if !tt.HasText("09:30 触发") {
		t.Fatalf("唤醒详情没有触发时刻：%v", tt.Texts())
	}
	// 没有触发时刻时写「触发时间未知」，不留空的 {time}。
	if err := tt.Click("没有触发时刻的记录"); err != nil {
		t.Fatalf("点不到第二条唤醒行：%v", err)
	}
	tt.Frame()
	if !tt.HasText("触发时间未知") {
		t.Fatalf("缺时刻时没有写「触发时间未知」：%v", tt.Texts())
	}
}

// TestTaskWakeupsCapabilityMissing：服务端没有唤醒读取时写能力缺失，不装作
// 「没有待唤醒事项」（K-19）。
func TestTaskWakeupsCapabilityMissing(t *testing.T) {
	src := &fakeSource{conn: Connection{Phase: "connected"},
		wakeups: WakeupsSnapshot{CapabilityMissing: true}}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.rightOpen, app.rightTab, app.panels.taskKind = true, tabTasks, 1
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText("暂无待唤醒事项（服务端没有提供唤醒读取）") {
		t.Fatalf("空态没有写能力缺失：%v", tt.Texts())
	}
}

// ---- 改动：骨架与时间线「本轮改动」块 ----

// TestChangesSkeletonFromReadState：读取中画三条骨架（D-16），读完后换成空态。
func TestChangesSkeletonFromReadState(t *testing.T) {
	src := newFakePanelSource("/work/demo")
	src.changes = ChangeSet{Phase: PanelReadLoading}
	_, tt := rightWiredTester(t, tabChanges, src)
	if _, ok := tt.Find("正在加载文件树"); !ok {
		t.Fatalf("读取中没有骨架容器：%v", tt.Texts())
	}
	src.changes = ChangeSet{Phase: PanelReadReady}
	tt.Frame()
	if _, ok := tt.Find("正在加载文件树"); ok {
		t.Fatal("读完以后骨架还在")
	}
	if !tt.HasText("没有未提交的改动") {
		t.Fatalf("读完以后没有空态：%v", tt.Texts())
	}
}

// TestTimelineFileChangesBlock：头行计数、展开态、状态字母与增删统计、行右键
// 菜单的分组（D-80–D-97）。
func TestTimelineFileChangesBlock(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	info, ok := tt.Access("1 个文件改动")
	if !ok {
		t.Fatalf("没有「本轮改动」折叠头：%v", tt.Texts())
	}
	if !strings.Contains(info.Description, "已收起") {
		t.Fatalf("折叠头没有写收起态：%q", info.Description)
	}
	if err := tt.Click("1 个文件改动"); err != nil {
		t.Fatalf("点不到折叠头：%v", err)
	}
	// 展开是动画（revealPanel）：把框架时钟推过去，再画一帧。
	tt.Advance(timelineCollapse.Duration + 50*time.Millisecond)
	tt.Frame()
	waitFrame(t, tt, "改动列表展开", func() bool {
		r, ok := tt.Find("internal/ui/composer.go")
		return ok && r.H > 0
	})
	if r, _ := tt.Find("internal/ui/composer.go"); r.Y < 0 || r.Y > WindowHeight {
		// 时间线比窗口长：滚到底再点。
		tt.Wheel(WindowWidth/2, WindowHeight/2, 0, 4000)
		tt.Advance(500 * time.Millisecond)
		tt.Frame()
	}
	if !tt.HasText("internal/ui/composer.go") {
		t.Fatalf("展开后没有改动文件行：%v", tt.Texts())
	}
	if !tt.HasText("+19/-2") {
		t.Fatalf("行内统计不是 +n/-n：%v", tt.Texts())
	}
	if _, ok := tt.Find("M"); !ok {
		t.Fatal("行内没有状态字母")
	}
	// 行右键：打开类两项置灰写原因，复制路径算不出工作目录时也置灰。
	// 菜单画在窗口里（不是原生菜单）：按文案与禁用态判。
	if err := tt.RightClick("改动文件 internal/ui/composer.go"); err != nil {
		t.Fatalf("点不到改动文件行：%v", err)
	}
	tt.Frame()
	// 菜单形状与顺序：打开类两项在复制类之上，复制路径在复制相对路径之上；
	// 两类之间一条分隔线（D-93–D-97）。
	var last float32 = -1
	for _, label := range []string{"用默认应用打开", "在文件管理器中显示",
		"复制路径", "复制相对路径"} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("右键菜单缺少 %q：%v", label, tt.Texts())
		}
		if r.Y <= last {
			t.Fatalf("右键菜单顺序不对：%q 不在上一项下面（y=%.1f，上一项 %.1f）", label, r.Y, last)
		}
		last = r.Y
	}
	if !tt.HasText("分隔线") {
		t.Fatal("打开类与复制类之间缺少分隔线")
	}
	for _, label := range []string{"用默认应用打开", "在文件管理器中显示", "复制路径"} {
		info, ok := tt.Access(label)
		if !ok || !info.Disabled {
			t.Fatalf("%q 应置灰：%+v", label, info)
		}
	}
	// 打开类两项要写原因（unavailable.go 口径），不能只给一个没有说明的灰项。
	if !tt.HasText(notePreviewIO) {
		t.Fatalf("置灰的打开类入口没有写原因 %q：%v", notePreviewIO, tt.Texts())
	}
	// 行上的路径本身就是工作区相对路径：复制相对路径不需要工作目录。
	if info, ok := tt.Access("复制相对路径"); !ok || info.Disabled {
		t.Fatalf("「复制相对路径」应可点：%+v", info)
	}
}

// TestTimelineFileChangeMenuCopiesPaths：有工作目录时「复制路径 / 复制相对路径」
// 都能点，内容分别是绝对与相对路径（D-96 / D-97）。
func TestTimelineFileChangeMenuCopiesPaths(t *testing.T) {
	src := &fakeSource{
		conn: Connection{Phase: "connected"},
		session: SessionSnapshot{ID: "s1", Title: "改动", Cwd: "/work/demo", Running: false,
			Rows: []RowSnapshot{{Kind: RowFileChanges, Children: []RowSnapshot{
				{Kind: RowFileChange, Text: "internal/ui/composer.go"},
			}}}},
		hasSess: true,
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("1 个文件改动"); err != nil {
		t.Fatalf("点不到折叠头：%v", err)
	}
	// 展开是动画（revealPanel）：把框架时钟推过去，再画一帧。
	tt.Advance(timelineCollapse.Duration + 50*time.Millisecond)
	tt.Frame()
	waitFrame(t, tt, "改动列表展开", func() bool {
		r, ok := tt.Find("internal/ui/composer.go")
		return ok && r.H > 0
	})
	if r, _ := tt.Find("internal/ui/composer.go"); r.Y < 0 || r.Y > WindowHeight {
		tt.Wheel(WindowWidth/2, WindowHeight/2, 0, 4000)
		tt.Advance(500 * time.Millisecond)
		tt.Frame()
	}
	if err := tt.RightClick("改动文件 internal/ui/composer.go"); err != nil {
		t.Fatalf("点不到改动文件行：%v", err)
	}
	tt.Frame()
	// 有工作目录时两项复制都能点：禁用态只留给打开类（宿主没有通道）。
	for _, label := range []string{"复制路径", "复制相对路径"} {
		info, ok := tt.Access(label)
		if !ok || info.Disabled {
			t.Fatalf("%q 应可点：%+v", label, info)
		}
	}
	if info, ok := tt.Access("用默认应用打开"); !ok || !info.Disabled {
		t.Fatalf("「用默认应用打开」应置灰：%+v", info)
	}
	if err := tt.Click("复制路径"); err != nil {
		t.Fatalf("点不到「复制路径」：%v", err)
	}
	if got := tt.Clipboard(); got != "/work/demo/internal/ui/composer.go" {
		t.Fatalf("复制的绝对路径 = %q", got)
	}
	tt.Frame()
	tt.Frame()
	if err := tt.RightClick("改动文件 internal/ui/composer.go"); err != nil {
		t.Fatalf("点不到改动文件行：%v", err)
	}
	tt.Frame()
	if err := tt.Click("复制相对路径"); err != nil {
		t.Fatalf("点不到「复制相对路径」：%v", err)
	}
	if got := tt.Clipboard(); got != "internal/ui/composer.go" {
		t.Fatalf("复制的相对路径 = %q", got)
	}
}
