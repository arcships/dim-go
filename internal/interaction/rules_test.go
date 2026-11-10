package interaction

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

// fakeRuleSource 是命令规则来源的测试替身：记录读写次数与每次写入的 revision。
type fakeRuleSource struct {
	mu          sync.Mutex
	readCount   int
	writeCount  int
	readSet     RuleSet
	readErr     error
	writeErr    error
	writeResult RuleSet
	expected    []string
	written     [][]CommandRule
}

func (f *fakeRuleSource) ReadRules(context.Context) (RuleSet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readCount++
	if f.readErr != nil {
		return RuleSet{}, f.readErr
	}
	return cloneRuleSet(f.readSet), nil
}

func (f *fakeRuleSource) WriteRules(_ context.Context, expectedRevision string, rules []CommandRule) (RuleSet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writeCount++
	f.expected = append(f.expected, expectedRevision)
	f.written = append(f.written, cloneRules(rules))
	if f.writeErr != nil {
		return RuleSet{}, f.writeErr
	}
	result := cloneRuleSet(f.writeResult)
	if result.Revision == "" {
		result.Revision = expectedRevision + "-next"
	}
	result.Rules = cloneRules(rules)
	return result, nil
}

func (f *fakeRuleSource) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.readCount, f.writeCount
}

func newRuleArbiter(t *testing.T, source RuleSource) *Arbiter {
	t.Helper()
	arbiter, err := New(Config{
		SessionID: testSession,
		Writer:    newFakeWriter(),
		Rules:     source,
	})
	if err != nil {
		t.Fatalf("创建交互裁决: %v", err)
	}
	return arbiter
}

func allowRule(command string, match CommandMatch) CommandRule {
	return CommandRule{Command: command, Match: match, Decision: CommandDecisionAllow}
}

// TestLoadRulesCachesByRevisionAndReturnsCopies 核对：缓存按 revision 更新，视图是拷贝。
func TestLoadRulesCachesByRevisionAndReturnsCopies(t *testing.T) {
	source := &fakeRuleSource{readSet: RuleSet{
		Revision: "r1",
		Rules:    []CommandRule{allowRule("printf one", CommandMatchExact)},
	}}
	arbiter := newRuleArbiter(t, source)
	ctx := context.Background()

	loaded, err := arbiter.LoadRules(ctx)
	if err != nil {
		t.Fatalf("读取规则: %v", err)
	}
	if loaded.Revision != "r1" || len(loaded.Rules) != 1 {
		t.Fatalf("读取结果不对: %+v", loaded)
	}

	// 视图是拷贝：改它不影响缓存。
	cached, ok := arbiter.Rules()
	if !ok {
		t.Fatal("应有规则缓存")
	}
	cached.Rules[0].Command = "被改了"
	again, _ := arbiter.Rules()
	if again.Rules[0].Command != "printf one" {
		t.Fatalf("规则视图不是拷贝: %+v", again.Rules)
	}

	// revision 相同：保留缓存，不用回执替换。
	source.mu.Lock()
	source.readSet = RuleSet{
		Revision: "r1",
		Rules:    []CommandRule{allowRule("printf two", CommandMatchExact), allowRule("printf three", CommandMatchExact)},
	}
	source.mu.Unlock()
	same, err := arbiter.LoadRules(ctx)
	if err != nil {
		t.Fatalf("按相同 revision 重读: %v", err)
	}
	if len(same.Rules) != 1 || same.Rules[0].Command != "printf one" {
		t.Fatalf("相同 revision 不应替换缓存: %+v", same.Rules)
	}

	// revision 变了：整体替换。
	source.mu.Lock()
	source.readSet.Revision = "r2"
	source.mu.Unlock()
	changed, err := arbiter.LoadRules(ctx)
	if err != nil {
		t.Fatalf("按新 revision 重读: %v", err)
	}
	if changed.Revision != "r2" || len(changed.Rules) != 2 {
		t.Fatalf("revision 变化应整体替换: %+v", changed)
	}
	reads, writes := source.counts()
	if reads != 3 || writes != 0 {
		t.Fatalf("读写次数不对: reads=%d writes=%d", reads, writes)
	}
}

// TestAddEditRevokeRewriteRuleSet 核对：新增、编辑、撤销都重写整个集合，带上一 revision。
func TestAddEditRevokeRewriteRuleSet(t *testing.T) {
	first := allowRule("printf one", CommandMatchExact)
	source := &fakeRuleSource{readSet: RuleSet{Revision: "r1", Rules: []CommandRule{first}}}
	arbiter := newRuleArbiter(t, source)
	ctx := context.Background()
	if _, err := arbiter.LoadRules(ctx); err != nil {
		t.Fatalf("读取规则: %v", err)
	}

	second := CommandRule{Command: "printf two", Match: CommandMatchPrefix, Decision: CommandDecisionDeny}
	added, err := arbiter.AddRule(ctx, second)
	if err != nil {
		t.Fatalf("新增规则: %v", err)
	}
	if added.Revision != "r1-next" || len(added.Rules) != 2 {
		t.Fatalf("新增后回执不对: %+v", added)
	}
	// 完全相同的规则不再写。
	if _, err := arbiter.AddRule(ctx, second); err != nil {
		t.Fatalf("重复新增: %v", err)
	}

	third := allowRule("printf three", CommandMatchExact)
	edited, err := arbiter.EditRule(ctx, 1, third)
	if err != nil {
		t.Fatalf("编辑规则: %v", err)
	}
	if edited.Revision != "r1-next-next" || len(edited.Rules) != 2 || edited.Rules[1] != third {
		t.Fatalf("编辑后回执不对: %+v", edited)
	}

	revoked, err := arbiter.RevokeRule(ctx, 0)
	if err != nil {
		t.Fatalf("撤销规则: %v", err)
	}
	if len(revoked.Rules) != 1 || revoked.Rules[0] != third {
		t.Fatalf("撤销后回执不对: %+v", revoked)
	}

	source.mu.Lock()
	defer source.mu.Unlock()
	if source.writeCount != 3 {
		t.Fatalf("应写三次（重复新增不写），实际 %d", source.writeCount)
	}
	wantExpected := []string{"r1", "r1-next", "r1-next-next"}
	if !reflect.DeepEqual(source.expected, wantExpected) {
		t.Fatalf("每次写入的 revision 不对: %v", source.expected)
	}
	if len(source.written[0]) != 2 ||
		len(source.written[1]) != 2 || source.written[1][1] != third ||
		len(source.written[2]) != 1 || source.written[2][0] != third {
		t.Fatalf("写入的规则集合不对: %+v", source.written)
	}
	cached, ok := arbiter.Rules()
	if !ok || len(cached.Rules) != 1 || cached.Rules[0] != third {
		t.Fatalf("缓存没有跟上最后一次写入: %+v ok=%v", cached, ok)
	}
}

// TestWriteReplyWithoutRulesUsesSentSet 核对：真实 set_rules 回执只带 revision 时，
// 缓存使用本次写入的规则集合，而不是被空集合覆盖。
func TestWriteReplyWithoutRulesUsesSentSet(t *testing.T) {
	source := &fakeRuleSource{
		readSet:     RuleSet{Revision: "r1"},
		writeResult: RuleSet{Revision: "r2"},
	}
	arbiter := newRuleArbiter(t, source)
	ctx := context.Background()
	if _, err := arbiter.LoadRules(ctx); err != nil {
		t.Fatalf("读取规则: %v", err)
	}
	rule := allowRule("printf one", CommandMatchExact)
	added, err := arbiter.AddRule(ctx, rule)
	if err != nil {
		t.Fatalf("新增规则: %v", err)
	}
	if added.Revision != "r2" || len(added.Rules) != 1 || added.Rules[0] != rule {
		t.Fatalf("回执缺少规则时应使用本次写入集合: %+v", added)
	}
	cached, ok := arbiter.Rules()
	if !ok || len(cached.Rules) != 1 || cached.Rules[0] != rule {
		t.Fatalf("缓存不应被空规则覆盖: %+v ok=%v", cached, ok)
	}
}

// TestRulesConflictDoesNotOverwrite 核对：revision 冲突返回错误，缓存与本地规则不覆盖。
func TestRulesConflictDoesNotOverwrite(t *testing.T) {
	first := allowRule("printf one", CommandMatchExact)
	source := &fakeRuleSource{
		readSet:  RuleSet{Revision: "r1", Rules: []CommandRule{first}},
		writeErr: ErrRulesConflict,
	}
	arbiter := newRuleArbiter(t, source)
	ctx := context.Background()
	if _, err := arbiter.LoadRules(ctx); err != nil {
		t.Fatalf("读取规则: %v", err)
	}

	if _, err := arbiter.AddRule(ctx, allowRule("printf two", CommandMatchExact)); !errors.Is(err, ErrRulesConflict) {
		t.Fatalf("冲突应报 ErrRulesConflict: %v", err)
	}
	cached, ok := arbiter.Rules()
	if !ok || cached.Revision != "r1" || len(cached.Rules) != 1 || cached.Rules[0] != first {
		t.Fatalf("冲突后不应覆盖缓存: %+v ok=%v", cached, ok)
	}
	reads, writes := source.counts()
	if reads != 1 || writes != 1 {
		t.Fatalf("冲突不应自动重试: reads=%d writes=%d", reads, writes)
	}
}

// TestRulesUnavailableMakesZeroRequests 核对：能力缺失后不再发请求。
func TestRulesUnavailableMakesZeroRequests(t *testing.T) {
	source := &fakeRuleSource{readErr: ErrRulesUnavailable}
	arbiter := newRuleArbiter(t, source)
	ctx := context.Background()

	if _, err := arbiter.LoadRules(ctx); !errors.Is(err, ErrRulesUnavailable) {
		t.Fatalf("应报能力缺失: %v", err)
	}
	if !arbiter.RulesUnavailable() {
		t.Fatal("应记住能力缺失")
	}
	if _, err := arbiter.AddRule(ctx, allowRule("printf one", CommandMatchExact)); !errors.Is(err, ErrRulesUnavailable) {
		t.Fatalf("能力缺失后新增应被拒: %v", err)
	}
	if _, err := arbiter.EditRule(ctx, 0, allowRule("printf one", CommandMatchExact)); !errors.Is(err, ErrRulesUnavailable) {
		t.Fatalf("能力缺失后编辑应被拒: %v", err)
	}
	if _, err := arbiter.RevokeRule(ctx, 0); !errors.Is(err, ErrRulesUnavailable) {
		t.Fatalf("能力缺失后撤销应被拒: %v", err)
	}
	reads, writes := source.counts()
	if reads != 1 || writes != 0 {
		t.Fatalf("能力缺失后不应再发请求: reads=%d writes=%d", reads, writes)
	}
	if _, ok := arbiter.Rules(); ok {
		t.Fatal("能力缺失时不应有规则视图")
	}

	// 没有规则来源时同样按能力缺失处理。
	bare := newRuleArbiter(t, nil)
	if _, err := bare.LoadRules(ctx); !errors.Is(err, ErrRulesUnavailable) {
		t.Fatalf("没有来源应报能力缺失: %v", err)
	}
	if !bare.RulesUnavailable() {
		t.Fatal("没有来源时应标记能力缺失")
	}
}

// TestRuleValidationAndIndex 核对：坏规则与越界下标不发写请求。
func TestRuleValidationAndIndex(t *testing.T) {
	source := &fakeRuleSource{readSet: RuleSet{
		Revision: "r1",
		Rules:    []CommandRule{allowRule("printf one", CommandMatchExact)},
	}}
	arbiter := newRuleArbiter(t, source)
	ctx := context.Background()

	invalid := []CommandRule{
		{Command: "", Match: CommandMatchExact, Decision: CommandDecisionAllow},
		{Command: "printf one; rm", Match: CommandMatchExact, Decision: CommandDecisionAllow},
		{Command: "printf one", Match: "fuzzy", Decision: CommandDecisionAllow},
		{Command: "printf one", Match: CommandMatchExact, Decision: "ask"},
	}
	for _, rule := range invalid {
		if _, err := arbiter.AddRule(ctx, rule); !errors.Is(err, ErrRuleInvalid) {
			t.Fatalf("坏规则应被拒 %+v: %v", rule, err)
		}
	}
	if _, err := arbiter.LoadRules(ctx); err != nil {
		t.Fatalf("读取规则: %v", err)
	}
	if _, err := arbiter.EditRule(ctx, 5, allowRule("printf two", CommandMatchExact)); !errors.Is(err, ErrRuleIndex) {
		t.Fatalf("越界编辑应被拒: %v", err)
	}
	if _, err := arbiter.RevokeRule(ctx, -1); !errors.Is(err, ErrRuleIndex) {
		t.Fatalf("越界撤销应被拒: %v", err)
	}
	reads, writes := source.counts()
	if reads != 1 || writes != 0 {
		t.Fatalf("坏输入不应产生写请求: reads=%d writes=%d", reads, writes)
	}
}

// TestRuleSourceErrorsArePropagated 核对：普通读写错误原样返回，不记为能力缺失。
func TestRuleSourceErrorsArePropagated(t *testing.T) {
	boom := errors.New("boom")
	source := &fakeRuleSource{readErr: boom}
	arbiter := newRuleArbiter(t, source)
	if _, err := arbiter.LoadRules(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("读取错误应原样返回: %v", err)
	}
	if arbiter.RulesUnavailable() {
		t.Fatal("普通错误不应标记能力缺失")
	}
}

// TestRulesDoNotUseWriteRegistry 核对：命令规则读写不碰写登记。
func TestRulesDoNotUseWriteRegistry(t *testing.T) {
	source := &fakeRuleSource{readSet: RuleSet{Revision: "r1"}}
	writer := newFakeWriter()
	arbiter, err := New(Config{SessionID: testSession, Writer: writer, Rules: source})
	if err != nil {
		t.Fatalf("创建交互裁决: %v", err)
	}
	if _, err := arbiter.LoadRules(context.Background()); err != nil {
		t.Fatalf("读取规则: %v", err)
	}
	if _, err := arbiter.AddRule(context.Background(), allowRule("printf one", CommandMatchExact)); err != nil {
		t.Fatalf("新增规则: %v", err)
	}
	if writer.count() != 0 {
		t.Fatalf("规则管理不应登记写意图: %d", writer.count())
	}
}
