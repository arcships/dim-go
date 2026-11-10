package write

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

// connectionSender 是 D1 连接集成形状的测试替身：按意图类型选方法，把会话、
// 分支、投递方式与输入标识合入参数，然后只调用一次。
type connectionSender struct {
	client *protocol.Client
}

func (s *connectionSender) Send(ctx context.Context, req Request) (Reply, error) {
	params := make(map[string]any)
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return Reply{}, err
		}
	}
	if req.SessionID != "" {
		params["sessionId"] = req.SessionID
	}
	if req.BranchID != "" {
		params["branchId"] = req.BranchID
	}
	if req.Kind == KindSubmitInput {
		params["mode"] = string(req.Delivery)
		params["submissionId"] = req.InputID
	}
	var result struct {
		InputID string `json:"inputId"`
	}
	if err := s.client.CallInto(ctx, methodForKind(req.Kind), params, &result); err != nil {
		return Reply{}, err
	}
	raw, _ := json.Marshal(result)
	return Reply{InputID: result.InputID, Result: raw}, nil
}

func methodForKind(kind Kind) string {
	switch kind {
	case KindSubmitInput:
		return "session.submit"
	case KindStop:
		return "session.stop"
	case KindApprovalResponse:
		return "approval.respond"
	case KindLongTermAllow:
		return "approval.allow_always"
	case KindQuestionAnswer:
		return "question.respond"
	case KindManagementWrite:
		return "config.write"
	case KindAccountAction:
		return "account.action"
	case KindAttachmentWrite:
		return "resource.write"
	}
	return "unknown"
}

func startFakeServer(t *testing.T, script fakeappserver.Script) (*fakeappserver.Server, *protocol.Client) {
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
	t.Cleanup(func() {
		supervisor.Close(nil)
		_ = server.Close()
	})
	return server, conn.Client
}

func newAppServerManager(
	t *testing.T,
	sender Sender,
	inspector Inspector,
	sendTimeout time.Duration,
	attempts int,
	interval time.Duration,
) *Manager {
	t.Helper()
	manager, err := New(Config{
		Sender:      sender,
		Inspector:   inspector,
		SendTimeout: sendTimeout,
		Verify: VerifyPolicy{
			MaxAttempts: attempts,
			Interval:    interval,
			Timeout:     time.Second,
		},
	})
	if err != nil {
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

// TestAppServerSendsExactlyOnce 用假服务端核对：一次提交只产生一条请求，回执的
// 输入标识被记录，重复登记不重发。
func TestAppServerSendsExactlyOnce(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: "s1"}},
		Replies: []fakeappserver.Reply{{
			Method: "session.submit",
			Result: map[string]any{"inputId": "server-input-1"},
		}},
	})
	manager := newAppServerManager(t,
		&connectionSender{client: client},
		&fakeInspector{},
		2*time.Second, 2, time.Millisecond,
	)

	registered := mustRegister(t, manager, submitIntent("appserver-key-1"))
	if registered.InputID == "" {
		t.Fatal("提交前没有生成输入标识")
	}
	view := waitStatus(t, manager, registered.ID, StatusSucceeded)
	if view.InputID != "server-input-1" {
		t.Fatalf("输入标识=%q 期望服务端回执", view.InputID)
	}

	requests := server.RequestsFor("session.submit")
	if len(requests) != 1 {
		t.Fatalf("session.submit 请求数=%d 期望 1", len(requests))
	}
	var params map[string]any
	if err := json.Unmarshal(requests[0].Params, &params); err != nil {
		t.Fatalf("解析请求参数: %v", err)
	}
	if params["sessionId"] != "s1" || params["branchId"] != "main" {
		t.Fatalf("请求目标=%v 期望 s1/main", params)
	}
	if params["mode"] != string(DeliveryQueue) {
		t.Fatalf("投递方式=%v 期望 queue", params["mode"])
	}
	if params["submissionId"] != registered.InputID {
		t.Fatalf("submissionId=%v 期望 %q", params["submissionId"], registered.InputID)
	}

	again, err := manager.Register(submitIntent("appserver-key-1"))
	if err != nil {
		t.Fatalf("重复登记: %v", err)
	}
	if again.ID != registered.ID {
		t.Fatalf("重复登记返回新意图 %s", again.ID)
	}
	if got := len(server.RequestsFor("session.submit")); got != 1 {
		t.Fatalf("重复登记后请求数=%d 期望 1", got)
	}
}

// TestAppServerTimeoutThenConfirmedMiss 用假服务端制造无回执，核对确认未发生。
func TestAppServerTimeoutThenConfirmedMiss(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: "s1"}},
		Faults: []fakeappserver.Fault{{
			Method:     "session.submit",
			Occurrence: 1,
			NoReply:    true,
		}},
		Replies: []fakeappserver.Reply{{
			Method: "session.submit",
			Result: map[string]any{"inputId": "server-input-1"},
		}},
	})
	inspector := &fakeInspector{handle: func(InputQuery, int) (InputState, error) {
		return InputState{Confirmed: true, Present: false}, nil
	}}
	manager := newAppServerManager(t,
		&connectionSender{client: client},
		inspector,
		50*time.Millisecond, 2, 5*time.Millisecond,
	)

	registered := mustRegister(t, manager, submitIntent("appserver-timeout"))
	view := waitStatus(t, manager, registered.ID, StatusNotOccurred)
	if len(view.Content) == 0 {
		t.Fatal("确认未发生后待发内容被清空")
	}
	if got := len(server.RequestsFor("session.submit")); got != 1 {
		t.Fatalf("核对后请求数=%d 期望 1（不自动重发）", got)
	}
	if inspector.count() == 0 {
		t.Fatal("未知状态没有触发核对")
	}
}

// TestAppServerDisconnectBecomesUnknownThenNotOccurred 用假服务端在回执前断开。
func TestAppServerDisconnectBecomesUnknownThenNotOccurred(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: "s1"}},
		Faults: []fakeappserver.Fault{{
			Method:           "session.submit",
			Occurrence:       1,
			DisconnectBefore: true,
		}},
	})
	inspector := &fakeInspector{handle: func(InputQuery, int) (InputState, error) {
		return InputState{Confirmed: true, Present: false}, nil
	}}
	manager := newAppServerManager(t,
		&connectionSender{client: client},
		inspector,
		2*time.Second, 2, 5*time.Millisecond,
	)

	registered := mustRegister(t, manager, submitIntent("appserver-disconnect"))
	waitStatus(t, manager, registered.ID, StatusNotOccurred)
	if got := len(server.RequestsFor("session.submit")); got != 1 {
		t.Fatalf("断线后请求数=%d 期望 1", got)
	}
}

// TestAppServerCapabilityMissingFails 用 unknown_method 制造明确失败。
func TestAppServerCapabilityMissingFails(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: "s1"}},
		Faults: []fakeappserver.Fault{{
			Method:        "session.submit",
			Occurrence:    1,
			UnknownMethod: true,
		}},
	})
	inspector := &fakeInspector{}
	manager := newAppServerManager(t,
		&connectionSender{client: client},
		inspector,
		2*time.Second, 2, time.Millisecond,
	)

	registered := mustRegister(t, manager, submitIntent("appserver-missing"))
	view := waitStatus(t, manager, registered.ID, StatusFailed)
	if len(view.Content) == 0 {
		t.Fatal("明确失败后待发内容被清空")
	}
	if got := len(server.RequestsFor("session.submit")); got != 1 {
		t.Fatalf("失败后请求数=%d 期望 1", got)
	}
	if inspector.count() != 0 {
		t.Fatalf("明确失败不应核对，核对次数=%d", inspector.count())
	}
}

// TestRestartDoesNotResend 用新实例模拟重启：旧实例的未结项意图不进入新实例，
// 新实例启动后不自动发出任何请求。
func TestRestartDoesNotResend(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: "s1"}},
		Faults: []fakeappserver.Fault{{
			Method:     "session.submit",
			Occurrence: 1,
			NoReply:    true,
		}},
		Replies: []fakeappserver.Reply{{
			Method: "session.submit",
			Result: map[string]any{"inputId": "server-input-after-restart"},
		}},
	})
	inspector := &fakeInspector{handle: func(InputQuery, int) (InputState, error) {
		return InputState{Confirmed: false}, nil
	}}
	oldManager := newAppServerManager(t,
		&connectionSender{client: client},
		inspector,
		30*time.Millisecond, 2, 5*time.Millisecond,
	)
	registered := mustRegister(t, oldManager, submitIntent("appserver-restart"))
	waitStatus(t, oldManager, registered.ID, StatusNeedsManualConfirmation)
	oldManager.Close()
	if got := len(server.RequestsFor("session.submit")); got != 1 {
		t.Fatalf("重启前请求数=%d 期望 1", got)
	}

	// 新实例没有旧意图；构造后不应补发。
	newManager := newAppServerManager(t,
		&connectionSender{client: client},
		inspector,
		2*time.Second, 2, time.Millisecond,
	)
	if table := newManager.Table(); len(table) != 0 {
		t.Fatalf("新实例继承了旧意图: %+v", table)
	}
	if inflight := newManager.InFlight(); len(inflight) != 0 {
		t.Fatalf("新实例在途表非空: %+v", inflight)
	}
	time.Sleep(30 * time.Millisecond)
	if got := len(server.RequestsFor("session.submit")); got != 1 {
		t.Fatalf("新实例启动后自动补发，请求数=%d", got)
	}

	// 用户明确重提（新幂等键）才发送一次。
	retried := mustRegister(t, newManager, submitIntent("appserver-after-restart"))
	waitStatus(t, newManager, retried.ID, StatusSucceeded)
	if got := len(server.RequestsFor("session.submit")); got != 2 {
		t.Fatalf("明确重提后请求数=%d 期望 2", got)
	}
}
