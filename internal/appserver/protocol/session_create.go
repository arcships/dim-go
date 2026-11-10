package protocol

// SessionCreateResult 是 `session.create` 回执。它只被连接领域的证据程序用到，
// 用来拿到一个可打开的真实会话；会话创建语义本身归会话目录领域。
type SessionCreateResult struct {
	SessionID string `json:"sessionId"`
}
