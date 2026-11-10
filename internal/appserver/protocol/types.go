package protocol

import "encoding/json"

// ProtocolVersion 是本客户端实现的 App Server 协议版本。
const ProtocolVersion = 3

// HelloParams 是 `hello` 请求参数。
type HelloParams struct {
	ProtocolVersion int    `json:"protocolVersion"`
	ClientKind      string `json:"clientKind"`
	ClientName      string `json:"clientName,omitempty"`
}

// DaemonProfile 是 `hello` 回执里的服务端画像。
type DaemonProfile struct {
	Kind              string `json:"kind"`
	ModelConfigSHA256 string `json:"modelConfigSha256,omitempty"`
}

// HelloResult 是 `hello` 回执。Features 里是服务端声明的可选方法；缺项按能力
// 缺失处理，不猜。
type HelloResult struct {
	InstanceID      string        `json:"instanceId"`
	ServerVersion   string        `json:"serverVersion"`
	ProtocolVersion int           `json:"protocolVersion"`
	DaemonProfile   DaemonProfile `json:"daemonProfile"`
	Features        []string      `json:"features,omitempty"`
}

// HasFeature 判断服务端是否声明了某个可选方法。
func (h HelloResult) HasFeature(name string) bool {
	for _, feature := range h.Features {
		if feature == name {
			return true
		}
	}
	return false
}

// PingResult 是 `daemon.ping` 回执。
type PingResult struct {
	Sessions      int    `json:"sessions"`
	Connections   int    `json:"connections"`
	ServerVersion string `json:"serverVersion"`
}

// ShutdownParams 是 `daemon.shutdown` 参数。IfIdle 只在服务端声明
// `daemon.shutdown.ifIdle` 时发送；旧服务端拒绝未知字段。
type ShutdownParams struct {
	Force  bool `json:"force"`
	IfIdle bool `json:"ifIdle,omitempty"`
}

// SessionAttachParams 是 `session.attach` 参数。
type SessionAttachParams struct {
	SessionID string `json:"sessionId"`
}

// SessionModel 是会话当前采用的模型。
type SessionModel struct {
	ProviderID string `json:"providerId,omitempty"`
	ModelID    string `json:"modelId,omitempty"`
}

// SessionAttachResult 是 `session.attach` 回执。Sequence 是主线头部序号；
// 旧服务端可能省略，此时由会话流补一次 `session.history` 取头部。
type SessionAttachResult struct {
	SessionID        string          `json:"sessionId"`
	Sequence         *int64          `json:"sequence,omitempty"`
	Running          *bool           `json:"running,omitempty"`
	Model            *SessionModel   `json:"model,omitempty"`
	ModelPreferences json.RawMessage `json:"modelPreferences,omitempty"`
}

// SessionDetachParams 是 `session.detach` 参数。
type SessionDetachParams struct {
	SessionID string `json:"sessionId"`
}

// SessionHistoryParams 是 `session.history` 参数。游标都是事件序号，自增。
type SessionHistoryParams struct {
	SessionID  string `json:"sessionId"`
	BranchID   string `json:"branchId,omitempty"`
	After      *int64 `json:"after,omitempty"`
	Limit      *int   `json:"limit,omitempty"`
	Through    *int64 `json:"through,omitempty"`
	LatestTurn *bool  `json:"latestTurn,omitempty"`
	Before     *int64 `json:"before,omitempty"`
}

// RawEvent 是服务端时间线上的一条事实。字段名沿用 wire 的 snake_case。
type RawEvent struct {
	ID       string          `json:"id"`
	Sequence int64           `json:"sequence"`
	CreateAt int64           `json:"create_at"`
	Tags     []string        `json:"tags,omitempty"`
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Data     json.RawMessage `json:"data,omitempty"`
}

// SessionHistoryResult 是 `session.history` 回执。
type SessionHistoryResult struct {
	Events     []RawEvent `json:"events"`
	Through    int64      `json:"through"`
	NextAfter  int64      `json:"nextAfter"`
	Done       bool       `json:"done"`
	NextBefore *int64     `json:"nextBefore,omitempty"`
	TurnStart  *int64     `json:"turnStart,omitempty"`
}

// SessionEventParams 是 `session.event` 通知参数。
type SessionEventParams struct {
	SessionID string   `json:"sessionId"`
	BranchID  string   `json:"branchId"`
	Event     RawEvent `json:"event"`
}
