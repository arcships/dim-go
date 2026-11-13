package ui

// 本文件是首片骨架的静态样例：内容固定，用来走查区域与状态，
// 不来自 App Server，也不代表任何真实会话。样例文案与顺序取自
// docs/domains/wireframes/a.html 的七个画面。

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

	return append(base, last)
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
