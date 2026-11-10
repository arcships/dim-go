package fakeappserver

import (
	"encoding/json"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// Script 是假服务端的应答脚本：固定事实、其他方法的回执与故障注入。
//
// 未列在 Replies 里的方法走内置处理（hello、daemon.ping、daemon.shutdown、
// session.attach、session.detach、session.history）；仍未命中的方法按未知
// 方法回 unknown_method，与真实服务端的能力缺失一致。
type Script struct {
	// InstanceID 是 hello 回执里的实例标识；空时用固定值，保证同一假服务端稳定。
	InstanceID string
	// ServerVersion 是 hello 回执里的服务端版本；空时用 "fake"。
	ServerVersion string
	// ProtocolVersion 是 hello 回执里的协议版本；0 表示当前协议版本。
	// 设为其他值即可注入版本不符。
	ProtocolVersion int
	// Profile 是 hello 回执里的服务端画像；零值时按 mock 处理。
	Profile protocol.DaemonProfile
	// Features 是 hello 回执声明的可选能力。缺项按能力缺失处理，不猜。
	Features []string
	// HelloError 非空时 hello 直接回错误回执，用于注入 appError 形式的版本不符。
	HelloError *protocol.RPCError
	// Sessions 是脚本里的全部会话事实。
	Sessions []Session
	// Replies 是按方法名给出的脚本应答；同名时后者覆盖前者。
	Replies []Reply
	// Faults 是故障注入规则，按方法名与出现次数命中。
	Faults []Fault
	// StopOnShutdown 为真时，daemon.shutdown 回执后停止监听并断开全部连接。
	StopOnShutdown bool
}

// Session 是一个会话的固定事实。
type Session struct {
	// ID 是会话标识，不能为空。
	ID string
	// BranchID 是主线标识；空时用 "main"。
	BranchID string
	// Running 是 attach 回执里的运行状态。
	Running bool
	// Head 是 attach 回执里的头部序号；0 时取事件里的最大序号。
	Head int64
	// LatestTurnStart 是最近一轮的起始序号；大于 0 时 latestTurn 读取从它开始。
	// 事件词表尚未冻结，因此用样例元数据表达轮次边界，不猜事件类型。
	LatestTurnStart int64
	// Events 是会话历史，按序号升序；假服务端按 after/through/limit/latestTurn
	// 过滤后返回，不重排、不合并。
	Events []protocol.RawEvent
	// AttachReplay 在 attach 回执之前逐条发送，模拟服务端的订阅重放。
	AttachReplay []protocol.RawEvent
}

// Reply 是一条按方法名给出的脚本应答。Result 与 Error 只能有一个非空；
// 都为空时回执 result 为 null。
type Reply struct {
	Method string
	// Result 是任意可 JSON 编码的回执值。目录、提交、停止等尚未在 protocol
	// 包定义形状的方法由调用方在这里给出回执，本包不复制 wire 类型。
	Result any
	// Error 是错误回执；用 AppError 构造带稳定 appError 的错误。
	Error *protocol.RPCError
}

// Fault 是一条故障注入规则。Occurrence 从 1 开始，表示同一方法的第几次请求；
// 0 表示每次都命中。同一方法有多条规则时按声明顺序取第一条命中的。
type Fault struct {
	Method     string
	Occurrence int
	// Delay 在应答前等待，用于制造乱序回执（后到的请求先回）。
	Delay time.Duration
	// NoReply 丢弃回执，让调用方走到超时分支。
	NoReply bool
	// DisconnectBefore 在应答前断开连接。
	DisconnectBefore bool
	// DisconnectAfter 在应答后断开连接。
	DisconnectAfter bool
	// UnknownMethod 忽略正常应答，回 unknown_method；用于注入能力缺失。
	UnknownMethod bool
	// OversizeFrameBytes 大于 0 时，在正常应答前插入一条该大小的帧。
	// 需要调用方把 protocol.Options.MaxFrameBytes 调到该大小以下，才能看到
	// 客户端丢弃超大帧并继续读后续帧。
	OversizeFrameBytes int
}

// AppError 构造带稳定 appError 的错误回执，供脚本应答与故障注入使用。
func AppError(code int, appError, message string) *protocol.RPCError {
	data, _ := json.Marshal(struct {
		AppError string `json:"appError"`
	}{AppError: appError})
	return &protocol.RPCError{Code: code, Message: message, Data: data}
}

func unknownMethodError(method string) *protocol.RPCError {
	return AppError(-32601, "unknown_method", "unknown method "+method)
}

// withDefaults 补齐脚本的默认值；不修改调用方传入的切片内容。
func (s Script) withDefaults() Script {
	if s.InstanceID == "" {
		s.InstanceID = "fake-instance"
	}
	if s.ServerVersion == "" {
		s.ServerVersion = "fake"
	}
	if s.ProtocolVersion == 0 {
		s.ProtocolVersion = protocol.ProtocolVersion
	}
	if s.Profile.Kind == "" {
		s.Profile = protocol.DaemonProfile{Kind: "mock"}
	}
	return s
}
