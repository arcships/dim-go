// Package notify 是宿主外壳的系统通知策略：什么时候发、发哪一类、点开之后回到
// 哪个会话。
//
// 它不碰平台 API：入口把系统通知的能力实现成 Presenter（mygo 的 Notification），
// 本包只决定事件与文案。这样「轮完成 / 轮失败 / 待回应」三类通知、去重与开关
// 都能在没有窗口的测试里逐条核对。
//
// 事实边界：Watcher 只读入口每帧交来的会话状态（运行中、待决、最后一轮的结果），
// 不读会话、不存事实；它记住的只是「已经发过哪一条」，用于去重。
package notify

import (
	"fmt"
	"strings"
	"sync"
)

// Kind 是通知的三类。与参考产品一致：轮完成、轮失败、需要用户回应。
type Kind string

const (
	// KindRunCompleted 是一轮正常结束。
	KindRunCompleted Kind = "run_completed"
	// KindRunFailed 是一轮以失败/取消收口。
	KindRunFailed Kind = "run_failed"
	// KindActionRequired 是审批或提问待决。
	KindActionRequired Kind = "interaction_required"
)

// Prefs 是设置 › 通知的三个开关。缺省（全开）由集成层的偏好给出，不在本包兜底。
type Prefs struct {
	// TaskDone 是「任务完成或失败时通知」。
	TaskDone bool
	// Sound 是「通知声音」：发通知时带系统提示音。
	Sound bool
	// NeedsResponse 是「需要我回应时通知」。
	NeedsResponse bool
}

// Session 是通知要读的会话状态。
type Session struct {
	ID    string
	Title string
	// Running 表示当前有一轮在跑。
	Running bool
	// Pending 表示有审批或提问待决；PendingID 是待决项的标识（去重用）。
	Pending   bool
	PendingID string
	// LastTurn 是最后一个已结束的轮：Ended 为假表示还没有结束的轮，
	// Outcome 是服务端给的结果词（failed / error / cancelled 算失败）。
	LastTurnEnded   bool
	LastTurnIndex   int
	LastTurnOutcome string
}

// Event 是一条要发的通知。
type Event struct {
	Kind         Kind
	SessionID    string
	SessionTitle string
	// ID 是通知标识：同一条事实重复到达时只发一次，也让系统替换旧通知。
	ID string
	// Silent 表示不发提示音（「通知声音」关掉时）。
	Silent bool
}

// Watcher 把连续观察到的会话状态变成通知事件。
type Watcher struct {
	mu       sync.Mutex
	previous map[string]observation
}

// observation 是一个会话上一次观察到的状态与已经发过的通知标识。
type observation struct {
	running  bool
	pending  bool
	runID    string
	actionID string
}

// NewWatcher 建一个观察器；第一次观察只建立基线，不发通知。
func NewWatcher() *Watcher {
	return &Watcher{previous: map[string]observation{}}
}

// Observe 读一次会话状态并返回要发的通知。
//
// 只有状态迁移才发：运行中 → 停止（按最后一轮的结果分完成/失败）、无待决 →
// 有待决。进程刚起来时已经处于停止/待决的会话不补发——那不是这段时间发生的事。
func (w *Watcher) Observe(s Session, prefs Prefs) []Event {
	if strings.TrimSpace(s.ID) == "" {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	prev, seen := w.previous[s.ID]
	next := prev
	next.running = s.Running
	next.pending = s.Pending
	var events []Event
	if !seen {
		if !s.Pending {
			next.actionID = ""
		}
		w.previous[s.ID] = next
		return nil
	}
	if prev.running && !s.Running && s.LastTurnEnded {
		runID := fmt.Sprintf("run:%s:%d", s.ID, s.LastTurnIndex)
		if prev.runID != runID {
			next.runID = runID
			if prefs.TaskDone {
				kind := KindRunCompleted
				if failed(s.LastTurnOutcome) {
					kind = KindRunFailed
				}
				events = append(events, Event{
					Kind: kind, SessionID: s.ID, SessionTitle: s.Title,
					ID: runID, Silent: !prefs.Sound,
				})
			}
		}
	}
	switch {
	case s.Pending:
		actionID := pendingID(s)
		if !prev.pending || (prev.actionID != "" && prev.actionID != actionID) {
			next.actionID = actionID
			if prefs.NeedsResponse && prev.actionID != actionID {
				events = append(events, Event{
					Kind: KindActionRequired, SessionID: s.ID, SessionTitle: s.Title,
					ID: actionID, Silent: !prefs.Sound,
				})
			}
		}
	default:
		next.actionID = ""
	}
	w.previous[s.ID] = next
	return events
}

// pendingID 是待决项的稳定标识：有具体标识就用它，否则按会话记一个。
func pendingID(s Session) string {
	key := strings.TrimSpace(s.PendingID)
	if key == "" {
		key = "pending"
	}
	return "interaction:" + s.ID + ":" + key
}

// failed 判断一轮的结果词是不是失败（服务端只给结果词，不给原因）。
func failed(outcome string) bool {
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "failed", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

// Note 是一条系统通知。入口按它创建平台的 Notification。
type Note struct {
	ID        string
	Title     string
	Subtitle  string
	Body      string
	Group     string
	Silent    bool
	Kind      Kind
	SessionID string
	// OnClick 在用户点开通知时调用；入口把它接到平台的通知点击上。
	OnClick func()
}

// Presenter 是系统通知的能力。Show 只负责把通知交给系统；用户点开时调用
// Note.OnClick。返回错误表示这条通知没能发出去。
type Presenter interface {
	Show(note Note) error
}

// Notifier 把事件变成系统通知，并记住点开时该回到哪个会话。
type Notifier struct {
	mu        sync.Mutex
	presenter Presenter
	focus     func(sessionID string)
	logf      func(format string, args ...any)
}

// NewNotifier 建一个通知器。focus 在用户点开通知时收到会话标识（入口负责把窗口
// 带到前台并定位到该会话）；logf 记录发送失败，可为 nil。
func NewNotifier(presenter Presenter, focus func(sessionID string), logf func(format string, args ...any)) *Notifier {
	return &Notifier{presenter: presenter, focus: focus, logf: logf}
}

// Notify 发一条通知，返回系统拒绝或发送失败的原因。
func (n *Notifier) Notify(event Event) error {
	if n == nil || n.presenter == nil {
		return nil
	}
	note := Note{
		ID:        event.ID,
		Title:     "DimAgent",
		Subtitle:  event.SessionTitle,
		Body:      body(event.Kind),
		Group:     group(event.Kind, event.SessionID),
		Silent:    event.Silent,
		Kind:      event.Kind,
		SessionID: event.SessionID,
	}
	sessionID := event.SessionID
	note.OnClick = func() {
		n.mu.Lock()
		focus := n.focus
		n.mu.Unlock()
		if focus != nil {
			focus(sessionID)
		}
	}
	if err := n.presenter.Show(note); err != nil {
		if n.logf != nil {
			n.logf("系统通知发送失败（%s / 会话 %s）：%v", event.Kind, event.SessionID, err)
		}
		return err
	}
	return nil
}

// body 是三条通知的正文，逐字取参考产品的中文文案。
func body(kind Kind) string {
	switch kind {
	case KindRunCompleted:
		return "当前任务已完成"
	case KindRunFailed:
		return "当前任务失败"
	case KindActionRequired:
		return "任务正在等待你的回答、选择或批准"
	default:
		return ""
	}
}

// group 让同一会话的通知在通知中心叠在一起（参考产品的分组口径）。
func group(kind Kind, sessionID string) string {
	if kind == KindActionRequired {
		return "session:" + sessionID + ":interactions"
	}
	return "session:" + sessionID + ":runs"
}
