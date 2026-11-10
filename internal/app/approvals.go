package app

import (
	"encoding/json"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/interaction"
)

// routeSessionEvent 是会话流通知在集成层的唯一路由：审批、标题、提问与其余通知各归其主。
//
// 认不出形状的通知只记一条诊断，不猜含义、不改状态。折叠仍由投影的监听器负责，
// 本函数只把不属于折叠的通知转给交互裁决与目录。
func (a *App) routeSessionEvent(sessionID string, arbiter *interaction.Arbiter, event session.Event) {
	switch event.Method {
	case protocol.MethodApprovalRequest:
		notice, err := protocol.ParseApprovalRequestNotice(event.Params)
		if err != nil {
			a.record("审批请求无法解析: %v", err)
			return
		}
		for _, fact := range approvalFacts(notice) {
			for _, rejection := range arbiter.Apply(fact) {
				a.record("审批事实被丢弃: 请求=%s 原因=%s", fact.RequestID, rejection.Reason)
			}
		}
	case protocol.MethodApprovalCancelled:
		notice, err := protocol.ParseApprovalCancelledNotice(event.Params)
		if err != nil {
			a.record("审批取消通知无法解析: %v", err)
			return
		}
		// 取消即服务端结算：调用没有开始，条目从队列移除并推进。
		arbiter.Settle(notice.RequestID)
	case protocol.MethodApprovalResolved:
		notice, err := protocol.ParseApprovalResolvedNotice(event.Params)
		if err != nil {
			a.record("审批结算通知无法解析: %v", err)
			return
		}
		arbiter.Settle(notice.RequestID)
	case protocol.MethodSessionNotice:
		a.routeSessionNotice(event.Params)
	case "session.event":
		a.routeTimelineFact(arbiter, event.Params)
	default:
		// 这些通知已经在别处消费或暂不消费，不重复记诊断：
		// session.model.progress 与 session.state 不影响本版持有的状态。
		switch event.Method {
		case "session.model.progress", "session.state":
			return
		}
		a.record("会话流收到未映射的通知: %s（会话 %s）", event.Method, sessionID)
	}
}

// routeTimelineFact 把时间线里的 ask_user 调用与工具结果转给交互裁决：
// 清单截止点之后新出现的提问要进队列，工具结果要结算已经答完的提问。
func (a *App) routeTimelineFact(arbiter *interaction.Arbiter, params json.RawMessage) {
	var envelope protocol.SessionEventParams
	if err := json.Unmarshal(params, &envelope); err != nil {
		return
	}
	event := envelope.Event
	data := decodeEventData(event.Data)
	switch event.Type + "/" + event.Name {
	case "step/tool-use":
		requestID := firstString(data, "request_id", "requestId")
		a.applyAskUser(arbiter, event, requestID, objectValue(data["call"]))
	case "step/tool-call":
		requestID := firstString(data, "request_id", "requestId")
		for _, value := range arrayValue(data["calls"]) {
			a.applyAskUser(arbiter, event, requestID, objectValue(value))
		}
	case "permission/changed":
		a.routePermissionChanged(arbiter, event.Data)
	case "step/tool-completed":
		a.settleQuestionCall(arbiter, event, data)
	case "step/tool-result":
		requestID := firstString(data, "request_id", "requestId")
		for _, value := range arrayValue(data["results"]) {
			callID := firstString(objectValue(value), "call_id", "callId")
			if callID == "" {
				continue
			}
			arbiter.SettleQuestion(requestID, callID, event.Sequence)
		}
	}
}

// applyAskUser 把一次 ask_user 调用转给交互裁决；非提问工具直接忽略。
func (a *App) applyAskUser(
	arbiter *interaction.Arbiter,
	event protocol.RawEvent,
	requestID string,
	call map[string]any,
) {
	if len(call) == 0 || !isAskUserTool(firstString(call, "name")) {
		return
	}
	item := interaction.QuestionItem{
		RequestID: requestID,
		CallID:    firstString(call, "id", "call_id", "callId"),
		EventID:   event.ID,
		Sequence:  event.Sequence,
		ToolName:  firstString(call, "name"),
		Input:     rawValue(call["input"]),
	}
	for _, rejection := range arbiter.ApplyAskUser(item) {
		// 截止点以下、已结算或重复的调用被丢弃是正常情况，只有其它原因值得记一条。
		switch rejection.Reason {
		case interaction.RejectionBelowCutoff,
			interaction.RejectionAlreadySettled,
			interaction.RejectionDuplicateRequest:
			continue
		}
		a.record("提问调用被丢弃: 请求=%s 调用=%s 原因=%s", item.RequestID, item.CallID, rejection.Reason)
	}
}

// settleQuestionCall 用一条工具结果结算 ask_user 调用。
func (a *App) settleQuestionCall(arbiter *interaction.Arbiter, event protocol.RawEvent, data map[string]any) {
	callID := firstString(data, "call_id", "callId")
	if callID == "" {
		return
	}
	arbiter.SettleQuestion(firstString(data, "request_id", "requestId"), callID, event.Sequence)
}

// routeSessionNotice 处理 session.notice：目前只认标题变更，交给目录按修订号合并。
func (a *App) routeSessionNotice(params json.RawMessage) {
	notice, err := protocol.ParseSessionNotice(params)
	if err != nil {
		a.record("会话通知无法解析: %v", err)
		return
	}
	changed, err := notice.ParseTitle()
	if err != nil {
		a.record("会话通知 %s/%s 暂不处理: %v", notice.Source, notice.Kind, err)
		return
	}
	catalogDomain := a.Catalog()
	if catalogDomain == nil {
		return
	}
	update := catalog.TitleUpdate{
		SessionID: changed.SessionID,
		Revision:  changed.TitleRevision,
	}
	if changed.Title != nil {
		update.Title = *changed.Title
	}
	if changed.TitleSource != nil {
		update.Source = catalog.TitleSource(*changed.TitleSource)
	}
	if catalogDomain.ApplyTitle(update) {
		a.record("标题已合并: 会话=%s 修订=%d", changed.SessionID, changed.TitleRevision)
	}
}

// approvalFacts 把一条审批请求映射成交互裁决的事实。
func approvalFacts(notice protocol.ApprovalRequestNotice) []interaction.Fact {
	fact := interaction.Fact{
		Kind:      interaction.PendingApproval,
		RequestID: notice.RequestID,
		SessionID: notice.SessionID,
		BranchID:  notice.BranchID,
		TurnID:    notice.Source.TurnID,
		Action:    approvalAction(notice),
	}
	fact.AllowLongTerm, fact.LongTermParams = longTermRule(notice)
	return []interaction.Fact{fact}
}

// approvalAction 给出队首展示用的动作描述：工具名，附带服务端给出的原因。
func approvalAction(notice protocol.ApprovalRequestNotice) string {
	switch {
	case notice.Call.Name == "":
		return notice.Reason
	case notice.Reason == "":
		return notice.Call.Name
	default:
		return notice.Call.Name + "：" + notice.Reason
	}
}

// longTermRule 决定这次审批能否长期允许，以及写入哪些规则参数。
//
// 服务端真实存在的长期允许只有用户级命令规则（`permission.rules` /
// `permission.set_rules`），不是审批应答里的第三种决定，也没有通用的
// tool/path 规则。因此：
//   - 只认 tool 来源的 `exec`/`execute` 调用，input.command 必须是简单的非空命令；
//   - 含命令分隔符（; & | ` ( ) < > $ 或换行）的复杂命令不长期允许；
//   - 规则参数为 {"command":<命令>,"match":"exact","decision":"allow"}。
//   - 其余工具（读写文件等）暂不支持长期允许，避免客户端假装有服务端没有的能力。
func longTermRule(notice protocol.ApprovalRequestNotice) (bool, json.RawMessage) {
	if notice.Source.Kind != "tool" || !isCommandTool(notice.Call.Name) {
		return false, nil
	}
	var input map[string]json.RawMessage
	if err := json.Unmarshal(notice.Call.Input, &input); err != nil {
		return false, nil
	}
	rawCommand, ok := input["command"]
	if !ok {
		return false, nil
	}
	var command string
	if err := json.Unmarshal(rawCommand, &command); err != nil {
		return false, nil
	}
	command = strings.TrimSpace(command)
	if !isSimpleCommand(command) {
		return false, nil
	}
	params, err := json.Marshal(map[string]string{
		"command":  command,
		"match":    "exact",
		"decision": "allow",
	})
	if err != nil {
		return false, nil
	}
	return true, params
}

// isCommandTool 判断工具是否是启动进程的命令工具；与服务端命令规则覆盖范围一致。
func isCommandTool(name string) bool {
	return name == "exec" || name == "execute"
}

// isSimpleCommand 判断命令是否没有服务端定义的命令分隔符。
func isSimpleCommand(command string) bool {
	if strings.TrimSpace(command) == "" {
		return false
	}
	return !strings.ContainsAny(command, ";&|`()<>$\n\r")
}

// isAskUserTool 判断工具名是否提问工具；与交互裁决的判定保持一致。
func isAskUserTool(name string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(name))
	return normalized == "askuser" || normalized == "askuserquestion"
}

// decodeEventData 把事实的 data 解成对象；解不开时返回空对象。
func decodeEventData(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	return data
}

func objectValue(value any) map[string]any {
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return nil
}

func arrayValue(value any) []any {
	if array, ok := value.([]any); ok {
		return array
	}
	return nil
}

func rawValue(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

// firstString 取第一个非空字符串字段。
func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}
