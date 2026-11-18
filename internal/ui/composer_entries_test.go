package ui

import (
	"errors"
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 空态的两条首次运行入口：点了跳设置对应分节，不新建对话框、不发请求。
func TestEmptyStateFirstRunGuide(t *testing.T) {
	for _, tc := range []struct {
		text  string
		label string
		want  settingsSectionID
	}{
		{"登录账号", "登录账号：打开设置里的账户分节", secAccount},
		{"添加供应商", "添加供应商：打开设置里的模型供应分节", secProviders},
	} {
		app, tt := newTester(t, StateEmpty)
		if _, ok := tt.Find(tc.label); !ok {
			t.Fatalf("空态缺少入口 %q", tc.label)
		}
		if err := tt.Click(tc.label); err != nil {
			t.Fatalf("点 %q：%v", tc.text, err)
		}
		if app.screen != ScreenSettings || app.settingsSection != tc.want {
			t.Fatalf("点「%s」后落在 %v/%v，想要设置屏的 %v",
				tc.text, app.screen, app.settingsSection, tc.want)
		}
	}
}

// 断线卡：两个入口都在；「查看日志」用系统默认程序打开入口交来的路径。
func TestDisconnectedCardShowsLogEntry(t *testing.T) {
	var opened []string
	restore := swapLogOpenRunner(func(name string, args ...string) error {
		opened = append(opened, name+" "+strings.Join(args, " "))
		return nil
	})
	defer restore()

	app, tt := disconnectedTester(t)
	if _, ok := tt.Find("重新连接"); !ok {
		t.Fatal("断线卡缺少「重新连接」")
	}
	if _, ok := tt.Find("查看日志：用系统默认程序打开客户端日志"); !ok {
		t.Fatal("断线卡缺少「查看日志」")
	}
	if err := tt.Click("查看日志：用系统默认程序打开客户端日志"); err != nil {
		t.Fatalf("点「查看日志」：%v", err)
	}
	if len(opened) != 1 || !strings.HasSuffix(opened[0], " "+sampleLogPath) {
		t.Fatalf("打开命令 = %v，想要打开 %s", opened, sampleLogPath)
	}
	if app.logNote != "" {
		t.Fatalf("打开成功后不该留提示，得到 %q", app.logNote)
	}
	if tt.HasText(sampleLogPath) {
		t.Fatal("打开成功后不该把路径画在断线卡里")
	}
}

// 打不开时把路径留在原位供复制，不静默失败。
func TestDisconnectedCardLogFailureShowsPath(t *testing.T) {
	restore := swapLogOpenRunner(func(string, ...string) error {
		return errors.New("无法启动 open")
	})
	defer restore()

	app, tt := disconnectedTester(t)
	if err := tt.Click("查看日志：用系统默认程序打开客户端日志"); err != nil {
		t.Fatalf("点「查看日志」：%v", err)
	}
	tt.Frame()
	if app.logNote != sampleLogPath {
		t.Fatalf("失败后留的路径 = %q，想要 %q", app.logNote, sampleLogPath)
	}
	if _, ok := tt.Find("复制日志路径"); !ok {
		t.Fatal("失败后缺少「复制日志路径」入口")
	}
}

// 入口没有记下日志路径时说明这一点，不去猜数据目录。
func TestDisconnectedLogWithoutPath(t *testing.T) {
	app := NewApp(Options{Source: &fakeSource{conn: disconnectedConn()}})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("查看日志：用系统默认程序打开客户端日志"); err != nil {
		t.Fatalf("点「查看日志」：%v", err)
	}
	if !strings.Contains(app.logNote, "没有记录日志路径") {
		t.Fatalf("没有路径时的提示 = %q", app.logNote)
	}
}

// 平台打开命令：三个平台与不支持的系统。
func TestOpenLogCommand(t *testing.T) {
	if name, args, err := openLogCommand("darwin", "/x/dimagent.log"); err != nil ||
		name != "open" || len(args) != 1 || args[0] != "/x/dimagent.log" {
		t.Fatalf("darwin: %q %v %v", name, args, err)
	}
	if name, _, err := openLogCommand("linux", "/x/dimagent.log"); err != nil || name != "xdg-open" {
		t.Fatalf("linux: %q %v", name, err)
	}
	if name, args, err := openLogCommand("windows", `C:\x\dimagent.log`); err != nil ||
		name != "cmd" || len(args) != 4 || args[2] != "" {
		t.Fatalf("windows: %q %v %v", name, args, err)
	}
	if _, _, err := openLogCommand("plan9", "/x"); err == nil {
		t.Fatal("不支持的平台应给原因")
	}
}

// sampleLogPath 是测试里的客户端日志路径（不进文档）。
const sampleLogPath = "/tmp/dimagent-test/logs/dimagent.log"

// disconnectedConn 是一个断线连接：原因写清，卡片原位显示。
func disconnectedConn() Connection {
	return Connection{Phase: "disconnected", Reason: "app server connection closed: EOF"}
}

// disconnectedTester 渲染一张带日志路径的断线卡。
func disconnectedTester(t *testing.T) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{Source: &fakeSource{conn: disconnectedConn()}, LogPath: sampleLogPath})
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// swapLogOpenRunner 换掉外部打开执行器，返回恢复函数。
func swapLogOpenRunner(run func(name string, args ...string) error) func() {
	prev := logOpenRunner
	logOpenRunner = run
	return func() { logOpenRunner = prev }
}
