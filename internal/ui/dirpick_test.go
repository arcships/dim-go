package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件的测试覆盖目录选择的三条结果（成功 / 取消 / 命令失败）与两处接线
// （左栏「添加工作区」、设置 › Hooks 的项目路径）。选择器一律注入执行器：
// 测试里不弹系统对话框，也不碰真实文件系统（目录用 t.TempDir）。

// dirPickIntents 只实现本地偏好这一条写入：其余方法留在嵌入的 nil 接口上，
// 被调到会 panic，说明测试点到了一个没接的动作。
type dirPickIntents struct {
	Intents
	prefs chan PreferencesSnapshot
}

func (d *dirPickIntents) SetPreferences(_ context.Context, prefs PreferencesSnapshot) error {
	d.prefs <- prefs
	return nil
}

// pickTester 建一个可注入选择器的 App 与它的 Tester：changed 是后台结果交回后的
// 重绘请求（与入口的 Options.Changed 同一条）。
func pickTester(t *testing.T, goos string, run func(context.Context, string, ...string) (string, error)) (*App, *gui.Tester, chan struct{}, chan PreferencesSnapshot) {
	t.Helper()
	changed := make(chan struct{}, 8)
	prefs := make(chan PreferencesSnapshot, 8)
	app := NewApp(Options{
		State: StateDefault, Source: wiredSource(),
		Intents: &dirPickIntents{prefs: prefs},
		Changed: func() {
			select {
			case changed <- struct{}{}:
			default:
			}
		},
	})
	app.dirPickGOOS = goos
	app.dirPickRunner = run
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight), changed, prefs
}

// pickedDir 造一个名字可辨认的目录（显示名取目录最后一段）。
func pickedDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("建目录：%v", err)
	}
	return dir
}

// waitChanged 等后台选择交回一次重绘请求。
func waitChanged(t *testing.T, changed chan struct{}, what string) {
	t.Helper()
	select {
	case <-changed:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s 没有交回结果", what)
	}
}

// waitPrefs 等一次本地偏好的写入。
func waitPrefs(t *testing.T, prefs chan PreferencesSnapshot, what string) PreferencesSnapshot {
	t.Helper()
	select {
	case got := <-prefs:
		return got
	case <-time.After(2 * time.Second):
		t.Fatalf("%s 没有落盘本地偏好", what)
		return PreferencesSnapshot{}
	}
}

// 选择成功：路径进本地偏好（左栏多一组），并落盘。
func TestAddWorkspacePicksDirectory(t *testing.T) {
	dir := pickedDir(t, "picked-workspace")
	app, tt, changed, prefs := pickTester(t, "darwin",
		func(context.Context, string, ...string) (string, error) {
			// osascript 的 POSIX path 带末尾分隔符。
			return dir + "/\n", nil
		})

	if err := tt.Click("添加工作区"); err != nil {
		t.Fatalf("点添加工作区：%v", err)
	}
	waitChanged(t, changed, "添加工作区")
	tt.Frame()

	if len(app.workspaces) != 1 || app.workspaces[0] != dir {
		t.Fatalf("登记的目录 = %v，想要 %q", app.workspaces, dir)
	}
	if !tt.HasText("picked-workspace") {
		t.Fatalf("左栏没有出现新工作区分组：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if app.addWorkspacePick.open || app.addWorkspacePick.failure != "" {
		t.Fatalf("成功后仍留着小框：%+v", app.addWorkspacePick)
	}
	got := waitPrefs(t, prefs, "添加工作区")
	if len(got.Workspaces) != 1 || got.Workspaces[0] != dir {
		t.Fatalf("落盘的目录 = %v，想要 %q", got.Workspaces, dir)
	}
}

// 取消：不改状态、不报错、不落盘。
func TestAddWorkspaceCancelKeepsState(t *testing.T) {
	app, tt, changed, prefs := pickTester(t, "darwin",
		func(context.Context, string, ...string) (string, error) {
			return "execution error: User canceled. (-128)", errors.New("exit status 1")
		})

	if err := tt.Click("添加工作区"); err != nil {
		t.Fatalf("点添加工作区：%v", err)
	}
	waitChanged(t, changed, "添加工作区")
	tt.Frame()

	if len(app.workspaces) != 0 {
		t.Fatalf("取消后登记的目录 = %v，想要空", app.workspaces)
	}
	if app.addWorkspacePick.open || app.addWorkspacePick.failure != "" || app.addWorkspacePick.busy {
		t.Fatalf("取消后界面状态 = %+v，想要原样", app.addWorkspacePick)
	}
	select {
	case got := <-prefs:
		t.Fatalf("取消却写了偏好：%+v", got)
	default:
	}
}

// 命令失败（无权限等）：留一句可读原因，不登记目录。
func TestAddWorkspaceCommandFailure(t *testing.T) {
	app, tt, changed, prefs := pickTester(t, "darwin",
		func(context.Context, string, ...string) (string, error) {
			return "execution error: Not authorized to send Apple events to System Events. (-1743)",
				errors.New("exit status 1")
		})

	if err := tt.Click("添加工作区"); err != nil {
		t.Fatalf("点添加工作区：%v", err)
	}
	waitChanged(t, changed, "添加工作区")
	tt.Frame()

	if len(app.workspaces) != 0 {
		t.Fatalf("失败后登记的目录 = %v，想要空", app.workspaces)
	}
	if !app.addWorkspacePick.open || app.addWorkspacePick.failure == "" {
		t.Fatalf("失败没有留在小框里：%+v", app.addWorkspacePick)
	}
	if !tt.HasText(app.addWorkspacePick.failure) {
		t.Fatalf("失败原因没有上屏：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	select {
	case got := <-prefs:
		t.Fatalf("失败却写了偏好：%+v", got)
	default:
	}
}

// 非 macOS：没有系统选择器，同一个入口改成手输路径。
func TestAddWorkspaceFallsBackToManualInput(t *testing.T) {
	dir := pickedDir(t, "manual-workspace")
	app, tt, _, _ := pickTester(t, "linux",
		func(context.Context, string, ...string) (string, error) {
			t.Fatal("非 macOS 不应执行选择命令")
			return "", nil
		})

	if err := tt.Click("添加工作区"); err != nil {
		t.Fatalf("点添加工作区：%v", err)
	}
	tt.Frame()
	if !app.addWorkspacePick.open || !app.addWorkspacePick.manual {
		t.Fatalf("非 macOS 没有开手输框：%+v", app.addWorkspacePick)
	}
	if !tt.HasText(dirPickCopyManualHint) {
		t.Fatalf("手输框没有说明：\n%s", strings.Join(tt.Texts(), "\n"))
	}

	app.addWorkspacePick.draft = dir
	tt.Frame()
	if err := tt.Click(dirPickCopyConfirmLabel); err != nil {
		t.Fatalf("点使用这个目录：%v", err)
	}
	tt.Frame()
	if len(app.workspaces) != 1 || app.workspaces[0] != dir {
		t.Fatalf("手输的目录 = %v，想要 %q", app.workspaces, dir)
	}
}

// Hooks 的项目路径：选中后写进偏好，来源卡的路径行换成它。
func TestHooksProjectPathPicked(t *testing.T) {
	dir := pickedDir(t, "hooks-project")
	app, tt, changed, prefs := pickTester(t, "darwin",
		func(context.Context, string, ...string) (string, error) {
			return dir + "\n", nil
		})
	app.source.(*fakeSource).extensions.Hooks = ExtListSnapshot{Items: []ExtItemSnapshot{{
		ID: "h1", Source: "project", Event: "pre_save", Command: "true", Enabled: true,
	}}}
	tt = openSettingsSection(t, app, secHooks)

	if err := tt.Click(hkCopyChooseDir); err != nil {
		t.Fatalf("点选择项目目录：%v", err)
	}
	waitChanged(t, changed, "选择项目目录")
	tt.Frame()

	if app.hooksProjectPath != dir {
		t.Fatalf("项目目录 = %q，想要 %q", app.hooksProjectPath, dir)
	}
	if !tt.HasText(dir) {
		t.Fatalf("来源卡没有显示选中的目录：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	got := waitPrefs(t, prefs, "选择项目目录")
	if got.HooksProjectPath != dir {
		t.Fatalf("落盘的项目目录 = %q，想要 %q", got.HooksProjectPath, dir)
	}
}
