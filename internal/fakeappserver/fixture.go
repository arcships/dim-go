package fakeappserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// 样例里的期望值。
const (
	// ExpectDelivered 表示该通知应当交付给上层。
	ExpectDelivered = "delivered"
	// ExpectDropped 表示该通知应当被丢弃（重复或旧序号）。
	ExpectDropped = "dropped"
)

// HistoryFixture 是 testdata 里的历史样例。每个样例自带 description 与逐条
// comment，说明它验证什么；加载后可直接转成 Session 交给假服务端。
type HistoryFixture struct {
	// Description 说明整份样例验证什么。
	Description string `json:"description"`
	// SessionID 是样例会话的标识。
	SessionID string `json:"sessionId"`
	// Features 是伴随样例声明的可选能力，例如 session.history.latestTurn。
	Features []string `json:"features"`
	// Head 是头部序号；0 时取事件里的最大序号。
	Head int64 `json:"head"`
	// LatestTurnStart 是最近一轮的起始序号；0 表示样例不区分轮次。
	// 事件词表尚未冻结，所以用元数据表达轮次边界，不猜事件类型。
	LatestTurnStart int64 `json:"latestTurnStart"`
	// Events 是按序号升序的事实序列。
	Events []FixtureEvent `json:"events"`
}

// FixtureEvent 是一条带说明的历史事实。
type FixtureEvent struct {
	// Comment 说明这条事实在样例里验证什么。
	Comment string `json:"comment"`
	protocol.RawEvent
}

// NotificationFixture 是 testdata 里的通知样例。每条通知带 comment 与
// expect，测试按 expect 判断交付还是丢弃。
type NotificationFixture struct {
	// Description 说明整份样例验证什么。
	Description string `json:"description"`
	// SessionID 是样例会话的标识；通知参数里的序号与历史样例对齐。
	SessionID string `json:"sessionId"`
	// Notifications 是按发送顺序排列的通知。
	Notifications []FixtureNotification `json:"notifications"`
}

// FixtureNotification 是一条带说明与期望结果的通知。
type FixtureNotification struct {
	// Comment 说明这条通知验证什么。
	Comment string `json:"comment"`
	// Method 是通知方法名。
	Method string `json:"method"`
	// Params 是通知参数的 wire 形状。
	Params json.RawMessage `json:"params"`
	// Expect 是 ExpectDelivered 或 ExpectDropped。
	Expect string `json:"expect"`
}

// LoadHistoryFile 从 JSON 文件加载历史样例并校验结构。
func LoadHistoryFile(path string) (HistoryFixture, error) {
	var fixture HistoryFixture
	if err := loadJSONFile(path, &fixture); err != nil {
		return fixture, err
	}
	if err := fixture.validate(); err != nil {
		return fixture, fmt.Errorf("%s: %w", path, err)
	}
	return fixture, nil
}

// LoadNotificationsFile 从 JSON 文件加载通知样例并校验结构。
func LoadNotificationsFile(path string) (NotificationFixture, error) {
	var fixture NotificationFixture
	if err := loadJSONFile(path, &fixture); err != nil {
		return fixture, err
	}
	if err := fixture.validate(); err != nil {
		return fixture, fmt.Errorf("%s: %w", path, err)
	}
	return fixture, nil
}

// Session 把历史样例转换成假服务端的会话事实。
// AttachReplay 不设置，实时与重放由调用方按需另外推送。
func (f HistoryFixture) Session() Session {
	events := f.RawEvents()
	return Session{
		ID:              f.SessionID,
		Head:            f.Head,
		LatestTurnStart: f.LatestTurnStart,
		Events:          events,
	}
}

// RawEvents 返回去掉 comment 的事实序列副本。
func (f HistoryFixture) RawEvents() []protocol.RawEvent {
	events := make([]protocol.RawEvent, 0, len(f.Events))
	for _, event := range f.Events {
		events = append(events, event.RawEvent)
	}
	return events
}

// LatestTurnSequences 返回最近一轮的序号；没有轮次信息时返回全部序号。
func (f HistoryFixture) LatestTurnSequences() []int64 {
	sequences := make([]int64, 0, len(f.Events))
	for _, event := range f.Events {
		if f.LatestTurnStart > 0 && event.Sequence < f.LatestTurnStart {
			continue
		}
		sequences = append(sequences, event.Sequence)
	}
	return sequences
}

// EventSequences 返回全部事实的序号，按样例顺序。
func (f HistoryFixture) EventSequences() []int64 {
	sequences := make([]int64, 0, len(f.Events))
	for _, event := range f.Events {
		sequences = append(sequences, event.Sequence)
	}
	return sequences
}

func (f HistoryFixture) validate() error {
	if f.SessionID == "" {
		return errors.New("样例缺少 sessionId")
	}
	if len(f.Events) == 0 {
		return errors.New("样例没有历史事件")
	}
	var last int64
	ids := make(map[string]struct{}, len(f.Events))
	for index, event := range f.Events {
		if event.ID == "" {
			return fmt.Errorf("第 %d 条事件缺少 id", index+1)
		}
		if _, duplicate := ids[event.ID]; duplicate {
			return fmt.Errorf("事件 id 重复: %q", event.ID)
		}
		ids[event.ID] = struct{}{}
		if event.Sequence <= 0 {
			return fmt.Errorf("第 %d 条事件的 sequence 必须大于 0", index+1)
		}
		if index > 0 && event.Sequence <= last {
			return fmt.Errorf("事件序号必须严格递增: %d 之后出现 %d", last, event.Sequence)
		}
		last = event.Sequence
	}
	if f.Head > 0 && f.Head < last {
		return fmt.Errorf("head %d 小于最大事件序号 %d", f.Head, last)
	}
	if f.LatestTurnStart > 0 {
		found := false
		for _, event := range f.Events {
			if event.Sequence == f.LatestTurnStart {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("latestTurnStart %d 不是任何事件的序号", f.LatestTurnStart)
		}
	}
	return nil
}

func (f NotificationFixture) validate() error {
	if f.SessionID == "" {
		return errors.New("样例缺少 sessionId")
	}
	if len(f.Notifications) == 0 {
		return errors.New("样例没有通知")
	}
	for index, notification := range f.Notifications {
		if notification.Method == "" {
			return fmt.Errorf("第 %d 条通知缺少 method", index+1)
		}
		if len(notification.Params) == 0 || !json.Valid(notification.Params) {
			return fmt.Errorf("第 %d 条通知的 params 不是合法 JSON", index+1)
		}
		switch notification.Expect {
		case ExpectDelivered, ExpectDropped:
		default:
			return fmt.Errorf("第 %d 条通知的 expect 必须是 %q 或 %q", index+1, ExpectDelivered, ExpectDropped)
		}
	}
	return nil
}

func loadJSONFile(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取样例 %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("解析样例 %s: %w", path, err)
	}
	return nil
}
