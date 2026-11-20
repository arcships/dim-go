package ui

import (
	"context"
	"fmt"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › MCP」编辑器与列表新增块的回归：配置问题横幅、已存值隐藏提示、
// 移动作用域入口、表单校验、测试结果与删除确认。需求 ID 写在各个用例注释里。

// mcpIntents 记录 MCP 的保存 / 删除 / 测试调用；返回可脚本化。
type mcpIntents struct {
	Intents
	calls    chan string
	testOK   bool
	testWhy  string
	testErr  error
	saveErr  error
	removeAs error
}

func newMCPIntents() *mcpIntents { return &mcpIntents{calls: make(chan string, 8), testOK: true} }

// openMCPTall 用一个更高的视口打开 MCP 屏：编辑器整块可见，点击不需要先滚动。
func openMCPTall(t *testing.T, app *App) *gui.Tester {
	t.Helper()
	app.screen, app.settingsSection = ScreenSettings, secMCP
	return gui.NewTester(app.View, WindowWidth, 1500)
}

func (m *mcpIntents) expect(t *testing.T, want string) {
	t.Helper()
	select {
	case got := <-m.calls:
		if got != want {
			t.Fatalf("调用了 %q，想要 %q", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("没有等到调用 %q", want)
	}
}

func (m *mcpIntents) SaveMCPServer(_ context.Context, draft MCPServerDraft) error {
	m.calls <- fmt.Sprintf("Save %s %s %s %s %s %v", draft.ID, draft.Name, draft.Scope,
		draft.Transport, draft.Command, draft.Args)
	return m.saveErr
}

func (m *mcpIntents) RemoveMCPServer(_ context.Context, id string) error {
	m.calls <- "Remove " + id
	return m.removeAs
}

func (m *mcpIntents) TestMCP(_ context.Context, id string) (bool, string, error) {
	m.calls <- "Test " + id
	return m.testOK, m.testWhy, m.testErr
}

func (m *mcpIntents) SetExtensionEnabled(_ context.Context, kind, id string, enabled bool) error {
	m.calls <- "SetExtensionEnabled " + kind + " " + id
	return nil
}

// mcpEditorSource 造一个带服务器的 MCP 读数与能力声明。
func mcpEditorSource(canSave, canTest, canRemove bool, scope string) *fakeSource {
	src := wiredSource()
	src.extensions.MCP = ExtListSnapshot{Items: []ExtItemSnapshot{{
		ID: "srv-1", Name: "srv-1", Source: scope, Transport: "stdio",
		Command: "npx", Args: []string{"-y", "server"}, Enabled: true, Tools: 2,
	}}}
	src.extensions.MCPCanSave = canSave
	src.extensions.MCPCanTest = canTest
	src.extensions.MCPCanRemove = canRemove
	return src
}

// TestMCPProblemBanner 断言配置问题横幅逐条列出作用域、标识与原因（S-mcp-10）。
func TestMCPProblemBanner(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.extensions.MCPProblems = []MCPProblemSnapshot{
		{Scope: "user", ServerID: "broken-1", Message: "缺少 command"},
		{Scope: "project", ServerID: "broken-2", Message: "地址不是 http"},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openMCPTall(t, app)

	for _, want := range []string{
		"2 个 MCP 配置问题需要注意",
		"user · broken-1 · 缺少 command",
		"project · broken-2 · 地址不是 http",
	} {
		if !tt.HasText(want) {
			t.Fatalf("配置问题横幅缺少 %q：%v", want, tt.Texts())
		}
	}
}

// TestMCPEditorValidates 断言保存前先本地校验：名称与启动命令缺一不可；补齐后
// 按编辑器字段提交一次 mcp.save（S-mcp-48）。
func TestMCPEditorValidates(t *testing.T) {
	resetSettingsScreenState()
	intents := newMCPIntents()
	app := NewApp(Options{State: StateDefault,
		Source: mcpEditorSource(true, true, true, "user"), Intents: intents})
	tt := openMCPTall(t, app)

	if err := tt.Click(mcpCopyAddServer); err != nil {
		t.Fatalf("点不到添加服务器：%v", err)
	}
	tt.Frame()
	if err := tt.Click(mcpCopySave); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if !tt.HasText("请填写服务器名称") {
		t.Fatalf("空名称没有校验：%v", tt.Texts())
	}
	mcpScreen.draft.name = "本地工具"
	tt.Frame()
	if err := tt.Click(mcpCopySave); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if !tt.HasText("请填写启动命令") {
		t.Fatalf("空命令没有校验：%v", tt.Texts())
	}
	mcpScreen.draft.command = "npx"
	mcpScreen.draft.args = []string{"-y", "server"}
	tt.Frame()
	if err := tt.Click(mcpCopySave); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	intents.expect(t, "Save mcp-server 本地工具 user stdio npx [-y server]")
	tt.Frame()
	if mcpScreen.view != mcpViewList {
		t.Fatal("保存成功后没有回到列表")
	}
}

// TestMCPEditorHTTPValidation 断言 http 传输要求合法地址（S-mcp-48）。
func TestMCPEditorHTTPValidation(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault,
		Source: mcpEditorSource(true, true, true, "user"), Intents: newMCPIntents()})
	tt := openMCPTall(t, app)

	if err := tt.Click(mcpCopyAddServer); err != nil {
		t.Fatalf("点不到添加服务器：%v", err)
	}
	tt.Frame()
	mcpScreen.draft.name = "远端工具"
	mcpScreen.draft.transport = mcpTransportHTTP
	mcpScreen.draft.url = "ftp://example.test"
	tt.Frame()
	if err := tt.Click(mcpCopySave); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if !tt.HasText("服务器地址需以 http:// 或 https:// 开头") {
		t.Fatalf("非法地址没有校验：%v", tt.Texts())
	}
}

// TestMCPEditWithheldAndMove 断言编辑已有条目时写清「已存储的值已隐藏」、
// 保存按缺口置灰并写原因，移动入口打开确认框（S-mcp-33/34）。
func TestMCPEditWithheldAndMove(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault,
		Source: mcpEditorSource(true, true, true, "user"), Intents: newMCPIntents()})
	tt := openMCPTall(t, app)

	if err := tt.Click(mcpCopyConfigure); err != nil {
		t.Fatalf("点不到配置服务器：%v", err)
	}
	tt.Frame()
	for _, want := range []string{mcpCopyWithheld, mcpCopyMoveHint, noteMCPEditSave,
		mcpCopyMoveToProject} {
		if !tt.HasText(want) {
			t.Fatalf("编辑态缺少 %q：%v", want, tt.Texts())
		}
	}
	if err := tt.Click(mcpCopyMoveToProject); err != nil {
		t.Fatalf("点不到移动：%v", err)
	}
	tt.Frame()
	if !tt.HasText(mcpCopyMoveConfirmTitle) || !tt.HasText(noteMCPMove) {
		t.Fatalf("移动确认框缺少标题或原因：%v", tt.Texts())
	}
	if err := tt.Click("取消移动"); err != nil {
		t.Fatalf("点不到取消：%v", err)
	}
	tt.Frame()
	if mcpScreen.moveOpen {
		t.Fatal("取消后移动确认框还开着")
	}
}

// TestMCPTestResult 断言「测试连接」走 mcp.test，结果写在编辑器里（S-mcp-49）。
func TestMCPTestResult(t *testing.T) {
	resetSettingsScreenState()
	intents := newMCPIntents()
	app := NewApp(Options{State: StateDefault,
		Source: mcpEditorSource(true, true, true, "user"), Intents: intents})
	tt := openMCPTall(t, app)

	if err := tt.Click(mcpCopyConfigure); err != nil {
		t.Fatalf("点不到配置服务器：%v", err)
	}
	tt.Frame()
	if err := tt.Click(mcpCopyTest); err != nil {
		t.Fatalf("点不到测试连接：%v", err)
	}
	intents.expect(t, "Test srv-1")
	tt.Frame()
	tt.Frame()
	if !tt.HasText(mcpCopyTestOK) {
		t.Fatalf("测试成功没有结果：%v", tt.Texts())
	}
}

// TestMCPTestFailure 断言测试失败的原因就地显示（S-mcp-49）。
func TestMCPTestFailure(t *testing.T) {
	resetSettingsScreenState()
	intents := newMCPIntents()
	intents.testOK = false
	intents.testWhy = "连接超时"
	app := NewApp(Options{State: StateDefault,
		Source: mcpEditorSource(true, true, true, "user"), Intents: intents})
	tt := openMCPTall(t, app)

	if err := tt.Click(mcpCopyConfigure); err != nil {
		t.Fatalf("点不到配置服务器：%v", err)
	}
	tt.Frame()
	if err := tt.Click(mcpCopyTest); err != nil {
		t.Fatalf("点不到测试连接：%v", err)
	}
	intents.expect(t, "Test srv-1")
	tt.Frame()
	tt.Frame()
	if !tt.HasText("连接失败：连接超时") {
		t.Fatalf("测试失败没有写原因：%v", tt.Texts())
	}
}

// TestMCPDeleteConfirm 断言删除确认框与 mcp.remove（S-mcp-51）。
func TestMCPDeleteConfirm(t *testing.T) {
	resetSettingsScreenState()
	intents := newMCPIntents()
	app := NewApp(Options{State: StateDefault,
		Source: mcpEditorSource(true, true, true, "user"), Intents: intents})
	tt := openMCPTall(t, app)

	if err := tt.Click("删除服务器"); err != nil {
		t.Fatalf("点不到删除：%v", err)
	}
	tt.Frame()
	for _, want := range []string{mcpCopyDeleteTitle, mcpCopyDeleteBody, mcpCopyDelete} {
		if !tt.HasText(want) {
			t.Fatalf("删除确认框缺少 %q：%v", want, tt.Texts())
		}
	}
	if err := tt.Click(mcpCopyDelete); err != nil {
		t.Fatalf("点不到确认删除：%v", err)
	}
	intents.expect(t, "Remove srv-1")
}

// TestMCPPluginServerReadOnly 断言插件来源的服务器不能删除，入口写原因且不弹框
// （S-mcp-51 的只读支）。
func TestMCPPluginServerReadOnly(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault,
		Source: mcpEditorSource(true, true, true, "plugin"), Intents: newMCPIntents()})
	tt := openMCPTall(t, app)

	if err := tt.Click("删除服务器"); err != nil {
		t.Fatalf("点不到删除入口：%v", err)
	}
	tt.Frame()
	if mcpScreen.deleteFor != "" {
		t.Fatalf("插件来源打开了删除框：%q", mcpScreen.deleteFor)
	}
	if mcpScreen.view != mcpViewList {
		t.Fatal("插件来源的删除入口不该进编辑器")
	}
}

// TestMCPCapabilityReasons 断言服务端没声明方法时，保存入口写原因、测试与删除
// 入口禁用（S-mcp-48/49/51 的缺口支）。
func TestMCPCapabilityReasons(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault,
		Source: mcpEditorSource(false, false, false, "user"), Intents: newMCPIntents()})
	tt := openMCPTall(t, app)

	if err := tt.Click(mcpCopyAddServer); err != nil {
		t.Fatalf("点不到添加服务器：%v", err)
	}
	tt.Frame()
	if !tt.HasText(noteMCPNoSave) {
		t.Fatalf("不可保存时没有写原因：%v", tt.Texts())
	}
	if err := tt.Click(mcpCopySave); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if mcpScreen.formErr != "" || mcpScreen.view != mcpViewEdit {
		t.Fatalf("没有保存通道时保存发了校验或请求：%+v", mcpScreen)
	}
	mcpScreen.view = mcpViewList
	tt.Frame()
	if err := tt.Click("删除服务器"); err != nil {
		t.Fatalf("点不到删除入口：%v", err)
	}
	tt.Frame()
	if mcpScreen.deleteFor != "" {
		t.Fatalf("没有删除通道时打开了删除框：%q", mcpScreen.deleteFor)
	}
}

// TestMCPServerID 断言名称归一化成条目标识：小写、非字母数字折成连字符。
func TestMCPServerID(t *testing.T) {
	cases := map[string]string{
		"my-server": "my-server",
		"My Server": "my-server",
		"  本地 工具  ": "mcp-server",
		"files__v2": "files-v2",
		"--edges--": "edges",
	}
	for name, want := range cases {
		if got := mcpServerID(name); got != want {
			t.Fatalf("mcpServerID(%q) = %q，想要 %q", name, got, want)
		}
	}
}
