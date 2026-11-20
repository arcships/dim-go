package ui

import (
	"context"
	"encoding/json"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 审批卡：贴在输入区上沿，整体替换输入区（参考产品 PermissionPrompt）。
//
// 三种卡共用一个外层骨架（描边 + 内边距 16 + 与页面同底色、无阴影）：
//   - 工具权限卡：标题「DimAgent 想要：{动作}」+ 成员区块（若有）+ 输入区 +
//     自动审批交回的提示块 + 错误行 + 三个决定；
//   - 沙箱越权卡：拦截标题 + 被拦域名/路径 + 说明 + 错误行 + 决定；
//   - 子代理模型卡：换模型标题 + 任务 + 模型块 + 价格标记 + 回退说明 + 错误行 +
//     决定。
//
// 事实从哪来：工具名、入参原文、长期允许标记、来源类别都来自待决审批快照。沙箱
// 与子代理模型两种形状 protocol v3 的 `approval.request` 没有字段（只有 tool /
// host 两种来源），连接层不会填，所以生产环境只会出现工具权限卡；成员审批用请求
// 的工具名 `member_approval_allow` 认出来，明细通道本版没有，按参考产品的
// 「当前客户端无法审查成员操作」画，只留「拒绝」。
//
// 文案逐字取参考产品 zh-CN（`permissionPrompt.*`）。

// 卡片与决定的文案。
const (
	// approvalWantPrefix 是工具权限卡标题的前半段（`permissionPrompt.actionRequest`）。
	approvalWantPrefix = "DimAgent 想要："
	// approvalUseToolAction 是没有专用动作标签的工具的标题词（`permissionPrompt.useTool`）。
	approvalUseToolAction = "使用工具"
	// 三个决定的文案（`permissionPrompt.reject` / `allowOnce` / `alwaysAllow*`）。
	approvalCopyReject          = "拒绝"
	approvalCopyDontAllow       = "不允许"
	approvalCopyAllowOnce       = "仅这次"
	approvalCopyAlwaysCommand   = "始终允许此命令"
	approvalCopyAlwaysNetwork   = "始终允许网络访问"
	approvalCopyAlwaysFile      = "始终允许文件访问"
	approvalCopyAllowForSession = "允许此会话期间访问"
	// 自动审批交回人工的两种提示块（`needsYourDecision` / `autoReviewFailed`）。
	approvalCopyNeedsDecision    = "需要你判断"
	approvalCopyAutoReviewFailed = "自动审批没有完成，请由你决定。"
	// 子代理换模型卡（`subagentModel.*`）。
	approvalCopySubagentTitle   = "DimAgent 想要为子任务使用更贵的模型"
	approvalCopySubagentTask    = "子任务"
	approvalCopySubagentPricier = "价格高于当前模型"
	approvalCopySubagentReject  = "使用当前模型"
	approvalCopySubagentAllow   = "本会话允许"
	// 成员审批区块（`memberApproval.*`）。
	approvalCopyMemberLoading          = "正在加载成员操作..."
	approvalCopyMemberErrorTitle       = "无法加载成员操作"
	approvalCopyMemberError            = "App Server 未能返回原始操作。请重试或拒绝。"
	approvalCopyMemberRetry            = "重试"
	approvalCopyMemberUnavailableTitle = "当前客户端无法审查成员操作"
	approvalCopyMemberUnavailable      = "当前客户端与 App Server 无法显示原始操作，只能拒绝。"
	approvalCopyMemberIDPrefix         = "审批 "
	// 沙箱越权卡的四个标题与四句说明（`sandbox*`）。
	approvalCopySandboxNetwork         = "网络访问已拦截"
	approvalCopySandboxRead            = "文件读取已拦截"
	approvalCopySandboxWrite           = "文件写入已拦截"
	approvalCopySandboxExecute         = "文件执行已拦截"
	approvalCopySandboxNetworkMessage  = "需要访问该域名。"
	approvalCopySandboxReadMessage     = "需要读取该路径。"
	approvalCopySandboxWriteMessage    = "需要写入该路径。"
	approvalCopySandboxExecuteMessage  = "需要执行该路径。"
	approvalCopySandboxFallbackMessage = "需要访问该路径。"
	// 自动审批返回码（`handoff.kind`）。
	approvalHandoffUncertain = "reviewer_uncertain"
	approvalHandoffFailed    = "reviewer_failed"
)

// 审批卡与决定按钮的键位提示（参考产品 PermissionPrompt 的窗口级键位）。
const (
	approvalTipAllowOnce = approvalCopyAllowOnce + " · Enter"
	approvalTipReject    = approvalCopyReject + " · N"
)

// approvalKeysReady 判断审批卡此刻能否用键位答复：主界面、没有叠着命令面板、
// 焦点不在输入框里（参考产品同样跳过输入焦点与弹层）。
func (a *App) approvalKeysReady(c *gui.Context) bool {
	if a.palette.open || a.screen != ScreenMain || c.FocusTakesText() {
		return false
	}
	_, ok := a.currentApproval()
	return ok
}

// approvalKeys 是审批卡的键位：Enter / y 以「仅这次」答复，n 拒绝。
// 不接 Esc：需求 R-AP-017 明确要求「Esc 不答复」，本端按需求留白（参考产品把
// Esc 绑成拒绝）。答复进行中或请求已不可答时一律不响应。
func (a *App) approvalKeys(c *gui.Context) {
	if !a.approvalKeysReady(c) {
		return
	}
	snap, _ := a.currentApproval()
	if a.approvalBusy(snap) {
		return
	}
	switch {
	case c.Shortcut(0, gui.KeyEnter), c.Shortcut(0, gui.KeyY):
		a.respondApproval("allow")
	case c.Shortcut(0, gui.KeyN):
		a.respondApproval("deny")
	}
}

// currentApproval 取当前要画的审批快照：连着 Source 用事实，否则用样例。
func (a *App) currentApproval() (ApprovalSnapshot, bool) {
	if snap, ok := a.liveApproval(); ok {
		return snap, true
	}
	if a.wired() {
		return ApprovalSnapshot{}, false
	}
	if a.state != StateWaiting {
		return ApprovalSnapshot{}, false
	}
	// 样例态：命令渲染器 + 可长期允许，与参考产品 A 态同一张卡。
	return ApprovalSnapshot{
		ID:       "sample-approval",
		Tool:     "exec",
		Command:  a.approvalCommand,
		LongTerm: true,
	}, true
}

// approvalBusy 判断三个决定此刻是否不可点：请求已答等结算、这次答复正在写、
// 或根本没有可审查的内容——工具名与入参都没有，或有工具名却读不到入参
// （看不到要执行什么就不放行）。
func (a *App) approvalBusy(snap ApprovalSnapshot) bool {
	if snap.Pending || a.approvalSubmitting {
		return true
	}
	_, _, hasInput := approvalFacts(snap)
	return !hasInput
}

// approvalFacts 把快照解成渲染器要的两样：工具名与入参对象。
//
// 入参不是对象时按参考产品的旧版纯文本入参处理：包成 `{input: 文本}` 交给通用
// 渲染器原样显示，不猜字段、也不折成"无效入参"；没有入参才是真的没有。
func approvalFacts(snap ApprovalSnapshot) (string, map[string]any, bool) {
	tool := strings.TrimSpace(snap.Tool)
	input := map[string]any{}
	if raw := strings.TrimSpace(snap.Input); raw != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil && parsed != nil {
			input = parsed
		} else {
			var text string
			if err := json.Unmarshal([]byte(raw), &text); err != nil {
				text = raw
			}
			input["input"] = text
		}
	}
	if _, ok := input["command"]; !ok && strings.TrimSpace(snap.Command) != "" {
		input["command"] = snap.Command
	}
	return tool, input, len(input) > 0
}

// approvalCard 画审批卡：外层骨架 + 三选一的主体。
func (a *App) approvalCard(c *gui.Context) {
	t := c.Theme()
	snap, ok := a.currentApproval()
	if !ok {
		return
	}
	// IN-43 / IN-66 / G-C1-01 审批卡容器
	gui.Column(c).WidthPercent(100).MaxWidth(composerMaxWidth).Padding(space4).
		Gap(space3).Background(t.Surface).Border(strokeHairline, t.Border).
		// G-C2-01 沙箱卡容器（与审批卡同一骨架）
		Radius(radiusWindowCard).Role(gui.RoleRegion).Label("审批卡").Children(func() {
		switch {
		case snap.Sandbox != nil:
			a.sandboxApprovalBody(c, snap)
		case snap.SubagentModel != nil:
			a.subagentModelBody(c, snap)
		default:
			a.toolApprovalBody(c, snap)
		}
	})
}

// toolApprovalBody 是工具权限卡的主体：标题、成员区块、输入区、提示块、错误行、
// 决定按钮。
func (a *App) toolApprovalBody(c *gui.Context, snap ApprovalSnapshot) {
	t := c.Theme()
	// IN-67 / G-C1-02 标题「DimAgent 想要：{action}」
	typeBodyStrong.style(t, gui.Text(c, approvalTitle(snap))).
		FontWeight(600).SingleLine()
	if snap.Member != nil {
		a.memberApprovalBlock(c, *snap.Member)
	}
	// 成员动作还没读到时不摆输入区：宁可只写"读不到"，也不画半张别人的卡。
	if snap.Member == nil || snap.Member.State == ApprovalMemberReady {
		tool, input, hasInput := approvalFacts(snap)
		switch {
		case hasInput:
			// IN-69 / G-C1-13 审批输入区（按工具类型挑面板）
			approvalInputPanel(c, snap.ID, approvalPanelFor(tool, input))
		case tool == "":
			// G-C1-08 / G-C1-09 读不到原始操作时的原因句
			typeMicro.style(t, gui.Text(c, noteApprovalCommand)).TextColor(t.TextMuted)
			typeMicro.style(t, gui.Text(c, noteApprovalBlocked)).TextColor(t.TextMuted)
		default:
			// 有工具名、没有入参：不画内容区，但要写清读不到什么，且三个决定
			// 一起置灰（approvalBusy 同因返回真）：看不到要执行什么就不放行。
			typeMicro.style(t, gui.Text(c, noteApprovalInput)).TextColor(t.TextMuted)
			typeMicro.style(t, gui.Text(c, noteApprovalBlocked)).TextColor(t.TextMuted)
		}
	}
	a.handoffBlock(c, snap.Handoff)
	a.approvalErrorLine(c, snap)
	a.approvalActions(c, snap)
}

// approvalTitle 是工具权限卡的标题：成员请求读成员标题，其余读「{动作}」。
func approvalTitle(snap ApprovalSnapshot) string {
	if snap.Member != nil {
		// G-C1-05 成员审批的标题几种状态
		switch snap.Member.State {
		case ApprovalMemberLoading:
			return approvalCopyMemberLoading
		case ApprovalMemberError:
			return approvalCopyMemberErrorTitle
		case ApprovalMemberUnavailable:
			return approvalCopyMemberUnavailableTitle
		}
	}
	_, input, _ := approvalFacts(snap)
	// G-C1-03 认不出工具时的动作词「使用工具」
	return approvalWantPrefix + approvalActionWord(snap.Tool, input)
}

// approvalActionWord 是标题里的动作词。工具名认不出来时按参考产品的回退写
// 「使用工具」；只有命令原文（没有工具名）时按命令工具写「执行」。
func approvalActionWord(tool string, input map[string]any) string {
	if strings.TrimSpace(tool) == "" {
		return "执行"
	}
	raw := strings.ToLower(strings.TrimSpace(tool))
	name := normalizeToolName(tool)
	switch name {
	case "read", "read_file", "cat":
		return "读取"
	case "write", "write_file":
		return "写入"
	case "edit", "edit_file":
		return "编辑"
	case "exec", "execute", "exec_command", "run_command", "bash", "shell", "terminal":
		if background, ok := input["runInBackground"].(bool); ok && background {
			return "后台执行"
		}
		return "执行"
	case "list_files", "ls":
		return "列出文件"
	case "grep", "rg":
		return "搜索内容"
	case "glob":
		return "查找"
	case "web_search", "WebSearch":
		return "搜索网络"
	case "web_fetch", "WebFetch":
		return "获取网页"
	case "todo_write", "todowrite":
		return "更新任务"
	case "generate_image":
		return "生成图片"
	case "skill":
		return "使用技能"
	case "browser":
		return "浏览器"
	case "wait":
		return "等待"
	case "agent", "subagent":
		switch approvalText(input, []string{"action"}) {
		case "create", "create_external":
			return "创建智能体"
		case "list":
			return "查询智能体状态"
		case "cancel":
			return "取消智能体任务"
		case "query_provider_model":
			return "查询可用模型"
		case "query_external_model":
			return "查询外部智能体模型"
		}
		return "创建智能体"
	}
	switch approvalRendererFor(raw, input) {
	case approvalRendererTodos:
		return "更新任务"
	case approvalRendererMCP:
		return approvalUseToolAction
	case approvalRendererCua:
		return "操作电脑"
	}
	switch approvalCategory(raw) {
	case approvalCategoryRead:
		return "读取"
	case approvalCategoryEdit:
		return "编辑"
	case approvalCategoryCommand:
		return "执行"
	case approvalCategorySearch:
		return "搜索内容"
	case approvalCategoryWebSearch:
		return "搜索网络"
	case approvalCategoryWebFetch:
		return "获取网页"
	case approvalCategorySkill:
		return "使用技能"
	case approvalCategoryBrowser:
		return "浏览器"
	case approvalCategoryWait:
		return "等待"
	case approvalCategorySubagent:
		return "创建智能体"
	case approvalCategoryComputerUse:
		return "操作电脑"
	}
	return approvalUseToolAction
}

// memberApprovalBlock 是成员审批区块：浅底、60% 描边、内边距 12；自上而下是
// 成员标识、主分支、错误/不可用说明、审批 ID、原因，失败时还有「重试」。
func (a *App) memberApprovalBlock(c *gui.Context, member ApprovalMemberSnapshot) {
	t := c.Theme()
	// IN-68 / G-C1-04 成员审批上下文块
	gui.Column(c).WidthPercent(100).Padding(space3).Gap(space1).
		Background(colorSurfaceHover.in(t)).Radius(radiusCard).
		Border(strokeHairline, t.Border).Label("成员审批").Children(func() {
		gui.Column(c).Gap(0).Children(func() {
			if member.AgentID != "" || member.BranchID != "" {
				// G-C1-06 成员标识行
				typeSecondary.style(t, gui.Text(c, memberLabel(member))).
					TextColor(t.TextMuted).SingleLine()
			}
			if member.LeaderBranchID != "" {
				// G-C1-07 主分支行
				typeSecondary.style(t, gui.Text(c, "主分支 "+member.LeaderBranchID)).
					TextColor(t.TextMuted).SingleLine()
			}
			switch member.State {
			case ApprovalMemberError:
				typeSecondary.style(t, gui.Text(c, approvalCopyMemberError)).TextColor(t.Danger)
			case ApprovalMemberUnavailable:
				typeSecondary.style(t, gui.Text(c, approvalCopyMemberUnavailable)).
					TextColor(t.TextMuted)
			case ApprovalMemberReady:
				if member.Reason != "" {
					// G-C1-11 审批原因
					typeSecondary.style(t, gui.Text(c, member.Reason)).TextColor(t.TextMuted)
				}
			}
			if member.State == ApprovalMemberUnavailable && member.ApprovalID != "" {
				// 审批 ID 是等宽、UI 正文一档的次要色（参考产品 font-mono text-xs）。
				// G-C1-10 审批 ID（等宽）
				typeSecondary.style(t, gui.Text(c, approvalCopyMemberIDPrefix+member.ApprovalID)).
					Font(codeFamily()).TextColor(t.TextMuted)
			}
			if member.State == ApprovalMemberError && member.CanRetry {
				label := approvalCopyMemberRetry + "（重新读取成员操作）"
				// G-C1-12 重新读取成员操作（重试）
				if actionButton(c, approvalCopyMemberRetry, label, false, false).Clicked() {
					a.retryMemberApproval()
				}
			}
		})
	})
}

// memberLabel 是成员标识行：两段都用中点分隔，缺哪段就不写哪段。
func memberLabel(member ApprovalMemberSnapshot) string {
	switch {
	case member.AgentID != "" && member.BranchID != "":
		return "成员 " + member.AgentID + " · 分支 " + member.BranchID
	case member.AgentID != "":
		return "成员 " + member.AgentID
	default:
		return "分支 " + member.BranchID
	}
}

// retryMemberApproval 重新拉取成员动作。本版没有成员审批明细通道，重试只会把
// 原因写清，不让按钮看起来像做了事。
func (a *App) retryMemberApproval() {
	a.approvalError = noteMemberApprovalDetail
}

// handoffBlock 是自动审批交回人工的提示块：不确定时写标题 + 原因，审核失败时
// 只写整句。
func (a *App) handoffBlock(c *gui.Context, handoff *ApprovalHandoffSnapshot) {
	if handoff == nil || (handoff.Kind != approvalHandoffUncertain && handoff.Kind != approvalHandoffFailed) {
		return
	}
	t := c.Theme()
	// IN-77 / G-C1-14 / G-C1-15 自动审批交回人工的提示块
	gui.Column(c).WidthPercent(100).Padding(space3).Gap(space1).
		Background(colorSurfaceHover.in(t)).Radius(radiusCard).Label("自动审批提示").
		Children(func() {
			if handoff.Kind == approvalHandoffFailed {
				typeSecondary.style(t, gui.Text(c, approvalCopyAutoReviewFailed)).
					TextColor(t.TextMuted)
				return
			}
			typeBodyStrong.style(t, gui.Text(c, approvalCopyNeedsDecision)).FontWeight(600)
			if handoff.Reason != "" {
				typeSecondary.style(t, gui.Text(c, handoff.Reason)).TextColor(t.TextMuted)
			}
		})
}

// approvalErrorLine 是卡片自己的失败回执：一行危险色文字，位于按钮行上方；
// 同一位置只留最近一次原因。
// G-C2-05 / G-C3-07 提交失败的错误行
func (a *App) approvalErrorLine(c *gui.Context, snap ApprovalSnapshot) {
	if a.approvalError == "" || a.approvalErrorFor != snap.ID {
		return
	}
	t := c.Theme()
	// IN-78 / G-C1-16 卡片自己的失败回执
	typeSecondary.style(t, gui.Text(c, a.approvalError)).TextColor(t.Danger).
		Role(gui.RoleAlert).Label(a.approvalError)
}

// approvalActions 画按钮组：右对齐，顺序是「拒绝 → 仅这次 → 长期允许」，与读屏
// 顺序一致。答复进行中或请求不可答时整组置灰。
func (a *App) approvalActions(c *gui.Context, snap ApprovalSnapshot) {
	busy := a.approvalBusy(snap)
	// IN-74 / G-C1-17 决定按钮组
	gui.Row(c).WidthPercent(100).Gap(space2).AlignItems(gui.Center).Wrap().Children(func() {
		gui.Spacer(c)
		for _, action := range approvalActionsFor(snap) {
			var button gui.Element
			if action.primary {
				button = gui.PrimaryButton(c, action.copy)
			} else {
				button = gui.Button(c, action.copy)
			}
			button = button.Label(action.copy)
			if action.tip != "" {
				button = button.Tooltip(action.tip)
			}
			// G-C1-21 答复进行中整组置灰
			if busy || action.disabled {
				button = button.Disabled(true)
			}
			if button.Clicked() {
				a.respondApproval(action.decision)
			}
		}
	})
}

// approvalAction 是一枚决定：文案、答复值与是否主按钮。
type approvalAction struct {
	copy     string
	decision string
	primary  bool
	tip      string
	disabled bool
}

// approvalActionsFor 按快照给出按钮组。成员审批只留「拒绝」（明细读不到时允许
// 等于同意一个没看见的操作）；沙箱与子代理模型卡按参考产品换文案；服务端没有给出
// 长期规则时，工具卡不画长期允许按钮（R-AP-214：不出现，而不是禁用）。
func approvalActionsFor(snap ApprovalSnapshot) []approvalAction {
	rejectCopy := approvalCopyReject
	if snap.Sandbox != nil {
		// G-C2-06 「不允许」
		rejectCopy = approvalCopyDontAllow
	}
	// G-C1-19 「拒绝」/「不允许」
	actions := []approvalAction{{
		copy: rejectCopy, decision: "deny", tip: approvalTipReject,
	}}
	switch {
	case snap.Sandbox != nil:
		// G-C2-07 「仅这次」
		actions = append(actions, approvalAction{
			copy: approvalCopyAllowOnce, decision: "allow", primary: true,
			tip: approvalTipAllowOnce,
		})
		actions = append(actions, approvalAction{
			copy: approvalSandboxLongTermCopy(snap.Sandbox), decision: "allow-always",
			// G-C2-08 始终允许网络访问 / 始终允许文件访问
			disabled: !snap.LongTerm,
		})
		return actions
	case snap.SubagentModel != nil:
		return []approvalAction{
			// G-C3-08 使用当前模型
			{copy: approvalCopySubagentReject, decision: "deny"},
			// G-C3-09 本会话允许
			{copy: approvalCopySubagentAllow, decision: "allow", primary: true},
		}
	case snap.Member != nil:
		return actions
	}
	// G-C1-18 「仅这次」
	actions = append(actions, approvalAction{
		copy: approvalCopyAllowOnce, decision: "allow", primary: true, tip: approvalTipAllowOnce,
	})
	// R-AP-214：请求没有给出长期规则（`allow_all`）时，长期允许这一项整块不画——
	// 不渲染成禁用态，免得看起来「只差一步就能点」。
	if snap.LongTerm {
		actions = append(actions, approvalAction{
			copy: approvalLongTermCopy(snap), decision: "allow-always",
		})
	}
	return actions
}

// approvalLongTermCopy 是长期允许按钮的文案：命令工具是「始终允许此命令」，
// 读写文件是「允许此会话期间访问」，其余工具是「始终允许文件访问」。
// G-C1-20 长期允许按钮的三句文案
func approvalLongTermCopy(snap ApprovalSnapshot) string {
	if snap.Sandbox != nil {
		return approvalSandboxLongTermCopy(snap.Sandbox)
	}
	switch approvalCategory(snap.Tool) {
	case approvalCategoryCommand:
		return approvalCopyAlwaysCommand
	case approvalCategoryRead, approvalCategoryEdit:
		return approvalCopyAllowForSession
	}
	if strings.TrimSpace(snap.Tool) == "" {
		return approvalCopyAlwaysCommand
	}
	return approvalCopyAlwaysFile
}

// approvalSandboxLongTermCopy 是沙箱卡长期允许的文案：域名与文件各一句。
// G-C2-08 沙箱长期允许的两句文案
func approvalSandboxLongTermCopy(sandbox *ApprovalSandboxSnapshot) string {
	if sandbox.ResourceType == "path" {
		return approvalCopyAlwaysFile
	}
	return approvalCopyAlwaysNetwork
}

// respondApproval 把审批决定发出去：接上事实时走 Intents，否则只记样例反馈。
func (a *App) respondApproval(decision string) {
	snap, ok := a.currentApproval()
	if !ok {
		return
	}
	if a.approvalBusy(snap) {
		return
	}
	a.approvalDecision = decision
	if a.intents == nil {
		return
	}
	// 失败原因写进卡片自己的错误行，不借用标题下方的全局错误行。
	a.approvalSubmitting = true
	a.approvalError = ""
	a.approvalErrorFor = snap.ID
	a.dispatchQuiet("", func(ctx context.Context) error {
		return a.intents.RespondApproval(ctx, decision)
	}, func() {
		a.approvalSubmitting = false
	}, func(reason string) {
		a.approvalSubmitting = false
		a.approvalError = reason
		a.approvalErrorFor = snap.ID
	})
}

// sandboxApprovalBody 是沙箱越权卡的主体：拦截标题、被拦资源、说明、错误行、
// 决定按钮。协议 v3 没有这种请求，连接层不会填；画法按参考产品。
// IN-75 沙箱拦截卡
func (a *App) sandboxApprovalBody(c *gui.Context, snap ApprovalSnapshot) {
	t := c.Theme()
	sandbox := snap.Sandbox
	// G-C2-02 网络访问已拦截 / 文件读取已拦截 / 文件写入已拦截 / 文件执行已拦截
	typeBodyStrong.style(t, gui.Text(c, sandboxTitle(sandbox))).FontWeight(600).SingleLine()
	if strings.TrimSpace(sandbox.Resource) != "" {
		// G-C2-03 被拦的域名或路径
		typeCode.style(t, gui.Text(c, sandbox.Resource)).TextColor(t.Text)
	}
	// G-C2-04 {tool} 需要访问该域名 / 需要读取该路径
	typeSecondary.style(t, gui.Text(c, sandboxMessage(sandbox))).TextColor(t.TextMuted)
	a.approvalErrorLine(c, snap)
	a.approvalActions(c, snap)
}

// sandboxTitle 是沙箱卡的标题：按被拦的资源与访问类型四选一。
func sandboxTitle(sandbox *ApprovalSandboxSnapshot) string {
	if sandbox.ResourceType != "path" {
		return approvalCopySandboxNetwork
	}
	switch sandbox.Access {
	case "read":
		return approvalCopySandboxRead
	case "write":
		return approvalCopySandboxWrite
	case "execute":
		return approvalCopySandboxExecute
	}
	return approvalCopySandboxNetwork
}

// sandboxMessage 是沙箱卡的说明句：有工具名时带上工具名，没有就不留空主语。
// G-C2-04 沙箱说明句的四种写法
func sandboxMessage(sandbox *ApprovalSandboxSnapshot) string {
	message := approvalCopySandboxFallbackMessage
	if sandbox.ResourceType != "path" {
		message = approvalCopySandboxNetworkMessage
	} else {
		switch sandbox.Access {
		case "read":
			message = approvalCopySandboxReadMessage
		case "write":
			message = approvalCopySandboxWriteMessage
		case "execute":
			message = approvalCopySandboxExecuteMessage
		}
	}
	tool := strings.TrimSpace(sandbox.Tool)
	if tool == "" {
		return message
	}
	return tool + " " + message
}

// subagentModelBody 是子代理模型卡的主体：标题、任务行、模型块、价格标记、回退
// 说明、错误行、决定按钮。协议 v3 没有这种请求，连接层不会填；画法按参考产品。
// IN-76 子代理模型卡
// G-C3-01 子代理模型卡容器
func (a *App) subagentModelBody(c *gui.Context, snap ApprovalSnapshot) {
	t := c.Theme()
	model := snap.SubagentModel
	// G-C3-02 「DimAgent 想要为子任务使用更贵的模型」
	typeBodyStrong.style(t, gui.Text(c, approvalCopySubagentTitle)).
		FontWeight(600).SingleLine()
	if strings.TrimSpace(model.TaskTitle) != "" {
		// G-C3-03 「子任务 · {taskTitle}」
		typeSecondary.style(t, gui.Text(c, approvalCopySubagentTask+" · "+model.TaskTitle)).
			TextColor(t.TextMuted).SingleLine()
	}
	gui.Row(c).WidthPercent(100).Padding(space3).Gap(space2).AlignItems(gui.Center).
		Background(colorSurfaceHover.in(t)).Radius(radiusCard).
		// G-C3-04 模型块（图标 + 模型名 + 提供方）
		Border(strokeHairline, t.Border).Label("模型").Children(func() {
		iconGlyph(c, iconSparkles, colorAccentInk.in(t), "model")
		gui.Column(c).Grow(1).Shrink(1).Gap(0).Children(func() {
			if model.ModelName != "" {
				typeSecondary.style(t, gui.Text(c, model.ModelName)).TextColor(t.Text).SingleLine()
			}
			if model.ProviderName != "" {
				typeMicro.style(t, gui.Text(c, model.ProviderName)).TextColor(t.TextMuted).SingleLine()
			}
		})
		// G-C3-05 「价格高于当前模型」
		if model.Pricier {
			typeMicro.style(t, gui.Text(c, approvalCopySubagentPricier)).TextColor(t.Warning)
		}
	})
	typeMicro.style(t, gui.Text(c, subagentFallbackCopy(model))).TextColor(t.TextMuted)
	a.approvalErrorLine(c, snap)
	a.approvalActions(c, snap)
}

// subagentFallbackCopy 是回退说明：有当前模型名时带上名字，没有就退成整句，
// 不留空占位。
// G-C3-06 拒绝时子任务将使用当前模型 {model} 继续
func subagentFallbackCopy(model *ApprovalSubagentModelSnapshot) string {
	if strings.TrimSpace(model.FallbackModel) == "" {
		return "拒绝时子任务将继续使用当前模型"
	}
	return "拒绝时子任务将使用当前模型 " + model.FallbackModel + " 继续"
}
