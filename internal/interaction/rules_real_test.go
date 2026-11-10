package interaction_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestRealCommandRules 验证真实 daemon 上的命令规则读取、写入、编辑与撤销。
//
//	DIM_REAL_MODEL_FIXTURE=/tmp/dimagent-real-model \
//	  go test -run TestRealCommandRules -v -count=1 -timeout 6m ./internal/interaction/
//
// 只输出 revision 与规则条数，不打印命令全文；夹具复制到新的 /tmp 目录，来源不改。
func TestRealCommandRules(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("设置 DIM_REAL_MODEL_FIXTURE 后运行真实命令规则证据")
	}
	home := prepareRulesHome(t, fixture)
	serverPath := filepath.Join(fixture, "bin", "dimcode-server")
	if _, err := os.Stat(serverPath); err != nil {
		t.Fatalf("夹具缺少 daemon: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	supervisor, err := daemon.New(daemon.Config{
		DataDir:        home,
		ServerPath:     serverPath,
		ServerArgs:     []string{"app-server"},
		RequestTimeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建 daemon 监督者: %v", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		supervisor.Close(nil)
		t.Fatalf("连接 daemon: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer shutdownCancel()
		_, _ = supervisor.Shutdown(shutdownCtx)
	})

	source := &protocolRuleSource{client: conn.Client}
	arbiter, err := interaction.New(interaction.Config{
		SessionID: "rules-evidence",
		Writer:    unusedWriter{},
		Rules:     source,
	})
	if err != nil {
		t.Fatalf("创建交互裁决: %v", err)
	}

	initial, err := arbiter.LoadRules(ctx)
	if errors.Is(err, interaction.ErrRulesUnavailable) {
		t.Skip("blocked: 服务端没有 permission.rules 能力")
	}
	if err != nil {
		t.Fatalf("读取命令规则: %v", err)
	}
	t.Logf("rules: read revision=%s count=%d", initial.Revision, len(initial.Rules))

	command := fmt.Sprintf("printf DIMAGENT_RULE_%d", time.Now().UnixNano())
	added, err := arbiter.AddRule(ctx, interaction.CommandRule{
		Command:  command,
		Match:    interaction.CommandMatchExact,
		Decision: interaction.CommandDecisionAllow,
	})
	if err != nil {
		t.Fatalf("新增命令规则: %v", err)
	}
	index := len(added.Rules) - 1
	t.Logf("rules: add revision=%s count=%d", added.Revision, len(added.Rules))

	edited, err := arbiter.EditRule(ctx, index, interaction.CommandRule{
		Command:  command,
		Match:    interaction.CommandMatchPrefix,
		Decision: interaction.CommandDecisionAllow,
	})
	if err != nil {
		t.Fatalf("编辑命令规则: %v", err)
	}
	t.Logf("rules: edit revision=%s count=%d", edited.Revision, len(edited.Rules))

	revoked, err := arbiter.RevokeRule(ctx, index)
	if err != nil {
		t.Fatalf("撤销命令规则: %v", err)
	}
	t.Logf("rules: revoke revision=%s count=%d", revoked.Revision, len(revoked.Rules))
	if len(revoked.Rules) != len(initial.Rules) {
		t.Fatalf("撤销后规则条数应回到初始值: initial=%d final=%d", len(initial.Rules), len(revoked.Rules))
	}

	cached, ok := arbiter.Rules()
	if !ok || cached.Revision != revoked.Revision || len(cached.Rules) != len(revoked.Rules) {
		t.Fatalf("本地缓存没有跟随最后 revision: %+v ok=%t", cached, ok)
	}
	t.Logf("rules: cache revision=%s count=%d", cached.Revision, len(cached.Rules))
}

// protocolRuleSource 用真实 permission.rules / permission.set_rules 实现 RuleSource。
type protocolRuleSource struct {
	client *protocol.Client
}

type wireRule struct {
	Command  string `json:"command"`
	Match    string `json:"match"`
	Decision string `json:"decision"`
}

type wireRules struct {
	Revision     string     `json:"revision"`
	CommandRules []wireRule `json:"commandRules"`
}

func (s *protocolRuleSource) ReadRules(ctx context.Context) (interaction.RuleSet, error) {
	var reply wireRules
	if err := s.client.CallInto(ctx, "permission.rules", map[string]any{}, &reply); err != nil {
		return interaction.RuleSet{}, mapRuleError(err)
	}
	return toRuleSet(reply), nil
}

func (s *protocolRuleSource) WriteRules(
	ctx context.Context,
	expectedRevision string,
	rules []interaction.CommandRule,
) (interaction.RuleSet, error) {
	wire := make([]wireRule, 0, len(rules))
	for _, rule := range rules {
		wire = append(wire, wireRule{
			Command:  rule.Command,
			Match:    string(rule.Match),
			Decision: string(rule.Decision),
		})
	}
	var reply wireRules
	if err := s.client.CallInto(ctx, "permission.set_rules", map[string]any{
		"expectedRevision": expectedRevision,
		"commandRules":     wire,
	}, &reply); err != nil {
		return interaction.RuleSet{}, mapRuleError(err)
	}
	return toRuleSet(reply), nil
}

func toRuleSet(reply wireRules) interaction.RuleSet {
	rules := make([]interaction.CommandRule, 0, len(reply.CommandRules))
	for _, rule := range reply.CommandRules {
		rules = append(rules, interaction.CommandRule{
			Command:  rule.Command,
			Match:    interaction.CommandMatch(rule.Match),
			Decision: interaction.CommandDecision(rule.Decision),
		})
	}
	return interaction.RuleSet{Revision: reply.Revision, Rules: rules}
}

func mapRuleError(err error) error {
	var rpcErr *protocol.RPCError
	if errors.As(err, &rpcErr) {
		switch rpcErr.AppError() {
		case "unknown_method":
			return interaction.ErrRulesUnavailable
		case "permission_rules_conflict":
			return interaction.ErrRulesConflict
		}
		if rpcErr.Code == -32601 {
			return interaction.ErrRulesUnavailable
		}
	}
	if failure.Is(err, failure.KindNonRetryable) && rpcErr == nil {
		return interaction.ErrRulesUnavailable
	}
	return err
}

// unusedWriter 只满足 Config 的必填 Writer；命令规则不走写登记。
type unusedWriter struct{}

func (unusedWriter) Register(write.Intent) (write.IntentView, error) {
	return write.IntentView{}, errors.New("命令规则不使用写登记")
}

func (unusedWriter) Wait(context.Context, string) (write.IntentView, error) {
	return write.IntentView{}, errors.New("命令规则不使用写登记")
}

// prepareRulesHome 复制夹具 home 到短路径临时目录，避免 Unix socket 路径超限。
func prepareRulesHome(t *testing.T, fixture string) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "dimagent-rules-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	home := filepath.Join(root, "home")
	if err := copyDir(filepath.Join(fixture, "home"), home); err != nil {
		t.Fatalf("复制夹具 home: %v", err)
	}
	for _, name := range []string{"daemon.sock", "daemon.sock.lock", "daemon.start.lock"} {
		_ = os.Remove(filepath.Join(home, "rust", name))
	}
	return home
}
