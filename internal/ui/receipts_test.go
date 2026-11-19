package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「失败 / 不可用 / 回执」这一类的测试：错误行带重试、复制回执来自真实
// 剪贴板、空名回落、工作区名不把样例名漏到真实会话上。

// flakyRenameIntents 的 RenameSession 先失败一次再成功，用来验证错误行的「重试」
// 重发同一次写入。
type flakyRenameIntents struct {
	Intents
	calls chan string
	fail  bool
}

func (f *flakyRenameIntents) RenameSession(_ context.Context, sessionID, title string) error {
	f.calls <- title
	if f.fail {
		f.fail = false
		return errors.New("目录拒绝写入")
	}
	return nil
}

// TestIntentErrorRowOffersRetry：写入失败 → 错误行出现「重试」；点重试再发一次
// 同一个写入，成功后错误行与重试入口一起消失。
func TestIntentErrorRowOffersRetry(t *testing.T) {
	src := wiredSource()
	stub := &flakyRenameIntents{calls: make(chan string, 4), fail: true}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	app.renaming, app.renameDraft = 0, "新名字"
	tt.Frame()
	if err := tt.Click("保存名称"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	select {
	case got := <-stub.calls:
		if got != "新名字" {
			t.Fatalf("RenameSession 收到 %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有发 RenameSession")
	}
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("重试未完成的动作"); !ok {
		t.Fatalf("失败后没有重试入口：%v", tt.Texts())
	}
	if !tt.HasText("重命名：目录拒绝写入") {
		t.Fatalf("错误行没有写失败原因：%v", tt.Texts())
	}

	if err := tt.Click("重试未完成的动作"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	select {
	case <-stub.calls:
	case <-time.After(2 * time.Second):
		t.Fatal("重试没有再发写入")
	}
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("重试未完成的动作"); ok {
		t.Fatal("成功后重试入口还在")
	}
}

// TestCopySessionIDWritesClipboardAndShowsReceipt：⋯ 菜单的「复制会话 ID」把真实
// 标识写进剪贴板，并在标题下给回执（不再只写一句没发生的「已复制」）。
func TestCopySessionIDWritesClipboardAndShowsReceipt(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯ 菜单：%v", err)
	}
	tt.Frame()
	if err := tt.Click("复制会话 ID"); err != nil {
		t.Fatalf("点不到复制会话 ID：%v", err)
	}
	if got := tt.Clipboard(); got != "s1" {
		t.Fatalf("剪贴板 = %q，想要 s1", got)
	}
	tt.Frame()
	if !tt.HasText("已复制会话 ID") {
		t.Fatalf("没有回执行：%v", tt.Texts())
	}
	if !tt.HasText("操作回执") {
		t.Fatal("回执行没有读屏名")
	}
}

// TestCopyFailureDoesNotClaimSuccess：剪贴板写失败时写失败原因，不显示「已复制」。
func TestCopyFailureDoesNotClaimSuccess(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.FailClipboard(errors.New("剪贴板被占用"))

	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯ 菜单：%v", err)
	}
	tt.Frame()
	if err := tt.Click("复制调试信息"); err != nil {
		t.Fatalf("点不到复制调试信息：%v", err)
	}
	tt.Frame()
	tt.Frame()
	if tt.HasText("已复制调试信息") {
		t.Fatalf("写失败却给了「已复制」：%v", tt.Texts())
	}
	if !tt.HasText("复制失败：剪贴板被占用") {
		t.Fatalf("没有写失败原因：%v", tt.Texts())
	}
}

// TestRenameWithEmptyNameFallsBack：会话改名留空 → 退出编辑并回落原名称，不存在
// 空标题。
func TestRenameWithEmptyNameFallsBack(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	name := app.sessionName(0, app.sessionTitle())
	app.renaming, app.renameDraft = 0, "   "
	tt.Frame()
	if err := tt.Click("保存名称"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if app.renaming != -1 {
		t.Fatal("空名保存后仍停在编辑态")
	}
	if got := app.sessionName(0, app.sessionTitle()); got != name {
		t.Fatalf("空名保存后名称 = %q，想要原名称 %q", got, name)
	}
}

// TestWiredWorkspaceTitleFallsBackToBuiltin：接上事实而会话快照没有工作区名时，
// 标题栏用内置工作区名，不把样例目录名漏上去（R-TB-145）。
func TestWiredWorkspaceTitleFallsBackToBuiltin(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	if got := app.workspaceTitle(); got != sidebarBuiltinName {
		t.Fatalf("工作区名 = %q，想要 %q", got, sidebarBuiltinName)
	}
	app.workspaceName = "真实工作区"
	if got := app.workspaceTitle(); got != "真实工作区" {
		t.Fatalf("已给的工作区名被覆盖成 %q", got)
	}
	if strings.Contains(app.debugInfo(), "apiKey") {
		t.Fatal("调试信息里不该有凭据字段")
	}
}
