package session

import (
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// decodeEvent 把连接领域已识别的通知映射为类型化字段。
//
// 未知通知不解析，Method/Params 原样交给上层；已识别但字段不合法的通知
// 只记录 DecodeErr，事实本身仍然交付，避免因为一个字段丢掉一条审批。
func decodeEvent(event *Event) {
	switch event.Method {
	case protocol.MethodApprovalRequest:
		notice, err := protocol.ParseApprovalRequestNotice(event.Params)
		if err != nil {
			event.DecodeErr = err
			return
		}
		event.ApprovalRequest = &notice
	case protocol.MethodApprovalCancelled:
		notice, err := protocol.ParseApprovalCancelledNotice(event.Params)
		if err != nil {
			event.DecodeErr = err
			return
		}
		event.ApprovalCancelled = &notice
	case protocol.MethodApprovalResolved:
		notice, err := protocol.ParseApprovalResolvedNotice(event.Params)
		if err != nil {
			event.DecodeErr = err
			return
		}
		event.ApprovalResolved = &notice
	case protocol.MethodSessionNotice:
		notice, err := protocol.ParseSessionNotice(event.Params)
		if err != nil {
			event.DecodeErr = err
			return
		}
		event.SessionNotice = &notice
		if notice.IsTitle() {
			title, err := notice.ParseTitle()
			if err != nil {
				event.DecodeErr = err
				return
			}
			event.Title = &title
		}
	}
}
