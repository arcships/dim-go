package ui

// 本文件是首片骨架的静态样例：内容固定，用来走查区域与状态，
// 不来自 App Server，也不代表任何真实会话。样例文案与顺序取自
// docs/domains/wireframes/a.html 的七个画面。

// commandRule 是一条命令规则：命令原文（内容）+ 决定 + 匹配方式。
type commandRule struct {
	command  string
	decision string // 允许 / 拒绝
	match    string // 精确 / 前缀
}

// taskState 是后台任务在界面上的三种形态。
type taskState int

const (
	taskRunning taskState = iota
	taskDone
	taskFailed
)

// task 是一个后台任务行：名字是内容，状态是样式。
type task struct {
	id    string
	name  string
	state taskState
}

type roundStatus int

const (
	roundDone roundStatus = iota
	roundRunning
	roundWaiting
	roundFailed
)

// op 是一次操作的折叠视图：动作、对象、结果，失败带原因。不带时长：时长
// 不进界面（§9.0 取向 2）。
type op struct {
	action string
	target string
	result string
	cause  string
	failed bool
}

// round 是一轮：用户消息、助手正文、轮次组头与展开后的操作清单。
type round struct {
	user   string
	reply  string
	status roundStatus
	// unknown 表示这一轮是否被创建还不确定：只显示占位与说明行，不画组头与
	// 操作行（没有事实就不编造状态）。
	unknown bool
	// note 是本轮下方的一行说明（「结果不确定」）。
	note string
	ops  []op
}

func doneOps() []op {
	return []op{
		{action: "读取", target: "internal/ui/composer.go", result: "完成"},
		{action: "改动", target: "internal/ui/composer.go", result: "完成"},
		{action: "改动", target: "internal/ui/timeline.go", result: "完成"},
		{action: "执行", target: "go test ./...", result: "失败",
			cause: "权限不足，命令没有执行", failed: true},
	}
}

// sampleRounds 返回当前状态的样例轮次。空态没有消息。
func sampleRounds(s State) []round {
	if s == StateEmpty {
		return nil
	}

	base := []round{
		{
			user:   "看看这次的失败",
			reply:  "消息在排队判定里丢了：运行中提交会覆盖队列。我把输入位改成“运行中先入队，结束后自动发出”。",
			status: roundDone,
			ops:    doneOps(),
		},
		{
			user:   "顺手把输入区的旧逻辑也看下",
			reply:  "输入区已经没有覆盖路径了：草稿与队列分开保存，队列只在上沿显示一行。",
			status: roundDone,
			ops: []op{
				{action: "读取", target: "internal/ui/composer.go", result: "完成"},
				{action: "改动", target: "internal/ui/composer.go", result: "完成"},
			},
		},
	}

	last := round{
		user:   "再跑一次",
		reply:  "已经改成排队：运行中的提交先入队，当前这一轮结束后自动发出。",
		status: roundDone,
		ops: []op{
			{action: "改动", target: "internal/ui/composer.go", result: "完成"},
			{action: "执行", target: "go test ./internal/ui", result: "完成"},
		},
	}

	// 富文本轮：默认态与富文本变体里都带一轮含块级结构的回复（U0 §3.1 M 系列）。
	rich := round{
		user:   "把排队的实现讲清楚",
		reply:  richReply,
		status: roundDone,
		ops: []op{
			{action: "读取", target: "internal/ui/composer.go", result: "完成"},
		},
	}

	switch s {
	case StateRunning:
		last.reply = "正在改写发送路径…"
		last.status = roundRunning
		last.ops = []op{
			{action: "读取", target: "internal/ui/composer.go", result: "完成"},
			{action: "改动", target: "internal/ui/composer.go", result: "完成"},
			{action: "改动", target: "internal/ui/timeline.go", result: "运行中"},
		}
	case StateWaiting:
		last = round{
			user:   "清一下临时目录",
			reply:  "需要跑一条命令，等你同意。",
			status: roundWaiting,
			ops: []op{
				{action: "读取", target: "tmp/", result: "完成"},
				{action: "执行", target: "rm -rf ./tmp/*", result: "等待同意"},
			},
		}
	case StateFailed:
		// 未收到事实：不知道这一轮有没有被创建，所以不画轮次组头，只留占位与
		// 说明行（U3 §6.1「未收到事实前不生成轮次」）。
		last = round{
			user:    "把临时目录清掉",
			unknown: true,
			note:    "结果不确定 · 正在核对",
		}
	}

	// 富文本轮排在最后：默认态因此"至少一轮回复含块级结构"，同时不打乱前几轮
	// 的位置（既有断言与截图都按前几轮取）。
	return append(base, last, rich)
}

// sessionSamples 是左栏的静态会话列表：按项目分组，行首标记当前状态。
func sessionSamples(s State) (sessions []string, markers []string) {
	if s == StateEmpty {
		return nil, nil
	}
	sessions = []string{"修发送路径", "参考", "旧实现对照"}
	switch s {
	case StateRunning:
		markers = []string{"●", "", ""}
	case StateWaiting:
		markers = []string{"◷", "", ""}
	case StateFailed:
		// 提交结果不确定：会话行给"还在处理"的时钟，而不是"失败"的叉——这一轮
		// 是否失败还没有事实（U3 §6.1）。
		markers = []string{"◷", "", ""}
	default:
		markers = []string{"", "", ""}
	}
	return sessions, markers
}

// branchSamples 是分支样例：当前分支与全部分支（名字是内容）。
func branchSamples() (current string, all []string) {
	return "feature/queue-rework", []string{"main", "feature/queue-rework"}
}

// taskSamples 是后台任务样例：一个在跑、一个完成。
func taskSamples() []task {
	return []task{
		{name: "go test ./...", state: taskRunning},
		{name: "npm run build", state: taskDone},
	}
}

// richReply 是默认态的富文本回复：一段话、两级标题、两种列表、行内代码、
// 代码块、引用、表格、链接——Markdown 的块级样式全都要在这条回复上被看见
// （U0 §3.1 的 M 系列；L20 用它做存在性检查）。
const richReply = "## 先看结论\n\n" +
	"发送路径的三处改动已经落地，**排队不再覆盖队列**。\n\n" +
	"### 改动清单\n\n" +
	"- 运行中的提交先入队\n" +
	"- 队列项在输入区上沿显示一行\n" +
	"- 这一轮结束后自动发出\n\n" +
	"1. 提交前检查草稿\n" +
	"2. 覆盖时保留原文\n" +
	"3. 结束后按序发出\n\n" +
	"`session.input.submit` 现在带 `queue` 参数。\n\n" +
	"```go\nfunc (a *App) send() {\n\ta.queue = append(a.queue, a.draft)\n}\n```\n\n" +
	"> 这条与 K2 的约定一致：运行中提交不打断当前轮。\n\n" +
	"| 场景 | 行为 |\n" +
	"|---|---|\n" +
	"| 未运行 | 直接发送 |\n" +
	"| 运行中 | 先入队 |\n\n" +
	"详见 [U3 的发送与排队](U3-interaction.md)。"

// extSamples 是扩展四类的样例数据（数量：技能 6、插件 8、MCP 4、钩子 3）。
// 名称与版本是内容；状态只用样式（可用 / 有更新 / 加载失败）。
func extSamples(kind extKind) []extItem {
	switch kind {
	case extSkills:
		return []extItem{
			{name: "代码评审", version: "v1.4.0", on: true},
			{name: "提交信息", version: "v0.9.2", on: true, status: extUpdate},
			{name: "测试生成", version: "v2.0.1", on: true},
			{name: "文档同步", version: "v0.3.0", on: false},
			{name: "依赖升级", version: "v1.0.4", on: false},
			{name: "发布检查", version: "v0.1.0", on: true, status: extError},
		}
	case extPlugins:
		return []extItem{
			{name: "GitHub", version: "v0.6.3", on: true},
			{name: "Linear", version: "v0.4.1", on: true},
			{name: "Sentry", version: "v0.2.0", on: false},
			{name: "Figma", version: "v0.1.7", on: false, status: extUpdate},
			{name: "Notion", version: "v0.3.2", on: true},
			{name: "Slack", version: "v0.5.0", on: true},
			{name: "Jira", version: "v0.2.8", on: false},
			{name: "Datadog", version: "v0.1.1", on: false, status: extError},
		}
	case extMCP:
		return []extItem{
			{name: "filesystem", version: "v0.9.0", on: true},
			{name: "postgres", version: "v0.5.2", on: true, status: extUpdate},
			{name: "playwright", version: "v0.3.0", on: false},
			{name: "sentry", version: "v0.2.1", on: false},
		}
	default:
		return []extItem{
			{name: "提交前检查", version: "v1.0.0", on: true},
			{name: "保存时格式化", version: "v1.0.0", on: true},
			{name: "推送后通知", version: "v0.4.0", on: false},
		}
	}
}

// cronSamples 是定时任务的样例（S49–S54：任务、计划、下次时间、历史）。
func cronSamples() []cronTask {
	return []cronTask{
		{name: "每晚依赖检查", schedule: "每天 02:00", next: "明天 02:00", on: true, result: "上次成功"},
		{name: "每周汇总", schedule: "周一 09:00", next: "周一 09:00", on: true, result: "上次成功"},
		{name: "清理临时文件", schedule: "每天 02:00", next: "明天 02:00", on: false, result: "已停用"},
		{name: "同步文档站", schedule: "每 6 小时", next: "今天 18:00", on: false, result: "上次失败", status: extError},
	}
}

// runtimeSamples 是运行时组件（T13–T15 的样例）。
func runtimeSamples() []runtimePart {
	return []runtimePart{
		{name: "dim-app-server", version: "v3.4.1"},
		{name: "protocol", version: "v3"},
		{name: "终端组件", version: "v0.2.0", status: extUpdate},
		{name: "渲染组件", version: "v0.1.3", status: extError},
	}
}
