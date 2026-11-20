package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 右栏面板的细节行为：分组与分隔线、焦点组与键盘、空态与零计数、标签条、
// 行上的悬停 / 选中 / 双击。每条测试对应 requirements-panels.md 里的一条或
// 一组需求；证据（函数与行号）写在需求表的状态列里。

// panelSmallTree 把样例树换成两行：一行没有改动、一行有改动。行都在窗口内，
// 指针能点到（样例只是视图数据，不影响被测行为）。
func panelSmallTree(app *App) {
	s := app.rp()
	var changed *rpChange
	for _, change := range s.changes {
		if change.path == "internal/ui/composer.go" {
			changed = change
		}
	}
	s.tree = []*rpNode{
		{name: "AGENTS.md"},
		{name: "composer.go", change: changed},
	}
}

// TestPanelGroupsSkipEmptyAndCollapse：空组不占位（D-28 / D-50）、组收起时批量
// 按钮不画（D-31）。
func TestPanelGroupsSkipEmptyAndCollapse(t *testing.T) {
	app, tt := panelTester(t, tabChanges)
	tt.Frame()
	if !tt.HasText("已暂存 (1)") || !tt.HasText("未暂存 (2)") {
		t.Fatalf("两组应当都在场：%v", tt.Texts())
	}
	if !tt.HasText("全部暂存") {
		t.Fatal("展开的未暂存组应当有「全部暂存」")
	}
	// 全部改成未暂存：已暂存组整组消失，不留一个「已暂存 (0)」占位。
	for _, change := range app.rp().changes {
		change.staged = false
	}
	tt.Frame()
	if tt.HasText("已暂存 (0)") || tt.HasText("已暂存 (1)") || tt.HasText("已暂存") {
		t.Fatalf("没有已暂存条目时该组不该渲染：%v", tt.Texts())
	}
	if !tt.HasText("未暂存 (3)") {
		t.Fatalf("未暂存组计数应重算为 3：%v", tt.Texts())
	}
	// 收起未暂存组：批量按钮随组一起不画，只剩标题与计数。
	if err := tt.Click("未暂存 (3)，切换展开"); err != nil {
		t.Fatalf("点不到分组标题：%v", err)
	}
	tt.Frame()
	if tt.HasText("全部暂存") {
		t.Fatal("组收起后批量按钮不该在场")
	}
	if _, ok := tt.Access("文件，internal/ui/composer.go，新增 2 行，删除 1 行，未暂存"); ok {
		t.Fatal("组收起后行不该在场")
	}
}

// TestPanelTreeIsOneFocusGroup：文件树是一个焦点组——Tab 进一次，↑/↓ 逐行移动，
// Home/End 到首尾（F-27 键盘）。
func TestPanelTreeIsOneFocusGroup(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	panelSmallTree(app)
	tt.Frame()
	reached := false
	for i := 0; i < 80; i++ {
		tt.Key(0, gui.KeyTab)
		if tt.Focused("文件，AGENTS.md") {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatalf("Tab 没有把焦点带进文件树的第一行：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyDown)
	if !tt.Focused("文件，composer.go，新增 2 行，删除 1 行，未暂存") {
		t.Fatalf("↓ 没有移动到下一行：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyHome)
	if !tt.Focused("文件，AGENTS.md") {
		t.Fatalf("Home 没有回到第一行：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEnd)
	if !tt.Focused("文件，composer.go，新增 2 行，删除 1 行，未暂存") {
		t.Fatalf("End 没有到最后一个可见行：%v", tt.Texts())
	}
}

// TestPanelTreeEnterActivatesRow：焦点在目录行上按 Enter 等同点击——展开该目录
// （F-18 键盘 / F-27 键盘）。
func TestPanelTreeEnterActivatesRow(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	app.rp().tree = app.rp().tree[:1] // 只剩第一个目录 assets
	tt.Frame()
	for i := 0; i < 80; i++ {
		tt.Key(0, gui.KeyTab)
		if tt.Focused("目录，assets，已收起") {
			break
		}
	}
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if !app.rp().dirOpen["assets"] {
		t.Fatal("Enter 没有展开聚焦的目录")
	}
	if !tt.HasText("目录，assets，已展开") {
		t.Fatalf("展开后读屏名应变成「已展开」：%v", tt.Texts())
	}
}

// TestPanelFileRowSelectsOnClick：文件行单击选中并把选中态交给读屏；预览页同时
// 打开该文件（F-27 交互 / 行选中态）。
func TestPanelFileRowSelectsOnClick(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	panelSmallTree(app)
	tt.Frame()
	if err := tt.Click("文件，AGENTS.md"); err != nil {
		t.Fatalf("点不到文件行：%v", err)
	}
	tt.Frame()
	if app.rp().selPath != "AGENTS.md" {
		t.Fatalf("单击没有选中该行：%q", app.rp().selPath)
	}
	if app.rightTab != tabFilePreview {
		t.Fatalf("单击文件行没有打开预览：tab=%d", app.rightTab)
	}
}

// TestPanelPreviewEscReturnsToFiles：预览页按 Esc 回到文件页（与「返回」同一
// 结果，焦点不在输入框时）。
func TestPanelPreviewEscReturnsToFiles(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	panelSmallTree(app)
	tt.Frame()
	if err := tt.Click("文件，AGENTS.md"); err != nil {
		t.Fatalf("点不到文件行：%v", err)
	}
	tt.Frame()
	if app.rightTab != tabFilePreview {
		t.Fatalf("没有进预览页：tab=%d", app.rightTab)
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.rightTab != tabFiles {
		t.Fatalf("Esc 没有回到文件页：tab=%d", app.rightTab)
	}
	if !app.rightOpen {
		t.Fatal("Esc 只回上一页，不该把右栏收起")
	}
}

// TestPanelFileMenuHasNoOrphanSeparator：右键菜单的分隔线只画在两组之间；没有
// 改动项时「预览」是菜单第一项，前面不留一条孤立线（F-36 异常 / F-38 / F-43）。
func TestPanelFileMenuHasNoOrphanSeparator(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	panelSmallTree(app)
	tt.Frame()

	base := menuSeparators(tt)
	if err := tt.RightClick("文件，AGENTS.md"); err != nil {
		t.Fatalf("右键没有改动的文件：%v", err)
	}
	tt.Frame()
	if !tt.HasText("预览") {
		t.Fatal("菜单里没有「预览」")
	}
	// 没有改动项时菜单从「预览」开始，只有一条分隔线（打开组｜复制组）。
	if got, want := menuSeparators(tt)-base, 1; got != want {
		t.Fatalf("没有改动项时应有 %d 条分隔线，实际 %d", want, got)
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()

	if err := tt.RightClick("文件，composer.go，新增 2 行，删除 1 行，未暂存"); err != nil {
		t.Fatalf("右键有改动的文件：%v", err)
	}
	tt.Frame()
	if !tt.HasText("暂存") {
		t.Fatal("菜单里没有「暂存」")
	}
	// 有改动项时四条分隔线：暂存｜差异｜打开｜复制。
	if got, want := menuSeparators(tt)-base, 3; got != want {
		t.Fatalf("有改动时应有 %d 条分隔线，实际 %d", want, got)
	}
}

// TestPanelTabBarArrowsMoveFocus：标签条是一个横向焦点组——Tab 进一次，←/→ 与
// Home/End 在页签之间移动，Enter 切页。
func TestPanelTabBarArrowsMoveFocus(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	app.rpOpen(tabChanges)
	tt.Frame()
	reached := false
	for i := 0; i < 80; i++ {
		tt.Key(0, gui.KeyTab)
		if tt.Focused("Review") {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatalf("Tab 没有把焦点带到当前页签：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyLeft)
	if !tt.Focused("文件") {
		t.Fatalf("← 没有移到前一个页签：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEnd)
	if !tt.Focused("Review") {
		t.Fatalf("End 没有到最后一个页签：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.rightTab != tabChanges {
		t.Fatalf("Enter 没有切到聚焦的页签：tab=%d", app.rightTab)
	}
}

// menuSeparators 数屏上「分隔线」标签的个数：右栏菜单的分隔条是带读屏名的
// 1px 面差（rpMenuSeparator），标签恒为「分隔线」，与菜单是否展开无关的基线
// 由调用方先减掉。
func menuSeparators(tt *gui.Tester) int {
	n := 0
	for _, text := range tt.Texts() {
		if text == "分隔线" {
			n++
		}
	}
	return n
}

// TestPanelChangesFilterNoMatch：改动页筛选 0 条时给「没有匹配的改动」，不说
// 「工作区是干净的」；底部统计仍按全量算（R-RD-175 / R-RD-741）。
func TestPanelChangesFilterNoMatch(t *testing.T) {
	app, tt := panelTester(t, tabChanges)
	app.rp().filter = "没有这个路径"
	tt.Frame()
	if !tt.HasText("没有匹配的改动") {
		t.Fatalf("筛选 0 条时应给「没有匹配的改动」：%v", tt.Texts())
	}
	if tt.HasText("没有未提交的改动") || tt.HasText("工作区是干净的") {
		t.Fatalf("筛选 0 条不该说工作区干净：%v", tt.Texts())
	}
	if !tt.HasText("3 个文件改动") {
		t.Fatalf("底部统计应按全量算：%v", tt.Texts())
	}
	// 命中一条：另一条不显示，计数按筛选后算。
	app.rp().filter = "composer"
	tt.Frame()
	if tt.HasText("没有匹配的改动") {
		t.Fatal("有命中时不该给空态")
	}
	if !tt.HasText("未暂存 (1)") {
		t.Fatalf("筛选后组计数应按命中算：%v", tt.Texts())
	}
	if !tt.HasText("3 个文件改动") {
		t.Fatalf("筛选不改底部统计：%v", tt.Texts())
	}
}

// TestPanelChangesScopeSwitchClearsFilter：切范围后筛选框清空（R-RD-006 /
// R-RD-020）。
func TestPanelChangesScopeSwitchClearsFilter(t *testing.T) {
	app, tt := panelTester(t, tabChanges)
	app.rp().filter = "composer"
	tt.Frame()
	if err := tt.Click("改动范围：未提交改动"); err != nil {
		t.Fatalf("点不到范围触发器：%v", err)
	}
	tt.Frame()
	if err := tt.Click("当前分支改动"); err != nil {
		t.Fatalf("点不到范围项：%v", err)
	}
	tt.Frame()
	if app.rp().filter != "" {
		t.Fatalf("切范围没有清空筛选词：%q", app.rp().filter)
	}
}

// TestPanelFileTreePartialWithoutEmpty：空树但根还有未读条目时只有「加载更多」，
// 不同时画空态（F-16 / R-RF-130）。
func TestPanelFileTreePartialWithoutEmpty(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	s := app.rp()
	s.tree = nil
	s.rootPartial = true
	tt.Frame()
	if !tt.HasText("加载更多") {
		t.Fatalf("空树且有未读条目时应给「加载更多」：%v", tt.Texts())
	}
	if tt.HasText("这个文件夹是空的") || tt.HasText("没有文件") {
		t.Fatalf("还有未读条目时不该给空态：%v", tt.Texts())
	}
}

// TestPanelFileSearchEmptyStateOnce：查询为空时永不进「无匹配文件」；树为空、
// 查询为空时只有一处空态（F-10 / F-11 / F-16）。
func TestPanelFileSearchEmptyStateOnce(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	s := app.rp()
	s.tree = nil
	tt.Frame()
	if tt.HasText("无匹配文件") {
		t.Fatalf("空查询不该进搜索空态：%v", tt.Texts())
	}
	if n := panelCountText(tt, "这个文件夹是空的"); n != 1 {
		t.Fatalf("空树应只有一处空态，实际 %d：%v", n, tt.Texts())
	}
	s.filter = "zzz"
	tt.Frame()
	if n := panelCountText(tt, "无匹配文件"); n != 1 {
		t.Fatalf("搜索无命中应只有一处空态，实际 %d：%v", n, tt.Texts())
	}
	if tt.HasText("这个文件夹是空的") {
		t.Fatal("搜索态不该同时画空树空态")
	}
}

// panelCountText 数屏上某段文字出现次数。
func panelCountText(tt *gui.Tester, text string) int {
	n := 0
	for _, s := range tt.Texts() {
		if s == text {
			n++
		}
	}
	return n
}

// TestPanelChangeRowDoubleClickOpensPreview：改动行双击打开预览；单击只展开
// diff（D-33 / 行上的双击）。
func TestPanelChangeRowDoubleClickOpensPreview(t *testing.T) {
	app, tt := panelTester(t, tabChanges)
	tt.Frame()
	row, ok := tt.Find("composer.go")
	if !ok {
		t.Fatalf("找不到改动行：%v", tt.Texts())
	}
	x, y := row.X+row.W/2, row.Y+row.H/2
	tt.ClickAt(x, y)
	tt.Frame()
	if app.rightTab != tabChanges {
		t.Fatalf("单击就跳走了：tab=%d", app.rightTab)
	}
	if !app.rp().diffOpen["internal/ui/composer.go"] {
		t.Fatal("单击没有展开行内 diff")
	}
	tt.ClickAt(x, y)
	tt.Frame()
	tt.ClickAt(x, y)
	tt.Frame()
	if app.rightTab != tabFilePreview {
		t.Fatalf("双击没有打开预览：tab=%d", app.rightTab)
	}
}

// TestPanelPreviewTabTooltipIsFullPath：预览页签的悬停提示给完整相对路径（标签
// 文案只有文件名，且会截断）。
func TestPanelPreviewTabTooltipIsFullPath(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	panelSmallTree(app)
	tt.Frame()
	if err := tt.Click("文件，AGENTS.md"); err != nil {
		t.Fatalf("点不到文件行：%v", err)
	}
	tt.Frame()
	info, ok := tt.Access("AGENTS.md")
	if !ok {
		t.Fatalf("找不到预览页签：%v", tt.Texts())
	}
	if info.Description != "AGENTS.md" {
		t.Fatalf("预览页签提示 = %q，想要完整路径 %q", info.Description, "AGENTS.md")
	}
}

// TestPanelTabCloseHitArea：标签的关闭入口命中区 16×16，图形仍按 14 画。
func TestPanelTabCloseHitArea(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	tt.Frame()
	tab, ok := tt.Find("文件")
	if !ok {
		t.Fatal("找不到文件标签")
	}
	tt.Move(tab.X+tab.W/2, tab.Y+tab.H/2)
	tt.Frame()
	tt.Frame()
	closeBox, ok := tt.Find("关闭标签页：文件")
	if !ok {
		t.Fatalf("悬停标签没有出现关闭入口：%v", tt.Texts())
	}
	if closeBox.W != panelTabClose || closeBox.H != panelTabClose {
		t.Fatalf("关闭入口命中区 = %.0f×%.0f，想要 %d×%d",
			closeBox.W, closeBox.H, panelTabClose, panelTabClose)
	}
	// 点命中区边缘（图形之外 1px 内）也能关掉：最后一页关完右栏收起。
	x := closeBox.X + closeBox.W - 1
	y := closeBox.Y + closeBox.H/2
	tt.ClickAt(x, y)
	tt.Frame()
	if app.rightOpen {
		t.Fatal("点关闭入口的边缘没有关掉最后一页")
	}
}

// TestPanelFooterIsNotFocusable：底部统计是纯文本，Tab 跳过它（D-77 键盘）。
func TestPanelFooterIsNotFocusable(t *testing.T) {
	_, tt := panelTester(t, tabChanges)
	tt.Frame()
	for i := 0; i < 90; i++ {
		tt.Key(0, gui.KeyTab)
		if tt.Focused("3 个文件改动") {
			t.Fatal("底部统计不该拿到键盘焦点")
		}
	}
}
