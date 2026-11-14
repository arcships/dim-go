package ui

import (
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// wiredSource 是一份"接上事实"的 Source：每个字段都与样例常量不同，用来证明
// -connect 下视图不再读样例。
func wiredSource() *fakeSource {
	return &fakeSource{
		conn:    Connection{Phase: "connected"},
		names:   []string{"s1"},
		session: SessionSnapshot{ID: "s1", Title: "真实会话"},
		hasSess: true,
		acct: AccountSnapshot{
			Initial: "A", Connected: true, Name: "Ada", Email: "ada@real.test", Plan: "Team",
			Status: "signed_in",
		},
		usage: UsageSnapshot{Available: true, Used: 7, Limit: 10, Unit: "GB", HasResets: true,
			ResetsAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		settings: SettingsSnapshot{
			Provider: "Acme", ProviderID: "acme", Providers: []ProviderOption{{ID: "acme", Name: "Acme"}},
			Model: "Model-X", ModelID: "model-x", Models: []ModelOption{{ID: "model-x", ProviderID: "acme", Name: "Model-X"}},
			Tier: "full-access", TierKnown: true,
			Storage:       StorageSnapshot{DatabaseBytes: 1536, WALBytes: 2048, ReclaimableBytes: 0},
			StorageKnown:  true,
			Runtimes:      []RuntimeSnapshot{{Name: "server-real", Version: "v9.0.0", Status: "ready"}},
			RuntimesKnown: true,
			Modality:      ModalitySnapshot{Supported: true, TaskSupported: true, TaskKnown: true},
		},
		extensions: ExtensionsSnapshot{
			Plugins: ExtListSnapshot{Items: []ExtItemSnapshot{{Name: "real-plugin", Version: "v2", Enabled: true}}},
			Skills:  ExtListSnapshot{Items: []ExtItemSnapshot{{Name: "real-skill", Enabled: true}}},
			MCP:     ExtListSnapshot{Items: []ExtItemSnapshot{{Name: "real-mcp", Enabled: true}}},
			Hooks:   ExtListSnapshot{Items: []ExtItemSnapshot{{Name: "real-hook", Enabled: false}}},
		},
		schedules: SchedulesSnapshot{Items: []ScheduleSnapshot{{
			ID: "job-1", Name: "real-job", Schedule: "0 2 * * *", Enabled: true,
			NextRunAt: time.Date(2026, 1, 3, 2, 0, 0, 0, time.UTC), LastStatus: "ok",
		}}},
		terminal: TerminalSnapshot{Active: true, Running: true},
	}
}

// samplePhrases 是各屏里出现过的样例常量；接上事实后任何一条都不该出现在视图里。
var samplePhrases = []string{
	"eric@example.com", "Pro · 每月", "34% 已用", "12 月 1 日",
	"Anthropic", "Opus 4.6", "$3 / 百万 token",
	"go test ./...", "rm -rf",
	"GitHub", "代码评审", "filesystem", "提交前检查",
	"每晚依赖检查", "每周汇总", "清理临时文件",
	"dim-app-server", "终端组件", "渲染组件", "v3.4.1",
	"1.2", "128", "84", "项目 A", "项目 B",
	"把日志也开上", "发送路径.md", "rm -rf ./tmp/*",
}

// TestPreferencesSeedView 本地偏好进入视图：主题、动效、时区与侧栏宽度。
func TestPreferencesSeedView(t *testing.T) {
	src := wiredSource()
	src.prefs = PreferencesSnapshot{Theme: "暗", MotionOff: true, TimeZone: "UTC", SidebarWidth: 300}
	app := NewApp(Options{State: StateDefault, Source: src})
	if app.themePref != "暗" || !app.motionOff || app.timeZone != "UTC" {
		t.Fatalf("偏好没有进入视图：theme=%q motion=%v tz=%q", app.themePref, app.motionOff, app.timeZone)
	}
	if app.sidebarWidthValue() != 300 {
		t.Fatalf("侧栏宽度 = %v，想要 300", app.sidebarWidthValue())
	}
}

// TestWiredScreensExcludeSamples 逐屏渲染 -connect 的视图，断言样例常量不再上屏。
func TestWiredScreensExcludeSamples(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})

	screens := []struct {
		name   string
		setup  func(*App)
		expect string // 必须出现的真实值（证明这一屏确实读了 Source）
	}{
		{"账号", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secAccount }, "ada@real.test"},
		{"供应商", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secProviders }, "Acme"},
		{"命令规则", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secRules }, "尚未读取"},
		{"扩展", func(a *App) { a.screen, a.settingsSection, a.extTab = ScreenSettings, secExtensions, 1 }, "real-plugin"},
		{"定时任务", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secCron }, "real-job"},
		{"运行时", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secRuntime }, "server-real"},
		{"存储", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secStorage }, "1.5 KB"},
		{"外观", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secAppearance }, "跟随系统"},
		{"终端", func(a *App) { a.screen, a.rightOpen, a.rightTab = ScreenMain, true, 4 }, "还没有输出"},
	}
	for _, screen := range screens {
		screen.setup(app)
		tester := gui.NewTester(app.View, WindowWidth, WindowHeight)
		text := strings.Join(tester.Texts(), "\n")
		if !strings.Contains(text, screen.expect) {
			t.Fatalf("%s：没有出现真实值 %q\n%s", screen.name, screen.expect, text)
		}
		for _, phrase := range samplePhrases {
			if strings.Contains(text, phrase) {
				t.Fatalf("%s：视图里出现了样例常量 %q\n%s", screen.name, phrase, text)
			}
		}
	}
}
