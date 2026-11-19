package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是 docs/audit-fakes.md §1「P0 界面显示假事实」的回归断言：一条 P0 一个
// 用例，断言的都是同一件事——**没有事实时屏上不出现编造的值**。
//
// 每个用例都连着注入的 Source（连接模式），因为假事实只在有事实可读、界面却读
// 样例时才算假；-demo 的样例不在本文件的口径里（那条口径由
// TestWiredScreensExcludeSamples 的样例黑名单守）。

// p0Intents 接住本批断言用到的写入；其余方法走嵌入的 nil 接口，被调用即 panic：
// 说明测试点到了一个没接的动作。
type p0Intents struct {
	Intents
	archived chan p0Archive
	renamed  chan [2]string
	prefs    chan PreferencesSnapshot
}

// p0Archive 是一次 ArchiveSession 的实参。
type p0Archive struct {
	id string
	on bool
}

func (p *p0Intents) SetPreferences(_ context.Context, prefs PreferencesSnapshot) error {
	if p.prefs != nil {
		p.prefs <- prefs
	}
	return nil
}

func (p *p0Intents) ArchiveSession(_ context.Context, sessionID string, archived bool) error {
	if p.archived != nil {
		p.archived <- p0Archive{id: sessionID, on: archived}
	}
	return nil
}

func (p *p0Intents) RenameSession(_ context.Context, sessionID, title string) error {
	if p.renamed != nil {
		p.renamed <- [2]string{sessionID, title}
	}
	return nil
}

// p0WindowText 渲染一帧并把可见文字拼成一段，便于断言"某个串根本没上屏"。
func p0WindowText(tt *gui.Tester) string { return strings.Join(tt.Texts(), "\n") }

// p0ButtonsDisabled 断言这几个入口都在场且都置灰。
func p0ButtonsDisabled(t *testing.T, tt *gui.Tester, labels ...string) {
	t.Helper()
	for _, label := range labels {
		info, ok := tt.Access(label)
		if !ok {
			t.Fatalf("找不到入口 %q：%v", label, tt.Texts())
		}
		if !info.Disabled {
			t.Fatalf("入口 %q 没有置灰：%+v", label, info)
		}
	}
}

// P0-1 审批卡的 cwd 恒为样例常量、空命令回落 "ls -la"：审批事实里没有 cwd 这一项，
// 命令为空时也不能替用户编一条要执行的命令。
func TestApprovalCardShowsOnlyCommandFact(t *testing.T) {
	t.Run("命令为空", func(t *testing.T) {
		src := wiredSource()
		src.hasAppr = true
		src.approval = ApprovalSnapshot{ID: "a1", Command: ""}
		app := NewApp(Options{State: StateDefault, Source: src})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

		text := p0WindowText(tt)
		for _, fake := range []string{"ls -la", "~/dim-agent", "cwd: "} {
			if strings.Contains(text, fake) {
				t.Fatalf("审批卡在命令为空时出现了编造值 %q：\n%s", fake, text)
			}
		}
		if !tt.HasText(noteApprovalCommand) {
			t.Fatalf("命令为空时没有写清读不到：\n%s", text)
		}
		p0ButtonsDisabled(t, tt, "拒绝", "仅这次", "始终允许此命令")
	})

	t.Run("命令来自事实", func(t *testing.T) {
		src := wiredSource()
		src.hasAppr = true
		src.approval = ApprovalSnapshot{ID: "a2", Command: "rm -rf /tmp/x"}
		app := NewApp(Options{State: StateDefault, Source: src})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

		text := p0WindowText(tt)
		if !strings.Contains(text, "rm -rf /tmp/x") {
			t.Fatalf("审批卡没有画事实里的命令：\n%s", text)
		}
		for _, fake := range []string{"~/dim-agent", "cwd: "} {
			if strings.Contains(text, fake) {
				t.Fatalf("审批卡画了没有事实来源的 %q：\n%s", fake, text)
			}
		}
		if info, ok := tt.Access("仅这次"); !ok || info.Disabled {
			t.Fatalf("命令读到了，决定却不可点：%+v（找到 %v）", info, ok)
		}
	})
}

// P0-2 模型槽在 Settings().Model 为空时回落样例品牌名：没有模型事实就说没有。
func TestModelSlotShowsUnsetWithoutModelFact(t *testing.T) {
	src := wiredSource()
	src.settings.Model, src.settings.ModelID, src.settings.Models = "", "", nil
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if got := app.currentModel(); got != modelUnset {
		t.Fatalf("当前模型 = %q，想要 %q", got, modelUnset)
	}
	text := p0WindowText(tt)
	if strings.Contains(text, sampleModel) {
		t.Fatalf("没有模型事实时屏上出现样例模型 %q：\n%s", sampleModel, text)
	}
	if !strings.Contains(text, modelUnset) {
		t.Fatalf("没有模型事实时没有显示 %q：\n%s", modelUnset, text)
	}
}

// P0-6 运行环境恒说「服务端未提供 runtime.list」：适配器早就在读
// Settings().Runtimes，屏上要读它。
func TestRuntimeScreenReadsRuntimes(t *testing.T) {
	src := wiredSource()
	src.settings.Runtimes = []RuntimeSnapshot{{
		Name: "server-real", Version: "v9.0.0", Status: "ready",
	}}
	src.settings.RuntimesKnown = true
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secRuntime)

	text := p0WindowText(tt)
	if !tt.HasText("server-real") || !tt.HasText("v9.0.0") || !tt.HasText("ready") {
		t.Fatalf("运行环境屏没有画读数：\n%s", text)
	}
	if tt.HasText(rtCopyError) {
		t.Fatalf("服务端给了清单，却仍断言「未提供」：\n%s", text)
	}
	if tt.HasText(noteRuntime) {
		t.Fatalf("运行环境只读不装，不该再说没有读取：\n%s", text)
	}
}

// P0-6 的另一半：没读到时写适配器给的原因，不写死那句英文。
func TestRuntimeScreenWritesReadReasonWhenUnread(t *testing.T) {
	src := wiredSource()
	src.settings.Runtimes = nil
	src.settings.RuntimesKnown = false
	src.settings.RuntimesReason = "连接层还没有读到运行环境"
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secRuntime)

	text := p0WindowText(tt)
	if !tt.HasText("连接层还没有读到运行环境") {
		t.Fatalf("没有用适配器给的原因：\n%s", text)
	}
	if tt.HasText(rtCopyError) {
		t.Fatalf("没读到时仍写死「服务端未提供」：\n%s", text)
	}
	if tt.HasText("配套版本") {
		t.Fatalf("没有安装状态事实却画了配套版本：\n%s", text)
	}
}

// P0-7 多模态恒说「未声明能力 / 没有任务」：读 Modality 与 ModalityTasks。
func TestModalityScreenReadsFacts(t *testing.T) {
	src := wiredSource()
	src.settings.Modality = ModalitySnapshot{
		Known: true, Supported: true, TaskSupported: true, TaskKnown: true,
	}
	src.settings.ModalityTasks = []ModalityTaskSnapshot{{
		ID: "task-real", Category: "图像", Operation: "生成", State: "running", Progress: "42%",
	}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secModality)

	text := p0WindowText(tt)
	if !tt.HasText("task-real") || !tt.HasText("42%") || !tt.HasText("running") {
		t.Fatalf("多模态屏没有画任务读数：\n%s", text)
	}
	for _, fake := range []string{mdCopyUnavailable, mdCopyNoTasks, "0 / 4"} {
		if tt.HasText(fake) {
			t.Fatalf("服务端声明了能力与任务，屏上仍有 %q：\n%s", fake, text)
		}
	}
}

// P0-7：没读到时说没读到；声明不可用时把服务端的原因写出来。
func TestModalityScreenCapabilityBranches(t *testing.T) {
	t.Run("读数未到", func(t *testing.T) {
		src := wiredSource()
		src.settings.Modality = ModalitySnapshot{}
		app := NewApp(Options{State: StateDefault, Source: src})
		tt := openSettingsSection(t, app, secModality)
		if !tt.HasText(mdCopyCapabilityUnread) {
			t.Fatalf("没有写「读不到」：%v", tt.Texts())
		}
		if tt.HasText(mdCopyUnavailable) {
			t.Fatalf("没读到就断言「未声明」：%v", tt.Texts())
		}
		if tt.HasText("0 / 4") {
			t.Fatalf("没有分组事实却画了样例计数：%v", tt.Texts())
		}
	})

	t.Run("服务端声明不可用", func(t *testing.T) {
		src := wiredSource()
		src.settings.Modality = ModalitySnapshot{
			Known: true, Supported: false, Reason: "服务端关闭了模态能力",
		}
		app := NewApp(Options{State: StateDefault, Source: src})
		tt := openSettingsSection(t, app, secModality)
		if !tt.HasText("服务端关闭了模态能力") {
			t.Fatalf("没有写服务端给的原因：%v", tt.Texts())
		}
		if tt.HasText(mdCopyUnavailable) {
			t.Fatalf("仍写死「未声明 capabilities」：%v", tt.Texts())
		}
	})
}

// P0-3 / P1-11 设置 › Git 画样例远程与基线分支：没有仓库设置事实时，参考里的
// 控件形态照留但全部置灰，原因写在原因条与 tooltip 上，样例清单不上屏。
func TestGitScreenKeepsFormWithoutSampleRemotes(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secGit)

	text := p0WindowText(tt)
	if !tt.HasText(noteGitSettings) {
		t.Fatalf("没有写这一屏读不到设置：\n%s", text)
	}
	for _, sample := range []string{"origin", "upstream", "develop", "release"} {
		if tt.HasText(sample) {
			t.Fatalf("连着 Source 仍画了样例 %q：\n%s", sample, text)
		}
	}
	// 形态在：两个下拉都在场，取值位置写"未读取"，整钮置灰、tooltip 写原因。
	for _, label := range []string{gitCopyRemoteA11y, gitCopyBaseA11y} {
		info, ok := tt.AccessByRole(gui.RoleButton, label)
		if !ok {
			t.Fatalf("连着 Source 时 %q 下拉不见了（形态要留）：\n%s", label, text)
		}
		if !info.Disabled || info.Description != noteGitSettings {
			t.Fatalf("%q 没有置灰或没写原因：%+v", label, info)
		}
	}
	if !tt.HasText(gitCopyUnread) {
		t.Fatalf("下拉的取值位置没有写「%s」：\n%s", gitCopyUnread, text)
	}
	for _, label := range []string{"始终强制推送", "提交后自动推送"} {
		info, ok := tt.AccessByRole(gui.RoleSwitch, label)
		if !ok {
			t.Fatalf("连着 Source 时 %q 开关不见了（形态要留）", label)
		}
		if !info.Disabled {
			t.Fatalf("%q 可以拨（拨了不生效）：%+v", label, info)
		}
	}
	if !tt.HasText("分支前缀") || !tt.HasText("Worktree 根目录") {
		t.Fatalf("分组标题没有保留：\n%s", text)
	}
}

// P0-4 设置 › 宠物画样例 Pet 且 dim-fox 已选中：参考里的开关与动作按钮照留、全部
// 置灰，列表区是写着原因的虚线空态，样例 Pet 不上屏。
func TestPetsScreenKeepsFormWithoutSamplePets(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secPets)

	text := p0WindowText(tt)
	if !tt.HasText(notePetSettings) {
		t.Fatalf("没有写这一屏读不到 Pet：\n%s", text)
	}
	if tt.HasText("dim-fox") {
		t.Fatalf("连着 Source 仍画了样例 Pet：\n%s", text)
	}
	if _, ok := tt.Find("已选择"); ok {
		t.Fatalf("连着 Source 仍画了「已选择」：\n%s", text)
	}
	if info, ok := tt.AccessByRole(gui.RoleSwitch, petsCopyShown); !ok || !info.Disabled {
		t.Fatalf("「显示 Pet」开关没有留下或没置灰：%+v（找到 %v）", info, ok)
	}
	for _, label := range []string{petsCopyCreate, petsCopyInstall, petsCopyRefresh, petsCopyOpenDir} {
		info, ok := tt.AccessByRole(gui.RoleButton, label)
		if !ok || !info.Disabled {
			t.Fatalf("动作 %q 没有留下或没置灰：%+v（找到 %v）", label, info, ok)
		}
	}
	if !tt.HasText(petsCopyChoose) || !tt.HasText(petsCopyCustom) {
		t.Fatalf("分组标题没有保留：\n%s", text)
	}
}

// P0-5 设置 › 插件设置画样例插件卡与字段：卡形与卡底两个按钮照留、置灰，卡内只
// 写原因，样例插件名不上屏。
func TestPluginSettingsScreenKeepsFormWithoutSamplePlugins(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secPluginSettings)

	text := p0WindowText(tt)
	if !tt.HasText(notePluginSettings) {
		t.Fatalf("没有写这一屏读不到插件设置：\n%s", text)
	}
	for _, sample := range []string{"Model log", "Prompt suggestion", "Max Chars"} {
		if tt.HasText(sample) {
			t.Fatalf("连着 Source 仍画了样例 %q：\n%s", sample, text)
		}
	}
	for _, label := range []string{plgCopySave, plgCopyReload} {
		info, ok := tt.AccessByRole(gui.RoleButton, label)
		if !ok || !info.Disabled {
			t.Fatalf("卡底按钮 %q 没有留下或没置灰：%+v（找到 %v）", label, info, ok)
		}
	}
}

// P0-8 外观主题卡的选中态与真正决定主题的 themePref 无关（点卡也不换主题）：
// 卡片要跟 themePref 走，点卡要真的改主题并落盘。
func TestAppearanceThemeCardAppliesAndFollowsPreference(t *testing.T) {
	src := wiredSource()
	src.prefs = PreferencesSnapshot{Theme: "暗"}
	intents := &p0Intents{prefs: make(chan PreferencesSnapshot, 8)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	if app.themePref != "暗" {
		t.Fatalf("偏好里的主题 = %q，想要「暗」", app.themePref)
	}

	tt := openSettingsSection(t, app, secAppearance)
	// 主题卡是单选项（RoleRadio），选中态在辅助技术那边读成 Checked。
	if info, ok := tt.Access("主题：深色"); !ok || !info.Checked {
		t.Fatalf("选中角标没有跟着 themePref：%+v（找到 %v）", info, ok)
	}

	if err := tt.Click("主题：浅色"); err != nil {
		t.Fatalf("点不到浅色卡：%v", err)
	}
	tt.Frame()
	if app.themePref != "明" {
		t.Fatalf("点卡后 themePref = %q，想要「明」（主题必须真的换）", app.themePref)
	}
	select {
	case prefs := <-intents.prefs:
		if prefs.Theme != "明" {
			t.Fatalf("落盘的主题 = %q，想要「明」", prefs.Theme)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("点主题卡没有落盘偏好")
	}
	if info, ok := tt.Access("主题：浅色"); !ok || !info.Checked {
		t.Fatalf("点卡后选中角标没有移到浅色：%+v（找到 %v）", info, ok)
	}
	if info, _ := tt.Access("主题：深色"); info.Checked {
		t.Fatal("点浅色后深色仍是选中态")
	}
}

// P0-9 设置 › 通用的开关只改内存且初值是样例：参考的开关形态照留，但整钮置灰、
// tooltip 与行下写原因——位置不代表现状。
// 存储用量是这一屏唯一有事实的项，要读 Settings().Storage。
func TestGeneralScreenDisablesSampleToggles(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secGeneral)

	if !tt.HasText(noteDesktopSettings) {
		t.Fatalf("没有写这些桌面设置读不到：\n%s", p0WindowText(tt))
	}
	for _, label := range []string{"保持电脑唤醒", "性能日志", "登录时自动启动"} {
		info, ok := tt.AccessByRole(gui.RoleSwitch, label)
		if !ok {
			t.Fatalf("%q 开关不见了（形态要留）", label)
		}
		if !info.Disabled || info.Description != noteDesktopSettings {
			t.Fatalf("%q 可以拨或没写原因：%+v", label, info)
		}
	}
}

// 存储报告：有读数画读数，没读到写没读到，都不写 0 B。
func TestGeneralStorageReportReadsFacts(t *testing.T) {
	src := wiredSource()
	src.settings.Storage = StorageSnapshot{DatabaseBytes: 1536, WALBytes: 2048, ReclaimableBytes: 4096}
	src.settings.StorageKnown = true
	app := NewApp(Options{State: StateDefault, Source: src})
	app.screen, app.settingsSection = ScreenSettings, secGeneral
	generalScreen.cleanupOpen = true
	defer func() { generalScreen.cleanupOpen = false }()

	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	text := p0WindowText(tt)
	for _, want := range []string{"1.5 KiB", "2.0 KiB", "4.0 KiB"} {
		if !tt.HasText(want) {
			t.Fatalf("存储报告没有画读数 %q：\n%s", want, text)
		}
	}
	if tt.HasText("0 B") {
		t.Fatalf("有读数却仍画样例 0 B：\n%s", text)
	}

	src.settings.StorageKnown = false
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	text = p0WindowText(tt)
	if !tt.HasText(noteStorageUnread) {
		t.Fatalf("没读到时没有写原因：\n%s", text)
	}
	if tt.HasText("0 B") {
		t.Fatalf("没读到时画了假的 0 B：\n%s", text)
	}
}

// P0-10 设置 › 个性化的控件只改屏内状态：没有写入通道就置灰，点击不改值。
func TestPersonalizationControlsDisabledWithoutChannel(t *testing.T) {
	resetSettingsScreenState()
	psUI.uiMode = psModeOffice
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secPersonalization)

	if !tt.HasText(notePersonalizationWrite) {
		t.Fatalf("没有写这项设置读不到：\n%s", p0WindowText(tt))
	}
	_ = tt.Click("编程模式")
	tt.Frame()
	if psUI.uiMode != psModeOffice {
		t.Fatal("置灰的界面模式卡仍改了屏内状态")
	}
	if info, ok := tt.AccessByRole(gui.RoleSwitch, psCopyShowACP); !ok || !info.Disabled {
		t.Fatalf("「展示 ACP 会话」开关没有留下或没置灰：%+v（找到 %v）", info, ok)
	}
	if info, ok := tt.Access(psCopyOpenAgentsFile); !ok || !info.Disabled {
		t.Fatalf("「打开或创建文件」没有置灰：%+v（找到 %v）", info, ok)
	}
}

// P0-11 设置 › 权限的两个隔离开关只改内存、且初值是样例：参考的开关形态照留，
// 但读不到当前值就整钮置灰并写原因（-demo 仍然可拨，样例是那一档的常态）。
func TestPermissionsIsolationDisablesSampleSwitches(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secPermissions)

	if !tt.HasText(noteIsolationUnread) {
		t.Fatalf("没有写隔离设置读不到：\n%s", p0WindowText(tt))
	}
	for _, label := range []string{permCopySandbox, permCopyNetwork} {
		info, ok := tt.AccessByRole(gui.RoleSwitch, label)
		if !ok {
			t.Fatalf("%q 开关不见了（形态要留）", label)
		}
		if !info.Disabled {
			t.Fatalf("%q 可以拨（拨了不生效）：%+v", label, info)
		}
	}

	demo := NewApp(Options{State: StateDefault})
	dtt := openSettingsSection(t, demo, secPermissions)
	if info, ok := dtt.AccessByRole(gui.RoleSwitch, permCopySandbox); !ok || info.Disabled {
		t.Fatalf("-demo 的样例开关被一起禁用了（样例只该在连接模式外出现）：%+v（找到 %v）", info, ok)
	}
}

// P0-12 标题栏自编回执：归档要真的发 session.archive；重命名不能假禁用；
// 打开方式没有通道就置灰写原因，不写「正在用 X 打开」。
func TestHeaderMenuUsesRealSessionActions(t *testing.T) {
	src := wiredSource()
	intents := &p0Intents{
		archived: make(chan p0Archive, 1),
		renamed:  make(chan [2]string, 1),
		prefs:    make(chan PreferencesSnapshot, 8),
	}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯ 菜单：%v", err)
	}
	tt.Frame()
	if err := tt.Click("归档"); err != nil {
		t.Fatalf("点不到归档：%v", err)
	}
	select {
	case got := <-intents.archived:
		if got.id != "s1" || !got.on {
			t.Fatalf("ArchiveSession 收到 %+v，想要 s1/true", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("归档没有发出 session.archive")
	}
	if app.headerNote == "已归档" {
		t.Fatal("归档仍写自编回执「已归档」")
	}
}

func TestHeaderRenameIsAvailable(t *testing.T) {
	src := wiredSource()
	intents := &p0Intents{
		renamed: make(chan [2]string, 1),
		prefs:   make(chan PreferencesSnapshot, 8),
	}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯ 菜单：%v", err)
	}
	tt.Frame()
	info, ok := tt.Access("重命名")
	if !ok {
		t.Fatalf("菜单里没有重命名：%v", tt.Texts())
	}
	if info.Disabled {
		t.Fatalf("重命名被假禁用：%+v", info)
	}
	if info.Description != "" {
		t.Fatalf("重命名仍带假括注 %q", info.Description)
	}
	if err := tt.Click("重命名"); err != nil {
		t.Fatalf("点不到重命名：%v", err)
	}
	tt.Frame()
	if !app.renameDialog.open || app.renameDialog.kind != renameSession {
		t.Fatalf("重命名没有打开弹窗（open=%v kind=%v）", app.renameDialog.open, app.renameDialog.kind)
	}
	if !tt.HasText("重命名会话") {
		t.Fatalf("弹窗没有标题：%v", tt.Texts())
	}
	app.renameDialog.draft = "新的名字"
	tt.Frame()
	if err := tt.Click("确认"); err != nil {
		t.Fatalf("点不到确认：%v", err)
	}
	select {
	case got := <-intents.renamed:
		if got[0] != "s1" || got[1] != "新的名字" {
			t.Fatalf("RenameSession 收到 %v，想要 s1 + 新的名字", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("确认重命名没有发出 session.rename")
	}
	tt.Frame()
	if app.renameDialog.open {
		t.Fatal("重命名成功后弹窗没有关闭")
	}
}

func TestHeaderOpenWithDisabledWithoutChannel(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if info, ok := tt.Access("打开方式"); !ok || !info.Disabled {
		t.Fatalf("「打开方式」没有置灰：%+v（找到 %v）", info, ok)
	}
	if tt.HasText("正在用") {
		t.Fatalf("屏上出现了自编回执：\n%s", p0WindowText(tt))
	}
	if err := tt.Click("更多打开方式"); err != nil {
		t.Fatalf("点不到更多打开方式：%v", err)
	}
	tt.Frame()
	text := p0WindowText(tt)
	if !tt.HasText(noteOpenWith) {
		t.Fatalf("没有写打开方式读不到：\n%s", text)
	}
	for _, app := range []string{"Ghostty", "Zed", "VS Code"} {
		if tt.HasText(app) {
			t.Fatalf("没有应用清单却列出了 %q：\n%s", app, text)
		}
	}
	for _, label := range []string{"在 Finder 中显示", "用系统默认方式打开"} {
		if info, ok := tt.Access(label); !ok || !info.Disabled {
			t.Fatalf("%q 没有置灰：%+v（找到 %v）", label, info, ok)
		}
	}
}
