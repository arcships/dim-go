package notify

import (
	"errors"
	"strings"
	"testing"
)

// allPrefs 是三个开关全开。
var allPrefs = Prefs{TaskDone: true, Sound: true, NeedsResponse: true}

// TestWatcherRunCompleted 核对运行中 → 停止发一条「完成」通知，且重复观察不重发。
func TestWatcherRunCompleted(t *testing.T) {
	w := NewWatcher()
	session := Session{ID: "sess-1", Title: "修发送路径", Running: true}
	if events := w.Observe(session, allPrefs); len(events) != 0 {
		t.Fatalf("第一次观察只建基线，不该发通知：%+v", events)
	}
	session.Running = false
	session.LastTurnEnded = true
	session.LastTurnIndex = 3
	session.LastTurnOutcome = "completed"
	events := w.Observe(session, allPrefs)
	if len(events) != 1 || events[0].Kind != KindRunCompleted {
		t.Fatalf("轮完成应发一条完成通知：%+v", events)
	}
	if events[0].ID != "run:sess-1:3" || events[0].Silent {
		t.Fatalf("通知标识或声音不对：%+v", events[0])
	}
	if again := w.Observe(session, allPrefs); len(again) != 0 {
		t.Fatalf("同一轮不该重复通知：%+v", again)
	}
}

// TestWatcherRunFailed 核对失败/取消的结果词发「失败」通知。
func TestWatcherRunFailed(t *testing.T) {
	for _, outcome := range []string{"failed", "error", "cancelled"} {
		w := NewWatcher()
		w.Observe(Session{ID: "sess-2", Running: true}, allPrefs)
		events := w.Observe(Session{
			ID: "sess-2", Running: false,
			LastTurnEnded: true, LastTurnIndex: 1, LastTurnOutcome: outcome,
		}, allPrefs)
		if len(events) != 1 || events[0].Kind != KindRunFailed {
			t.Fatalf("结果词 %q 应发失败通知：%+v", outcome, events)
		}
		if body(events[0].Kind) != "当前任务失败" {
			t.Fatalf("失败通知正文：%q", body(events[0].Kind))
		}
	}
}

// TestWatcherPrefsGate 核对三个开关真的决定发不发：关掉完成任务就不发，声音关掉
// 时通知标为静音。
func TestWatcherPrefsGate(t *testing.T) {
	w := NewWatcher()
	closed := Prefs{TaskDone: false, Sound: true, NeedsResponse: false}
	w.Observe(Session{ID: "sess-3", Running: true}, closed)
	done := Session{ID: "sess-3", Running: false, LastTurnEnded: true, LastTurnIndex: 1, LastTurnOutcome: "completed"}
	if events := w.Observe(done, closed); len(events) != 0 {
		t.Fatalf("关掉「任务完成时通知」后不该发：%+v", events)
	}

	silent := Prefs{TaskDone: true, Sound: false, NeedsResponse: true}
	w2 := NewWatcher()
	w2.Observe(Session{ID: "sess-4", Running: true}, silent)
	events := w2.Observe(Session{
		ID: "sess-4", Running: false, LastTurnEnded: true, LastTurnIndex: 2, LastTurnOutcome: "completed",
	}, silent)
	if len(events) != 1 || !events[0].Silent {
		t.Fatalf("声音关掉时应发静音通知：%+v", events)
	}
}

// TestWatcherActionRequired 核对待决出现时发一条「需要回应」，待决解除后同一个
// 标识不再重发，换成新待决才再发。
func TestWatcherActionRequired(t *testing.T) {
	w := NewWatcher()
	w.Observe(Session{ID: "sess-5"}, allPrefs)
	events := w.Observe(Session{ID: "sess-5", Pending: true, PendingID: "apr-1"}, allPrefs)
	if len(events) != 1 || events[0].Kind != KindActionRequired {
		t.Fatalf("审批待决应发一条通知：%+v", events)
	}
	if events[0].ID != "interaction:sess-5:apr-1" {
		t.Fatalf("待决通知标识：%q", events[0].ID)
	}
	if again := w.Observe(Session{ID: "sess-5", Pending: true, PendingID: "apr-1"}, allPrefs); len(again) != 0 {
		t.Fatalf("同一个待决不该重复通知：%+v", again)
	}
	// 解除后再来一个新的待决：换标识，再发一次。
	w.Observe(Session{ID: "sess-5"}, allPrefs)
	next := w.Observe(Session{ID: "sess-5", Pending: true, PendingID: "apr-2"}, allPrefs)
	if len(next) != 1 || next[0].ID != "interaction:sess-5:apr-2" {
		t.Fatalf("新待决应再发一条：%+v", next)
	}

	// 开关关掉时不发。
	off := NewWatcher()
	off.Observe(Session{ID: "sess-6"}, Prefs{})
	if events := off.Observe(Session{ID: "sess-6", Pending: true, PendingID: "apr-3"}, Prefs{}); len(events) != 0 {
		t.Fatalf("关掉「需要我回应时通知」后不该发：%+v", events)
	}
}

// stubPresenter 是注入桩：记录收到的通知，可以让测试模拟用户点开。
type stubPresenter struct {
	notes []Note
	err   error
}

func (s *stubPresenter) Show(note Note) error {
	if s.err != nil {
		return s.err
	}
	s.notes = append(s.notes, note)
	return nil
}

// TestNotifierClickFocusesSession 核对通知点击回焦：点开通知后把窗口带到前台并
// 定位到该会话（focus 注入桩收到的就是通知里的会话标识）。
func TestNotifierClickFocusesSession(t *testing.T) {
	presenter := &stubPresenter{}
	var focused []string
	notifier := NewNotifier(presenter, func(sessionID string) { focused = append(focused, sessionID) }, nil)
	event := Event{Kind: KindActionRequired, SessionID: "sess-9", SessionTitle: "补托盘", ID: "interaction:sess-9:apr-9"}
	if err := notifier.Notify(event); err != nil {
		t.Fatalf("发通知：%v", err)
	}
	if len(presenter.notes) != 1 {
		t.Fatalf("应发一条通知：%+v", presenter.notes)
	}
	note := presenter.notes[0]
	if note.Title != "DimAgent" || note.Subtitle != "补托盘" || note.Body != "任务正在等待你的回答、选择或批准" {
		t.Fatalf("通知文案不对：%+v", note)
	}
	if note.Group != "session:sess-9:interactions" {
		t.Fatalf("通知分组不对：%q", note.Group)
	}
	if focused != nil {
		t.Fatalf("还没点开就回焦了：%v", focused)
	}
	note.OnClick()
	if strings.Join(focused, ",") != "sess-9" {
		t.Fatalf("点开通知后应回焦 sess-9：%v", focused)
	}
}

// TestNotifierReportsFailure 核对发送失败会写日志并返回原因（不静默吞掉）。
func TestNotifierReportsFailure(t *testing.T) {
	presenter := &stubPresenter{err: errors.New("系统拒绝通知")}
	var logs []string
	notifier := NewNotifier(presenter, nil, func(format string, args ...any) {
		logs = append(logs, format)
	})
	err := notifier.Notify(Event{Kind: KindRunCompleted, SessionID: "sess-10", ID: "run:sess-10:1"})
	if err == nil {
		t.Fatal("发送失败应返回原因")
	}
	if len(logs) != 1 {
		t.Fatalf("发送失败应写一条日志：%v", logs)
	}
}

// TestNotifierWithoutPresenter 核对没有通知能力时是空操作（入口在系统不支持时
// 传 nil 也能跑）。
func TestNotifierWithoutPresenter(t *testing.T) {
	var nilNotifier *Notifier
	if err := nilNotifier.Notify(Event{Kind: KindRunCompleted}); err != nil {
		t.Fatalf("空通知器应静默：%v", err)
	}
	notifier := NewNotifier(nil, nil, nil)
	if err := notifier.Notify(Event{Kind: KindRunCompleted}); err != nil {
		t.Fatalf("没有 presenter 时应静默：%v", err)
	}
}
