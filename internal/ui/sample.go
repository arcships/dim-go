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

// op 是一次操作的折叠视图：序号、动作、对象、结果、耗时，失败带原因。
type op struct {
	action string
	target string
	result string
	took   string
	cause  string
	failed bool
}

// round 是一轮：用户消息、助手正文、过程摘要行与展开后的操作清单。
type round struct {
	user   string
	reply  string
	status roundStatus
	took   string
	// note 是本轮下方的一行说明（目前只用于“结果不确定”）。
	note string
	ops  []op
}

func doneOps() []op {
	return []op{
		{action: "读取", target: "internal/ui/composer.go", result: "完成", took: "0.8s"},
		{action: "改动", target: "internal/ui/composer.go", result: "完成", took: "0.3s"},
		{action: "改动", target: "internal/ui/timeline.go", result: "完成", took: "0.4s"},
		{action: "执行", target: "go test ./...", result: "失败", took: "2.1s",
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
			took:   "12s",
			ops:    doneOps(),
		},
		{
			user:   "顺手把输入区的旧逻辑也看下",
			reply:  "输入区已经没有覆盖路径了：草稿与队列分开保存，队列只在上沿显示一行。",
			status: roundDone,
			took:   "6s",
			ops: []op{
				{action: "读取", target: "internal/ui/composer.go", result: "完成", took: "0.9s"},
				{action: "改动", target: "internal/ui/composer.go", result: "完成", took: "0.5s"},
			},
		},
	}

	last := round{
		user:   "再跑一次",
		reply:  "已经改成排队：运行中的提交先入队，当前这一轮结束后自动发出。",
		status: roundDone,
		took:   "4s",
		ops: []op{
			{action: "改动", target: "internal/ui/composer.go", result: "完成", took: "0.6s"},
			{action: "执行", target: "go test ./internal/ui", result: "完成", took: "1.2s"},
		},
	}

	switch s {
	case StateRunning:
		last.reply = "正在改写发送路径…"
		last.status = roundRunning
		last.took = "12s"
		last.ops = []op{
			{action: "读取", target: "internal/ui/composer.go", result: "完成", took: "0.8s"},
			{action: "改动", target: "internal/ui/composer.go", result: "完成", took: "0.4s"},
			{action: "改动", target: "internal/ui/timeline.go", result: "运行中", took: "1.1s"},
		}
	case StateWaiting:
		last = round{
			user:   "清一下临时目录",
			reply:  "需要跑一条命令，等你同意。",
			status: roundWaiting,
			took:   "—",
			ops: []op{
				{action: "读取", target: "tmp/", result: "完成", took: "0.2s"},
				{action: "执行", target: "rm -rf ./tmp/*", result: "等待同意", took: "—"},
			},
		}
	case StateFailed:
		last = round{
			user:   "把临时目录清掉",
			reply:  "这次提交没有走完，结果还在核对。",
			status: roundFailed,
			took:   "2.1s",
			note:   "结果不确定 · 这次提交可能已经送达，正在核对，不会自动重发。",
			ops: []op{
				{action: "读取", target: "tmp/", result: "完成", took: "0.2s"},
				{action: "执行", target: "rm -rf ./tmp/*", result: "失败", took: "2.1s",
					cause: "权限不足，命令没有执行", failed: true},
			},
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
		markers = []string{"✕", "", ""}
	default:
		markers = []string{"", "", ""}
	}
	return sessions, markers
}
