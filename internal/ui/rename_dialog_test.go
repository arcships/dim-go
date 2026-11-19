package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是重命名弹窗（§6）的回归：入口、四条校验、Esc / 回车、失败原因，以及
// 行内快捷编辑仍然保留。

// renameTestIntents 记录一次重命名；fail 非空时返回失败原因。
type renameTestIntents struct {
	Intents
	renamed chan [2]string
	fail    error
}

func (r *renameTestIntents) RenameSession(_ context.Context, sessionID, title string) error {
	if r.fail != nil {
		return r.fail
	}
	if r.renamed != nil {
		r.renamed <- [2]string{sessionID, title}
	}
	return nil
}

// openRenameFromHeader 打开标题栏 ⋯ 菜单并点「重命名」。
func openRenameFromHeader(t *testing.T, tt *gui.Tester) {
	t.Helper()
	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯ 菜单：%v", err)
	}
	tt.Frame()
	if err := tt.Click("重命名"); err != nil {
		t.Fatalf("点不到重命名：%v", err)
	}
	tt.Frame()
}

// TestRenameDialogValidation：四条校验按固定顺序取第一条，任一成立时「确认」禁用。
func TestRenameDialogValidation(t *testing.T) {
	cases := []struct {
		name string
		set  func(st *renameDialogState)
		hint string
	}{
		{"超长", func(st *renameDialogState) { st.draft = strings.Repeat("字", 81) }, renameCopyTooLong},
		{"与原名相同", func(st *renameDialogState) { st.draft = st.original }, renameCopySameSession},
		{"控制字符", func(st *renameDialogState) { st.draft = "会话\n名" }, renameCopyCtrlSession},
		{"去空白后为空", func(st *renameDialogState) { st.draft = "   " }, renameCopyEmptySession},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := NewApp(Options{State: StateDefault, Source: wiredSource()})
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			openRenameFromHeader(t, tt)
			if !app.renameDialog.open {
				t.Fatal("重命名弹窗没有打开")
			}
			tc.set(&app.renameDialog)
			tt.Frame()
			if !tt.HasText(tc.hint) {
				t.Fatalf("没有显示校验提示 %q：%v", tc.hint, tt.Texts())
			}
			if info, ok := tt.Access("确认"); !ok || !info.Disabled {
				t.Fatalf("校验未通过时「确认」没有禁用：%+v（找到 %v）", info, ok)
			}
		})
	}
}

// TestRenameDialogEnterSavesAndEscCancels：回车保存、Esc 取消。
func TestRenameDialogEnterSavesAndEscCancels(t *testing.T) {
	t.Run("回车保存", func(t *testing.T) {
		intents := &renameTestIntents{renamed: make(chan [2]string, 1)}
		app := NewApp(Options{State: StateDefault, Source: wiredSource(), Intents: intents})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		openRenameFromHeader(t, tt)
		app.renameDialog.draft = "改名之后"
		tt.Frame()
		tt.Key(0, gui.KeyEnter)
		select {
		case got := <-intents.renamed:
			if got[0] != "s1" || got[1] != "改名之后" {
				t.Fatalf("RenameSession 收到 %v", got)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("回车没有发出 RenameSession")
		}
	})

	t.Run("Esc 取消", func(t *testing.T) {
		app := NewApp(Options{State: StateDefault, Source: wiredSource()})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		openRenameFromHeader(t, tt)
		tt.Key(0, gui.KeyEscape)
		tt.Frame()
		if app.renameDialog.open {
			t.Fatal("Esc 没有关闭重命名弹窗")
		}
	})
}

// TestRenameDialogShowsFailureReason：写入失败时弹窗保留，原因显示在弹窗里。
func TestRenameDialogShowsFailureReason(t *testing.T) {
	intents := &renameTestIntents{fail: errors.New("服务端拒绝了这次改名")}
	app := NewApp(Options{State: StateDefault, Source: wiredSource(), Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openRenameFromHeader(t, tt)
	app.renameDialog.draft = "另一个名字"
	tt.Frame()
	if err := tt.Click("确认"); err != nil {
		t.Fatalf("点不到确认：%v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if strings.Contains(app.renameDialog.err, "服务端拒绝") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !strings.Contains(app.renameDialog.err, "服务端拒绝") {
		t.Fatalf("失败原因没有回到弹窗：%q", app.renameDialog.err)
	}
	if !app.renameDialog.open {
		t.Fatal("失败后弹窗被关掉了")
	}
	if !tt.HasText("重命名：服务端拒绝了这次改名") {
		t.Fatalf("失败原因没有上屏：%v", tt.Texts())
	}
}

// TestWorkspaceRenameDialogShowsPathAndSaves：工作区弹窗有标题 / 描述 / 路径行，
// 确认后写显示名并落到视图状态。
func TestWorkspaceRenameDialogShowsPathAndSaves(t *testing.T) {
	app := cwdGroupApp(cwdGroupPlaces())
	ws := app.sidebarWorkspaces(app.sidebarThreads())[1]
	app.openWorkspaceRenameDialog(ws)
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	for _, want := range []string{"重命名工作区", "输入新的工作区名称", "工作区名称", "/work/dim-agent"} {
		if !tt.HasText(want) {
			t.Fatalf("工作区弹窗缺少 %q：%v", want, tt.Texts())
		}
	}
	app.renameDialog.draft = "评审目录"
	tt.Frame()
	if err := tt.Click("确认"); err != nil {
		t.Fatalf("点不到确认：%v", err)
	}
	tt.Frame()
	if app.renameDialog.open {
		t.Fatal("保存后弹窗没有关闭")
	}
	if got := app.sidebarView.groupNames[ws.id]; got != "评审目录" {
		t.Fatalf("显示名没有写入视图状态：%q", got)
	}
}

// TestInlineRenameStillReachable：弹窗之外，行内快捷改名仍能用 F2 进入。
func TestInlineRenameStillReachable(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Key(0, gui.KeyF2)
	tt.Frame()
	if app.renaming != 0 {
		t.Fatalf("F2 没有进入行内编辑（renaming=%d）", app.renaming)
	}
	if !tt.HasText("重命名会话") {
		t.Fatalf("行内编辑行没有名字：%v", tt.Texts())
	}
}
