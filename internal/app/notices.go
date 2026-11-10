package app

import (
	"encoding/json"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/extensions"
)

// 控制连接上的通知方法。只有这些形状已经在领域证据里出现，才在这里映射；
// 其余通知原样记一条诊断，不猜含义、不改任何状态。
const (
	noticeSchedulerChanged = "scheduler.changed"
	noticeWakeupChanged    = "wakeup.changed"
	noticeWorkflowChanged  = "workflow.changed"
)

// handleControlNotice 是控制连接的通知路由。
//
// 审批属于会话流（需要会话的裁决器），不在这里解释；标题变更没有序号、也可从
// 进程级通道到达，这里同样转给目录按修订号合并（合并是幂等的）。
func (a *App) handleControlNotice(message protocol.Message) {
	switch message.Method {
	case noticeSchedulerChanged:
		a.notifyExtensions(extensions.Notice{Topic: extensions.TopicScheduler})
	case noticeWakeupChanged:
		var params struct {
			SessionID string `json:"sessionId"`
			BranchID  string `json:"branchId"`
		}
		if err := json.Unmarshal(message.Params, &params); err != nil {
			a.record("通知 %s 参数无法解析: %v", message.Method, err)
			return
		}
		a.notifyExtensions(extensions.Notice{
			Topic:     extensions.TopicWakeups,
			SessionID: params.SessionID,
			BranchID:  params.BranchID,
		})
	case noticeWorkflowChanged:
		var params struct {
			WorkflowID string `json:"workflowId"`
		}
		if err := json.Unmarshal(message.Params, &params); err != nil {
			a.record("通知 %s 参数无法解析: %v", message.Method, err)
			return
		}
		a.notifyExtensions(extensions.Notice{
			Topic:      extensions.TopicWorkflows,
			WorkflowID: params.WorkflowID,
		})
	case protocol.MethodSessionNotice:
		a.routeSessionNotice(message.Params)
	case protocol.MethodApprovalRequest, protocol.MethodApprovalCancelled, protocol.MethodApprovalResolved:
		// 审批通知按会话归属解释；控制连接上收到时直接丢弃，不重复入队。
	case "session.event", "session.model.progress":
		// 时间线事实只走会话流。
	default:
		// 未映射的通知不解释；只留一条诊断，供后续接线时对照。
		a.record("控制连接收到未映射的通知: %s", message.Method)
	}
}

// notifyExtensions 把通知交给扩展与后台领域的事件跟随。
func (a *App) notifyExtensions(notice extensions.Notice) {
	service := a.Extensions()
	if service == nil {
		return
	}
	if err := service.Notify(notice); err != nil {
		a.record("事件跟随: %s 通知入队失败: %v", notice.Topic, err)
	}
}
