package app

import (
	"context"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/interaction"
)

// TestRuleSourceReadAndWrite 核对命令规则的读取与写入映射。
func TestRuleSourceReadAndWrite(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"permission.rules", "permission.set_rules"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "permission.rules", Result: map[string]any{
				"revision": "r1",
				"commandRules": []any{map[string]any{
					"command": "go test", "match": "prefix", "decision": "allow",
				}},
			}},
			{Method: "permission.set_rules", Result: map[string]any{"revision": "r2"}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	source := &ruleSource{app: result.base}
	ctx := context.Background()

	set, err := source.ReadRules(ctx)
	if err != nil {
		t.Fatalf("读取命令规则: %v", err)
	}
	if set.Revision != "r1" || len(set.Rules) != 1 {
		t.Fatalf("规则读取映射不对: %+v", set)
	}
	if set.Rules[0].Command != "go test" || set.Rules[0].Match != interaction.CommandMatchPrefix ||
		set.Rules[0].Decision != interaction.CommandDecisionAllow {
		t.Fatalf("规则字段映射不对: %+v", set.Rules[0])
	}

	written, err := source.WriteRules(ctx, "r1", []interaction.CommandRule{{
		Command: "gofmt", Match: interaction.CommandMatchExact, Decision: interaction.CommandDecisionAllow,
	}})
	if err != nil {
		t.Fatalf("写入命令规则: %v", err)
	}
	if written.Revision != "r2" || len(written.Rules) != 1 || written.Rules[0].Command != "gofmt" {
		t.Fatalf("写入回执映射不对: %+v", written)
	}
	params := requestParams(t, server.RequestsFor("permission.set_rules")[0])
	if params["expectedRevision"] != "r1" {
		t.Fatalf("写入应带版本前置条件: %v", params)
	}
	rules, _ := params["commandRules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("写入规则集合不对: %v", params)
	}
	rule, _ := rules[0].(map[string]any)
	if rule["command"] != "gofmt" || rule["match"] != "exact" || rule["decision"] != "allow" {
		t.Fatalf("写入规则字段不对: %v", rule)
	}
}

// TestRuleSourceConflict 核对版本冲突原样交回领域分类，不重试、不覆盖。
func TestRuleSourceConflict(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"permission.rules", "permission.set_rules"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "permission.set_rules", Error: fakeappserver.AppError(-32000, "permission_rules_conflict", "stale")},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	source := &ruleSource{app: result.base}
	if _, err := source.WriteRules(context.Background(), "r1", nil); !errors.Is(err, interaction.ErrRulesConflict) {
		t.Fatalf("冲突应映射成 ErrRulesConflict: %v", err)
	}
	if got := len(server.RequestsFor("permission.set_rules")); got != 1 {
		t.Fatalf("冲突不应重试，实际请求 %d 次", got)
	}
}

// TestRuleSourceMissingCapability 核对 hello 未声明时零请求且归为能力缺失。
func TestRuleSourceMissingCapability(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	source := &ruleSource{app: result.base}
	ctx := context.Background()

	if _, err := source.ReadRules(ctx); !errors.Is(err, interaction.ErrRulesUnavailable) {
		t.Fatalf("读取缺能力应报 ErrRulesUnavailable: %v", err)
	}
	if _, err := source.WriteRules(ctx, "", nil); !errors.Is(err, interaction.ErrRulesUnavailable) {
		t.Fatalf("写入缺能力应报 ErrRulesUnavailable: %v", err)
	}
	for _, method := range []string{"permission.rules", "permission.set_rules"} {
		if got := len(server.RequestsFor(method)); got != 0 {
			t.Fatalf("缺能力时不应请求 %s（%d 次）", method, got)
		}
	}

	// 裁决确认缺失后不再发请求（缓存「不可用」只在内存）。
	arbiter, err := interaction.New(interaction.Config{
		SessionID: testSession,
		Writer:    result.base.Writes(),
		Rules:     source,
	})
	if err != nil {
		t.Fatalf("创建裁决: %v", err)
	}
	if _, err := arbiter.LoadRules(ctx); !errors.Is(err, interaction.ErrRulesUnavailable) {
		t.Fatalf("裁决加载规则应报能力缺失: %v", err)
	}
	if !arbiter.RulesUnavailable() {
		t.Fatal("裁决应记住规则能力缺失")
	}
	if got := len(server.RequestsFor("permission.rules")); got != 0 {
		t.Fatalf("记缺失后不应再请求 permission.rules（%d 次）", got)
	}
}
