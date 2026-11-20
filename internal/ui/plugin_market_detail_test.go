package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是 G-H2（市场安装详情，G-H2-01…G-H2-09）的回归：条目字段、加载与失败
// 块、技能勾选行、安装 / 取消按钮、安装中的状态，以及成功与失败回执。
//
// 事实来自插件源条目（ui.PluginMarketEntrySnapshot）：作者、安装位置、Git 地址
// 与技能清单都是条目声明的，不是本端猜的。没有可安装地址、没有写通道、同名插件
// 已装时按钮置灰并写原因。

// marketSource 是一份配置了源、带指定条目的市场读数。
func marketSource(entries ...PluginMarketEntrySnapshot) *fakeSource {
	src := wiredSource()
	src.market = PluginMarketplaceSnapshot{
		Configured: true, FeedURL: "https://feed.test/plugins.json", Entries: entries,
	}
	return src
}

// marketEntry 是一个字段齐全的市场条目（含两个技能与 Git 地址）。
func marketEntry() PluginMarketEntrySnapshot {
	return PluginMarketEntrySnapshot{
		Name:        "market-plugin",
		Version:     "3.1.0",
		Author:      "Ada",
		Description: "来自上游的说明",
		Source:      "owner/repo",
		Location:    "~/.dim/plugins/market-plugin",
		Git:         "owner/repo",
		GitRef:      "main",
		Skills: []PluginMarketSkillSnapshot{
			{Name: "alpha", Description: "第一条技能"},
			{Name: "beta"},
		},
	}
}

// openMarketDetailFor 打开某条目的安装详情。
func openMarketDetailFor(t *testing.T, src Source, intents Intents) (*App, *gui.Tester) {
	t.Helper()
	app, tt := pluginSettingsTester(src, intents)
	if err := tt.Click(pluginMarketInstall); err != nil {
		t.Fatalf("点不到市场条目的「安装」：%v", err)
	}
	tt.Frame()
	if !app.pluginManage.marketOpen {
		t.Fatalf("详情没有打开：%v", tt.Texts())
	}
	return app, tt
}

// clickAccessRole 按角色 + 读屏名找到元素，再点它自己的包围盒中心（同一个可见
// 文字在一屏上出现多次时，只能靠可区分的读屏名定位）。
func clickAccessRole(t *testing.T, tt *gui.Tester, role gui.Role, name string) {
	t.Helper()
	info, ok := tt.AccessByRole(role, name)
	if !ok {
		t.Fatalf("找不到 %v「%s」", role, name)
	}
	if info.Disabled {
		t.Fatalf("%v「%s」是禁用的，点不响", role, name)
	}
	r := info.Bounds
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
}

// TestPluginMarketDetailShowsEntryFields：详情框把条目的名称、版本、作者、来源、
// 安装位置与说明逐项画出来，标题是「安装 {name}」（G-H2-01 / G-H2-02）。
func TestPluginMarketDetailShowsEntryFields(t *testing.T) {
	_, tt := openMarketDetailFor(t, marketSource(marketEntry()), &pluginTestIntents{})

	if _, ok := tt.AccessByRole(gui.RoleDialog, "安装 market-plugin"); !ok {
		t.Fatalf("对话框名不是「安装 market-plugin」：%v", tt.Texts())
	}
	for _, want := range []string{
		"安装 market-plugin",
		pluginMarketInstallDesc,
		"名称：market-plugin",
		"版本：v3.1.0",
		"作者：Ada",
		"来源：owner/repo",
		"安装位置：~/.dim/plugins/market-plugin",
		"来自上游的说明",
	} {
		if !tt.HasText(want) {
			t.Fatalf("详情缺少 %q：%v", want, tt.Texts())
		}
	}
	if info, ok := tt.AccessByRole(gui.RoleHeading, "安装 market-plugin"); !ok || info.Level != 2 {
		t.Fatalf("标题不是二级标题：%+v（找到 %v）", info, ok)
	}
}

// TestPluginMarketDetailWritesFieldReasons：条目缺哪一项就写哪一项的原因，不留
// 空行、不猜；技能清单没给时也不断言「不包含技能」。
func TestPluginMarketDetailWritesFieldReasons(t *testing.T) {
	_, tt := openMarketDetailFor(t, marketSource(PluginMarketEntrySnapshot{Name: "bare-plugin"}),
		&pluginTestIntents{})

	for _, want := range []string{
		"名称：bare-plugin",
		pluginMarketNoVersion,
		pluginMarketNoAuthor,
		pluginMarketNoSource,
		pluginMarketNoLocation,
		pluginMarketNoDescription,
		pluginMarketNoSkills,
	} {
		if !tt.HasText(want) {
			t.Fatalf("缺项没有写原因 %q：%v", want, tt.Texts())
		}
	}
	if tt.HasText(pluginEmptySkills) {
		t.Fatal("源没给技能清单时却断言「此插件不包含技能」")
	}
}

// TestPluginMarketDetailSkillsDefaultAllSelected：技能勾选行默认全选，点一行只
// 改该行；「全选」位是三态，点一次改全部（G-H2-05 / G-H2-06）。
func TestPluginMarketDetailSkillsDefaultAllSelected(t *testing.T) {
	app, tt := openMarketDetailFor(t, marketSource(marketEntry()), &pluginTestIntents{})

	for _, want := range []string{"alpha", "第一条技能", "beta"} {
		if !tt.HasText(want) {
			t.Fatalf("技能清单缺少 %q：%v", want, tt.Texts())
		}
	}
	// 读屏名把名称与描述一起读出（G-H2-06 无障碍），两个技能都默认勾上。
	for _, name := range []string{"alpha，第一条技能", "beta"} {
		info, ok := tt.AccessByRole(gui.RoleCheckBox, name)
		if !ok || !info.Checked {
			t.Fatalf("技能 %q 默认没有勾上：%+v（找到 %v）", name, info, ok)
		}
	}
	// 点 alpha 只改 alpha。
	if err := tt.Click("alpha"); err != nil {
		t.Fatalf("点不到技能行：%v", err)
	}
	tt.Frame()
	if app.pluginManage.marketSkills["alpha"] {
		t.Fatal("点了技能行没有取消勾选")
	}
	if !app.pluginManage.marketSkills["beta"] {
		t.Fatal("点一行把别的技能也改了")
	}
	// 全选位此时是「部分」，点一次回到全选。
	info, ok := tt.AccessByRole(gui.RoleCheckBox, pluginMarketSelectAll)
	if !ok || !info.Mixed {
		t.Fatalf("全选位没有按部分勾选画：%+v（找到 %v）", info, ok)
	}
	clickAccessRole(t, tt, gui.RoleCheckBox, pluginMarketSelectAll)
	if !app.pluginManage.marketSkills["alpha"] || !app.pluginManage.marketSkills["beta"] {
		t.Fatal("全选没有把所有技能勾上")
	}
	// 再点一次：全不选也允许（G-H2-06：安装为无技能插件）。
	clickAccessRole(t, tt, gui.RoleCheckBox, pluginMarketSelectAll)
	if app.pluginManage.marketSkills["alpha"] || app.pluginManage.marketSkills["beta"] {
		t.Fatal("再点全选没有全部取消")
	}
}

// TestPluginMarketDetailEmptySkills：源给了空清单时画「此插件不包含技能。」，
// 不画分组标题以外的勾选行（G-H2-07）。
func TestPluginMarketDetailEmptySkills(t *testing.T) {
	entry := marketEntry()
	entry.Skills = []PluginMarketSkillSnapshot{}
	_, tt := openMarketDetailFor(t, marketSource(entry), &pluginTestIntents{})
	if !tt.HasText(pluginEmptySkills) {
		t.Fatalf("空技能清单没有画空态句：%v", tt.Texts())
	}
	// 无技能时描述仍在（G-H2-02 状态：说明与空态不冲突）。
	if !tt.HasText(pluginMarketInstallDesc) {
		t.Fatalf("无技能时描述不见了：%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(gui.RoleCheckBox, pluginMarketSelectAll); ok {
		t.Fatal("没有技能时仍画了「全选」")
	}
}

// TestPluginMarketDetailLoadingState：清单正在重拉时详情给加载行，主按钮禁用并
// 写原因（G-H2-03）。
func TestPluginMarketDetailLoadingState(t *testing.T) {
	src := marketSource(marketEntry())
	src.market.Loading = true
	_, tt := openMarketDetailFor(t, src, &pluginTestIntents{})
	if !tt.HasText(pluginMarketLoading) {
		t.Fatalf("没有画「正在加载...」：%v", tt.Texts())
	}
	// 标题只读条目名，不等详情（G-H2-01 状态）。
	if !tt.HasText("安装 market-plugin") {
		t.Fatalf("加载中标题丢了条目名：%v", tt.Texts())
	}
	info, ok := tt.Access(pluginAdd)
	if !ok || !info.Disabled {
		t.Fatalf("加载中「添加插件」没有置灰：%+v（找到 %v）", info, ok)
	}
}

// TestPluginMarketDetailLoadFailureRetries：清单拉取失败时给错误块与「重试」，
// 点「重试」真的重拉一次（G-H2-04）。
func TestPluginMarketDetailLoadFailureRetries(t *testing.T) {
	src := marketSource(marketEntry())
	src.market.Err = "插件源返回 HTTP 503"
	intents := &pluginTestIntents{refreshed: make(chan struct{}, 1)}
	app, tt := openMarketDetailFor(t, src, intents)

	if !tt.HasText("插件源返回 HTTP 503") {
		t.Fatalf("没有画失败原因：%v", tt.Texts())
	}
	if info, ok := tt.Access(pluginMarketRetryDetail); !ok || info.Disabled {
		t.Fatalf("失败块的「重试」不可点：%+v（找到 %v）", info, ok)
	}
	if !app.pluginManage.marketOpen {
		t.Fatal("失败把详情关掉了")
	}
	if !tt.HasText(pluginMarketDetailFailed) {
		t.Fatalf("读取失败时没有写主按钮置灰的原因：%v", tt.Texts())
	}
	clickAccessRole(t, tt, gui.RoleButton, pluginMarketRetryDetail)
	select {
	case <-intents.refreshed:
	case <-time.After(2 * time.Second):
		t.Fatal("点「重试」没有重拉清单")
	}
}

// TestPluginMarketDetailInstallsViaGitSource：条目给了 Git 地址时「添加插件」走
// plugin.install 的 Git 来源、装后启用；成功后关框并写「已安装 {name}」
// （G-H2-09 交互 / 成功回执）。
func TestPluginMarketDetailInstallsViaGitSource(t *testing.T) {
	intents := &pluginTestIntents{installed: make(chan pluginTestInstall, 1)}
	app, tt := openMarketDetailFor(t, marketSource(marketEntry()), intents)

	if info, ok := tt.Access(pluginAdd); !ok || info.Disabled {
		t.Fatalf("有条目地址与写通道时「添加插件」应当可点：%+v（找到 %v）", info, ok)
	}
	clickAccessRole(t, tt, gui.RoleButton, pluginAdd)
	select {
	case got := <-intents.installed:
		if got.kind != "git" || got.ref != "owner/repo" || got.gitRef != "main" || !got.enable {
			t.Fatalf("plugin.install 参数不对：%+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有发出 plugin.install")
	}
	waitPluginFrame(t, tt, func() bool { return !app.pluginManage.marketOpen })
	if !tt.HasText("已安装 market-plugin") {
		t.Fatalf("成功回执没有上屏：%v", tt.Texts())
	}
}

// TestPluginMarketDetailInstallFailureStays：安装失败时详情保留，原因可见，按钮
// 恢复可点（G-H2-09 异常）。
func TestPluginMarketDetailInstallFailureStays(t *testing.T) {
	intents := &pluginTestIntents{failInstall: errors.New("manifest 不被服务端接受")}
	app, tt := openMarketDetailFor(t, marketSource(marketEntry()), intents)

	clickAccessRole(t, tt, gui.RoleButton, pluginAdd)
	waitPluginFrame(t, tt, func() bool {
		return strings.Contains(app.pluginManage.marketErr, "manifest 不被服务端接受")
	})
	if !app.pluginManage.marketOpen {
		t.Fatal("失败把详情关掉了")
	}
	if !tt.HasText("添加插件：manifest 不被服务端接受") {
		t.Fatalf("失败原因没有上屏：%v", tt.Texts())
	}
	if info, ok := tt.Access(pluginAdd); !ok || info.Disabled {
		t.Fatalf("失败后主按钮没有恢复可点：%+v（找到 %v）", info, ok)
	}
}

// TestPluginMarketDetailInstallNeedsSource：条目没有 Git 地址时主按钮置灰并写
// 原因；同名插件已装时也不直接覆盖（替换确认 G-H3 未建）。
func TestPluginMarketDetailInstallNeedsSource(t *testing.T) {
	entry := marketEntry()
	entry.Git, entry.GitRef = "", ""
	_, tt := openMarketDetailFor(t, marketSource(entry), &pluginTestIntents{})
	if info, ok := tt.Access(pluginAdd); !ok || !info.Disabled {
		t.Fatalf("没有可安装地址时主按钮没有置灰：%+v（找到 %v）", info, ok)
	}
	if !tt.HasText(pluginMarketNoInstallSource) {
		t.Fatalf("没有写「没有可安装地址」的原因：%v", tt.Texts())
	}

	src := marketSource(marketEntry())
	src.extensions.Plugins = ExtListSnapshot{Items: []ExtItemSnapshot{
		{ID: "market-plugin", Name: "market-plugin", Version: "2.0.0", Enabled: true},
	}}
	_, tt = openMarketDetailFor(t, src, &pluginTestIntents{})
	if info, ok := tt.Access(pluginAdd); !ok || !info.Disabled {
		t.Fatalf("同名插件已装时主按钮没有置灰：%+v（找到 %v）", info, ok)
	}
	if !tt.HasText("已安装同名插件「market-plugin」") {
		t.Fatalf("没有写同名冲突的原因：%v", tt.Texts())
	}
}

// TestPluginMarketDetailInstallNeedsWriteChannel：没有写通道时主按钮置灰并写
// 「当前连接不支持写入」，取消仍可用。
func TestPluginMarketDetailInstallNeedsWriteChannel(t *testing.T) {
	app, tt := openMarketDetailFor(t, marketSource(marketEntry()), nil)
	if info, ok := tt.Access(pluginAdd); !ok || !info.Disabled {
		t.Fatalf("没有写通道时主按钮没有置灰：%+v（找到 %v）", info, ok)
	}
	if !tt.HasText(noteNoWriteChannel) {
		t.Fatalf("没有写缺通道的原因：%v", tt.Texts())
	}
	if err := tt.Click(pluginCancel); err != nil {
		t.Fatalf("点不到取消：%v", err)
	}
	tt.Frame()
	if app.pluginManage.marketOpen {
		t.Fatal("取消没有关掉详情")
	}
	if tt.HasText(pluginMarketInstallDesc) {
		t.Fatalf("关掉详情后描述仍在屏上：%v", tt.Texts())
	}
}

// TestPluginMarketDetailInstallingShowsSpinnerAndReason：安装中主按钮换成旋转
// 图标、取消禁用，进度条没有读数就写原因（G-H2-08 安装中 / G-H2-09 安装中）。
func TestPluginMarketDetailInstallingShowsSpinnerAndReason(t *testing.T) {
	app, tt := openMarketDetailFor(t, marketSource(marketEntry()), &pluginTestIntents{})
	app.pluginManageState().marketInstalling = true
	tt.Frame()

	if !tt.HasText(pluginAdding) {
		t.Fatalf("安装中没有把主按钮文案换掉：%v", tt.Texts())
	}
	if !tt.HasText(pluginMarketInstallProgress) {
		t.Fatalf("安装中没有写「没有进度读数」：%v", tt.Texts())
	}
	if info, ok := tt.Access(pluginAdding); !ok || !info.Disabled {
		t.Fatalf("安装中主按钮没有置灰：%+v（找到 %v）", info, ok)
	}
	if info, ok := tt.Access(pluginCancel); !ok || !info.Disabled {
		t.Fatalf("安装中「取消」没有置灰：%+v（找到 %v）", info, ok)
	}
}

// TestPluginMarketDetailEscCancels：按 Esc 等同「取消」；安装中 Esc 不生效
// （G-H2-08 键盘 / 异常）。
func TestPluginMarketDetailEscCancels(t *testing.T) {
	app, tt := openMarketDetailFor(t, marketSource(marketEntry()), &pluginTestIntents{})
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.pluginManage.marketOpen {
		t.Fatal("Esc 没有关掉详情")
	}

	// 安装中：框留着，不允许半途取消。
	app2, tt2 := openMarketDetailFor(t, marketSource(marketEntry()), &pluginTestIntents{})
	app2.pluginManageState().marketInstalling = true
	tt2.Frame()
	tt2.Key(0, gui.KeyEscape)
	tt2.Frame()
	if !app2.pluginManage.marketOpen {
		t.Fatal("安装中按 Esc 把详情关掉了")
	}
}
