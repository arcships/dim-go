package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/workspace"
)

// pushFacts 把一串事实追加到会话历史并推送实时通知。
func pushFacts(t *testing.T, server *fakeappserver.Server, sessionID string, events ...protocol.RawEvent) {
	t.Helper()
	for _, event := range events {
		if err := server.PushEvent(sessionID, event); err != nil {
			t.Fatalf("推送事实 %s: %v", event.ID, err)
		}
	}
}

// TestRunChangesMergesToolDeclarations 核对变更读取：多工具合并、去重、越界与
// 按 run 过滤。
func TestRunChangesMergesToolDeclarations(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("准备工作区文件: %v", err)
	}
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession, workspaceRoot: root})
	ctx := context.Background()

	pushFacts(t, server, testSession,
		// write 工具的 structuredContent.path
		rawEvent("e30", 30, "step", "tool-use", `{"turn_id":"t1","request_id":"r1","call":{"id":"c-write","name":"write","input":{"path":"a.txt"}}}`),
		rawEvent("e31", 31, "step", "tool-completed", `{"turn_id":"t1","request_id":"r1","call_id":"c-write","result":{"structuredContent":{"path":"a.txt"}}}`),
		// 多路径声明 + 越界路径 + fileWritten effect
		rawEvent("e32", 32, "step", "tool-use", `{"turn_id":"t1","request_id":"r1","call":{"id":"c-batch","name":"apply_patch","input":{}}}`),
		rawEvent("e33", 33, "step", "tool-completed", `{"turn_id":"t1","request_id":"r1","call_id":"c-batch","result":{"structuredContent":{"affectedPaths":["a.txt","sub/b.txt","../outside.txt"],"effects":[{"type":"fileWritten","path":"sub/c.txt"}]}}}`),
		// edit 的 path（与 write 重复）与批量结果的形状
		rawEvent("e34", 34, "step", "tool-use", `{"turn_id":"t1","request_id":"r1","call":{"id":"c-edit","name":"edit","input":{}}}`),
		rawEvent("e35", 35, "step", "tool-call", `{"turn_id":"t1","request_id":"r1","calls":[]}`),
		rawEvent("e36", 36, "step", "tool-result", `{"turn_id":"t1","request_id":"r1","results":[{"call_id":"c-edit","result":{"structuredContent":{"path":"a.txt"}}}]}`),
		// 别的 run 的声明不进本次清单
		rawEvent("e37", 37, "step", "tool-use", `{"turn_id":"t2","request_id":"r2","call":{"id":"c-other","name":"write","input":{}}}`),
		rawEvent("e38", 38, "step", "tool-completed", `{"turn_id":"t2","request_id":"r2","call_id":"c-other","result":{"structuredContent":{"path":"other.txt"}}}`),
	)

	changes, err := result.base.RunChanges(ctx, testSession, "t1")
	if err != nil {
		t.Fatalf("读取变更: %v", err)
	}
	if changes.Coverage != workspace.CoverageToolReports {
		t.Fatalf("覆盖级别应为工具声明级: %q", changes.Coverage)
	}
	if changes.Truncated {
		t.Fatalf("不应截断: %+v", changes)
	}
	if changes.Declarations != 3 {
		t.Fatalf("应有三条带路径的声明，实际 %d", changes.Declarations)
	}
	wantWorkspace := map[string]bool{"a.txt": true, "sub/b.txt": false, "sub/c.txt": false}
	got := map[string]workspace.RunChange{}
	for _, change := range changes.Paths {
		got[change.Path] = change
	}
	for path, exists := range wantWorkspace {
		change, ok := got[path]
		if !ok {
			t.Fatalf("缺少工作区内路径 %q: %+v", path, changes.Paths)
		}
		if change.Scope != workspace.ChangeWorkspace {
			t.Fatalf("%q 应判为工作区内: %+v", path, change)
		}
		if change.Exists != exists {
			t.Fatalf("%q 的存在性应为 %t: %+v", path, exists, change)
		}
	}
	if _, ok := got["other.txt"]; ok {
		t.Fatalf("别的 run 的声明混进来了: %+v", changes.Paths)
	}
	external := 0
	for _, change := range changes.Paths {
		if change.Scope == workspace.ChangeExternal {
			external++
			if !filepath.IsAbs(change.Path) {
				t.Fatalf("工作区外的路径应是绝对路径: %+v", change)
			}
			if !strings.HasSuffix(change.Path, "outside.txt") {
				t.Fatalf("工作区外路径不对: %+v", change)
			}
		}
	}
	if external != 1 {
		t.Fatalf("应有一条工作区外路径，实际 %d: %+v", external, changes.Paths)
	}
	if changes.Total != 4 {
		t.Fatalf("去重后应有 4 条路径，实际 %d: %+v", changes.Total, changes.Paths)
	}
	// 工作区内在前：第一条必须是工作区内路径。
	if changes.Paths[0].Scope != workspace.ChangeWorkspace {
		t.Fatalf("工作区内路径应排在前面: %+v", changes.Paths)
	}
}

// TestRunChangesEmptyAndErrors 核对：声明缺失返回空、未知 run 为空、会话未打开报错、
// 没有工作区根报错。
func TestRunChangesEmptyAndErrors(t *testing.T) {
	root := t.TempDir()
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession, workspaceRoot: root})
	ctx := context.Background()

	pushFacts(t, server, testSession,
		// 只有文本结果、没有声明
		rawEvent("e40", 40, "step", "tool-use", `{"turn_id":"t1","request_id":"r1","call":{"id":"c-read","name":"read_file","input":{}}}`),
		rawEvent("e41", 41, "step", "tool-completed", `{"turn_id":"t1","request_id":"r1","call_id":"c-read","result":{"content":[{"type":"text","text":"package main"}]}}`),
		// 非 write/edit 工具的 structuredContent.path 不算声明
		rawEvent("e42", 42, "step", "tool-use", `{"turn_id":"t1","request_id":"r1","call":{"id":"c-grep","name":"grep","input":{}}}`),
		rawEvent("e43", 43, "step", "tool-completed", `{"turn_id":"t1","request_id":"r1","call_id":"c-grep","result":{"structuredContent":{"path":"ignored.txt"}}}`),
	)

	changes, err := result.base.RunChanges(ctx, testSession, "t1")
	if err != nil {
		t.Fatalf("读取变更: %v", err)
	}
	if len(changes.Paths) != 0 || changes.Declarations != 0 {
		t.Fatalf("没有声明时应返回空清单: %+v", changes)
	}
	if changes.Coverage != workspace.CoverageToolReports {
		t.Fatalf("空清单也要标明覆盖级别: %+v", changes)
	}

	// 未知 run：空清单，不是错误。
	unknown, err := result.base.RunChanges(ctx, testSession, "turn-does-not-exist")
	if err != nil || len(unknown.Paths) != 0 {
		t.Fatalf("未知 run 应返回空清单: %+v err=%v", unknown, err)
	}

	// 会话没打开：来源报错。
	if _, err := result.base.RunChanges(ctx, "not-open", "t1"); !errors.Is(err, ErrSessionNotOpen) {
		t.Fatalf("未打开会话应报错: %v", err)
	}

	// 没有工作区根：直接报不可用，不扫描任何目录。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	noWorkspace := startAssembly(t, server2, assemblyOptions{session: testSession})
	if _, err := noWorkspace.base.RunChanges(ctx, testSession, "t1"); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("没有工作区根应报错: %v", err)
	}
}

// TestRunChangesTruncates 核对上限：声明多于上限时截断并显式标记。
func TestRunChangesTruncates(t *testing.T) {
	root := t.TempDir()
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession, workspaceRoot: root})
	ctx := context.Background()

	paths := make([]string, 0, workspace.MaxChangePaths+20)
	for i := 0; i < workspace.MaxChangePaths+20; i++ {
		paths = append(paths, filepath.ToSlash(filepath.Join("dir", "file-"+itoa(i)+".txt")))
	}
	payload, err := json.Marshal(map[string]any{
		"structuredContent": map[string]any{"affectedPaths": paths},
	})
	if err != nil {
		t.Fatalf("编码载荷: %v", err)
	}
	pushFacts(t, server, testSession,
		rawEvent("e50", 50, "step", "tool-use", `{"turn_id":"t1","request_id":"r1","call":{"id":"c-many","name":"write","input":{}}}`),
		rawEvent("e51", 51, "step", "tool-completed",
			`{"turn_id":"t1","request_id":"r1","call_id":"c-many","result":`+string(payload)+`}`),
	)

	changes, err := result.base.RunChanges(ctx, testSession, "t1")
	if err != nil {
		t.Fatalf("读取变更: %v", err)
	}
	if !changes.Truncated || changes.Total != workspace.MaxChangePaths+20 {
		t.Fatalf("应截断且保留总数: %+v", changes)
	}
	if len(changes.Paths) != workspace.MaxChangePaths {
		t.Fatalf("应只保留 %d 条，实际 %d", workspace.MaxChangePaths, len(changes.Paths))
	}
	if changes.Note == "" {
		t.Fatal("截断应带说明")
	}
}

// itoa 是测试里的小工具，避免为拼路径引入额外依赖。
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
