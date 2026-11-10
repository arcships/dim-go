package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/workspace"
)

// ErrWorkspaceUnavailable 表示没有配置工作区根，变更读取无处规范化路径。
var ErrWorkspaceUnavailable = errors.New("未配置工作区根")

// 变更读取的扫描上限：一次读取最多多少页、每页多少条事实。超出即报错，
// 不返回一份看起来完整、实际缺项的清单。
const (
	changeHistoryPageLimit  = 500
	changeHistoryPageBudget = 40
)

// changeSource 是工作区变更读取的来源适配器：从会话历史里的工具结果事实中
// 提取路径声明，按轮次（run）归属过滤。
//
// 它只做映射：不扫描工作区、不推断 diff、不读 git 历史、不执行工具。折叠后的
// 工具行只保留文本结果，因此这里读同一批工具结果事实的原始载荷。
type changeSource struct {
	app *App
}

// ToolResultDeclarations 实现 workspace.ChangeSource。
//
// runID 是一轮的标识（`turn_id`）：只取该轮工具结果里的声明，别轮的事实一律不看。
// 没有可解析的声明时返回空切片，不把"没有声明"当成"没有改动"。
func (s *changeSource) ToolResultDeclarations(
	ctx context.Context,
	sessionID string,
	runID string,
) ([]workspace.ToolResultDeclaration, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(runID) == "" {
		return nil, nil
	}
	runtime, ok := s.app.session(sessionID)
	if !ok {
		return nil, fmt.Errorf("%w: 变更读取需要先打开会话 %s", ErrSessionNotOpen, sessionID)
	}
	events, err := s.collectFacts(ctx, runtime)
	if err != nil {
		return nil, err
	}

	// 第一遍：记住 call_id → 工具名，用来判断 write/edit 的 path 声明。
	names := make(map[string]string, len(events))
	for _, event := range events {
		if event.Type != "step" {
			continue
		}
		data := decodeEventData(event.Data)
		switch event.Name {
		case "tool-use":
			call := objectValue(data["call"])
			if id := firstString(call, "id", "call_id", "callId"); id != "" {
				names[id] = firstString(call, "name")
			}
		case "tool-call":
			for _, value := range arrayValue(data["calls"]) {
				call := objectValue(value)
				if id := firstString(call, "id", "call_id", "callId"); id != "" {
					names[id] = firstString(call, "name")
				}
			}
		}
	}

	// 第二遍：只取本轮工具结果里的声明。
	var declarations []workspace.ToolResultDeclaration
	for _, event := range events {
		if event.Type != "step" {
			continue
		}
		data := decodeEventData(event.Data)
		if firstString(data, "turn_id", "turnId") != runID {
			continue
		}
		switch event.Name {
		case "tool-completed":
			callID := firstString(data, "call_id", "callId")
			if callID == "" {
				continue
			}
			if declaration, ok := buildDeclaration(callID, names[callID], data["result"]); ok {
				declarations = append(declarations, declaration)
			}
		case "tool-result":
			for _, value := range arrayValue(data["results"]) {
				result := objectValue(value)
				callID := firstString(result, "call_id", "callId")
				if callID == "" {
					continue
				}
				if declaration, ok := buildDeclaration(callID, names[callID], result["result"]); ok {
					declarations = append(declarations, declaration)
				}
			}
		}
	}
	return declarations, nil
}

// collectFacts 按页读该会话的历史事实。只读，不折叠、不缓存。
func (s *changeSource) collectFacts(ctx context.Context, runtime *sessionRuntime) ([]protocol.RawEvent, error) {
	branchID := runtime.folder.Selected()
	if branchID == "" {
		branchID = mainBranchID
	}
	after := int64(0)
	limit := changeHistoryPageLimit
	var events []protocol.RawEvent
	for page := 0; page < changeHistoryPageBudget; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result, err := runtime.lease.Subscription().History(ctx, protocol.SessionHistoryParams{
			BranchID: branchID,
			After:    &after,
			Limit:    &limit,
		})
		if err != nil {
			return nil, fmt.Errorf("读取会话历史: %w", err)
		}
		events = append(events, result.Events...)
		if result.Done {
			return events, nil
		}
		if result.NextAfter <= after {
			return nil, fmt.Errorf("读取会话历史: 游标不前进（after=%d）", after)
		}
		after = result.NextAfter
	}
	return nil, fmt.Errorf("读取会话历史: 超过 %d 页，结果不完整", changeHistoryPageBudget)
}

// buildDeclaration 从一个工具结果载荷里取出声明路径。
func buildDeclaration(callID, toolName string, rawResult any) (workspace.ToolResultDeclaration, bool) {
	payload := decodeEventData(rawValue(rawResult))
	if len(payload) == 0 {
		return workspace.ToolResultDeclaration{}, false
	}
	paths := declarationPaths(payload, toolName)
	if len(paths) == 0 {
		return workspace.ToolResultDeclaration{}, false
	}
	return workspace.ToolResultDeclaration{ToolCallID: callID, ToolName: toolName, Paths: paths}, true
}

// declarationPaths 按约定取声明路径：
//
//   - `structuredContent.affectedPaths`（也接受 snake_case）；
//   - `write`/`edit` 工具的 `structuredContent.path`；
//   - `fileWritten` effect（结构化内容里的 effects，或载荷顶层的 effects/fileWritten）。
//
// 只认这些字段；其余结构不猜。
func declarationPaths(payload map[string]any, toolName string) []string {
	var paths []string
	structured := objectValue(payload["structuredContent"])
	if structured == nil {
		structured = objectValue(payload["structured_content"])
	}
	if structured != nil {
		paths = append(paths, stringList(structured["affectedPaths"])...)
		paths = append(paths, stringList(structured["affected_paths"])...)
		if isWriteTool(toolName) {
			paths = append(paths, stringList(structured["path"])...)
		}
		paths = append(paths, effectPaths(structured["effects"])...)
	}
	paths = append(paths, effectPaths(payload["effects"])...)
	paths = append(paths, effectPaths(payload["fileWritten"])...)
	return paths
}

// effectPaths 从 effect 清单里只取 fileWritten 的路径。
func effectPaths(raw any) []string {
	var paths []string
	for _, value := range arrayValue(raw) {
		effect := objectValue(value)
		if effect == nil {
			continue
		}
		if !isFileWrittenEffect(firstString(effect, "kind", "type", "name")) {
			continue
		}
		paths = append(paths, effectPathFields(effect)...)
	}
	// 顶层 fileWritten 也可能是单个对象。
	if effect := objectValue(raw); effect != nil {
		paths = append(paths, effectPathFields(effect)...)
	}
	return paths
}

// effectPathFields 取 effect 里的路径字段：字符串、数组或 filePath 别名都接受。
func effectPathFields(effect map[string]any) []string {
	for _, key := range []string{"path", "filePath", "file_path", "file", "paths"} {
		if value, ok := effect[key]; ok {
			if paths := stringList(value); len(paths) > 0 {
				return paths
			}
		}
	}
	return nil
}

// isFileWrittenEffect 判断 effect 类型是否文件写入；大小写与分隔符不敏感。
func isFileWrittenEffect(kind string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(kind))
	return normalized == "filewritten"
}

// isWriteTool 判断工具名是否 write/edit。
func isWriteTool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "write", "edit", "write_file", "edit_file":
		return true
	default:
		return false
	}
}

// stringList 把字符串或字符串数组统一成字符串列表；其它形状忽略。
func stringList(value any) []string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	case []any:
		paths := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				paths = append(paths, text)
			}
		}
		return paths
	case json.RawMessage:
		var text string
		if err := json.Unmarshal(typed, &text); err == nil {
			return stringList(text)
		}
		var list []string
		if err := json.Unmarshal(typed, &list); err == nil {
			return stringList(toAny(list))
		}
	}
	return nil
}

func toAny(values []string) []any {
	items := make([]any, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return items
}

// RunChanges 返回指定会话某一轮里工具声明过的改动路径。
//
// 结果是工具声明级的部分覆盖（`coverage=tool_reports`）：只含工具结果明确声明的
// 路径，不覆盖会话内命令（exec）、外部修改或没有声明的写入；调用方据此展示，
// 不应把它当成权威的 run 级改动清单。
func (a *App) RunChanges(ctx context.Context, sessionID, runID string) (workspace.RunChanges, error) {
	ws := a.Workspace()
	if ws == nil {
		return workspace.RunChanges{}, ErrWorkspaceUnavailable
	}
	if strings.TrimSpace(sessionID) == "" {
		return workspace.RunChanges{}, fmt.Errorf("%w: 缺少会话标识", ErrInvalidConfig)
	}
	return ws.ReadRunChanges(ctx, a.changes, sessionID, runID)
}
