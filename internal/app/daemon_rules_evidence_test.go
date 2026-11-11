package app

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/interaction"
)

// TestRealRuleSourceEvidence 是命令规则来源的真实证据。
//
// 只有设置 DIMAGENT_RULES_EVIDENCE=1 时运行；夹具 home 从 /tmp 只读复制到临时
// 目录，写入只影响副本。证据只留 revision 与规则条数。
func TestRealRuleSourceEvidence(t *testing.T) {
	if os.Getenv("DIMAGENT_RULES_EVIDENCE") != "1" {
		t.Skip("未设置 DIMAGENT_RULES_EVIDENCE=1，跳过真实命令规则证据")
	}
	fixtureHome := evidenceFixtureHome(t)
	serverPath := evidenceServerBinary(t)
	dataDir, err := os.MkdirTemp("", "dimagent-evidence-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	if err := copyFixtureTree(fixtureHome, dataDir); err != nil {
		t.Fatalf("复制夹具 home: %v", err)
	}

	base, err := New(Config{
		DataDir:       dataDir,
		ServerPath:    serverPath,
		ClientName:    "dimagent-mygo-rules-evidence",
		WorkspaceRoot: dataDir,
		Version:       testVersion,
		SendTimeout:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := base.Connect(ctx); err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	defer base.CloseConnection(ctx)

	source := &ruleSource{app: base}
	current, err := source.ReadRules(ctx)
	if err != nil {
		t.Fatalf("读取命令规则: %v", err)
	}
	t.Logf("permission.rules: revisionSet=%t count=%d", current.Revision != "", len(current.Rules))

	// 原样写回：只推进 revision，不改变规则语义。
	written, err := source.WriteRules(ctx, current.Revision, current.Rules)
	if err != nil {
		t.Fatalf("写回命令规则: %v", err)
	}
	t.Logf("permission.set_rules: revisionChanged=%t count=%d",
		written.Revision != current.Revision, len(written.Rules))

	// 用旧 revision 再写一次：应报冲突且不重试。
	if _, err := source.WriteRules(ctx, current.Revision, current.Rules); err == nil {
		t.Fatalf("旧 revision 写入应报冲突")
	} else if !errors.Is(err, interaction.ErrRulesConflict) {
		t.Fatalf("旧 revision 应映射成 ErrRulesConflict: %v", err)
	} else {
		t.Logf("permission.set_rules conflict: mapped=%t", errors.Is(err, interaction.ErrRulesConflict))
	}
}
