package app

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/stubmodel"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// 真实回路的入口参数。路径只从环境变量取：
//
//	DIM_APP_SERVER_BIN  真实 App Server 可执行文件（必需）
//	DIM_E2E_HOME        本次运行的客户端数据目录（可选，默认 /tmp/e2e-home）
//
// 服务端用 `--model-config` 拉起一个「开发档」配置：它带产品级装配（权限档位
// 读写、工作区工具、本地执行），模型指向本进程内的 OpenAI 兼容替身。服务端、
// 会话、工具执行与审批都是真的；只有语言模型是替身（不依赖任何外部凭据，
// 也不读本机 home）。替身按请求里的工具清单决定下一步：带工具且还没有工具结果
// 时发起一次 exec 调用，拿到工具结果后回一段文本。
const (
	envE2EHome       = "DIM_E2E_HOME"
	defaultE2EHome   = "/tmp/e2e-home"
	e2ePrompt        = "执行 echo e2e-ok 并把输出报给我"
	e2eToolOutput    = "e2e-ok"
	e2eAnswer        = "工具已经执行，输出是 e2e-ok。"
	e2eLoopTimeout   = 3 * time.Minute
	e2eTurnPollEvery = 100 * time.Millisecond
)

// TestLiveLoopWithRealServer 跑通真实服务端上的整条回路：
// 起服务端 → 建会话 → 提交一条会触发工具的消息 → 收到流式事实 → 审批（出现就批准）
// → 轮结束，并断言时间线里既有工具行也有助手文本、轮状态为已结束。
func TestLiveLoopWithRealServer(t *testing.T) {
	if testing.Short() {
		t.Skip("短模式跳过真实服务端回路（需要真实 App Server 进程）")
	}
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("缺少 DIM_APP_SERVER_BIN（真实 App Server 可执行文件）；跳过真实回路")
	}
	if info, err := os.Stat(serverPath); err != nil || info.IsDir() {
		t.Fatalf("DIM_APP_SERVER_BIN 不可用: %v", err)
	}

	home := os.Getenv(envE2EHome)
	if home == "" {
		home = defaultE2EHome
	}
	// 每次运行都从空 home 开始：上一次运行的会话、锁与 socket 会改变这次的事实。
	if err := os.RemoveAll(home); err != nil {
		t.Fatalf("清理数据目录: %v", err)
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatalf("建数据目录: %v", err)
	}
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o700); err != nil {
		t.Fatalf("建工作目录: %v", err)
	}

	stub := stubmodel.New()
	modelServer := httptest.NewServer(stub)
	t.Cleanup(modelServer.Close)
	modelConfig := filepath.Join(home, "model-config.json")
	if err := stubmodel.WriteConfig(modelConfig, modelServer.URL+"/v1"); err != nil {
		t.Fatalf("写模型配置: %v", err)
	}
	t.Logf("模型: 替身地址=%s/v1 配置=%s", modelServer.URL, filepath.Base(modelConfig))

	base, err := New(Config{
		DataDir:        home,
		ServerPath:     serverPath,
		ModelConfig:    modelConfig,
		ClientName:     "dimagent-mygo-e2e",
		WorkspaceRoot:  work,
		Version:        testVersion,
		HistoryLimit:   50,
		SendTimeout:    90 * time.Second,
		RequestTimeout: 30 * time.Second,
		CloseTimeout:   20 * time.Second,
		Logf: func(format string, args ...any) {
			t.Logf("app: "+format, args...)
		},
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer closeCancel()
		if err := base.Close(closeCtx); err != nil {
			t.Logf("收尾: %v", err)
		}
	})

	// ① 启动：五阶段按宿主外壳的顺序走一遍。
	started := time.Now()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := base.Connect(ctx); err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	conn := base.connection()
	if conn == nil {
		t.Fatal("连接后没有 hello 结果")
	}
	t.Logf("启动: phase=%s 用时=%s 端点=%s", base.Phase(), time.Since(started).Round(time.Millisecond),
		filepath.Join(home, "rust", "daemon.sock"))
	t.Logf("hello: 实例=%s 版本=%s 协议=%d 档位=%s 自启=%t 能力=%d 项",
		conn.Hello.InstanceID, conn.Hello.ServerVersion, conn.Hello.ProtocolVersion,
		conn.Hello.DaemonProfile.Kind, conn.Owned, len(conn.Hello.Features))
	var ping protocol.PingResult
	if err := base.control().CallInto(ctx, "daemon.ping", map[string]any{}, &ping); err != nil {
		t.Fatalf("daemon.ping: %v", err)
	}
	t.Logf("服务端存活: 会话=%d 连接=%d 版本=%s", ping.Sessions, ping.Connections, ping.ServerVersion)
	if pid := serverPID(filepath.Join(home, "rust", "daemon.sock")); pid != "" {
		t.Logf("服务端进程: pid=%s 可执行文件=%s", pid, filepath.Base(serverPath))
	}

	if err := base.Subscribe(ctx); err != nil {
		t.Fatalf("订阅: %v", err)
	}
	if err := base.FirstHistory(ctx); err != nil {
		t.Fatalf("首轮历史: %v", err)
	}
	if err := base.AllowWrites(ctx); err != nil {
		t.Fatalf("放行写入: %v", err)
	}
	t.Logf("启动阶段: %s", base.Phase())

	// ② 会话：workspace-write 下 exec 需要审批。
	var created protocol.SessionCreateResult
	if err := base.control().CallInto(ctx, "session.create", map[string]any{
		"cwd":              work,
		"permissionPolicy": map[string]any{"preset": "workspace-write"},
	}, &created); err != nil {
		t.Fatalf("创建会话: %v", err)
	}
	if created.SessionID == "" {
		t.Fatalf("会话回执没有标识: %+v", created)
	}
	if _, err := base.OpenSession(ctx, created.SessionID); err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	tier, ok := base.PermissionTier(created.SessionID)
	if !ok {
		t.Fatalf("会话没有读到权限档位（服务端未提供 session.permission.get）")
	}
	t.Logf("会话: id=%s 已打开=true 权限档位=%q", created.SessionID, tier)

	// ③ 提交一条会触发工具的消息。
	intent, err := base.Submit(ctx, SubmitRequest{SessionID: created.SessionID, Text: e2ePrompt})
	if err != nil {
		t.Fatalf("提交: %v", err)
	}
	t.Logf("提交: intent=%s input=%s", intent.ID, intent.InputID)

	// ④ 等流式事实：审批出现就批准，直到这一轮结束。
	approvals := 0
	answered := make(map[string]bool)
	lastShape := ""
	deadline := time.Now().Add(e2eLoopTimeout)
	for {
		if time.Now().After(deadline) {
			view, _ := base.View(created.SessionID)
			t.Fatalf("超时：轮没有结束 approvals=%d 行=%s 替身请求=%d",
				approvals, describeRows(view.Rows), stub.Requests.Load())
		}
		view, open := base.View(created.SessionID)
		if !open {
			t.Fatal("会话视图消失")
		}
		// 流式事实：行序列每变一次记一行，作为协议层逐步推进的证据。
		if shape := describeRows(view.Rows); shape != lastShape {
			t.Logf("流: %s", shape)
			lastShape = shape
		}
		// 同一请求在应答写入成功到服务端结算之间仍留在队首：答过的就不再答一次。
		if view.HasCurrent && view.Current.Kind == interaction.PendingApproval && !answered[view.Current.RequestID] {
			approvals++
			answered[view.Current.RequestID] = true
			t.Logf("审批: 第 %d 次 request=%s action=%q 长期可允许=%t",
				approvals, view.Current.RequestID, view.Current.Action, view.Current.AllowLongTerm)
			outcome, respondErr := base.Respond(ctx, created.SessionID, view.Current.RequestID, interaction.Response{
				Decision: interaction.DecisionApprove,
			})
			if respondErr != nil {
				t.Fatalf("同意审批: %v", respondErr)
			}
			if outcome.Response == nil || outcome.Response.Status != write.StatusSucceeded {
				t.Fatalf("审批应答没有写入成功: %+v", outcome.Response)
			}
			continue
		}
		if view.LastTurnEnded && !view.HasCurrent && view.PendingCounts.Approvals == 0 {
			t.Logf("轮: index=%d outcome=%s 待决=%t 审批=%d", view.LastTurnIndex,
				view.LastTurnOutcome, view.HasCurrent, approvals)
			if assertErr := assertLoopFacts(view, approvals); assertErr != nil {
				t.Fatalf("%v；实际行=%s 替身请求=%d", assertErr, describeRows(view.Rows), stub.Requests.Load())
			}
			t.Logf("断言: 时间线=%s 审批次数=%d 替身请求=%d", describeRows(view.Rows), approvals, stub.Requests.Load())
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("等待轮结束超时: %v", ctx.Err())
		case <-time.After(e2eTurnPollEvery):
		}
	}

	// ⑤ 协议层事实序列：按序号读回时间线上的每一条事实，只留类型与序号。
	facts, err := (&changeSource{app: base}).collectFacts(ctx, mustRuntime(t, base, created.SessionID))
	if err != nil {
		t.Fatalf("读回历史事实: %v", err)
	}
	kinds := make(map[string]int, len(facts))
	sequence := make([]string, 0, len(facts))
	for _, event := range facts {
		key := event.Type + "/" + event.Name
		kinds[key]++
		sequence = append(sequence, fmt.Sprintf("%d:%s", event.Sequence, key))
	}
	t.Logf("事实序列: %s", strings.Join(sequence, " "))

	// ⑥ 收尾后服务端仍可用：证明整条回路没有把服务端弄坏。
	var final protocol.PingResult
	if err := base.control().CallInto(ctx, "daemon.ping", map[string]any{}, &final); err != nil {
		t.Fatalf("收尾 daemon.ping: %v", err)
	}
	t.Logf("收尾: 会话=%d 连接=%d", final.Sessions, final.Connections)
}

// serverPID 用 socket 路径找到服务端进程；找不到时返回空（不失败）。
func serverPID(socket string) string {
	output, err := exec.Command("pgrep", "-f", socket).Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(output))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// mustRuntime 取会话运行时；没有打开时直接失败。
func mustRuntime(t *testing.T, base *App, sessionID string) *sessionRuntime {
	t.Helper()
	runtime, err := base.openRuntime(sessionID, "session.history")
	if err != nil {
		t.Fatalf("取会话运行时: %v", err)
	}
	return runtime
}

// assertLoopFacts 断言时间线里既有工具行也有助手文本，且工具按提交执行过。
func assertLoopFacts(view SessionView, approvals int) error {
	var userRows, assistantRows, toolRows int
	toolRan, toolFailed := false, false
	for _, row := range view.Rows {
		switch row.Kind {
		case projection.RowUserMessage:
			userRows++
		case projection.RowAssistantText:
			if strings.TrimSpace(row.Text) != "" {
				assistantRows++
			}
		case projection.RowTool:
			toolRows++
			if row.Tool == nil {
				continue
			}
			if strings.Contains(row.Tool.Result, e2eToolOutput) {
				toolRan = true
			}
			if row.Tool.IsError {
				toolFailed = true
			}
		}
	}
	if userRows == 0 {
		return errText("时间线里没有用户消息行")
	}
	if toolRows == 0 {
		return errText("时间线里没有工具行")
	}
	if !toolRan {
		return errText("工具结果里没有 " + e2eToolOutput)
	}
	if toolFailed {
		return errText("工具以失败结束")
	}
	if assistantRows == 0 {
		return errText("助手文本为空")
	}
	if approvals == 0 {
		return errText("没有走审批：workspace-write 下 exec 应请求批准")
	}
	return nil
}

// describeRows 把时间线压成一行可读摘要：种类 + 状态 + 工具名，不带正文。
func describeRows(rows []projection.Row) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		part := string(row.Kind) + "/" + string(row.Status)
		if row.Tool != nil {
			part += "(" + row.Tool.Name + ")"
		}
		parts = append(parts, part)
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// errText 是断言里的纯文本错误。
type errText string

func (e errText) Error() string { return string(e) }
