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

// 本文件是 §8 导入与更新的回归：插件安装（本地 manifest 预览 / Git）、安装中与
// 失败原因、可更新清单与逐项/全部更新、移除确认、插件市场的配置门槛。

// pluginTestInstall 记录一次 plugin.install 的实参。
type pluginTestInstall struct {
	kind, ref, gitRef, id string
	enable                bool
}

// pluginTestIntents 接住插件管理的写入与市场动作。
type pluginTestIntents struct {
	Intents
	installed   chan pluginTestInstall
	updated     chan string
	removed     chan string
	feed        chan string
	refreshed   chan struct{}
	failInstall error
	failUpdate  error
	failRemove  error
}

func (p *pluginTestIntents) InstallPlugin(_ context.Context, kind, ref, gitRef, id string, enable bool) error {
	if p.failInstall != nil {
		return p.failInstall
	}
	if p.installed != nil {
		p.installed <- pluginTestInstall{kind: kind, ref: ref, gitRef: gitRef, id: id, enable: enable}
	}
	return nil
}

func (p *pluginTestIntents) UpdatePlugin(_ context.Context, id string) error {
	if p.failUpdate != nil {
		return p.failUpdate
	}
	if p.updated != nil {
		p.updated <- id
	}
	return nil
}

func (p *pluginTestIntents) UninstallExtension(_ context.Context, kind, id string) error {
	if p.failRemove != nil {
		return p.failRemove
	}
	if p.removed != nil && kind == "plugin" {
		p.removed <- id
	}
	return nil
}

func (p *pluginTestIntents) SetPluginFeed(_ context.Context, feed string) error {
	if p.feed != nil {
		p.feed <- feed
	}
	return nil
}

func (p *pluginTestIntents) RefreshPluginMarketplace(context.Context) error {
	if p.refreshed != nil {
		p.refreshed <- struct{}{}
	}
	return nil
}

// pluginSettingsTester 直接把界面停在设置 › 插件设置。
func pluginSettingsTester(src Source, intents Intents) (*App, *gui.Tester) {
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	app.screen = ScreenSettings
	app.settingsSection = secPluginSettings
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// waitPluginFrame 反复渲染，直到 cond 成立或超时。
func waitPluginFrame(t *testing.T, tt *gui.Tester, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("等待界面状态超时")
}

// writeTestPlugin 写一个最小插件目录：dim-plugin.json + skills/*.md。
func writeTestPlugin(t *testing.T, name, version string, skills ...string) string {
	t.Helper()
	dir := t.TempDir()
	manifest := `{"name":"` + name + `","version":"` + version + `","skills":"skills"}`
	if err := os.WriteFile(filepath.Join(dir, "dim-plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if len(skills) > 0 {
		if err := os.MkdirAll(filepath.Join(dir, "skills"), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, skill := range skills {
			if err := os.WriteFile(filepath.Join(dir, "skills", skill+".md"), []byte("# "+skill), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

// TestPluginManageShowsFactsNotSamples：已装清单来自事实，样例插件名不上屏；
// 未配置市场源时整块写「未配置插件源」，不画样例条目。
func TestPluginManageShowsFactsNotSamples(t *testing.T) {
	src := wiredSource()
	src.extensions.Plugins = ExtListSnapshot{Items: []ExtItemSnapshot{
		{ID: "real-plugin", Name: "real-plugin", Version: "2.0.0", Enabled: true},
	}}
	_, tt := pluginSettingsTester(src, &pluginTestIntents{})

	for _, want := range []string{"我的插件", "real-plugin", "已启用", "插件市场", "未配置插件源"} {
		if !tt.HasText(want) {
			t.Fatalf("插件管理块缺少 %q：%v", want, tt.Texts())
		}
	}
	for _, sample := range []string{"Model log", "Prompt suggestion", "Web"} {
		if tt.HasText(sample) {
			t.Fatalf("连接模式下仍在画样例插件 %q", sample)
		}
	}
	if info, ok := tt.Access("打开添加插件弹窗"); !ok || info.Disabled {
		t.Fatalf("有写通道时「添加插件」不可点：%+v", info)
	}
}

// TestPluginInstallLocalPreviewAndInstall：选目录 → 校验 manifest → 预览条目 →
// 安装 → 结果；安装中禁用，成功后弹窗关闭。
func TestPluginInstallLocalPreviewAndInstall(t *testing.T) {
	dir := writeTestPlugin(t, "demo-plugin", "1.2.0", "alpha", "beta")
	intents := &pluginTestIntents{installed: make(chan pluginTestInstall, 1)}
	app, tt := pluginSettingsTester(wiredSource(), intents)

	if err := tt.Click("打开添加插件弹窗"); err != nil {
		t.Fatalf("点不到添加插件：%v", err)
	}
	tt.Frame()
	if !app.pluginInstall.open {
		t.Fatal("安装弹窗没有打开")
	}
	app.pluginInstall.path = dir
	tt.Frame()
	if err := tt.Click("校验并预览"); err != nil {
		t.Fatalf("点不到校验并预览：%v", err)
	}
	tt.Frame()
	for _, want := range []string{"demo-plugin", "技能（2）", "alpha", "beta"} {
		if !tt.HasText(want) {
			t.Fatalf("预览缺少 %q：%v", want, tt.Texts())
		}
	}
	app.confirmPluginInstall()
	select {
	case got := <-intents.installed:
		if got.kind != "local" || got.ref != dir || got.id != "" || !got.enable {
			t.Fatalf("plugin.install 参数不对：%+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有发出 plugin.install")
	}
	waitPluginFrame(t, tt, func() bool { return !app.pluginInstall.open })
	if !tt.HasText("已安装 " + dir) {
		t.Fatalf("安装结果没有上屏：%v", tt.Texts())
	}
}

// TestPluginInstallRejectsBadManifest：目录里没有 manifest / manifest 无效时预览
// 报错，且「添加插件」禁用。
func TestPluginInstallRejectsBadManifest(t *testing.T) {
	app, tt := pluginSettingsTester(wiredSource(), &pluginTestIntents{})
	if err := tt.Click("打开添加插件弹窗"); err != nil {
		t.Fatalf("点不到添加插件：%v", err)
	}
	tt.Frame()
	app.pluginInstall.path = t.TempDir()
	tt.Frame()
	if err := tt.Click("校验并预览"); err != nil {
		t.Fatalf("点不到校验并预览：%v", err)
	}
	tt.Frame()
	if !tt.HasText(pluginNoManifest) {
		t.Fatalf("没有写 manifest 缺失：%v", tt.Texts())
	}
	if problem := app.pluginInstallProblem(); problem == "" {
		t.Fatal("manifest 无效时仍可添加")
	}
}

// TestPluginInstallGitUsesRefAndEnable：Git 模式的来源、ref 与「安装后启用」都
// 原样交给写入。
func TestPluginInstallGitUsesRefAndEnable(t *testing.T) {
	intents := &pluginTestIntents{installed: make(chan pluginTestInstall, 1)}
	app, tt := pluginSettingsTester(wiredSource(), intents)
	if err := tt.Click("打开添加插件弹窗"); err != nil {
		t.Fatalf("点不到添加插件：%v", err)
	}
	tt.Frame()
	app.pluginInstall.mode = pluginInstallModeGit
	app.pluginInstall.git = "owner/repo"
	app.pluginInstall.ref = "main"
	app.pluginInstall.id = "custom-id"
	app.pluginInstall.enable = false
	tt.Frame()
	app.confirmPluginInstall()
	select {
	case got := <-intents.installed:
		if got.kind != "git" || got.ref != "owner/repo" || got.gitRef != "main" ||
			got.id != "custom-id" || got.enable {
			t.Fatalf("Git 安装参数不对：%+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有发出 plugin.install")
	}
}

// TestPluginInstallFailureKeepsDialog：安装失败时弹窗保留并显示原因。
func TestPluginInstallFailureKeepsDialog(t *testing.T) {
	intents := &pluginTestIntents{failInstall: errors.New("manifest 不被服务端接受")}
	app, tt := pluginSettingsTester(wiredSource(), intents)
	if err := tt.Click("打开添加插件弹窗"); err != nil {
		t.Fatalf("点不到添加插件：%v", err)
	}
	tt.Frame()
	dir := writeTestPlugin(t, "demo-plugin", "1.0.0", "alpha")
	app.pluginInstall.path = dir
	tt.Frame()
	if err := tt.Click("校验并预览"); err != nil {
		t.Fatalf("点不到校验并预览：%v", err)
	}
	tt.Frame()
	app.confirmPluginInstall()
	waitPluginFrame(t, tt, func() bool { return strings.Contains(app.pluginInstall.err, "manifest 不被服务端接受") })
	if !app.pluginInstall.open {
		t.Fatal("失败后弹窗被关掉了")
	}
	if !tt.HasText("添加插件：manifest 不被服务端接受") {
		t.Fatalf("失败原因没有上屏：%v", tt.Texts())
	}
}

// TestPluginUpdatesRowAndAll：可更新行显示版本变化；逐项更新与全部更新都走
// plugin.update，失败原因留在列表里。
func TestPluginUpdatesRowAndAll(t *testing.T) {
	src := wiredSource()
	src.extensions.Plugins = ExtListSnapshot{Items: []ExtItemSnapshot{
		{ID: "old-a", Name: "old-a", Version: "1.0.0", Enabled: true, UpdateAvailable: true, LatestVersion: "1.2.0"},
		{ID: "old-b", Name: "old-b", Version: "2.0.0", Enabled: true, UpdateAvailable: true, LatestVersion: "2.1.0"},
	}}
	intents := &pluginTestIntents{updated: make(chan string, 4)}
	_, tt := pluginSettingsTester(src, intents)

	if !tt.HasText("可用更新") {
		t.Fatalf("没有可用更新区：%v", tt.Texts())
	}
	if !tt.HasText("v1.0.0 → v1.2.0") {
		t.Fatalf("版本行不对：%v", tt.Texts())
	}
	if !tt.HasText("全部更新（2）") {
		t.Fatalf("没有全部更新入口：%v", tt.Texts())
	}
	if err := tt.Click("更新"); err != nil {
		t.Fatalf("点不到更新：%v", err)
	}
	select {
	case id := <-intents.updated:
		if id != "old-a" && id != "old-b" {
			t.Fatalf("UpdatePlugin 收到 %q", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有发出 plugin.update")
	}
	waitPluginFrame(t, tt, func() bool { return len(intents.updated) == 0 })
	if err := tt.Click("全部更新（2）"); err != nil {
		t.Fatalf("点不到全部更新：%v", err)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-intents.updated:
		case <-time.After(2 * time.Second):
			t.Fatal("全部更新没有逐项发出 plugin.update")
		}
	}
}

// TestPluginUpdateFailureStays：单行更新失败时该行保留并给原因。
func TestPluginUpdateFailureStays(t *testing.T) {
	src := wiredSource()
	src.extensions.Plugins = ExtListSnapshot{Items: []ExtItemSnapshot{
		{ID: "old-a", Name: "old-a", Version: "1.0.0", Enabled: true, UpdateAvailable: true, LatestVersion: "1.2.0"},
	}}
	intents := &pluginTestIntents{failUpdate: errors.New("下载失败")}
	_, tt := pluginSettingsTester(src, intents)
	if err := tt.Click("更新"); err != nil {
		t.Fatalf("点不到更新：%v", err)
	}
	waitPluginFrame(t, tt, func() bool { return tt.HasText("更新插件：下载失败") })
}

// TestPluginRemoveConfirmsAndReports：移除先确认；确认后走 UninstallExtension，
// 成功写结果，失败留在确认框里。
func TestPluginRemoveConfirmsAndReports(t *testing.T) {
	src := wiredSource()
	src.extensions.Plugins = ExtListSnapshot{Items: []ExtItemSnapshot{
		{ID: "real-plugin", Name: "real-plugin", Version: "2.0.0", Enabled: true},
	}}
	intents := &pluginTestIntents{removed: make(chan string, 1)}
	app, tt := pluginSettingsTester(src, intents)

	if err := tt.Click("移除"); err != nil {
		t.Fatalf("点不到移除：%v", err)
	}
	tt.Frame()
	if !tt.HasText(pluginRemoveTitle) {
		t.Fatalf("没有移除确认框：%v", tt.Texts())
	}
	if !tt.HasText(`将移除「real-plugin」及其技能。此操作无法撤销。`) {
		t.Fatalf("确认框描述不对：%v", tt.Texts())
	}
	if err := tt.Click("确认移除插件"); err != nil {
		t.Fatalf("点不到确认移除：%v", err)
	}
	select {
	case id := <-intents.removed:
		if id != "real-plugin" {
			t.Fatalf("UninstallExtension 收到 %q", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有发出卸载")
	}
	waitPluginFrame(t, tt, func() bool { return app.pluginManage.removeID == "" })
	if !tt.HasText("已移除 real-plugin") {
		t.Fatalf("移除结果没有上屏：%v", tt.Texts())
	}
}

// TestPluginMarketplaceConfiguredList：配置了插件源时画上游条目；安装入口置灰
// （协议没有市场安装来源），并且不出现样例卡片。
func TestPluginMarketplaceConfiguredList(t *testing.T) {
	src := wiredSource()
	src.market = PluginMarketplaceSnapshot{
		Configured: true, FeedURL: "https://feed.test/plugins.json",
		Entries: []PluginMarketEntrySnapshot{
			{Name: "market-plugin", Version: "3.1.0", Description: "来自上游的说明", Source: "owner/repo"},
		},
	}
	_, tt := pluginSettingsTester(src, &pluginTestIntents{})
	for _, want := range []string{"插件市场", "market-plugin · v3.1.0", "来自上游的说明"} {
		if !tt.HasText(want) {
			t.Fatalf("市场块缺少 %q：%v", want, tt.Texts())
		}
	}
	if info, ok := tt.Access("安装"); !ok || !info.Disabled {
		t.Fatalf("市场安装入口没有置灰：%+v（找到 %v）", info, ok)
	}
	if tt.HasText(pluginMarketUnset) {
		t.Fatal("已配置源时仍写「未配置插件源」")
	}
	if tt.HasText("Model log") {
		t.Fatal("市场块画了样例卡片")
	}
}

// TestPluginMarketplaceSavesFeed：保存插件源走 SetPluginFeed；未配置时整块禁用。
func TestPluginMarketplaceSavesFeed(t *testing.T) {
	intents := &pluginTestIntents{feed: make(chan string, 1)}
	app, tt := pluginSettingsTester(wiredSource(), intents)
	if !tt.HasText(pluginMarketUnset) {
		t.Fatalf("未配置源时没有写原因：%v", tt.Texts())
	}
	app.pluginManage.feedDraft = "https://feed.test/plugins.json"
	tt.Frame()
	if err := tt.Click("保存插件源"); err != nil {
		t.Fatalf("点不到保存插件源：%v", err)
	}
	select {
	case got := <-intents.feed:
		if got != "https://feed.test/plugins.json" {
			t.Fatalf("SetPluginFeed 收到 %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有发出 SetPluginFeed")
	}
}

// TestPluginActionsDisabledWithoutWriteChannel：连着 Source 但没有写通道时，安装 /
// 更新 / 移除都置灰并写原因。
func TestPluginActionsDisabledWithoutWriteChannel(t *testing.T) {
	src := wiredSource()
	src.extensions.Plugins = ExtListSnapshot{Items: []ExtItemSnapshot{
		{ID: "old-a", Name: "old-a", Version: "1.0.0", Enabled: true, UpdateAvailable: true, LatestVersion: "1.2.0"},
	}}
	_, tt := pluginSettingsTester(src, nil)
	if info, ok := tt.Access("打开添加插件弹窗"); !ok || !info.Disabled {
		t.Fatalf("没有写通道时「添加插件」没有置灰：%+v", info)
	}
	if info, ok := tt.Access("更新"); !ok || !info.Disabled {
		t.Fatalf("没有写通道时「更新」没有置灰：%+v", info)
	}
	if info, ok := tt.Access("移除"); !ok || !info.Disabled {
		t.Fatalf("没有写通道时「移除」没有置灰：%+v", info)
	}
	if !tt.HasText(noteNoWriteChannel) {
		t.Fatalf("没有写原因：%v", tt.Texts())
	}
}
