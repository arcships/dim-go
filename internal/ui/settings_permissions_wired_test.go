package ui

import (
	"context"
	"errors"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// rulesIntents 只接 WriteCommandRules：记下写入的整份规则，可按需失败。
type rulesIntents struct {
	Intents
	written chan []RuleSnapshot
	fail    error
}

func newRulesIntents() *rulesIntents {
	return &rulesIntents{written: make(chan []RuleSnapshot, 4)}
}

func (r *rulesIntents) WriteCommandRules(_ context.Context, rules []RuleSnapshot) error {
	select {
	case r.written <- rules:
	default:
	}
	return r.fail
}

// expectRules 等一次写入并核对规则集合。
func (r *rulesIntents) expectRules(t *testing.T, want []RuleSnapshot) {
	t.Helper()
	select {
	case got := <-r.written:
		if len(got) != len(want) {
			t.Fatalf("写入 %d 条规则，想要 %d 条：%+v", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("第 %d 条规则 = %+v，想要 %+v", i+1, got[i], want[i])
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有等到 WriteCommandRules")
	}
}

// openPermissions 打开设置 › 权限并渲染一帧（规则草稿从零开始）。
func openPermissions(t *testing.T, src *fakeSource, intents Intents) (*App, *gui.Tester) {
	t.Helper()
	resetSettingsScreenState()
	permissionsScreen = permissionsState{loaded: true}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	return app, openSettingsSection(t, app, secPermissions)
}

// TestPermissionsRulesComeFromFacts：草稿按 Source.Settings().Rules 铺一次，
// 服务端的规则原样出现在表里（命令、匹配、决策）。
func TestPermissionsRulesComeFromFacts(t *testing.T) {
	src := wiredSource()
	src.settings.RulesKnown = true
	src.settings.Rules = []RuleSnapshot{
		{Command: "terraform apply", Decision: "deny", Match: "prefix"},
		{Command: "git status", Decision: "allow", Match: "exact"},
	}
	_, tt := openPermissions(t, src, newRulesIntents())

	if len(permissionsScreen.rules) != 2 {
		t.Fatalf("草稿 %d 条，想要 2 条", len(permissionsScreen.rules))
	}
	first, second := permissionsScreen.rules[0], permissionsScreen.rules[1]
	if first.matcher != "terraform apply" || first.match != 1 || first.decision != 1 {
		t.Fatalf("prefix/deny 没有按服务端取值：%+v", first)
	}
	if second.matcher != "git status" || second.match != 0 || second.decision != 0 {
		t.Fatalf("exact/allow 没有按服务端取值：%+v", second)
	}
	// 两个下拉显示的就是事实里的取值（前缀 / 拒绝）。
	if !tt.HasText(permMatchPrefix) || !tt.HasText(permDecisionDeny) {
		t.Fatalf("下拉没有显示服务端的匹配与决策：%v", tt.Texts())
	}
}

// TestPermissionsSaveWritesRules：保存权限把整份规则交给写入通道，成功后写
// 「已保存。」，并按回执重铺草稿。
func TestPermissionsSaveWritesRules(t *testing.T) {
	src := wiredSource()
	src.settings.RulesKnown = true
	src.settings.Rules = []RuleSnapshot{{Command: "ls", Decision: "allow", Match: "exact"}}
	intents := newRulesIntents()
	_, tt := openPermissions(t, src, intents)

	// 加一条规则并填好：命令 + 前缀匹配 + 拒绝。
	if err := tt.Click(permCopyAddRule); err != nil {
		t.Fatalf("点不到「添加规则」：%v", err)
	}
	tt.Frame()
	permissionsScreen.rules[1].matcher = "  kubectl delete  "
	permissionsScreen.rules[1].match = 1
	permissionsScreen.rules[1].decision = 1
	tt.Frame()
	if err := tt.Click(permCopySaveRules); err != nil {
		t.Fatalf("点不到「保存权限」：%v", err)
	}
	intents.expectRules(t, []RuleSnapshot{
		{Command: "ls", Decision: "allow", Match: "exact"},
		{Command: "kubectl delete", Decision: "deny", Match: "prefix"},
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if permissionsScreen.permissionFeed == permCopySaved {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if permissionsScreen.permissionFeed != permCopySaved {
		t.Fatalf("保存成功后没有写回执：%q", permissionsScreen.permissionFeed)
	}
	if !tt.HasText(permCopySaved) {
		t.Fatalf("回执没有上屏：%v", tt.Texts())
	}
}

// TestPermissionsSaveFailureShowsReason：写入失败时把原因写进错误行，草稿不动。
func TestPermissionsSaveFailureShowsReason(t *testing.T) {
	src := wiredSource()
	src.settings.RulesKnown = true
	src.settings.Rules = []RuleSnapshot{{Command: "ls", Decision: "allow", Match: "exact"}}
	intents := newRulesIntents()
	intents.fail = errors.New("规则版本冲突")
	_, tt := openPermissions(t, src, intents)

	if err := tt.Click(permCopySaveRules); err != nil {
		t.Fatalf("点不到「保存权限」：%v", err)
	}
	intents.expectRules(t, []RuleSnapshot{{Command: "ls", Decision: "allow", Match: "exact"}})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if permissionsScreen.permissionErr != "" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if permissionsScreen.permissionErr == "" {
		t.Fatal("写入失败后没有写原因")
	}
	if !tt.HasText(permissionsScreen.permissionErr) {
		t.Fatalf("失败原因没有上屏：%v", tt.Texts())
	}
	if len(permissionsScreen.rules) != 1 {
		t.Fatalf("失败后草稿被改了：%+v", permissionsScreen.rules)
	}
}

// TestPermissionsSaveDisabledWithoutChannel：没有连接（-demo）时保存入口置灰并
// 写原因，不造「已保存」的假象。
func TestPermissionsSaveDisabledWithoutChannel(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	resetSettingsScreenState()
	tt := openSettingsSection(t, app, secPermissions)

	info, ok := tt.Access(permCopySaveRules)
	if !ok {
		t.Fatal("保存权限入口要保留")
	}
	if !info.Disabled {
		t.Fatalf("-demo 的保存权限没有置灰：%+v", info)
	}
}

// TestPermissionsRowErrorBlocksSave：行内校验错误时保存入口置灰（R-PRM-197）。
func TestPermissionsRowErrorBlocksSave(t *testing.T) {
	src := wiredSource()
	_, tt := openPermissions(t, src, newRulesIntents())

	if err := tt.Click(permCopyAddRule); err != nil {
		t.Fatalf("点不到「添加规则」：%v", err)
	}
	tt.Frame()
	// 填过又清空：行内报「请输入命令匹配式」。
	permissionsScreen.rules[0].matcher = "ls"
	tt.Frame()
	permissionsScreen.rules[0].matcher = ""
	tt.Frame()
	if !tt.HasText(permCopyErrEmpty) {
		t.Fatalf("空匹配式没有报错：%v", tt.Texts())
	}
	info, ok := tt.Access(permCopySaveRules)
	if !ok || !info.Disabled {
		t.Fatalf("有行内错误时保存入口应置灰：%+v（找到 %v）", info, ok)
	}
}
