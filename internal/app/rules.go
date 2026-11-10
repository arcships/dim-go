package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/interaction"
)

// 命令规则在服务端上的方法（alpha.21 实测，见 D1 §3.9 与 D11a）。
const (
	methodPermissionReadRules  = "permission.rules"
	methodPermissionWriteRules = "permission.set_rules"
)

// conflictAppError 是规则版本冲突的稳定错误码。
const conflictAppError = "permission_rules_conflict"

// ruleSource 实现 interaction.RuleSource：permission.rules 读取、permission.set_rules 写入。
//
// 只读缓存由交互裁决持有，本适配器不缓存、不落盘；版本冲突原样交回领域分类，
// 不做重试。
type ruleSource struct {
	app *App
}

func (s *ruleSource) ReadRules(ctx context.Context) (interaction.RuleSet, error) {
	if !s.app.controlSupports(methodPermissionReadRules) {
		return interaction.RuleSet{}, fmt.Errorf("%w: 服务端未声明 %s",
			interaction.ErrRulesUnavailable, methodPermissionReadRules)
	}
	client, err := s.app.requireControl("读取命令规则")
	if err != nil {
		return interaction.RuleSet{}, err
	}
	raw, err := client.Call(ctx, methodPermissionReadRules, map[string]any{})
	if err != nil {
		return interaction.RuleSet{}, classifyRulesError(methodPermissionReadRules, err)
	}
	reply, err := parseRuleSet(raw)
	if err != nil {
		return interaction.RuleSet{}, err
	}
	return reply, nil
}

func (s *ruleSource) WriteRules(
	ctx context.Context,
	expectedRevision string,
	rules []interaction.CommandRule,
) (interaction.RuleSet, error) {
	if !s.app.controlSupports(methodPermissionWriteRules) {
		return interaction.RuleSet{}, fmt.Errorf("%w: 服务端未声明 %s",
			interaction.ErrRulesUnavailable, methodPermissionWriteRules)
	}
	client, err := s.app.requireControl("写入命令规则")
	if err != nil {
		return interaction.RuleSet{}, err
	}
	wireRules := make([]commandRule, 0, len(rules))
	for _, rule := range rules {
		wireRules = append(wireRules, commandRule{
			Command:  rule.Command,
			Match:    string(rule.Match),
			Decision: string(rule.Decision),
		})
	}
	raw, err := client.Call(ctx, methodPermissionWriteRules, map[string]any{
		"expectedRevision": expectedRevision,
		"commandRules":     wireRules,
	})
	if err != nil {
		return interaction.RuleSet{}, classifyRulesError(methodPermissionWriteRules, err)
	}
	// 真实回执只带 revision；规则集合以本次写入的集合为准。
	var reply struct {
		Revision     string        `json:"revision"`
		CommandRules []commandRule `json:"commandRules"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &reply); err != nil {
			return interaction.RuleSet{}, fmt.Errorf("命令规则写入回执无法解析: %w", err)
		}
	}
	set := interaction.RuleSet{
		Revision: reply.Revision,
		Rules:    append([]interaction.CommandRule(nil), rules...),
	}
	if len(reply.CommandRules) > 0 {
		set.Rules = interactionRules(reply.CommandRules)
	}
	if set.Revision == "" {
		// 没有 revision 时沿用服务端已知的旧值会掩盖冲突；交回空 revision 由领域
		// 决定下一次读取，而不是伪造版本。
		set.Revision = expectedRevision
	}
	return set, nil
}

// parseRuleSet 把 permission.rules 回执映射成领域视图。
func parseRuleSet(raw json.RawMessage) (interaction.RuleSet, error) {
	var reply commandRulesReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		return interaction.RuleSet{}, fmt.Errorf("命令规则回执无法解析: %w", err)
	}
	return interaction.RuleSet{
		Revision: reply.Revision,
		Rules:    interactionRules(reply.CommandRules),
	}, nil
}

// interactionRules 把 wire 规则映射成领域规则；字段原样保留，语义由领域解释。
func interactionRules(rules []commandRule) []interaction.CommandRule {
	mapped := make([]interaction.CommandRule, 0, len(rules))
	for _, rule := range rules {
		mapped = append(mapped, interaction.CommandRule{
			Command:  rule.Command,
			Match:    interaction.CommandMatch(rule.Match),
			Decision: interaction.CommandDecision(rule.Decision),
		})
	}
	return mapped
}

// classifyRulesError 把版本冲突与能力缺失映射成领域错误；其余错误原样透传。
func classifyRulesError(method string, err error) error {
	var rpcErr *protocol.RPCError
	if !errors.As(err, &rpcErr) {
		return err
	}
	switch rpcErr.AppError() {
	case conflictAppError:
		// 冲突：写入未发生，本地缓存不覆盖；不重试。
		return fmt.Errorf("%w: %s: %v", interaction.ErrRulesConflict, method, err)
	case "unknown_method", "method_unavailable", "management_unavailable":
		return fmt.Errorf("%w: %s: %v", interaction.ErrRulesUnavailable, method, err)
	default:
		return err
	}
}
