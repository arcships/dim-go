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
			Model: "Model-X", ModelID: "model-x", ModelsKnown: true,
			Models: []ModelOption{{ID: "model-x", ProviderID: "acme", Name: "Model-X"}},
			Tier:   "full-access", TierKnown: true,
			Storage:       StorageSnapshot{DatabaseBytes: 1536, WALBytes: 2048, ReclaimableBytes: 0},
			StorageKnown:  true,
			Runtimes:      []RuntimeSnapshot{{Name: "server-real", Version: "v9.0.0", Status: "ready"}},
			RuntimesKnown: true,
			Modality: ModalitySnapshot{
				Known: true, Supported: true, TaskSupported: true, TaskKnown: true,
			},
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
	"修发送路径", "参考", "旧实现对照",
	"把日志也开上", "发送路径.md", "rm -rf ./tmp/*",
	// 审核报告 docs/audit-fakes.md §1 的 P0 假事实：审批卡的 cwd 与空命令回落、
	// 模型槽的样例品牌、设置 › Git 的远程、设置 › 宠物的 Pet、设置 › 插件设置的
	// 插件名。它们此前都不在这份清单里，所以缺一条事实时屏上出现样例也没人拦。
	"~/dim-agent", "ls -la", "seed-2.1-pro", "dim-fox", "Model log", "origin",
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

// TestWiredNoSessionHidesSamples 连上 Source 但没有会话时，标题与目标行不落样例。
func TestWiredNoSessionHidesSamples(t *testing.T) {
	src := wiredSource()
	src.names = nil
	src.hasSess = false
	src.session = SessionSnapshot{}
	app := NewApp(Options{State: StateDefault, Source: src})
	tester := gui.NewTester(app.View, WindowWidth, WindowHeight)
	text := strings.Join(tester.Texts(), "\n")
	if !strings.Contains(text, "DimAgent") {
		t.Fatalf("无会话时标题不是应用名：\n%s", text)
	}
	for _, phrase := range []string{"修发送路径", "设定目标", "项目 A", "项目 B", "参考", "旧实现对照"} {
		if strings.Contains(text, phrase) {
			t.Fatalf("无会话时出现样例串 %q：\n%s", phrase, text)
		}
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
		{"主界面", func(a *App) { a.screen = ScreenMain }, "真实会话"},
		{"账号", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secAccount }, "ada@real.test"},
		{"终端", func(a *App) { a.screen, a.rightOpen, a.rightTab = ScreenMain, true, 4 }, termNoCwdText},
		// 设置区：这些屏曾经全靠样例（docs/audit-fakes.md 的 P0-3…P0-8 与 T-3）。
		// 现在它们要么读事实（运行环境），要么只写这一屏读不到的原因。
		{"设置·运行环境", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secRuntime }, "server-real"},
		{"设置·多模态", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secModality }, noteModalityGroups},
		{"设置·Git", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secGit }, noteGitSettings},
		{"设置·宠物", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secPets }, notePetSettings},
		{"设置·插件设置", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secPluginSettings }, notePluginSettings},
		{"设置·通用", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secGeneral }, noteDesktopSettings},
		{"设置·个性化", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secPersonalization }, notePersonalizationWrite},
		{"设置·权限", func(a *App) { a.screen, a.settingsSection = ScreenSettings, secPermissions }, noteIsolationUnread},
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
