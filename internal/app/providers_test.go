package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/settings"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestProviderAdminReads 核对 provider 清单、详情、探测与协作/agent 读取的映射。
func TestProviderAdminReads(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{
			"provider.list", "provider.management", "provider.probe",
			"collaboration.model_policy", "session.agent.get",
		},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "provider.list", Result: map[string]any{
				"providers": []any{map[string]any{
					"providerId": "p1", "source": "custom", "displayName": "P",
					"driverKind": "openai-compatible", "auth": map[string]any{"type": "bearer"},
					"credentialAuthority": "local", "baseUrl": "http://x",
					"proxyConfigured": false, "timeoutMs": 5000, "headerNames": []any{"X-Trace"},
					"credentialConfigured": true, "remoteModelList": false, "enabled": true, "version": 2,
				}},
				"errors": []any{map[string]any{"providerId": "p2", "message": "坏记录"}},
			}},
			{Method: "provider.management", Result: map[string]any{
				"providers": []any{map[string]any{
					"providerId": "p1", "source": "custom", "displayName": "P",
					"credentialAuthority": "local", "version": 2, "enabled": true,
					"actions": []any{"update"},
					"connection": map[string]any{
						"driverKind": "openai-compatible", "auth": map[string]any{"type": "bearer"},
						"baseUrl": "http://x", "credentialConfigured": true,
						"headerNames": []any{}, "proxyUrl": "", "proxyAuthConfigured": false,
					},
					"models": []any{map[string]any{
						"model": "m1", "source": "explicit", "enabled": true,
						"endpoint": "http://x", "missingLimits": []any{}, "unavailableReason": "",
						"actions": []any{},
					}},
					"preferences": map[string]any{"m1": map[string]any{"reasoningEffort": "low"}},
				}},
				"errors": []any{},
			}},
			{Method: "provider.probe", Result: map[string]any{"models": []any{map[string]any{
				"modelId": "probe-1", "displayName": "Probe",
				"capabilities": map[string]any{
					"contextWindow": 32000, "maxOutputTokens": 1000, "toolCalling": true, "vision": false,
				},
			}}}},
			{Method: "collaboration.model_policy", Result: map[string]any{
				"revision": "native:user/collaboration:3",
				"policy": map[string]any{
					"mode": "fixed", "model": map[string]any{"providerId": "p1", "modelId": "m1"},
				},
			}},
			{Method: "agent.list", Result: []any{map[string]any{
				"agentId": "a1", "name": "A", "description": "d", "sessions": 2, "default": true,
			}}},
			{Method: "session.agent.get", Result: map[string]any{"agentId": "a1"}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	domain := result.base.Settings()

	list, err := domain.RefreshProviders(ctx)
	if err != nil {
		t.Fatalf("读取 provider 清单: %v", err)
	}
	if len(list.Providers) != 1 || list.Providers[0].ID != "p1" ||
		list.Providers[0].AuthType != "bearer" || list.Providers[0].Version != 2 {
		t.Fatalf("provider 清单映射不对: %+v", list.Providers)
	}
	if len(list.Errors) != 1 || list.Errors[0].ProviderID != "p2" {
		t.Fatalf("provider 错误映射不对: %+v", list.Errors)
	}

	detail, err := domain.ReadProvider(ctx, "p1")
	if err != nil {
		t.Fatalf("读取 provider 详情: %v", err)
	}
	if detail.ID != "p1" || detail.Version == nil || *detail.Version != 2 {
		t.Fatalf("provider 详情映射不对: %+v", detail)
	}
	if len(detail.Models) != 1 || detail.Models[0].ID != "m1" || detail.Connection == nil ||
		detail.Connection.DriverKind != "openai-compatible" {
		t.Fatalf("provider 模型或连接映射不对: %+v", detail)
	}
	if detail.Preferences["m1"].ReasoningEffort != "low" {
		t.Fatalf("provider 偏好映射不对: %+v", detail.Preferences)
	}

	probe, err := domain.ProbeProvider(ctx, settings.ProviderProbeRequest{
		ProviderID: "p1",
		Connection: &settings.ProviderConnectionInput{
			DriverKind: "openai-compatible",
			Auth:       settings.ProviderAuth{Type: "anonymous"},
			BaseURL:    "http://127.0.0.1:9",
		},
	})
	if err != nil {
		t.Fatalf("探测 provider: %v", err)
	}
	if len(probe.Models) != 1 || probe.Models[0].ID != "probe-1" ||
		probe.Models[0].ContextWindow == nil || *probe.Models[0].ContextWindow != 32000 {
		t.Fatalf("探测结果映射不对: %+v", probe.Models)
	}
	probeParams := requestParams(t, server.RequestsFor("provider.probe")[0])
	if probeParams["providerId"] != "p1" {
		t.Fatalf("探测参数不对: %v", probeParams)
	}
	connection, _ := probeParams["connection"].(map[string]any)
	if connection["driverKind"] != "openai-compatible" || connection["baseUrl"] != "http://127.0.0.1:9" {
		t.Fatalf("探测连接参数不对: %v", probeParams)
	}

	policy, err := domain.RefreshCollaborationPolicy(ctx)
	if err != nil {
		t.Fatalf("读取协作策略: %v", err)
	}
	if policy.Revision != "native:user/collaboration:3" || policy.Policy.Mode != "fixed" ||
		policy.Policy.ModelProviderID != "p1" || policy.Policy.ModelID != "m1" {
		t.Fatalf("协作策略映射不对: %+v", policy)
	}

	agents, err := domain.RefreshAgents(ctx)
	if err != nil {
		t.Fatalf("读取 agent 清单: %v", err)
	}
	if len(agents) != 1 || agents[0].ID != "a1" || !agents[0].Default || agents[0].Sessions != 2 {
		t.Fatalf("agent 清单映射不对: %+v", agents)
	}

	agent, err := domain.ReadSessionAgent(ctx, testSession, mainBranchID)
	if err != nil {
		t.Fatalf("读取会话 agent: %v", err)
	}
	if agent.AgentID != "a1" || agent.BranchID != mainBranchID {
		t.Fatalf("会话 agent 映射不对: %+v", agent)
	}

	// 缺能力：hello 不声明 provider.list 时零请求。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	if _, err := result2.base.Settings().RefreshProviders(ctx); !errors.Is(err, settings.ErrCapabilityMissing) {
		t.Fatalf("缺能力应报能力缺失: %v", err)
	}
	if got := len(server2.RequestsFor("provider.list")); got != 0 {
		t.Fatalf("缺能力时不应请求 provider.list（%d 次）", got)
	}
}

// TestProviderAndAgentWritesUseRealMethods 核对 provider 管理、协作策略与
// 会话 agent 写入的映射：只发一次、op 不下发、expectedAgentId 不下发。
func TestProviderAndAgentWritesUseRealMethods(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{
			"provider.create", "provider.update",
			"provider.models.set_enabled", "provider.models.edit", "provider.models.reset",
			"provider.preferences.update", "collaboration.set_model_policy",
			"session.agent.get", "session.agent.update",
		},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "provider.create", Result: map[string]any{"providers": []any{}}},
			{Method: "provider.update", Result: map[string]any{"providers": []any{}}},
			{Method: "provider.models.set_enabled", Result: map[string]any{"providers": []any{}}},
			{Method: "provider.models.edit", Result: map[string]any{"providers": []any{}}},
			{Method: "provider.models.reset", Result: map[string]any{"providers": []any{}}},
			{Method: "provider.preferences.update", Result: map[string]any{"providers": []any{}}},
			{Method: "collaboration.set_model_policy", Result: map[string]any{
				"revision": "r2", "policy": map[string]any{"mode": "agent"},
			}},
			{Method: "session.agent.get", Result: map[string]any{"agentId": ""}},
			{Method: "session.agent.update", Result: map[string]any{"agentId": "a1"}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	writes := result.base.Writes()

	cases := []struct {
		name    string
		params  string
		method  string
		forbid  []string
		require map[string]any
	}{
		{
			name:   "provider create",
			params: `{"op":"provider.create","providerId":"6f3c1d2e-0000-4000-8000-000000000001","displayName":"P","connection":{"driverKind":"openai-compatible","auth":{"type":"anonymous"},"baseUrl":"http://x"},"models":[],"enabled":true}`,
			method: "provider.create",
			forbid: []string{"op"},
			require: map[string]any{
				"providerId":  "6f3c1d2e-0000-4000-8000-000000000001",
				"displayName": "P",
			},
		},
		{
			name:    "provider update",
			params:  `{"op":"provider.update","providerId":"p1","expectedVersion":2,"patch":{"displayName":"新名字"}}`,
			method:  "provider.update",
			forbid:  []string{"op"},
			require: map[string]any{"providerId": "p1", "expectedVersion": float64(2)},
		},
		{
			name:    "models set enabled",
			params:  `{"op":"provider.models.set_enabled","providerId":"p1","expectedVersion":3,"modelIds":["m1","m2"]}`,
			method:  "provider.models.set_enabled",
			forbid:  []string{"op"},
			require: map[string]any{"providerId": "p1", "expectedVersion": float64(3)},
		},
		{
			name:    "model edit",
			params:  `{"op":"provider.models.edit","providerId":"p1","expectedVersion":4,"modelId":"m1","edit":{"displayName":"M"}}`,
			method:  "provider.models.edit",
			forbid:  []string{"op"},
			require: map[string]any{"providerId": "p1", "modelId": "m1"},
		},
		{
			name:    "model reset",
			params:  `{"op":"provider.models.reset","providerId":"p1","expectedVersion":5,"modelId":"m1"}`,
			method:  "provider.models.reset",
			forbid:  []string{"op"},
			require: map[string]any{"providerId": "p1", "modelId": "m1"},
		},
		{
			name:    "preferences update",
			params:  `{"op":"provider.preferences.update","providerId":"p1","expectedVersion":6,"modelId":"m1","patch":{"serviceTier":{"action":"clear"}}}`,
			method:  "provider.preferences.update",
			forbid:  []string{"op"},
			require: map[string]any{"providerId": "p1", "modelId": "m1"},
		},
		{
			name:    "collaboration set policy",
			params:  `{"op":"collaboration.set_model_policy","expectedRevision":"r1","policy":{"mode":"agent"}}`,
			method:  "collaboration.set_model_policy",
			forbid:  []string{"op"},
			require: map[string]any{"expectedRevision": "r1"},
		},
		{
			name:    "session agent update",
			params:  `{"op":"session.agent.update","sessionId":"` + testSession + `","branchId":"main","agentId":"a1","expectedAgentId":""}`,
			method:  "session.agent.update",
			forbid:  []string{"op", "expectedAgentId"},
			require: map[string]any{"sessionId": testSession, "branchId": "main", "agentId": "a1"},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			registered, err := writes.Register(write.Intent{
				Key:       "settings-" + testCase.name,
				Kind:      write.KindManagementWrite,
				SessionID: testSession,
				Params:    json.RawMessage(testCase.params),
			})
			if err != nil {
				t.Fatalf("登记意图: %v", err)
			}
			settled, err := writes.Wait(ctx, registered.ID)
			if err != nil || settled.Status != write.StatusSucceeded {
				t.Fatalf("应成功: %v %+v", err, settled)
			}
			requests := server.RequestsFor(testCase.method)
			if len(requests) != 1 {
				t.Fatalf("%s 应只发一次，实际 %d", testCase.method, len(requests))
			}
			params := requestParams(t, requests[0])
			for _, forbidden := range testCase.forbid {
				if _, ok := params[forbidden]; ok {
					t.Fatalf("不应下发 %s: %v", forbidden, params)
				}
			}
			for key, want := range testCase.require {
				if got := params[key]; got != want {
					t.Fatalf("参数 %s=%v，期望 %v（%v）", key, got, want, params)
				}
			}
		})
	}
}

// TestProviderWritesMissingCapability 核对 hello 未声明时不发请求。
func TestProviderWritesMissingCapability(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	writes := result.base.Writes()

	ops := []struct{ op, params, method string }{
		{"provider.create", `{"providerId":"6f3c1d2e-0000-4000-8000-000000000001","displayName":"P","connection":{},"models":[]}`, "provider.create"},
		{"provider.update", `{"providerId":"p1","expectedVersion":2,"patch":{}}`, "provider.update"},
		{"provider.models.set_enabled", `{"providerId":"p1","expectedVersion":2,"modelIds":[]}`, "provider.models.set_enabled"},
		{"provider.models.edit", `{"providerId":"p1","expectedVersion":2,"modelId":"m1","edit":{}}`, "provider.models.edit"},
		{"provider.models.reset", `{"providerId":"p1","expectedVersion":2,"modelId":"m1"}`, "provider.models.reset"},
		{"provider.preferences.update", `{"providerId":"p1","expectedVersion":2,"modelId":"m1","patch":{}}`, "provider.preferences.update"},
		{"collaboration.set_model_policy", `{"expectedRevision":"r1","policy":{"mode":"agent"}}`, "collaboration.set_model_policy"},
	}
	for _, item := range ops {
		t.Run(item.op, func(t *testing.T) {
			params, err := json.Marshal(map[string]any{"op": item.op, "payload": json.RawMessage(item.params)})
			if err != nil {
				t.Fatalf("编码参数: %v", err)
			}
			registered, err := writes.Register(write.Intent{
				Key: "cap-" + item.op, Kind: write.KindManagementWrite, Params: params,
			})
			if err != nil {
				t.Fatalf("登记意图: %v", err)
			}
			settled, err := writes.Wait(ctx, registered.ID)
			if err != nil || settled.Status != write.StatusFailed || !strings.Contains(settled.Reason, "能力") {
				t.Fatalf("缺能力应明确失败: %v %+v", err, settled)
			}
			if got := len(server.RequestsFor(item.method)); got != 0 {
				t.Fatalf("缺能力时不应请求 %s（%d 次）", item.method, got)
			}
		})
	}
}
