package interaction

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// CommandMatch 是命令规则的匹配方式。取值与服务端一致。
type CommandMatch string

const (
	// CommandMatchExact 表示整条命令完全一致。
	CommandMatchExact CommandMatch = "exact"
	// CommandMatchPrefix 表示命令的前几个词一致。
	CommandMatchPrefix CommandMatch = "prefix"
)

// CommandDecision 是命令规则的决定。取值与服务端一致。
type CommandDecision string

const (
	// CommandDecisionAllow 表示允许匹配的命令。
	CommandDecisionAllow CommandDecision = "allow"
	// CommandDecisionDeny 表示拒绝匹配的命令。
	CommandDecisionDeny CommandDecision = "deny"
)

// CommandRule 是一条用户级命令规则。字段是归一化后的业务字段，wire 形状由
// 连接/集成层负责。
type CommandRule struct {
	Command  string
	Match    CommandMatch
	Decision CommandDecision
}

// RuleSet 是一次命令规则读取或写入回执的只读视图。Revision 是服务端给出的
// 不透明版本号；Rules 是规则集合。
type RuleSet struct {
	Revision string
	Rules    []CommandRule
}

// RuleSource 是命令规则的读取与写入入口。集成层实现它，把 permission.rules 与
// permission.set_rules 的 wire 形状留在连接侧。
type RuleSource interface {
	ReadRules(ctx context.Context) (RuleSet, error)
	// WriteRules 用 expectedRevision 作为前置条件重写整个规则集合。
	WriteRules(ctx context.Context, expectedRevision string, rules []CommandRule) (RuleSet, error)
}

// 命令规则相关错误。
var (
	// ErrRulesUnavailable 表示服务端没有命令规则能力，或调用方没有提供规则来源。
	ErrRulesUnavailable = errors.New("命令规则能力缺失")
	// ErrRulesConflict 表示规则集合的 revision 已过期；写入未发生，本地缓存不覆盖。
	ErrRulesConflict = errors.New("命令规则版本冲突")
	// ErrRuleInvalid 表示规则字段不合法。
	ErrRuleInvalid = errors.New("命令规则不合法")
	// ErrRuleIndex 表示编辑或撤销的规则下标越界。
	ErrRuleIndex = errors.New("命令规则下标越界")
)

// LoadRules 读取一次命令规则并更新本地只读缓存。Revision 相同时保留缓存，
// Revision 变化时整体替换；缓存只在内存中，不落盘。
func (a *Arbiter) LoadRules(ctx context.Context) (RuleSet, error) {
	if err := a.rulesAvailable(); err != nil {
		return RuleSet{}, err
	}
	set, err := a.ruleSource.ReadRules(nonNilContext(ctx))
	if err != nil {
		a.markRulesError(err)
		return RuleSet{}, err
	}
	set = cloneRuleSet(set)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.applyRuleSetLocked(set)
	return cloneRuleSet(a.ruleCache), nil
}

// Rules 返回命令规则缓存的只读拷贝；从未读取或能力缺失时第二个返回值为 false。
func (a *Arbiter) Rules() (RuleSet, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.ruleCacheKnown || a.ruleCacheUnavailable {
		return RuleSet{}, false
	}
	return cloneRuleSet(a.ruleCache), true
}

// RulesUnavailable 报告本会话是否已确认没有命令规则能力。确认后不再发请求。
func (a *Arbiter) RulesUnavailable() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ruleCacheUnavailable
}

// AddRule 在缓存规则集合末尾追加一条规则，并以重写整个集合的方式写入。
// 完全相同的规则已存在时直接返回缓存，不产生写请求。
func (a *Arbiter) AddRule(ctx context.Context, rule CommandRule) (RuleSet, error) {
	normalized, err := normalizeCommandRule(rule)
	if err != nil {
		return RuleSet{}, err
	}
	current, err := a.ensureRules(ctx)
	if err != nil {
		return RuleSet{}, err
	}
	for _, existing := range current.Rules {
		if existing == normalized {
			return current, nil
		}
	}
	next := cloneRules(current.Rules)
	next = append(next, normalized)
	return a.writeRules(ctx, current.Revision, next)
}

// EditRule 用新规则替换缓存里的第 index 条规则，并重写整个集合。
func (a *Arbiter) EditRule(ctx context.Context, index int, rule CommandRule) (RuleSet, error) {
	normalized, err := normalizeCommandRule(rule)
	if err != nil {
		return RuleSet{}, err
	}
	current, err := a.ensureRules(ctx)
	if err != nil {
		return RuleSet{}, err
	}
	if index < 0 || index >= len(current.Rules) {
		return RuleSet{}, fmt.Errorf("%w: %d（共 %d 条）", ErrRuleIndex, index, len(current.Rules))
	}
	next := cloneRules(current.Rules)
	if next[index] == normalized {
		return current, nil
	}
	next[index] = normalized
	return a.writeRules(ctx, current.Revision, next)
}

// RevokeRule 删除缓存里的第 index 条规则，并重写整个集合。
func (a *Arbiter) RevokeRule(ctx context.Context, index int) (RuleSet, error) {
	current, err := a.ensureRules(ctx)
	if err != nil {
		return RuleSet{}, err
	}
	if index < 0 || index >= len(current.Rules) {
		return RuleSet{}, fmt.Errorf("%w: %d（共 %d 条）", ErrRuleIndex, index, len(current.Rules))
	}
	next := cloneRules(current.Rules)
	next = append(next[:index], next[index+1:]...)
	return a.writeRules(ctx, current.Revision, next)
}

// ensureRules 返回可用的缓存；缓存为空时先读取一次。能力缺失后不再发请求。
func (a *Arbiter) ensureRules(ctx context.Context) (RuleSet, error) {
	a.mu.Lock()
	if a.ruleCacheUnavailable {
		a.mu.Unlock()
		return RuleSet{}, ErrRulesUnavailable
	}
	if a.ruleCacheKnown {
		current := cloneRuleSet(a.ruleCache)
		a.mu.Unlock()
		return current, nil
	}
	a.mu.Unlock()
	return a.LoadRules(ctx)
}

// writeRules 以 expectedRevision 为前置条件重写规则集合；冲突或能力缺失时不覆盖缓存。
func (a *Arbiter) writeRules(ctx context.Context, expectedRevision string, rules []CommandRule) (RuleSet, error) {
	if err := a.rulesAvailable(); err != nil {
		return RuleSet{}, err
	}
	set, err := a.ruleSource.WriteRules(nonNilContext(ctx), expectedRevision, cloneRules(rules))
	if err != nil {
		a.markRulesError(err)
		return RuleSet{}, err
	}
	set = cloneRuleSet(set)
	// permission.set_rules 的真实回执只带 revision；规则集合以本次写入的集合为准。
	if len(set.Rules) == 0 && len(rules) > 0 {
		set.Rules = cloneRules(rules)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.applyRuleSetLocked(set)
	return cloneRuleSet(a.ruleCache), nil
}

// rulesAvailable 判断规则来源是否可用；没有来源时按能力缺失处理，且不发请求。
func (a *Arbiter) rulesAvailable() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ruleSource == nil || a.ruleCacheUnavailable {
		a.ruleCacheUnavailable = true
		return ErrRulesUnavailable
	}
	return nil
}

// markRulesError 把能力缺失记入缓存；版本冲突只返回错误，不改变缓存。
func (a *Arbiter) markRulesError(err error) {
	if !errors.Is(err, ErrRulesUnavailable) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ruleCacheUnavailable = true
}

// applyRuleSetLocked 按 revision 更新缓存：revision 相同则保留，不同则整体替换。
// 调用方持锁。
func (a *Arbiter) applyRuleSetLocked(set RuleSet) {
	if a.ruleCacheKnown && a.ruleCache.Revision == set.Revision {
		return
	}
	a.ruleCache = set
	a.ruleCacheKnown = true
	a.ruleCacheUnavailable = false
}

// normalizeCommandRule 校验并规范化一条规则；只做结构校验，不解释命令语义。
func normalizeCommandRule(rule CommandRule) (CommandRule, error) {
	rule.Command = strings.TrimSpace(rule.Command)
	if rule.Command == "" || !isSimpleCommand(rule.Command) {
		return CommandRule{}, fmt.Errorf("%w: 命令为空或含分隔符", ErrRuleInvalid)
	}
	switch rule.Match {
	case CommandMatchExact, CommandMatchPrefix:
	default:
		return CommandRule{}, fmt.Errorf("%w: 未知匹配方式 %q", ErrRuleInvalid, rule.Match)
	}
	switch rule.Decision {
	case CommandDecisionAllow, CommandDecisionDeny:
	default:
		return CommandRule{}, fmt.Errorf("%w: 未知决定 %q", ErrRuleInvalid, rule.Decision)
	}
	return rule, nil
}

// isSimpleCommand 判断命令是否没有服务端定义的命令分隔符。
func isSimpleCommand(command string) bool {
	if strings.TrimSpace(command) == "" {
		return false
	}
	return !strings.ContainsAny(command, ";&|`()<>$\n\r")
}

func cloneRuleSet(set RuleSet) RuleSet {
	return RuleSet{Revision: set.Revision, Rules: cloneRules(set.Rules)}
}

func cloneRules(rules []CommandRule) []CommandRule {
	if len(rules) == 0 {
		return []CommandRule{}
	}
	return append([]CommandRule(nil), rules...)
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
