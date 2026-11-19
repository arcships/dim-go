package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是 Fork 的回归断言：「在新分支继续」/「从此处分叉」走
// Intents.CreateBranch（以轮结束处为切点），成功后切到新分支并写切点；不可用
// 时置灰并把原因写在菜单项、可读名或提示里。

// forkTestIntents 接住分叉与切换分支，并把新分支并进假 Source（模拟适配器在
// 建分支成功后的目录更新）。
type forkTestIntents struct {
	Intents
	src       *fakeSource
	newBranch string
	cuts      []p1BranchCut
	selected  []string
}

func (f *forkTestIntents) CreateBranch(
	_ context.Context, parentID string, turnIndex int, prompt string,
) error {
	f.cuts = append(f.cuts, p1BranchCut{parent: parentID, turn: turnIndex, prompt: prompt})
	if f.src != nil && f.newBranch != "" {
		f.src.session.Branches = append(f.src.session.Branches,
			BranchSnapshot{ID: f.newBranch, Name: f.newBranch, Status: "open"})
	}
	return nil
}

func (f *forkTestIntents) SelectBranch(_ context.Context, branchID string) error {
	f.selected = append(f.selected, branchID)
	if f.src != nil {
		f.src.session.Branch = branchID
	}
	return nil
}

// newForkTestApp 建一个连着事实与写通道的界面，会话是两轮、最后一轮已结束。
func newForkTestApp(t *testing.T) (*App, *gui.Tester, *fakeSource, *forkTestIntents) {
	t.Helper()
	src := wiredSource()
	src.names = []string{"s1"}
	src.titles = []string{"真实会话"}
	snap := exportTestSession()
	src.session = snap
	src.hasSess = true
	intents := &forkTestIntents{src: src, newBranch: "side-1"}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	return app, tt, src, intents
}

// waitFork 反复渲染，直到条件成立；分叉与跳转都在后台意图完成后的下一帧落下。
func waitFork(t *testing.T, tt *gui.Tester, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("等待 %s 超时", what)
}

// TestForkFromUserMessageCreatesBranchAndJumps：悬停用户消息 →「从此处分叉」→
// CreateBranch(父分支=main，切点=第 1 轮) → SelectBranch(新分支) → 切点回执。
func TestForkFromUserMessageCreatesBranchAndJumps(t *testing.T) {
	app, tt, _, intents := newForkTestApp(t)
	bubble, ok := tt.Find("用户消息 第一条消息")
	if !ok {
		t.Fatalf("找不到第一条用户消息：%v", tt.Texts())
	}
	tt.Move(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if err := tt.Click("从此处分叉 1"); err != nil {
		t.Fatalf("点不到从此处分叉：%v", err)
	}
	waitFork(t, tt, "分叉与跳转", func() bool {
		return len(intents.cuts) == 1 && len(intents.selected) == 1
	})
	cut := intents.cuts[0]
	if cut.parent != "main" || cut.turn != 1 {
		t.Fatalf("CreateBranch 收到 %+v，想要父分支 main / 切点 1", cut)
	}
	if strings.TrimSpace(cut.prompt) == "" {
		t.Fatal("CreateBranch 的提示词为空（协议要求非空）")
	}
	if intents.selected[0] != "side-1" {
		t.Fatalf("没有切到新分支：%v", intents.selected)
	}
	if !strings.Contains(app.headerNote, "已切到分支 side-1") ||
		!strings.Contains(app.headerNote, "切点：第 1 轮结束处") {
		t.Fatalf("回执没有写分支与切点：%q", app.headerNote)
	}
}

// TestForkLatestFromHeaderMenu：「在新分支继续」以最后一轮结束处为切点。
func TestForkLatestFromHeaderMenu(t *testing.T) {
	app, tt, _, intents := newForkTestApp(t)
	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯ 菜单：%v", err)
	}
	tt.Frame()
	if err := tt.Click("在新分支继续"); err != nil {
		t.Fatalf("点不到在新分支继续：%v", err)
	}
	waitFork(t, tt, "在新分支继续", func() bool {
		return len(intents.cuts) == 1 && len(intents.selected) == 1
	})
	if intents.cuts[0].turn != 2 || intents.cuts[0].parent != "main" {
		t.Fatalf("切点不是最后一轮结束处：%+v", intents.cuts[0])
	}
	if !strings.Contains(app.headerNote, "切点：第 2 轮结束处") {
		t.Fatalf("回执没有写切点：%q", app.headerNote)
	}
}

// TestForkDisabledWithoutTurnFact：快照没给轮次时置灰并写原因，不拿别的轮次顶替。
func TestForkDisabledWithoutTurnFact(t *testing.T) {
	src := wiredSource()
	src.names = []string{"s1"}
	src.session = SessionSnapshot{
		ID: "s1", Title: "真实会话", Branch: "main", LastTurnEnded: true, LastTurnIndex: 1,
		Rows: []RowSnapshot{{Kind: RowUser, Text: "没有轮次的消息", Status: RowStatusDone}},
	}
	src.hasSess = true
	app := NewApp(Options{State: StateDefault, Source: src, Intents: &forkTestIntents{}})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	bubble, ok := tt.Find("用户消息 没有轮次的消息")
	if !ok {
		t.Fatalf("找不到用户消息：%v", tt.Texts())
	}
	tt.Move(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if info, ok := tt.Access("从此处分叉 1（" + forkNoteNoTurn + "）"); !ok || !info.Disabled {
		t.Fatalf("没有轮次事实时分叉没有置灰：%+v（找到 %v）", info, ok)
	}
	if !tt.HasText("分叉不可用：" + forkNoteNoTurn) {
		t.Fatalf("没有写不可用原因：%v", tt.Texts())
	}
}

// TestForkDisabledWithoutWriteChannel：连着事实但没有写通道时置灰并写原因。
func TestForkDisabledWithoutWriteChannel(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	src := wiredSource()
	src.names = []string{"s1"}
	src.session = exportTestSession()
	src.hasSess = true
	app.source = src
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	bubble, ok := tt.Find("用户消息 第一条消息")
	if !ok {
		t.Fatalf("找不到用户消息：%v", tt.Texts())
	}
	tt.Move(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if info, ok := tt.Access("从此处分叉 1（" + noteNoWriteChannel + "）"); !ok || !info.Disabled {
		t.Fatalf("没有写通道时分叉没有置灰：%+v（找到 %v）", info, ok)
	}
}

// TestForkLatestDisabledWhileRunning：最后一轮没结束时「在新分支继续」置灰，
// 原因写在菜单项里。
func TestForkLatestDisabledWhileRunning(t *testing.T) {
	src := wiredSource()
	src.names = []string{"s1"}
	snap := exportTestSession()
	snap.Running = true
	snap.LastTurnEnded = false
	src.session = snap
	src.hasSess = true
	app := NewApp(Options{State: StateDefault, Source: src, Intents: &forkTestIntents{}})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯ 菜单：%v", err)
	}
	tt.Frame()
	want := "在新分支继续（" + forkNoteTurnRunning + "）"
	found := false
	for _, item := range tt.Menu() {
		if item == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("菜单里没有 %q：%v", want, tt.Menu())
	}
	if err := tt.ChooseMenuItem(want); err == nil {
		t.Fatalf("置灰的 %q 仍被选中", want)
	}
	_ = app
}

// TestForkSampleSwitchesBranch：样例模式不发写入，只改视图里的分支名并写回执。
func TestForkSampleSwitchesBranch(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	bubble, ok := tt.Find("我的消息")
	if !ok {
		t.Fatalf("找不到用户气泡：%v", tt.Texts())
	}
	tt.Move(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if info, ok := tt.Access("从此处分叉 1"); !ok || info.Disabled {
		t.Fatalf("样例里分叉应当可点：%+v（找到 %v）", info, ok)
	}
	if err := tt.Click("从此处分叉 1"); err != nil {
		t.Fatalf("点不到从此处分叉：%v", err)
	}
	if app.branch != "main-1" {
		t.Fatalf("样例分支 = %q，想要 main-1", app.branch)
	}
	if !strings.Contains(app.headerNote, "切点：第 1 轮结束处") {
		t.Fatalf("样例回执没有切点：%q", app.headerNote)
	}
}

// TestForkFromMessageMenu：右键菜单里的「从此处分叉」与悬停动作同一落点。
func TestForkFromMessageMenu(t *testing.T) {
	_, tt, _, intents := newForkTestApp(t)
	if _, ok := tt.Find("用户消息 第二条消息"); !ok {
		t.Fatalf("找不到第二条用户消息：%v", tt.Texts())
	}
	tt.RightClick("第二条消息")
	tt.Frame()
	if err := tt.ChooseMenuItem("从此处分叉"); err != nil {
		t.Fatalf("点不到菜单里的从此处分叉：%v", err)
	}
	waitFork(t, tt, "菜单分叉", func() bool { return len(intents.cuts) == 1 })
	if intents.cuts[0].turn != 2 {
		t.Fatalf("菜单分叉的切点 = %d，想要 2", intents.cuts[0].turn)
	}
}
