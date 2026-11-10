package projection

import (
	"regexp"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// 目标记录用输入块承载：key 说明这条块是目标记录，正文是带属性的标记。
const (
	goalStateKey     = "goal:state"
	goalObjectiveKey = "goal:objective"
	goalStatusKey    = "goal:status"
)

// goalAttrPattern 匹配目标记录的属性。
var goalAttrPattern = regexp.MustCompile(`(\w+)="([^"]*)"`)

// goalBlock 是一条目标输入块。
type goalBlock struct {
	inputID  string
	sequence int64
	at       int64
	key      string
	text     string
	retire   bool
	attrs    map[string]string
}

// goalFold 折叠目标。目标没有独立查询：本分支最新的目标记录就是目标；
// 记录被删除后按剩下的事实重算。
type goalFold struct {
	blocks  []goalBlock
	deleted map[string]struct{}
}

func newGoalFold() *goalFold {
	return &goalFold{deleted: make(map[string]struct{})}
}

// record 收下一条输入里的目标块，并给出这条输入造成的目标变更摘要。
// 没有目标块时返回 nil。
func (g *goalFold) record(event protocol.RawEvent, data map[string]any) *GoalChange {
	var change *GoalChange
	for _, value := range array(data["blocks"]) {
		block := object(value)
		key := stringValue(block["key"])
		if key != goalStateKey && key != goalObjectiveKey && key != goalStatusKey {
			continue
		}
		text := stringValue(block["text"])
		attrs, ok := parseGoalRecord(text)
		if !ok {
			continue
		}
		g.blocks = append(g.blocks, goalBlock{
			inputID:  event.ID,
			sequence: event.Sequence,
			at:       event.CreateAt,
			key:      key,
			text:     text,
			retire:   boolValue(block["retire"]),
			attrs:    attrs,
		})
		change = goalChangeOf(key, attrs, boolValue(block["retire"]))
	}
	return change
}

// deleteInput 记下被删除的输入；删除后重算时它的目标块不再参与。
func (g *goalFold) deleteInput(inputID string) {
	g.deleted[inputID] = struct{}{}
}

// snapshot 从剩下的事实重算目标。最新的一条记录决定整个目标；记录被退休或
// 清空、正文里没有目标、或者承载它的输入被删除，都表示本分支没有目标。
func (g *goalFold) snapshot() *Goal {
	var latest *goalBlock
	revision := 0
	for index := range g.blocks {
		block := &g.blocks[index]
		if _, removed := g.deleted[block.inputID]; removed {
			continue
		}
		if block.key != goalStateKey {
			continue
		}
		latest = block
		revision++
	}
	if latest == nil {
		return nil
	}
	status := goalStateStatus(latest.attrs)
	if latest.retire || status == "" {
		return nil
	}
	objective, ok := goalTag(latest.text, "objective")
	if !ok || objective == "" {
		return nil
	}
	goal := &Goal{
		Objective: objective,
		Status:    status,
		Judge:     "model",
		Revision:  revision,
		UpdatedAt: latest.at,
		InputID:   latest.inputID,
	}
	if latest.attrs["judge"] == "evaluator" {
		goal.Judge = "evaluator"
	}
	if status == GoalPaused {
		goal.Reason = latest.attrs["reason"]
		if goal.Reason == "" {
			goal.Reason = "user"
		}
	}
	if status == GoalBlocked {
		if blocker, ok := goalTag(latest.text, "blocker"); ok {
			goal.Blocker = blocker
		}
	}
	if status == GoalComplete {
		if result, ok := goalTag(latest.text, "result"); ok {
			goal.Result = result
		}
	}
	return goal
}

// goalStateStatus 把记录属性里的状态映射成目标状态。空串表示记录声明"没有目标"。
func goalStateStatus(attrs map[string]string) GoalStatus {
	switch attrs["status"] {
	case "active":
		return GoalActive
	case "paused":
		return GoalPaused
	case "completed", "complete":
		return GoalComplete
	case "blocked":
		return GoalBlocked
	case "none":
		return GoalNone
	case "cleared":
		return ""
	default:
		return GoalNone
	}
}

// goalChangeOf 把一条目标块概括成变更。旧键按它声明的操作归类。
func goalChangeOf(key string, attrs map[string]string, retire bool) *GoalChange {
	if retire {
		return &GoalChange{Action: "cleared", Status: GoalNone}
	}
	status := goalStateStatus(attrs)
	if key == goalObjectiveKey {
		return &GoalChange{Action: "set", Status: status}
	}
	if key == goalStatusKey {
		return &GoalChange{Action: "status", Status: status}
	}
	switch attrs["op"] {
	case "set":
		return &GoalChange{Action: "set", Status: status}
	case "pause":
		return &GoalChange{Action: "pause", Status: GoalPaused}
	case "resume":
		return &GoalChange{Action: "resume", Status: GoalActive}
	case "completed":
		return &GoalChange{Action: "completed", Status: GoalComplete}
	case "blocked":
		return &GoalChange{Action: "blocked", Status: GoalBlocked}
	case "clear":
		return &GoalChange{Action: "cleared", Status: GoalNone}
	}
	if status == "" {
		return &GoalChange{Action: "cleared", Status: GoalNone}
	}
	return &GoalChange{Action: "status", Status: status}
}

// parseGoalRecord 读目标记录的属性。正文以标记开头才算一条记录。
func parseGoalRecord(text string) (map[string]string, bool) {
	if !strings.HasPrefix(text, "<dim-goal ") {
		return nil, false
	}
	end := strings.Index(text, ">")
	if end < 0 {
		return nil, false
	}
	attrs := make(map[string]string)
	for _, match := range goalAttrPattern.FindAllStringSubmatch(text[:end], -1) {
		attrs[match[1]] = goalUnescape(match[2])
	}
	return attrs, true
}

// goalTag 取记录里某个自由文本字段的内容。字段独占一段，内容逐个字符原样读回。
func goalTag(text, tag string) (string, bool) {
	open := "<" + tag + ">\n"
	start := strings.Index(text, open)
	if start < 0 {
		return "", false
	}
	contentStart := start + len(open)
	close := "\n</" + tag + ">"
	end := strings.Index(text[contentStart:], close)
	if end < 0 {
		return "", false
	}
	return goalUnescape(text[contentStart : contentStart+end]), true
}

// goalUnescape 解一层实体，顺序与服务端一致。
func goalUnescape(value string) string {
	value = strings.ReplaceAll(value, "&quot;", "\"")
	value = strings.ReplaceAll(value, "&gt;", ">")
	value = strings.ReplaceAll(value, "&lt;", "<")
	value = strings.ReplaceAll(value, "&amp;", "&")
	return value
}
