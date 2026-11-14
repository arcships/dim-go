package ui

import (
	"context"
	"time"
)

// Source 是视图读取事实的唯一入口（A 领域边界）：视图只认这个接口，
// 不 import internal/app。适配器在 internal/uiwire。
//
// 约定：每帧从 Source 读；view 不缓存事实（只在 Source 返回的快照上读），
// 可丢的视图状态（展开、焦点、草稿、滚动）仍留 view。
type Source interface {
	// Connection 是连接阶段；Phase 为 "connected" 时其余读才有意义。
	Connection() Connection
	// SessionNames 是左栏的会话列表（按目录顺序）。
	SessionNames() []string
	// Session 是当前会话的快照；ok 为假表示还没打开会话。
	Session() (SessionSnapshot, bool)
	// Approval 是当前待决审批；ok 为假表示没有。
	Approval() (ApprovalSnapshot, bool)
	// Question 是当前待答提问；ok 为假表示没有。
	Question() (QuestionSnapshot, bool)
	// Account 是页脚的身份与连接态。
	Account() AccountSnapshot
	// Settings 是设置里来自事实的那几项。
	Settings() SettingsSnapshot
}

// Intents 是视图发起写的唯一入口：写走既有的 app 方法，不新开协议通道。
type Intents interface {
	OpenSession(ctx context.Context, name string) error
	Submit(ctx context.Context, text string) error
	Queue(ctx context.Context, text string) error
	Stop(ctx context.Context) error
	RespondApproval(ctx context.Context, decision string) error
	RespondQuestion(ctx context.Context, answer string) error
	RetrySubmit(ctx context.Context) error
	CreateSession(ctx context.Context, cwd string) error
}

// Connection 是连接阶段与原因（断线要能看见原因，不是白屏）。
type Connection struct {
	Phase  string // "connecting" / "connected" / "disconnected"
	Reason string
}

// SessionSnapshot 是首片要用的会话事实（与 app.SessionView 对齐，不多造）。
type SessionSnapshot struct {
	ID        string
	Title     string
	Running   bool
	Rows      []RowSnapshot
	Pending   int
	Goal      string
	OpenedAt  time.Time
}

// RowSnapshot 是时间线的一行（用户 / 助手 / 过程组）。
type RowSnapshot struct {
	Kind  string // "user" / "assistant" / "group"
	Text  string
	Ops   int
}

// ApprovalSnapshot 是一条待决审批。
type ApprovalSnapshot struct {
	ID       string
	Command  string
	Impact   string
	LongTerm bool
}

// QuestionSnapshot 是一条待答提问。
type QuestionSnapshot struct {
	ID       string
	Text     string
	Options  []string
}

// AccountSnapshot 是页脚身份。
type AccountSnapshot struct {
	Initial   string
	Connected bool
}

// SettingsSnapshot 是设置里来自事实的几项。
type SettingsSnapshot struct {
	Provider string
	Model    string
	Tier     string
}
