package ui

import (
	"context"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// notifyIntents 是通知屏的写入桩：只收 SetPreferences，其余入口走嵌入的 nil
// 接口（被调用会直接 panic，说明测试点到了没接的动作）。
type notifyIntents struct {
	Intents
	prefs chan PreferencesSnapshot
}

func (n *notifyIntents) SetPreferences(_ context.Context, prefs PreferencesSnapshot) error {
	n.prefs <- prefs
	return nil
}

// waitPrefs 等一次偏好写入，超时即失败。

// TestNotificationsReadPreferences 核对三个开关的初值来自本地偏好，不是写死的
// 样例：关掉「需要我回应时通知」后重进设置，开关仍在关位。
func TestNotificationsReadPreferences(t *testing.T) {
	src := wiredSource()
	src.prefs = PreferencesSnapshot{
		NotifyTaskDone: true, NotifySound: true, NotifyNeedsResponse: false,
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	openSettingsSection(t, app, secNotifications)

	if !app.notifyTaskDone || !app.notifySound || app.notifyNeedsResponse {
		t.Fatalf("开关没有按偏好初始化：done=%v sound=%v needs=%v",
			app.notifyTaskDone, app.notifySound, app.notifyNeedsResponse)
	}
}

// TestNotificationsTogglePersists 核对拨动开关后落盘的是整份偏好（三个开关都在，
// 其他字段不丢），这就是「设置 › 通知的三个开关真正生效并持久化」的界面侧。
func TestNotificationsTogglePersists(t *testing.T) {
	src := wiredSource()
	src.prefs = PreferencesSnapshot{
		Theme: "暗", NotifyTaskDone: true, NotifySound: true, NotifyNeedsResponse: false,
	}
	intents := &notifyIntents{prefs: make(chan PreferencesSnapshot, 1)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := openSettingsSection(t, app, secNotifications)

	if err := tt.Click("任务完成或失败时通知开关"); err != nil {
		t.Fatalf("点不到开关：%v", err)
	}
	got := waitPrefs(t, intents.prefs, "拨动通知开关")
	if got.NotifyTaskDone {
		t.Fatalf("保存的偏好里开关还是开：%+v", got)
	}
	if !got.NotifySound || got.NotifyNeedsResponse {
		t.Fatalf("其他两个开关被改坏了：%+v", got)
	}
	if got.Theme != "暗" {
		t.Fatalf("保存偏好时丢了主题：%+v", got)
	}

	// 再拨一次：整份偏好里声音与回应开关保持不变，完成开关回到开。
	if err := tt.Click("任务完成或失败时通知开关"); err != nil {
		t.Fatalf("点不到开关：%v", err)
	}
	again := waitPrefs(t, intents.prefs, "再次拨动通知开关")
	if !again.NotifyTaskDone || !again.NotifySound || again.NotifyNeedsResponse {
		t.Fatalf("第二次拨动后的偏好不对：%+v", again)
	}
}

// TestNotificationsDemoTogglesLocally 核对样例模式（没有 Intents）只改进程内的
// 界面状态，不发写入。
func TestNotificationsDemoTogglesLocally(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := openSettingsSection(t, app, secNotifications)
	if !app.notifyNeedsResponse {
		t.Fatal("样例模式的出厂状态应为全开")
	}
	if err := tt.Click("需要我回应时通知开关"); err != nil {
		t.Fatalf("点不到开关：%v", err)
	}
	tt.Frame()
	if app.notifyNeedsResponse {
		t.Fatal("样例模式关掉开关没有生效")
	}
}

// TestNotificationsVariantOpensSection：-variant settings-notifications 直接停在
// 通知节（截图与走查用）。
func TestNotificationsVariantOpensSection(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Variant: VariantSettingsNotifications})
	if app.screen != ScreenSettings || app.settingsSection != secNotifications {
		t.Fatalf("变体没有打开通知节：screen=%v section=%v", app.screen, app.settingsSection)
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText(notifyCopyTask) || !tt.HasText(notifyCopySound) || !tt.HasText(notifyCopyNeedsMe) {
		t.Fatalf("通知节没有画出三张卡：%v", tt.Texts())
	}
}

// TestFocusSessionLocatesRow 核对通知点开后的落点：回到主界面、左栏选中该会话，
// 并让下一帧重新打开它（清掉 lastSelectedSession）。
func TestFocusSessionLocatesRow(t *testing.T) {
	src := wiredSource()
	src.names = []string{"sess-a", "sess-b", "sess-c"}
	src.titles = []string{"修发送路径", "补托盘", "排查连接"}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.screen = ScreenSettings
	app.selectedSession = "s1"
	app.lastSelectedSession = "s1"

	app.FocusSession("sess-c")
	if app.screen != ScreenMain {
		t.Fatal("通知点开应回到主界面")
	}
	if app.selectedSession != "s3" || app.lastSelectedSession != "" {
		t.Fatalf("没有定位到 sess-c：selected=%q last=%q", app.selectedSession, app.lastSelectedSession)
	}

	// 不存在的会话不改选中项（不把左栏跳到别处）。
	app.selectedSession = "s2"
	app.FocusSession("sess-missing")
	if app.selectedSession != "s2" {
		t.Fatalf("未知会话不该改选中项：%q", app.selectedSession)
	}
}

// TestHostSnapshotFeedsTrayAndNotify 核对每帧交给宿主外壳的快照：当前会话的
// 运行/待决/最后一轮与最近会话都来自 Source 事实。
func TestHostSnapshotFeedsTrayAndNotify(t *testing.T) {
	src := wiredSource()
	src.names = []string{"sess-a", "sess-b", "sess-c"}
	src.titles = []string{"修发送路径", "补托盘", "排查连接"}
	src.places = []SessionPlaceSnapshot{
		{UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{UpdatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
		{UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
	}
	src.session = SessionSnapshot{
		ID: "sess-b", Title: "补托盘", Running: false, Pending: 1,
		LastTurnEnded: true, LastTurnIndex: 4, LastTurnOutcome: "failed",
	}
	src.hasSess = true
	src.approval = ApprovalSnapshot{ID: "apr-1"}
	src.hasAppr = true

	var snapshots []HostSnapshot
	app := NewApp(Options{State: StateDefault, Source: src,
		OnFrame: func(s HostSnapshot) { snapshots = append(snapshots, s) }})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	if len(snapshots) == 0 {
		t.Fatal("每帧没有把宿主快照交出去")
	}
	got := snapshots[len(snapshots)-1]
	if got.SessionID != "sess-b" || !got.Pending || got.PendingID != "apr-1" {
		t.Fatalf("会话状态不对：%+v", got)
	}
	if !got.LastTurnEnded || got.LastTurnIndex != 4 || got.LastTurnOutcome != "failed" {
		t.Fatalf("最后一轮不对：%+v", got)
	}
	if len(got.Recent) != 3 {
		t.Fatalf("最近会话数量：%d", len(got.Recent))
	}
	if got.Recent[0].ID != "sess-b" || got.Recent[1].ID != "sess-c" || got.Recent[2].ID != "sess-a" {
		t.Fatalf("最近会话应按最近活动排序：%+v", got.Recent)
	}
}
