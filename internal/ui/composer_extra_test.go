package ui

import (
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 活动条：待办 / 子代理 / 分支 / 唤醒 / 进程 / 文件改动六项都从事实来；
// 点「进程」打开右栏任务页，点文件改动打开改动页。
func TestActivityBarChipsFromFacts(t *testing.T) {
	src := wiredSource()
	src.session = SessionSnapshot{
		ID: "s1", Branch: "b2",
		Branches: []BranchSnapshot{
			{ID: "b1", Name: "b1", Status: "ended", Ended: true},
			{ID: "b2", Name: "b2", Status: "active"},
			{ID: "b3", Name: "b3", Status: "active"},
		},
		Tasks: []TaskSnapshot{{ID: "t1", Description: "后台跑测试"}},
		Rows: []RowSnapshot{
			{Kind: RowTool, Title: "todo_write", Status: RowStatusRunning},
			{Kind: RowTool, Title: "task", Args: "子代理 A", Status: RowStatusRunning},
			{Kind: RowFileChanges, Children: []RowSnapshot{
				{Kind: RowFileChange, Text: "internal/ui/source.go"},
			}},
		},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	for _, want := range []string{"待办", "子代理", "分支 2", "唤醒 1", "进程 1", "1 个文件改动"} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("活动条缺 %q：%v", want, tt.Texts())
		}
	}
	if err := tt.Click("进程 1"); err != nil {
		t.Fatalf("点不到进程 chip：%v", err)
	}
	if !app.rightOpen || app.rightTab != tabTasks {
		t.Fatalf("进程 chip 没有打开任务页：open=%v tab=%d", app.rightOpen, app.rightTab)
	}
	if err := tt.Click("活动条：1 个文件改动"); err != nil {
		t.Fatalf("点不到文件改动 chip：%v", err)
	}
	if app.rightTab != tabChanges {
		t.Fatalf("文件改动 chip 没有打开改动页：tab=%d", app.rightTab)
	}
}

// 没有事实时不摆 chip：空会话没有活动条。
func TestActivityBarEmptyWithoutFacts(t *testing.T) {
	src := wiredSource()
	src.schedules = SchedulesSnapshot{}
	src.session = SessionSnapshot{ID: "s1"}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("活动条"); ok {
		t.Fatalf("没有事实时仍画了活动条：%v", tt.Texts())
	}
}

// 紧凑浮层的 `+` 附件菜单：添加附件、Agent / 目标子菜单与技能三个入口。
func TestAttachmentMenuEntries(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, CompactWidth-40, WindowHeight)
	if err := tt.Click("附件"); err != nil {
		t.Fatalf("点不到紧凑态 +：%v", err)
	}
	tt.Frame()
	for _, want := range []string{"添加附件", "Agent / 目标", "技能"} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("附件菜单缺 %q：%v", want, tt.Texts())
		}
	}
	if err := tt.Click("添加附件"); err != nil {
		t.Fatalf("点不到添加附件：%v", err)
	}
	tt.Frame()
	if !app.attachmentEditing {
		t.Fatal("点「添加附件」没有打开附件编辑")
	}
}

// 模型浮层四态：加载中 / 加载失败 / 未配置 / 暂无可用模型。
func TestModelOverlayStates(t *testing.T) {
	cases := []struct {
		name     string
		settings SettingsSnapshot
		want     string
	}{
		{"loading", SettingsSnapshot{}, "模型加载中…"},
		{"failed", SettingsSnapshot{ModelsError: "读取失败"}, "加载失败"},
		{"unconfigured", SettingsSnapshot{ModelsKnown: true}, "未配置"},
		{"empty", SettingsSnapshot{ModelsKnown: true,
			Providers: []ProviderOption{{ID: "acme", Name: "Acme"}}}, "暂无可用模型"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := wiredSource()
			src.settings = tc.settings
			app := NewApp(Options{State: StateDefault, Source: src})
			app.modelOpen = true
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			if !tt.HasText(tc.want) {
				t.Fatalf("模型浮层没有画 %q：%v", tc.want, tt.Texts())
			}
		})
	}
}

// 加载失败时的「重试」没有通道：入口在，但置灰并写原因。
func TestModelOverlayRetryDisabled(t *testing.T) {
	src := wiredSource()
	src.settings = SettingsSnapshot{ModelsError: "服务端拒绝"}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.modelOpen = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	info, ok := tt.Access("重试")
	if !ok {
		t.Fatalf("加载失败没有重试入口：%v", tt.Texts())
	}
	if !info.Disabled || info.Description != noteModelsRetry {
		t.Fatalf("重试没有置灰或没写原因：%+v", info)
	}
}

// 子代理页与外部 Agent 页：没有策略事实时写明原因，不拿模型列表顶替。
func TestModelOverlayTabs(t *testing.T) {
	src := wiredSource()
	app := NewApp(Options{State: StateDefault, Source: src})
	app.modelOpen = true
	app.modelTab = 1
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText(noteSubagentFacts) {
		t.Fatalf("子代理页没有写原因：%v", tt.Texts())
	}
	app.modelTab = 2
	tt.Frame()
	if !tt.HasText("App Server 尚未实现外部代理") {
		t.Fatalf("外部 Agent 页没有写原因：%v", tt.Texts())
	}
}

// priority 服务档画 Zap；ZDR 没有事实时在浮层底部写原因。
func TestModelOverlayPriorityAndZDR(t *testing.T) {
	src := wiredSource()
	src.settings.ServiceTier = "priority"
	app := NewApp(Options{State: StateDefault, Source: src})
	app.modelOpen = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("图标：priority"); !ok {
		t.Fatalf("priority 服务档没有图标：%v", tt.Texts())
	}
	if !tt.HasText(noteZDR) {
		t.Fatalf("没有写 ZDR 缺事实的原因：%v", tt.Texts())
	}
	src.settings.ServiceTier = ""
	tt.Frame()
	if _, ok := tt.Find("图标：priority"); ok {
		t.Fatal("没有 priority 事实时仍画了图标")
	}
}

// Esc 两段确认：首次 Esc 武装并出现键帽，3 秒内第二次 Esc 停止；点停止键一点即停，
// 不进入武装态。
func TestStopArmAndEscKeycap(t *testing.T) {
	app := NewApp(Options{State: StateRunning, Intents: newConfirmIntents()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if !app.stopArmed {
		t.Fatal("第一次 Esc 没有武装停止确认")
	}
	if _, ok := tt.Find("Esc 键：再按一次停止"); !ok {
		t.Fatalf("武装后没有 Esc 键帽：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.stopArmed {
		t.Fatal("第二次 Esc 后武装态没有撤除")
	}
	if _, ok := tt.Find("Esc 键：再按一次停止"); ok {
		t.Fatal("停止后 Esc 键帽还在")
	}
	select {
	case <-app.intents.(*stubConfirmIntents).stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("第二次 Esc 没有发停止写入")
	}
}

// 武装满 3 秒自动撤除；之后按 Esc 重新武装，而不是直接停止。
func TestStopArmExpiresThenRearms(t *testing.T) {
	app := NewApp(Options{State: StateRunning, Intents: newConfirmIntents()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if !app.stopArmed {
		t.Fatal("第一次 Esc 没有武装")
	}
	app.stopArmedAt = time.Now().Add(-stopConfirmWindow - time.Second)
	tt.Frame()
	if app.stopArmed {
		t.Fatal("超过 3 秒后武装态没有撤除")
	}
	if _, ok := tt.Find("Esc 键：再按一次停止"); ok {
		t.Fatal("超时后键帽还在")
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if !app.stopArmed {
		t.Fatal("超时后再按 Esc 没有重新武装")
	}
	select {
	case <-app.intents.(*stubConfirmIntents).stopped:
		t.Fatal("超时后第一次 Esc 不该停止")
	default:
	}
}

func TestStopButtonStopsOnceWithoutArming(t *testing.T) {
	app := NewApp(Options{State: StateRunning, Intents: newConfirmIntents()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("停止"); err != nil {
		t.Fatalf("点不到停止：%v", err)
	}
	if app.stopArmed {
		t.Fatal("点停止键进入了 Esc 武装态")
	}
	select {
	case <-app.intents.(*stubConfirmIntents).stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("点停止键没有发停止写入")
	}
}

// 发送不可用原因行：连着事实却没有写通道时写原因，重试入口置灰写原因。
func TestSendUnavailableReasonRow(t *testing.T) {
	src := wiredSource()
	app := NewApp(Options{State: StateDefault, Source: src})
	app.draft = "这条发不出去"
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText(noteSendUnknown) {
		t.Fatalf("没有写发送不可用原因：%v", tt.Texts())
	}
	info, ok := tt.Access("重试发送")
	if !ok {
		t.Fatalf("没有重试发送入口：%v", tt.Texts())
	}
	if !info.Disabled || info.Description != noteNoWriteChannel {
		t.Fatalf("重试发送没有置灰或没写原因：%+v", info)
	}
	// 草稿为空是常态，不写原因。
	app.draft = ""
	tt.Frame()
	if tt.HasText(noteSendUnknown) {
		t.Fatal("草稿为空时仍写了发送原因")
	}
}

// 活动条不把同一事实画两遍（例如会话没有唤醒项时不出现唤醒 chip）。
func TestActivityBarWakeOnlyWithEnabledSchedules(t *testing.T) {
	src := wiredSource()
	src.schedules = SchedulesSnapshot{Items: []ScheduleSnapshot{{ID: "job-1", Enabled: false}}}
	src.session = SessionSnapshot{ID: "s1"}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("唤醒 1"); ok {
		t.Fatalf("停用的唤醒项仍画了 chip：%v", tt.Texts())
	}
	if strings.Contains(strings.Join(tt.Texts(), "\n"), "唤醒") {
		t.Fatalf("没有启用的唤醒项时不该出现唤醒 chip：%v", tt.Texts())
	}
}
