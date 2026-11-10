package interaction

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// Outcome 是一次应答的可观察结果。LongTerm 与 Response 是写登记返回的意图视图；
// 长期允许时先有 LongTerm，写入成功后才会有 Response。
type Outcome struct {
	Item     Item
	LongTerm *write.IntentView
	Response *write.IntentView
}

// Respond 对当前队首待决项作出决定，并经由写登记发出应答。
//
// 顺序约束：
//   - 只有队首可以应答（ErrNotCurrent）；同一请求同一时刻只允许一次应答
//     （ErrResponseInProgress）；已应答或已结算的请求拒绝再次应答
//     （ErrAlreadyAnswered）。
//   - 长期允许先写规则；规则没有明确成功时不登记应答，条目留在待决集合里
//     （ErrLongTermWriteFailed）。
//   - 应答写入明确失败时视为应答未发出：审批回到待呈现、提问回到待答，可以重试；
//     写入成功或无法确认后不再重发（已答的不重发）。
//
// 本方法等待写登记收敛到终态，调用方应给出合适的 ctx；ctx 结束时写登记可能仍在
// 在途，本包不会自行重发。
func (a *Arbiter) Respond(ctx context.Context, requestID string, response Response) (Outcome, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	a.mu.Lock()
	entry, ok := a.byID[requestID]
	if !ok {
		_, answered := a.answered[requestID]
		a.mu.Unlock()
		if answered {
			return Outcome{}, fmt.Errorf("%w: %s", ErrAlreadyAnswered, requestID)
		}
		return Outcome{}, fmt.Errorf("%w: %s", ErrUnknownPending, requestID)
	}
	if len(a.queue) == 0 || a.queue[0] != entry {
		a.mu.Unlock()
		return Outcome{}, fmt.Errorf("%w: %s", ErrNotCurrent, requestID)
	}
	if entry.state == StateAnswered ||
		(entry.state == StateAwaitingResponse && !entry.retryResponse) {
		a.mu.Unlock()
		return Outcome{}, fmt.Errorf("%w: %s", ErrAlreadyAnswered, requestID)
	}
	if _, busy := a.inProgress[requestID]; busy {
		a.mu.Unlock()
		return Outcome{}, fmt.Errorf("%w: %s", ErrResponseInProgress, requestID)
	}

	view := entry.item(0)
	if err := validateResponse(entry, response); err != nil {
		a.mu.Unlock()
		return Outcome{Item: view}, err
	}
	// 长期规则一旦写入，就只能重试同一决定，不能改口成拒绝。
	if entry.ruleWritten && response.Decision != entry.decision {
		a.mu.Unlock()
		return Outcome{Item: view}, fmt.Errorf(
			"%w: 长期规则已写入，只能重试 %s", ErrUnsupportedDecision, entry.decision)
	}

	fact := entry.fact
	ruleWritten := entry.ruleWritten
	a.inProgress[requestID] = struct{}{}
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.inProgress, requestID)
		a.mu.Unlock()
	}()

	outcome := Outcome{Item: view}

	// 第一步：长期允许必须先写规则。规则未明确成功就不发应答。
	if response.Decision == DecisionAllowLongTerm && !ruleWritten {
		a.advance(requestID, eventAllowLongTerm)

		key := a.nextAttemptKey(requestID, "long-term", response.Decision)
		ruleView, err := a.send(ctx, a.longTermIntent(fact, key))
		outcome.LongTerm = &ruleView
		if err != nil || ruleView.Status != write.StatusSucceeded {
			// A.md「待写长期 + 写入失败 → 待呈现」：不发应答，保留待决。
			a.advance(requestID, eventLongTermFailed)
			outcome.Item = a.itemOr(requestID, view)
			if err != nil {
				return outcome, fmt.Errorf("%w: %v", ErrLongTermWriteFailed, err)
			}
			return outcome, fmt.Errorf("%w: 写入状态 %s", ErrLongTermWriteFailed, ruleView.Status)
		}
		a.markRuleWritten(requestID)
		a.advance(requestID, eventLongTermWritten)
	}

	// 第二步：发出应答。参数在写之前发现的问题同样不发应答。
	params, err := responseParams(fact.Kind, response)
	if err != nil {
		a.markRetryResponse(requestID)
		outcome.Item = a.itemOr(requestID, view)
		return outcome, err
	}
	key := a.nextAttemptKey(requestID, "response", response.Decision)
	respView, err := a.send(ctx, a.responseIntent(fact, key, params))
	outcome.Response = &respView

	// 成功或需人工确认都按已发出处理：可能已经到达服务端，因此不重发。
	if acceptedStatus(respView.Status) {
		a.finishAccepted(requestID, response.Decision)
		outcome.Item = a.itemOr(requestID, view)
		if err != nil {
			return outcome, fmt.Errorf("%w: %v", ErrResponseUnsettled, err)
		}
		return outcome, nil
	}
	// 未收敛（等待被取消或写登记关闭）：可能已到达，按已答处理，不重发。
	if !settledWriteStatus(respView.Status) {
		a.finishAccepted(requestID, response.Decision)
		outcome.Item = a.itemOr(requestID, view)
		if err != nil {
			return outcome, fmt.Errorf("%w: %v", ErrResponseUnsettled, err)
		}
		return outcome, fmt.Errorf("%w: 写入状态 %s", ErrResponseUnsettled, respView.Status)
	}
	// 明确失败或未发生：应答没有发出，允许重试；长期规则已写入时只重试应答。
	a.markRetryResponse(requestID)
	outcome.Item = a.itemOr(requestID, view)
	if err != nil {
		return outcome, fmt.Errorf("%w: %v", ErrResponseWriteFailed, err)
	}
	return outcome, fmt.Errorf("%w: 写入状态 %s", ErrResponseWriteFailed, respView.Status)
}

// validateResponse 校验决定与待决类型是否匹配。只做结构校验。
func validateResponse(entry *entry, response Response) error {
	switch entry.fact.Kind {
	case PendingApproval:
		switch response.Decision {
		case DecisionApprove, DecisionReject:
			return nil
		case DecisionAllowLongTerm:
			if !entry.fact.AllowLongTerm || len(entry.fact.LongTermParams) == 0 {
				return fmt.Errorf("%w: %s 未提供可写入的长期规则参数",
					ErrLongTermUnavailable, entry.fact.RequestID)
			}
			return nil
		default:
			return fmt.Errorf("%w: 审批不接受决定 %q", ErrUnsupportedDecision, response.Decision)
		}
	case PendingQuestion:
		switch response.Decision {
		case DecisionAnswer:
			if len(response.Answer) == 0 || !json.Valid(response.Answer) {
				return fmt.Errorf("%w: %s", ErrAnswerRequired, entry.fact.RequestID)
			}
			return nil
		case DecisionReject, DecisionCancel:
			return nil
		default:
			return fmt.Errorf("%w: 提问不接受决定 %q", ErrUnsupportedDecision, response.Decision)
		}
	}
	return fmt.Errorf("%w: 未知待决类型 %q", ErrUnsupportedDecision, entry.fact.Kind)
}

// responseParams 生成写登记的透明参数。只表达决定与应答原文，wire 方法名由
// 发送方按 Kind 映射。
func responseParams(kind PendingKind, response Response) (json.RawMessage, error) {
	params := map[string]any{"decision": response.Decision.wire()}
	if kind == PendingQuestion && response.Decision == DecisionAnswer {
		if len(response.Answer) == 0 || !json.Valid(response.Answer) {
			return nil, ErrAnswerRequired
		}
		params["answer"] = cloneBytes(response.Answer)
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("编码应答参数: %w", err)
	}
	return raw, nil
}

// wire 把内部决定映射成透传给服务端的取值。
func (d Decision) wire() string {
	switch d {
	case DecisionApprove:
		return "approve"
	case DecisionReject:
		return "reject"
	case DecisionAllowLongTerm:
		// 长期规则写入成功后，应答本身只是一次普通允许。
		return "allow"
	case DecisionCancel:
		return "cancel"
	case DecisionAnswer:
		return "answer"
	}
	return string(d)
}

// longTermIntent 生成长期允许的写意图；规则参数原样透传。
func (a *Arbiter) longTermIntent(fact Fact, key string) write.Intent {
	return write.Intent{
		Key:       key,
		Kind:      write.KindLongTermAllow,
		SessionID: fact.SessionID,
		BranchID:  fact.BranchID,
		InputID:   fact.RequestID,
		Params:    cloneBytes(fact.LongTermParams),
	}
}

// responseIntent 生成审批应答或提问应答的写意图。
func (a *Arbiter) responseIntent(fact Fact, key string, params json.RawMessage) write.Intent {
	kind := write.KindApprovalResponse
	if fact.Kind == PendingQuestion {
		kind = write.KindQuestionAnswer
	}
	return write.Intent{
		Key:       key,
		Kind:      kind,
		SessionID: fact.SessionID,
		BranchID:  fact.BranchID,
		InputID:   fact.RequestID,
		Params:    params,
	}
}

// send 登记并等到写登记收敛到终态。登记后状态已是终态时不再等待。
func (a *Arbiter) send(ctx context.Context, intent write.Intent) (write.IntentView, error) {
	view, err := a.writer.Register(intent)
	if err != nil {
		return view, err
	}
	if settledWriteStatus(view.Status) {
		return view, nil
	}
	return a.writer.Wait(ctx, view.ID)
}

// settledWriteStatus 判断写意图是否已经到终态。与 D3 的终态集合一致。
func settledWriteStatus(status write.Status) bool {
	switch status {
	case write.StatusSucceeded, write.StatusFailed,
		write.StatusNotOccurred, write.StatusNeedsManualConfirmation:
		return true
	}
	return false
}

// acceptedStatus 判断应答是否应视为已发出：成功，或结果需人工确认（可能已到达，
// 因此不重发）。明确失败与未发生不算发出。
func acceptedStatus(status write.Status) bool {
	return status == write.StatusSucceeded || status == write.StatusNeedsManualConfirmation
}

// nextAttemptKey 给同一请求的每一步、每次尝试生成新的幂等键。只有上一次写入
// 明确失败后才会再次进入这里，因此新键不会掩盖重复发送。
func (a *Arbiter) nextAttemptKey(requestID, step string, decision Decision) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	slot := requestID + "\x00" + step + "\x00" + string(decision)
	a.attempts[slot]++
	return fmt.Sprintf("interaction/%s/%s/%s/%s/%d",
		step, a.sessionID, requestID, decision, a.attempts[slot])
}

// advance 按迁移表推进一步；非法迁移丢弃、状态不变。
func (a *Arbiter) advance(requestID string, event stateEvent) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, ok := a.byID[requestID]
	if !ok {
		return false
	}
	next, ok := nextState(entry.fact.Kind, entry.state, event)
	if !ok {
		return false
	}
	entry.state = next
	return true
}

// markRuleWritten 记录长期规则已写入；重试时不再重写。
func (a *Arbiter) markRuleWritten(requestID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rules[requestID] = struct{}{}
	if entry, ok := a.byID[requestID]; ok {
		entry.ruleWritten = true
		// 规则写入即视为长期允许决定已提交；应答重试只允许同一决定。
		entry.decision = DecisionAllowLongTerm
	}
}

// markRetryResponse 在长期规则已写入、应答却明确失败时，允许只重试应答。
func (a *Arbiter) markRetryResponse(requestID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, ok := a.byID[requestID]
	if !ok || !entry.ruleWritten || entry.state != StateAwaitingResponse {
		return
	}
	entry.retryResponse = true
}

// finishAccepted 在应答被写登记受理后记录决定并推进状态。
func (a *Arbiter) finishAccepted(requestID string, decision Decision) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.answered[requestID] = decision
	entry, ok := a.byID[requestID]
	if !ok {
		return
	}
	entry.decision = decision
	entry.retryResponse = false
	if entry.state == StatePresented {
		if next, ok := nextState(entry.fact.Kind, entry.state, eventDecide); ok {
			entry.state = next
		}
	}
}

// itemOr 取当前视图；条目已被并发结算时退回应答前的快照。
func (a *Arbiter) itemOr(requestID string, fallback Item) Item {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, ok := a.byID[requestID]
	if !ok {
		return fallback
	}
	return entry.item(a.positionLocked(entry))
}
