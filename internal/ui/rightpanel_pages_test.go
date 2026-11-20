package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 右栏标签条、文件页与改动页的走查断言：按 inventory-panels-rt.md §1/§2 与
// requirements-panels.md §1/§2 核对在场元素、文案逐字与主要分支。像素与几何
// 由截图（docs/domains/screens 的 variant-right-panel 等）负责。

// rightTester 造一个右栏打开在指定标签上的界面。
func rightTester(t *testing.T, tab int) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: StateDefault})
	app.rightOpen, app.rightTab = true, tab
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// 标签条：文件标签、＋新建；展开面板只在 Review 页；右键五项；⌘ 徽标按标签数出现。
func TestRightPanelTabBar(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	for _, label := range []string{"文件", "新建标签页"} {
		if _, ok := tt.Access(label); !ok {
			t.Fatalf("标签条缺少 %q", label)
		}
	}
	// 参考的文件页顶栏只有 ＋：展开面板不出现在文件页。
	if _, ok := tt.Access("展开面板"); ok {
		t.Fatal("文件页顶栏不该有展开面板")
	}
	// 标签右键：五项（清单 RP-A-31…35）。菜单画在窗口里，按文案判在场。
	if err := tt.RightClick("文件"); err != nil {
		t.Fatalf("右键标签：%v", err)
	}
	tt.Frame()
	for _, label := range []string{"在右侧新建标签页", "重新加载", "关闭", "关闭其他标签页",
		"关闭右侧标签页"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("标签右键菜单缺少 %q", label)
		}
	}
	// 单击 ＋：菜单里按可用性列出可开的页（本产品有会话，故含 Review 与侧边对话）。
	if err := tt.Click("新建标签页"); err != nil {
		t.Fatalf("点不到新建标签页：%v", err)
	}
	tt.Frame()
	for _, label := range []string{"Review", "终端", "侧边对话"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("＋ 菜单缺少 %q", label)
		}
	}
	// 从菜单打开 Review：新标签出现，当前页换成改动页。
	if err := tt.Click("Review"); err != nil {
		t.Fatalf("点不到 Review 菜单项：%v", err)
	}
	tt.Frame()
	if app.rightTab != tabChanges {
		t.Fatalf("打开 Review 后当前页 = %d，想要 %d", app.rightTab, tabChanges)
	}
	if _, ok := tt.Access("展开面板"); !ok {
		t.Fatal("Review 页顶栏缺少展开面板")
	}
	if _, ok := tt.Find("未提交改动"); !ok {
		t.Fatal("改动页没有范围栏")
	}
	// 关掉 Review：右栏仍开着，回到文件页。
	if err := tt.RightClick("Review"); err != nil {
		t.Fatalf("右键 Review：%v", err)
	}
	if err := tt.Click("关闭"); err != nil {
		t.Fatalf("点不到关闭：%v", err)
	}
	tt.Frame()
	if app.rightTab != tabFiles || !app.rightOpen {
		t.Fatalf("关掉 Review 后应回文件页，实际 tab=%d open=%v", app.rightTab, app.rightOpen)
	}
}

// P1：标签条左端的「返回」只在有上层（文件预览）时可点；点了回文件页。
func TestRightPanelBackButton(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	if _, ok := tt.Access("返回"); !ok {
		t.Fatal("标签条缺少「返回」")
	}
	// 文件页本身是根：返回无处可去，点了不动。
	if err := tt.Click("返回"); err != nil {
		t.Fatalf("点不到返回：%v", err)
	}
	tt.Frame()
	if app.rightTab != tabFiles {
		t.Fatalf("在文件页点返回后换到了 %d", app.rightTab)
	}

	// 打开文件预览：返回到文件页。
	app.rpOpenPreview("internal/ui/composer.go")
	tt.Frame()
	if app.rightTab != tabFilePreview {
		t.Fatalf("预览没有打开：tab=%d", app.rightTab)
	}
	if err := tt.Click("返回"); err != nil {
		t.Fatalf("点不到返回：%v", err)
	}
	tt.Frame()
	if app.rightTab != tabFiles {
		t.Fatalf("从预览点返回后 tab=%d，想要文件页 %d", app.rightTab, tabFiles)
	}
}

// 文件页：筛选框、目录行、文件行、搜索四态里的三态（搜索中 / 结果 / 空态）
// 与「加载更多」。
func TestRightPanelFilesPage(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	if info, ok := tt.Access("筛选文件"); !ok || info.Placeholder != "筛选文件..." {
		t.Fatalf("筛选框占位 = %q，想要「筛选文件...」", info.Placeholder)
	}
	// 与运行时清单同数量：33 个根目录 + 31 个根文件（目录全部在前）。
	if len(app.right.tree) != 64 {
		t.Fatalf("样例树 %d 项，想要 64（33 目录 + 31 文件）", len(app.right.tree))
	}
	dirs := 0
	for i, node := range app.right.tree {
		if node.dir {
			dirs++
			continue
		}
		if i < 33 {
			t.Fatalf("第 %d 项是文件，目录应全部在前", i)
		}
	}
	if dirs != 33 {
		t.Fatalf("样例树目录 %d 个，想要 33", dirs)
	}
	// R 态：目录全部收起，文件行在目录之后。
	if _, ok := tt.Access("目录，assets，已收起"); !ok {
		t.Fatal("没有目录行")
	}
	if _, ok := tt.Access("文件，README.md"); !ok {
		t.Fatal("没有文件行")
	}
	// 展开目录：再点一次，出现子项。
	if err := tt.Click("目录，assets，已收起"); err != nil {
		t.Fatalf("点不到目录行：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("logo.svg"); !ok {
		t.Fatal("展开目录后没有子项")
	}
	// 搜索中：由真实的异步搜索读数点亮（本机样例是同步匹配）。
	app.right.searching = true
	app.right.filter = "README"
	tt.Frame()
	if _, ok := tt.Find("正在搜索整个工作区…"); !ok {
		t.Fatal("搜索态没有状态行")
	}
	app.right.searching = false
	app.right.filter = "README"
	tt.Frame()
	if _, ok := tt.Find("README.md"); !ok {
		t.Fatal("搜索没有列出命中文件")
	}
	// 无命中：空态标题与说明（说明带查询词）。
	app.right.filter = "zzzz"
	tt.Frame()
	if _, ok := tt.Find("无匹配文件"); !ok {
		t.Fatal("搜索无命中没有空态标题")
	}
	if _, ok := tt.Find("没有找到包含“zzzz”的文件"); !ok {
		t.Fatal("搜索无命中没有说明文案")
	}
}

// clickAccess 按可访问名找元素，再点它的中心（纯图标入口没有可见文字）。
func clickAccess(t *testing.T, tt *gui.Tester, name string) {
	t.Helper()
	info, ok := tt.Access(name)
	if !ok {
		t.Fatalf("找不到入口 %q", name)
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Frame()
}

// 标签右键的关闭类动作：关闭右侧 / 关闭其他。
func TestRightPanelTabMenuClose(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	app.rp().tabs = []int{tabFiles, tabChanges, tabTerminal}
	app.rightTab = tabFiles
	tt.Frame()
	if err := tt.RightClick("Review"); err != nil {
		t.Fatalf("右键 Review：%v", err)
	}
	if err := tt.Click("关闭右侧标签页"); err != nil {
		t.Fatalf("点不到关闭右侧标签页：%v", err)
	}
	tt.Frame()
	if len(app.rp().tabs) != 2 || app.rightTab != tabFiles {
		t.Fatalf("关闭右侧后 tabs=%v active=%d，想要 [文件 Review] 且仍在文件页",
			app.rp().tabs, app.rightTab)
	}
	if err := tt.RightClick("文件"); err != nil {
		t.Fatalf("右键文件：%v", err)
	}
	if err := tt.Click("关闭其他标签页"); err != nil {
		t.Fatalf("点不到关闭其他标签页：%v", err)
	}
	tt.Frame()
	if len(app.rp().tabs) != 1 || app.rp().tabs[0] != tabFiles {
		t.Fatalf("关闭其他后 tabs=%v，想要只剩文件", app.rp().tabs)
	}
}

// 标签右键的「在右侧新建标签页」：新页插在该标签之后。
func TestRightPanelTabMenuNewToRight(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	app.rp().tabs = []int{tabFiles, tabChanges}
	app.rightTab = tabFiles
	tt.Frame()
	if err := tt.RightClick("文件"); err != nil {
		t.Fatalf("右键文件：%v", err)
	}
	if err := tt.Click("在右侧新建标签页"); err != nil {
		t.Fatalf("点不到在右侧新建标签页：%v", err)
	}
	tt.Frame()
	if err := tt.Click("终端"); err != nil {
		t.Fatalf("点不到终端：%v", err)
	}
	tt.Frame()
	want := []int{tabFiles, tabTerminal, tabChanges}
	got := app.rp().tabs
	if len(got) != len(want) {
		t.Fatalf("新页插入后 tabs=%v，想要 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("新页插入后 tabs=%v，想要 %v", got, want)
		}
	}
	if app.rightTab != tabTerminal {
		t.Fatalf("新页插入后当前页 = %d，想要终端", app.rightTab)
	}
}

// 文件行右键：暂存类一项（暂存 / 取消暂存二选一）+ 查看差异 + 打开类三项 +
// 另存为 + 复制路径（F-34–F-44）。
func TestRightPanelFileMenu(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	app.right.dirOpen["internal"] = true
	app.right.dirOpen["internal/ui"] = true
	tt.Frame()
	if err := tt.RightClick("composer.go"); err != nil {
		t.Fatalf("右键未暂存文件：%v", err)
	}
	tt.Frame()
	for _, item := range []string{"暂存", "查看差异", "预览", "用外部工具打开",
		"在文件管理器中显示", "另存为…", "复制路径"} {
		if _, ok := tt.Find(item); !ok {
			t.Fatalf("文件右键菜单缺少 %q", item)
		}
	}
	if _, ok := tt.Find("取消暂存"); ok {
		t.Fatal("未暂存文件不该出现「取消暂存」")
	}
	// 外部打开、文件管理器、另存为要宿主能力（DEC-09）：菜单项留在原位但置灰。
	for _, item := range []string{"用外部工具打开", "在文件管理器中显示", "另存为…"} {
		info, ok := tt.Access(item)
		if !ok || !info.Disabled {
			t.Fatalf("%q 没有置灰：%+v（找到 %v）", item, info, ok)
		}
	}
}

// 文件页的读取失败与首次加载：骨架、根级错误、根级「加载更多」。
func TestRightPanelFilesReadStates(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	app.right.tree = nil
	app.right.loading = true
	tt.Frame()
	if _, ok := tt.Find("正在加载文件树"); !ok {
		t.Fatal("首次加载没有骨架")
	}
	app.right.loading = false
	app.right.rootFailed = true
	tt.Frame()
	if _, ok := tt.Find("无法加载文件树。"); !ok {
		t.Fatal("根级读取失败没有错误行")
	}
	if _, ok := tt.Find("重试读取文件树"); !ok {
		t.Fatal("根级错误行没有重试")
	}
}

// 标签条：按住 ⌘ 显示 ⌘n 徽标；⌘2 切到第二个标签（RP-A-22）。
func TestRightPanelTabShortcuts(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	app.rp().tabs = []int{tabFiles, tabChanges}
	tt.Frame()
	tt.HoldModifiers(gui.Cmd)
	tt.Frame()
	for _, badge := range []string{"⌘1", "⌘2"} {
		if _, ok := tt.Find(badge); !ok {
			t.Fatalf("按住 ⌘ 时标签上没有 %q 徽标", badge)
		}
	}
	tt.Key(gui.Cmd, gui.Key2)
	tt.Frame()
	if app.rightTab != tabChanges {
		t.Fatalf("⌘2 后当前页 = %d，想要 %d", app.rightTab, tabChanges)
	}
}

// 标签条：悬停标签出 ×；点 × 关掉最后一页后右栏自动收起（R-TB-029）。
func TestRightPanelTabClose(t *testing.T) {
	app, tt := rightTester(t, tabFiles)
	tab, ok := tt.Find("文件")
	if !ok {
		t.Fatal("找不到文件标签")
	}
	tt.Move(tab.X+tab.W/2, tab.Y+tab.H/2)
	tt.Frame()
	tt.Frame()
	closeBox, ok := tt.Find("关闭标签页：文件")
	if !ok {
		t.Fatal("悬停标签没有出现关闭入口")
	}
	tt.ClickAt(closeBox.X+closeBox.W/2, closeBox.Y+closeBox.H/2)
	tt.Frame()
	if app.rightOpen {
		t.Fatal("关掉最后一页后右栏没有收起")
	}
}

// 标签条：展开面板（Review 页）占满主区，恢复后回到并排；拖拽手柄改宽度。
func TestRightPanelExpandAndResize(t *testing.T) {
	app, tt := rightTester(t, tabChanges)
	before, ok := tt.Find("右栏")
	if !ok {
		t.Fatal("找不到右栏")
	}
	if err := tt.Click("展开面板"); err != nil {
		t.Fatalf("点不到展开面板：%v", err)
	}
	tt.Frame()
	if !app.right.expanded {
		t.Fatal("展开面板没有进入展开态")
	}
	if _, ok := tt.Find("对话流"); ok {
		t.Fatal("展开后主区仍然在场")
	}
	wide, ok := tt.Find("右栏")
	if !ok || wide.W <= before.W {
		t.Fatalf("展开后右栏宽 %v，展开前 %v", wide.W, before.W)
	}
	if err := tt.Click("恢复面板宽度"); err != nil {
		t.Fatalf("点不到恢复面板宽度：%v", err)
	}
	tt.Frame()
	if app.right.expanded {
		t.Fatal("恢复后仍是展开态")
	}
	if _, ok := tt.Find("对话流"); !ok {
		t.Fatal("恢复后主区没有回来")
	}
	// 拖拽手柄：按住往左拖 40，右栏变宽（下一帧生效）。
	handle, ok := tt.Access("拖拽调整面板宽度")
	if !ok {
		t.Fatal("找不到拖拽手柄")
	}
	x, y := handle.Bounds.X+handle.Bounds.W/2, handle.Bounds.Y+handle.Bounds.H/2
	tt.Press(x, y)
	tt.Move(x-40, y)
	tt.Release(x-40, y)
	tt.Frame()
	if app.right.width <= RightPanelPreferredWidth {
		t.Fatalf("拖拽后宽度 = %v，想要大于默认 %v", app.right.width, RightPanelPreferredWidth)
	}
}

// 改动页：范围栏、两组、行内 stats、底部统计、行内 diff、全部暂存、还原确认。
func TestRightPanelChangesPage(t *testing.T) {
	app, tt := rightTester(t, tabChanges)
	for _, text := range []string{"未提交改动", "已暂存 (1)", "全部取消暂存",
		"未暂存 (2)", "全部暂存", "3 个文件改动", "+4", "-1"} {
		if _, ok := tt.Find(text); !ok {
			t.Fatalf("改动页缺少 %q", text)
		}
	}
	// 行内 diff：点文件名展开。
	if err := tt.Click("composer.go"); err != nil {
		t.Fatalf("点不到改动行：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("diff --git a/internal/ui/composer.go b/internal/ui/composer.go"); !ok {
		t.Fatal("没有展开行内 diff")
	}
	// 行尾「暂存」：这一行的改动进入已暂存组（纯图标入口按可访问名点）。
	clickAccess(t, tt, "暂存")
	staged := 0
	for _, change := range app.right.changes {
		if change.staged {
			staged++
		}
	}
	if staged != 2 {
		t.Fatalf("暂存后已暂存 = %d，想要 2", staged)
	}
	// 还原确认：未暂存行的「还原」先出确认框，取消后不动数据。
	clickAccess(t, tt, "还原")
	if _, ok := tt.Find("确认还原未暂存改动？"); !ok {
		t.Fatal("没有还原确认框")
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatalf("点不到取消：%v", err)
	}
	tt.Frame()
	if app.right.revertPath != "" {
		t.Fatal("取消后确认框状态没有清掉")
	}
	// 全部暂存：未暂存组消失，已暂存 3。
	if err := tt.Click("全部暂存"); err != nil {
		t.Fatalf("点不到全部暂存：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("未暂存 (2)"); ok {
		t.Fatal("全部暂存后未暂存组还在")
	}
	if _, ok := tt.Find("已暂存 (3)"); !ok {
		t.Fatal("全部暂存后已暂存组没有变 3")
	}
}

// 改动页：范围切换的三条空态（当前分支 / Last Run），以及提交或推送菜单。
func TestRightPanelChangesScopes(t *testing.T) {
	app, tt := rightTester(t, tabChanges)
	// 范围下拉：三项，各带副行。
	if err := tt.Click("未提交改动"); err != nil {
		t.Fatalf("点不到范围下拉：%v", err)
	}
	tt.Frame()
	for _, label := range []string{"当前分支改动", "Last Run 改动", "分支与基线差异"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("范围菜单缺少 %q", label)
		}
	}
	if err := tt.Click("当前分支改动"); err != nil {
		t.Fatalf("点不到当前分支改动：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("当前分支没有改动"); !ok {
		t.Fatal("当前分支范围没有空态")
	}
	app.right.scope = scopeLastRun
	tt.Frame()
	if _, ok := tt.Find("没有 Last Run 改动"); !ok {
		t.Fatal("Last Run 范围没有空态")
	}
	app.right.lastRunUnavailable = true
	tt.Frame()
	if _, ok := tt.Find("Last Run 改动不可用"); !ok {
		t.Fatal("Last Run 读取失败没有覆盖文案")
	}
	app.right.lastRunUnavailable = false
	app.right.scope = scopeBranch
	app.right.gitFailed = true
	tt.Frame()
	if _, ok := tt.Find("Git 改动不可用"); !ok {
		t.Fatal("Git 读取失败没有覆盖文案")
	}
	app.right.gitFailed = false
	// 提交或推送：菜单里先分支，再提交 / 提交并推送 / 推送。
	app.right.scope = scopeUncommitted
	tt.Frame()
	if err := tt.Click("提交或推送"); err != nil {
		t.Fatalf("点不到提交或推送：%v", err)
	}
	tt.Frame()
	for _, label := range []string{"提交", "提交并推送", "推送"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("提交菜单缺少 %q", label)
		}
	}
	tt.CloseMenu()

	// 出现条件（D-08）：有会话 ∧ 范围不是 Last Run。没有会话时这一行只有刷新，
	// 与参考的空会话页（panel-review-git.png）一致。
	app.selectedSession = ""
	tt.Frame()
	if _, ok := tt.Find("提交或推送"); ok {
		t.Fatal("没有会话时不应出现提交或推送")
	}
}
