package interaction_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/interaction"
)

// TestRealRuleMatching 验证真实服务端的命令规则匹配行为：exact、prefix、
// 带管道的命令不被 allow 规则放行、deny 规则优先。
//
//	DIM_REAL_MODEL_FIXTURE=<fixture> DIM_REAL_MODEL_SERVER=<server> \
//	  go test -run TestRealRuleMatching -v -count=1 -timeout 12m ./internal/interaction/
//
// 只记录规则 revision/条数、匹配类型与匹配结果，不打印命令正文。
func TestRealRuleMatching(t *testing.T) {
	fixture := realFixtureHome(t)
	server := realServerPath(t)
	home := prepareApprovalHome(t, fixture)
	base := startRealApp(t, home, server)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	t.Cleanup(func() { _ = base.Close(context.Background()) })

	rules := startRuleConnection(t, home)
	t.Logf("rules: initial revision=%s count=%d", currentRules(t, rules).Revision, len(currentRules(t, rules).Rules))

	exactRule := interaction.CommandRule{
		Command:  "printf DIMAGENT_RULE_EXACT",
		Match:    interaction.CommandMatchExact,
		Decision: interaction.CommandDecisionAllow,
	}
	addRule(t, rules, exactRule)
	exactSession := newWorkspaceSession(t, ctx, base)
	runRuleCase(t, ctx, base, exactSession,
		"Use the shell tool to run exactly: printf DIMAGENT_RULE_EXACT. You must call the shell tool; do not answer from memory.",
		"DIMAGENT_RULE_EXACT", exactRule.Command, false)
	t.Logf("match: exact command allowed=true")

	prefixRule := interaction.CommandRule{
		Command:  "printf DIMAGENT_RULE_PREFIX",
		Match:    interaction.CommandMatchPrefix,
		Decision: interaction.CommandDecisionAllow,
	}
	addRule(t, rules, prefixRule)
	prefixSession := newWorkspaceSession(t, ctx, base)
	prefixCommand := "printf DIMAGENT_RULE_PREFIX extra"
	runRuleCase(t, ctx, base, prefixSession,
		"Use the shell tool to run exactly: "+prefixCommand+". You must call the shell tool with the extra argument; do not answer from memory.",
		"DIMAGENT_RULE_PREFIX", prefixCommand, false)
	t.Logf("match: prefix leading words allowed=true")

	pipeBase := "printf DIMAGENT_RULE_PIPE"
	addRule(t, rules, interaction.CommandRule{
		Command:  pipeBase,
		Match:    interaction.CommandMatchExact,
		Decision: interaction.CommandDecisionAllow,
	})
	pipeSession := newWorkspaceSession(t, ctx, base)
	pipeCommand := pipeBase + " | cat"
	pipeApproval := submitAndWaitApproval(t, ctx, base, pipeSession,
		"Use the shell tool to run exactly this shell command, including the pipe: "+pipeCommand+". You must call the shell tool; do not answer from memory.")
	command := lastToolCommand(t, ctx, base, pipeSession)
	if !strings.Contains(command, "|") {
		t.Fatalf("模型没有按提示执行管道命令，无法验证 allow 规则边界")
	}
	t.Logf("match: pipeline approval=true")
	if _, err := base.Respond(ctx, pipeSession, pipeApproval.RequestID, interaction.Response{
		Decision: interaction.DecisionApprove,
	}); err != nil {
		t.Fatalf("同意管道命令审批: %v", err)
	}
	waitToolResultMarker(t, ctx, base, pipeSession, "DIMAGENT_RULE_PIPE")
	t.Logf("match: pipeline approved and executed=true")

	denyRule := interaction.CommandRule{
		Command:  "printf DIMAGENT_RULE_DENY",
		Match:    interaction.CommandMatchExact,
		Decision: interaction.CommandDecisionDeny,
	}
	addRule(t, rules, denyRule)
	denySession := newWorkspaceSession(t, ctx, base)
	submitRealPromptText(t, ctx, base, denySession,
		"Use the shell tool to run exactly: printf DIMAGENT_RULE_DENY. You must call the shell tool; do not answer from memory.")
	waitDeniedWithoutApproval(t, ctx, base, denySession)
	t.Logf("match: deny rule applied_without_approval=true")
}

func startRuleConnection(t *testing.T, home string) *interaction.Arbiter {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	supervisor, err := daemon.New(daemon.Config{DataDir: home, RequestTimeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("创建规则连接: %v", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		supervisor.Close(nil)
		t.Fatalf("连接规则 daemon: %v", err)
	}
	t.Cleanup(func() { supervisor.Close(nil) })
	arbiter, err := interaction.New(interaction.Config{
		SessionID: "rule-matching-evidence",
		Writer:    unusedWriter{},
		Rules:     &protocolRuleSource{client: conn.Client},
	})
	if err != nil {
		t.Fatalf("创建规则裁决器: %v", err)
	}
	if _, err := arbiter.LoadRules(ctx); err != nil {
		t.Fatalf("读取规则: %v", err)
	}
	return arbiter
}

func currentRules(t *testing.T, arbiter *interaction.Arbiter) interaction.RuleSet {
	t.Helper()
	set, ok := arbiter.Rules()
	if !ok {
		t.Fatal("没有规则缓存")
	}
	return set
}

func addRule(t *testing.T, arbiter *interaction.Arbiter, rule interaction.CommandRule) {
	t.Helper()
	set, err := arbiter.AddRule(context.Background(), rule)
	if err != nil {
		t.Fatalf("新增规则: %v", err)
	}
	t.Logf("rules: revision=%s count=%d", set.Revision, len(set.Rules))
}

func newWorkspaceSession(t *testing.T, ctx context.Context, base *app.App) string {
	t.Helper()
	sessionID := createRealSession(t, ctx, base)
	if err := setWorkspaceWrite(ctx, base, sessionID); err != nil {
		t.Fatalf("设置 workspace-write: %v", err)
	}
	return sessionID
}

// runRuleCase 要求命令在没有审批的情况下执行成功，并且工具调用的命令包含预期文本。
func runRuleCase(t *testing.T, ctx context.Context, base *app.App, sessionID, prompt, marker, wantCommand string, wantApproval bool) {
	t.Helper()
	submitRealPromptText(t, ctx, base, sessionID, prompt)
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		view, _ := base.ReloadLatest(ctx, sessionID)
		if view.HasCurrent && view.Current.Kind == interaction.PendingApproval && !wantApproval {
			t.Fatalf("规则本应放行，但仍触发审批")
		}
		for _, command := range toolCommandsSince(view, 0) {
			if command == wantCommand && toolResultContains(view, marker) {
				return
			}
		}
		if turnEnded(view) && !toolResultContains(view, marker) {
			t.Fatalf("命令没有成功执行，无法验证规则匹配")
		}
		select {
		case <-ctx.Done():
			t.Fatalf("等待规则匹配超时: %v", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatal("等待规则匹配超时")
}

func submitAndWaitApproval(t *testing.T, ctx context.Context, base *app.App, sessionID, prompt string) interaction.Item {
	t.Helper()
	submitRealPromptText(t, ctx, base, sessionID, prompt)
	return waitApproval(t, ctx, base, sessionID)
}

func lastToolCommand(t *testing.T, ctx context.Context, base *app.App, sessionID string) string {
	t.Helper()
	view, _ := base.ReloadLatest(ctx, sessionID)
	commands := toolCommandsSince(view, 0)
	if len(commands) == 0 {
		return ""
	}
	return commands[len(commands)-1]
}

func toolCommandsSince(view app.SessionView, from int) []string {
	if from < 0 || from > len(view.Rows) {
		from = 0
	}
	var commands []string
	for _, row := range view.Rows[from:] {
		if row.Tool == nil || len(row.Tool.Input) == 0 {
			continue
		}
		var payload struct {
			Command string `json:"command"`
		}
		if json.Unmarshal(row.Tool.Input, &payload) == nil && payload.Command != "" {
			commands = append(commands, payload.Command)
		}
	}
	return commands
}

func waitToolResultMarker(t *testing.T, ctx context.Context, base *app.App, sessionID, marker string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		view, _ := base.ReloadLatest(ctx, sessionID)
		if toolResultContains(view, marker) {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("等待工具结果超时: %v", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatal("等待工具结果超时")
}

func toolResultContains(view app.SessionView, marker string) bool {
	for _, row := range view.Rows {
		if row.Tool != nil && !row.Tool.IsError && strings.Contains(row.Tool.Result, marker) {
			return true
		}
	}
	return false
}

func waitDeniedWithoutApproval(t *testing.T, ctx context.Context, base *app.App, sessionID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		view, _ := base.ReloadLatest(ctx, sessionID)
		if view.HasCurrent && view.Current.Kind == interaction.PendingApproval {
			t.Fatalf("deny 规则不应再触发审批")
		}
		if _, failed := toolOutcomeSince(view, 0); failed {
			return
		}
		if turnEnded(view) {
			t.Fatalf("deny 规则没有产生工具错误结果")
		}
		select {
		case <-ctx.Done():
			t.Fatalf("等待 deny 结果超时: %v", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatal("等待 deny 结果超时")
}
