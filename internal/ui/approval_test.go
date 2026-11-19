package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是审批卡的可执行验收：十类输入渲染器各一条、三个决定与长期允许的文案、
// 成员审批区块、沙箱越权卡、子代理模型卡、自动审批提示块、失败回执与提交中态、
// 键位。每条都构造对应的审批事实，再断言屏上真的出现期望元素。
//
// 审批事实走 Source（ApprovalSnapshot）：Tool / Input 是服务端原文，界面不补字段。

// approvalTester 连上 Source 造一张审批卡。
func approvalTester(t *testing.T, snapshot ApprovalSnapshot) (*App, *gui.Tester) {
	t.Helper()
	src := wiredSource()
	src.hasAppr = true
	src.approval = snapshot
	app := NewApp(Options{State: StateDefault, Source: src})
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// approvalWants 断言这些文案都在屏上，缺一条就报出全屏文字便于定位。
func approvalWants(t *testing.T, tt *gui.Tester, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !tt.HasText(want) {
			t.Fatalf("审批卡缺少 %q：\n%s", want, p0WindowText(tt))
		}
	}
}

// approvalDoesNotWant 断言这些文案都没上屏。
func approvalDoesNotWant(t *testing.T, tt *gui.Tester, unwanted ...string) {
	t.Helper()
	text := p0WindowText(tt)
	for _, item := range unwanted {
		if strings.Contains(text, item) {
			t.Fatalf("审批卡不该出现 %q：\n%s", item, text)
		}
	}
}

// TestApprovalPanelExec：命令渲染器画命令、cwd、超时与后台标记。
func TestApprovalPanelExec(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-exec", Tool: "exec", LongTerm: true,
		Input: `{"command":"ls -la","cwd":"/work/dim","timeoutMs":60000,"runInBackground":true}`,
	})
	approvalWants(t, tt,
		"DimAgent 想要：后台执行", approvalInputLabel, "ls -la",
		"cwd: /work/dim", "timeoutMs: 60 s", "runInBackground: true")
	approvalDoesNotWant(t, tt, "command:")
}

// TestApprovalPanelRead：文件渲染器用路径做标签，读出范围参数。
func TestApprovalPanelRead(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-read", Tool: "read",
		Input: `{"path":"/work/internal/app/app.go","offset":10,"limit":20}`,
	})
	approvalWants(t, tt, "DimAgent 想要：读取", "/work/internal/app/app.go", "offset: 10", "limit: 20")
}

// TestApprovalPanelWriteAndEdit：写入是带行号的代码块，编辑是双栏差异。
func TestApprovalPanelWriteAndEdit(t *testing.T) {
	t.Run("写入", func(t *testing.T) {
		_, tt := approvalTester(t, ApprovalSnapshot{
			ID: "a-write", Tool: "write",
			Input: `{"path":"/work/tmp/out.txt","content":"第一行\n第二行"}`,
		})
		approvalWants(t, tt, "DimAgent 想要：写入", "/work/tmp/out.txt", "第一行", "第二行", "1", "2")
	})
	t.Run("编辑", func(t *testing.T) {
		_, tt := approvalTester(t, ApprovalSnapshot{
			ID: "a-edit", Tool: "edit",
			Input: `{"path":"/work/tmp/out.txt","oldText":"旧文本","newText":"新文本","replaceAll":true}`,
		})
		approvalWants(t, tt, "DimAgent 想要：编辑", "/work/tmp/out.txt",
			"旧文本", "新文本", "replaceAll: true")
	})
}

// TestApprovalPanelSearch：检索渲染器画模式主值行、路径与其余参数。
func TestApprovalPanelSearch(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-grep", Tool: "grep",
		Input: `{"pattern":"func main","path":"/work","include":"*.go","lineNumbers":true}`,
	})
	approvalWants(t, tt, "DimAgent 想要：搜索内容", approvalInputLabel,
		"pattern: func main", "path: /work", "include: *.go", "lineNumbers: true")

	_, list := approvalTester(t, ApprovalSnapshot{
		ID: "a-glob", Tool: "glob",
		Input: `{"pattern":"*.md","path":"/work/docs"}`,
	})
	approvalWants(t, list, "DimAgent 想要：查找", "pattern: *.md", "path: /work/docs")
}

// TestApprovalPanelWeb：抓取用 URL 做标签并画站点徽记，搜索画查询与分页。
func TestApprovalPanelWeb(t *testing.T) {
	t.Run("抓取", func(t *testing.T) {
		_, tt := approvalTester(t, ApprovalSnapshot{
			ID: "a-fetch", Tool: "web_fetch",
			Input: `{"url":"https://example.com/doc?page=2","startIndex":0,"mode":"text"}`,
		})
		approvalWants(t, tt, "DimAgent 想要：获取网页",
			"https://example.com/doc?page=2", "startIndex: 0", "mode: text", "站点：example.com")
	})
	t.Run("搜索", func(t *testing.T) {
		_, tt := approvalTester(t, ApprovalSnapshot{
			ID: "a-search", Tool: "web_search",
			Input: `{"query":"mygo 文档","page":1,"pageSize":10}`,
		})
		approvalWants(t, tt, "DimAgent 想要：搜索网络", approvalInputLabel,
			"query: mygo 文档", "page: 1", "pageSize: 10")
	})
}

// TestApprovalPanelSkill：技能渲染器画技能名（命名空间原样保留）。
func TestApprovalPanelSkill(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-skill", Tool: "skill", Input: `{"skill_name":"plugin:release-notes"}`,
	})
	approvalWants(t, tt, "DimAgent 想要：使用技能", "skill_name: plugin:release-notes")
}

// TestApprovalPanelAgent：创建类子代理画头行与提示词，其余动作画行列表。
func TestApprovalPanelAgent(t *testing.T) {
	t.Run("创建", func(t *testing.T) {
		_, tt := approvalTester(t, ApprovalSnapshot{
			ID: "a-agent", Tool: "agent",
			Input: `{"action":"create","agentName":"reader","taskTitle":"读一遍需求","modelId":"m-1","prompt":"请把需求读一遍","effort":"high","forkContext":true}`,
		})
		approvalWants(t, tt, "DimAgent 想要：创建智能体", "reader", "读一遍需求", "m-1",
			"请把需求读一遍", "effort: high", "forkContext: true")
	})
	t.Run("查询", func(t *testing.T) {
		_, tt := approvalTester(t, ApprovalSnapshot{
			ID: "a-agent-list", Tool: "agent",
			Input: `{"action":"list","taskId":"t-1","agentName":"reader","reason":"查看进度"}`,
		})
		approvalWants(t, tt, "DimAgent 想要：查询智能体状态", approvalInputLabel,
			"action: list", "taskId: t-1", "agentName: reader", "reason: 查看进度")
	})
}

// TestApprovalPanelMCP：MCP 工具的服务器、工具与参数分行画出，标题仍写「使用工具」。
func TestApprovalPanelMCP(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-mcp", Tool: "github__create_issue", LongTerm: true,
		Input: `{"title":"崩溃","body":"复现步骤"}`,
	})
	approvalWants(t, tt, "DimAgent 想要：使用工具", approvalInputLabel,
		"server: github", "tool: create_issue", "title: 崩溃", "body: 复现步骤")
	approvalDoesNotWant(t, tt, "DimAgent 想要：github__create_issue")
}

// TestApprovalPanelImage：图片渲染器画缩略图位与入参里的尺寸。
func TestApprovalPanelImage(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-image", Tool: "view_image",
		Input: `{"path":"/work/shots/narrow.png","width":640,"height":480}`,
	})
	approvalWants(t, tt, "/work/shots/narrow.png", "640 x 480")
	if !tt.HasText("图片缩略图") {
		t.Fatalf("没有画缩略图位（没有像素时也不留空白）：\n%s", p0WindowText(tt))
	}
}

// TestApprovalPanelTodos：待办渲染器逐条画标记、内容、状态与优先级。
func TestApprovalPanelTodos(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-todos", Tool: "todo_write",
		Input: `{"todos":[` +
			`{"id":"1","content":"写渲染器","status":"completed","priority":"high"},` +
			`{"id":"2","content":"补测试","status":"in_progress","priority":"medium"},` +
			`{"id":"3","content":"更新需求","status":"pending","priority":"low"}]}`,
	})
	approvalWants(t, tt,
		"DimAgent 想要：更新任务",
		"1. [x] 写渲染器 - completed - high",
		"2. [~] 补测试 - in progress - medium",
		"3. [ ] 更新需求 - pending - low")
}

// TestApprovalDecisionsFollowTool：长期允许按钮的文案随工具切换，答复值分明细。
func TestApprovalDecisionsFollowTool(t *testing.T) {
	cases := []struct {
		name  string
		snap  ApprovalSnapshot
		long  string
		title string
	}{
		{"命令", ApprovalSnapshot{ID: "d1", Tool: "exec", LongTerm: true,
			Input: `{"command":"ls"}`}, approvalCopyAlwaysCommand, "DimAgent 想要：执行"},
		{"读文件", ApprovalSnapshot{ID: "d2", Tool: "read", LongTerm: true,
			Input: `{"path":"/work/a.go"}`}, approvalCopyAllowForSession, "DimAgent 想要：读取"},
		{"其它工具", ApprovalSnapshot{ID: "d3", Tool: "recall", LongTerm: true,
			Input: `{"query":"上次结论"}`}, approvalCopyAlwaysFile, "DimAgent 想要：使用工具"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, tt := approvalTester(t, tc.snap)
			approvalWants(t, tt, tc.title, approvalCopyReject, approvalCopyAllowOnce, tc.long)
			info, ok := tt.Access(tc.long)
			if !ok || info.Disabled {
				t.Fatalf("长期允许按钮不可点（事实说可以长期允许）：%+v（找到 %v）", info, ok)
			}
		})
	}

	// 服务端没有给出长期规则参数时，长期允许置灰，不假装能写。
	_, tt := approvalTester(t, ApprovalSnapshot{ID: "d4", Tool: "read",
		Input: `{"path":"/work/a.go"}`})
	info, ok := tt.Access(approvalCopyAllowForSession)
	if !ok || !info.Disabled {
		t.Fatalf("服务端没有长期规则时按钮仍可点：%+v（找到 %v）", info, ok)
	}
}

// TestApprovalMemberUnreviewable：成员审批按请求工具名认出来，只画审批 ID 与
// 「拒绝」，不编成员与分支。
func TestApprovalMemberUnreviewable(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-member", Tool: "member_approval_allow",
		Input: `{"approval_id":"apr-9"}`,
		Member: &ApprovalMemberSnapshot{
			State: ApprovalMemberUnavailable, ApprovalID: "apr-9",
		},
	})
	approvalWants(t, tt,
		approvalCopyMemberUnavailableTitle, approvalCopyMemberUnavailable,
		approvalCopyMemberIDPrefix+"apr-9", approvalCopyReject)
	approvalDoesNotWant(t, tt, approvalCopyAllowOnce, approvalCopyAlwaysCommand,
		approvalCopyAlwaysFile, "成员 ", "主分支 ")
	if info, ok := tt.Access(approvalCopyReject); !ok || info.Disabled {
		t.Fatalf("成员审批的「拒绝」不可点：%+v（找到 %v）", info, ok)
	}
}

// TestApprovalSandboxCard：沙箱越权卡按被拦资源换标题与说明，长期允许按网络 /
// 文件换文案。
func TestApprovalSandboxCard(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-sandbox", LongTerm: true,
		Sandbox: &ApprovalSandboxSnapshot{
			ResourceType: "domain", Resource: "example.com", Tool: "web_fetch",
		},
	})
	approvalWants(t, tt, approvalCopySandboxNetwork, "example.com",
		"web_fetch 需要访问该域名。", approvalCopyDontAllow, approvalCopyAllowOnce,
		approvalCopyAlwaysNetwork)

	_, file := approvalTester(t, ApprovalSnapshot{
		ID: "a-sandbox-file", LongTerm: true,
		Sandbox: &ApprovalSandboxSnapshot{
			ResourceType: "path", Access: "write", Resource: "/etc/hosts", Tool: "write",
		},
	})
	approvalWants(t, file, approvalCopySandboxWrite, "/etc/hosts", "write 需要写入该路径。",
		approvalCopyAlwaysFile)
}

// TestApprovalSubagentModelCard：子代理换模型卡画任务、模型、价格标记与回退说明。
func TestApprovalSubagentModelCard(t *testing.T) {
	_, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-model",
		SubagentModel: &ApprovalSubagentModelSnapshot{
			TaskTitle: "审计依赖", ModelName: "Seed 3 Pro", ProviderName: "Acme",
			FallbackModel: "Seed 2.1 Pro", Pricier: true,
		},
	})
	approvalWants(t, tt, approvalCopySubagentTitle, approvalCopySubagentTask+" · 审计依赖",
		"Seed 3 Pro", "Acme", approvalCopySubagentPricier,
		"拒绝时子任务将使用当前模型 Seed 2.1 Pro 继续",
		approvalCopySubagentReject, approvalCopySubagentAllow)
}

// TestApprovalHandoffBlocks：自动审批交回人工的两种提示块互斥且各写各的。
func TestApprovalHandoffBlocks(t *testing.T) {
	_, uncertain := approvalTester(t, ApprovalSnapshot{
		ID: "a-handoff", Tool: "exec", Input: `{"command":"ls"}`,
		Handoff: &ApprovalHandoffSnapshot{Kind: approvalHandoffUncertain, Reason: "命令改动了工作区外的文件"},
	})
	approvalWants(t, uncertain, approvalCopyNeedsDecision, "命令改动了工作区外的文件")
	approvalDoesNotWant(t, uncertain, approvalCopyAutoReviewFailed)

	_, failed := approvalTester(t, ApprovalSnapshot{
		ID: "a-handoff2", Tool: "exec", Input: `{"command":"ls"}`,
		Handoff: &ApprovalHandoffSnapshot{Kind: approvalHandoffFailed},
	})
	approvalWants(t, failed, approvalCopyAutoReviewFailed)
	approvalDoesNotWant(t, failed, approvalCopyNeedsDecision)
}

// TestApprovalSubmittingDisablesDecisions：提交中整组按钮置灰，文案不变。
func TestApprovalSubmittingDisablesDecisions(t *testing.T) {
	app, tt := approvalTester(t, ApprovalSnapshot{
		ID: "a-busy", Tool: "exec", LongTerm: true, Input: `{"command":"ls"}`,
	})
	app.approvalSubmitting = true
	tt.Frame()
	for _, label := range []string{approvalCopyReject, approvalCopyAllowOnce, approvalCopyAlwaysCommand} {
		info, ok := tt.Access(label)
		if !ok || !info.Disabled {
			t.Fatalf("提交中 %q 没有置灰：%+v（找到 %v）", label, info, ok)
		}
	}
	// 请求已答等结算（服务端事实）同样置灰。
	app.approvalSubmitting = false
	src := wiredSource()
	src.hasAppr = true
	src.approval = ApprovalSnapshot{ID: "a-pending", Tool: "exec", LongTerm: true,
		Input: `{"command":"ls"}`, Pending: true}
	pending := NewApp(Options{State: StateDefault, Source: src})
	pendingTester := gui.NewTester(pending.View, WindowWidth, WindowHeight)
	if info, ok := pendingTester.Access(approvalCopyAllowOnce); !ok || !info.Disabled {
		t.Fatalf("已答未结算时按钮仍可点：%+v（找到 %v）", info, ok)
	}
}

// TestApprovalFailureShowsErrorAndRetries：失败写进卡片自己的错误行，按钮恢复；
// 再点同一枚按钮就是重试，错误先清掉。
func TestApprovalFailureShowsErrorAndRetries(t *testing.T) {
	src := wiredSource()
	src.hasAppr = true
	src.approval = ApprovalSnapshot{ID: "a-fail", Tool: "exec", LongTerm: true,
		Input: `{"command":"ls -la"}`}
	intents := &approvalFailIntents{responded: make(chan string, 4), failures: 1}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if err := tt.Click(approvalCopyAllowOnce); err != nil {
		t.Fatalf("点不到「仅这次」：%v", err)
	}
	if got := waitApprovalDecision(t, intents.responded); got != "allow" {
		t.Fatalf("答复值 = %q，想要 allow", got)
	}
	waitApprovalError(t, app, tt, "应答未发出")

	approvalWants(t, tt, "应答未发出")
	if info, ok := tt.Access(approvalCopyAllowOnce); !ok || info.Disabled {
		t.Fatalf("失败后按钮没有恢复可点：%+v（找到 %v）", info, ok)
	}

	// 同一枚按钮再点一次：先清错误再提交，成功后错误行消失。
	if err := tt.Click(approvalCopyAllowOnce); err != nil {
		t.Fatalf("重试点不到「仅这次」：%v", err)
	}
	if got := waitApprovalDecision(t, intents.responded); got != "allow" {
		t.Fatalf("重试答复值 = %q，想要 allow", got)
	}
	tt.Frame()
	if app.approvalError != "" {
		t.Fatalf("重试成功后错误还在：%q", app.approvalError)
	}
	approvalDoesNotWant(t, tt, "应答未发出")
}

// TestApprovalKeysAnswerDecision：Enter / y 以「仅这次」答复，Esc / n 拒绝。
func TestApprovalKeysAnswerDecision(t *testing.T) {
	cases := []struct {
		name     string
		key      gui.Key
		expected string
	}{
		{"Enter", gui.KeyEnter, "allow"},
		{"y", gui.KeyY, "allow"},
		{"Esc", gui.KeyEscape, "deny"},
		{"n", gui.KeyN, "deny"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, tt := newTester(t, StateWaiting)
			tt.Key(0, tc.key)
			tt.Frame()
			if app.approvalDecision != tc.expected {
				t.Fatalf("%s 的答复 = %q，想要 %q", tc.name, app.approvalDecision, tc.expected)
			}
		})
	}
}

// TestApprovalSampleKeepsReferenceCopy：样例态（只有命令原文）仍是一张命令卡：
// 标题「执行」，三枚按钮与参考产品同文案。
func TestApprovalSampleKeepsReferenceCopy(t *testing.T) {
	_, tt := newTester(t, StateWaiting)
	approvalWants(t, tt, "DimAgent 想要：执行", approvalInputLabel, "ls -la",
		approvalCopyReject, approvalCopyAllowOnce, approvalCopyAlwaysCommand)
}

// approvalFailIntents 只接审批应答：前 failures 次失败，之后成功。
type approvalFailIntents struct {
	Intents
	responded chan string
	failures  int
}

func (i *approvalFailIntents) RespondApproval(_ context.Context, decision string) error {
	if i.responded != nil {
		i.responded <- decision
	}
	if i.failures > 0 {
		i.failures--
		return errors.New("应答未发出")
	}
	return nil
}

// waitApprovalDecision 等一次答复落到写入通道上。
func waitApprovalDecision(t *testing.T, responded chan string) string {
	t.Helper()
	select {
	case got := <-responded:
		return got
	case <-time.After(2 * time.Second):
		t.Fatal("审批答复没有发出")
		return ""
	}
}

// waitApprovalError 等后台写入的结果回到渲染线程：失败原因出现在卡片错误行上。
func waitApprovalError(t *testing.T, app *App, tt *gui.Tester, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if app.approvalError == want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("失败回执没有写进卡片：approvalError = %q，屏上：\n%s", app.approvalError, p0WindowText(tt))
}
