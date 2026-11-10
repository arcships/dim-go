package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// waitView 轮询视图直到条件成立；超时即失败。
func waitView(t *testing.T, base *App, sessionID string, what string, ok func(SessionView) bool) SessionView {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		view, found := base.View(sessionID)
		if found && ok(view) {
			return view
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待 %s 超时，当前 %+v", what, view)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestApprovalNoticesRouteToArbiter 核对审批三类通知：请求入队、结算与取消出队。
func TestApprovalNoticesRouteToArbiter(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})

	approval := map[string]any{
		"sessionId": testSession,
		"branchId":  "main",
		"requestId": "ap-1",
		"source": map[string]any{
			"kind": "tool", "turn_id": "t1", "request_id": "r1", "call_id": "c1",
		},
		"call":   map[string]any{"name": "read_file", "input": map[string]any{"path": "/work/a.go"}},
		"reason": "读取文件",
	}
	if err := server.Notify("approval.request", approval); err != nil {
		t.Fatalf("推送审批请求: %v", err)
	}
	view := waitView(t, result.base, testSession, "审批入队", func(view SessionView) bool {
		return view.HasCurrent
	})
	if view.Current.RequestID != "ap-1" || view.Current.SessionID != testSession || view.Current.BranchID != "main" {
		t.Fatalf("审批事实归属不对: %+v", view.Current)
	}
	if view.Current.Action != "read_file：读取文件" {
		t.Fatalf("审批动作描述不对: %q", view.Current.Action)
	}
	if !view.Current.AllowLongTerm {
		t.Fatal("单字段绝对路径的读操作应可长期允许")
	}
	if view.PendingCounts.Approvals != 1 {
		t.Fatalf("待决审批条数应为 1，实际 %d", view.PendingCounts.Approvals)
	}

	if err := server.Notify("approval.resolved", map[string]any{
		"sessionId": testSession, "branchId": "main", "requestId": "ap-1", "decision": "allow",
	}); err != nil {
		t.Fatalf("推送审批结算: %v", err)
	}
	waitView(t, result.base, testSession, "审批结算", func(view SessionView) bool {
		return !view.HasCurrent && view.PendingCounts.Approvals == 0
	})

	// 取消：调用没有开始，同样移出队列。
	if err := server.Notify("approval.request", map[string]any{
		"sessionId": testSession, "branchId": "main", "requestId": "ap-2",
		"source": map[string]any{"kind": "host", "operation_id": "op-1"},
		"call":   map[string]any{"name": "workspace_switch", "input": map[string]any{"dir": "/work"}},
	}); err != nil {
		t.Fatalf("推送第二条审批: %v", err)
	}
	view = waitView(t, result.base, testSession, "第二条审批入队", func(view SessionView) bool {
		return view.HasCurrent
	})
	if view.Current.AllowLongTerm {
		t.Fatal("host 来源或多字段输入不应可长期允许")
	}
	if err := server.Notify("approval.cancelled", map[string]any{
		"sessionId": testSession, "branchId": "main", "requestId": "ap-2", "reason": "调用取消",
	}); err != nil {
		t.Fatalf("推送审批取消: %v", err)
	}
	waitView(t, result.base, testSession, "审批取消", func(view SessionView) bool {
		return !view.HasCurrent
	})
}

// TestTitleNoticeMergesIntoCatalog 核对标题通知按修订号合并到目录条目。
func TestTitleNoticeMergesIntoCatalog(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{{
			Method: "session.catalog",
			Result: map[string]any{"sessions": []any{map[string]any{
				"sessionId":     testSession,
				"title":         "旧标题",
				"titleRevision": 1,
			}}},
		}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	if _, err := result.base.Catalog().Refresh(context.Background(), catalog.Query{}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}
	if entry, _ := result.base.Catalog().Entry(testSession); entry.Title != "旧标题" {
		t.Fatalf("目录初始标题不对: %+v", entry)
	}

	if err := server.Notify("session.notice", map[string]any{
		"sessionId": testSession,
		"source":    "SessionStore",
		"kind":      "title",
		"data":      map[string]any{"title": "自动标题", "titleSource": "auto", "titleRevision": 2},
	}); err != nil {
		t.Fatalf("推送标题通知: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		entry, _ := result.base.Catalog().Entry(testSession)
		if entry.Title == "自动标题" {
			if entry.TitleSource != catalog.TitleSourceAuto || entry.TitleRevision != 2 {
				t.Fatalf("标题合并字段不对: %+v", entry)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("标题通知没有合并进目录: %+v", entry)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// 修订号更低的标题不覆盖。
	if err := server.Notify("session.notice", map[string]any{
		"sessionId": testSession,
		"source":    "SessionStore",
		"kind":      "title",
		"data":      map[string]any{"title": "旧标题", "titleSource": "auto", "titleRevision": 1},
	}); err != nil {
		t.Fatalf("推送过期标题: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if entry, _ := result.base.Catalog().Entry(testSession); entry.Title != "自动标题" {
		t.Fatalf("低修订号标题不应覆盖: %+v", entry)
	}
}

// TestPendingSeedingOnAttach 核对四类未完成事项在打开会话时按截止点注入。
func TestPendingSeedingOnAttach(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.input.pending", Result: map[string]any{
				"branchId": "main", "through": 5, "next": nil,
				"inputs": []any{map[string]any{
					"inputId": "i-1", "sequence": 5, "mode": "queue",
					"user":    map[string]any{"content": []any{map[string]any{"type": "text", "text": "排队的话"}}},
					"applied": false,
				}},
			}},
			{Method: "session.exec.pending", Result: map[string]any{
				"branchId": "main", "through": 5, "next": nil,
				"tasks": []any{map[string]any{
					"eventId": "e-9", "sequence": 9, "startedAt": 1700000000,
					"taskId": "k1", "description": "go test ./...",
				}},
			}},
			{Method: "session.question.pending", Result: map[string]any{
				"branchId": "main", "through": 6, "next": nil,
				"batches": []any{map[string]any{
					"eventId": "e-10", "sequence": 10, "requestId": "q-1",
					"calls": []any{map[string]any{
						"id": "c1", "name": "ask_user",
						"input": map[string]any{"questions": []any{
							map[string]any{"question": "继续吗", "options": []any{"继续", "停止"}},
						}},
					}},
				}},
			}},
			{Method: "session.collaboration.pending", Result: map[string]any{
				"branchId": "main", "through": 7, "next": nil,
				"requests": []any{map[string]any{
					"eventId": "e-11", "sequence": 11, "startedAt": 1700000001,
					"request": map[string]any{
						"turn_id": "t1", "request_id": "r1", "call_id": "cc1",
						"target": map[string]any{"kind": "agent", "agent_id": "reviewer"},
						"prompt": "看一下",
					},
				}},
			}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})

	view, _ := result.base.View(testSession)
	// 四类清单都注入了；历史里推出来的未消费输入（e1）仍按事件保留，二者不冲突。
	if !hasPendingInput(view, "i-1", "排队的话") {
		t.Fatalf("排队输入映射不对: %+v", view.Pending.Inputs)
	}
	if len(view.Pending.Tasks) != 1 || view.Pending.Tasks[0].Description != "go test ./..." {
		t.Fatalf("后台任务映射不对: %+v", view.Pending.Tasks)
	}
	if len(view.Pending.Questions) != 1 || view.Pending.Questions[0].ToolName != "ask_user" {
		t.Fatalf("提问清单映射不对: %+v", view.Pending.Questions)
	}
	if len(view.Pending.Collaborations) != 1 || view.Pending.Collaborations[0].AgentID != "reviewer" {
		t.Fatalf("协作请求映射不对: %+v", view.Pending.Collaborations)
	}
	// 提问同时进了交互裁决队列。
	if !view.HasCurrent || view.Current.RequestID != "q-1" {
		t.Fatalf("待答提问没有进交互裁决: %+v", view.Current)
	}
	if len(view.Current.Questions) != 1 || view.Current.Questions[0].Text != "继续吗" ||
		len(view.Current.Questions[0].Options) != 2 {
		t.Fatalf("提问内容映射不对: %+v", view.Current.Questions)
	}

	// 清单一类读失败时退回历史推导，其余类照常注入：这里只给 input.pending 回执。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.input.pending", Result: map[string]any{
				"branchId": "main", "through": 5, "next": nil,
				"inputs": []any{map[string]any{"inputId": "i-2", "sequence": 5, "mode": "queue", "applied": false}},
			}},
		},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	view2, _ := result2.base.View(testSession)
	if !hasPendingInput(view2, "i-2", "") {
		t.Fatalf("可读的一类应注入: %+v", view2.Pending)
	}
	if len(view2.Pending.Tasks) != 0 || len(view2.Pending.Collaborations) != 0 {
		t.Fatalf("读失败的类应保持历史推导结果: %+v", view2.Pending)
	}
}

// hasPendingInput 判断排队输入清单里是否有指定标识（文本非空时一并核对）。
func hasPendingInput(view SessionView, inputID, text string) bool {
	for _, input := range view.Pending.Inputs {
		if input.InputID == inputID && (text == "" || input.Text == text) {
			return true
		}
	}
	return false
}

// TestInspectorPrefersInputLookup 核对核对优先用 session.input_lookup，失败退回只读推导。
func TestInspectorPrefersInputLookup(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              testSession,
			LatestTurnStart: 1,
			Events:          testEvents(),
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.methods", Result: map[string]any{"methods": []any{
				map[string]any{"name": "session.input_lookup"},
			}}},
			{Method: "session.input_lookup", Result: map[string]any{
				"status": "accepted",
				"event":  map[string]any{"id": "srv-input-1", "sequence": 3, "type": "input", "name": "message"},
			}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	inspector := &appInspector{app: result.base}

	hit, err := inspector.LookupInput(context.Background(), write.InputQuery{
		Kind: write.KindSubmitInput, SessionID: testSession, InputID: "srv-input-1",
	})
	if err != nil || !hit.Confirmed || !hit.Present {
		t.Fatalf("accepted 应命中: %+v err=%v", hit, err)
	}
	requests := server.RequestsFor("session.input_lookup")
	if len(requests) != 1 {
		t.Fatalf("应只发一次核对请求，实际 %d", len(requests))
	}
	params := requestParams(t, requests[0])
	if params["sessionId"] != testSession || params["branchId"] != "main" || params["submissionId"] != "srv-input-1" {
		t.Fatalf("核对参数不对: %v", params)
	}

	// 服务端说没有这次写入。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.methods", Result: map[string]any{"methods": []any{
				map[string]any{"name": "session.input_lookup"},
			}}},
			{Method: "session.input_lookup", Result: map[string]any{"status": "unknown"}},
		},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	miss, err := (&appInspector{app: result2.base}).LookupInput(context.Background(), write.InputQuery{
		Kind: write.KindSubmitInput, SessionID: testSession, InputID: "never-sent",
	})
	if err != nil || !miss.Confirmed || miss.Present {
		t.Fatalf("unknown 应确认未发生: %+v err=%v", miss, err)
	}

	// 方法缺失：退回投影与历史扫描，已落库的输入仍然命中。
	server3 := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              testSession,
			LatestTurnStart: 1,
			Events:          testEvents(),
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.methods", Result: map[string]any{"methods": []any{
				map[string]any{"name": "session.input.pending"},
			}}},
		},
	})
	result3 := startAssembly(t, server3, assemblyOptions{session: testSession})
	fallback, err := (&appInspector{app: result3.base}).LookupInput(context.Background(), write.InputQuery{
		Kind: write.KindSubmitInput, SessionID: testSession, InputID: "e1",
	})
	if err != nil || !fallback.Confirmed || !fallback.Present {
		t.Fatalf("退回推导后应命中历史里的输入: %+v err=%v", fallback, err)
	}
	if got := len(server3.RequestsFor("session.input_lookup")); got != 0 {
		t.Fatalf("能力缺失时不应发核对请求，实际 %d", got)
	}
}

// TestAttachmentWriteUsesPutAndChecksMethods 核对附件写入：走 attachment.put，
// 缺方法时报能力缺失且不发请求。
func TestAttachmentWriteUsesPutAndChecksMethods(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.methods", Result: map[string]any{"methods": []any{
				map[string]any{"name": "attachment.put", "mutating": true},
			}}},
			{Method: "attachment.put", Result: map[string]any{
				"attachmentId": "att-1", "path": "/files/att-1/paste.txt", "bytes": 15,
			}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	writes := result.base.Writes()
	ctx := context.Background()

	registered, err := writes.Register(write.Intent{
		Key: "k-attachment", Kind: write.KindAttachmentWrite, SessionID: testSession,
		Params: json.RawMessage(`{"name":"paste.txt","text":"一段要存成文件的内容"}`),
	})
	if err != nil {
		t.Fatalf("登记附件写入: %v", err)
	}
	settled, err := writes.Wait(ctx, registered.ID)
	if err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("附件写入应成功: %v %+v", err, settled)
	}
	requests := server.RequestsFor("attachment.put")
	if len(requests) != 1 {
		t.Fatalf("应只发一次 attachment.put，实际 %d", len(requests))
	}
	params := requestParams(t, requests[0])
	if params["sessionId"] != testSession || params["name"] != "paste.txt" || params["text"] != "一段要存成文件的内容" {
		t.Fatalf("附件写入参数不对: %v", params)
	}

	// 缺方法：不发请求，按明确失败收敛并说明能力缺失。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.methods", Result: map[string]any{"methods": []any{
				map[string]any{"name": "session.input.pending"},
			}}},
		},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	registered2, err := result2.base.Writes().Register(write.Intent{
		Key: "k-attachment-missing", Kind: write.KindAttachmentWrite, SessionID: testSession,
		Params: json.RawMessage(`{"text":"不该发出"}`),
	})
	if err != nil {
		t.Fatalf("登记附件写入: %v", err)
	}
	settled2, err := result2.base.Writes().Wait(ctx, registered2.ID)
	if err != nil {
		t.Fatalf("等待收敛: %v", err)
	}
	if settled2.Status != write.StatusFailed {
		t.Fatalf("缺方法应明确失败，实际 %s（%s）", settled2.Status, settled2.Reason)
	}
	if !strings.Contains(settled2.Reason, "能力") {
		t.Fatalf("失败原因应说明能力缺失: %q", settled2.Reason)
	}
	if got := len(server2.RequestsFor("attachment.put")); got != 0 {
		t.Fatalf("能力缺失时不应发 attachment.put，实际 %d", got)
	}
}

// TestAskUserTimelineRoutesToArbiter 核对时间线里的 ask_user 调用与工具结果：
// 截止点之后的新提问进队列，工具结果把它结算掉。
func TestAskUserTimelineRoutesToArbiter(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			// 清单为空：确认走下界之后的工具事实通道。
			{Method: "session.question.pending", Result: map[string]any{
				"branchId": "main", "through": 5, "next": nil, "batches": []any{},
			}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})

	ask := rawEvent("e20", 20, "step", "tool-call",
		`{"turn_id":"t2","request_id":"r2","calls":[{"id":"q1","name":"ask_user","input":{"questions":[{"question":"继续吗","options":["继续","停止"]}]}}]}`)
	if err := server.PushEvent(testSession, ask); err != nil {
		t.Fatalf("推送 ask_user 事实: %v", err)
	}
	view := waitView(t, result.base, testSession, "提问入队", func(view SessionView) bool {
		return view.HasCurrent && view.Current.RequestID == "r2"
	})
	if len(view.Current.Questions) != 1 || view.Current.Questions[0].Text != "继续吗" {
		t.Fatalf("提问内容没有从 ask_user 输入里解出来: %+v", view.Current.Questions)
	}

	answer := rawEvent("e21", 21, "step", "tool-completed",
		`{"turn_id":"t2","request_id":"r2","call_id":"q1","result":{"content":[{"type":"text","text":"继续"}]}}`)
	if err := server.PushEvent(testSession, answer); err != nil {
		t.Fatalf("推送工具结果: %v", err)
	}
	waitView(t, result.base, testSession, "提问结算", func(view SessionView) bool {
		return !view.HasCurrent && view.PendingCounts.Questions == 0
	})
}
