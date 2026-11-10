package workspace

// 真实模型证据：用 /tmp 夹具（真实 daemon + 真实 provider）跑一次写入会话与一次
// exec 会话，验证变更读取只认工具结果声明的路径，不把 exec 造成的改动算进来。
//
// 日志只留路径数量、分类、coverage 与布尔值；不打印文件内容、命令正文、提示词、
// 模型输出或凭据。夹具只读：只复制到新的临时目录后使用。

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

const evidenceMainBranch = "main"

// TestRealDaemonRunChangesEvidence 在真实模型上验证变更读取的声明来源与边界。
//
// 用法：
//
//	DIM_REAL_MODEL_FIXTURE=/tmp/dimagent-real-model \
//	  go test -run TestRealDaemonRunChangesEvidence -v -count=1 ./internal/workspace/
func TestRealDaemonRunChangesEvidence(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("未设置 DIM_REAL_MODEL_FIXTURE：真实模型证据 blocked（缺 /tmp 夹具）")
	}
	serverPath := filepath.Join(fixture, "bin", "dimcode-server")
	if _, err := os.Stat(serverPath); err != nil {
		t.Skipf("夹具缺少 daemon 可执行文件: %v", err)
	}

	runDir, err := os.MkdirTemp("/tmp", "d7changes-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runDir) })
	home := filepath.Join(runDir, "home")
	evidenceCopyTree(t, filepath.Join(fixture, "home"), home)
	for _, stale := range []string{"daemon.sock", "daemon.sock.lock", "daemon.start.lock"} {
		_ = os.Remove(filepath.Join(home, "rust", stale))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	supervisor, err := daemon.New(daemon.Config{
		DataDir:    home,
		ServerPath: serverPath,
		ServerArgs: []string{"app-server"},
	})
	if err != nil {
		t.Fatalf("创建监督者: %v", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		t.Skipf("连接真实 daemon 失败（夹具账号/网络不可用）: %v", err)
	}
	workDir := filepath.Join(runDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}
	// macOS 的 /tmp 是指向 /private/tmp 的符号链接；工具结果声明的是解析后路径。
	resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		t.Fatalf("解析工作目录: %v", err)
	}
	ws, err := NewWorkspace(workDir)
	if err != nil {
		t.Fatalf("创建工作区: %v", err)
	}

	var created protocol.SessionCreateResult
	// 不传 tools：会话声明宿主全部工具（写工具与 exec 都要在）；full-access 避免审批打断。
	if err := conn.Client.CallInto(ctx, "session.create", map[string]any{
		"cwd":              resolvedWorkDir,
		"permissionPolicy": map[string]any{"preset": "full-access"},
	}, &created); err != nil {
		t.Skipf("创建真实会话失败（账号/模型不可用）: %v", err)
	}
	sessionID := created.SessionID

	recorder := newEvidenceRecorder()
	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	_, lease, err := manager.Acquire(ctx, sessionID, session.Listener{
		OnEvent: func(event session.Event) {
			if event.Method != "session.event" {
				return
			}
			var params protocol.SessionEventParams
			if err := json.Unmarshal(event.Params, &params); err != nil {
				return
			}
			recorder.add(params.Event)
		},
		OnResync: func(session.Resync) {},
	})
	if err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	t.Cleanup(func() {
		lease.Release()
		manager.Close()
		_, _ = supervisor.Shutdown(context.Background())
	})

	writeTarget := filepath.Join(resolvedWorkDir, "d7-declared-write.txt")
	writeFloor := recorder.head()
	evidenceSubmit(t, ctx, conn.Client, sessionID,
		"Use the write tool exactly once to create the file "+
			writeTarget+
			" with the exact text DIMAGENT_D7_WRITE_OK. Do not use the exec tool. "+
			"After the write tool succeeds, reply with the single word DONE.")
	evidenceWaitTurnEnd(t, ctx, recorder, writeFloor)
	writeEvents := recorder.after(writeFloor)
	writeDecls, writeTools := evidenceDeclarations(writeEvents)
	if len(writeDecls) == 0 {
		t.Skipf("模型没有产生可用的写入声明（工具调用：%v；事件：%v）",
			writeTools, recorder.kinds())
	}

	writeChanges, err := ws.ReadRunChanges(ctx, evidenceSource(writeDecls), sessionID, "evidence-write")
	if err != nil {
		t.Fatalf("读取写入轮变更: %v", err)
	}
	writeEntry, writeFound := findChange(writeChanges, writeTarget)
	if !writeFound {
		t.Fatalf("写入轮的工具声明没有进入变更清单：声明 %d 条，清单 %d 条",
			len(writeDecls), len(writeChanges.Paths))
	}
	if writeEntry.Scope != ChangeWorkspace || !writeEntry.Exists {
		t.Fatalf("写入轮分类或存在性错误：scope=%s exists=%v",
			writeEntry.Scope, writeEntry.Exists)
	}
	if writeChanges.Coverage != CoverageToolReports {
		t.Fatalf("覆盖级别 = %q，期望 %q", writeChanges.Coverage, CoverageToolReports)
	}
	if _, statErr := os.Stat(writeTarget); statErr != nil {
		t.Fatalf("声明的写入路径在磁盘上不存在: %v", statErr)
	}
	writeWorkspace, writeExternal := countScopes(writeChanges)
	t.Logf("写入轮: 声明=%d 路径=%d workspace=%d external=%d 目标存在=%v coverage=%s",
		len(writeDecls), writeChanges.Total, writeWorkspace, writeExternal,
		writeEntry.Exists, writeChanges.Coverage)

	// 边界：exec 写出的文件不做声明，不能出现在变更清单里。
	execTarget := filepath.Join(resolvedWorkDir, "d7-exec-only.txt")
	execFloor := recorder.head()
	evidenceSubmit(t, ctx, conn.Client, sessionID,
		"Use the exec tool exactly once to run this shell command: printf DIMAGENT_D7_EXEC_OK > "+
			execTarget+". Do not use the write or edit tools. After the command succeeds, "+
			"reply with the single word DONE.")
	evidenceWaitTurnEnd(t, ctx, recorder, execFloor)
	execEvents := recorder.after(execFloor)
	execDecls, execTools := evidenceDeclarations(execEvents)
	if !containsString(execTools, "exec") {
		t.Skipf("模型没有使用 exec（工具调用：%v；事件：%v）", execTools, recorder.kinds())
	}
	if containsString(execTools, "write") || containsString(execTools, "edit") {
		t.Skipf("模型在 exec 轮同时用了写工具（工具调用：%v），无法证明边界", execTools)
	}
	if _, statErr := os.Stat(execTarget); statErr != nil {
		t.Skipf("exec 没有在磁盘上产生文件（工具调用：%v），无法证明边界", execTools)
	}
	execChanges, err := ws.ReadRunChanges(ctx, evidenceSource(execDecls), sessionID, "evidence-exec")
	if err != nil {
		t.Fatalf("读取 exec 轮变更: %v", err)
	}
	if _, found := findChange(execChanges, execTarget); found {
		t.Fatalf("exec 造成的改动被计入了变更清单（清单 %d 条）", execChanges.Total)
	}
	execWorkspace, execExternal := countScopes(execChanges)
	t.Logf("exec 轮: 声明=%d 路径=%d workspace=%d external=%d 目标在清单=%v 目标存在=%v coverage=%s",
		len(execDecls), execChanges.Total, execWorkspace, execExternal,
		false, true, execChanges.Coverage)
}

// evidenceSource 用真实工具结果构造的最小 ChangeSource。
type evidenceSource []ToolResultDeclaration

func (s evidenceSource) ToolResultDeclarations(context.Context, string, string) ([]ToolResultDeclaration, error) {
	return []ToolResultDeclaration(s), nil
}

func evidenceSubmit(t *testing.T, ctx context.Context, conn interface {
	CallInto(context.Context, string, any, any) error
}, sessionID, text string) {
	t.Helper()
	var reply struct {
		Event protocol.RawEvent `json:"event"`
	}
	if err := conn.CallInto(ctx, "session.submit", map[string]any{
		"sessionId":    sessionID,
		"branchId":     evidenceMainBranch,
		"submissionId": evidenceUUID(t),
		"text":         text,
	}, &reply); err != nil {
		t.Fatalf("提交输入: %v", err)
	}
}

// evidenceWaitTurnEnd 等到一条序号大于 after 的 turn/end。
func evidenceWaitTurnEnd(t *testing.T, ctx context.Context, recorder *evidenceRecorder, after int64) {
	t.Helper()
	_, ok := recorder.wait(ctx, 8*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > after && event.Type == "turn" && event.Name == "end"
	})
	if !ok {
		t.Skipf("没有等到轮次结束（观察到事件：%v）", recorder.kinds())
	}
}

// evidenceDeclarations 从工具结果事件里提取声明，并返回观察到的工具名。
func evidenceDeclarations(events []protocol.RawEvent) ([]ToolResultDeclaration, []string) {
	var declarations []ToolResultDeclaration
	names := make(map[string]string)
	seenNames := make(map[string]struct{})
	var tools []string
	noteName := func(name string) {
		if name == "" {
			return
		}
		if _, ok := seenNames[name]; ok {
			return
		}
		seenNames[name] = struct{}{}
		tools = append(tools, name)
	}
	for _, event := range events {
		data := evidenceDecode(event.Data)
		switch event.Type + "/" + event.Name {
		case "step/tool-use":
			noteName(evidenceString(data, "name"))
		case "step/tool-call":
			for _, value := range evidenceArray(data["calls"]) {
				call := evidenceMap(value)
				noteName(evidenceString(call, "name"))
			}
		}
	}
	// 第二轮补名称：工具调用与结果可能不在同一批事件里，先扫一遍再提取。
	for _, event := range events {
		data := evidenceDecode(event.Data)
		if event.Type+"/"+event.Name != "step/tool-use" {
			continue
		}
		callID := evidenceString(data, "call_id", "callId", "id")
		if callID != "" {
			names[callID] = evidenceString(data, "name")
		}
	}
	for _, event := range events {
		data := evidenceDecode(event.Data)
		switch event.Type + "/" + event.Name {
		case "step/tool-call":
			for _, value := range evidenceArray(data["calls"]) {
				call := evidenceMap(value)
				callID := evidenceString(call, "id", "call_id", "callId")
				if callID != "" {
					names[callID] = evidenceString(call, "name")
				}
			}
		case "step/tool-completed":
			callID := evidenceString(data, "call_id", "callId")
			declaration := evidenceDeclaration(callID, names[callID], evidenceMap(data["result"]))
			if len(declaration.Paths) > 0 {
				declaration.Paths = dedupeStrings(declaration.Paths)
			}
			// 只保留有路径的声明，空结果不进入证据。
			if len(declaration.Paths) > 0 {
				declaration.ToolCallID = callID
				declaration.ToolName = names[callID]
				declarations = append(declarations, declaration)
			}
		case "step/tool-result":
			for _, value := range evidenceArray(data["results"]) {
				result := evidenceMap(value)
				callID := evidenceString(result, "call_id", "callId")
				declaration := evidenceDeclaration(callID, names[callID], evidenceMap(result["result"]))
				if len(declaration.Paths) > 0 {
					declaration.ToolCallID = callID
					declaration.ToolName = names[callID]
					declarations = append(declarations, declaration)
				}
			}
		}
	}
	return declarations, tools
}

// evidenceDeclaration 从一个工具结果里提取声明路径。
// 来源与产品基线一致：structuredContent.affectedPaths、write/edit 的 path、
// fileWritten effect；同时兼容 snake_case 字段。
func evidenceDeclaration(callID, toolName string, payload map[string]any) ToolResultDeclaration {
	declaration := ToolResultDeclaration{ToolCallID: callID, ToolName: toolName}
	if len(payload) == 0 {
		return declaration
	}
	structured := evidenceMap(payload["structuredContent"])
	if len(structured) == 0 {
		structured = evidenceMap(payload["structured_content"])
	}
	for _, value := range evidenceArray(structured["affectedPaths"]) {
		if path, ok := value.(string); ok && strings.TrimSpace(path) != "" {
			declaration.Paths = append(declaration.Paths, path)
		}
	}
	if toolName == "write" || toolName == "edit" {
		if path := evidenceString(structured, "path"); path != "" {
			declaration.Paths = append(declaration.Paths, path)
		}
	}
	for _, value := range evidenceArray(payload["effects"]) {
		effect := evidenceMap(value)
		if evidenceString(effect, "kind") == "fileWritten" {
			if path := evidenceString(effect, "path"); path != "" {
				declaration.Paths = append(declaration.Paths, path)
			}
		}
	}
	return declaration
}

// findChange 按解析后的绝对路径查找清单项。
func findChange(changes RunChanges, absPath string) (RunChange, bool) {
	for _, change := range changes.Paths {
		if change.AbsPath == absPath {
			return change, true
		}
	}
	return RunChange{}, false
}

func countScopes(changes RunChanges) (workspaceCount, externalCount int) {
	for _, change := range changes.Paths {
		if change.Scope == ChangeWorkspace {
			workspaceCount++
		} else {
			externalCount++
		}
	}
	return workspaceCount, externalCount
}

// evidenceRecorder 按到达顺序收事件；日志只用类型、序号与状态。
type evidenceRecorder struct {
	mu     sync.Mutex
	events []protocol.RawEvent
	signal chan struct{}
}

func newEvidenceRecorder() *evidenceRecorder {
	return &evidenceRecorder{signal: make(chan struct{}, 1)}
}

func (r *evidenceRecorder) add(event protocol.RawEvent) {
	r.mu.Lock()
	r.events = append(r.events, event)
	r.mu.Unlock()
	select {
	case r.signal <- struct{}{}:
	default:
	}
}

func (r *evidenceRecorder) head() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	var head int64
	for _, event := range r.events {
		if event.Sequence > head {
			head = event.Sequence
		}
	}
	return head
}

// after 返回序号大于 floor 的事件副本。
func (r *evidenceRecorder) after(floor int64) []protocol.RawEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]protocol.RawEvent, 0, len(r.events))
	for _, event := range r.events {
		if event.Sequence > floor {
			out = append(out, event)
		}
	}
	return out
}

func (r *evidenceRecorder) wait(
	ctx context.Context,
	timeout time.Duration,
	match func(protocol.RawEvent) bool,
) (protocol.RawEvent, bool) {
	deadline := time.Now().Add(timeout)
	for {
		r.mu.Lock()
		for _, event := range r.events {
			if match(event) {
				r.mu.Unlock()
				return event, true
			}
		}
		r.mu.Unlock()
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return protocol.RawEvent{}, false
		}
		wait := 200 * time.Millisecond
		if remaining < wait {
			wait = remaining
		}
		select {
		case <-r.signal:
		case <-ctx.Done():
			return protocol.RawEvent{}, false
		case <-time.After(wait):
		}
	}
}

func (r *evidenceRecorder) kinds() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := make(map[string]struct{})
	var out []string
	for _, event := range r.events {
		key := event.Type + "/" + event.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func evidenceDecode(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func evidenceMap(value any) map[string]any {
	if mapped, ok := value.(map[string]any); ok {
		return mapped
	}
	return nil
}

func evidenceArray(value any) []any {
	if values, ok := value.([]any); ok {
		return values
	}
	return nil
}

func evidenceString(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if text, ok := value[key].(string); ok && text != "" {
			return text
		}
	}
	return ""
}

func evidenceUUID(t *testing.T) string {
	t.Helper()
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		t.Fatalf("生成标识: %v", err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// evidenceCopyTree 递归复制夹具目录，保留权限位；夹具只读，只从来源复制。
func evidenceCopyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relative)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("复制夹具: %v", err)
	}
}
