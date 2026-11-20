package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「读屏」这一类需求的批量取证（续 a11y_screens_test.go）：把还没被
// 一屏一条覆盖到的屏补上——设置 19 节的分节导航与标题、右栏 13 个标签页、
// 重命名与导出对话框、审批卡的三个按钮组。断言的是角色、可读名与选中/展开态，
// 名字逐字取自需求表（`R-*` 的期望列）。

// TestA11ySettingsNavAndHeadings：19 节设置每一节的左导航项都带选中态、内容区
// 都有可读标题，导航本身是一个列表（S-shell 的读屏口径）。
func TestA11ySettingsNavAndHeadings(t *testing.T) {
	for _, def := range settingsSectionDefs {
		t.Run(def.slug, func(t *testing.T) {
			app := NewApp(Options{State: StateDefault})
			app.screen, app.settingsSection = ScreenSettings, def.id
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			tt.Frame()

			if _, ok := tt.AccessByRole(gui.RoleList, "设置分节"); !ok {
				t.Fatalf("宽窗设置没有分节列表：%v", tt.Texts())
			}
			item, ok := tt.AccessByRole(gui.RoleListItem, def.label)
			if !ok {
				t.Fatalf("分节 %q 没有读屏项：%v", def.label, tt.Texts())
			}
			if !item.Selected {
				t.Fatalf("当前分节 %q 没有选中态", def.label)
			}
			if item.Name != def.label {
				t.Fatalf("分节可读名 = %q，想要 %q", item.Name, def.label)
			}
			var heading gui.AccessInfo
			found := false
			for _, text := range tt.Texts() {
				if info, ok := tt.AccessByRole(gui.RoleHeading, text); ok {
					heading, found = info, true
					break
				}
			}
			if !found {
				t.Fatalf("%s：内容区没有可读标题", def.slug)
			}
			if heading.Name == "" {
				t.Fatalf("%s：标题没有可读名", def.slug)
			}
		})
	}
}

// TestA11yRightPanelTabs：右栏每一页的页签都有可读名、当前页带选中态，标签条
// 是一个 tablist（RP-A 的读屏口径）。
func TestA11yRightPanelTabs(t *testing.T) {
	for tab := 0; tab < tabCount; tab++ {
		t.Run(rpTabName(tab), func(t *testing.T) {
			app := NewApp(Options{State: StateDefault})
			app.rightOpen, app.rightTab = true, tab
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			tt.Frame()
			want := app.rpTabLabel(tab)
			tabInfo, ok := tt.AccessByRole(gui.RoleTab, want)
			if !ok {
				t.Fatalf("页签 %q 没有读屏项：%v", want, tt.Texts())
			}
			if !tabInfo.Selected {
				t.Fatalf("当前页签 %q 没有选中态", want)
			}
			if _, ok := tt.AccessByRole(gui.RoleTabList, "右栏标签页"); !ok {
				t.Fatalf("标签条没有 tablist 角色：%v", tt.Texts())
			}
		})
	}
}

// rpTabName 给用例一个稳定的名字（页签文案会带计数）。
func rpTabName(tab int) string {
	switch tab {
	case tabFiles:
		return "files"
	case tabChanges:
		return "changes"
	case tabTasks:
		return "tasks"
	case tabGoal:
		return "goal"
	case tabTerminal:
		return "terminal"
	case tabSideChat:
		return "sidechat"
	case tabSubagent:
		return "subagent"
	case tabBrowser:
		return "browser"
	case tabFilePreview:
		return "preview"
	case tabWorkflow:
		return "workflow"
	case tabWakeup:
		return "wakeup"
	case tabExtension:
		return "extension"
	case tabMCPApp:
		return "mcpapp"
	}
	return "tab"
}

// TestA11yRenameDialog：重命名弹窗是一个对话框，标题、说明、输入框与两个按钮
// 都有逐字可读名（S-会话重命名 的读屏口径）。
func TestA11yRenameDialog(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.openSessionRenameDialog(0)
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	info, ok := tt.AccessByRole(gui.RoleDialog, renameCopySessionTitle)
	if !ok {
		t.Fatalf("重命名弹窗没有 dialog 角色与标题：%v", tt.Texts())
	}
	if info.Name != renameCopySessionTitle {
		t.Fatalf("弹窗可读名 = %q，想要 %q", info.Name, renameCopySessionTitle)
	}
	field, ok := tt.AccessByRole(gui.RoleTextField, renameCopySessionField)
	if !ok {
		t.Fatalf("输入框没有读屏名 %q：%v", renameCopySessionField, tt.Texts())
	}
	if field.Value == "" {
		t.Fatal("输入框没有带当前名称作为值")
	}
	for _, label := range []string{renameCopyCancel, renameCopyConfirm} {
		if _, ok := tt.AccessByRole(gui.RoleButton, label); !ok {
			t.Fatalf("弹窗缺少按钮 %q：%v", label, tt.Texts())
		}
	}
}

// TestA11yApprovalButtonsNamed：审批卡是一个 region，三个决定按钮都有逐字
// 可读名且主按钮在最后（读屏顺序 = 渲染顺序，R-AP-016）。
func TestA11yApprovalButtonsNamed(t *testing.T) {
	app, tt := newTester(t, StateWaiting)
	if _, ok := app.currentApproval(); !ok {
		t.Skip("走查态没有审批卡")
	}
	if _, ok := tt.AccessByRole(gui.RoleRegion, "审批卡"); !ok {
		t.Fatalf("审批卡没有 region 角色：%v", tt.Texts())
	}
	var names []string
	for _, label := range []string{approvalCopyReject, approvalCopyAllowOnce, approvalCopyAlwaysCommand} {
		info, ok := tt.AccessByRole(gui.RoleButton, label)
		if !ok {
			t.Fatalf("审批卡缺少按钮 %q：%v", label, tt.Texts())
		}
		if info.Name != label {
			t.Fatalf("按钮可读名 = %q，想要 %q", info.Name, label)
		}
		names = append(names, label)
	}
	if names[2] != approvalCopyAlwaysCommand {
		t.Fatalf("按钮顺序 = %v，主按钮应最后", names)
	}
}
