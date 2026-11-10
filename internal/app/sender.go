package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/extensions"
	"github.com/arcships/dimagent-mygo/internal/settings"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// 写意图到服务端方法的映射。方法名与参数形状只出现在这里，来源是各领域文档与
// 各领域在假服务端上的集成证据：
//
//	write.KindSubmitInput      → session.submit        参数 {sessionId, branchId?, mode, submissionId, text}
//	write.KindStop             → session.stop          参数 {sessionId, branchId?, memberId?}
//	write.KindApprovalResponse → approval.respond      参数 {sessionId, branchId?, requestId, decision: allow|deny}
//	write.KindLongTermAllow    → permission.rules 读、permission.set_rules 写命令规则；
//	                             规则写成功后才由交互裁决发 approval.respond（allow）
//	write.KindQuestionAnswer   → question.respond      参数 {sessionId, branchId?, requestId, decision, answer?}
//	write.KindManagementWrite  → 按参数里的 op 分发：plugin.*、skill.*、hook.write、
//	                             tool_server.*、workflow.*、wakeup.cancel（扩展与后台领域），
//	                             settings.update、provider.select、runtime.install（设置领域）
//	write.KindAccountAction    → account.<op>          参数沿用登录、登出与重置的现有形状
//	write.KindAttachmentWrite  → attachment.put        参数 {sessionId, name?, text}；走会话流，
//	                             先按会话方法清单判定能力
//	write.KindSessionLifecycle → session.create / session.rename / session.delete / session.branch_create
//	                             / session.branch_end，按参数里的 op
//	write.KindShellRun         → shell.run / shell.stop    参数 {sessionId, branchId, command?, runId?}
//	write.KindModalityWrite    → modality.task.submit / modality.task.cancel /
//	                             modality.realtime.open / modality.realtime.close，参数 {sessionId, branchId, <载荷>}
//
// 会话贡献方法（attachment.put、shell.*、modality.*、session.branch_*）先按 hello 能力与
// 会话方法清单判定，缺方法即报能力缺失、不发请求。所有写意图在发送前都过一次写闸门。
const (
	methodSubmitInput        = "session.submit"
	methodStop               = "session.stop"
	methodApprovalResponse   = "approval.respond"
	methodQuestionAnswer     = "question.respond"
	methodPermissionRules    = "permission.rules"
	methodPermissionSetRule  = "permission.set_rules"
	methodAttachmentWrite    = protocol.MethodAttachmentPut
	methodSessionCreate      = "session.create"
	methodSessionRename      = "session.rename"
	methodSessionDelete      = "session.delete"
	methodBranchCreate       = "session.branch_create"
	methodBranchEnd          = "session.branch_end"
	methodShellRun           = "shell.run"
	methodShellStop          = "shell.stop"
	methodModalityTaskSubmit = "modality.task.submit"
	methodModalityTaskCancel = "modality.task.cancel"
	methodModalityOpen       = "modality.realtime.open"
	methodModalityClose      = "modality.realtime.close"
)

// mainBranchID 是主线标识。会话流与投影都固定按主线工作；意图没有指定分支时
// 按主线下发，服务端在 v2/v3 上要求 branchId 必须存在。
const mainBranchID = "main"

// appSender 是写登记的发送方：按写意图类型选方法，合入会话、分支、投递方式与输入
// 标识，只调用一次，错误原样返回交给写登记分类。
type appSender struct {
	app *App
}

// Send 实现 write.Sender。
//
// 发送前先过写闸门：未就绪、未连接或正在退出时一律拒绝，请求不会发出。
func (s *appSender) Send(ctx context.Context, req write.Request) (write.Reply, error) {
	if err := s.app.Writable(); err != nil {
		kind := failure.KindNonRetryable
		if errors.Is(err, ErrNotConnected) {
			kind = failure.KindConnectionMissing
		}
		return write.Reply{}, failure.New(kind, "发送写意图 "+string(req.Kind), err)
	}
	client := s.app.control()
	if client == nil {
		return write.Reply{}, failure.New(failure.KindConnectionMissing,
			"发送写意图 "+string(req.Kind), ErrNotConnected)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	switch req.Kind {
	case write.KindSubmitInput:
		return s.sendSubmit(ctx, client, req)
	case write.KindStop:
		return s.sendWithRequestID(ctx, client, methodStop, req, false)
	case write.KindApprovalResponse:
		return s.sendApprovalResponse(ctx, client, req)
	case write.KindLongTermAllow:
		return s.sendLongTermAllow(ctx, client, req)
	case write.KindQuestionAnswer:
		return s.sendWithRequestID(ctx, client, methodQuestionAnswer, req, true)
	case write.KindManagementWrite:
		return s.sendManagement(ctx, client, req)
	case write.KindAccountAction:
		return s.sendAccountAction(ctx, client, req)
	case write.KindAttachmentWrite:
		return s.sendAttachment(ctx, client, req)
	case write.KindSessionLifecycle:
		return s.sendSessionLifecycle(ctx, client, req)
	case write.KindShellRun:
		return s.sendShellRun(ctx, req)
	case write.KindModalityWrite:
		return s.sendModality(ctx, req)
	default:
		return write.Reply{}, fmt.Errorf("%w: 未知写意图类型 %q", write.ErrInvalidIntent, req.Kind)
	}
}

// sendSubmit 发一次提交。mode 取投递方式；submissionId 是服务端去重标识。
func (s *appSender) sendSubmit(
	ctx context.Context,
	client *protocol.Client,
	req write.Request,
) (write.Reply, error) {
	if req.SessionID == "" {
		return write.Reply{}, fmt.Errorf("%w: 提交缺少会话标识", write.ErrInvalidIntent)
	}
	params, err := objectParams(req)
	if err != nil {
		return write.Reply{}, err
	}
	params["sessionId"] = req.SessionID
	params["branchId"] = branchWireValue(req.BranchID)
	if req.Delivery != "" {
		params["mode"] = deliveryWireValue(req.Delivery)
	}
	if req.InputID != "" {
		params["submissionId"] = req.InputID
	}
	raw, err := client.Call(ctx, methodSubmitInput, params)
	if err != nil {
		return write.Reply{}, err
	}
	reply := write.Reply{InputID: req.InputID, Result: raw}
	var result struct {
		InputID   string `json:"inputId"`
		SessionID string `json:"sessionId"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &result); err == nil && result.InputID != "" {
			reply.InputID = result.InputID
		}
	}
	return reply, nil
}

// sendWithRequestID 发一次带请求标识的写入（停止、审批应答、长期允许、提问应答）。
// withRequestID 为真时把意图的输入标识作为 requestId 下发。
func (s *appSender) sendWithRequestID(
	ctx context.Context,
	client *protocol.Client,
	method string,
	req write.Request,
	withRequestID bool,
) (write.Reply, error) {
	if req.SessionID == "" && !strings.HasPrefix(method, "question.") {
		return write.Reply{}, fmt.Errorf("%w: %s 缺少会话标识", write.ErrInvalidIntent, method)
	}
	params, err := objectParams(req)
	if err != nil {
		return write.Reply{}, err
	}
	if req.SessionID != "" {
		params["sessionId"] = req.SessionID
		params["branchId"] = branchWireValue(req.BranchID)
	}
	if withRequestID && req.InputID != "" {
		params["requestId"] = req.InputID
	}
	raw, err := client.Call(ctx, method, params)
	if err != nil {
		return write.Reply{}, err
	}
	return write.Reply{InputID: req.InputID, Result: raw}, nil
}

// sendApprovalResponse 发一次审批应答。交互裁决使用 approve/reject，服务端只认
// allow/deny；映射只发生在这一层。
func (s *appSender) sendApprovalResponse(
	ctx context.Context,
	client *protocol.Client,
	req write.Request,
) (write.Reply, error) {
	if req.SessionID == "" || req.InputID == "" {
		return write.Reply{}, fmt.Errorf("%w: 审批应答缺少会话或请求标识", write.ErrInvalidIntent)
	}
	params, err := objectParams(req)
	if err != nil {
		return write.Reply{}, err
	}
	decision, _ := params["decision"].(string)
	switch decision {
	case "approve", "allow", "allow_long_term":
		params["decision"] = "allow"
	case "reject", "deny":
		params["decision"] = "deny"
	default:
		return write.Reply{}, fmt.Errorf("%w: 未知审批决定 %q", write.ErrInvalidIntent, decision)
	}
	params["sessionId"] = req.SessionID
	params["branchId"] = branchWireValue(req.BranchID)
	params["requestId"] = req.InputID
	raw, err := client.Call(ctx, methodApprovalResponse, params)
	if err != nil {
		return write.Reply{}, err
	}
	return write.Reply{InputID: req.InputID, Result: raw}, nil
}

// commandRule 是服务端用户级命令规则；字段与服务端 wire 一致。
type commandRule struct {
	Command  string `json:"command"`
	Match    string `json:"match"`
	Decision string `json:"decision"`
}

type commandRulesReply struct {
	Revision     string        `json:"revision"`
	CommandRules []commandRule `json:"commandRules"`
}

// sendLongTermAllow 把长期允许写成服务端命令规则：先读 permission.rules，再带
// revision 写 permission.set_rules；revision 冲突时重读重试。规则写成功后交回
// 交互裁决，由它发审批应答。
func (s *appSender) sendLongTermAllow(
	ctx context.Context,
	client *protocol.Client,
	req write.Request,
) (write.Reply, error) {
	params, err := objectParams(req)
	if err != nil {
		return write.Reply{}, err
	}
	rule, err := commandRuleFromParams(params)
	if err != nil {
		return write.Reply{}, err
	}
	for attempt := 0; attempt < 3; attempt++ {
		current, err := readCommandRules(ctx, client)
		if err != nil {
			return write.Reply{}, err
		}
		if hasCommandRule(current.CommandRules, rule) {
			return write.Reply{Result: json.RawMessage(`{"status":"present"}`)}, nil
		}
		next := append(append([]commandRule{}, current.CommandRules...), rule)
		raw, err := client.Call(ctx, methodPermissionSetRule, map[string]any{
			"expectedRevision": current.Revision,
			"commandRules":     next,
		})
		if err == nil {
			return write.Reply{Result: raw}, nil
		}
		if appErrorOf(err) != "permission_rules_conflict" {
			return write.Reply{}, err
		}
	}
	return write.Reply{}, errors.New("命令规则持续变化，长期允许未写入")
}

func readCommandRules(ctx context.Context, client *protocol.Client) (commandRulesReply, error) {
	var reply commandRulesReply
	raw, err := client.Call(ctx, methodPermissionRules, map[string]any{})
	if err != nil {
		return reply, err
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return reply, fmt.Errorf("permission.rules 回执无法解析: %w", err)
	}
	if reply.Revision == "" {
		return reply, errors.New("permission.rules 回执缺少 revision")
	}
	return reply, nil
}

func commandRuleFromParams(params map[string]any) (commandRule, error) {
	command, _ := params["command"].(string)
	command = strings.TrimSpace(command)
	match, _ := params["match"].(string)
	if match == "" {
		match = "exact"
	}
	decision, _ := params["decision"].(string)
	if decision == "" {
		decision = "allow"
	}
	if command == "" || (match != "exact" && match != "prefix") || decision != "allow" {
		return commandRule{}, fmt.Errorf("%w: 长期允许规则不合法", write.ErrInvalidIntent)
	}
	return commandRule{Command: command, Match: match, Decision: decision}, nil
}

func hasCommandRule(rules []commandRule, target commandRule) bool {
	for _, rule := range rules {
		if rule == target {
			return true
		}
	}
	return false
}

func appErrorOf(err error) string {
	var rpcErr *protocol.RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr.AppError()
	}
	return ""
}

// sendManagement 按参数里的 op 分发管理写入。
func (s *appSender) sendManagement(
	ctx context.Context,
	client *protocol.Client,
	req write.Request,
) (write.Reply, error) {
	var head struct {
		Op string `json:"op"`
	}
	if err := json.Unmarshal(req.Params, &head); err != nil {
		return write.Reply{}, fmt.Errorf("%w: 管理写入参数无法解析: %v", write.ErrInvalidIntent, err)
	}
	switch {
	case isExtensionOp(head.Op):
		return s.sendExtensionWrite(ctx, client, req)
	case head.Op == "settings.update":
		var payload struct {
			Source           string          `json:"source"`
			Scope            settings.Scope  `json:"scope"`
			ExpectedRevision string          `json:"expectedRevision"`
			Document         json.RawMessage `json:"document"`
		}
		if err := json.Unmarshal(req.Params, &payload); err != nil {
			return write.Reply{}, fmt.Errorf("%w: 设置写入参数无法解析: %v", write.ErrInvalidIntent, err)
		}
		raw, err := client.Call(ctx, "settings.update", map[string]any{
			"source":           payload.Source,
			"scope":            scopeWireValue(payload.Scope),
			"expectedRevision": payload.ExpectedRevision,
			"document":         payload.Document,
		})
		if err != nil {
			return write.Reply{}, err
		}
		return write.Reply{Result: raw}, nil
	case head.Op == "provider.select":
		var payload struct {
			ProviderID       string `json:"providerId"`
			ModelID          string `json:"modelId"`
			ExpectedRevision string `json:"expectedRevision"`
		}
		if err := json.Unmarshal(req.Params, &payload); err != nil {
			return write.Reply{}, fmt.Errorf("%w: 模型选择参数无法解析: %v", write.ErrInvalidIntent, err)
		}
		raw, err := client.Call(ctx, "provider.select", map[string]any{
			"providerId":      payload.ProviderID,
			"modelId":         payload.ModelID,
			"expectedVersion": payload.ExpectedRevision,
		})
		if err != nil {
			return write.Reply{}, err
		}
		return write.Reply{Result: raw}, nil
	case head.Op == "runtime.install":
		// runtime.* 在 alpha.21 不注册：按能力缺失处理，不发请求、不发明方法。
		return write.Reply{}, runtimeCapabilityMissing()
	default:
		return write.Reply{}, fmt.Errorf("%w: 未支持的管理写入 %q", write.ErrInvalidIntent, head.Op)
	}
}

func isExtensionOp(op string) bool {
	switch {
	case strings.HasPrefix(op, "plugin."),
		strings.HasPrefix(op, "skill."),
		strings.HasPrefix(op, "skills."),
		strings.HasPrefix(op, "hook."),
		strings.HasPrefix(op, "hooks."),
		strings.HasPrefix(op, "mcp."),
		strings.HasPrefix(op, "tool_server."),
		strings.HasPrefix(op, "workflow."),
		op == "wakeup.cancel",
		op == "session.wakeup.cancel":
		return true
	default:
		return false
	}
}

// sendExtensionWrite 把扩展与后台领域的写意图载荷映射成服务端方法。
func (s *appSender) sendExtensionWrite(
	ctx context.Context,
	client *protocol.Client,
	req write.Request,
) (write.Reply, error) {
	managed, err := extensions.DecodeManagementWrite(req.Params)
	if err != nil {
		return write.Reply{}, fmt.Errorf("%w: %v", write.ErrInvalidIntent, err)
	}
	// 领域包的 WriteOp 取值在两轮 wire 对齐中变过（toolServer.* → mcp.*、scheduler.* →
	// schedule.* 等）。这里按 wire 取值分发，同时接受旧写法，避免把集成层绑在
	// 领域包内部的常量演进上。
	switch string(managed.Op) {
	case "plugin.install":
		var payload struct {
			Source extensions.InstallSource `json:"source"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		return callAndReply(ctx, client, methodPluginInstall, map[string]any{
			"source": pluginSourceWire(payload.Source),
			"enable": true,
		})
	case "plugin.enable", "plugin.set_enabled", "plugin.disable":
		var payload struct {
			PluginID string `json:"pluginId"`
			Name     string `json:"name"`
			Enabled  *bool  `json:"enabled"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		enabled := true
		if payload.Enabled != nil {
			enabled = *payload.Enabled
		}
		if string(managed.Op) == "plugin.disable" {
			enabled = false
		}
		method := methodPluginEnable
		if !enabled {
			method = methodPluginDisable
		}
		return callAndReply(ctx, client, method, map[string]any{
			"name": firstNonEmpty(payload.PluginID, payload.Name),
		})
	case "plugin.update":
		var payload struct {
			PluginID string `json:"pluginId"`
			ID       string `json:"id"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		return callAndReply(ctx, client, methodPluginUpdate, map[string]any{
			"id": firstNonEmpty(payload.PluginID, payload.ID),
		})
	case "skills.install", "skill.install":
		var payload struct {
			Source extensions.InstallSource `json:"source"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		return callAndReply(ctx, client, methodSkillsInstall, protocol.SkillsInstallParams{
			Source: pluginSourceWire(payload.Source),
			Skills: []protocol.SelectedSkill{},
		})
	case "skills.set_enabled", "skill.set_enabled":
		var payload struct {
			SkillID string `json:"skillId"`
			ID      string `json:"id"`
			Enabled *bool  `json:"enabled"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		enabled := true
		if payload.Enabled != nil {
			enabled = *payload.Enabled
		}
		return callAndReply(ctx, client, methodSkillsSetEnabled, protocol.SkillsSetEnabledParams{
			ID:      firstNonEmpty(payload.SkillID, payload.ID),
			Enabled: enabled,
		})
	case "hooks.save", "hook.write":
		var payload struct {
			Hook             extensions.Hook `json:"hook"`
			ExpectedRevision string          `json:"expectedRevision"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		revision, err := s.hooksRevision(ctx, client, payload.ExpectedRevision)
		if err != nil {
			return write.Reply{}, err
		}
		document, marshalErr := json.Marshal(map[string]any{"hooks": []any{map[string]any{
			"id":      payload.Hook.ID,
			"name":    payload.Hook.Name,
			"event":   payload.Hook.Event,
			"command": payload.Hook.Command,
			"source":  string(payload.Hook.Source),
			"enabled": payload.Hook.Enabled,
		}}})
		if marshalErr != nil {
			return write.Reply{}, invalidPayload(marshalErr)
		}
		return callAndReply(ctx, client, methodHooksSave, protocol.HooksSaveParams{
			ExpectedRevision: revision,
			Document:         document,
		})
	case "mcp.import.apply":
		var payload struct {
			ImportSource string   `json:"importSource"`
			Source       string   `json:"source"`
			CandidateKey string   `json:"candidateKey"`
			IDs          []string `json:"ids"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		ids := append([]string(nil), payload.IDs...)
		if len(ids) == 0 && payload.CandidateKey != "" {
			ids = []string{payload.CandidateKey}
		}
		return callAndReply(ctx, client, methodMCPImportApply, protocol.MCPImportApplyParams{
			Source: firstNonEmpty(payload.ImportSource, payload.Source),
			IDs:    ids,
		})
	case "mcp.save", "tool_server.add", "tool_server.update":
		var payload struct {
			ServerID     string                    `json:"serverId"`
			Server       extensions.ToolServerSpec `json:"server"`
			ImportSource string                    `json:"importSource"`
			CandidateKey string                    `json:"candidateKey"`
			Source       string                    `json:"source"`
			ExpectedVer  string                    `json:"expectedVersion"`
			Config       json.RawMessage           `json:"configuration"`
			Name         string                    `json:"name"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		if payload.ImportSource != "" && payload.CandidateKey != "" {
			return callAndReply(ctx, client, methodMCPImportApply, protocol.MCPImportApplyParams{
				Source: payload.ImportSource,
				IDs:    []string{payload.CandidateKey},
			})
		}
		spec := payload.Server
		if spec.Name == "" {
			spec.Name = firstNonEmpty(payload.Name, payload.ServerID)
		}
		if len(spec.Configuration) == 0 && len(payload.Config) > 0 {
			spec.Configuration = payload.Config
		}
		if spec.Source == "" && payload.Source != "" {
			spec.Source = extensions.Source(payload.Source)
		}
		if spec.ExpectedVersion == "" {
			spec.ExpectedVersion = payload.ExpectedVer
		}
		return callAndReply(ctx, client, methodMCPSave, mcpSaveParams(payload.ServerID, spec))
	case "mcp.set_enabled":
		var payload struct {
			ServerID string          `json:"serverId"`
			ID       string          `json:"id"`
			Source   string          `json:"source"`
			Enabled  bool            `json:"enabled"`
			Expected string          `json:"expectedVersion"`
			Raw      json.RawMessage `json:"-"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		source := payload.Source
		if source == "" {
			source = "user"
		}
		return callAndReply(ctx, client, methodMCPSetEnabled, protocol.MCPSetEnabledParams{
			ID:              firstNonEmpty(payload.ServerID, payload.ID),
			Source:          source,
			ExpectedVersion: payload.Expected,
			Enabled:         payload.Enabled,
		})
	case "mcp.remove", "tool_server.remove":
		var payload struct {
			ServerID string `json:"serverId"`
			ID       string `json:"id"`
			Source   string `json:"source"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		source := payload.Source
		if source == "" {
			source = "user"
		}
		return callAndReply(ctx, client, methodMCPRemove, protocol.MCPRemoveParams{
			ID:     firstNonEmpty(payload.ServerID, payload.ID),
			Source: source,
		})
	case "workflow.start", "workflow.stop":
		// workflow.* 在 alpha.21 不注册：按能力缺失处理，不发请求。
		return write.Reply{}, workflowCapabilityMissing()
	case "session.wakeup.cancel", "wakeup.cancel":
		var payload struct {
			SessionID string `json:"sessionId"`
			WakeupID  string `json:"wakeupId"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, invalidPayload(err)
		}
		sessionID := firstNonEmpty(payload.SessionID, req.SessionID)
		if sessionID == "" {
			return write.Reply{}, failure.New(failure.KindNonRetryable, methodSessionWakeupCancel,
				fmt.Errorf("%w: 取消唤醒需要会话标识", write.ErrInvalidIntent))
		}
		runtime, runtimeErr := s.sessionRuntime(sessionID, methodSessionWakeupCancel)
		if runtimeErr != nil {
			return write.Reply{}, runtimeErr
		}
		if !s.app.sessionSupportsMethod(ctx, runtime, methodSessionWakeupCancel) {
			return write.Reply{}, capabilityMissing(methodSessionWakeupCancel)
		}
		raw, callErr := runtime.lease.Subscription().Call(ctx, methodSessionWakeupCancel, protocol.SessionWakeupCancelParams{
			SessionID: sessionID,
			WakeupID:  payload.WakeupID,
		})
		if callErr != nil {
			return write.Reply{}, callErr
		}
		return write.Reply{Result: raw}, nil
	default:
		return write.Reply{}, fmt.Errorf("%w: 未支持的管理写入 %q", write.ErrInvalidIntent, managed.Op)
	}
}

// hooksRevision 取 hooks.save 的版本前置条件：调用方给了就原样使用，
// 没给时读一次当前目录的 userRevision（D11b 的约定是「由集成层补齐」）。
func (s *appSender) hooksRevision(
	ctx context.Context,
	client *protocol.Client,
	expected string,
) (string, error) {
	if expected != "" {
		return expected, nil
	}
	raw, err := client.Call(ctx, methodHooksCatalog, nil)
	if err != nil {
		return "", err
	}
	catalog, err := protocol.ParseHooksCatalogResult(raw)
	if err != nil {
		return "", fmt.Errorf("钩子目录无法解析: %w", err)
	}
	return catalog.UserRevision, nil
}

// pluginSourceWire 把安装来源映射成真实 wire 形状：本地路径或 Git 地址。
func pluginSourceWire(source extensions.InstallSource) protocol.PluginSource {
	switch source.Kind {
	case extensions.SourceGit:
		return protocol.PluginSource{Git: source.Ref}
	default:
		return protocol.PluginSource{Path: source.Ref}
	}
}

// mcpSaveParams 把工具服务器配置映射成 `mcp.save` 参数。
//
// 载荷里带了原始配置文档就原样使用；否则按 transport/command/args/url/env 写一份
// 最小文档。source 与版本前置条件优先取载荷，没给时按用户作用域与空版本。
func mcpSaveParams(id string, spec extensions.ToolServerSpec) protocol.MCPSaveParams {
	configuration := spec.Configuration
	if len(configuration) == 0 {
		document := map[string]any{
			"transport": string(spec.Transport),
			"command":   spec.Command,
			"url":       spec.URL,
		}
		if len(spec.Args) > 0 {
			document["args"] = append([]string(nil), spec.Args...)
		}
		if len(spec.Env) > 0 {
			env := make(map[string]string, len(spec.Env))
			for key, value := range spec.Env {
				env[key] = value
			}
			document["env"] = env
		}
		configuration, _ = json.Marshal(document)
	}
	source := string(spec.Source)
	if source == "" {
		source = "user"
	}
	return protocol.MCPSaveParams{
		ID:              firstNonEmpty(id, spec.Name),
		Source:          source,
		ExpectedVersion: spec.ExpectedVersion,
		Configuration:   configuration,
	}
}

// sendAccountAction 把账号动作映射成 account.<op>。
func (s *appSender) sendAccountAction(
	ctx context.Context,
	client *protocol.Client,
	req write.Request,
) (write.Reply, error) {
	params, err := objectParams(req)
	if err != nil {
		return write.Reply{}, err
	}
	op, _ := params["op"].(string)
	if op == "" {
		return write.Reply{}, fmt.Errorf("%w: 账号动作缺少 op", write.ErrInvalidIntent)
	}
	delete(params, "op")
	var payload any
	if len(params) > 0 {
		payload = params
	}
	raw, err := client.Call(ctx, "account."+op, payload)
	if err != nil {
		return write.Reply{}, err
	}
	return write.Reply{Result: raw}, nil
}

// sendAttachment 用 attachment.put 把一段文本存成会话文件。
//
// 这是会话贡献方法：只在本会话的流上发，发送前先按会话方法清单判定能力，
// 缺方法即报能力缺失，不按服务端版本猜。暂存引用的生命周期仍归调用方。
func (s *appSender) sendAttachment(
	ctx context.Context,
	client *protocol.Client,
	req write.Request,
) (write.Reply, error) {
	if req.SessionID == "" {
		return write.Reply{}, fmt.Errorf("%w: 附件写入缺少会话标识", write.ErrInvalidIntent)
	}
	runtime, ok := s.app.session(req.SessionID)
	if !ok {
		return write.Reply{}, failure.New(failure.KindNonRetryable, methodAttachmentWrite,
			fmt.Errorf("%w: 附件写入需要先打开会话", ErrSessionNotOpen))
	}
	params, err := objectParams(req)
	if err != nil {
		return write.Reply{}, err
	}
	name, _ := params["name"].(string)
	text, _ := params["text"].(string)
	if text == "" {
		return write.Reply{}, fmt.Errorf("%w: 附件写入缺少 text", write.ErrInvalidIntent)
	}
	if !s.app.sessionSupportsMethod(ctx, runtime, methodAttachmentWrite) {
		return write.Reply{}, capabilityMissing(methodAttachmentWrite)
	}
	result, err := runtime.lease.Subscription().PutAttachment(ctx, protocol.AttachmentPutParams{
		SessionID: req.SessionID,
		Name:      name,
		Text:      text,
	})
	if err != nil {
		return write.Reply{}, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return write.Reply{}, fmt.Errorf("编码附件回执: %w", err)
	}
	return write.Reply{Result: raw}, nil
}

// sendSessionLifecycle 按参数里的 op 发一次会话生命周期写入。
func (s *appSender) sendSessionLifecycle(
	ctx context.Context,
	client *protocol.Client,
	req write.Request,
) (write.Reply, error) {
	var payload lifecyclePayload
	if err := json.Unmarshal(req.Params, &payload); err != nil {
		return write.Reply{}, invalidPayload(err)
	}
	switch payload.Op {
	case "branch.create", "catalog.branch_create", "branch_create":
		return s.sendBranchCreate(ctx, req, payload)
	case "branch.end", "catalog.branch_end", "branch_end":
		return s.sendBranchEnd(ctx, req, payload)
	case "create", "catalog.create":
		raw, err := client.Call(ctx, methodSessionCreate, map[string]any{"cwd": payload.Cwd})
		if err != nil {
			return write.Reply{}, err
		}
		return write.Reply{Result: raw}, nil
	case "rename", "catalog.rename":
		// 真实方法 `session.rename` 是会话方法：走本会话的流，回执带 revision。
		runtime, err := s.sessionRuntime(req.SessionID, methodSessionRename)
		if err != nil {
			return write.Reply{}, err
		}
		raw, err := runtime.lease.Subscription().Call(ctx, methodSessionRename, protocol.SessionRenameParams{
			SessionID: req.SessionID,
			Title:     payload.Title,
		})
		if err != nil {
			return write.Reply{}, err
		}
		return write.Reply{Result: raw}, nil
	case "delete", "catalog.delete":
		raw, err := client.Call(ctx, methodSessionDelete, map[string]any{"sessionId": req.SessionID})
		if err != nil {
			return write.Reply{}, err
		}
		return write.Reply{Result: raw}, nil
	default:
		return write.Reply{}, fmt.Errorf("%w: 未知会话生命周期操作 %q", write.ErrInvalidIntent, payload.Op)
	}
}

// branchWireValue 把意图的分支映射成 wire 取值；空表示主线。
func branchWireValue(branchID string) string {
	if branchID == "" {
		return mainBranchID
	}
	return branchID
}

// lifecyclePayload 是会话生命周期意图的透明参数。
type lifecyclePayload struct {
	Op             string  `json:"op"`
	Cwd            string  `json:"cwd"`
	Title          *string `json:"title"`
	ParentBranchID string  `json:"parentBranchId"`
	At             int64   `json:"at"`
	Prompt         string  `json:"prompt"`
	BranchID       string  `json:"branchId"`
}

// sendBranchCreate 从指定轮次的切点开分支：源会话 + 切点序号 + 首条输入。
func (s *appSender) sendBranchCreate(
	ctx context.Context,
	req write.Request,
	payload lifecyclePayload,
) (write.Reply, error) {
	runtime, err := s.sessionRuntime(req.SessionID, methodBranchCreate)
	if err != nil {
		return write.Reply{}, err
	}
	if payload.Prompt == "" {
		return write.Reply{}, fmt.Errorf("%w: 开分支缺少首条输入", write.ErrInvalidIntent)
	}
	if payload.At <= 0 {
		return write.Reply{}, fmt.Errorf("%w: 开分支缺少切点序号", write.ErrInvalidIntent)
	}
	params := map[string]any{
		"sessionId": req.SessionID,
		"at":        payload.At,
		"prompt":    payload.Prompt,
	}
	if payload.ParentBranchID != "" {
		params["parentBranchId"] = payload.ParentBranchID
	} else {
		params["parentBranchId"] = mainBranchID
	}
	if !s.app.sessionSupportsMethod(ctx, runtime, methodBranchCreate) {
		return write.Reply{}, capabilityMissing(methodBranchCreate)
	}
	raw, err := runtime.lease.Subscription().Call(ctx, methodBranchCreate, params)
	if err != nil {
		return write.Reply{}, err
	}
	return write.Reply{Result: raw}, nil
}

// sendBranchEnd 结束一个分支：源会话 + 分支标识。
func (s *appSender) sendBranchEnd(
	ctx context.Context,
	req write.Request,
	payload lifecyclePayload,
) (write.Reply, error) {
	branchID := firstNonEmpty(req.BranchID, payload.BranchID)
	if branchID == "" {
		return write.Reply{}, fmt.Errorf("%w: 结束分支缺少分支标识", write.ErrInvalidIntent)
	}
	runtime, err := s.sessionRuntime(req.SessionID, methodBranchEnd)
	if err != nil {
		return write.Reply{}, err
	}
	if !s.app.sessionSupportsMethod(ctx, runtime, methodBranchEnd) {
		return write.Reply{}, capabilityMissing(methodBranchEnd)
	}
	raw, err := runtime.lease.Subscription().Call(ctx, methodBranchEnd, map[string]any{
		"sessionId": req.SessionID,
		"branchId":  branchID,
	})
	if err != nil {
		return write.Reply{}, err
	}
	return write.Reply{Result: raw}, nil
}

// sendShellRun 发一次会话内命令的执行或停止。命令由服务端执行，本机终端另走 D9。
func (s *appSender) sendShellRun(ctx context.Context, req write.Request) (write.Reply, error) {
	runtime, err := s.sessionRuntime(req.SessionID, "shell")
	if err != nil {
		return write.Reply{}, err
	}
	params, err := objectParams(req)
	if err != nil {
		return write.Reply{}, err
	}
	op, _ := params["op"].(string)
	delete(params, "op")
	method := methodShellRun
	switch op {
	case "run":
		command, _ := params["command"].(string)
		if strings.TrimSpace(command) == "" {
			return write.Reply{}, fmt.Errorf("%w: 命令执行缺少 command", write.ErrInvalidIntent)
		}
	case "stop":
		method = methodShellStop
		if _, ok := params["runId"].(string); !ok || params["runId"] == "" {
			if req.InputID == "" {
				return write.Reply{}, fmt.Errorf("%w: 命令停止缺少运行标识", write.ErrInvalidIntent)
			}
			params["runId"] = req.InputID
		}
	default:
		return write.Reply{}, fmt.Errorf("%w: 未知命令执行操作 %q", write.ErrInvalidIntent, op)
	}
	params["sessionId"] = req.SessionID
	params["branchId"] = branchWireValue(req.BranchID)
	if !s.app.sessionSupportsMethod(ctx, runtime, method) {
		return write.Reply{}, capabilityMissing(method)
	}
	raw, err := runtime.lease.Subscription().Call(ctx, method, params)
	if err != nil {
		return write.Reply{}, err
	}
	reply := write.Reply{InputID: req.InputID, Result: raw}
	// shell.run 由服务端给出运行标识时以服务端为准，供结果未知时的核对使用。
	var result struct {
		RunID string `json:"runId"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &result); err == nil && result.RunID != "" {
			reply.InputID = result.RunID
		}
	}
	return reply, nil
}

// sendModality 发一次模态写入：任务提交/取消与实时传输打开/关闭。
func (s *appSender) sendModality(ctx context.Context, req write.Request) (write.Reply, error) {
	runtime, err := s.sessionRuntime(req.SessionID, "modality")
	if err != nil {
		return write.Reply{}, err
	}
	params, err := objectParams(req)
	if err != nil {
		return write.Reply{}, err
	}
	op, _ := params["op"].(string)
	delete(params, "op")
	var method string
	switch op {
	case "task.submit":
		method = methodModalityTaskSubmit
	case "task.cancel":
		method = methodModalityTaskCancel
	case "realtime.open":
		method = methodModalityOpen
	case "realtime.close":
		method = methodModalityClose
	default:
		return write.Reply{}, fmt.Errorf("%w: 未知模态写入操作 %q", write.ErrInvalidIntent, op)
	}
	params["sessionId"] = req.SessionID
	params["branchId"] = branchWireValue(req.BranchID)
	if !s.app.sessionSupportsMethod(ctx, runtime, method) {
		return write.Reply{}, capabilityMissing(method)
	}
	raw, err := runtime.lease.Subscription().Call(ctx, method, params)
	if err != nil {
		return write.Reply{}, err
	}
	reply := write.Reply{InputID: req.InputID, Result: raw}
	// 服务端给出运行/任务/连接标识时以服务端为准，供结果未知时的核对使用。
	var result struct {
		RunID        string `json:"runId"`
		TaskID       string `json:"taskId"`
		ConnectionID string `json:"connectionId"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &result); err == nil {
			switch {
			case result.RunID != "":
				reply.InputID = result.RunID
			case result.TaskID != "":
				reply.InputID = result.TaskID
			case result.ConnectionID != "":
				reply.InputID = result.ConnectionID
			}
		}
	}
	return reply, nil
}

// sessionRuntime 取已打开会话；没有打开时按不可重试失败处理，不发请求。
func (s *appSender) sessionRuntime(sessionID, method string) (*sessionRuntime, error) {
	return s.app.openRuntime(sessionID, method)
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// objectParams 把意图的透明参数解析为 JSON 对象；非对象按不合法意图拒绝。
func objectParams(req write.Request) (map[string]any, error) {
	params := make(map[string]any)
	if len(req.Params) == 0 {
		return params, nil
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("%w: 意图参数不是 JSON 对象: %v", write.ErrInvalidIntent, err)
	}
	return params, nil
}

// deliveryWireValue 把投递方式映射成 wire 取值。
//
// 排队取 queue；插话取 steer（投影按同一个取值识别插话进入当前运行）。
// 插话的 wire 取值只在投影的词表里有证据，真实服务端上未单独验证。
func deliveryWireValue(delivery write.Delivery) string {
	if delivery == write.DeliveryInterject {
		return "steer"
	}
	return "queue"
}

// callAndReply 发一次请求并把回执原文交回写登记。
func callAndReply(
	ctx context.Context,
	client *protocol.Client,
	method string,
	params any,
) (write.Reply, error) {
	raw, err := client.Call(ctx, method, params)
	if err != nil {
		return write.Reply{}, err
	}
	return write.Reply{Result: raw}, nil
}

// toolServerParams 把工具服务器配置映射成服务端参数。
func toolServerParams(spec extensions.ToolServerSpec) map[string]any {
	params := map[string]any{
		"name":      spec.Name,
		"transport": string(spec.Transport),
		"command":   spec.Command,
		"url":       spec.URL,
	}
	if len(spec.Args) > 0 {
		params["args"] = append([]string(nil), spec.Args...)
	}
	if len(spec.Env) > 0 {
		env := make(map[string]string, len(spec.Env))
		for key, value := range spec.Env {
			env[key] = value
		}
		params["env"] = env
	}
	return params
}

// ErrCapabilityMissing 表示该会话没有提供这个贡献方法；发送前就已判定，请求没有发出。
var ErrCapabilityMissing = errors.New("服务端缺少该能力")

// capabilityMissing 把缺方法包成不可重试失败，写登记据此按明确失败处理。
func capabilityMissing(method string) error {
	return failure.New(failure.KindNonRetryable, method,
		fmt.Errorf("%w: %s", ErrCapabilityMissing, method))
}

func invalidPayload(err error) error {
	return fmt.Errorf("%w: 写意图载荷无法解析: %v", write.ErrInvalidIntent, err)
}
