package app

import (
	"context"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// 核对扫描的上限：一次核对最多读多少页、每页多少条事实。超限即"无法确认"，
// 不猜"没有发生"。
const (
	verifyPageLimit  = 200
	verifyPageBudget = 50
)

// appInspector 是写登记的核对入口：只读，不发送任何写请求。
//
// 判定顺序：
//  1. 会话未打开、或没有可读的历史订阅 → 无法确认（Confirmed=false）。
//  2. 投影的行里已有该标识 → 命中，归属取该行所在分支。
//  3. 否则按页读一遍历史：读完仍未出现 → 确认没有这次写入（可重提）；
//     读不完（出错或超过页数上限）→ 无法确认。
//
// 会话生命周期的新建没有目标会话：只按目录里是否出现该标识判断，不另造事实。
type appInspector struct {
	app *App
}

// LookupInput 实现 write.Inspector。
func (i *appInspector) LookupInput(
	ctx context.Context,
	query write.InputQuery,
) (write.InputState, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if query.InputID == "" {
		// 没有可核对的标识：结果未知只能收敛到需人工确认，不返回"未发生"。
		return write.InputState{}, nil
	}
	if query.SessionID == "" {
		return i.lookupLifecycle(query), nil
	}
	runtime, ok := i.app.session(query.SessionID)
	if !ok {
		return write.InputState{}, nil
	}
	running := false
	if runtime.attached.Running != nil {
		running = *runtime.attached.Running
	}
	state := write.InputState{SessionID: runtime.id, Running: running}

	// 提交类先问服务端：accepted/unknown 是确定答案，失败才退回只读推导。
	if query.Kind == write.KindSubmitInput && query.InputID != "" &&
		i.app.sessionSupportsMethod(ctx, runtime, protocol.MethodSessionInputLookup) {
		branchID := branchWireValue(query.BranchID)
		result, err := runtime.lease.Subscription().InputLookup(ctx, protocol.InputLookupParams{
			SessionID:    runtime.id,
			BranchID:     branchID,
			SubmissionID: query.InputID,
		})
		switch {
		case err != nil:
			i.app.record("提交核对失败，退回投影与历史扫描: %v", err)
		case result.Status == protocol.InputLookupAccepted:
			state.Confirmed = true
			state.Present = true
			state.BranchID = branchID
			return state, nil
		case result.Status == protocol.InputLookupUnknown:
			state.Confirmed = true
			state.Present = false
			state.BranchID = branchID
			return state, nil
		}
	}

	// 先看已折叠的行：命中不需要再读历史。
	for _, branch := range runtime.folder.Branches() {
		view, ok := runtime.folder.BranchView(branch.BranchID)
		if !ok {
			continue
		}
		for _, row := range view.Rows {
			if row.ID == query.InputID {
				state.Confirmed = true
				state.Present = true
				state.BranchID = branch.BranchID
				return state, nil
			}
		}
	}

	if runtime.lease == nil {
		return state, nil
	}
	present, complete := scanHistoryForInput(ctx, runtime.lease.Subscription(), query.InputID)
	if present {
		state.Confirmed = true
		state.Present = true
		return state, nil
	}
	if complete {
		state.Confirmed = true
		state.Present = false
	}
	return state, nil
}

// lookupLifecycle 处理没有目标会话的意图（会话新建）。
func (i *appInspector) lookupLifecycle(query write.InputQuery) write.InputState {
	catalogDomain := i.app.Catalog()
	if catalogDomain == nil {
		return write.InputState{}
	}
	if _, ok := catalogDomain.Entry(query.InputID); ok {
		return write.InputState{Confirmed: true, Present: true, SessionID: query.InputID}
	}
	// 目录没读过就无法确认"服务端没有这个会话"。
	if len(catalogDomain.Snapshot().Entries) == 0 {
		return write.InputState{}
	}
	return write.InputState{Confirmed: true, Present: false}
}

// scanHistoryForInput 按页读完历史，判断服务端历史里是否已有该输入标识。
// 第二个返回值表示读取是否完整（读完整才能把"没找到"当成确定结论）。
func scanHistoryForInput(
	ctx context.Context,
	subscription interface {
		History(context.Context, protocol.SessionHistoryParams) (protocol.SessionHistoryResult, error)
	},
	inputID string,
) (present bool, complete bool) {
	after := int64(0)
	limit := verifyPageLimit
	for page := 0; page < verifyPageBudget; page++ {
		if err := ctx.Err(); err != nil {
			return false, false
		}
		params := protocol.SessionHistoryParams{After: &after, Limit: &limit}
		result, err := subscription.History(ctx, params)
		if err != nil {
			return false, false
		}
		for _, event := range result.Events {
			if inputEventID(event) == inputID {
				return true, false
			}
		}
		if result.Done {
			return false, true
		}
		if result.NextAfter <= after {
			// 游标不前进：再读只会原地打转，按无法确认处理。
			return false, false
		}
		after = result.NextAfter
	}
	return false, false
}

// inputEventID 取一条输入事实的标识；非输入事实返回空。
func inputEventID(event protocol.RawEvent) string {
	if event.Type != "input" {
		return ""
	}
	switch event.Name {
	case "message", "applied", "consumed":
		return event.ID
	default:
		return ""
	}
}
