package interaction

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// appServerSender 是 D1 连接集成形状的测试适配器：按写意图类型选方法，合入会话、
// 分支、请求标识与透明参数，只调用一次。
type appServerSender struct {
	client *protocol.Client
}

func (s *appServerSender) Send(ctx context.Context, req write.Request) (write.Reply, error) {
	params := make(map[string]any)
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return write.Reply{}, err
		}
	}
	if req.SessionID != "" {
		params["sessionId"] = req.SessionID
	}
	if req.BranchID != "" {
		params["branchId"] = req.BranchID
	}
	switch req.Kind {
	case write.KindApprovalResponse:
		// 交互裁决用 approve/reject；服务端只认 allow/deny。
		switch params["decision"] {
		case "approve", "allow", "allow_long_term":
			params["decision"] = "allow"
		case "reject", "deny":
			params["decision"] = "deny"
		}
		params["requestId"] = req.InputID
	case write.KindQuestionAnswer:
		params["requestId"] = req.InputID
	case write.KindLongTermAllow:
		return s.sendCommandRule(ctx, req)
	}
	var result json.RawMessage
	if err := s.client.CallInto(ctx, methodForKind(req.Kind), params, &result); err != nil {
		return write.Reply{}, err
	}
	return write.Reply{InputID: req.InputID, Result: result}, nil
}

// sendCommandRule 按真实服务端的长期允许路径写用户命令规则。
func (s *appServerSender) sendCommandRule(ctx context.Context, req write.Request) (write.Reply, error) {
	params := make(map[string]any)
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return write.Reply{}, err
		}
	}
	var current struct {
		Revision     string           `json:"revision"`
		CommandRules []map[string]any `json:"commandRules"`
	}
	if err := s.client.CallInto(ctx, "permission.rules", map[string]any{}, &current); err != nil {
		return write.Reply{}, err
	}
	rules := append(append([]map[string]any{}, current.CommandRules...), map[string]any{
		"command":  params["command"],
		"match":    params["match"],
		"decision": params["decision"],
	})
	var result json.RawMessage
	if err := s.client.CallInto(ctx, "permission.set_rules", map[string]any{
		"expectedRevision": current.Revision,
		"commandRules":     rules,
	}, &result); err != nil {
		return write.Reply{}, err
	}
	return write.Reply{InputID: req.InputID, Result: result}, nil
}

func methodForKind(kind write.Kind) string {
	switch kind {
	case write.KindApprovalResponse:
		return "approval.respond"
	case write.KindQuestionAnswer:
		return "question.respond"
	case write.KindStop:
		return "session.stop"
	}
	return "unknown"
}

// rejectingInspector 表示本测试不做未知核对；成功路径不会用到它。
type rejectingInspector struct{}

func (rejectingInspector) LookupInput(context.Context, write.InputQuery) (write.InputState, error) {
	return write.InputState{}, errors.New("测试不做核对")
}

// startInteractionServer 起一个假服务端，并用真实写登记接到交互裁决上。
func startInteractionServer(t *testing.T, script fakeappserver.Script) (*fakeappserver.Server, *write.Manager) {
	t.Helper()
	server, err := fakeappserver.Start(script)
	if err != nil {
		t.Fatalf("启动假服务端: %v", err)
	}
	supervisor, err := daemon.New(daemon.Config{
		DataDir:        server.DataDir(),
		RequestTimeout: 2 * time.Second,
	})
	if err != nil {
		_ = server.Close()
		t.Fatalf("创建连接监督者: %v", err)
	}
	conn, err := supervisor.Connect(context.Background())
	if err != nil {
		supervisor.Close(nil)
		_ = server.Close()
		t.Fatalf("连接假服务端: %v", err)
	}
	manager, err := write.New(write.Config{
		Sender:      &appServerSender{client: conn.Client},
		Inspector:   rejectingInspector{},
		SendTimeout: 2 * time.Second,
		Verify: write.VerifyPolicy{
			MaxAttempts: 1,
			Interval:    time.Millisecond,
			Timeout:     time.Second,
		},
	})
	if err != nil {
		supervisor.Close(nil)
		_ = server.Close()
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(func() {
		manager.Close()
		supervisor.Close(nil)
		_ = server.Close()
	})
	return server, manager
}

// TestAppServerLongTermWritesRuleBeforeResponse 用假服务端核对请求顺序：长期规则
// 先到，应答后到；结算后不再重发。
func TestAppServerLongTermWritesRuleBeforeResponse(t *testing.T) {
	server, manager := startInteractionServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession}},
		Replies: []fakeappserver.Reply{
			{Method: "permission.rules", Result: map[string]any{
				"revision": "r1", "commandRules": []any{},
			}},
			{Method: "permission.set_rules", Result: map[string]any{"revision": "r2"}},
			{Method: "approval.respond", Result: map[string]any{"ok": true}},
		},
	})
	arbiter, err := New(Config{SessionID: testSession, Writer: manager})
	if err != nil {
		t.Fatalf("创建交互裁决: %v", err)
	}
	arbiter.Apply(longTermApprovalFact("a1"))

	outcome, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm})
	if err != nil {
		t.Fatalf("长期允许: %v", err)
	}
	if outcome.Item.State != StateAwaitingResponse {
		t.Fatalf("应答后状态不对: %+v", outcome.Item)
	}

	methods := make([]string, 0, 3)
	for _, request := range server.Requests() {
		switch request.Method {
		case "permission.rules", "permission.set_rules", "approval.respond":
			methods = append(methods, request.Method)
		}
	}
	want := []string{"permission.rules", "permission.set_rules", "approval.respond"}
	if len(methods) != len(want) {
		t.Fatalf("服务端收到的顺序不对: %v", methods)
	}
	for i := range want {
		if methods[i] != want[i] {
			t.Fatalf("服务端收到的顺序不对: %v", methods)
		}
	}

	ruleParams := requestParams(t, server.RequestsFor("permission.set_rules")[0])
	rules, ok := ruleParams["commandRules"].([]any)
	if !ok || len(rules) != 1 {
		t.Fatalf("长期规则列表不对: %v", ruleParams["commandRules"])
	}
	rule, _ := rules[0].(map[string]any)
	if rule["command"] != "printf DIMAGENT_APPROVAL_OK" || rule["match"] != "exact" || rule["decision"] != "allow" {
		t.Fatalf("长期规则内容不对: %v", rule)
	}
	respondParams := requestParams(t, server.RequestsFor("approval.respond")[0])
	if respondParams["requestId"] != "a1" || respondParams["decision"] != "allow" {
		t.Fatalf("审批应答请求参数不对: %v", respondParams)
	}

	if _, ok := arbiter.Current(); !ok {
		t.Fatal("结算前应停在队首")
	}
	if !arbiter.Settle("a1") {
		t.Fatal("结算应成功")
	}
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm}); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("结算后不应再应答: %v", err)
	}
	if got := len(server.RequestsFor("approval.respond")); got != 1 {
		t.Fatalf("应答应只发一次，实际 %d", got)
	}
	if got := len(server.RequestsFor("permission.set_rules")); got != 1 {
		t.Fatalf("长期规则应只写一次，实际 %d", got)
	}
}

// TestAppServerLongTermFailureSendsNoResponse 用假服务端核对：规则写入能力缺失时
// 不发应答，审批留在待决集合。
func TestAppServerLongTermFailureSendsNoResponse(t *testing.T) {
	server, manager := startInteractionServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession}},
		Faults: []fakeappserver.Fault{{
			Method:        "permission.rules",
			UnknownMethod: true,
		}},
	})
	arbiter, err := New(Config{SessionID: testSession, Writer: manager})
	if err != nil {
		t.Fatalf("创建交互裁决: %v", err)
	}
	arbiter.Apply(longTermApprovalFact("a1"))

	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm}); !errors.Is(err, ErrLongTermWriteFailed) {
		t.Fatalf("应报长期规则写入失败: %v", err)
	}
	if got := len(server.RequestsFor("approval.respond")); got != 0 {
		t.Fatalf("写入失败不得发应答，实际 %d", got)
	}
	item, ok := arbiter.Current()
	if !ok || item.RequestID != "a1" || item.State != StatePresented {
		t.Fatalf("审批应留在待呈现: %+v ok=%v", item, ok)
	}
}

// TestAppServerQuestionsOneAtATime 用假服务端核对：两个提问只应答一个，结算后
// 才发第二个。
func TestAppServerQuestionsOneAtATime(t *testing.T) {
	server, manager := startInteractionServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession}},
		Replies: []fakeappserver.Reply{
			{Method: "question.respond", Result: map[string]any{"ok": true}},
		},
	})
	arbiter, err := New(Config{SessionID: testSession, Writer: manager})
	if err != nil {
		t.Fatalf("创建交互裁决: %v", err)
	}
	seedQuestionList(t, arbiter, 10,
		questionItem("q1", "c1", 1),
		questionItem("q2", "c2", 2),
	)

	if _, err := arbiter.Respond(context.Background(), "q2", answerResponse("第二条")); !errors.Is(err, ErrNotCurrent) {
		t.Fatalf("第二条提问不应越前应答: %v", err)
	}
	if got := len(server.RequestsFor("question.respond")); got != 0 {
		t.Fatalf("越前应答不应发请求: %d", got)
	}

	if _, err := arbiter.Respond(context.Background(), "q1", answerResponse("第一条")); err != nil {
		t.Fatalf("应答第一条: %v", err)
	}
	if got := len(server.RequestsFor("question.respond")); got != 1 {
		t.Fatalf("结算前只应有一个应答请求: %d", got)
	}
	arbiter.SettleQuestion("q1", "c1", 20)
	if _, err := arbiter.Respond(context.Background(), "q2", answerResponse("第二条")); err != nil {
		t.Fatalf("应答第二条: %v", err)
	}
	requests := server.RequestsFor("question.respond")
	if len(requests) != 2 {
		t.Fatalf("两个提问应各发一次: %d", len(requests))
	}
	first := requestParams(t, requests[0])
	second := requestParams(t, requests[1])
	if first["requestId"] != "q1" || second["requestId"] != "q2" {
		t.Fatalf("提问应答对象不对: %v / %v", first, second)
	}
}

func requestParams(t *testing.T, request fakeappserver.Request) map[string]any {
	t.Helper()
	params := make(map[string]any)
	if err := json.Unmarshal(request.Params, &params); err != nil {
		t.Fatalf("请求参数不是 JSON 对象: %s (%v)", request.Params, err)
	}
	return params
}
