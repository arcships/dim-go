package ui

import "time"

// 本文件是首片骨架的静态样例：内容固定，用来走查区域与状态，
// 不来自 App Server，也不代表任何真实会话。样例文案与顺序取自
// docs/domains/wireframes/a.html 的七个画面。

// taskState 是后台任务在界面上的四种形态（K-05–K-08）。
type taskState int

const (
	taskRunning taskState = iota
	taskDone
	taskFailed
	// taskStopped 是「已停止」（取消 / 被信号结束）：图标是方块，文案取已停止。
	taskStopped
	// taskStopping 是「已发停止、还在等结束」（K-15）：既不是已完成，也不是已
	// 停止；等结束事实到了才换状态词。
	taskStopping
	// taskNoStatus 是读到任务、读不到状态时的兜底（K-63 的「状态未知」）。
	taskNoStatus
)

// taskKind 是任务行的种类：后台进程或唤醒（K-01 / K-02 两个子标签）。
type taskKind int

const (
	taskProcess taskKind = iota
	taskWakeup
)

// task 是一个后台任务行：名字是内容，状态是样式。
//
// 事实口径：id / name / state 来自会话快照的未结束后台任务；pid、结束时刻、
// 触发时刻、命令行与日志状态在 Go 侧拿不到时留空（详情按段拼接，空段不画），
// 不编造。
type task struct {
	id    string
	kind  taskKind
	name  string
	state taskState
	// pending 表示已经发出停止、还在等结束事实（K-15 的旋转圈）。
	pending bool
	// branch 是任务所属分支（K-11 徽标；main 不画）。
	branch string
	// startedAt 是开始时刻的墙钟文本（K-12 / K-21 / K-28）。
	startedAt string
	// startAt 是同一时刻的 time 形态（运行时长按它算）；零值表示没有。
	startAt time.Time
	// endedAt 是结束时刻的墙钟文本（K-22–K-24 / K-28）。
	endedAt string
	// pid 是进程号（K-28）；空表示没有这项事实。
	pid string
	// toolName 是发起该任务的工具名（K-28）；空表示没有这项事实。
	toolName string
	// command 是命令行原文（K-29）；空表示没有这项事实。
	command string
	// logTruncated 表示输出达到服务端上限（K-32）。
	logTruncated bool
	// logFailed 表示输出读取失败（K-33），重连后清掉（K-34）。
	logFailed bool
	// wakeAt 是唤醒触发时刻（K-35），wakeBody 是唤醒提示正文（K-36）。
	wakeAt   string
	wakeBody string
}

type roundStatus int

const (
	roundDone roundStatus = iota
	roundRunning
	roundWaiting
	roundFailed
	// roundStopped 是运行被用户停止的一轮：标题写「已停止」，另给一行「用户中断」
	// （TL-47 / TL-60），不与失败混用。
	roundStopped
)

// op 是一次操作的折叠视图：动作、对象、结果，失败带原因。差异计数只在
// 工具行上出现；时长不进界面（§9.0 取向 2）。
type op struct {
	action  string
	target  string
	result  string
	cause   string
	input   string
	output  string
	added   int
	removed int
	failed  bool
	// terminal 表示这是终端类工具（执行）的事实行：失败时详情用「终端错误详情」
	// 收起（TL-59）。样例行不置这一位，仍按输入 / 输出 / 失败原因三段画。
	terminal bool
}

// round 是一轮：用户消息、助手正文、思考块、工具分组与轮末块。
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
	// at 是用户消息的发送时间（悬停动作行上显示）；空表示没有。
	at string
	// thinking / thinkingDuration 是思考块的原文与时长。
	thinking         string
	thinkingDuration string
	// duration 是运行中轮末的「正在处理 · {时长}」。
	duration string
	// retrySeconds 是失败后的自动重试倒计时；0 表示没有重试通知。
	retrySeconds int
	// fileChanges 是本轮改动的文件；artifacts 是本轮产物。
	fileChanges []fileChange
	artifacts   []artifact
	// divider 是轮前的分隔文案（压缩/截断）；空表示没有。
	divider string
	// images / attachments / quotes / marks / wakeup 是用户消息里的附件与引用
	// （TL-21–TL-25）。它们是视图样例字段：RowSnapshot 还没有这些事实（DEC-07），
	// 线上一律留空，样例与测试可以直接给出条目；留空时这一区整块不画。
	images      []string
	attachments []messageAttachment
	quotes      []messageQuote
	marks       []messageMark
	wakeup      string
}

// messageAttachment 是用户消息里的一个附件 chip（TL-22）；missing 表示附件
// 已不可用（R-TL-212），降对比并写「不可用」。
type messageAttachment struct {
	name    string
	missing bool
}

// messageQuote 是一条已发送的引用（TL-23）：标题或首行摘要 + 被引用的片段。
type messageQuote struct {
	title string
	text  string
}

// messageMark 是一条标注（TL-24）：序号 + 标注原文。
type messageMark struct {
	num  int
	text string
}

// fileChange 是一个改动文件。
type fileChange struct {
	path    string
	status  string
	added   int
	removed int
}

// artifact 是一个产物。
type artifact struct {
	name string
	// missing 是「文件已不存在」的事实（A-11）：整行半透明、行内标「已删除」，
	// 不再给「打开」。没有这条事实时按文件存在画。
	missing bool
	// canOpen 是「能否打开」的事实（A-12）：nil 表示没有这条事实（按可打开画）；
	// 指向 false 时「打开」的位置换成「无法预览文件」。
	canOpen *bool
}

func doneOps() []op {
	return []op{
		{action: "读取", target: "internal/ui/composer.go", result: "完成",
			input: "path: internal/ui/composer.go"},
		{action: "改动", target: "internal/ui/composer.go", result: "完成",
			added: 19, removed: 2},
		{action: "改动", target: "internal/ui/timeline.go", result: "完成",
			added: 6, removed: 4},
		{action: "执行", target: "go test ./...", result: "失败",
			input: "command: go test ./...", output: "$ go test ./...\nFAIL\t./internal/ui",
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
			user:             "看看这次的失败",
			reply:            "消息在排队判定里丢了：运行中提交会覆盖队列。我把输入位改成“运行中先入队，结束后自动发出”。",
			status:           roundDone,
			ops:              doneOps(),
			at:               "星期二 21:22",
			thinking:         "用户要看失败原因。先读输入区与时间线，确认排队判定覆盖了草稿。",
			thinkingDuration: "0.3s",
		},
		{
			user:   "顺手把输入区的旧逻辑也看下",
			reply:  "输入区已经没有覆盖路径了：草稿与队列分开保存，队列只在上沿显示一行。",
			status: roundDone,
			ops: []op{
				{action: "读取", target: "internal/ui/composer.go", result: "完成",
					input: "path: internal/ui/composer.go"},
				{action: "改动", target: "internal/ui/composer.go", result: "完成",
					added: 8, removed: 3},
			},
			at:               "星期二 21:23",
			thinking:         "检查草稿与队列的保存位置，确认没有第二条覆盖路径。",
			thinkingDuration: "0.5s",
			divider:          "上下文已压缩",
		},
	}

	last := round{
		user:   "再跑一次",
		reply:  "已经改成排队：运行中的提交先入队，当前这一轮结束后自动发出。",
		status: roundDone,
		ops: []op{
			{action: "改动", target: "internal/ui/composer.go", result: "完成",
				added: 19, removed: 2},
			{action: "执行", target: "go test ./internal/ui", result: "完成",
				input: "command: go test ./internal/ui", output: "ok  \tgithub.com/arcships/dimagent-mygo/internal/ui"},
		},
		at: "星期二 21:25",
		fileChanges: []fileChange{
			{path: "internal/ui/composer.go", status: "M", added: 19, removed: 2},
		},
	}

	// 富文本轮：默认态与富文本变体里都带一轮含块级结构的回复（U0 §3.1 M 系列）。
	rich := round{
		user:   "把排队的实现讲清楚",
		reply:  richReply,
		status: roundDone,
		ops: []op{
			{action: "读取", target: "internal/ui/composer.go", result: "完成",
				input: "path: internal/ui/composer.go"},
		},
		at:               "星期二 21:26",
		thinking:         "把改动按结论、清单与代码块组织，便于逐条核对。",
		thinkingDuration: "0.4s",
		artifacts:        []artifact{{name: "排队说明.md"}},
	}

	switch s {
	case StateRunning:
		last.reply = "正在改写发送路径…"
		last.status = roundRunning
		last.duration = "1m 28s"
		last.ops = []op{
			{action: "读取", target: "internal/ui/composer.go", result: "完成",
				input: "path: internal/ui/composer.go"},
			{action: "改动", target: "internal/ui/composer.go", result: "完成",
				added: 19, removed: 2},
			{action: "改动", target: "internal/ui/timeline.go", result: "运行中",
				added: 6, removed: 4},
		}
	case StateWaiting:
		last = round{
			user:   "清一下临时目录",
			reply:  "需要跑一条命令，等你同意。",
			status: roundWaiting,
			ops: []op{
				{action: "读取", target: "tmp/", result: "完成"},
				{action: "执行", target: "rm -rf ./tmp/*", result: "等待同意",
					input: "command: rm -rf ./tmp/*"},
			},
		}
	case StateFailed:
		// 先展示一次确实失败的轮次（失败文案 + 自动重试），再给一条「未收到事实」
		// 的占位轮：不知道它有没有被创建，所以不画组头（U3 §6.1）。
		last = round{
			user:   "再跑一次完整测试",
			reply:  "命令没有拿到执行权限，这一轮到此结束。",
			status: roundFailed,
			ops: []op{
				{action: "读取", target: "internal/ui/composer.go", result: "完成"},
				{action: "执行", target: "go test ./...", result: "失败",
					input: "command: go test ./...", output: "$ go test ./...\nFAIL\t./internal/ui",
					cause: "权限不足，命令没有执行", failed: true},
			},
			at:               "星期二 21:27",
			thinking:         "先读失败点，再重跑完整测试确认。",
			thinkingDuration: "0.6s",
			retrySeconds:     5,
		}
		return append(base, last, round{
			user:    "把临时目录清掉",
			unknown: true,
			note:    "结果不确定 · 正在核对",
		}, rich)
	}

	// 富文本轮排在最后：默认态因此"至少一轮回复含块级结构"，同时不打乱前几轮
	// 的位置（既有断言与截图都按前几轮取）。
	return append(base, last, rich)
}

// sampleSessionName 是 -demo 标题行用的样例会话名；接上 Source 时不用。
func sampleSessionName(state State) string {
	if state == StateEmpty {
		return "新会话"
	}
	names, _ := sessionSamples(state)
	if len(names) == 0 {
		return "新会话"
	}
	return names[0]
}

// 样例项目分组的名字；接上 Source 时侧栏用中性「会话」分组。
const (
	sampleProjectA = "项目 A"
	sampleProjectB = "项目 B"
)

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

// sampleSessionPlaces 是样例会话的工作目录：两条在 dim-agent、一条在 rutis，
// 用来走查按目录分组（组名取目录末段）。这是展示用的占位目录，不是本机路径。
func sampleSessionPlaces(s State) []SessionPlaceSnapshot {
	if s == StateEmpty {
		return nil
	}
	return []SessionPlaceSnapshot{
		{Cwd: "/example/dim-agent"},
		{Cwd: "/example/rutis"},
		{Cwd: "/example/dim-agent"},
	}
}

// branchSamples 是分支样例：当前分支与全部分支（名字是内容）。
func branchSamples() (current string, all []string) {
	return "feature/queue-rework", []string{"main", "feature/queue-rework"}
}

// taskSamples 是后台任务样例：一个在跑、一个完成。
//
// 样例带命令行、pid、工具名与时刻，用来走查详情的元信息行（K-28）与命令行
// （K-29）；这些字段只有 -demo 才有，连着 Source 时由服务端事实决定有无。
func taskSamples() []task {
	started := time.Now().Add(-12 * time.Second)
	return []task{
		{
			name: "go test ./...", kind: taskProcess, state: taskRunning,
			command: "go test ./...", toolName: "exec", pid: "48213",
			startedAt: taskClock(started), startAt: started,
		},
		{
			name: "npm run build", kind: taskProcess, state: taskDone,
			command: "npm run build", toolName: "exec", pid: "47902",
			startedAt: "21:10", endedAt: "21:12",
		},
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

// SampleSessionNames 是样例模式的会话名（与左栏行同序）。入口在样例模式下用
// 它核对深链与开窗的会话名：样例没有真实会话标识，名字就是它的标识。
func SampleSessionNames(s State) []string {
	names, _ := sessionSamples(s)
	return names
}
