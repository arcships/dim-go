package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/settings"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestCollaborationMembersAndStop 核对成员标识来源与停止映射。
func TestCollaborationMembersAndStop(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.collaboration.pending", "session.collaboration.cancel"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.collaboration.pending", Result: map[string]any{
				"branchId": "main", "through": 5,
				"requests": []any{map[string]any{
					"eventId": "c1", "sequence": 4,
					"request":  map[string]any{"turn_id": "t1", "request_id": "r1", "call_id": "call-1", "prompt": "p", "target": map[string]any{"kind": "agent", "agent_id": "a1"}},
					"delivery": map[string]any{"status": "ok", "agent_id": "a1", "session_id": "member-1"},
				}},
			}},
			{Method: "session.collaboration.cancel", Result: map[string]any{"status": "idle"}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()

	// 历史事实里的 delivered 也贡献成员标识。
	delivered := rawEvent("e60", 60, "collaboration", "delivered",
		`{"delivery":{"status":"ok","session_id":"member-2"}}`)
	if err := server.PushEvent(testSession, delivered); err != nil {
		t.Fatalf("推送 delivered: %v", err)
	}

	members, err := result.base.CollaborationMembers(ctx, testSession)
	if err != nil {
		t.Fatalf("读取成员: %v", err)
	}
	if len(members) != 2 || members[0] != "member-1" || members[1] != "member-2" {
		t.Fatalf("成员标识来源不对: %v", members)
	}

	registered, err := result.base.StopTeamMember(ctx, MemberStopRequest{
		SessionID: testSession, MemberSessionID: "member-1",
	})
	if err != nil {
		t.Fatalf("登记成员停止: %v", err)
	}
	settled, err := result.base.Wait(ctx, registered.ID)
	if err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("成员停止应成功: %v %+v", err, settled)
	}
	if got := len(server.RequestsFor("session.collaboration.cancel")); got != 1 {
		t.Fatalf("成员停止应只发一次，实际 %d", got)
	}
	params := requestParams(t, server.RequestsFor("session.collaboration.cancel")[0])
	if params["sessionId"] != testSession || params["branchId"] != mainBranchID ||
		params["memberSessionId"] != "member-1" {
		t.Fatalf("成员停止参数不对: %v", params)
	}
	if _, ok := params["op"]; ok {
		t.Fatalf("op 是意图级字段，不应下发: %v", params)
	}
	status, stopped, err := MemberStopStatus(settled.Result)
	if err != nil || status != "idle" || !stopped {
		t.Fatalf("idle 应视为已停: status=%q stopped=%t err=%v", status, stopped, err)
	}

	// 缺能力：入口直接报能力缺失，零请求。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	if _, err := result2.base.StopTeamMember(ctx, MemberStopRequest{
		SessionID: testSession, MemberSessionID: "member-1",
	}); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺能力应报能力缺失: %v", err)
	}
	if got := len(server2.RequestsFor("session.collaboration.cancel")); got != 0 {
		t.Fatalf("缺能力时不应请求（%d 次）", got)
	}
}

// TestRuntimeListReadsWhenDeclared 核对带默认预设（hello 声明 runtime.*）时的读取与安装。
func TestRuntimeListReadsWhenDeclared(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"runtime.list", "runtime.install"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "runtime.list", Result: []any{
				map[string]any{"name": "node", "source": "download", "auto": false, "status": "off"},
				map[string]any{"name": "python", "source": "download", "auto": false, "status": "off"},
				map[string]any{"name": "uv", "source": "download", "auto": false, "status": "off"},
			}},
			{Method: "runtime.install", Result: map[string]any{}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	entries, err := result.base.Settings().RefreshRuntimes(ctx)
	if err != nil {
		t.Fatalf("读取运行组件: %v", err)
	}
	if len(entries) != 3 || entries[0].Name != "node" || entries[0].Status != settings.RuntimeStatusOff {
		t.Fatalf("运行组件映射不对: %+v", entries)
	}
	registered, err := result.base.Writes().Register(write.Intent{
		Key: "runtime-install", Kind: write.KindManagementWrite,
		Params: []byte(`{"op":"runtime.install","name":"uv"}`),
	})
	if err != nil {
		t.Fatalf("登记组件安装: %v", err)
	}
	settled, err := result.base.Writes().Wait(ctx, registered.ID)
	if err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("组件安装应成功: %v %+v", err, settled)
	}
	params := requestParams(t, server.RequestsFor("runtime.install")[0])
	if params["name"] != "uv" {
		t.Fatalf("组件安装参数不对: %v", params)
	}
	_ = strings.TrimSpace
}
