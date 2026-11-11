package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/settings"
)

// TestModalityGateChecksHelloBeforeSessionMethods 核对门禁顺序：hello 声明即放行，
// 其次才看会话方法表；两者都没有才算缺失。
func TestModalityGateChecksHelloBeforeSessionMethods(t *testing.T) {
	// 只要 hello 声明，即可放行，且不需要读会话方法表。
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"modality.task.status", "modality.task.list"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	gate := &modalitySession{app: result.base}
	before := len(server.RequestsFor("session.methods"))
	ok, err := gate.SupportsModalityMethod(context.Background(), testSession, "modality.task.status")
	if err != nil || !ok {
		t.Fatalf("hello 声明时应放行: ok=%t err=%v", ok, err)
	}
	if got := len(server.RequestsFor("session.methods")); got != before {
		t.Fatalf("hello 命中时不应额外读会话方法表（%d -> %d 次）", before, got)
	}

	// hello 没声明，但会话方法表声明了：同样放行。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.methods", Result: map[string]any{"methods": []any{
				map[string]any{"name": "modality.realtime.send"},
			}}},
		},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	gate2 := &modalitySession{app: result2.base}
	ok, err = gate2.SupportsModalityMethod(context.Background(), testSession, "modality.realtime.send")
	if err != nil || !ok {
		t.Fatalf("会话方法表声明时应放行: ok=%t err=%v", ok, err)
	}

	// 两边都没有：缺失，且不发出任何 modality 请求。
	server3 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result3 := startAssembly(t, server3, assemblyOptions{session: testSession})
	gate3 := &modalitySession{app: result3.base}
	ok, err = gate3.SupportsModalityMethod(context.Background(), testSession, "modality.task.result")
	if err != nil || ok {
		t.Fatalf("两边都没有应判缺失: ok=%t err=%v", ok, err)
	}
	if got := len(server3.RequestsFor("modality.task.result")); got != 0 {
		t.Fatalf("判缺失时不应请求 modality 方法（%d 次）", got)
	}
}

// TestModalitySessionMapsTasks 核对模态适配器：任务状态、清单与结果读取的映射。
func TestModalitySessionMapsTasks(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{
			"modality.task.status", "modality.task.list", "modality.task.result", "modality.realtime.send",
		},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "modality.task.status", Result: map[string]any{"task": map[string]any{
				"version": 1, "task_id": "task-1", "revision": 3,
				"category": "image", "operation": "ocr", "state": "running",
				"created_at": "2026-11-11T01:00:00Z", "updated_at": "2026-11-11T01:00:05Z",
				"progress": map[string]any{"fraction": 0.5, "message": "半程"},
			}}},
			{Method: "modality.task.list", Result: map[string]any{
				"tasks": []any{map[string]any{
					"task_id": "task-2", "revision": 1, "category": "image", "operation": "ocr",
					"state": "succeeded", "created_at": "2026-11-11T01:00:00Z",
					"updated_at": "2026-11-11T01:00:02Z",
					"result": map[string]any{
						"files": []any{map[string]any{
							"name": "text.txt", "mediaType": "text/plain", "bytes": 3, "sha256": "abc",
						}},
						"data": map[string]any{"text": "ok"},
					},
				}},
				"next": "cursor-1",
			}},
			{Method: "modality.task.result", Result: map[string]any{"data_base64": "aGk=", "eof": true}},
			{Method: "modality.realtime.send", Result: map[string]any{}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	gate := &modalitySession{app: result.base}

	task, err := gate.ReadModalityTask(ctx, testSession, "task-1")
	if err != nil {
		t.Fatalf("读取任务状态: %v", err)
	}
	if task.ID != "task-1" || task.State != settings.ModalityTaskRunning || task.Revision != 3 {
		t.Fatalf("任务状态映射不对: %+v", task)
	}
	if task.Progress == nil || task.Progress.Fraction == nil || *task.Progress.Fraction != 0.5 {
		t.Fatalf("任务进度映射不对: %+v", task.Progress)
	}
	statusParams := requestParams(t, server.RequestsFor("modality.task.status")[0])
	if statusParams["task_id"] != "task-1" {
		t.Fatalf("任务状态参数不对: %v", statusParams)
	}

	page, err := result.base.Settings().ListTasks(ctx, testSession, settings.ModalityTaskQuery{Limit: 5})
	if err != nil {
		t.Fatalf("读取任务清单: %v", err)
	}
	if len(page.Tasks) != 1 || page.Next != "cursor-1" {
		t.Fatalf("任务清单映射不对: %+v", page)
	}
	if page.Tasks[0].Result == nil || len(page.Tasks[0].Result.Files) != 1 ||
		page.Tasks[0].Result.Files[0].Name != "text.txt" {
		t.Fatalf("任务结果文件映射不对: %+v", page.Tasks[0].Result)
	}
	listParams := requestParams(t, server.RequestsFor("modality.task.list")[0])
	if listParams["limit"] != float64(5) {
		t.Fatalf("任务清单参数不对: %v", listParams)
	}

	chunk, err := result.base.Settings().ReadTaskResult(ctx, testSession, settings.ModalityTaskResultQuery{
		TaskID: "task-1", Name: "text.txt", Length: 8,
	})
	if err != nil {
		t.Fatalf("读取任务结果: %v", err)
	}
	if chunk.DataBase64 != "aGk=" || !chunk.EOF {
		t.Fatalf("任务结果映射不对: %+v", chunk)
	}

	if err := gate.SendModalityRealtime(ctx, testSession, "conn-1", settings.ModalityRealtimeFrame{
		Kind: "text", Data: json.RawMessage(`{"text":"hi"}`),
	}); err != nil {
		t.Fatalf("发送实时帧: %v", err)
	}
	sendParams := requestParams(t, server.RequestsFor("modality.realtime.send")[0])
	if sendParams["connectionId"] != "conn-1" {
		t.Fatalf("实时帧参数不对: %v", sendParams)
	}

	// 能力缺失时零请求：不声明任何 modality 能力的实例上读任务应报能力缺失。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	if _, err := result2.base.Settings().ListTasks(ctx, testSession, settings.ModalityTaskQuery{}); !errors.Is(err, settings.ErrCapabilityMissing) {
		t.Fatalf("缺能力时应报能力缺失: %v", err)
	}
	if got := len(server2.RequestsFor("modality.task.list")); got != 0 {
		t.Fatalf("缺能力时不应请求 modality.task.list（%d 次）", got)
	}
}

// TestShellRunsReadsListWithoutContent 核对 shell.list 摘要：只保留标识与元信息，
// 不回传命令正文。
func TestShellRunsReadsListWithoutContent(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"shell.list"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "shell.list", Result: map[string]any{"runs": []any{map[string]any{
				"runId": "run-1", "branchId": "main", "mode": "exec",
				"command": "printf DIMAGENT_SECRET", "startedAt": 1700000000,
			}}}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	runs, err := result.base.ShellRuns(context.Background(), testSession)
	if err != nil {
		t.Fatalf("读取运行清单: %v", err)
	}
	if len(runs) != 1 || runs[0].RunID != "run-1" || runs[0].Mode != "exec" {
		t.Fatalf("运行摘要映射不对: %+v", runs)
	}
	// 结构里没有命令字段；再确认 JSON 序列化后也不含命令正文。
	encoded, err := json.Marshal(runs)
	if err != nil {
		t.Fatalf("序列化运行摘要: %v", err)
	}
	if containsText(string(encoded), "DIMAGENT_SECRET") {
		t.Fatalf("运行摘要泄露了命令正文: %s", encoded)
	}
	params := requestParams(t, server.RequestsFor("shell.list")[0])
	if params["sessionId"] != testSession {
		t.Fatalf("运行清单参数不对: %v", params)
	}
}

func containsText(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		len(needle) == 0 || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for index := 0; index+len(needle) <= len(haystack); index++ {
		if haystack[index:index+len(needle)] == needle {
			return index
		}
	}
	return -1
}

// 保证 protocol 的引用在编译期成立（result 类型在 shell.run 回执里用到）。
var _ = protocol.ShellRunResult{}
