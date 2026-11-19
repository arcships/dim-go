package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是四类确认框（全局 E1–E4）的可执行验收：标题、说明（含不可撤销的那半句）、
// 取消/确认、危险色、默认焦点、Esc 与执行回执/错误行。

// stubConfirmIntents 接住删除、归档与停止，其余方法走嵌入的 nil 接口（被调用即
// panic，说明测试点到了一个没接的动作）。
type stubConfirmIntents struct {
	stubBaseIntents
	deleted  chan string
	archived chan string
	stopped  chan struct{}
	failNext bool
}

func (s *stubConfirmIntents) DeleteSession(_ context.Context, sessionID string) error {
	if s.failNext {
		s.failNext = false
		return errors.New("会话已被别处删除")
	}
	s.deleted <- sessionID
	return nil
}

func (s *stubConfirmIntents) ArchiveSession(_ context.Context, sessionID string, _ bool) error {
	s.archived <- sessionID
	return nil
}

func (s *stubConfirmIntents) Stop(context.Context) error {
	s.stopped <- struct{}{}
	return nil
}

// newConfirmIntents 造一个接住这三类写入的替身。
func newConfirmIntents() *stubConfirmIntents {
	return &stubConfirmIntents{
		deleted:  make(chan string, 4),
		archived: make(chan string, 4),
		stopped:  make(chan struct{}, 1),
	}
}

// confirmText 取对话框里的整屏文字（失败时用来定位）。
func confirmText(tt *gui.Tester) string { return p0WindowText(tt) }

// TestConfirmDeleteSessionDialog：删除会话的确认框——标题、不可撤销说明、取消与
// 危险色删除、默认焦点在取消、Esc 不删、执行后有回执。
func TestConfirmDeleteSessionDialog(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.openDeleteConfirm([]int{0})
	tt.Frame()

	for _, want := range []string{
		confirmCopyDeleteOne,
		"确定要永久删除“修发送路径”吗？删除后会话将立即从侧边栏移除，且无法恢复。",
		confirmCopyCancel, confirmCopyDelete,
	} {
		if !tt.HasText(want) {
			t.Fatalf("删除确认框缺少 %q：\n%s", want, confirmText(tt))
		}
	}
	// 只框内两枚按钮：标题 + 描述 + 取消 + 删除，没有第三个决定。
	if info, ok := tt.Access(confirmCopyCancel); !ok || !info.Focused {
		t.Fatalf("危险动作的默认焦点不在「取消」：%+v（ok=%v）", info, ok)
	}
	if app.archived[0] {
		t.Fatal("确认框刚打开就删掉了会话")
	}

	// Esc 关闭：不删除任何会话。
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.confirm.kind != confirmKindNone {
		t.Fatalf("Esc 没有关闭确认框：%s", app.confirm.kind)
	}
	if app.archived[0] {
		t.Fatal("Esc 关闭后会话被删掉了")
	}

	// 再打开并确认：删除生效、弹窗关闭、回执写出。
	app.openDeleteConfirm([]int{0})
	tt.Frame()
	if err := tt.Click(confirmCopyDelete); err != nil {
		t.Fatalf("点不到删除：%v", err)
	}
	tt.Frame()
	if !app.archived[0] {
		t.Fatal("确认后会话没有删除")
	}
	if app.confirm.kind != confirmKindNone {
		t.Fatal("删除后弹窗没有关闭")
	}
	if !strings.Contains(confirmText(tt), "已删除 1 个会话") {
		t.Fatalf("没有写回执：\n%s", confirmText(tt))
	}
}

// TestConfirmDeleteFailureKeepsDialog：删除失败留在弹窗里给原因，可再点一次重试。
func TestConfirmDeleteFailureKeepsDialog(t *testing.T) {
	src := wiredSource()
	src.names = []string{"s1"}
	src.titles = []string{"真实会话"}
	stub := newConfirmIntents()
	stub.failNext = true
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	app.openDeleteConfirm([]int{0})
	tt.Frame()
	if err := tt.Click(confirmCopyDelete); err != nil {
		t.Fatalf("点不到删除：%v", err)
	}
	tt.Frame()
	if !tt.HasText("会话已被别处删除") {
		t.Fatalf("失败原因没有写进弹窗：\n%s", confirmText(tt))
	}
	if app.confirm.kind != confirmKindDelete || app.confirm.busy {
		t.Fatalf("失败后弹窗应保留且按钮恢复：kind=%q busy=%v", app.confirm.kind, app.confirm.busy)
	}
	if err := tt.Click(confirmCopyDelete); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	select {
	case <-stub.deleted:
	case <-time.After(2 * time.Second):
		t.Fatal("重试没有发出删除")
	}
}

// TestConfirmArchiveSessionsDialog：批量归档用主色（可恢复），运行中的会话带警告；
// 单条归档不弹框。
func TestConfirmArchiveSessionsDialog(t *testing.T) {
	app, tt := newTester(t, StateRunning)
	if !app.anyRunning([]int{0}) {
		t.Fatal("样例运行态的第一行没有被判成运行中")
	}
	app.openArchiveConfirm([]int{0})
	tt.Frame()

	for _, want := range []string{
		"归档 1 个会话", "确定要归档以下会话吗？", confirmCopyArchiveRun, confirmCopyCancel,
	} {
		if !tt.HasText(want) {
			t.Fatalf("归档确认框缺少 %q：\n%s", want, confirmText(tt))
		}
	}
	if !tt.HasText("归档 (1)") {
		t.Fatalf("归档按钮没有带数量：\n%s", confirmText(tt))
	}
	// 归档可恢复：默认焦点在主按钮上，回车就是确认。
	if info, ok := tt.Access("归档 (1)"); !ok || !info.Focused {
		t.Fatalf("归档的默认焦点不在主按钮：%+v（ok=%v）", info, ok)
	}
	if err := tt.Click("归档 (1)"); err != nil {
		t.Fatalf("点不到归档：%v", err)
	}
	tt.Frame()
	if !app.archived[0] {
		t.Fatal("确认后会话没有归档")
	}
	if app.confirm.kind != confirmKindNone {
		t.Fatal("归档后弹窗没有关闭")
	}

	// 单条归档没有确认框：菜单里一步完成（AC-22）。
	app2, tt2 := newTester(t, StateDefault)
	app2.archiveSession(0)
	tt2.Frame()
	if app2.confirm.kind != confirmKindNone || !app2.archived[0] {
		t.Fatalf("单条归档不该弹框：kind=%q archived=%v", app2.confirm.kind, app2.archived[0])
	}
}

// TestConfirmDeleteWorkspaceDialog：工作区菜单的「移除工作区」先弹确认框——标题、
// 带计数与名称的说明、路径行、危险色删除；确认后会话删除、组一并去掉。
func TestConfirmDeleteWorkspaceDialog(t *testing.T) {
	app := cwdGroupApp([]SessionPlaceSnapshot{
		{Cwd: "/work/dim-agent"},
		{Cwd: "/work/rutis"},
	})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	groups := app.sidebarWorkspaces(app.sidebarThreads())
	var target sidebarWorkspace
	for _, group := range groups {
		if group.name == "rutis" {
			target = group
		}
	}
	if target.id == "" {
		t.Fatalf("找不到 rutis 组：%+v", groups)
	}
	row, ok := tt.Find("rutis")
	if !ok {
		t.Fatalf("找不到 rutis 组头：%v", tt.Texts())
	}
	tt.Move(row.X+row.W-20, row.Y+row.H/2)
	if err := tt.Click("工作区菜单：rutis"); err != nil {
		t.Fatalf("点不到工作区菜单：%v", err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem("移除工作区"); err != nil {
		t.Fatalf("点不到移除工作区：%v", err)
	}
	tt.Frame()

	for _, want := range []string{
		confirmCopyWorkspace, "将删除“rutis”下的 1 条会话。", "/work/rutis", confirmCopyCancel, confirmCopyDelete,
	} {
		if !tt.HasText(want) {
			t.Fatalf("工作区确认框缺少 %q：\n%s", want, confirmText(tt))
		}
	}
	deletedBefore := len(app.archived)
	if err := tt.Click(confirmCopyDelete); err != nil {
		t.Fatalf("点不到删除：%v", err)
	}
	tt.Frame()
	if len(app.archived) <= deletedBefore {
		t.Fatal("确认后工作区下的会话没有删除")
	}
	if app.confirm.kind != confirmKindNone {
		t.Fatal("删除后弹窗没有关闭")
	}
	for _, group := range app.sidebarWorkspaces(app.sidebarThreads()) {
		if group.id == target.id {
			t.Fatal("删除工作区会话后空组仍留在侧栏")
		}
	}
}

// TestConfirmEditRunningDialog：会话运行中保存编辑先弹「会话正在运行」——标题、说明、
// 取消与「停止并发送」；取消不丢草稿，确认先停止再落定编辑。
func TestConfirmEditRunningDialog(t *testing.T) {
	src := wiredSource()
	src.session.Running = true
	src.session.Rows = []RowSnapshot{{Kind: "user", Text: "原文"}}
	stub := newConfirmIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	// 先画一帧：连上事实后的第一帧会铺回视图状态（行内编辑器也会被清掉）。
	tt.Frame()

	app.beginInlineEdit(0, "原文")
	app.inlineDraft = "改过的正文"
	app.saveInlineEdit()
	tt.Frame()

	for _, want := range []string{
		confirmCopyEditTitle,
		"当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？",
		confirmCopyCancel, confirmCopyStopSend,
	} {
		if !tt.HasText(want) {
			t.Fatalf("运行中确认框缺少 %q：\n%s", want, confirmText(tt))
		}
	}
	if info, ok := tt.Access(confirmCopyCancel); !ok || !info.Focused {
		t.Fatalf("默认焦点不在「取消」：%+v（ok=%v）", info, ok)
	}

	// 取消（Esc）：弹窗关闭，编辑内容与编辑器都还在（不丢用户输入）。框后面的
	// 行内编辑器也认 Esc，模态框开着时 Esc 只关框，先关编辑器是错的。
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.confirm.kind != confirmKindNone {
		t.Fatal("取消后弹窗没有关闭")
	}
	if app.inlineEditRow != 0 || app.inlineDraft != "改过的正文" {
		t.Fatalf("取消丢掉了编辑：row=%d draft=%q", app.inlineEditRow, app.inlineDraft)
	}

	// 确认：先停止当前回合并落定编辑；消息写入没有通道时把原因写在编辑器下方。
	app.saveInlineEdit()
	tt.Frame()
	if err := tt.Click(confirmCopyStopSend); err != nil {
		t.Fatalf("点不到停止并发送：%v", err)
	}
	select {
	case <-stub.stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("确认后没有停止当前回合")
	}
	tt.Frame()
	if app.confirm.kind != confirmKindNone {
		t.Fatal("执行后弹窗没有关闭")
	}
	if app.inlineNotice != noteInlineEdit {
		t.Fatalf("消息写入没有通道时的原因 = %q，想要 %q", app.inlineNotice, noteInlineEdit)
	}
}
