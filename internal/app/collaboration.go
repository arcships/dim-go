package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// 成员停止的真实方法（alpha.21 实测，见 D1 §549-558）。
const methodCollaborationCancel = "session.collaboration.cancel"

// MemberStopRequest 是停止一个协作成员会话的请求。
type MemberStopRequest struct {
	SessionID string
	BranchID  string
	// MemberSessionID 是成员会话标识，来自协作事实，不来自 agent 清单。
	MemberSessionID string
}

// CollaborationMembers 读本会话当前可停止的成员会话标识。
//
// 来源：`session.collaboration.pending` 的 `delivery.session_id`（status=ok）与历史
// 事实 `collaboration/delivered` 的 `delivery.session_id`；`agent.list` 只有计数，
// 不是成员标识。只读、去重、保持出现顺序。
func (a *App) CollaborationMembers(ctx context.Context, sessionID string) ([]string, error) {
	runtime, err := a.openRuntime(sessionID, "session.collaboration.pending")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	members := make([]string, 0, 4)
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		members = append(members, id)
	}

	// 1) 未结束的协作请求：delivery 给出成员会话。
	branchID := branchOrMain(runtime.folder.Selected())
	after := int64(0)
	through := int64(0)
	for page := 0; page < 5; page++ {
		raw, callErr := runtime.lease.Subscription().Call(ctx, "session.collaboration.pending", map[string]any{
			"sessionId": sessionID,
			"branchId":  branchID,
			"after":     after,
		})
		if callErr != nil {
			return nil, callErr
		}
		result, parseErr := protocol.ParseSessionCollaborationPendingResult(raw)
		if parseErr != nil {
			return nil, fmt.Errorf("协作清单回执无法解析: %w", parseErr)
		}
		for _, request := range result.Requests {
			if request.Delivery != nil && request.Delivery.Status == protocol.CollaborationDeliveryOK {
				add(request.Delivery.SessionID)
			}
		}
		through = result.Through
		var next string
		if len(result.Next) > 0 && string(result.Next) != "null" {
			_ = json.Unmarshal(result.Next, &next)
		}
		if next == "" || next == fmt.Sprint(after) {
			break
		}
		after = result.Through
	}
	_ = through

	// 2) 历史事实：delivered 的 delivery.session_id。
	events, err := (&changeSource{app: a}).collectFacts(ctx, runtime)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.Type != "collaboration" || event.Name != "delivered" {
			continue
		}
		var data struct {
			Delivery struct {
				Status    string `json:"status"`
				SessionID string `json:"session_id"`
			} `json:"delivery"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			continue
		}
		if data.Delivery.Status == "" || data.Delivery.Status == protocol.CollaborationDeliveryOK {
			add(data.Delivery.SessionID)
		}
	}
	return members, nil
}

// StopTeamMember 停止一个协作成员：登记一次 KindManagementWrite，发送方转到
// `session.collaboration.cancel`。服务端回 `idle` 表示成员已不在运行。
func (a *App) StopTeamMember(
	ctx context.Context,
	request MemberStopRequest,
) (write.IntentView, error) {
	if strings.TrimSpace(request.SessionID) == "" {
		return write.IntentView{}, fmt.Errorf("%w: 缺少会话标识", ErrInvalidConfig)
	}
	if strings.TrimSpace(request.MemberSessionID) == "" {
		return write.IntentView{}, fmt.Errorf("%w: 缺少成员会话标识", write.ErrInvalidIntent)
	}
	if err := a.Writable(); err != nil {
		return write.IntentView{}, err
	}
	runtime, err := a.openRuntime(request.SessionID, methodCollaborationCancel)
	if err != nil {
		return write.IntentView{}, err
	}
	// 能力门禁放在入口：缺方法时零请求、不登记。
	if !a.sessionSupportsMethod(ctx, runtime, methodCollaborationCancel) {
		return write.IntentView{}, capabilityMissing(methodCollaborationCancel)
	}
	writes := a.Writes()
	if writes == nil {
		return write.IntentView{}, ErrNotConnected
	}
	branchID := branchOrMain(firstNonEmpty(request.BranchID, runtime.folder.Selected()))
	params, err := json.Marshal(map[string]any{
		"op":              "session.collaboration.cancel",
		"sessionId":       request.SessionID,
		"branchId":        branchID,
		"memberSessionId": request.MemberSessionID,
	})
	if err != nil {
		return write.IntentView{}, fmt.Errorf("编码成员停止参数: %w", err)
	}
	return writes.Register(write.Intent{
		Key:       "collaboration.cancel:" + request.SessionID + ":" + request.MemberSessionID,
		Kind:      write.KindManagementWrite,
		SessionID: request.SessionID,
		BranchID:  branchID,
		Params:    params,
	})
}

// MemberStopStatus 解析成员停止回执：`cancelled` 表示已停，`idle` 表示本来就没在运行。
func MemberStopStatus(raw json.RawMessage) (status string, stopped bool, err error) {
	if len(raw) == 0 {
		return "", false, nil
	}
	var reply struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return "", false, fmt.Errorf("成员停止回执无法解析: %w", err)
	}
	switch reply.Status {
	case "cancelled", "idle":
		return reply.Status, true, nil
	default:
		return reply.Status, false, nil
	}
}
