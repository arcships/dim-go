package ui

import (
	"fmt"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「读屏标注」的批量化验收：一屏一条用例，逐项断言可读名、角色与
// 状态。名字为空、角色不对、选中/展开态没落到元素上，都在这里报出来。
//
// 每个用例先按屏的打开方式造界面，再走 items；items 里列的是这一屏「读屏
// 必须有」的项。RoleAuto 表示不限定角色（按钮、文本框这类由控件自己给）。

// a11yItem 是一条读屏期望。
type a11yItem struct {
	role gui.Role
	name string
	// selected / expanded 为真时断言元素带选中态 / 展开态。
	selected, expanded bool
}

// a11yScreen 是一屏的读屏期望。
type a11yScreen struct {
	name  string
	open  func(t *testing.T) (*App, *gui.Tester)
	items []a11yItem
}

// a11yMainScreen 造主界面（可带状态与变体）。
func a11yMainScreen(state State) func(t *testing.T) (*App, *gui.Tester) {
	return func(t *testing.T) (*App, *gui.Tester) {
		t.Helper()
		app := NewApp(Options{State: state})
		return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
	}
}

// a11yRightPanel 打开右栏的指定标签页。
func a11yRightPanel(tab int) func(t *testing.T) (*App, *gui.Tester) {
	return func(t *testing.T) (*App, *gui.Tester) {
		t.Helper()
		app := NewApp(Options{State: StateDefault})
		app.rightOpen, app.rightTab = true, tab
		return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
	}
}

// a11ySettings 打开设置屏的指定分节。
func a11ySettings(section settingsSectionID) func(t *testing.T) (*App, *gui.Tester) {
	return func(t *testing.T) (*App, *gui.Tester) {
		t.Helper()
		app := NewApp(Options{State: StateDefault})
		app.screen, app.settingsSection = ScreenSettings, section
		return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
	}
}

// a11yVariant 造一个走查态。
func a11yVariant(v Variant) func(t *testing.T) (*App, *gui.Tester) {
	return func(t *testing.T) (*App, *gui.Tester) {
		t.Helper()
		app := NewApp(Options{State: StateDefault, Variant: v})
		return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
	}
}

// a11yScreens 是批量验收的表。
func a11yScreens() []a11yScreen {
	return []a11yScreen{
		{"主界面·会话列表", a11yMainScreen(StateDefault), []a11yItem{
			{gui.RoleList, "会话列表", false, false},
			{gui.RoleListItem, "修发送路径", true, false},
			{gui.RoleListItem, "旧实现对照", false, false},
			{gui.RoleTreeItem, "工作区", false, false},
			{gui.RoleButton, "新建会话", false, false},
			{gui.RoleButton, "收起侧边栏 ⌘B", false, false},
		}},
		{"主界面·对话流与输入位", a11yMainScreen(StateDefault), []a11yItem{
			{gui.RoleLog, "对话流", false, false},
			{gui.RoleArticle, "用户消息 看看这次的失败", false, false},
			{gui.RoleArticle, "助手消息", false, false},
			{gui.RoleButton, "上下文用量 ~12%", false, false},
			{gui.RoleTextField, "消息", false, false},
		}},
		{"主界面·审批卡", a11yMainScreen(StateWaiting), []a11yItem{
			{gui.RoleRegion, "审批卡", false, false},
			{gui.RoleButton, "仅这次", false, false},
			{gui.RoleButton, "拒绝", false, false},
		}},
		{"右栏·文件树", a11yRightPanel(tabFiles), []a11yItem{
			{gui.RoleTabList, "右栏标签页", false, false},
			{gui.RoleTab, "文件", true, false},
			{gui.RoleTree, "文件树", false, false},
			{gui.RoleTreeItem, "目录，assets，已收起", false, false},
			{gui.RoleTreeItem, "文件，README.md", false, false},
		}},
		{"右栏·改动", a11yRightPanel(tabChanges), []a11yItem{
			{gui.RoleTab, "Review", true, false},
			{gui.RoleTreeItem, "", false, false},
			{gui.RoleListItem, "", false, false},
			{gui.RoleButton, "全部暂存", false, false},
		}},
		{"右栏·后台任务", func(t *testing.T) (*App, *gui.Tester) {
			t.Helper()
			app := NewApp(Options{State: StateDefault})
			app.rightOpen, app.rightTab = true, tabTasks
			app.panels.taskSel = 0
			return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
		}, []a11yItem{
			{gui.RoleList, "后台进程", false, false},
			{gui.RoleListItem, "后台任务：go test ./... · 运行中", true, false},
		}},
		{"右栏·终端", a11yRightPanel(tabTerminal), []a11yItem{
			{gui.RoleButton, "新建终端", false, false},
			{gui.RoleButton, "关闭终端", false, false},
			{gui.RoleButton, "清屏", false, false},
		}},
		{"右栏·侧边对话", a11yRightPanel(tabSideChat), []a11yItem{
			{gui.RoleTab, "侧边对话", true, false},
			{gui.RoleTextField, "侧边对话输入", false, false},
			{gui.RoleButton, "发送", false, false},
		}},
		{"设置·导航", a11ySettings(secAccount), []a11yItem{
			{gui.RoleList, "设置分节", false, false},
			{gui.RoleListItem, "账户", true, false},
			{gui.RoleListItem, "外观", false, false},
			{gui.RoleButton, "返回主界面", false, false},
		}},
		{"设置·账户", a11ySettings(secAccount), []a11yItem{
			{gui.RoleHeading, "账户", false, false},
			{gui.RoleButton, "刷新", false, false},
		}},
		{"设置·外观", a11ySettings(secAppearance), []a11yItem{
			{gui.RoleHeading, "外观", false, false},
			{gui.RoleRadio, "主题：浅色", false, false},
		}},
		{"设置·快捷键", a11ySettings(secKeybindings), []a11yItem{
			{gui.RoleHeading, "快捷键", false, false},
			{gui.RoleTextField, "搜索快捷键", false, false},
		}},
		{"设置·快捷键行", a11ySettings(secKeybindings), []a11yItem{
			{gui.RoleListItem, "", false, false},
		}},
		{"设置·Hooks 条目", a11ySettings(secHooks), []a11yItem{
			{gui.RoleListItem, "", false, false},
		}},
		{"设置·归档会话", a11ySettings(secArchived), []a11yItem{
			{gui.RoleListItem, "", false, false},
		}},
		{"设置·宠物", a11ySettings(secPets), []a11yItem{
			{gui.RoleList, "Pets", false, false},
			{gui.RoleListItem, "", false, false},
		}},
		{"设置·MCP", a11ySettings(secMCP), []a11yItem{
			{gui.RoleHeading, "MCP 服务器", false, false},
			{gui.RoleButton, "添加服务器", false, false},
		}},
		{"命令面板", a11yVariant(VariantPalette), []a11yItem{
			{gui.RoleDialog, "命令面板", false, false},
			{gui.RoleListBox, "命令候选", false, false},
			{gui.RoleTextField, "搜索命令", false, false},
		}},
		{"左栏搜索", a11yVariant(VariantSearch), []a11yItem{
			{gui.RoleTextField, "搜索会话", false, false},
			{gui.RoleButton, "搜索", false, false},
			{gui.RoleList, "会话列表", false, false},
		}},
		{"侧边栏收起", func(t *testing.T) (*App, *gui.Tester) {
			t.Helper()
			app := NewApp(Options{State: StateDefault})
			app.sidebarCollapsed = true
			return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
		}, []a11yItem{
			{gui.RoleButton, "展开侧边栏 ⌘B", false, false},
		}},
		{"删除会话确认", a11yVariant(VariantSessionDelete), []a11yItem{
			{gui.RoleAlertDialog, "删除会话", false, false},
			{gui.RoleButton, "取消", false, false},
		}},
		{"退出登录确认", func(t *testing.T) (*App, *gui.Tester) {
			t.Helper()
			app := NewApp(Options{State: StateDefault})
			app.logoutConfirm = true
			return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
		}, []a11yItem{
			{gui.RoleAlertDialog, "", false, false},
		}},
	}
}

// TestA11yScreens 是批量验收：一屏一条用例，逐项断言可读名、角色与状态。
func TestA11yScreens(t *testing.T) {
	for _, screen := range a11yScreens() {
		t.Run(screen.name, func(t *testing.T) {
			_, tt := screen.open(t)
			for _, item := range screen.items {
				if item.name == "" {
					// 只查角色：这一屏至少有一个该角色的读屏项。
					if _, ok := tt.AccessByRole(item.role, ""); !ok {
						t.Fatalf("缺少角色 %v 的读屏项", item.role)
					}
					continue
				}
				var (
					info gui.AccessInfo
					ok   bool
				)
				if item.role == gui.RoleAuto {
					info, ok = tt.Access(item.name)
				} else {
					info, ok = tt.AccessByRole(item.role, item.name)
				}
				if !ok {
					t.Fatalf("读屏项 %q（角色 %v）不存在", item.name, item.role)
				}
				if info.Name == "" {
					t.Fatalf("读屏项 %q 没有可读名", item.name)
				}
				if item.selected && !info.Selected && !info.Checked {
					t.Fatalf("%q 没有选中态", item.name)
				}
				if item.expanded && !info.Expanded {
					t.Fatalf("%q 没有展开态", item.name)
				}
			}
		})
	}
}

// TestFocusStopsHaveNames：Tab 走遍每一屏的焦点项，每一项都要有可读名——
// 读屏不会停在一个没有名字的控件上。
func TestFocusStopsHaveNames(t *testing.T) {
	type screen struct {
		name string
		open func(t *testing.T) (*App, *gui.Tester)
	}
	var screens []screen
	for _, state := range []State{StateDefault, StateRunning, StateWaiting, StateFailed, StateEmpty} {
		screens = append(screens, screen{fmt.Sprintf("主界面·状态 %d", state), a11yMainScreen(state)})
	}
	for _, tab := range []int{tabFiles, tabChanges, tabTasks, tabGoal, tabTerminal, tabSideChat, tabSubagent} {
		screens = append(screens, screen{fmt.Sprintf("右栏·标签 %d", tab), a11yRightPanel(tab)})
	}
	for _, def := range settingsSectionDefs {
		screens = append(screens, screen{"设置·" + def.slug, a11ySettings(def.id)})
	}
	for _, v := range []Variant{VariantPalette, VariantSearch, VariantSessionMenu, VariantSessionRename, VariantSessionDelete} {
		screens = append(screens, screen{fmt.Sprintf("走查态 %d", v), a11yVariant(v)})
	}
	for _, s := range screens {
		t.Run(s.name, func(t *testing.T) {
			_, tt := s.open(t)
			if tt.Menu() != nil {
				return // 原生菜单不在窗口的读屏树里，键盘走查无从谈起
			}
			seen := map[string]bool{}
			for i := 0; i < 80; i++ {
				tt.Key(0, gui.KeyTab)
				names := focusedNames(tt)
				if len(names) == 0 {
					t.Fatalf("第 %d 个焦点项没有可读名", i+1)
				}
				key := fmt.Sprint(names)
				if seen[key] && i > 0 {
					return // 回到起点：这一屏的焦点项走完了
				}
				seen[key] = true
			}
		})
	}
}

// focusedNames 是当前焦点项的可读名：焦点在某个元素上时，它自己或包着它的
// 容器报到名字；空表示焦点停在一个没有名字的元素上（读屏只会念「未知」）。
func focusedNames(tt *gui.Tester) []string {
	var out []string
	seen := map[string]bool{}
	for _, s := range tt.Texts() {
		if s == "" || seen[s] || !tt.Focused(s) {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
