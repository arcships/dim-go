package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/projection"
)

// maxPendingPages 是每类未完成事项一次最多读几页，避免游标不前进时原地打转。
const maxPendingPages = 5

// seedPending 在首轮历史之后按固定截止点读四类未完成事项，装进投影与交互裁决。
//
// 四类各自独立：某一类读失败只记诊断并跳过，视图里的这一类就退回按历史推导的
// 那部分事实。读取成功时以服务端清单为准（覆盖对应截止点以上的推导结果）。
func (a *App) seedPending(ctx context.Context, runtime *sessionRuntime) {
	branchID := runtime.folder.Selected()
	if branchID == "" {
		branchID = mainBranchID
	}
	subscription := runtime.lease.Subscription()

	if through, inputs, err := fetchPendingPages(ctx, branchID,
		func(ctx context.Context, query protocol.PendingQuery) (protocol.PendingPage, []protocol.PendingInput, error) {
			result, err := subscription.PendingInputs(ctx, query)
			return result.PendingPage, result.Inputs, err
		}); err != nil {
		a.record("读取未消费输入失败，退回历史推导: %v", err)
	} else {
		items := make([]projection.PendingInput, 0, len(inputs))
		for _, input := range inputs {
			items = append(items, projection.PendingInput{
				InputID:  input.InputID,
				Sequence: input.Sequence,
				Mode:     input.Mode,
				Text:     pendingInputText(input.User),
				Applied:  input.Applied,
			})
		}
		runtime.folder.SeedPendingInputs(branchID, through, items)
	}

	if through, tasks, err := fetchPendingPages(ctx, branchID,
		func(ctx context.Context, query protocol.PendingQuery) (protocol.PendingPage, []protocol.PendingExecTask, error) {
			result, err := subscription.PendingExecTasks(ctx, query)
			return result.PendingPage, result.Tasks, err
		}); err != nil {
		a.record("读取未结束后台任务失败，退回历史推导: %v", err)
	} else {
		items := make([]projection.PendingTask, 0, len(tasks))
		for _, task := range tasks {
			items = append(items, projection.PendingTask{
				TaskID:      task.TaskID,
				EventID:     task.EventID,
				Sequence:    task.Sequence,
				Description: task.Description,
				StartedAt:   task.StartedAt,
			})
		}
		runtime.folder.SeedPendingTasks(branchID, through, items)
	}

	if through, batches, err := fetchPendingPages(ctx, branchID,
		func(ctx context.Context, query protocol.PendingQuery) (protocol.PendingPage, []protocol.PendingQuestionBatch, error) {
			result, err := subscription.PendingQuestions(ctx, query)
			return result.PendingPage, result.Batches, err
		}); err != nil {
		a.record("读取待答提问失败，退回历史推导: %v", err)
		a.seedQuestionsFromHistory(runtime, branchID)
	} else {
		a.seedQuestionsFromList(runtime, branchID, through, batches)
	}

	if through, requests, err := fetchPendingPages(ctx, branchID,
		func(ctx context.Context, query protocol.PendingQuery) (protocol.PendingPage, []protocol.PendingCollaboration, error) {
			result, err := subscription.PendingCollaboration(ctx, query)
			return result.PendingPage, result.Requests, err
		}); err != nil {
		a.record("读取未结束协作请求失败，退回历史推导: %v", err)
	} else {
		items := make([]projection.PendingCollaboration, 0, len(requests))
		for _, request := range requests {
			item := projection.PendingCollaboration{
				EventID:    request.EventID,
				Sequence:   request.Sequence,
				CallID:     request.Request.CallID,
				AgentID:    request.Request.Target.AgentID,
				TargetKind: request.Request.Target.Kind,
				SessionID:  request.Request.Target.SessionID,
				TurnID:     request.Request.TurnID,
				Prompt:     request.Request.Prompt,
				StartedAt:  request.StartedAt,
			}
			if request.Delivery != nil && request.Delivery.Status == protocol.CollaborationDeliveryOK {
				item.Delivered = true
			}
			items = append(items, item)
		}
		runtime.folder.SeedPendingCollaborations(branchID, through, items)
	}
}

// fetchPendingPages 读一页一页的未完成事项，直到没有下一页或到页数上限。
//
// 第一页不带截止点（服务端取当前头部），之后把第一页的 through 固定下来原样带回，
// `next` 也原样带回；两者都是 wire 不透明值，本包不解释。
func fetchPendingPages[T any](
	ctx context.Context,
	branchID string,
	fetch func(context.Context, protocol.PendingQuery) (protocol.PendingPage, []T, error),
) (int64, []T, error) {
	var (
		items   []T
		through int64
		after   json.RawMessage
	)
	for page := 0; page < maxPendingPages; page++ {
		query := protocol.PendingQuery{BranchID: branchID, After: after}
		if page > 0 {
			cutoff := through
			query.Through = &cutoff
		}
		pageResult, pageItems, err := fetch(ctx, query)
		if err != nil {
			return through, items, err
		}
		if pageResult.Through > 0 {
			through = pageResult.Through
		}
		items = append(items, pageItems...)
		if !pageResult.HasNext() {
			return through, items, nil
		}
		after = pageResult.Next
	}
	return through, items, fmt.Errorf("未完成事项超过 %d 页，结果不完整", maxPendingPages)
}

// pendingInputText 从待消费输入里取出可展示的文本：只拼接文本块，其余块不解释。
func pendingInputText(user *protocol.PendingInputUser) string {
	if user == nil {
		return ""
	}
	var builder strings.Builder
	for _, block := range user.Content {
		var parsed struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(block, &parsed); err != nil {
			continue
		}
		if parsed.Type != "" && parsed.Type != "text" {
			continue
		}
		builder.WriteString(parsed.Text)
	}
	return builder.String()
}

// seedQuestionsFromList 把清单一页里未结算的 ask_user 调用装进投影与交互裁决。
// 清单优先：交互裁决按这份截止点整体替换提问集合。
func (a *App) seedQuestionsFromList(
	runtime *sessionRuntime,
	branchID string,
	through int64,
	batches []protocol.PendingQuestionBatch,
) {
	viewItems := make([]projection.PendingQuestion, 0, len(batches))
	arbiterItems := make([]interaction.QuestionItem, 0, len(batches))
	for _, batch := range batches {
		for _, call := range batch.Calls {
			input := append(json.RawMessage(nil), call.Input...)
			viewItems = append(viewItems, projection.PendingQuestion{
				RequestID: batch.RequestID,
				CallID:    call.ID,
				EventID:   batch.EventID,
				Sequence:  batch.Sequence,
				ToolName:  call.Name,
				Input:     input,
			})
			arbiterItems = append(arbiterItems, interaction.QuestionItem{
				RequestID: batch.RequestID,
				CallID:    call.ID,
				EventID:   batch.EventID,
				Sequence:  batch.Sequence,
				ToolName:  call.Name,
				Input:     input,
			})
		}
	}
	runtime.folder.SeedPendingQuestions(branchID, through, viewItems)
	seed := interaction.QuestionSeed{
		BranchID: branchID,
		Through:  through,
		Source:   interaction.QuestionSourceList,
		Items:    arbiterItems,
	}
	for _, rejection := range runtime.arbiter.SeedQuestions(seed) {
		a.record("提问清单被丢弃: 请求=%s 原因=%s", rejection.RequestID, rejection.Reason)
	}
}

// seedQuestionsFromHistory 是清单读取失败时的退路：用已折叠的历史推导提问集合。
//
// 截止点取分支当前头部：之后到达的 ask_user 事实与工具结果才参与结算。
func (a *App) seedQuestionsFromHistory(runtime *sessionRuntime, branchID string) {
	view := runtime.folder.View()
	items := make([]interaction.QuestionItem, 0, len(view.Pending.Questions))
	for _, question := range view.Pending.Questions {
		items = append(items, interaction.QuestionItem{
			RequestID: question.RequestID,
			CallID:    question.CallID,
			EventID:   question.EventID,
			Sequence:  question.Sequence,
			ToolName:  question.ToolName,
			Input:     append(json.RawMessage(nil), question.Input...),
		})
	}
	seed := interaction.QuestionSeed{
		BranchID: branchID,
		Through:  runtime.folder.Head(branchID),
		Source:   interaction.QuestionSourceDerived,
		Items:    items,
	}
	for _, rejection := range runtime.arbiter.SeedQuestions(seed) {
		a.record("推导提问被丢弃: 请求=%s 原因=%s", rejection.RequestID, rejection.Reason)
	}
}
