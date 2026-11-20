package ui

import (
	"context"
	"errors"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// archiveIntents 只接归档与删除：记下每次写入口的动作。
type archiveIntents struct {
	Intents
	archived chan [2]any
	deleted  chan string
	fail     error
}

func newArchiveIntents() *archiveIntents {
	return &archiveIntents{archived: make(chan [2]any, 4), deleted: make(chan string, 4)}
}

func (a *archiveIntents) ArchiveSession(_ context.Context, sessionID string, archived bool) error {
	select {
	case a.archived <- [2]any{sessionID, archived}:
	default:
	}
	return a.fail
}

func (a *archiveIntents) DeleteSession(_ context.Context, sessionID string) error {
	select {
	case a.deleted <- sessionID:
	default:
	}
	return a.fail
}

// openArchived 打开设置 › 归档会话并渲染一帧。
func openArchived(t *testing.T, src *fakeSource, intents Intents) (*App, *gui.Tester) {
	t.Helper()
	resetSettingsScreenState()
	archivedScreen = archivedState{projectFilter: arcCopyAllProjects}
	archivedView = struct {
		threads []archivedThread
		wired   bool
	}{}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	return app, openSettingsSection(t, app, secArchived)
}

// TestArchivedListComesFromFacts：归档行读事实——标题、工作目录分组、归档日期都
// 来自 Source，屏上不出现样例行；没有归档时是空态。
func TestArchivedListComesFromFacts(t *testing.T) {
	src := wiredSource()
	src.archived = []ArchivedSessionSnapshot{
		{ID: "arc-1", Title: "真实归档会话", Workspace: "/w/alpha",
			ArchivedAt: time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)},
		{ID: "arc-2", Preview: "只有预览的会话", Workspace: "/w/beta",
			ArchivedAt: time.Date(2026, 3, 5, 9, 0, 0, 0, time.UTC)},
	}
	_, tt := openArchived(t, src, newArchiveIntents())

	if !tt.HasText("真实归档会话") {
		t.Fatalf("没有画出事实里的归档行：%v", tt.Texts())
	}
	if !tt.HasText("只有预览的会话") {
		t.Fatalf("标题为空时没有用首条预览当显示名：%v", tt.Texts())
	}
	// 项目分组用工作目录；归档日期按月日显示。
	if !tt.HasText("/w/alpha") || !tt.HasText(arcArchivedLabel("3 月 4 日")) {
		t.Fatalf("分组或归档日期不对：%v", tt.Texts())
	}
}

// TestArchivedEmptyState：没有归档行时是空态，「全部删除」置灰。
func TestArchivedEmptyState(t *testing.T) {
	_, tt := openArchived(t, wiredSource(), newArchiveIntents())
	if !tt.HasText(arcCopyEmpty) {
		t.Fatalf("没有归档时应是空态：%v", tt.Texts())
	}
	info, ok := tt.Access(arcCopyDeleteAll)
	if !ok {
		t.Fatal("「全部删除」入口要保留")
	}
	if !info.Disabled {
		t.Fatalf("没有归档行时「全部删除」应置灰：%+v", info)
	}
}

// TestArchivedRestoreCallsArchiveSession：恢复写 ArchiveSession(id, false)。
func TestArchivedRestoreCallsArchiveSession(t *testing.T) {
	src := wiredSource()
	src.archived = []ArchivedSessionSnapshot{{ID: "arc-1", Title: "要恢复的会话"}}
	intents := newArchiveIntents()
	_, tt := openArchived(t, src, intents)

	if err := tt.Click(arcCopyRestore); err != nil {
		t.Fatalf("点不到「恢复」：%v", err)
	}
	select {
	case got := <-intents.archived:
		if got[0] != "arc-1" || got[1] != false {
			t.Fatalf("写入了 %v，想要 [arc-1 false]", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有等到 ArchiveSession")
	}
}

// TestArchivedDeleteCallsDeleteSession：删除归档写 DeleteSession，按钮在对话框里。
func TestArchivedDeleteCallsDeleteSession(t *testing.T) {
	src := wiredSource()
	src.archived = []ArchivedSessionSnapshot{{ID: "arc-1", Title: "要删除的会话"}}
	intents := newArchiveIntents()
	_, tt := openArchived(t, src, intents)

	if err := tt.Click(arcCopyDeleteThread); err != nil {
		t.Fatalf("点不到删除按钮：%v", err)
	}
	tt.Frame()
	if !tt.HasText(arcDeleteOneDesc("要删除的会话")) {
		t.Fatalf("确认框没有写清要删哪一条：%v", tt.Texts())
	}
	if err := tt.Click(arcCopyDelete); err != nil {
		t.Fatalf("点不到确认框里的「删除」：%v", err)
	}
	select {
	case got := <-intents.deleted:
		if got != "arc-1" {
			t.Fatalf("删除了 %q，想要 arc-1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有等到 DeleteSession")
	}
}

// TestArchivedDeleteAllDeletesEveryRow：「全部删除」对每一行发一次 DeleteSession。
func TestArchivedDeleteAllDeletesEveryRow(t *testing.T) {
	src := wiredSource()
	src.archived = []ArchivedSessionSnapshot{
		{ID: "arc-1", Title: "第一条"},
		{ID: "arc-2", Title: "第二条"},
	}
	intents := newArchiveIntents()
	_, tt := openArchived(t, src, intents)

	if err := tt.Click(arcCopyDeleteAll); err != nil {
		t.Fatalf("点不到「全部删除」：%v", err)
	}
	tt.Frame()
	if !tt.HasText(arcDeleteAllDesc(2)) {
		t.Fatalf("确认框的条数不对：%v", tt.Texts())
	}
	if err := tt.Click(arcCopyDelete); err != nil {
		t.Fatalf("点不到确认框里的「删除」：%v", err)
	}
	got := map[string]bool{}
	deadline := time.After(2 * time.Second)
	for len(got) < 2 {
		select {
		case id := <-intents.deleted:
			got[id] = true
		case <-deadline:
			t.Fatalf("只删了 %v，想要两条都删", got)
		}
	}
	if !got["arc-1"] || !got["arc-2"] {
		t.Fatalf("删的不是两条：%v", got)
	}
}

// TestArchivedFailureShowsReason：删除失败把原因写在页首错误行，行不消失（事实没变）。
func TestArchivedFailureShowsReason(t *testing.T) {
	src := wiredSource()
	src.archived = []ArchivedSessionSnapshot{{ID: "arc-1", Title: "删不掉的会话"}}
	intents := newArchiveIntents()
	intents.fail = errors.New("会话正在运行，无法删除")
	_, tt := openArchived(t, src, intents)

	if err := tt.Click(arcCopyDeleteThread); err != nil {
		t.Fatalf("点不到删除按钮：%v", err)
	}
	tt.Frame()
	if err := tt.Click(arcCopyDelete); err != nil {
		t.Fatalf("点不到确认框里的「删除」：%v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if archivedScreen.opErr != "" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if archivedScreen.opErr == "" {
		t.Fatal("删除失败后没有写原因")
	}
	if !tt.HasText("删不掉的会话") {
		t.Fatalf("失败后行不应消失：%v", tt.Texts())
	}
	if !tt.HasText(archivedScreen.opErr) {
		t.Fatalf("失败原因没有上屏：%v", tt.Texts())
	}
}

// TestArchivedSampleStillRemovesLocally：没有 Source（-demo）时恢复只改屏内表。
func TestArchivedSampleStillRemovesLocally(t *testing.T) {
	resetSettingsScreenState()
	archivedScreen = archivedState{
		projectFilter: arcCopyAllProjects,
		threads:       []archivedThread{{id: "s-1", title: "样例归档会话"}},
	}
	archivedView = struct {
		threads []archivedThread
		wired   bool
	}{}
	app := NewApp(Options{State: StateDefault})
	tt := openSettingsSection(t, app, secArchived)

	if err := tt.Click(arcCopyRestore); err != nil {
		t.Fatalf("点不到「恢复」：%v", err)
	}
	tt.Frame()
	if len(archivedScreen.threads) != 0 {
		t.Fatalf("样例模式下恢复没有移除行：%+v", archivedScreen.threads)
	}
	if !tt.HasText(arcCopyEmpty) {
		t.Fatalf("移走后应是空态：%v", tt.Texts())
	}
}
