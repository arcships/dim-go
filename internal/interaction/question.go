package interaction

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidQuestionInput 表示 ask_user 的输入无法解析出问题。
var ErrInvalidQuestionInput = errors.New("ask_user 输入不合法")

// QuestionSource 是待答提问集合的来源。清单优先；清单读取失败时才退回历史推导。
type QuestionSource string

const (
	// QuestionSourceList 表示按固定截止点读到的 `session.question.pending` 清单。
	QuestionSourceList QuestionSource = "list"
	// QuestionSourceDerived 表示清单读取失败，按时间线里的 ask_user 调用与工具结果推导。
	QuestionSourceDerived QuestionSource = "derived"
)

// QuestionItem 是一条归一化后的 ask_user 调用。它来自清单的一个 call，或时间线里
// 一条 `ask_user` 工具事实。wire 形状由连接领域负责，本包只消费这个结构。
type QuestionItem struct {
	RequestID string
	CallID    string
	EventID   string
	Sequence  int64
	ToolName  string
	// Input 是 ask_user 的原始输入，本包从中解析问题与候选项。
	Input json.RawMessage
}

// QuestionSeed 是一次待答清单装载（或清单失败后的历史推导装载）。
type QuestionSeed struct {
	// BranchID 是清单读取的分支，必填。
	BranchID string
	// Through 是这次装载的固定截止点；之后的工具事实按序号结算。
	Through int64
	Source  QuestionSource
	// Items 按服务端清单顺序给出；同一请求标识的多个调用会合成一批提问。
	Items []QuestionItem
}

// QuestionSeedState 是当前提问来源的可观察状态。
type QuestionSeedState struct {
	BranchID string
	Through  int64
	Source   QuestionSource
}

// questionCall 是一个尚未结算的 ask_user 调用。
type questionCall struct {
	CallID    string
	EventID   string
	Sequence  int64
	ToolName  string
	Questions []Question
}

// SeedQuestions 用清单（或推导结果）整体替换待答提问集合，并记录截止点。
//
// 规则：
//   - 清单优先；清单读取失败时由调用方改用 QuestionSourceDerived 装载历史推导。
//   - 同一次装载整体替换本地提问集合，不累加、不与旧集合合并；审批集合不受影响。
//   - 装载后，序号不晚于截止点的工具事实不再改动提问集合（清单已经反映过它们），
//     只有更晚的 ask_user 调用与工具结果才生效。
func (a *Arbiter) SeedQuestions(seed QuestionSeed) []Rejection {
	a.mu.Lock()
	defer a.mu.Unlock()

	if seed.Source != QuestionSourceList && seed.Source != QuestionSourceDerived {
		return []Rejection{{Kind: PendingQuestion, Reason: RejectionInvalidFact}}
	}
	if seed.BranchID == "" || seed.Through < 0 {
		return []Rejection{{Kind: PendingQuestion, Reason: RejectionInvalidFact}}
	}

	var rejections []Rejection
	byRequest := make(map[string]*entry)
	var order []string
	seenCalls := make(map[string]bool)

	for _, item := range seed.Items {
		call, reason := normalizeQuestionItem(item)
		if reason != "" {
			rejections = append(rejections, Rejection{
				Kind:      PendingQuestion,
				RequestID: item.RequestID,
				Reason:    reason,
			})
			continue
		}
		key := questionCallKey(item.RequestID, item.CallID)
		if seenCalls[key] {
			rejections = append(rejections, Rejection{
				Kind:      PendingQuestion,
				RequestID: item.RequestID,
				Reason:    RejectionDuplicateRequest,
			})
			continue
		}
		// 审批与提问共用请求标识空间；已有审批占着这个标识时不接受同名提问。
		if existing, ok := a.byID[item.RequestID]; ok && existing.fact.Kind == PendingApproval {
			rejections = append(rejections, Rejection{
				Kind:      PendingQuestion,
				RequestID: item.RequestID,
				Reason:    RejectionDuplicateRequest,
			})
			continue
		}
		seenCalls[key] = true

		batch := byRequest[item.RequestID]
		if batch == nil {
			batch = &entry{
				fact: Fact{
					Kind:      PendingQuestion,
					RequestID: item.RequestID,
					SessionID: a.sessionID,
					BranchID:  seed.BranchID,
				},
				state:  StatePresented,
				source: seed.Source,
				calls:  make(map[string]*questionCall),
			}
			if decision, ok := a.answered[item.RequestID]; ok {
				batch.state = StateAnswered
				batch.decision = decision
			}
			byRequest[item.RequestID] = batch
			order = append(order, item.RequestID)
		}
		batch.addCall(call)
	}

	// 保留审批条目，移除旧提问条目，再把新提问按清单顺序接在后面。
	queue := make([]*entry, 0, len(a.queue)+len(order))
	byID := make(map[string]*entry, len(a.queue)+len(order))
	for _, old := range a.queue {
		if old.fact.Kind == PendingQuestion {
			continue
		}
		queue = append(queue, old)
		byID[old.fact.RequestID] = old
	}
	for _, requestID := range order {
		batch := byRequest[requestID]
		queue = append(queue, batch)
		byID[requestID] = batch
	}

	a.queue = queue
	a.byID = byID
	a.questionFloor = seed.Through
	a.questionBranch = seed.BranchID
	a.questionSource = seed.Source
	a.questionSeeded = true
	a.questionSettled = make(map[string]int64)
	return rejections
}

// ApplyAskUser 接收一条时间线上的 ask_user 调用事实。它只对序号晚于清单截止点的
// 调用生效；截止点及以下由清单负责，不重复装回。
func (a *Arbiter) ApplyAskUser(item QuestionItem) []Rejection {
	a.mu.Lock()
	defer a.mu.Unlock()

	call, reason := normalizeQuestionItem(item)
	if reason != "" {
		return []Rejection{{
			Kind:      PendingQuestion,
			RequestID: item.RequestID,
			Reason:    reason,
		}}
	}
	if a.questionSeeded && item.Sequence <= a.questionFloor {
		return []Rejection{{
			Kind:      PendingQuestion,
			RequestID: item.RequestID,
			Reason:    RejectionBelowCutoff,
		}}
	}
	if settled, ok := a.questionSettled[questionCallKey(item.RequestID, item.CallID)]; ok &&
		item.Sequence <= settled {
		return []Rejection{{
			Kind:      PendingQuestion,
			RequestID: item.RequestID,
			Reason:    RejectionAlreadySettled,
		}}
	}

	batch := a.byID[item.RequestID]
	if batch != nil && batch.fact.Kind != PendingQuestion {
		return []Rejection{{
			Kind:      PendingQuestion,
			RequestID: item.RequestID,
			Reason:    RejectionDuplicateRequest,
		}}
	}
	if batch == nil {
		batch = &entry{
			fact: Fact{
				Kind:      PendingQuestion,
				RequestID: item.RequestID,
				SessionID: a.sessionID,
				BranchID:  a.questionBranch,
			},
			state:  StatePresented,
			source: a.questionSource,
			calls:  make(map[string]*questionCall),
		}
		if decision, ok := a.answered[item.RequestID]; ok {
			batch.state = StateAnswered
			batch.decision = decision
		}
		a.queue = append(a.queue, batch)
		a.byID[item.RequestID] = batch
	}
	if _, exists := batch.calls[item.CallID]; exists {
		return []Rejection{{
			Kind:      PendingQuestion,
			RequestID: item.RequestID,
			Reason:    RejectionDuplicateRequest,
		}}
	}
	batch.addCall(call)
	return nil
}

// SettleQuestion 按工具结果/结算事实结算一条 ask_user 调用：移除该调用；批次里
// 没有剩余调用时，整条提问从队列移除并推进下一条。callID 为空表示结算整批。
//
// 序号不晚于清单截止点的结算不改动集合（清单已反映）；重复结算按无操作处理。
func (a *Arbiter) SettleQuestion(requestID, callID string, sequence int64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	batch := a.byID[requestID]
	if batch == nil || batch.fact.Kind != PendingQuestion {
		return false
	}
	// 截止点及以下（含无法定位的 0）由清单负责；没有序号就不能证明在截止点之后。
	if a.questionSeeded && sequence <= a.questionFloor {
		return false
	}

	if callID == "" {
		if len(batch.calls) == 0 {
			return false
		}
		for _, open := range batch.calls {
			a.markQuestionSettled(requestID, open.CallID, sequence)
		}
		a.removeEntryLocked(batch)
		return true
	}

	if _, ok := batch.calls[callID]; !ok {
		return false
	}
	a.markQuestionSettled(requestID, callID, sequence)
	batch.removeCall(callID)
	if len(batch.calls) == 0 {
		a.removeEntryLocked(batch)
	}
	return true
}

// QuestionSeedState 返回当前提问来源；从未装载过时第二个返回值为 false。
func (a *Arbiter) QuestionSeedState() (QuestionSeedState, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.questionSeeded {
		return QuestionSeedState{}, false
	}
	return QuestionSeedState{
		BranchID: a.questionBranch,
		Through:  a.questionFloor,
		Source:   a.questionSource,
	}, true
}

// markQuestionSettled 记录已结算的调用序号，防止旧事实把它装回来；调用方持锁。
func (a *Arbiter) markQuestionSettled(requestID, callID string, sequence int64) {
	key := questionCallKey(requestID, callID)
	if previous, ok := a.questionSettled[key]; !ok || sequence > previous {
		a.questionSettled[key] = sequence
	}
}

// removeEntryLocked 从队列与索引移除一条条目；调用方持锁。
func (a *Arbiter) removeEntryLocked(target *entry) {
	delete(a.byID, target.fact.RequestID)
	for i, candidate := range a.queue {
		if candidate == target {
			a.queue = append(a.queue[:i], a.queue[i+1:]...)
			return
		}
	}
}

// addCall 把一个尚未结算的调用加入批次；调用方持锁。
func (e *entry) addCall(call *questionCall) {
	if e.calls == nil {
		e.calls = make(map[string]*questionCall)
	}
	e.calls[call.CallID] = call
	e.callOrder = append(e.callOrder, call.CallID)
}

// removeCall 移除一个已结算的调用；调用方持锁。
func (e *entry) removeCall(callID string) {
	delete(e.calls, callID)
	for i, candidate := range e.callOrder {
		if candidate == callID {
			e.callOrder = append(e.callOrder[:i], e.callOrder[i+1:]...)
			return
		}
	}
}

// normalizeQuestionItem 校验一条调用并解析问题；只做结构校验。
func normalizeQuestionItem(item QuestionItem) (*questionCall, string) {
	if item.RequestID == "" || item.CallID == "" {
		return nil, RejectionInvalidFact
	}
	if !isAskUserTool(item.ToolName) {
		return nil, RejectionInvalidFact
	}
	questions, err := parseQuestionInput(item.Input)
	if err != nil {
		return nil, RejectionInvalidFact
	}
	return &questionCall{
		CallID:    item.CallID,
		EventID:   item.EventID,
		Sequence:  item.Sequence,
		ToolName:  item.ToolName,
		Questions: questions,
	}, ""
}

// questionCallKey 用请求标识加调用标识定位一次 ask_user 调用。
func questionCallKey(requestID, callID string) string {
	return requestID + "\x00" + callID
}

// isAskUserTool 判断工具名是否是提问工具；大小写、下划线与连字符不敏感。
func isAskUserTool(name string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(name))
	return normalized == "askuser" || normalized == "askuserquestion"
}

// parseQuestionInput 从 ask_user 的 input 里读出问题与候选项。只做结构解析，
// 不解释问题语义。
func parseQuestionInput(raw json.RawMessage) ([]Question, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: ask_user 缺少输入", ErrInvalidQuestionInput)
	}
	var payload struct {
		Questions []struct {
			ID          string            `json:"id"`
			Question    string            `json:"question"`
			Text        string            `json:"text"`
			Prompt      string            `json:"prompt"`
			Options     []json.RawMessage `json:"options"`
			Multiple    bool              `json:"multiSelect"`
			MultipleAlt bool              `json:"multiple"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuestionInput, err)
	}
	if len(payload.Questions) == 0 {
		return nil, fmt.Errorf("%w: questions 为空", ErrInvalidQuestionInput)
	}
	questions := make([]Question, 0, len(payload.Questions))
	for _, item := range payload.Questions {
		text := item.Question
		if text == "" {
			text = item.Text
		}
		if text == "" {
			text = item.Prompt
		}
		options := make([]string, 0, len(item.Options))
		for _, option := range item.Options {
			if text := optionText(option); text != "" {
				options = append(options, text)
			}
		}
		questions = append(questions, Question{
			ID:       item.ID,
			Text:     text,
			Options:  options,
			Multiple: item.Multiple || item.MultipleAlt,
		})
	}
	return questions, nil
}

// optionText 取候选项的展示文本；兼容字符串与带 label/value/text 的对象。
func optionText(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var object struct {
		Label string `json:"label"`
		Value string `json:"value"`
		Text  string `json:"text"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return ""
	}
	switch {
	case object.Label != "":
		return object.Label
	case object.Value != "":
		return object.Value
	default:
		return object.Text
	}
}
