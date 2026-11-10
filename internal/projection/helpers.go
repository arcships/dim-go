package projection

import (
	"encoding/json"
	"regexp"
	"strings"
)

// object 把任意 JSON 值当对象读。解不出就当空对象。
func object(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

// decodeObject 解一条事实的 data。解不出就当空对象：事实本身仍按序号去重，
// 不会因为 data 形状陌生而影响顺序与身份。
func decodeObject(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

// array 把任意 JSON 值当数组读。
func array(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

// stringValue 把任意 JSON 值当字符串读。
func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

// boolValue 把任意 JSON 值当布尔读。
func boolValue(value any) bool {
	flag, _ := value.(bool)
	return flag
}

// firstString 按候选键取第一个非空字符串。只认文档化的 wire 键，不做猜测别名。
func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(data[key]); value != "" {
			return value
		}
	}
	return ""
}

// toInt64 把任意 JSON 数值读成 int64。
func toInt64(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case json.Number:
		number, err := typed.Int64()
		if err != nil {
			return 0
		}
		return number
	default:
		return 0
	}
}

// firstInt64 按候选键取第一个非零整数。
func firstInt64(data map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if number := toInt64(value); number != 0 {
				return number
			}
		}
	}
	return 0
}

// marshalJSON 把值编码成原始 JSON；编码失败返回 nil，由调用方按空处理。
func marshalJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return encoded
}

// cloneRaw 复制一段原始 JSON，避免视图与折叠状态共享同一份字节。
func cloneRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

// cloneObject 复制一层对象，用于放进视图的只读快照。
func cloneObject(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// assistantKey 是助手消息的键：同一轮同一请求只有一条助手消息。
// 缺任一标识时拒绝创建匿名消息。
func assistantKey(data map[string]any) (string, bool) {
	turn := firstString(data, "turn_id", "turnId")
	request := firstString(data, "request_id", "requestId")
	if turn == "" || request == "" {
		return "", false
	}
	return turn + "\x00" + request, true
}

// shellInputPattern 匹配服务端记录的会话内命令输入标记：一条 `!` 命令存成
// 一条带该标记的 raw 块，可见内容是记录的正文。
var shellInputPattern = regexp.MustCompile(`(?s)^<dim-shell-input\b[^>]*>(.*)</dim-shell-input>\s*$`)

// inputText 抽取一条输入的可见文本。优先级与服务端一致：用户自己的文本块、
// 普通文本块、最后是会话内命令的 raw 块。其他 raw 块（环境状态、目标块等）
// 属于模型上下文，不进时间线。
func inputText(data map[string]any) string {
	if user := object(data["user"]); user != nil {
		if text := textBlocks(user["content"]); text != "" {
			return text
		}
	}
	blocks := array(data["blocks"])
	if text := textBlocks(blocks); text != "" {
		return text
	}
	bodies := make([]string, 0, len(blocks))
	for _, value := range blocks {
		block := object(value)
		if stringValue(block["type"]) != "raw" {
			continue
		}
		if body := shellInputBody(stringValue(block["text"])); body != "" {
			bodies = append(bodies, body)
		}
	}
	return strings.Join(bodies, "\n")
}

// textBlocks 拼接 type 为 text 的内容块；其他块不贡献文本。
func textBlocks(parts any) string {
	var builder strings.Builder
	for _, value := range array(parts) {
		block := object(value)
		if stringValue(block["type"]) == "text" {
			builder.WriteString(stringValue(block["text"]))
		}
	}
	return builder.String()
}

// shellInputBody 解出会话内命令的正文。实体只解一层，顺序与服务端一致。
func shellInputBody(text string) string {
	match := shellInputPattern.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	body := match[1]
	body = strings.ReplaceAll(body, "&lt;", "<")
	body = strings.ReplaceAll(body, "&gt;", ">")
	body = strings.ReplaceAll(body, "&quot;", "\"")
	body = strings.ReplaceAll(body, "&#39;", "'")
	body = strings.ReplaceAll(body, "&amp;", "&")
	return strings.TrimSpace(body)
}

// outcomeStatus 把服务端的结果状态映射成行状态。
func outcomeStatus(value string) RowStatus {
	switch value {
	case "failed", "error":
		return StatusError
	case "cancelled", "canceled":
		return StatusCancelled
	default:
		return StatusComplete
	}
}
