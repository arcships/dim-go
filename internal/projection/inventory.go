package projection

import (
	"sort"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// seeded 是一份按固定截止点读到的清单。through 是这次读取覆盖到的序号：
// 序号不超过它的结算事实已经被这份清单反映过，不再改它。
type seeded[T any] struct {
	through int64
	items   []T
}

// seedSet 是四类未完成事项的截止点清单。
type seedSet struct {
	inputs    seeded[PendingInput]
	questions seeded[PendingQuestion]
	tasks     seeded[PendingTask]
	collabs   seeded[PendingCollaboration]
}

// applyTo 在重建折叠时先装清单，再按序号重放事件：装载与结算只有一处实现。
func (s seedSet) applyTo(state *foldState) {
	for _, item := range s.inputs.items {
		state.inv.seedInput(item, s.inputs.through)
	}
	for _, item := range s.questions.items {
		state.inv.seedQuestion(item, s.questions.through)
	}
	for _, item := range s.tasks.items {
		state.inv.seedTask(item, s.tasks.through)
	}
	for _, item := range s.collabs.items {
		state.inv.seedCollaboration(item, s.collabs.through)
	}
}

// inventory 是四类未完成事项的清单。它只由清单装载与事件结算驱动。
type inventory struct {
	inputs        map[string]*pendingInputEntry
	questions     map[string]*pendingQuestionEntry
	tasks         map[string]*pendingTaskEntry
	collabs       map[string]*pendingCollaborationEntry
	collabByEvent map[string]string
}

type pendingInputEntry struct {
	item  PendingInput
	floor int64
}

type pendingQuestionEntry struct {
	item  PendingQuestion
	floor int64
}

type pendingTaskEntry struct {
	item  PendingTask
	floor int64
}

type pendingCollaborationEntry struct {
	item  PendingCollaboration
	floor int64
}

func newInventory() *inventory {
	return &inventory{
		inputs:        make(map[string]*pendingInputEntry),
		questions:     make(map[string]*pendingQuestionEntry),
		tasks:         make(map[string]*pendingTaskEntry),
		collabs:       make(map[string]*pendingCollaborationEntry),
		collabByEvent: make(map[string]string),
	}
}

// questionKey 用请求标识加调用标识定位一批提问。
func questionKey(requestID, callID string) string {
	return requestID + "\x00" + callID
}

// ----- 装载 -----

func (i *inventory) seedInput(item PendingInput, through int64) {
	i.inputs[item.InputID] = &pendingInputEntry{item: item, floor: through}
}

func (i *inventory) seedQuestion(item PendingQuestion, through int64) {
	i.questions[questionKey(item.RequestID, item.CallID)] = &pendingQuestionEntry{item: item, floor: through}
}

func (i *inventory) seedTask(item PendingTask, through int64) {
	i.tasks[item.TaskID] = &pendingTaskEntry{item: item, floor: through}
}

func (i *inventory) seedCollaboration(item PendingCollaboration, through int64) {
	i.collabs[item.EventID] = &pendingCollaborationEntry{item: item, floor: through}
	i.collabByEvent[item.EventID] = item.EventID
}

// ----- 排队输入 -----

func (i *inventory) openInput(event protocol.RawEvent, data map[string]any, text string) {
	if entry, ok := i.inputs[event.ID]; ok && event.Sequence <= entry.floor {
		// 截止点清单已经反映过这条事实：清单是那一时刻的权威结果。
		return
	}
	i.inputs[event.ID] = &pendingInputEntry{item: PendingInput{
		InputID:  event.ID,
		Sequence: event.Sequence,
		Mode:     firstString(data, "mode"),
		Text:     text,
	}}
}

func (i *inventory) markApplied(inputID string, sequence int64) {
	entry, ok := i.inputs[inputID]
	if !ok || sequence <= entry.floor {
		return
	}
	entry.item.Applied = true
}

func (i *inventory) settleInput(inputID string, sequence int64) {
	entry, ok := i.inputs[inputID]
	if !ok || sequence <= entry.floor {
		return
	}
	delete(i.inputs, inputID)
}

// ----- 待答提问 -----

func (i *inventory) openQuestion(event protocol.RawEvent, requestID string, call map[string]any) {
	callID := firstString(call, "id", "call_id", "callId")
	if callID == "" {
		return
	}
	key := questionKey(requestID, callID)
	if entry, ok := i.questions[key]; ok && event.Sequence <= entry.floor {
		return
	}
	input, _ := call["input"].(map[string]any)
	raw := marshalJSON(cloneObject(input))
	i.questions[key] = &pendingQuestionEntry{item: PendingQuestion{
		RequestID: requestID,
		CallID:    callID,
		EventID:   event.ID,
		Sequence:  event.Sequence,
		ToolName:  firstString(call, "name"),
		Input:     raw,
	}}
}

func (i *inventory) settleQuestion(requestID, callID string, sequence int64) {
	key := questionKey(requestID, callID)
	entry, ok := i.questions[key]
	if !ok || sequence <= entry.floor {
		return
	}
	delete(i.questions, key)
}

// closeQuestions 在一轮开始或结束时清空上一轮的提问。截止点清单里的项只有被
// 更晚的事实覆盖才清除，不会被更早的历史回放抹掉。
func (i *inventory) closeQuestions(sequence int64) {
	for key, entry := range i.questions {
		if sequence > entry.floor {
			delete(i.questions, key)
		}
	}
}

// ----- 未结束后台任务 -----

func (i *inventory) openTask(event protocol.RawEvent, data map[string]any) {
	taskID := firstString(data, "task_id", "taskId")
	if taskID == "" {
		return
	}
	if entry, ok := i.tasks[taskID]; ok && event.Sequence <= entry.floor {
		return
	}
	i.tasks[taskID] = &pendingTaskEntry{item: PendingTask{
		TaskID:      taskID,
		EventID:     event.ID,
		Sequence:    event.Sequence,
		Description: firstString(data, "description"),
		StartedAt:   event.CreateAt,
	}}
}

// hasTask 报告任务是否还在未结束清单里。
func (i *inventory) hasTask(taskID string) bool {
	_, ok := i.tasks[taskID]
	return ok
}

// markTaskStopping 给未结束任务打上"正在停止"。它只影响展示；结算仍由结束事实负责。
func (i *inventory) markTaskStopping(taskID string) {
	if entry, ok := i.tasks[taskID]; ok {
		entry.item.Stopping = true
	}
}

func (i *inventory) settleTask(data map[string]any, sequence int64) {
	taskID := firstString(data, "task_id", "taskId")
	entry, ok := i.tasks[taskID]
	if !ok || sequence <= entry.floor {
		return
	}
	delete(i.tasks, taskID)
}

// ----- 协作请求 -----

func (i *inventory) openCollaboration(event protocol.RawEvent, data map[string]any) {
	callID := firstString(data, "call_id", "callId")
	if callID == "" {
		return
	}
	target := object(data["target"])
	item := PendingCollaboration{
		EventID:    event.ID,
		Sequence:   event.Sequence,
		CallID:     callID,
		AgentID:    firstString(target, "agent_id", "agentId"),
		TargetKind: firstString(target, "kind"),
		SessionID:  firstString(target, "session_id", "sessionId"),
		TurnID:     firstString(data, "turn_id", "turnId"),
		Prompt:     firstString(data, "prompt"),
		StartedAt:  event.CreateAt,
	}
	if entry, ok := i.collabs[item.EventID]; ok && event.Sequence <= entry.floor {
		return
	}
	i.collabs[item.EventID] = &pendingCollaborationEntry{item: item}
	i.collabByEvent[item.EventID] = item.EventID
}

// markCollaborationDelivered 记录成员已接入；请求仍未结算，仍留在清单里。
func (i *inventory) markCollaborationDelivered(data map[string]any, sequence int64) {
	if firstString(data, "status") != "ok" {
		return
	}
	eventID := firstString(data, "op_event_id", "opEventId")
	key, ok := i.collabByEvent[eventID]
	if !ok {
		return
	}
	entry, ok := i.collabs[key]
	if !ok {
		return
	}
	if sequence > entry.floor {
		entry.item.Delivered = true
	}
	if agent := firstString(data, "agent_id", "agentId"); agent != "" {
		entry.item.AgentID = agent
	}
	if session := firstString(data, "session_id", "sessionId"); session != "" {
		entry.item.SessionID = session
	}
}

// settleCollaboration 按请求事件、调用标识或成员会话结算请求。
func (i *inventory) settleCollaboration(data map[string]any, sequence int64) {
	eventID := firstString(data, "request_event_id", "requestEventId")
	if eventID != "" {
		i.settleCollabEntry(eventID, sequence)
	}
	callID := firstString(data, "call_id", "callId")
	sessionID := firstString(data, "session_id", "sessionId")
	for key, entry := range i.collabs {
		if callID != "" && entry.item.CallID == callID {
			if sequence > entry.floor {
				delete(i.collabs, key)
				delete(i.collabByEvent, key)
			}
			continue
		}
		if sessionID != "" && entry.item.SessionID == sessionID {
			if sequence > entry.floor {
				delete(i.collabs, key)
				delete(i.collabByEvent, key)
			}
		}
	}
}

func (i *inventory) settleCollabEntry(eventID string, sequence int64) {
	entry, ok := i.collabs[eventID]
	if !ok || sequence <= entry.floor {
		return
	}
	delete(i.collabs, eventID)
	delete(i.collabByEvent, eventID)
}

// snapshot 输出按序号排好的四类清单。
func (i *inventory) snapshot() PendingItems {
	out := PendingItems{}
	for _, entry := range i.inputs {
		out.Inputs = append(out.Inputs, entry.item)
	}
	for _, entry := range i.questions {
		item := entry.item
		item.Input = cloneRaw(item.Input)
		out.Questions = append(out.Questions, item)
	}
	for _, entry := range i.tasks {
		out.Tasks = append(out.Tasks, entry.item)
	}
	for _, entry := range i.collabs {
		out.Collaborations = append(out.Collaborations, entry.item)
	}
	sort.SliceStable(out.Inputs, func(left, right int) bool {
		if out.Inputs[left].Sequence != out.Inputs[right].Sequence {
			return out.Inputs[left].Sequence < out.Inputs[right].Sequence
		}
		return out.Inputs[left].InputID < out.Inputs[right].InputID
	})
	sort.SliceStable(out.Questions, func(left, right int) bool {
		if out.Questions[left].Sequence != out.Questions[right].Sequence {
			return out.Questions[left].Sequence < out.Questions[right].Sequence
		}
		return out.Questions[left].CallID < out.Questions[right].CallID
	})
	sort.SliceStable(out.Tasks, func(left, right int) bool {
		if out.Tasks[left].Sequence != out.Tasks[right].Sequence {
			return out.Tasks[left].Sequence < out.Tasks[right].Sequence
		}
		return out.Tasks[left].TaskID < out.Tasks[right].TaskID
	})
	sort.SliceStable(out.Collaborations, func(left, right int) bool {
		if out.Collaborations[left].Sequence != out.Collaborations[right].Sequence {
			return out.Collaborations[left].Sequence < out.Collaborations[right].Sequence
		}
		return out.Collaborations[left].EventID < out.Collaborations[right].EventID
	})
	return out
}
