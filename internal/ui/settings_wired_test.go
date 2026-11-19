package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 设置 › MCP 与 Hooks 两屏接的是扩展读数：这里用注入源证明清单、读态与编辑器
// 带入的配置都来自 Source，且 -demo 的样例不再上屏。
//
// 真实服务端的扩展读数在 internal/uiwire 与 internal/extensions 里验。

// resetSettingsScreenState 清掉三块设置屏的包级视图状态：它们跨用例共享，
// 不清会让上一个用例的选中项与开关泄漏到下一个。
func resetSettingsScreenState() {
	mcpScreen = mcpState{view: mcpViewList, enabled: map[string]*bool{}}
	hooksScreen = hooksState{}
	configScreen = configState{
		selected: configDimName,
		models:   append([]configModel(nil), configModelTable...),
	}
}

// openSettingsSection 打开设置屏的一个分节并渲染一帧。
func openSettingsSection(t *testing.T, app *App, section settingsSectionID) *gui.Tester {
	t.Helper()
	app.screen, app.settingsSection = ScreenSettings, section
	return gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// TestMCPSettingsReadsSource 断言 MCP 清单来自 Extensions().MCP：服务器名、工具数、
// 来源与状态都是读数，样例一条都不出现。
func TestMCPSettingsReadsSource(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.extensions.MCP = ExtListSnapshot{Items: []ExtItemSnapshot{{
		ID: "mcp-1", Name: "fs-real", Source: "user", Enabled: true,
		Status: "authenticated", Transport: "stdio", Tools: 7,
		Command: "npx -y real-server",
	}}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secMCP)

	if _, ok := tt.Find("fs-real"); !ok {
		t.Fatal("MCP 屏没有画出读数里的服务器")
	}
	if _, ok := tt.Find("dim-desktop-browser"); ok {
		t.Fatal("连着 Source 时仍画了样例服务器")
	}
	if _, ok := tt.Access("7 个工具"); !ok {
		t.Fatal("工具数没有来自读数（读数是 7）")
	}
	if _, ok := tt.Find("user"); !ok {
		t.Fatal("来源没有来自读数")
	}
	if _, ok := tt.Find("authenticated"); !ok {
		t.Fatal("状态没有来自读数")
	}
}

// TestMCPSettingsRowOpensRealConfig 断言点一行服务器进编辑器时带入这一条的
// 真实配置（名称、传输方式与启动命令），不是样例那一条。
func TestMCPSettingsRowOpensRealConfig(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.extensions.MCP = ExtListSnapshot{Items: []ExtItemSnapshot{{
		ID: "mcp-1", Name: "fs-real", Source: "user", Enabled: true,
		Status: "authenticated", Transport: "stdio", Tools: 7,
		Command: "npx -y real-server",
	}}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secMCP)

	if err := tt.Click("fs-real"); err != nil {
		t.Fatalf("点不到 MCP 行：%v", err)
	}
	tt.Frame()
	if mcpScreen.view != mcpViewEdit {
		t.Fatal("点 MCP 行没有进编辑器")
	}
	if _, ok := tt.Find(mcpCopyConnectTitle); !ok {
		t.Fatal("编辑器没有出现")
	}
	if mcpScreen.draft.name != "fs-real" {
		t.Fatalf("编辑器带入的名称 = %q，想要 fs-real", mcpScreen.draft.name)
	}
	if mcpScreen.draft.command != "npx -y real-server" {
		t.Fatalf("编辑器带入的启动命令 = %q", mcpScreen.draft.command)
	}
	// 表单控件只带读屏名，取值留在草稿上：这两条断言证明带进来的是读数。
	if _, ok := tt.Find(mcpCopyCommand); !ok {
		t.Fatal("编辑器没有画启动命令这一栏")
	}
	if mcpScreen.draft.transport != mcpTransportStdio {
		t.Fatalf("编辑器带入的传输方式 = %d，想要 stdio", mcpScreen.draft.transport)
	}
}

// TestMCPSettingsReadError 断言读数失败与能力缺失各有渲染分支，且不清空已有行。
func TestMCPSettingsReadError(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.extensions.MCP = ExtListSnapshot{
		ReadError:         "mcp.catalog 超时",
		CapabilityMissing: true,
		Items:             []ExtItemSnapshot{{ID: "mcp-1", Name: "fs-real"}},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secMCP)

	if _, ok := tt.Find("读取 MCP 服务器失败：mcp.catalog 超时"); !ok {
		t.Fatal("读数失败没有给出原因")
	}
	if _, ok := tt.Find("服务端没有声明 MCP 能力，清单可能不完整。"); !ok {
		t.Fatal("能力缺失没有给出说明")
	}
	if _, ok := tt.Find("fs-real"); !ok {
		t.Fatal("读数失败时把上一次确认的行清空了")
	}
}

// TestMCPSettingsToggleWrites 断言服务器开关走 SetExtensionEnabled("mcp")。
func TestMCPSettingsToggleWrites(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.extensions.MCP = ExtListSnapshot{Items: []ExtItemSnapshot{{
		ID: "mcp-1", Name: "fs-real", Enabled: true, Transport: "stdio",
	}}}
	rec := newRecordIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: rec})
	tt := openSettingsSection(t, app, secMCP)

	if err := tt.Click("启用服务器 fs-real"); err != nil {
		t.Fatalf("点不到服务器开关：%v", err)
	}
	tt.Frame()
	rec.expect(t, "SetExtensionEnabled mcp mcp-1 false")
}

// TestHooksSettingsReadsSource 断言 Hooks 屏按来源分组画读数里的条目：事件、
// 命令与启停状态都来自读数，样例的修订号不再上屏。
func TestHooksSettingsReadsSource(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.extensions.Hooks = ExtListSnapshot{Items: []ExtItemSnapshot{
		{ID: "h1", Name: "保存前检查", Source: "user", Event: "pre_save",
			Command: "gofmt -l .", Enabled: true, Matcher: "*.go"},
		{ID: "h2", Source: "project", Event: "post_tool", Command: "make lint"},
		{ID: "h3", Source: "plugin", Event: "pre_prompt", Command: "check"},
	}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secHooks)

	for _, want := range []string{"pre_save", "post_tool", "pre_prompt", "命令：gofmt -l .", "匹配器 *.go"} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("Hooks 屏缺少读数里的 %q", want)
		}
	}
	// 三个来源各一张卡（用户 / 项目 / 插件），插件来源带只读徽标。
	if got := countText(tt, "用户"); got == 0 {
		t.Fatal("没有用户来源卡")
	}
	if _, ok := tt.Find("插件"); !ok {
		t.Fatal("没有插件来源卡")
	}
	if _, ok := tt.Find("只读"); !ok {
		t.Fatal("插件来源没有只读徽标")
	}
	// 没有条目时不该被画成样例：样例的修订号与空态说明都不出现。
	for _, sample := range []string{"native:user/hooks:1", "用户修订号"} {
		if _, ok := tt.Find(sample); ok {
			t.Fatalf("连着 Source 时仍画了样例的修订号 %q", sample)
		}
	}
	if _, ok := tt.Find("没有条目"); ok {
		t.Fatal("有读数条目时仍画了空态")
	}
}

// TestHooksSettingsReadError 断言读数失败时给原因，条目照画。
func TestHooksSettingsReadError(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.extensions.Hooks = ExtListSnapshot{
		ReadError: "hooks.catalog 不可用",
		Items:     []ExtItemSnapshot{{ID: "h1", Source: "user", Event: "pre_save", Command: "true"}},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secHooks)

	if _, ok := tt.Find("加载设置失败：hooks.catalog 不可用"); !ok {
		t.Fatal("读数失败没有给出原因")
	}
	if _, ok := tt.Find("pre_save"); !ok {
		t.Fatal("读数失败时把条目清空了")
	}
}

// recordIntents 记录视图发出的写入：只实现被断言的方法，其余留在嵌入的
// Intents 上（没被调到的方法不会生效，调到会 panic，测试里按需要补）。
type recordIntents struct {
	Intents
	calls chan string
}

// newRecordIntents 建一个带缓冲的记录器。
func newRecordIntents() *recordIntents {
	return &recordIntents{calls: make(chan string, 8)}
}

// expect 等一条写入记录：视图的写入在后台协程执行，这里给它两秒。
func (r *recordIntents) expect(t *testing.T, want string) {
	t.Helper()
	select {
	case got := <-r.calls:
		if got != want {
			t.Fatalf("写入了 %q，想要 %q", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("没有等到写入 %q", want)
	}
}

// SetExtensionEnabled 记录扩展启停。
func (r *recordIntents) SetExtensionEnabled(_ context.Context, kind, id string, enabled bool) error {
	r.calls <- fmt.Sprintf("SetExtensionEnabled %s %s %v", kind, id, enabled)
	return nil
}

// SetProviderModelEnabled 记录模型启停。
func (r *recordIntents) SetProviderModelEnabled(_ context.Context, providerID, modelID string, enabled bool) error {
	r.calls <- fmt.Sprintf("SetProviderModelEnabled %s %s %v", providerID, modelID, enabled)
	return nil
}

// ProbeProvider 记录探测（只读，返回模型数）。
func (r *recordIntents) ProbeProvider(_ context.Context, id string) (int, error) {
	r.calls <- "ProbeProvider " + id
	return 2, nil
}

// CreateProvider 记录新增（含表单里的模型标识）。
func (r *recordIntents) CreateProvider(
	_ context.Context,
	id, name, baseURL, apiKey string,
	models []ProviderModelDraft,
) error {
	call := "CreateProvider " + name
	if len(models) > 0 {
		ids := make([]string, 0, len(models))
		for _, model := range models {
			ids = append(ids, model.ID)
		}
		call += " models=" + strings.Join(ids, ",")
	}
	r.calls <- call
	return nil
}

// UpdateProvider 记录编辑。
func (r *recordIntents) UpdateProvider(_ context.Context, id, name, baseURL, apiKey string) error {
	r.calls <- "UpdateProvider " + id
	return nil
}

// joinedText 把一帧的文字连起来，便于断言「某段文案不在屏上」。
func joinedText(tt *gui.Tester) string {
	return strings.Join(tt.Texts(), "\n")
}

// TestSettingsScreensDemoKeepSamples 断言没有 Source（-demo）时三屏仍是样例：
// 截图与走查要能出图，样例不是事实。
func TestSettingsScreensDemoKeepSamples(t *testing.T) {
	cases := []struct {
		section settingsSectionID
		want    string
	}{
		{secMCP, "dim-desktop-browser"},
		{secHooks, "native:user/hooks:1"},
		{secProviders, configDimName},
	}
	for _, tc := range cases {
		resetSettingsScreenState()
		app := NewApp(Options{State: StateDefault})
		tt := openSettingsSection(t, app, tc.section)
		if !strings.Contains(joinedText(tt), tc.want) {
			t.Fatalf("分节 %d 在 -demo 下没有样例 %q", tc.section, tc.want)
		}
	}
}

// MCP 的写入口没有通道（DEC-04）：查看连接、断开连接与保存都置灰，并带原因。
func TestMCPSettingsWriteEntriesDisabled(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.extensions.MCP = ExtListSnapshot{Items: []ExtItemSnapshot{{
		ID: "mcp-1", Name: "fs-real", Source: "user", Enabled: true,
		Status: "authenticated", Transport: "stdio", Tools: 7,
		Command: "npx -y real-server", Connected: true,
	}}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secMCP)

	for _, label := range []string{mcpCopyInspect, mcpCopyDisconnect} {
		info, ok := tt.Access(label)
		if !ok || !info.Disabled {
			t.Fatalf("%q 没有置灰：%+v（找到 %v）", label, info, ok)
		}
	}
	if err := tt.Click("fs-real"); err != nil {
		t.Fatalf("点不到 MCP 行：%v", err)
	}
	tt.Frame()
	if !tt.HasText(noteMCPWrite) {
		t.Fatalf("编辑器没有写原因：%v", tt.Texts())
	}
	if info, ok := tt.Access(mcpCopySave); !ok || !info.Disabled {
		t.Fatalf("保存没有置灰：%+v（找到 %v）", info, ok)
	}
}

// 系统目录选择器没有宿主能力（DEC-10）：Worktree 路径改为手动输入，选择目录
// 按钮置灰并写原因。
func TestGitWorktreeManualPath(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault})
	tt := openSettingsSection(t, app, secGit)

	if !tt.HasText("手动输入路径（系统目录选择器未接入）") {
		t.Fatalf("没有写手动输入路径：%v", tt.Texts())
	}
	info, ok := tt.Access(gitCopyChooseDirA11y)
	if !ok || !info.Disabled {
		t.Fatalf("选择目录按钮没有置灰：%+v（找到 %v）", info, ok)
	}
	if info.Description != noteDirPicker {
		t.Fatalf("选择目录的 tooltip = %q，想要 %q", info.Description, noteDirPicker)
	}
}
