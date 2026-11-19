package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是审批卡的输入区：按工具名与入参挑一个渲染器，只画要执行的东西。
//
// 结构照参考产品的 approval-panels/*（CommandApprovalPanel 等）：一个带描边的
// 区块，最上面是一行 11px 次要色标签，下面是等宽正文；内容超过 240 高时区块内部
// 滚动，卡片本身不再长。文案（「输入」、`$` 前缀、`cwd:`、`pattern:` 这类键名）
// 逐字取参考产品的 zh-CN 与源码字面量；工具入参里的值原样显示，不翻译、不改写。
//
// 渲染器的挑法与参考产品的工具轮询表一致（先具体后通用）；这里是纯函数，界面只
// 负责把行画出来，测试可以直接断言行内容。

// 渲染器 ID。它们只用于诊断与测试，不上屏。
const (
	// approvalRendererCommand 画命令：`$ {command}` + cwd / timeoutMs /
	// runInBackground 参数行。
	approvalRendererCommand = "command"
	// approvalRendererFile 画文件：标签是路径，编辑是双栏差异，写入是带行号的
	// 代码块，补丁是无行号的代码块。
	approvalRendererFile = "file"
	// approvalRendererSearch 画检索：模式 / 查询 / 路径主值行 + 其余参数。
	approvalRendererSearch = "search"
	// approvalRendererWeb 画网页：搜索是 query + 分页，抓取是 URL 标签 + 附加参数。
	approvalRendererWeb = "web"
	// approvalRendererSkill 画技能：`skill_name: {名称}`。
	approvalRendererSkill = "skill"
	// approvalRendererAgent 画子代理：创建类的头行 + prompt + effort / forkContext；
	// 其余动作是行列表。
	approvalRendererAgent = "agent"
	// approvalRendererBrowser 画浏览器工具：行列表。
	approvalRendererBrowser = "browser"
	// approvalRendererCua 画电脑操作工具：`tool: {tool}` + 全部入参行。
	approvalRendererCua = "cua"
	// approvalRendererWait 画等待：taskId 与 timeoutMs 两行。
	approvalRendererWait = "wait"
	// approvalRendererMCP 画 MCP 工具：服务器、工具与参数三组行。
	approvalRendererMCP = "mcp"
	// approvalRendererImage 画图片：缩略图 + 尺寸。
	approvalRendererImage = "image"
	// approvalRendererTodos 画待办：清单。
	approvalRendererTodos = "todos"
	// approvalRendererGeneric 是兜底：工具名做标签，入参原文做代码块。
	approvalRendererGeneric = "generic"
)

// 工具类别，与参考产品 classifyTimelineToolCategory 同名同义。
const (
	approvalCategoryRead        = "read"
	approvalCategoryEdit        = "edit"
	approvalCategoryCommand     = "command"
	approvalCategorySearch      = "search"
	approvalCategoryWebSearch   = "web_search"
	approvalCategoryWebFetch    = "web_fetch"
	approvalCategorySubagent    = "subagent"
	approvalCategoryBrowser     = "browser"
	approvalCategoryComputerUse = "computer_use"
	approvalCategoryWait        = "wait"
	approvalCategorySkill       = "skill"
	approvalCategoryTodos       = "todos"
	approvalCategoryImage       = "image"
	approvalCategoryUnknown     = "unknown"
)

const (
	// approvalInputLabel 是命令 / 检索 / 网页搜索 / 行列表 / 通用渲染器的顶部标签
	// （参考产品 i18n `timelineTool.input`）。
	approvalInputLabel = "输入"
	// approvalInputMaxHeight 是输入区块的最高高度（参考产品 max-h-[240px]）。
	approvalInputMaxHeight = 240
	// approvalCodeMaxHeight 是代码块的最高高度：比输入区块再矮一档，给标签留位。
	approvalCodeMaxHeight = 180
	// approvalTodoMaxLines 是待办清单一次最多画几行（参考产品 DETAIL_LINE_MAX 同量级）。
	approvalTodoMaxLines = 12
	// approvalImageThumb 是图片缩略图的边长。
	approvalImageThumb = 64
	// approvalLineNumberWidth 是代码块行号列的宽度：两位数行号放得下。
	approvalLineNumberWidth = 21
	// approvalFaviconSize 是网页抓取标签里站点徽记的直径（参考 FaviconImg 的 16）。
	approvalFaviconSize = 16
	// approvalPatchLabel 是补丁请求没有路径时的标签（参考产品字面量 `patch.diff`）。
	approvalPatchLabel = "patch.diff"
)

// 参数行的键名（参考产品源码字面量，顺序也是源码里的顺序）。
var (
	approvalCommandKeys = []string{"cwd", "timeoutMs", "runInBackground"}
	approvalSearchKeys  = []string{
		"include", "glob", "outputMode", "output_mode",
		"beforeContext", "afterContext", "context", "-B", "-A", "-C",
		"lineNumbers", "-n", "literal", "limit",
	}
	approvalWebSearchKeys = []string{"page", "pageSize"}
	approvalWebFetchKeys  = []string{"startIndex", "contextMaxCharacters", "mode"}
	approvalAgentCreate   = []string{"effort", "forkContext"}
	approvalAgentOther    = []string{
		"action", "taskId", "agentName", "taskTitle", "agentType",
		"providerId", "modelId", "model", "effort", "reason",
	}
	approvalBrowserKeys = []string{
		"action", "url", "ref", "value", "key", "direction", "amount",
		"type", "timeout", "expression", "method", "params",
		"selector", "interactiveOnly", "compact", "fullPage", "clear",
	}
	approvalFileEditKeys = []string{"replaceAll"}
	approvalFileReadKeys = []string{"offset", "limit", "encoding"}
)

// approvalPanelContent 是一次输入区要画的内容。纯数据：界面按它画，测试可以直接
// 断言行与文案。
type approvalPanelContent struct {
	// Renderer 是命中的渲染器 ID。
	Renderer string
	// Label 是顶部标签：多为「输入」，文件类是路径，网页抓取是 URL。
	Label string
	// FaviconHost 非空时，标签行前画一个站点徽记（网页抓取）。
	FaviconHost string
	// Primary 是主值行（命令、pattern/query/path）：比参数行亮一档。
	Primary string
	// Prefix 是主值行的等宽前缀（命令行的 `$`）。
	Prefix string
	// Lines 是参数行 / 行列表，逐行 `键: 值`。
	Lines []string
	// Head 是子代理创建类的头行四段（图标 + 名称 + 任务标题 + 模型 id）；
	// 全空时不画头行。
	Head approvalHead
	// Prompt 是子代理 prompt 全文；空表示没有。
	Prompt string
	// Code 是非主值的代码块（写入内容、补丁、JSON 入参）。
	Code string
	// CodeSet 表示代码块这个事实在场：内容为空（新建空文件）时也要画一个空块。
	CodeSet bool
	// CodeLines 为真时给代码块加行号列（写入文件是行号，补丁不是）。
	CodeLines bool
	// DiffOld / DiffNew 是编辑前后两栏；Set 为假表示这一侧没有事实（不留空栏）。
	DiffOld    string
	DiffNew    string
	DiffOldSet bool
	DiffNewSet bool
	// Image 是能画出来的缩略图（入参里的 data URI）；空表示没有像素，只画占位。
	HasImage bool
	// ImageData 是解码用的字节（PNG / JPEG / SVG）。
	ImageData []byte
	// SizeText 是图片尺寸（`宽 × 高`）；空表示入参没有尺寸。
	SizeText string
	// Todos 是待办清单的行（含状态标记）。
	Todos []string
}

// approvalHead 是子代理创建类的头行。
type approvalHead struct {
	Name  string
	Task  string
	Model string
}

// empty 判断这次输入区有没有可画的内容。
func (p approvalPanelContent) empty() bool {
	if p.Primary != "" || len(p.Lines) > 0 || p.CodeSet || p.Prompt != "" {
		return false
	}
	if p.DiffOldSet || p.DiffNewSet || len(p.Todos) > 0 || p.HasImage || p.SizeText != "" {
		return false
	}
	return p.Head == approvalHead{}
}

// approvalPanelFor 把工具名与入参换成输入区内容。工具名为空但入参里有命令时按
// 命令渲染器画（样例态与只有命令原文的场景）。
func approvalPanelFor(tool string, input map[string]any) approvalPanelContent {
	renderer := approvalRendererFor(tool, input)
	switch renderer {
	case approvalRendererCommand:
		return approvalCommandPanel(input)
	case approvalRendererFile:
		return approvalFilePanel(tool, input)
	case approvalRendererSearch:
		return approvalSearchPanel(input)
	case approvalRendererWeb:
		return approvalWebPanel(tool, input)
	case approvalRendererSkill:
		// G-C4-17 skill_name 行
		return approvalInputLinesPanel(approvalRendererSkill, input, []string{"skill_name"})
	case approvalRendererAgent:
		return approvalAgentPanel(input)
	case approvalRendererBrowser:
		// G-C4-22 浏览器工具的工具名与全部入参
		return approvalInputLinesPanel(approvalRendererBrowser, input, approvalBrowserKeys)
	case approvalRendererCua:
		return approvalCuaPanel(tool, input)
	case approvalRendererWait:
		return approvalWaitPanel(input)
	case approvalRendererMCP:
		return approvalMCPPanel(tool, input)
	case approvalRendererImage:
		return approvalImagePanel(tool, input)
	case approvalRendererTodos:
		return approvalTodosPanel(input)
	default:
		return approvalGenericPanel(tool, input)
	}
}

// approvalRendererFor 挑渲染器：先具体后通用，与参考产品的注册顺序同一口径。
func approvalRendererFor(tool string, input map[string]any) string {
	raw := strings.ToLower(strings.TrimSpace(tool))
	if raw == "" {
		return approvalRendererCommand
	}
	// 旧版纯文本入参（{input: 文本}）一律走通用渲染器：没有字段可挑。
	if len(input) == 1 {
		if _, ok := input["input"]; ok {
			return approvalRendererGeneric
		}
	}
	switch {
	case strings.HasPrefix(raw, "cua-driver__") || raw == "cua-driver":
		return approvalRendererCua
	case isTodoTool(raw):
		return approvalRendererTodos
	case isImageTool(raw):
		return approvalRendererImage
	}
	switch approvalCategory(raw) {
	case approvalCategorySubagent:
		return approvalRendererAgent
	case approvalCategoryWait:
		return approvalRendererWait
	case approvalCategoryRead, approvalCategoryEdit:
		return approvalRendererFile
	case approvalCategoryCommand:
		return approvalRendererCommand
	case approvalCategorySearch:
		return approvalRendererSearch
	case approvalCategoryWebSearch, approvalCategoryWebFetch:
		return approvalRendererWeb
	case approvalCategorySkill:
		return approvalRendererSkill
	case approvalCategoryBrowser:
		return approvalRendererBrowser
	}
	// MCP 工具的 wire 名是 `server__tool`：拆得开就按 MCP 画，拆不开按通用画。
	if server, _, ok := splitMCPTool(raw); ok && server != "" {
		return approvalRendererMCP
	}
	return approvalRendererGeneric
}

// approvalCategory 是参考产品 classifyTimelineToolCategory 的同一张表。
func approvalCategory(tool string) string {
	raw := strings.ToLower(strings.TrimSpace(tool))
	name := normalizeToolName(tool)
	switch {
	case strings.Contains(raw, "web_search") || strings.Contains(raw, "search_query") || name == "websearch":
		return approvalCategoryWebSearch
	case strings.Contains(raw, "web_fetch") || name == "webfetch":
		return approvalCategoryWebFetch
	}
	switch name {
	case "read", "read_file", "open", "cat", "view":
		return approvalCategoryRead
	case "write", "write_file", "edit", "edit_file", "apply_patch", "replace", "update", "delete":
		return approvalCategoryEdit
	case "grep", "rg", "glob", "find", "list_files", "ls":
		return approvalCategorySearch
	case "bash", "shell", "terminal", "exec", "run_command", "exec_command":
		return approvalCategoryCommand
	case "agent", "subagent":
		return approvalCategorySubagent
	case "browser":
		return approvalCategoryBrowser
	case "wait":
		return approvalCategoryWait
	case "skill":
		return approvalCategorySkill
	}
	switch {
	case strings.Contains(raw, "search_files"):
		return approvalCategorySearch
	case strings.Contains(raw, "subagent"):
		return approvalCategorySubagent
	case strings.Contains(raw, "todo"):
		return approvalCategoryTodos
	case strings.Contains(raw, "image"):
		return approvalCategoryImage
	}
	return approvalCategoryUnknown
}

// normalizeToolName 去掉下划线、连字符与空格并转小写（参考产品同名函数）。
func normalizeToolName(tool string) string {
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(tool)))
}

// isTodoTool 判断工具名是否待办清单工具。
func isTodoTool(tool string) bool {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "todowrite", "todo_write", "todos", "todo", "update_todos":
		return true
	}
	return false
}

// isImageTool 判断工具名是否图片类工具（查看图片、生成图片）。
func isImageTool(tool string) bool {
	raw := strings.ToLower(strings.TrimSpace(tool))
	if raw == "" || strings.Contains(raw, "imagery") {
		return false
	}
	return strings.Contains(raw, "image")
}

// splitMCPTool 把 `server__tool` 拆成服务器与工具；没有分隔符时 ok 为假。
func splitMCPTool(tool string) (server, name string, ok bool) {
	index := strings.Index(tool, "__")
	if index <= 0 || index+2 >= len(tool) {
		return "", "", false
	}
	return tool[:index], tool[index+2:], true
}

// approvalInputLines 按固定键序取「键: 值」行；值缺失的键不出行。
func approvalInputLines(input map[string]any, keys []string) []string {
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value, ok := input[key]
		if !ok {
			continue
		}
		lines = append(lines, key+": "+approvalDisplayValue(value))
	}
	return lines
}

// approvalDisplayValue 把入参值写成一行文本：字符串原样，数字与布尔按字面，
// 对象与数组按 JSON（不出现 `[object Object]`）。
func approvalDisplayValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		return trimFloat(typed)
	case json.Number:
		return typed.String()
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(raw)
}

// trimFloat 把 JSON 数字写成不带多余小数点的形式：60 不写成 60.0。
func trimFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// approvalText 取入参里第一个非空字符串字段。
func approvalText(input map[string]any, keys []string) string {
	for _, key := range keys {
		value, ok := input[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// approvalRawText 取入参里第一个存在的字符串字段（允许空串，报告"将写入空内容"）。
func approvalRawText(input map[string]any, keys []string) (string, bool) {
	for _, key := range keys {
		value, ok := input[key].(string)
		if ok {
			return value, true
		}
	}
	return "", false
}

// approvalBoolText 把入参里的布尔字段写成 true / false；不是布尔时为空。
func approvalBoolText(input map[string]any, key string) string {
	value, ok := input[key].(bool)
	if !ok {
		return ""
	}
	if value {
		return "true"
	}
	return "false"
}

// approvalTimeoutText 把 timeoutMs 写成参考产品的单位：整秒写 `60 s`，否则写
// `1500 ms`；不是数字时为空。
func approvalTimeoutText(input map[string]any) string {
	value, ok := input["timeoutMs"].(float64)
	if !ok {
		return ""
	}
	if value >= 0 && value == float64(int64(value)) && int64(value)%1000 == 0 {
		return fmt.Sprintf("%d s", int64(value)/1000)
	}
	return fmt.Sprintf("%s ms", trimFloat(value))
}

// approvalCommandPanel 是命令渲染器。
// G-C4-03 / IN-71 「$ {command}」等宽行
func approvalCommandPanel(input map[string]any) approvalPanelContent {
	panel := approvalPanelContent{
		Renderer: approvalRendererCommand,
		Label:    approvalInputLabel,
		Prefix:   "$",
		Primary:  approvalText(input, []string{"command", "cmd", "commandPreview", "shellCommand", "script"}),
	}
	// G-C4-04 / IN-72 / IN-73 cwd / timeoutMs / runInBackground 参数行
	for _, key := range approvalCommandKeys {
		switch key {
		case "timeoutMs":
			if text := approvalTimeoutText(input); text != "" {
				panel.Lines = append(panel.Lines, key+": "+text)
			}
		case "runInBackground":
			if text := approvalBoolText(input, key); text != "" {
				panel.Lines = append(panel.Lines, key+": "+text)
			}
		default:
			if text, ok := input[key].(string); ok && strings.TrimSpace(text) != "" {
				panel.Lines = append(panel.Lines, key+": "+text)
			}
		}
	}
	return panel
}

// approvalFilePanel 是文件渲染器：标签是路径（补丁没有路径时写 patch.diff，
// 再没有就写工具名），正文是差异 / 写入内容 / 补丁之一，再加参数行。
// G-C4-05 文件名 / patch.diff / 工具名标签
func approvalFilePanel(tool string, input map[string]any) approvalPanelContent {
	name := strings.ToLower(strings.TrimSpace(tool))
	isEdit := name == "edit" || name == "edit_file"
	isWrite := name == "write" || name == "write_file"
	isPatch := strings.Contains(name, "apply_patch") || name == "patch"

	path := approvalText(input, []string{"path", "filePath", "file_path", "file", "target"})
	label := path
	if label == "" {
		label = strings.TrimSpace(tool)
	}
	if label == "" {
		label = approvalInputLabel
	}
	if path == "" && isPatch {
		label = approvalPatchLabel
	}

	panel := approvalPanelContent{Renderer: approvalRendererFile, Label: label}
	oldText, hasOld := approvalRawText(input, []string{"oldText", "old_string"})
	newText, hasNew := approvalRawText(input, []string{"newText", "new_string"})
	content, hasContent := approvalRawText(input, []string{"content"})
	patch, hasPatch := approvalRawText(input, []string{"patch", "input"})

	switch {
	case isEdit && (hasOld || hasNew):
		panel.DiffOld, panel.DiffOldSet = oldText, hasOld
		panel.DiffNew, panel.DiffNewSet = newText, hasNew
	case isWrite && hasContent:
		// G-C4-07 写入内容（代码块 + 行号）
		panel.Code, panel.CodeSet, panel.CodeLines = content, true, true
	case isPatch && hasPatch:
		// G-C4-08 补丁文本
		panel.Code, panel.CodeSet = patch, true
	}

	switch {
	case isEdit:
		// G-C4-09 replaceAll / offset / limit / encoding 参数行
		panel.Lines = approvalInputLines(input, approvalFileEditKeys)
	case !isWrite && !isPatch:
		panel.Lines = approvalInputLines(input, approvalFileReadKeys)
	}
	return panel
}

// approvalSearchPanel 是检索渲染器：主值行按命中键写字面键名，路径另起一行。
func approvalSearchPanel(input map[string]any) approvalPanelContent {
	pattern := approvalText(input, []string{"pattern"})
	path := approvalText(input, []string{"path", "filePath", "file_path"})
	query := approvalText(input, []string{"query", "search_query"})
	isListing := pattern == "*" || (pattern == "" && query == "")

	panel := approvalPanelContent{Renderer: approvalRendererSearch, Label: approvalInputLabel}
	switch {
	case isListing:
		panel.Primary = "path: " + path
	case pattern != "":
		// G-C4-10 pattern: {值}
		panel.Primary = "pattern: " + pattern
		// G-C4-11 path: {值}
		panel.Lines = approvalLinesWith("path", path)
	default:
		panel.Primary = "query: " + query
		panel.Lines = approvalLinesWith("path", path)
	}
	// G-C4-12 include / glob / 上下文行数等检索参数
	panel.Lines = append(panel.Lines, approvalInputLines(input, approvalSearchKeys)...)
	return panel
}

// approvalLinesWith 在值非空时给一行「键: 值」。
func approvalLinesWith(key, value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{key + ": " + value}
}

// approvalWebPanel 是网页渲染器：搜索画查询与分页，抓取画 URL 标签与附加参数。
func approvalWebPanel(tool string, input map[string]any) approvalPanelContent {
	if approvalCategory(tool) == approvalCategoryWebSearch || approvalText(input, []string{"query", "search_query"}) != "" {
		// G-C4-13 query: {值}
		panel := approvalPanelContent{
			Renderer: approvalRendererWeb,
			Label:    approvalInputLabel,
			Primary:  "query: " + approvalText(input, []string{"query", "search_query"}),
		}
		// G-C4-14 page / pageSize
		panel.Lines = approvalInputLines(input, approvalWebSearchKeys)
		return panel
	}
	url := approvalText(input, []string{"url"})
	// G-C4-15 URL 标签与站点徽记
	// G-C4-16 startIndex / contextMaxCharacters / mode
	panel := approvalPanelContent{
		Renderer:    approvalRendererWeb,
		Label:       url,
		FaviconHost: approvalHostOf(url),
		Lines:       approvalInputLines(input, approvalWebFetchKeys),
	}
	if panel.Label == "" {
		panel.Label = approvalInputLabel
	}
	return panel
}

// approvalAgentPanel 是子代理渲染器：创建类画头行 + prompt + 参数，其余画行列表。
func approvalAgentPanel(input map[string]any) approvalPanelContent {
	action := approvalText(input, []string{"action"})
	isCreate := action == "create" || action == "create_external"
	panel := approvalPanelContent{Renderer: approvalRendererAgent, Label: approvalInputLabel}
	if !isCreate {
		// G-C4-21 action / taskId / agentName 等参数行
		panel.Lines = approvalInputLines(input, approvalAgentOther)
		return panel
	}
	panel.Head = approvalHead{
		Name:  approvalText(input, []string{"agentName"}),
		Task:  approvalText(input, []string{"taskTitle"}),
		Model: approvalText(input, []string{"modelId", "model"}),
	}
	// G-C4-19 子代理 prompt
	panel.Prompt = approvalText(input, []string{"prompt"})
	// G-C4-20 effort / forkContext
	panel.Lines = approvalInputLines(input, approvalAgentCreate)
	return panel
}

// approvalCuaPanel 是电脑操作渲染器：首行工具名（去掉 `cua-driver__` 前缀），
// 后接全部入参。
func approvalCuaPanel(tool string, input map[string]any) approvalPanelContent {
	raw := strings.ToLower(strings.TrimSpace(tool))
	name := strings.TrimPrefix(raw, "cua-driver__")
	panel := approvalPanelContent{Renderer: approvalRendererCua, Label: approvalInputLabel}
	// G-C4-23 tool: {tool} 与全部入参行
	panel.Lines = append(panel.Lines, "tool: "+name)
	panel.Lines = append(panel.Lines, approvalRemainingLines(input, map[string]bool{})...)
	return panel
}

// approvalWaitPanel 是等待渲染器：taskId 与 timeoutMs 两行。
func approvalWaitPanel(input map[string]any) approvalPanelContent {
	panel := approvalPanelContent{Renderer: approvalRendererWait, Label: approvalInputLabel}
	// G-C4-24 taskId / timeoutMs
	panel.Lines = approvalInputLines(input, []string{"taskId"})
	if text := approvalTimeoutText(input); text != "" {
		panel.Lines = append(panel.Lines, "timeoutMs: "+text)
	}
	return panel
}

// approvalMCPPanel 是 MCP 渲染器：服务器、工具与其余参数各占一行。原始工具名
// 只出现在这里（拆成两段），标题位仍是「使用工具」。
func approvalMCPPanel(tool string, input map[string]any) approvalPanelContent {
	server, name, ok := splitMCPTool(tool)
	panel := approvalPanelContent{Renderer: approvalRendererMCP, Label: approvalInputLabel}
	if ok {
		// G-C4-23 MCP 的 server / tool 行
		panel.Lines = append(panel.Lines, "server: "+server, "tool: "+name)
	} else {
		panel.Lines = append(panel.Lines, "tool: "+strings.TrimSpace(tool))
	}
	panel.Lines = append(panel.Lines, approvalRemainingLines(input, map[string]bool{})...)
	return panel
}

// approvalImagePanel 是图片渲染器：能解出 data URI 时画缩略图，尺寸按入参的
// 宽高写出；都没有时只留标签，不拿别的图顶替。
func approvalImagePanel(tool string, input map[string]any) approvalPanelContent {
	path := approvalText(input, []string{"path", "filePath", "file_path", "url", "file"})
	panel := approvalPanelContent{Renderer: approvalRendererImage, Label: path}
	if panel.Label == "" {
		panel.Label = approvalText(input, []string{"name"})
	}
	if panel.Label == "" {
		panel.Label = strings.TrimSpace(tool)
	}
	if panel.Label == "" {
		panel.Label = approvalInputLabel
	}
	if data, ok := approvalImageBytes(input); ok {
		panel.HasImage = true
		panel.ImageData = data
	}
	panel.SizeText = approvalImageSize(input)
	return panel
}

// approvalImageBytes 取出入参里的图片字节：只认 data URI（`data:image/...;base64,`），
// 本端不读文件、不联网取图。
func approvalImageBytes(input map[string]any) ([]byte, bool) {
	for _, key := range []string{"data", "dataUrl", "dataURL", "image"} {
		text, ok := input[key].(string)
		if !ok {
			continue
		}
		if data, ok := decodeImageDataURI(text); ok {
			return data, true
		}
	}
	return nil, false
}

// approvalImageSize 是入参里的图片尺寸文本；没有宽高时为空。
func approvalImageSize(input map[string]any) string {
	width, widthOK := input["width"].(float64)
	height, heightOK := input["height"].(float64)
	if !widthOK || !heightOK {
		return ""
	}
	return trimFloat(width) + " x " + trimFloat(height)
}

// approvalTodosPanel 是待办渲染器：逐条 `序号. 标记 内容 - 状态 - 优先级`，
// 条目过多时末尾写省略数。
func approvalTodosPanel(input map[string]any) approvalPanelContent {
	panel := approvalPanelContent{Renderer: approvalRendererTodos, Label: approvalInputLabel}
	items := approvalTodoItems(input)
	for index, item := range items {
		if index >= approvalTodoMaxLines {
			break
		}
		panel.Todos = append(panel.Todos, fmt.Sprintf("%d. %s %s - %s - %s",
			index+1, approvalTodoMark(item.status), item.content,
			approvalTodoStatusText(item.status), item.priority))
	}
	if len(items) > approvalTodoMaxLines {
		panel.Todos = append(panel.Todos, fmt.Sprintf("... omitted %d items", len(items)-approvalTodoMaxLines))
	}
	return panel
}

// approvalTodo 是一条待办（与服务端 todo 入参的同一组字段）。
type approvalTodo struct {
	content  string
	status   string
	priority string
}

// approvalTodoItems 从入参的 todos / items 里读待办；形状不对的条目跳过（不猜）。
func approvalTodoItems(input map[string]any) []approvalTodo {
	raw, ok := input["todos"]
	if !ok {
		raw = input["items"]
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	items := make([]approvalTodo, 0, len(list))
	for _, entry := range list {
		record, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		content, _ := record["content"].(string)
		if strings.TrimSpace(content) == "" {
			continue
		}
		status, _ := record["status"].(string)
		priority, _ := record["priority"].(string)
		items = append(items, approvalTodo{content: content, status: status, priority: priority})
	}
	return items
}

// approvalTodoMark 是待办状态标记（参考产品 todos-renderer 的 [x] / [~] / [ ]）。
func approvalTodoMark(status string) string {
	switch status {
	case "completed":
		return "[x]"
	case "in_progress":
		return "[~]"
	default:
		return "[ ]"
	}
}

// approvalTodoStatusText 是待办状态词；进行中按参考产品写 in progress。
func approvalTodoStatusText(status string) string {
	if status == "in_progress" {
		return "in progress"
	}
	if status == "" {
		return "pending"
	}
	return status
}

// approvalGenericPanel 是兜底渲染器：工具名做等宽标签，入参原文按两空格缩进画。
// G-C4-25 工具名 + 入参原文
func approvalGenericPanel(tool string, input map[string]any) approvalPanelContent {
	label := strings.TrimSpace(tool)
	if label == "" {
		label = approvalInputLabel
	}
	panel := approvalPanelContent{Renderer: approvalRendererGeneric, Label: label}
	if len(input) > 0 {
		if raw, err := json.MarshalIndent(input, "", "  "); err == nil {
			panel.Code, panel.CodeSet = string(raw), true
		}
	}
	return panel
}

// approvalInputLinesPanel 是几类「行列表」渲染器的共用实现：标签「输入」+ 键序行。
func approvalInputLinesPanel(renderer string, input map[string]any, keys []string) approvalPanelContent {
	return approvalPanelContent{
		Renderer: renderer,
		Label:    approvalInputLabel,
		Lines:    approvalInputLines(input, keys),
	}
}

// approvalRemainingLines 按入参键名的字典序列出除 skip 以外的全部字段。MCP 与
// 电脑操作工具的参数集合不固定，只能按到达顺序稳定地排（键名排序后固定）。
func approvalRemainingLines(input map[string]any, skip map[string]bool) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		if skip[key] {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+": "+approvalDisplayValue(input[key]))
	}
	return lines
}

// ---- 画 ----

// approvalInputPanel 画输入区块：描边 + 卡面 + 标签 + 正文，超过最高高度时内部
// 滚动。整块只读，不进 Tab 顺序。
// G-C4-01 审批输入区容器
func approvalInputPanel(c *gui.Context, panel approvalPanelContent) {
	if panel.empty() && panel.Label == "" {
		return
	}
	t := c.Theme()
	gui.Column(c).WidthPercent(100).Shrink(0).MaxHeight(approvalInputMaxHeight).
		Background(t.Surface).Border(strokeHairline, t.Border).Radius(radiusCard).
		Label("审批输入：" + panel.Label).Children(func() {
		gui.Scroll(c).Shrink(0).Children(func() {
			gui.Column(c).Padding(space2, space3).Gap(space1).Children(func() {
				approvalPanelLabel(c, panel)
				approvalPanelBody(c, panel)
			})
		})
	})
}

// approvalPanelLabel 画顶部标签：普通标签是 11px 次要色；网页抓取在 URL 前画
// 站点徽记（参考产品 FaviconImg 的回落：首字母圆徽）。
func approvalPanelLabel(c *gui.Context, panel approvalPanelContent) {
	t := c.Theme()
	// IN-70 面板小标题「输入」（命令类审批面板）
	if panel.Label == "" {
		return
	}
	gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
		if panel.FaviconHost != "" {
			siteBadge(c, panel.FaviconHost, approvalFaviconSize)
		}
		// G-C4-02 输入区标签（命令类是「输入」）
		label := typeMicro.style(t, gui.Text(c, panel.Label)).TextColor(t.TextMuted).SingleLine()
		if panel.Renderer == approvalRendererGeneric {
			// 通用渲染器的标签是工具名，取等宽（参考 GenericApprovalPanel 的字面量）。
			label.Font(codeFamily())
		}
	})
}

// approvalPanelBody 画正文：差异块 / 代码块 / 图片 / 主值行 / 参数行 / 待办。
func approvalPanelBody(c *gui.Context, panel approvalPanelContent) {
	t := c.Theme()
	if panel.DiffOldSet || panel.DiffNewSet {
		approvalDiffBlock(c, panel)
	}
	if panel.Head != (approvalHead{}) {
		approvalHeadRow(c, panel.Head)
	}
	if panel.Prompt != "" {
		approvalCodeBlock(c, panel.Prompt, false)
	}
	if panel.CodeSet {
		approvalCodeBlock(c, panel.Code, panel.CodeLines)
	}
	if panel.HasImage || panel.SizeText != "" {
		approvalImageBlock(c, panel)
	}
	// G-C4-03 主值行（命令带 `$` 前缀）
	if panel.Primary != "" {
		gui.Row(c).Gap(space1).AlignItems(gui.Start).Children(func() {
			if panel.Prefix != "" {
				typeCode.style(t, gui.Text(c, panel.Prefix)).TextColor(t.TextMuted)
			}
			text := panel.Primary
			if panel.Prefix != "" {
				text = strings.TrimPrefix(text, panel.Prefix+" ")
			}
			typeCode.style(t, gui.Text(c, text)).TextColor(t.Text)
		})
	}
	if len(panel.Lines) > 0 {
		gui.Column(c).PaddingStart(space2).Gap(0).Children(func() {
			for _, line := range panel.Lines {
				typeCode.style(t, gui.Text(c, line)).TextColor(t.TextMuted)
			}
		})
	}
	for _, line := range panel.Todos {
		typeCode.style(t, gui.Text(c, line)).TextColor(t.TextMuted)
	}
}

// approvalHeadRow 画子代理创建类的头行：图标 + 名称 + 任务标题 + 模型 id。
// G-C4-18 子代理创建头行
func approvalHeadRow(c *gui.Context, head approvalHead) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
		iconGlyph(c, iconBranch, colorAccentInk.in(t), "agent")
		if head.Name != "" {
			typeSecondary.style(t, gui.Text(c, head.Name)).TextColor(t.Text)
		}
		if head.Task != "" {
			typeSecondary.style(t, gui.Text(c, head.Task)).TextColor(t.TextMuted).Grow(1).SingleLine()
		} else {
			gui.Spacer(c)
		}
		if head.Model != "" {
			typeMicro.style(t, gui.Text(c, head.Model)).TextColor(t.TextMuted)
		}
	})
}

// approvalCodeBlock 画只读代码块：等宽、卡面、最高 180 后内部滚动；可选行号列。
func approvalCodeBlock(c *gui.Context, code string, lineNumbers bool) {
	t := c.Theme()
	gui.Column(c).WidthPercent(100).Shrink(0).MaxHeight(approvalCodeMaxHeight).
		Radius(radiusSmall).Background(t.SurfacePressed).Children(func() {
		gui.Scroll(c).Shrink(0).Children(func() {
			gui.Column(c).Padding(space1, space2).Gap(0).Children(func() {
				for index, line := range strings.Split(code, "\n") {
					gui.Row(c).Gap(space2).AlignItems(gui.Start).Children(func() {
						if lineNumbers {
							typeMicro.style(t, gui.Text(c, fmt.Sprintf("%d", index+1))).
								TextColor(t.TextMuted).Width(approvalLineNumberWidth)
						}
						typeCode.style(t, gui.Text(c, line)).TextColor(t.Text)
					})
				}
			})
		})
	})
}

// approvalDiffBlock 画编辑前后两栏：左旧右新、无行号，旧栏危险色、新栏成功色，
// 两侧等高（按较长一侧的行数补齐空行）；只有一侧有事实时只画那一栏。
// G-C4-06 编辑前后两栏差异
func approvalDiffBlock(c *gui.Context, panel approvalPanelContent) {
	t := c.Theme()
	rows := 0
	if panel.DiffOldSet {
		rows = approvalLineCount(panel.DiffOld)
	}
	if panel.DiffNewSet && approvalLineCount(panel.DiffNew) > rows {
		rows = approvalLineCount(panel.DiffNew)
	}
	gui.Row(c).WidthPercent(100).Gap(space2).AlignItems(gui.Stretch).Children(func() {
		if panel.DiffOldSet {
			approvalDiffColumn(c, panel.DiffOld, rows, t.Danger)
		}
		if panel.DiffNewSet {
			approvalDiffColumn(c, panel.DiffNew, rows, t.Success)
		}
	})
}

// approvalLineCount 是文本的行数：空串也算一行（一个空行）。
func approvalLineCount(text string) int {
	return len(strings.Split(text, "\n"))
}

// approvalDiffColumn 画差异的一栏：行首一条色条 + 等宽文本。
func approvalDiffColumn(c *gui.Context, text string, rows int, color gui.Color) {
	t := c.Theme()
	lines := strings.Split(text, "\n")
	gui.Row(c).Grow(1).Shrink(1).Gap(space1).AlignItems(gui.Stretch).Children(func() {
		bar := gui.Box(c).Width(strokeMarker).Shrink(0).Label("差异标记").AccessHidden(true)
		bar.DrawOver(func(p *gui.Painter, r gui.Rect) { p.Fill(r, color, 0) })
		gui.Column(c).Grow(1).Shrink(1).Gap(0).Children(func() {
			for index := 0; index < rows; index++ {
				line := ""
				if index < len(lines) {
					line = lines[index]
				}
				typeCode.style(t, gui.Text(c, line)).TextColor(t.Text)
			}
		})
	})
}

// approvalImageBlock 画图片块：能解码时画缩略图，否则留一个占位框；尺寸写在
// 缩略图下方（入参没有宽高时不写）。
func approvalImageBlock(c *gui.Context, panel approvalPanelContent) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
		if panel.HasImage {
			if src, err := gui.DecodeImage(panel.ImageData); err == nil {
				gui.Image(c, src).Width(approvalImageThumb).Height(approvalImageThumb).
					Radius(radiusSmall).Label("图片预览")
			}
		} else {
			box := gui.Box(c).Width(approvalImageThumb).Height(approvalImageThumb).
				Shrink(0).Background(t.SurfacePressed).Radius(radiusSmall).Label("图片缩略图")
			box.DrawOver(func(p *gui.Painter, r gui.Rect) {
				quarter := float32(approvalImageThumb) / 4
				inner := gui.Rect{X: r.X + quarter, Y: r.Y + quarter,
					W: r.W - 2*quarter, H: r.H - 2*quarter}
				paintIcon(p, iconEye, inner, t.TextMuted)
			})
		}
		if panel.SizeText != "" {
			typeCode.style(t, gui.Text(c, panel.SizeText)).TextColor(t.TextMuted)
		}
	})
}

// siteBadge 是站点徽记的本地回落：首字母 + 由主机名推出来的固定底色（参考产品
// FaviconImg 在取不到 favicon 时画的同一个徽记）。本端不联网取图标。
func siteBadge(c *gui.Context, host string, size float32) {
	initial := strings.ToUpper(string([]rune(host)[0]))
	color := hostColor(host)
	box := gui.Box(c).Width(size).Height(size).Shrink(0).Label("站点：" + host)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, color, size/2)
	})
	box.Children(func() {
		typeMicro.style(c.Theme(), gui.Text(c, initial)).TextColor(gui.Hex("#ffffff")).
			TextAlign(gui.Center)
	})
}

// approvalHostOf 取 URL 的主机名；不是 http(s) URL 或解析不出时为空。
func approvalHostOf(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	lower := strings.ToLower(trimmed)
	for _, scheme := range []string{"http://", "https://"} {
		if !strings.HasPrefix(lower, scheme) {
			continue
		}
		rest := trimmed[len(scheme):]
		if index := strings.IndexAny(rest, "/?#"); index >= 0 {
			rest = rest[:index]
		}
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		if colon := strings.LastIndex(rest, ":"); colon >= 0 {
			rest = rest[:colon]
		}
		return rest
	}
	return ""
}

// hostAvatarColors 是站点徽记的底色表（参考产品 hostnameColor 的同一串颜色）。
var hostAvatarColors = []gui.Color{
	gui.Hex("#ef4444"), gui.Hex("#f97316"), gui.Hex("#f59e0b"), gui.Hex("#84cc16"),
	gui.Hex("#22c55e"), gui.Hex("#14b8a6"), gui.Hex("#06b6d4"), gui.Hex("#3b82f6"),
	gui.Hex("#6366f1"), gui.Hex("#8b5cf6"), gui.Hex("#a855f7"), gui.Hex("#d946ef"),
	gui.Hex("#ec4899"), gui.Hex("#f43f5e"),
}

// hostColor 由主机名推一个固定底色（参考产品 hostnameColor 的同一算法）。
func hostColor(host string) gui.Color {
	var hash int32
	for _, char := range host {
		hash = (hash << 5) - hash + int32(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hostAvatarColors[int(hash)%len(hostAvatarColors)]
}

// decodeImageDataURI 解出 data URI 里的图片字节；只认 `data:image/...;base64,`。
func decodeImageDataURI(text string) ([]byte, bool) {
	const marker = ";base64,"
	lower := strings.ToLower(text)
	if !strings.HasPrefix(lower, "data:image/") {
		return nil, false
	}
	index := strings.Index(lower, marker)
	if index < 0 {
		return nil, false
	}
	encoded := strings.TrimSpace(text[index+len(marker):])
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}
