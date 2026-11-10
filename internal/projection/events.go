package projection

import (
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// apply 是折叠的唯一实现：一条事实按类型与名字落到它对应的事实生命周期。
// 本版不认识的事实不建行、不改状态，也不猜它的含义。
func (s *foldState) apply(event protocol.RawEvent) {
	if event.Sequence > s.maxSequence {
		s.maxSequence = event.Sequence
	}
	data := decodeObject(event.Data)
	switch event.Type + "/" + event.Name {
	case "input/message":
		s.applyInput(event, data)
	case "input/applied":
		s.applyInputApplied(event, data)
	case "input/consumed":
		s.applyInputConsumed(event, data)
	case "input/deleted", "input/withdraw":
		s.applyInputDeleted(event, data)
	case "turn/start":
		s.applyTurnStart(event, data)
	case "turn/end":
		s.applyTurnEnd(event, data)
	case "step/text":
		s.applyDelta(event, data, RowAssistantText)
	case "step/thinking":
		s.applyDelta(event, data, RowThinking)
	case "step/model-output-end":
		s.applyModelOutputEnd(event, data)
	case "step/tool-use-start":
		s.applyToolStart(event, data)
	case "step/tool-use":
		s.applyToolUse(event, data)
	case "step/tool-call":
		s.applyToolCalls(event, data)
	case "step/tool-completed":
		s.applyToolCompleted(event, data)
	case "step/tool-result":
		s.applyToolResults(event, data)
	case "context/compaction":
		s.applyCompaction(event, data)
		s.applyCompactionRecord(event, data)
	case "context/budget":
		s.applyBudget(event, data)
	case "model/usage":
		s.applyUsage(data)
	case "exec/started":
		s.inv.openTask(event, data)
	case "exec/ended":
		s.inv.settleTask(data, event.Sequence)
	case "collaboration/request":
		s.inv.openCollaboration(event, data)
	case "collaboration/delivered":
		s.inv.markCollaborationDelivered(data, event.Sequence)
	case "collaboration/result", "collaboration/cancel", "collaboration/close":
		s.inv.settleCollaboration(data, event.Sequence)
	}
}

// applyInput 折叠一条输入。环境/日期等上下文块没有用户也没有可见文本，
// 不进时间线；只带目标块的输入产生一行目标变更（目标写入本身不占一行用户消息，
// 这样一行只有一个服务端标识，不会出现两行同标识）。
func (s *foldState) applyInput(event protocol.RawEvent, data map[string]any) {
	change := s.goal.record(event, data)
	text := inputText(data)
	hasUser := data["user"] != nil
	if !hasUser && text == "" {
		if change == nil {
			return
		}
		row := s.ensureRow("goal:"+event.ID, event.ID, RowGoalChange, event)
		row.goal = change
		row.status = StatusComplete
		row.touch(event)
		return
	}
	row := s.ensureRow("input:"+event.ID, event.ID, RowUserMessage, event)
	row.text = text
	row.status = StatusPending
	// 插话进入当前运行中的一轮；排队输入等下一轮开始时归入那一轮。
	if mode := firstString(data, "mode"); mode == "steer" && s.current != nil {
		row.turnIndex = s.current.index
		row.turnID = s.current.id
	}
	row.touch(event)
	s.inv.openInput(event, data, text)
}

// applyInputApplied 记下"已应用但未消费"：输入仍在清单里，只是不可撤回。
func (s *foldState) applyInputApplied(event protocol.RawEvent, data map[string]any) {
	inputID := firstString(data, "input_id", "inputId")
	if inputID == "" {
		return
	}
	s.inv.markApplied(inputID, event.Sequence)
}

// applyInputConsumed 结算一条输入。
func (s *foldState) applyInputConsumed(event protocol.RawEvent, data map[string]any) {
	inputID := firstString(data, "input_id", "inputId")
	if inputID == "" {
		return
	}
	s.inv.settleInput(inputID, event.Sequence)
	if row, ok := s.index["input:"+inputID]; ok {
		row.status = StatusComplete
		row.touch(event)
	}
}

// applyInputDeleted 删除一条输入：行从视图消失，目标按剩下的事实重算。
// `input/withdraw`（用户撤回未消费的输入）与删除同一形状，按同一事实处理。
func (s *foldState) applyInputDeleted(event protocol.RawEvent, data map[string]any) {
	inputID := firstString(data, "input_id", "inputId")
	if inputID == "" {
		return
	}
	s.inv.settleInput(inputID, event.Sequence)
	s.goal.deleteInput(inputID)
	if row, ok := s.index["input:"+inputID]; ok {
		row.deleted = true
		row.touch(event)
	}
	if row, ok := s.index["goal:"+inputID]; ok {
		row.deleted = true
		row.touch(event)
	}
}

// applyTurnStart 开一轮。之前创建、尚未归属的用户输入归入这一轮：用户先说话，
// 服务端随后才开轮。服务端不会让两轮重叠，遇到重叠按已结束收敛上一轮。
func (s *foldState) applyTurnStart(event protocol.RawEvent, data map[string]any) {
	turnID := firstString(data, "turn_id", "turnId")
	if turnID == "" {
		return
	}
	if s.current != nil && s.current.status == TurnRunning && s.current.id != turnID {
		s.current.status = TurnEnded
		if event.CreateAt > s.current.endedAt {
			s.current.endedAt = event.CreateAt
		}
	}
	turn := s.ensureTurn(turnID, event)
	turn.status = TurnRunning
	turn.sequence = event.Sequence
	turn.startedAt = event.CreateAt
	s.current = turn
	for index := len(s.rows) - 1; index >= 0; index-- {
		row := s.rows[index]
		if row.deleted || row.kind != RowUserMessage {
			continue
		}
		if row.turnIndex != 0 {
			break
		}
		row.turnIndex = turn.index
		row.turnID = turn.id
	}
	s.inv.closeQuestions(event.Sequence)
}

// applyTurnEnd 结束一轮，并把它上面还在增长的块按结果状态收口。
func (s *foldState) applyTurnEnd(event protocol.RawEvent, data map[string]any) {
	turnID := firstString(data, "turn_id", "turnId")
	turn := s.current
	if turnID != "" {
		if known, ok := s.turnByID[turnID]; ok {
			turn = known
		}
	}
	if turn == nil {
		return
	}
	turn.status = TurnEnded
	turn.endedAt = event.CreateAt
	turn.endSequence = event.Sequence
	turn.outcome = firstString(data, "status")
	status := outcomeStatus(turn.outcome)
	s.closeAllBlocks(status, event)
	if s.current == turn {
		s.current = nil
	}
	s.inv.closeQuestions(event.Sequence)
}

// applyDelta 追加一段文本或思考。同一助手消息上连续的同种增量共用一个行；
// 换种类会关闭上一段，因此思考、文本、工具交替时会分成多行。
func (s *foldState) applyDelta(event protocol.RawEvent, data map[string]any, kind RowKind) {
	messageKey, ok := assistantKey(data)
	if !ok {
		// 缺轮次或请求标识的增量无法确定归属，按契约不建匿名行。
		return
	}
	turnIndex, turnID := s.turnFor(data, event)
	opened, closed := s.openText, s.openThinking
	if kind == RowThinking {
		opened, closed = s.openThinking, s.openText
	}
	if previous, ok := closed[messageKey]; ok {
		previous.status = StatusComplete
		previous.touch(event)
		delete(closed, messageKey)
	}
	row, ok := opened[messageKey]
	if !ok || row.status != StatusRunning {
		row = s.ensureRow("block:"+event.ID, event.ID, kind, event)
		row.status = StatusRunning
		row.turnIndex = turnIndex
		row.turnID = turnID
		opened[messageKey] = row
	}
	row.text += stringValue(data["delta"])
	row.touch(event)
	s.seenMessage[messageKey] = struct{}{}
}

// applyModelOutputEnd 收口一条助手消息：文本与思考块按结果状态结束。
func (s *foldState) applyModelOutputEnd(event protocol.RawEvent, data map[string]any) {
	messageKey, ok := assistantKey(data)
	if !ok {
		return
	}
	status := outcomeStatus(firstString(data, "status"))
	s.closeBlocksFor(messageKey, status, event)
	s.seenMessage[messageKey] = struct{}{}
}

// applyToolStart 收下一条只有调用标识与工具名的开始事实。
func (s *foldState) applyToolStart(event protocol.RawEvent, data map[string]any) {
	s.closeAssistantBlocks(data, event)
	callID := firstString(data, "call_id", "callId")
	if callID == "" {
		return
	}
	s.ensureTool(event, data, callID, firstString(data, "name"), nil)
}

// applyToolUse 收下一次完整的工具调用。
func (s *foldState) applyToolUse(event protocol.RawEvent, data map[string]any) {
	s.closeAssistantBlocks(data, event)
	call := object(data["call"])
	callID := firstString(call, "id", "call_id", "callId")
	if callID == "" {
		return
	}
	input, _ := call["input"].(map[string]any)
	s.ensureTool(event, data, callID, firstString(call, "name"), input)
	if isQuestionTool(firstString(call, "name")) {
		s.inv.openQuestion(event, firstString(data, "request_id", "requestId"), call)
	}
}

// applyToolCalls 收下一批工具调用。
func (s *foldState) applyToolCalls(event protocol.RawEvent, data map[string]any) {
	s.closeAssistantBlocks(data, event)
	requestID := firstString(data, "request_id", "requestId")
	for _, value := range array(data["calls"]) {
		call := object(value)
		callID := firstString(call, "id", "call_id", "callId")
		if callID == "" {
			continue
		}
		name := firstString(call, "name")
		input, _ := call["input"].(map[string]any)
		s.ensureTool(event, data, callID, name, input)
		if isQuestionTool(name) {
			s.inv.openQuestion(event, requestID, call)
		}
	}
}

// applyToolCompleted 收下一次工具调用的结果。
func (s *foldState) applyToolCompleted(event protocol.RawEvent, data map[string]any) {
	callID := firstString(data, "call_id", "callId")
	if callID == "" {
		return
	}
	result := object(data["result"])
	s.finishTool(callID, toolResult{
		sequence: event.Sequence,
		at:       event.CreateAt,
		text:     toolResultText(result),
		isError:  boolValue(result["isError"]),
	})
	requestID := firstString(data, "request_id", "requestId")
	s.inv.settleQuestion(requestID, callID, event.Sequence)
}

// applyToolResults 收下一批工具结果。
func (s *foldState) applyToolResults(event protocol.RawEvent, data map[string]any) {
	requestID := firstString(data, "request_id", "requestId")
	for _, value := range array(data["results"]) {
		result := object(value)
		callID := firstString(result, "call_id", "callId")
		if callID == "" {
			continue
		}
		payload := object(result["result"])
		s.finishTool(callID, toolResult{
			sequence: event.Sequence,
			at:       event.CreateAt,
			text:     toolResultText(payload),
			isError:  boolValue(payload["isError"]),
		})
		s.inv.settleQuestion(requestID, callID, event.Sequence)
	}
}

// applyCompaction 生成一条系统提示：上下文被压缩了。
func (s *foldState) applyCompaction(event protocol.RawEvent, data map[string]any) {
	row := s.ensureRow("system:"+event.ID, event.ID, RowSystem, event)
	row.text = compactionSummary(data)
	row.status = StatusComplete
	row.touch(event)
}

// ensureTool 取或建一次工具调用的行。结果先于调用到达时先暂存，调用到达后再合并，
// 不会因此多出一行。
func (s *foldState) ensureTool(
	event protocol.RawEvent,
	data map[string]any,
	callID, name string,
	input map[string]any,
) *rowState {
	if row, ok := s.tools[callID]; ok {
		if name != "" {
			row.tool.Name = name
		}
		if input != nil {
			row.tool.Input = marshalJSON(input)
		}
		row.touch(event)
		return row
	}
	turnIndex, turnID := s.turnFor(data, event)
	row := s.ensureRow("tool:"+callID, callID, RowTool, event)
	row.status = StatusRunning
	row.turnIndex = turnIndex
	row.turnID = turnID
	row.tool = &ToolCall{CallID: callID, Name: name, Input: marshalJSON(input)}
	s.tools[callID] = row
	for _, pending := range s.orphanResult[callID] {
		s.applyToolResult(row, pending)
	}
	delete(s.orphanResult, callID)
	return row
}

// finishTool 用一条结果事实结束工具行；行还没出现时先暂存。
func (s *foldState) finishTool(callID string, result toolResult) {
	row, ok := s.tools[callID]
	if !ok {
		s.orphanResult[callID] = append(s.orphanResult[callID], result)
		return
	}
	s.applyToolResult(row, result)
}

func (s *foldState) applyToolResult(row *rowState, result toolResult) {
	row.tool.Result = result.text
	row.tool.IsError = result.isError
	row.status = StatusComplete
	if result.isError {
		row.status = StatusError
	}
	if result.sequence > row.lastSeq {
		row.lastSeq = result.sequence
	}
	if result.at > row.updatedAt {
		row.updatedAt = result.at
	}
}

// closeAssistantBlocks 在工具调用开始时关闭该助手消息上还在增长的文本与思考。
func (s *foldState) closeAssistantBlocks(data map[string]any, event protocol.RawEvent) {
	messageKey, ok := assistantKey(data)
	if !ok {
		s.closeAllBlocks(StatusComplete, event)
		return
	}
	s.closeBlocksFor(messageKey, StatusComplete, event)
	s.seenMessage[messageKey] = struct{}{}
}

// closeBlocksFor 关闭一条助手消息上正在增长的块。
func (s *foldState) closeBlocksFor(messageKey string, status RowStatus, event protocol.RawEvent) {
	for _, opened := range []map[string]*rowState{s.openText, s.openThinking} {
		row, ok := opened[messageKey]
		if !ok {
			continue
		}
		row.status = status
		row.touch(event)
		delete(opened, messageKey)
	}
}

// closeAllBlocks 关闭本分支上所有正在增长的块。
func (s *foldState) closeAllBlocks(status RowStatus, event protocol.RawEvent) {
	for _, opened := range []map[string]*rowState{s.openText, s.openThinking} {
		for key, row := range opened {
			row.status = status
			row.touch(event)
			delete(opened, key)
		}
	}
}

// isQuestionTool 判断一次工具调用是不是向用户要一个答案。
func isQuestionTool(name string) bool {
	normalized := strings.ToLower(name)
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	return normalized == "askuserquestion" || normalized == "askuser"
}

// toolResultText 取工具结果里可显示的那部分文本。
func toolResultText(result map[string]any) string {
	if len(result) == 0 {
		return ""
	}
	switch content := result["content"].(type) {
	case string:
		return content
	case []any:
		var builder strings.Builder
		for _, value := range content {
			block := object(value)
			if stringValue(block["type"]) == "text" {
				builder.WriteString(stringValue(block["text"]))
			}
		}
		return builder.String()
	}
	return ""
}

// compactionSummary 取压缩事实里的摘要。老记录直接带 summary，新记录把摘要放在
// replacement 的 strategy 段里。
func compactionSummary(data map[string]any) string {
	if legacy := stringValue(data["summary"]); legacy != "" {
		return legacy
	}
	parts := object(data["replacement_parts"])
	strategy := len(array(data["replacement"]))
	if count := toInt64(parts["strategy"]); count > 0 && int(count) < strategy {
		strategy = int(count)
	}
	var builder strings.Builder
	for index, value := range array(data["replacement"]) {
		if index >= strategy {
			break
		}
		text := textBlocks(object(value)["content"])
		if text == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(text)
	}
	return strings.TrimSpace(builder.String())
}
