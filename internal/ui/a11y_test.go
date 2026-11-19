package ui

import (
	"strings"
	"testing"
)

// 本文件是「读屏标注」这一类的测试：可读名与补充说明由屏幕元素承担，不把图标
// 与状态词拆成独立读屏项。

// TestSessionRowDescription：会话行的说明带状态、置顶、选中与自然语言时间。
func TestSessionRowDescription(t *testing.T) {
	cases := []struct {
		marker, when string
		chosen, pin  bool
		want         []string
	}{
		{"", "now", false, false, []string{"刚刚"}},
		{"●", "1h", false, false, []string{"正在运行", "1 小时前"}},
		{"◷", "16m", true, false, []string{"等待你同意", "已选中", "16 分钟前"}},
		{"✕", "", false, true, []string{"失败", "已置顶"}},
	}
	for _, c := range cases {
		got := sessionRowDescription(c.marker, c.when, c.chosen, c.pin)
		for _, part := range c.want {
			if !strings.Contains(got, part) {
				t.Fatalf("说明 %q 缺 %q", got, part)
			}
		}
	}
	if got := sessionRowDescription("", "", false, false); got != "" {
		t.Fatalf("没有状态时说明应为空，得到 %q", got)
	}
}

// TestGroupRowDescription：工作区分组的说明写内置、展开态与目录；内置组不把
// 目录当自己的路径。
func TestGroupRowDescription(t *testing.T) {
	builtin := sidebarWorkspace{id: sidebarBuiltinID, name: sidebarBuiltinName, builtin: true}
	got := groupDescription(builtin, true)
	if !strings.Contains(got, "内置工作区") || !strings.Contains(got, "已展开") {
		t.Fatalf("内置组说明 = %q", got)
	}
	group := sidebarWorkspace{id: "local:/work/dim-agent", name: "dim-agent", path: "/work/dim-agent"}
	got = groupDescription(group, false)
	if !strings.Contains(got, "已收起") || !strings.Contains(got, "/work/dim-agent") {
		t.Fatalf("目录组说明 = %q", got)
	}
	if strings.Contains(got, "内置") {
		t.Fatalf("目录组被说成内置：%q", got)
	}
}

// TestAccountRowDescription：账户名缺失时给占位，读屏说明写登录状态。
func TestAccountRowDescription(t *testing.T) {
	src := &fakeSource{conn: Connection{Phase: "connected"}}
	app := NewApp(Options{State: StateDefault, Source: src})
	if got := app.accountNameValue(); got != "未命名账户" {
		t.Fatalf("空账户名 = %q，想要「未命名账户」", got)
	}
	if got := app.accountStatusPhrase(); !strings.Contains(got, "未登录") {
		t.Fatalf("未登录账户说明 = %q", got)
	}

	src.acct = AccountSnapshot{Connected: true, Name: "Ada"}
	if got := app.accountStatusPhrase(); !strings.Contains(got, "已登录") || !strings.Contains(got, "Ada") {
		t.Fatalf("已登录账户说明 = %q", got)
	}
}

// TestContextUsagePhrase：上限未知时不编百分比。
func TestContextUsagePhrase(t *testing.T) {
	if got := contextUsagePhrase(0, false); strings.Contains(got, "%") {
		t.Fatalf("上限未知时说了百分比：%q", got)
	}
	if got := contextUsagePhrase(6, true); !strings.Contains(got, "约 6%") {
		t.Fatalf("已知用量说明 = %q", got)
	}
}

// TestSidebarRailNamedForScreenReader：收起态的边条有可读名与快捷键，读屏按名字
// 找到它并展开。
func TestSidebarRailNamedForScreenReader(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.setSidebarCollapsed(true)
	tt.Frame()
	if _, ok := tt.Find("展开侧边栏 ⌘B"); !ok {
		t.Fatalf("边条没有可读名「展开侧边栏 ⌘B」：%v", tt.Texts())
	}
}
