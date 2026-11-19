package uiwire

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/workspace"
)

// 本文件是时间线的行投射：把 app.SessionView 的行事实换成界面要的
// ui.RowSnapshot。它只映射，不推断：投影没给的字段（差异行数、产物）一律留空，
// 不用界面上的样子倒推事实。

// runChangesTimeout 是一次「本轮改了什么」读取的上限；读不到就不出改动块。
const runChangesTimeout = 10 * time.Second

// changeReadSlots 是同时在读的改动数量上限：一次读取要扫一遍会话历史，
// 同时放太多会把会话连接压满。
const changeReadSlots = 2

// rows 把一行一行的事实投成时间线的行。
//
// 结构规则：
//   - 连续的 ≥2 个工具行合成一个 toolGroup（Children 是工具行），单个工具行
//     仍是 tool；工具组不跨轮、也不跨中间的正文；
//   - 一轮收口后补本轮的改动块（来自 workspace 的声明级改动），轮以失败收口时
//     补一行 error；
//   - 认不出的行种类在正文非空时按助手正文画（纯文本回退，不丢内容）。
func (w *App) rows(v app.SessionView) []ui.RowSnapshot {
	out := make([]ui.RowSnapshot, 0, len(v.Rows)+len(v.Turns))
	turns := make(map[int]projection.Turn, len(v.Turns))
	for _, turn := range v.Turns {
		turns[turn.Index] = turn
	}
	var group []ui.RowSnapshot
	current := 0
	// currentBranch 是当前结算轮次所属的分支：失败行与改动块没有自己的行事实，
	// 沿用这一轮最后一行的分支（导出按分支取范围要用）。
	currentBranch := ""
	flushGroup := func() {
		switch len(group) {
		case 0:
		case 1:
			out = append(out, group[0])
		default:
			out = append(out, ui.RowSnapshot{
				Kind: ui.RowToolGroup, Status: groupStatus(group), Children: group,
				BranchID: group[0].BranchID, TurnIndex: group[0].TurnIndex,
			})
		}
		group = nil
	}
	// closeTurn 结算一轮：工具组落行，再补失败行与本轮改动块。
	closeTurn := func(index int) {
		flushGroup()
		if index <= 0 {
			return
		}
		turn, known := turns[index]
		if known && turnFailed(turn.Outcome) {
			out = append(out, ui.RowSnapshot{
				Kind: ui.RowError, Text: turn.Outcome,
				BranchID: currentBranch, TurnIndex: index,
			})
		}
		if changes := w.runChangeRows(v.SessionID, turn); len(changes) > 0 {
			for i := range changes {
				changes[i].BranchID, changes[i].TurnIndex = currentBranch, index
			}
			out = append(out, ui.RowSnapshot{Kind: ui.RowFileChanges, Children: changes})
		}
	}
	for _, row := range v.Rows {
		if row.TurnIndex != current {
			closeTurn(current)
			current = row.TurnIndex
		}
		if row.BranchID != "" {
			currentBranch = row.BranchID
		}
		snapshot, ok := rowSnapshot(row)
		if !ok {
			continue
		}
		snapshot.BranchID = row.BranchID
		snapshot.TurnIndex = row.TurnIndex
		// 工具行先攒着：连续的一段才是一个组。
		if row.Kind == projection.RowTool {
			group = append(group, snapshot)
			continue
		}
		flushGroup()
		out = append(out, snapshot)
	}
	closeTurn(current)
	return out
}

// rowSnapshot 投射单行事实。ok 为假表示这一行没有可画的内容。
//
// 这里不产 ui.RowRetry：协议 v3 没有"还剩多久重试"这条事实，投影也没有对应的
// 行种。界面按 RetryIn 画倒计时、RetryIn 为 0 就不画；将来若有了剩余时间，按
// ui.RowSnapshot.RetryIn 给（不拿 CreatedAt/UpdatedAt 的差值当剩余）。
func rowSnapshot(row projection.Row) (ui.RowSnapshot, bool) {
	status := rowStatus(row.Status)
	switch row.Kind {
	case projection.RowUserMessage:
		return ui.RowSnapshot{Kind: ui.RowUser, Text: row.Text, Status: status}, true
	case projection.RowAssistantText:
		return ui.RowSnapshot{
			Kind: ui.RowAssistant, Text: row.Text, Status: status, Duration: rowDuration(row),
		}, true
	case projection.RowThinking:
		return ui.RowSnapshot{
			Kind: ui.RowThinking, Text: row.Text, Status: status, Duration: rowDuration(row),
		}, true
	case projection.RowTool:
		return toolSnapshot(row), true
	case projection.RowSystem:
		// 服务端的系统提示目前只有上下文压缩一种，按压缩分隔画。
		return ui.RowSnapshot{Kind: ui.RowCompression, Text: row.Text, Status: status}, true
	}
	// 认不出的种类：正文还在就按助手正文画，不丢内容；没有正文就没有可画的。
	if strings.TrimSpace(row.Text) == "" {
		return ui.RowSnapshot{}, false
	}
	return ui.RowSnapshot{Kind: ui.RowAssistant, Text: row.Text, Status: status}, true
}

// toolSnapshot 投射一次工具调用：Title 是动作、Args 是参数摘要、Text 是结果原文。
func toolSnapshot(row projection.Row) ui.RowSnapshot {
	out := ui.RowSnapshot{Kind: ui.RowTool, Status: rowStatus(row.Status), Duration: rowDuration(row)}
	tool := row.Tool
	if tool == nil {
		return out
	}
	name := strings.TrimSpace(tool.Name)
	out.Title = toolAction(name)
	if out.Title == "" {
		out.Title = name
	}
	out.Args = toolArgs(tool.Input)
	out.Text = strings.TrimSpace(tool.Result)
	return out
}

// rowStatus 把投影的行状态换成界面词；未知状态原样给出，视图按结束画。
func rowStatus(status projection.RowStatus) string {
	switch status {
	case projection.StatusPending:
		return ui.RowStatusPending
	case projection.StatusRunning:
		return ui.RowStatusRunning
	case projection.StatusError:
		return ui.RowStatusFailed
	case projection.StatusCancelled:
		return ui.RowStatusCancelled
	case projection.StatusComplete:
		return ui.RowStatusDone
	default:
		return string(status)
	}
}

// rowDuration 是一行从首条事实到最近一条事实的时间；没有起点或没有增长时为 0。
func rowDuration(row projection.Row) time.Duration {
	if row.CreatedAt <= 0 || row.UpdatedAt <= row.CreatedAt {
		return 0
	}
	return time.Duration(row.UpdatedAt-row.CreatedAt) * time.Millisecond
}

// groupStatus 是一组工具行的状态：还在跑就是运行中，有失败就是失败，其余完成。
func groupStatus(rows []ui.RowSnapshot) string {
	status := ui.RowStatusDone
	for _, row := range rows {
		switch row.Status {
		case ui.RowStatusRunning, ui.RowStatusPending:
			return ui.RowStatusRunning
		case ui.RowStatusFailed:
			status = ui.RowStatusFailed
		}
	}
	return status
}

// turnFailed 判断一轮的结果词是不是失败（服务端只给结果词，不给原因）。
func turnFailed(outcome string) bool {
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "failed", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

// runChangeRows 取一轮的改动文件行；还没读到或读失败时为空（不拿空清单冒充"没改动"）。
func (w *App) runChangeRows(sessionID string, turn projection.Turn) []ui.RowSnapshot {
	if strings.TrimSpace(turn.ID) == "" {
		return nil
	}
	w.changesMu.Lock()
	defer w.changesMu.Unlock()
	return w.changesByRun[changeKey(sessionID, turn.ID)]
}

// changeKey 是一轮改动缓存与「已问过」标记的键。
func changeKey(sessionID, runID string) string { return sessionID + "\x00" + runID }

// requestRunChanges 为已经收口的轮次排一次改动读取；同一轮只问一次。
//
// 界面每帧读 Session()，所以这里只登记与派发，不等待：结果到了再触发重绘。
// 只有带工具行的一轮才可能有改动声明，没有工具行的轮不去读。
func (w *App) requestRunChanges(v app.SessionView) {
	turnsWithTools := make(map[int]bool, len(v.Turns))
	for _, row := range v.Rows {
		if row.Kind == projection.RowTool {
			turnsWithTools[row.TurnIndex] = true
		}
	}
	for _, turn := range v.Turns {
		if turn.Status != projection.TurnEnded || strings.TrimSpace(turn.ID) == "" {
			continue
		}
		if !turnsWithTools[turn.Index] {
			continue
		}
		key := changeKey(v.SessionID, turn.ID)
		w.changesMu.Lock()
		asked := w.changesAsked[key]
		w.changesAsked[key] = true
		w.changesMu.Unlock()
		if asked {
			continue
		}
		go w.refreshRunChanges(v.SessionID, turn.ID)
	}
}

// refreshRunChanges 读一轮的声明级改动（D7 的只读事实），存成改动行。
//
// 没有工作区、没有声明或读失败都不出改动块：读失败不编文件清单，同一轮只问一次。
// 一次读取要扫一遍会话历史，所以同一时刻只放 changeReadSlots 个在跑。
func (w *App) refreshRunChanges(sessionID, runID string) {
	w.changeSlots <- struct{}{}
	defer func() { <-w.changeSlots }()
	ctx, cancel := context.WithTimeout(context.Background(), runChangesTimeout)
	defer cancel()
	key := changeKey(sessionID, runID)
	changes, err := w.A.RunChanges(ctx, sessionID, runID)
	w.changesMu.Lock()
	if err != nil {
		delete(w.changesByRun, key)
	} else {
		w.changesByRun[key] = changeRows(changes)
	}
	w.changesMu.Unlock()
	w.changed()
}

// changeRows 把一次改动读取投成改动文件行。差异行数在事实里没有（工具结果只有
// 字节数与替换次数，display diff 不出现在投影里），所以两个计数留 0。
func changeRows(changes workspace.RunChanges) []ui.RowSnapshot {
	out := make([]ui.RowSnapshot, 0, len(changes.Paths))
	for _, change := range changes.Paths {
		row := ui.RowSnapshot{Kind: ui.RowFileChange, Text: change.Path}
		if !change.Exists {
			row.Args = "已删除"
		}
		out = append(out, row)
	}
	return out
}

// toolAction 把工具名换成工具行上的动作词（与界面既有的动作词汇一致）；
// 认不出的工具返回空串，由调用方退回工具原名。
func toolAction(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	// MCP 工具名形如 `mcp__fs__read_file`：动作看最后一段。
	if index := strings.LastIndex(normalized, "__"); index >= 0 {
		normalized = normalized[index+2:]
	}
	switch normalized {
	case "read", "read_file", "readfile", "file_read", "view", "cat":
		return "读取"
	case "write", "write_file", "writefile", "create", "create_file":
		return "写入"
	case "edit", "edit_file", "editfile", "str_replace", "apply_patch", "multi_edit", "patch", "update_file":
		return "改动"
	case "exec", "exec_command", "bash", "run", "shell", "run_shell", "terminal", "command":
		return "执行"
	case "grep", "glob", "search", "find", "code_search", "semantic_search":
		return "搜索"
	case "fetch", "web_fetch", "webfetch", "web_search", "browse", "http_get":
		return "抓取"
	}
	return ""
}

// toolArgs 取工具输入里的主参数：命令取原文，文件取路径，检索取查询词，
// 其余取紧凑 JSON。只读参数、不做解析判断。
func toolArgs(input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}
	var fields map[string]any
	if err := json.Unmarshal(input, &fields); err != nil {
		return compactJSON(input)
	}
	for _, keys := range [][]string{
		{"command", "cmd", "cmdline", "script", "shell_command"},
		{"path", "file_path", "filepath", "file", "filename", "target_file"},
		{"pattern", "query", "q", "regex"},
		{"url", "href"},
		{"prompt", "text", "description"},
	} {
		for _, key := range keys {
			if value, ok := fields[key]; ok {
				if text := argText(value); text != "" {
					return text
				}
			}
		}
	}
	return compactJSON(input)
}

// argText 取一个参数值的文本形式；结构化值与空值不算主参数。
func argText(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	}
	return ""
}

// compactJSON 把没有主参数的输入写成一行紧凑 JSON。
func compactJSON(input json.RawMessage) string {
	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		return strings.TrimSpace(string(input))
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return strings.TrimSpace(string(input))
	}
	return strings.TrimSpace(string(encoded))
}
