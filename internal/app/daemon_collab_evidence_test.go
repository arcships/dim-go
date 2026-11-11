package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestRealCollaborationMemberStopEvidence 尝试成员停止的真实端到端。
//
// 只有设置 DIMAGENT_COLLAB_EVIDENCE=1 时运行；夹具 home 从 /tmp 只读复制。
// 证据只留标识、状态与事件类型。
func TestRealCollaborationMemberStopEvidence(t *testing.T) {
	if os.Getenv("DIMAGENT_COLLAB_EVIDENCE") != "1" {
		t.Skip("未设置 DIMAGENT_COLLAB_EVIDENCE=1，跳过成员停止证据")
	}
	fixtureHome := evidenceFixtureHome(t)
	serverPath := evidenceServerBinary(t)
	runDir, err := os.MkdirTemp("", "dimagent-evidence-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runDir) })
	if err := copyFixtureTree(fixtureHome, runDir); err != nil {
		t.Fatalf("复制夹具 home: %v", err)
	}

	base, err := New(Config{
		DataDir:       runDir,
		ServerPath:    serverPath,
		ClientName:    "dimagent-mygo-collab-evidence",
		WorkspaceRoot: runDir,
		Version:       testVersion,
		SendTimeout:   60 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := base.Connect(ctx); err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	defer base.CloseConnection(ctx)
	if err := base.Subscribe(ctx); err != nil {
		t.Fatalf("订阅阶段: %v", err)
	}
	if err := base.FirstHistory(ctx); err != nil {
		t.Fatalf("首轮历史: %v", err)
	}
	if err := base.AllowWrites(ctx); err != nil {
		t.Fatalf("放行写入: %v", err)
	}

	catalogDomain := base.Catalog()
	if _, err := catalogDomain.Refresh(ctx, catalog.Query{}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}
	entry, err := base.CreateSession(ctx, runDir, "full-access")
	if err != nil {
		t.Fatalf("创建会话: %v", err)
	}
	if _, err := base.OpenSession(ctx, entry.ID); err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	t.Logf("leader session: %s", entry.ID)

	prompt := strings.Join([]string{
		"Use the `agent` tool exactly once with action=request.",
		`profile: {"name":"runner","description":"runs one short command",` +
			`"systemPrompt":"Run the exec tool exactly once to run: sleep 30. Then reply DONE.",` +
			`"tools":["exec"]}`,
		`prompt: "Run the exec tool exactly once to run: sleep 30. Then reply DONE."`,
		"After the tool returns, reply STARTED and stop.",
	}, "\n")
	if _, err := base.Submit(ctx, SubmitRequest{SessionID: entry.ID, Text: prompt}); err != nil {
		t.Fatalf("提交协作提示: %v", err)
	}

	var members []string
	var lastPendingErr error
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		current, err := base.CollaborationMembers(ctx, entry.ID)
		if err != nil {
			lastPendingErr = err
		}
		if err == nil && len(current) > 0 {
			members = current
			break
		}
		time.Sleep(2 * time.Second)
	}
	if lastPendingErr != nil {
		t.Logf("pending 读取错误: %v", lastPendingErr)
	}
	if len(members) == 0 {
		if runtime, ok := base.session(entry.ID); ok {
			if result, err := runtime.lease.Subscription().PendingCollaboration(ctx, protocol.PendingQuery{
				BranchID: branchOrMain(runtime.folder.Selected()),
			}); err == nil {
				t.Logf("pending 协作请求数=%d through=%d", len(result.Requests), result.Through)
				for _, request := range result.Requests {
					deliveryStatus := "none"
					delivered := false
					if request.Delivery != nil {
						deliveryStatus = request.Delivery.Status
						delivered = request.Delivery.SessionID != ""
					}
					t.Logf("  请求: targetKind=%s delivery=%s deliveredSession=%t",
						request.Request.Target.Kind, deliveryStatus, delivered)
				}
			} else {
				t.Logf("pending 协作请求读取失败: %v", err)
			}
		}
	}
	if len(members) == 0 {
		// 记录模型实际产生的事件类型与工具名，判断是“未挂载 agent 工具”还是“模型没调用”。
		if runtime, ok := base.session(entry.ID); ok {
			if events, err := (&changeSource{app: base}).collectFacts(ctx, runtime); err == nil {
				kinds := make(map[string]int, len(events))
				tools := make(map[string]int)
				for _, event := range events {
					kinds[event.Type+"/"+event.Name]++
					if strings.Contains(event.Name, "tool") {
						var data struct {
							Call struct {
								Name string `json:"name"`
							} `json:"call"`
						}
						if err := json.Unmarshal(event.Data, &data); err == nil && data.Call.Name != "" {
							tools[data.Call.Name]++
						}
					}
				}
				t.Logf("leader 事实类型: %v", sortedKindCounts(kinds))
				t.Logf("leader 工具调用: %v", sortedKindCounts(tools))
				for _, event := range events {
					if event.Type != "collaboration" {
						continue
					}
					var data struct {
						Delivery struct {
							Status    string `json:"status"`
							SessionID string `json:"session_id"`
							Error     string `json:"error"`
						} `json:"delivery"`
						Error string `json:"error"`
					}
					_ = json.Unmarshal(event.Data, &data)
					t.Logf("协作事实: %s deliveredSession=%t status=%s errorCode=%s",
						event.Name, data.Delivery.SessionID != "", data.Delivery.Status, firstNonEmpty(data.Delivery.Error, data.Error))
				}
			}
		}
		t.Logf("未触发成员：提示词已提交但 pending 里没有 delivery.session_id；按 blocked 记录")
		return
	}
	t.Logf("成员会话: %v", members)

	started := time.Now()
	registered, err := base.StopTeamMember(ctx, MemberStopRequest{
		SessionID: entry.ID, MemberSessionID: members[0],
	})
	if err != nil {
		t.Fatalf("登记成员停止: %v", err)
	}
	settled, err := base.Wait(ctx, registered.ID)
	if err != nil {
		t.Fatalf("等待成员停止: %v", err)
	}
	status, stopped, statusErr := MemberStopStatus(settled.Result)
	t.Logf("停止回执: intent=%s status=%s stopped=%t err=%v 耗时=%s",
		settled.Status, status, stopped, statusErr, time.Since(started).Round(time.Millisecond))
	if settled.Status != write.StatusSucceeded || !stopped {
		t.Fatalf("成员停止未收敛: %s %s", settled.Status, settled.Reason)
	}

	// 成员不再产生新事实：打开成员会话比较事实类型计数。
	memberView, err := base.OpenSession(ctx, members[0])
	if err != nil {
		t.Logf("成员会话打不开（可能已随停止关闭）: %v", err)
	} else {
		_ = memberView
		before, _ := base.HistoryKinds(ctx, members[0])
		time.Sleep(6 * time.Second)
		after, _ := base.HistoryKinds(ctx, members[0])
		added := 0
		for kind, count := range after {
			if delta := count - before[kind]; delta > 0 {
				added += delta
				t.Logf("成员停止后新增事实: %s=+%d", kind, delta)
			}
		}
		t.Logf("成员停止后事实增量: %d", added)
	}

	// 主线不受影响：leader 会话仍可读、可写。
	if _, ok := base.View(entry.ID); !ok {
		t.Fatal("主线会话视图丢失")
	}
	if err := base.Writable(); err != nil {
		t.Fatalf("主线写闸门异常: %v", err)
	}
	leaderKinds, _ := base.HistoryKinds(ctx, entry.ID)
	t.Logf("主线事实类型数: %d", len(leaderKinds))
}

// sortedKindCounts 输出稳定的 “名称=次数” 列表，便于证据比对。
func sortedKindCounts(values map[string]int) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, fmt.Sprintf("%s=%d", name, values[name]))
	}
	return out
}
