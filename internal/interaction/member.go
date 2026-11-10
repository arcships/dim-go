package interaction

import (
	"encoding/json"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// MemberRef 指向一个成员运行。BranchID 是该成员所在分支；空表示由发送方按
// 成员自己的运行位置处理。
type MemberRef struct {
	MemberID string
	BranchID string
}

// StopMember 对指定成员的运行发一次停止意图，不触碰主线运行，也不改变待决集合。
//
// 停止意图带 memberId，发送方必须据此把作用范围收在该成员上，不得退化成会话
// 停止。幂等键按（会话、分支、成员）固定：重复点击返回同一写意图，不重复发送。
func (a *Arbiter) StopMember(member MemberRef) (write.IntentView, error) {
	if member.MemberID == "" {
		return write.IntentView{}, fmt.Errorf("%w: %s", ErrInvalidMember, a.sessionID)
	}
	params, err := json.Marshal(map[string]string{"memberId": member.MemberID})
	if err != nil {
		return write.IntentView{}, fmt.Errorf("编码成员停止参数: %w", err)
	}
	intent := write.Intent{
		Key:       fmt.Sprintf("interaction/member-stop/%s/%s/%s", a.sessionID, member.BranchID, member.MemberID),
		Kind:      write.KindStop,
		SessionID: a.sessionID,
		BranchID:  member.BranchID,
		Params:    params,
	}
	return a.writer.Register(intent)
}
