package ui

// 本文件是 Fork 的界面与动作：「在新分支继续」与「从此处分叉」都以某一轮结束处
// 为切点建分支，走既有的 Intents.CreateBranch（协议不变），成功后切到新分支并
// 在标题行下方写切点。
//
// 为什么不是会话复制：协议 v3 没有会话级分叉的写入通道（catalog 只有领域类型），
// 而分支创建有。因此这里做的是「按轮切分支」，Fork 的会话副本对话框仍留空（见
// requirements-global §7.2 与 §7.4 的实现口径）。
//
// 不可用时一律置灰并把原因写在菜单项 / 可读名 / 提示里，不拿别的轮次顶替。

import (
	"context"
	"fmt"
	"strings"
)

// Fork 不可用的原因（用户可见；集中在这里，入口处复用同一句）。
const (
	// forkNoteNoTurn 是拿不到这一行所属轮次时的原因。
	forkNoteNoTurn = "这条消息还没有对应的轮次，无法确定切点"
	// forkNoteTurnRunning 是这一轮还没结束时的原因：切点只认已结束的轮次。
	forkNoteTurnRunning = "这一轮还没结束，等结束后再分叉"
	// forkNoteNoTurns 是会话还没有可切出的轮次时的原因。
	forkNoteNoTurns = "还没有可切出的轮次"
	// forkNoteNoPrompt 是输入框没有草稿时的原因：新分支的第一条提问取输入框
	// 草稿，草稿为空就不发——界面不替用户编一句话（协议要求非空 prompt，编出来
	// 的那句会变成新分支里用户没写过的第一条消息）。
	forkNoteNoPrompt = "需要先写一句分支说明"
	// forkNoteJumpPending 是分支建好了但分支清单还没刷新时的回执。
	forkNoteJumpPending = "已建分支；分支清单还没刷新，未切换"
)

// forkTurnForRow 取时间线第 index 行对应的轮次序号。
//
// 有 Source 时只认快照里的 RowSnapshot.TurnIndex（导出与分叉用同一份事实）；
// 样例模式每一轮就是一行，取行号 + 1。第二个返回值非空表示不可用及原因。
func (a *App) forkTurnForRow(index int) (int, string) {
	if snap, ok := a.liveSession(); ok {
		if index < 0 || index >= len(snap.Rows) {
			return 0, forkNoteNoTurn
		}
		row := snap.Rows[index]
		if row.TurnIndex <= 0 {
			return 0, forkNoteNoTurn
		}
		switch row.Status {
		case RowStatusRunning, RowStatusPending:
			return 0, forkNoteTurnRunning
		}
		return row.TurnIndex, ""
	}
	if index < 0 || index >= len(a.rounds) {
		return 0, forkNoteNoTurn
	}
	if a.rounds[index].status == roundRunning {
		return 0, forkNoteTurnRunning
	}
	return index + 1, ""
}

// forkBlocked 是写通道缺失时的原因；空表示可写。
func (a *App) forkBlocked() string {
	if a.wired() && a.intents == nil {
		return noteNoWriteChannel
	}
	return ""
}

// forkReason 合并一次分叉的不可用原因：轮次问题优先于写通道问题，其次是没有
// 起始提问（先看得见这一行为什么不能切）。
func (a *App) forkReason(turn int, reason string) string {
	if reason != "" {
		return reason
	}
	if reason := a.forkBlocked(); reason != "" {
		return reason
	}
	return a.forkPromptReason()
}

// forkPromptReason 是「新分支的第一条提问」缺失时的原因：分支的起始提问只取
// 输入框草稿，草稿为空就没有可发的提问——这时置灰，不代写一句用户没说过的话。
func (a *App) forkPromptReason() string {
	if strings.TrimSpace(a.draft) == "" {
		return forkNoteNoPrompt
	}
	return ""
}

// forkParent 是切出分支的父分支：快照的当前分支；样例取样例分支名。
func (a *App) forkParent() string {
	if snap, ok := a.liveSession(); ok {
		if branch := strings.TrimSpace(snap.Branch); branch != "" {
			return branch
		}
	}
	if a.branch != "" {
		return a.branch
	}
	return "main"
}

// forkStarter 是分支的起始提问：只取输入框草稿（用户正在写的那一句）。没有草稿
// 时返回空串并给出原因——协议要求提示词非空，但界面不替用户编一句：编出来的
// 那句话会作为用户消息落进新分支。
func (a *App) forkStarter() (string, string) {
	prompt := strings.TrimSpace(a.draft)
	if prompt == "" {
		return "", forkNoteNoPrompt
	}
	return prompt, ""
}

// forkAtTurn 以某一轮结束处建分支：成功回调里切到新分支并写切点。
func (a *App) forkAtTurn(turn int, label string) {
	if turn <= 0 {
		a.headerNote = forkNoteNoTurn
		return
	}
	parent := a.forkParent()
	prompt, reason := a.forkStarter()
	if reason != "" {
		// 没有用户写的起始提问：不发任何写入，只写原因（入口也已置灰）。
		a.headerNote = reason
		return
	}
	if a.intents == nil {
		if a.wired() {
			a.headerNote = noteNoWriteChannel
			return
		}
		// 样例：只改视图里的分支名，不发任何写入。
		a.branch = fmt.Sprintf("%s-%d", nonEmpty(a.branch, "main"), turn)
		if !containsString(a.branches, a.branch) {
			a.branches = append(a.branches, a.branch)
		}
		a.headerNote = fmt.Sprintf("已切到分支 %s（切点：第 %d 轮结束处）", a.branch, turn)
		return
	}
	before := a.branchIDs()
	a.dispatch(label, func(ctx context.Context) error {
		return a.intents.CreateBranch(ctx, parent, turn, prompt)
	}, func() {
		a.jumpToNewBranch(before, turn)
	})
}

// branchIDs 取当前分支清单的标识集合：建分支前后做差找出新分支。
func (a *App) branchIDs() map[string]bool {
	out := map[string]bool{}
	if snap, ok := a.liveSession(); ok {
		for _, branch := range snap.Branches {
			out[branch.ID] = true
		}
	}
	for _, name := range a.branches {
		out[name] = true
	}
	return out
}

// jumpToNewBranch 在分支建好后切到新分支：新标识按建分支前的集合做差取；
// 清单还没刷新时只写回执，不拿旧分支冒充新分支。
func (a *App) jumpToNewBranch(before map[string]bool, turn int) {
	next := ""
	if snap, ok := a.liveSession(); ok {
		for _, branch := range snap.Branches {
			if !before[branch.ID] {
				next = branch.ID
				break
			}
		}
	}
	if next == "" {
		a.headerNote = fmt.Sprintf("%s（切点：第 %d 轮结束处）", forkNoteJumpPending, turn)
		return
	}
	a.headerNote = fmt.Sprintf("已从第 %d 轮结束处分叉，正在切到分支 %s", turn, next)
	a.dispatch("切换分支", func(ctx context.Context) error {
		return a.intents.SelectBranch(ctx, next)
	}, func() {
		a.headerNote = fmt.Sprintf("已切到分支 %s（切点：第 %d 轮结束处）", next, turn)
	})
}

// forkLatest 是「在新分支继续」：以最后一轮结束处为切点。最后一轮没结束、
// 没有轮次或没有会话时给原因，不往前找别的轮次。
func (a *App) forkLatest() (int, string) {
	if snap, ok := a.liveSession(); ok {
		if snap.LastTurnEnded && snap.LastTurnIndex > 0 {
			return snap.LastTurnIndex, ""
		}
		if snap.Running || snap.TurnCount > 0 {
			return 0, forkNoteTurnRunning
		}
		return 0, forkNoteNoTurns
	}
	if a.wired() {
		// 连着事实但没有打开的会话：没有可切的轮次，原因写“没有打开的会话”。
		return 0, noteNoSessionID
	}
	if len(a.rounds) == 0 {
		return 0, forkNoteNoTurns
	}
	last := len(a.rounds) - 1
	if a.rounds[last].status == roundRunning {
		return 0, forkNoteTurnRunning
	}
	return last + 1, ""
}

// forkLatestFromHeader 是标题栏 ⋯ 的「在新分支继续」落点。
func (a *App) forkLatestFromHeader() {
	if a.wired() && a.intents == nil {
		a.headerNote = noteNoWriteChannel
		return
	}
	turn, reason := a.forkLatest()
	if reason != "" {
		a.headerNote = reason
		return
	}
	a.forkAtTurn(turn, "在新分支继续")
}
