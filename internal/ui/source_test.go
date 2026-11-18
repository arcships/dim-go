package ui

import (
	"context"
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// fakeSource 是一个可注入的 Source：用来证明"视图只吃 Source、不吃样例"。
type fakeSource struct {
	names      []string
	workspaces []WorkspaceSnapshot
	session    SessionSnapshot
	hasSess    bool
	approval   ApprovalSnapshot
	hasAppr    bool
	conn       Connection
	acct       AccountSnapshot
	usage      UsageSnapshot
	settings   SettingsSnapshot
	extensions ExtensionsSnapshot
	schedules  SchedulesSnapshot
	terminal   TerminalSnapshot
	prefs      PreferencesSnapshot
}

func (f *fakeSource) Connection() Connection           { return f.conn }
func (f *fakeSource) SessionNames() []string           { return f.names }
func (f *fakeSource) Workspaces() []WorkspaceSnapshot  { return f.workspaces }
func (f *fakeSource) Session() (SessionSnapshot, bool) { return f.session, f.hasSess }
func (f *fakeSource) Approval() (ApprovalSnapshot, bool) {
	return f.approval, f.hasAppr
}
func (f *fakeSource) Question() (QuestionSnapshot, bool) { return QuestionSnapshot{}, false }
func (f *fakeSource) Account() AccountSnapshot           { return f.acct }
func (f *fakeSource) Usage() UsageSnapshot               { return f.usage }
func (f *fakeSource) Settings() SettingsSnapshot         { return f.settings }
func (f *fakeSource) Extensions() ExtensionsSnapshot     { return f.extensions }
func (f *fakeSource) Schedules() SchedulesSnapshot       { return f.schedules }
func (f *fakeSource) Terminal() TerminalSnapshot         { return f.terminal }
func (f *fakeSource) Preferences() PreferencesSnapshot   { return f.prefs }

// TestSourceDrivesSidebarAndTimeline：把 Source 的会话列表改成空 → 界面进空态，
// 而不是保留样例；把 Running 翻成 true → 停止槽出现。
func TestSourceDrivesSidebarAndTimeline(t *testing.T) {
	src := &fakeSource{
		conn: Connection{Phase: "connected"},
		acct: AccountSnapshot{Initial: "E", Connected: true},
	}
	app := NewApp(Options{State: StateDefault})
	app.source = src
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	// 空列表：左栏没有样例会话，时间线进空态。
	for _, name := range []string{"修发送路径", "参考", "旧实现对照"} {
		if _, ok := tt.Find(name); ok {
			t.Fatalf("Source 为空时仍显示样例会话 %q", name)
		}
	}
	if _, ok := tt.Find("当前会话还没有消息"); !ok {
		t.Fatal("空列表没有进空态")
	}

	// 打开一个会话：行与运行态都来自 Source。
	src.names = []string{"接入真实事实"}
	src.session = SessionSnapshot{ID: "s1", Title: "接入真实事实", Running: true,
		Rows: []RowSnapshot{{Kind: "user", Text: "跑一下测试"}}}
	src.hasSess = true
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("接入真实事实"); !ok {
		t.Fatal("会话名没有来自 Source")
	}
	if _, ok := tt.Find("跑一下测试"); !ok {
		t.Fatal("时间线没有来自 Source 的行")
	}
	if _, ok := tt.Access("停止"); !ok {
		t.Fatal("Running=true 时没有出现停止槽")
	}
	if _, ok := tt.Access("发送"); ok {
		t.Fatal("Running=true 时不该还有发送槽")
	}
}

// TestSourceDisconnect：断线时状态点转 danger、输入位禁用并给出原因。
func TestSourceDisconnect(t *testing.T) {
	src := &fakeSource{
		conn:    Connection{Phase: "disconnected", Reason: "socket closed"},
		names:   []string{"x"},
		session: SessionSnapshot{ID: "s1", Title: "x"},
		hasSess: true,
	}
	app := NewApp(Options{State: StateDefault})
	app.source = src
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Access("连接断开"); !ok {
		t.Fatal("断线没有在原位给原因")
	}
	if _, ok := tt.Access("发送"); ok {
		t.Fatal("断线时不该还有发送入口")
	}
	// 状态点：断线时用 danger。
	if got := connectionColor(theme(false, gui.Preferences{}), src.conn); got != theme(false, gui.Preferences{}).Danger {
		t.Fatalf("状态点颜色 = %v，想要 danger", got)
	}
}

// TestSourceApprovalReplacesComposer：待决审批来自 Source，占住输入位。
func TestSourceApprovalReplacesComposer(t *testing.T) {
	src := &fakeSource{
		conn:     Connection{Phase: "connected"},
		names:    []string{"x"},
		session:  SessionSnapshot{ID: "s1", Title: "x"},
		hasSess:  true,
		approval: ApprovalSnapshot{ID: "a1", Command: "rm -rf ./tmp/*", Impact: "工作区内的临时文件"},
		hasAppr:  true,
	}
	app := NewApp(Options{State: StateDefault})
	app.source = src
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("rm -rf ./tmp/*"); !ok {
		t.Fatal("审批命令没有来自 Source")
	}
	if _, ok := tt.Access("仅这次"); !ok {
		t.Fatal("审批决定缺失")
	}
}

// TestViewStateSurvivesFactCount：视图状态（展开态）按轮次索引存，事实变多
// 不需要重建视图状态，也不 panic。
func TestViewStateSurvivesFactCount(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.source = &fakeSource{conn: Connection{Phase: "connected"}, names: []string{"x"}}
	// 200 轮：展开态按需创建，不随事实数量重建。
	app.rounds = benchRounds(200)
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if len(tt.Texts()) == 0 {
		t.Fatal("200 轮没有画出来")
	}
	if got := len(app.summaryOpen); got > 4 {
		t.Fatalf("视图状态建了 %d 条，只该建被触发的那几条", got)
	}
}

// unusedCtx 让接口签名里的 context 在本文件里有着落。
func unusedCtx() context.Context { return context.Background() }

var _ = strings.TrimSpace
var _ = unusedCtx
