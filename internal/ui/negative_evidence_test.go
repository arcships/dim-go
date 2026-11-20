package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是四份需求里「需负向取证」那一档的可执行证据：每一条断言的都是
// 「某操作之后，某个现象没有发生」——不出现、不残留、不重复、不发请求。
//
// 写法与仓库里其它用例一致：从屏幕上的入口做起（点、键入、按键、换读数），
// 断言落在看得见的文案、读屏项与写入口的调用次数上。负向用例的失败就是缺
// 陷，不改断言去迁就实现；每条用例的名字会写回需求表的状态列
// （已实现（证据 = <测试名>））。
//
// 每类一条用例：同一类里的条目共享同一条可观察的负向事实（见各用例的注释）。

// negScreenText 是屏上可见文案的整段文本。
func negScreenText(tt *gui.Tester) string { return strings.Join(tt.Texts(), "\n") }

// negAbsent 断言屏上没有任何一行包含这段文案（比 negAbsentExact 严在「包着它的
// 更长的行」也算出现）。
func negAbsent(t *testing.T, tt *gui.Tester, what string, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if tt.HasText(text) {
			t.Fatalf("%s：屏上出现了 %q\n屏上文字：\n%s", what, text, negScreenText(tt))
		}
	}
}

// negAbsentExact 断言屏上没有任何一个节点的文案恰好等于 text。
func negAbsentExact(t *testing.T, tt *gui.Tester, what string, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if _, ok := tt.Find(text); ok {
			t.Fatalf("%s：屏上有一行恰好是 %q\n屏上文字：\n%s", what, text, negScreenText(tt))
		}
	}
}

// negPresentExact 断言屏上有一行恰好是 text。
func negPresentExact(t *testing.T, tt *gui.Tester, what string, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if _, ok := tt.Find(text); !ok {
			t.Fatalf("%s：屏上没有一行恰好是 %q\n屏上文字：\n%s", what, text, negScreenText(tt))
		}
	}
}

// negPresent 断言这些文案都在屏上（证明负向断言不是空转：同一句话在触发态里
// 确实会画出来）。
func negPresent(t *testing.T, tt *gui.Tester, what string, texts ...string) {
	t.Helper()
	screen := negScreenText(tt)
	for _, text := range texts {
		if !strings.Contains(screen, text) {
			t.Fatalf("%s：屏上没有 %q\n屏上文字：\n%s", what, text, screen)
		}
	}
}

// negNoAccessible 断言这些名字不在读屏树里（按钮、开关这类没有可见文字的入口
// 只能按可读名查）。
func negNoAccessible(t *testing.T, tt *gui.Tester, names ...string) {
	t.Helper()
	for _, name := range names {
		if info, ok := tt.Access(name); ok {
			t.Fatalf("读屏树里仍有 %q（角色 %v，禁用 %v）", name, info.Role, info.Disabled)
		}
	}
}

// negCalls 是写入口被调用的次数（按方法名）。
func negCalls(f *failIntents) map[string]int {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]int, len(f.calls))
	for k, v := range f.calls {
		out[k] = v
	}
	return out
}

// negNoWrites 断言一次操作之后没有任何写入口被调用。
func negNoWrites(t *testing.T, f *failIntents, what string) {
	t.Helper()
	if calls := negCalls(f); len(calls) > 0 {
		t.Fatalf("%s：不该发出写入，实际发了 %v", what, calls)
	}
}

// ------------------------------------------------------------------ 输入区

// TestNegativeComposerHoldsFireWithoutDraftOrConnection：没有草稿、或连接已断
// 时，「发送」不发任何提交（R-IN-150、R-IN-043 一类的负向要求：点了没反应，
// 也不把空串发出去）。
func TestNegativeComposerHoldsFireWithoutDraftOrConnection(t *testing.T) {
	src := wiredSource()
	intents := newFailIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	// 空草稿：发送键在场但置灰；点了不发。
	if info, ok := tt.AccessByRole(gui.RoleButton, "发送"); !ok || !info.Disabled {
		t.Fatalf("空草稿时发送键没有置灰：%+v（找到 %v）", info, ok)
	}
	_ = tt.Click("发送")
	tt.Frame()
	negNoWrites(t, intents, "空草稿点发送")

	// 有草稿但断线：输入卡整块让位给断线说明，没有发送键可点。
	app.draft = "这条不该发出去"
	src.conn = Connection{Phase: "disconnected", Reason: "测试断线"}
	tt.Frame()
	negAbsent(t, tt, "断线时", "发送")
	negAbsent(t, tt, "断线时", "输入卡")
	negPresent(t, tt, "断线时", "连接已断开")
	_ = tt.Click("发送")
	tt.Frame()
	negNoWrites(t, intents, "断线后点发送")
}

// negBlockingIntents 让每次提交一直挂着，用来观察「请求还没回来时又点了几次」。
type negBlockingIntents struct {
	Intents
	mu      sync.Mutex
	submits int
	release chan struct{}
}

func (n *negBlockingIntents) SubmitWithContent(context.Context, string, []ContentBlockSnapshot) error {
	n.mu.Lock()
	n.submits++
	n.mu.Unlock()
	<-n.release
	return nil
}

func (n *negBlockingIntents) SetPreferences(context.Context, PreferencesSnapshot) error { return nil }

func (n *negBlockingIntents) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.submits
}

// TestNegativeRepeatedSubmitSendsOnce：发送请求还没回来时连点三次「发送」，
// 只发出一条提交（R-IN-136「防止二次提交」、R-IN-140「只发送一次」）。
func TestNegativeRepeatedSubmitSendsOnce(t *testing.T) {
	src := wiredSource()
	intents := newNegBlockingIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	defer close(intents.release)

	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点不到输入框：%v", err)
	}
	tt.Type("连点三次")
	tt.Frame()
	for i := 0; i < 3; i++ {
		if err := tt.Click("发送"); err != nil {
			t.Fatalf("第 %d 次点不到发送：%v", i+1, err)
		}
		tt.Frame()
	}
	deadline := time.Now().Add(3 * time.Second)
	for intents.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	if n := intents.count(); n != 1 {
		t.Fatalf("请求未回来时连点三次发出了 %d 条提交（期望 1 条）", n)
	}
}

func newNegBlockingIntents() *negBlockingIntents {
	return &negBlockingIntents{release: make(chan struct{})}
}

// TestNegativeApprovalStateHidesComposer：有待决审批时，输入位只剩审批卡：
// 不出现输入卡、发送键与停止键，工具调用在答复前不执行（R-IN-156、G-C1 的
// 「按钮不出现」几条）。
func TestNegativeApprovalStateHidesComposer(t *testing.T) {
	src := wiredSource()
	src.approval = ApprovalSnapshot{ID: "ap-1", Tool: "exec", Command: "rm -rf ./tmp/*", LongTerm: false}
	src.hasAppr = true
	intents := newFailIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	negPresent(t, tt, "审批态", "拒绝")
	negPresent(t, tt, "审批态", "仅这次")
	negAbsent(t, tt, "审批态", "输入卡")
	negNoAccessible(t, tt, "发送", "停止")
	// 答复之前工具不执行：没有任何写入口被调用。
	negNoWrites(t, intents, "审批态")
}

// TestNegativeApprovalWithoutLongTermHidesButton：审批快照没有给长期规则时，
// 长期允许按钮不出现——「不显示为禁用」（G-C1-20 / R-AP-214）。
func TestNegativeApprovalWithoutLongTermHidesButton(t *testing.T) {
	src := wiredSource()
	src.approval = ApprovalSnapshot{ID: "ap-1", Tool: "exec", Command: "rm -rf ./tmp/*", LongTerm: false}
	src.hasAppr = true
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	// 对照：这一句在给了长期规则时确实会上屏（证明断言不是空转）。
	src.approval.LongTerm = true
	tt.Frame()
	negPresent(t, tt, "审批态（可长期允许）", "始终允许此命令")

	src.approval.LongTerm = false
	tt.Frame()
	negNoAccessible(t, tt, "始终允许此命令")
	negAbsent(t, tt, "审批态（无长期规则）", "始终允许此命令")
}

// TestNegativeRunningStateHidesSendButton：运行中发送槽换成停止，屏上不再有
// 「发送」（R-IN-146、R-IN-155 的负向部分：两个键不同时在场）。
func TestNegativeRunningStateHidesSendButton(t *testing.T) {
	src := wiredSource()
	src.session.Running = true
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	if _, ok := tt.AccessByRole(gui.RoleButton, "停止"); !ok {
		t.Fatalf("运行中没有停止键：%v", tt.Texts())
	}
	negNoAccessible(t, tt, "发送")

	// 对照：同一屏在非运行态有发送键。
	src.session.Running = false
	tt.Frame()
	if _, ok := tt.AccessByRole(gui.RoleButton, "发送"); !ok {
		t.Fatalf("非运行态没有发送键：%v", tt.Texts())
	}
	negNoAccessible(t, tt, "停止")
}

// ------------------------------------------------------------------ 空态与切会话

// TestNegativeEmptyThreadShowsNoHistory：空态里没有任何历史消息与会话行
// （R-NT 一类的「不出现」：空态不是把样例铺上去）。
func TestNegativeEmptyThreadShowsNoHistory(t *testing.T) {
	app := NewApp(Options{State: StateEmpty})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	negPresent(t, tt, "空态", "开始构建")
	negAbsent(t, tt, "空态", "看看这次的失败", "先看结论", "修发送路径", "旧实现对照", "等待你同意")
	// 空态不画状态条：权限入口改由空态控制行承载。
	negAbsent(t, tt, "空态", "状态条")

	// 对照：默认态里这些内容确实在。
	app2 := NewApp(Options{State: StateDefault})
	tt2 := gui.NewTester(app2.View, WindowWidth, WindowHeight)
	tt2.Frame()
	negPresent(t, tt2, "默认态", "看看这次的失败", "修发送路径", "状态条")
}

// TestNegativeSessionSwitchLeavesNoStaleRows：切到另一个会话后，时间线按新会话
// 重建，不残留上一个会话的消息（R-XR-005、R-TB-163）。
func TestNegativeSessionSwitchLeavesNoStaleRows(t *testing.T) {
	src := wiredSource()
	src.names = []string{"s1", "s2"}
	src.titles = []string{"会话一", "会话二"}
	src.session = SessionSnapshot{ID: "s1", Title: "会话一", Rows: []RowSnapshot{
		{Kind: RowUser, Text: "会话一的问题"},
		{Kind: RowAssistant, Text: "会话一的回答"},
	}}
	intents := newFailIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	negPresent(t, tt, "会话一", "会话一的问题", "会话一的回答")

	// 点左栏第二个会话：宿主随即把读数换成该会话。
	if err := tt.Click("会话二"); err != nil {
		t.Fatalf("点不到会话二：%v", err)
	}
	src.session = SessionSnapshot{ID: "s2", Title: "会话二", Rows: []RowSnapshot{
		{Kind: RowUser, Text: "会话二的问题"},
	}}
	tt.Frame()

	negPresent(t, tt, "会话二", "会话二的问题", "会话 会话二")
	negAbsent(t, tt, "切会话后", "会话一的问题", "会话一的回答", "会话 会话一")
}

// TestNegativeSidebarTextClicksAreInert：左栏的标题文字不是按钮：点它们不折叠、
// 不排序、不跳转、不开菜单（R-LB-123、R-LB-160、R-LB-182、R-LB-198、R-LB-322、
// R-TB-152 一类）。
func TestNegativeSidebarTextClicksAreInert(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	for _, label := range []string{"会话", "排序", "视图", "已置顶", "暂无会话"} {
		if err := tt.Click(label); err != nil {
			continue // 这一态下没有这个标题（例如没有置顶时「已置顶」不渲染）
		}
		tt.Frame()
		if items := tt.Menu(); len(items) > 0 {
			t.Fatalf("点 %q 开出了菜单：%v", label, items)
		}
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("点 %q 之后这一行消失了（说明它被当成可折叠/可跳转的入口）", label)
		}
	}
	// 屏上仍是同一份会话列表：没有因为点标题而新建/切换会话。
	negPresent(t, tt, "点标题后", "修发送路径", "旧实现对照")
}

// TestNegativeNoPinnedGroupWithoutPins：没有置顶会话时，左栏不渲染「已置顶」这一
// 组（R-LB-112）；置顶后出现，取消置顶后再次消失。
func TestNegativeNoPinnedGroupWithoutPins(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	negAbsentExact(t, tt, "无置顶", "已置顶")

	if err := tt.RightClick("修发送路径"); err != nil {
		t.Fatalf("右键会话行：%v", err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem("置顶"); err != nil {
		t.Fatalf("菜单里点不到置顶：%v", err)
	}
	tt.Frame()
	negPresentExact(t, tt, "置顶后", "已置顶")

	if err := tt.RightClick("修发送路径"); err != nil {
		t.Fatalf("右键会话行：%v", err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem("取消置顶"); err != nil {
		t.Fatalf("菜单里点不到取消置顶：%v", err)
	}
	tt.Frame()
	negAbsentExact(t, tt, "取消置顶后", "已置顶")
}

// ------------------------------------------------------------------ 设置屏

// negSectionCopy 是每一节「只在这一节出现」的一句文案：用来判断切节后上一节的
// 内容有没有留在屏上。取的都是本屏独有、不在导航里重复出现的句子。
func negSectionCopy() map[settingsSectionID]string {
	return map[settingsSectionID]string{
		secAccount:         "退出登录",
		secUsage:           "不可用",
		secPersonalization: "界面模式",
		secAppearance:      "程序语言",
		secGeneral:         "保持电脑唤醒",
		secKeybindings:     "搜索快捷键",
		secProviders:       "搜索供应商",
		secModality:        "所有者任务",
		secRuntime:         "功能组件",
		secMCP:             "MCP 服务器",
		secNotifications:   "通知声音",
		secPets:            "显示 Pet",
		secPermissions:     "Exec command 权限",
		secGit:             "始终强制推送",
		secHooks:           "命令钩子",
		secPluginSettings:  "插件市场",
		secExternalAgents:  "App Server 尚未实现外部代理。此模式下 Desktop 无法检测或配置它们。",
		secArchived:        "暂无归档会话",
		secAbout:           "软件更新",
	}
}

// TestNegativeSettingsSectionsDoNotShowEachOther：逐节点过 19 节，每一节只画
// 自己那一节的内容：上一节的标题与说明不残留，也不会两节的内容同时在场
// （R-ACC-005、R-APP-003、R-CFG-083 一类）。
func TestNegativeSettingsSectionsDoNotShowEachOther(t *testing.T) {
	resetSettingsScreenState()
	copies := negSectionCopy()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	app.screen = ScreenSettings
	tt.Frame()

	for _, def := range settingsSectionDefs {
		if err := tt.Click(def.label); err != nil {
			t.Fatalf("点不到设置分节 %q：%v", def.label, err)
		}
		tt.Frame()
		if app.settingsSection != def.id {
			t.Fatalf("点 %q 后当前分节 = %d，想要 %d", def.label, app.settingsSection, def.id)
		}
		negPresentExact(t, tt, "分节 "+def.label, copies[def.id])
		for other, copy := range copies {
			if other == def.id {
				continue
			}
			negAbsentExact(t, tt, "分节 "+def.label, copy)
		}
	}
}

// negJunk 是文案不该出现的几种形状：i18n 键名、未定义值、占位对象。
var negJunk = []string{
	"option.label", "option.labelKey", "formError", "errorText",
	"undefined", "NaN", "[object Object]",
}

// TestNegativeScreensShowNoKeyNames：各屏都不把键名、`undefined`、`NaN` 这类
// 内部值画到界面上（R-PER-149、R-APP-103、R-CFG-146、R-MOD-026、R-LB-476 等）。
func TestNegativeScreensShowNoKeyNames(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	negAbsent(t, tt, "主界面", negJunk...)

	app.screen = ScreenSettings
	for _, def := range settingsSectionDefs {
		app.settingsSection = def.id
		tt.Frame()
		negAbsent(t, tt, "设置 › "+def.label, negJunk...)
	}

	app.screen = ScreenMain
	for _, v := range []Variant{VariantPalette, VariantSearch, VariantSessionMenu, VariantSessionRename,
		VariantSessionDelete, VariantBranch, VariantBranchEnd, VariantGoal} {
		app.variant = v
		tt.Frame()
		negAbsent(t, tt, fmt.Sprintf("走查态 %d", v), negJunk...)
	}
}

// TestNegativeAccountLeavesNoIdentityWithoutFacts：账户事实缺一项时，对应那一行
// 整行不渲染，也不留上一次的旧值（R-ACC-067、R-ACC-070、R-ACC-071、R-ACC-085、
// R-ACC-094）。
func TestNegativeAccountLeavesNoIdentityWithoutFacts(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.acct = AccountSnapshot{
		Initial: "A", Connected: true, Name: "Ada", Email: "ada@real.test", Status: "signed_in",
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.screen, app.settingsSection = ScreenSettings, secAccount
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	negPresent(t, tt, "已登录", "ada@real.test")

	// 没有邮箱：邮箱行整行不渲染（UID 行在另一条事实到位时才画，这里不掺进来）。
	src.acct.Email = ""
	tt.Frame()
	negAbsent(t, tt, "无邮箱", "ada@real.test")
	negPresent(t, tt, "无邮箱", "Ada")

	// 退出登录（登录态失效）：身份卡换成未登录卡，不残留旧邮箱。
	src.acct = AccountSnapshot{Connected: false}
	tt.Frame()
	negPresent(t, tt, "退出登录", "登录 Dim")
	negAbsent(t, tt, "退出登录", "ada@real.test", "退出登录")
}

// TestNegativeUsageShowsNoFabricatedNumbers：用量只有「App Server 只读」这一支
// 在本产品成立，屏上不出现需连接/加载/失败卡，也不出现任何聚合数字
// （R-USG-010、R-USG-019、R-USG-042、R-USG-048、R-USG-067、R-USG-082）。
func TestNegativeUsageShowsNoFabricatedNumbers(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.usage = UsageSnapshot{Available: true, Used: 7, Limit: 10, Unit: "GB"}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.screen, app.settingsSection = ScreenSettings, secUsage
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	negPresent(t, tt, "用量", "不可用")
	negAbsent(t, tt, "用量",
		"查看用量活动需要连接", "重试", "正在加载", "7", "10", "GB", "%")
}

// TestNegativeDisabledSettingsWriteNothing：设置里那些置灰/未接的入口，点下去
// 不写任何事实（R-PER-094、R-PER-097、R-GEN-099、R-TM-043 一类）。
func TestNegativeDisabledSettingsWriteNothing(t *testing.T) {
	cases := []struct {
		section settingsSectionID
		labels  []string
	}{
		{secGeneral, []string{"重启 App", "导入…", "清理数据…", "代理模式"}},
		{secPersonalization, []string{"展示 ACP 会话", "默认打开目标应用"}},
		{secPermissions, []string{"在沙箱中运行命令", "保存隔离设置"}},
		{secPets, []string{"选择 Pet", "显示 Pet"}},
		{secPluginSettings, []string{"添加插件"}},
		{secKeybindings, []string{"全局唤起开关"}},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("section-%d", c.section), func(t *testing.T) {
			resetSettingsScreenState()
			intents := newFailIntents()
			app := NewApp(Options{State: StateDefault, Source: wiredSource(), Intents: intents})
			app.screen, app.settingsSection = ScreenSettings, c.section
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			tt.Frame()
			for _, label := range c.labels {
				_ = tt.Click(label)
				tt.Frame()
			}
			// 全局唤起是本地偏好，允许落盘 SetPreferences；其余写入口一次都不该动。
			for method, n := range negCalls(intents) {
				if method == "SetPreferences" {
					continue
				}
				t.Fatalf("点了 %v 之后写入口 %s 被调用 %d 次", c.labels, method, n)
			}
		})
	}
}

// ------------------------------------------------------------------ 会话内搜索

// TestNegativeSessionSearchForgetsQuery：关掉搜索浮层再打开，查询不记忆：查询为
// 空时没有结果行，也不留上一次的命中（R-GS-055、R-GS-045、R-F-091）。
func TestNegativeSessionSearchForgetsQuery(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "失败")
	if !tt.HasText("个结果") {
		t.Fatalf("查询有命中时没有结果行：%v", tt.Texts())
	}

	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.sessionSearch.open {
		t.Fatal("Esc 没有关掉搜索浮层")
	}
	// 浮层卸载后输入框的可读名与命中一起消失，不留在时间线上。
	negAbsent(t, tt, "关闭浮层后", sessionSearchLabel, "个结果", "没有匹配项")
	openSessionSearchForTest(t, tt)
	if app.sessionSearch.query != "" {
		t.Fatalf("重新打开浮层时记着上一次的查询 %q", app.sessionSearch.query)
	}
	negAbsent(t, tt, "重开浮层", "个结果")
}

// TestNegativeSearchWithoutHitsHidesJumpButtons：查询没有命中时，不出现「上一个 /
// 下一个结果」这类无对象可跳的按钮（R-TB-091）。
func TestNegativeSearchWithoutHitsHidesJumpButtons(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "这段文字在样例时间线里不存在")

	negAbsent(t, tt, "零命中", "下一个搜索结果", "上一个搜索结果")
	// 结果计数只是文字：点它不跳转、不切换当前命中。
	if err := tt.Click("0 个结果"); err == nil {
		tt.Frame()
		if _, ok := tt.Find("0 个结果"); !ok {
			t.Fatal("点结果计数之后这一行变了（它被当成可点的入口）")
		}
	}

	// 对照：有命中时这两个入口在场。
	searchFor(t, tt, "失败")
	negPresent(t, tt, "有命中", "下一个搜索结果")
}

// TestNegativeStatusBarWithoutUsageShowsNoPercent：没有用量快照时，状态条的百分比
// 位显示「—」，不编 0%、也不出现 NaN（R-SB-158、R-XR-090）。
func TestNegativeStatusBarWithoutUsageShowsNoPercent(t *testing.T) {
	src := wiredSource()
	src.session.Rows = nil
	src.usage = UsageSnapshot{}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	negPresentExact(t, tt, "状态条", "—")
	negAbsent(t, tt, "状态条", "0%", "NaN", "undefined")
}

// ------------------------------------------------------------------ 右栏

// TestNegativeTasksWithoutSelectionShowNoDetail：后台进程页没有选中行时不画
// 详情区（R-TK-134），也不编一条「已结束」的假记录。
func TestNegativeTasksWithoutSelectionShowNoDetail(t *testing.T) {
	src := newFakePanelSource("/work")
	app := NewApp(Options{State: StateDefault, Source: src})
	app.rightOpen, app.rightTab = true, tabTasks
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	negPresent(t, tt, "后台进程页", "暂无运行进程")
	negAbsent(t, tt, "后台进程页", "结束于", "退出码", "复制命令")
}

// TestNegativeReviewWithoutChangesShowsNoActions：没有改动时不画分组与底部统计条
// （R-RD-255、R-RD-739、R-RD-749、R-RD-762）。
func TestNegativeReviewWithoutChangesShowsNoActions(t *testing.T) {
	src := newFakePanelSource("/work")
	src.changes = ChangeSet{Phase: PanelReadReady, Repo: true}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.rightOpen, app.rightTab = true, tabChanges
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	negPresent(t, tt, "改动页", "没有未提交的改动")
	negAbsent(t, tt, "改动页", "全部暂存", "取消暂存", "个文件改动")
}

// TestNegativeFilesWithoutEntriesShowNoRows：空目录里不画任何文件或目录行
// （R-F-123），也不出现底部「加载更多」（R-F-205）。
func TestNegativeFilesWithoutEntriesShowNoRows(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree()
	app := NewApp(Options{State: StateDefault, Source: src})
	app.rightOpen, app.rightTab = true, tabFiles
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	negAbsent(t, tt, "空目录", "加载更多")
	// 空目录的空态不是搜索空态：不写「没有找到包含“”的文件」，也不借改动页的
	// 「工作区是干净的」凑一句话。
	negAbsent(t, tt, "空目录", "没有找到包含", "工作区是干净的")
	// 树行（目录 / 文件）靠可读名前缀认得：空目录里一条都不该有。
	if info, ok := tt.AccessByRole(gui.RoleTreeItem, ""); ok && strings.HasPrefix(info.Name, "目录，") {
		t.Fatalf("空目录里画出了目录行 %q", info.Name)
	}
}
