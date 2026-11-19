package uiwire

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/stubmodel"
	"github.com/arcships/dimagent-mygo/internal/ui"

	gui "github.com/egoist/mygo/ui"
)

// 界面层真实回路的入口参数：与 internal/app 的证据用例同一组环境变量。
//
//	DIM_APP_SERVER_BIN  真实 App Server 可执行文件（必需）
//	DIM_E2E_UI_HOME     本次运行的客户端数据目录（可选，默认 /tmp/e2e-ui-home）
const (
	envE2EUiHome     = "DIM_E2E_UI_HOME"
	defaultE2EUiHome = "/tmp/e2e-ui-home"
	uiE2EPrompt      = "执行 echo e2e-ok 并把输出报给我"
	uiE2ETimeout     = 3 * time.Minute
	uiE2EToolLine    = "执行 echo e2e-ok"
)

// TestLiveLoopRenders 用真实 App Server + 真实 Source 走一遍界面：
// 输入一条会触发工具的消息 → 发送 → 审批卡出现并点「仅这次」→ 工具行与助手
// 正文上屏 → 输入框清空、发送槽回到「发送」。
func TestLiveLoopRenders(t *testing.T) {
	if testing.Short() {
		t.Skip("短模式跳过界面真实回路（需要真实 App Server 进程）")
	}
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("缺少 DIM_APP_SERVER_BIN（真实 App Server 可执行文件）；跳过界面回路")
	}
	home := os.Getenv(envE2EUiHome)
	if home == "" {
		home = defaultE2EUiHome
	}
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

	base, err := app.New(app.Config{
		DataDir:        home,
		ServerPath:     serverPath,
		ModelConfig:    modelConfig,
		ClientName:     "dimagent-mygo-ui-e2e",
		WorkspaceRoot:  work,
		Version:        "0.1.0",
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

	// ① 连接与目录：界面的 Source 就是这一层适配器。
	w := New(base)
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接: %v", err)
	}
	if conn := w.Connection(); conn.Phase != "connected" {
		t.Fatalf("连接阶段=%q（%s）", conn.Phase, conn.Reason)
	}
	if err := w.CreateSession(ctx, work); err != nil {
		t.Fatalf("新建会话: %v", err)
	}
	names := w.SessionNames()
	if len(names) == 0 {
		t.Fatal("目录里没有会话")
	}
	sessionID := names[0]
	if err := w.OpenSession(ctx, sessionID); err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	snap, ok := w.Session()
	if !ok {
		t.Fatal("打开后没有会话快照")
	}
	t.Logf("会话: id=%s 行=%d 目录=%q 阶段=%s", snap.ID, len(snap.Rows), snap.Cwd,
		w.Connection().Phase)

	// ② 界面：Source + Intents 都接真实实现，渲染走 mygo 的 headless Tester。
	screen := ui.NewApp(ui.Options{
		State:   ui.StateDefault,
		Source:  w,
		Intents: w,
		Changed: func() {},
		LogPath: filepath.Join(home, "logs", "dimagent.log"),
	})
	tt := gui.NewTester(screen.View, ui.WindowWidth, ui.WindowHeight)
	tt.Frame()

	// 输入一条会触发工具的消息。
	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点不到输入框: %v", err)
	}
	tt.Type(uiE2EPrompt)
	tt.Frame()
	if info, ok := tt.Access("消息"); !ok || !strings.Contains(info.Value, uiE2EPrompt) {
		t.Fatalf("草稿没有进输入框: %+v ok=%t", info, ok)
	}
	if err := tt.Click("发送"); err != nil {
		t.Fatalf("点不到发送: %v", err)
	}

	// ③ 等审批卡出现并点「仅这次」；再等工具行与正文上屏。
	approved := false
	approvalText := ""
	deadline := time.Now().Add(uiE2ETimeout)
	shapes := make([]string, 0, 16)
	lastShape := ""
	for {
		if time.Now().After(deadline) {
			t.Fatalf("超时：approved=%t 替身请求=%d 界面文本=%s",
				approved, stub.Requests.Load(), strings.Join(tt.Texts(), " | "))
		}
		tt.Frame()
		if shape := composerShape(tt); shape != lastShape {
			shapes = append(shapes, shape)
			lastShape = shape
		}
		if !approved {
			if _, ok := tt.Find("仅这次"); ok {
				if ap, ok := w.Approval(); ok {
					approvalText = ap.Command
					if !tt.HasText(ap.Command) {
						t.Errorf("审批卡没有画上服务端给的动作原文: %q", ap.Command)
					}
				}
				if !tt.HasText("DimAgent 想要：执行") {
					t.Errorf("审批卡标题没有上屏: %s", compactTexts(tt.Texts()))
				}
				t.Logf("审批卡: 标题=DimAgent 想要：执行 动作=%q 替身请求=%d",
					approvalText, stub.Requests.Load())
				if err := tt.Click("仅这次"); err != nil {
					t.Fatalf("点不到「仅这次」: %v", err)
				}
				approved = true
				tt.Frame()
			}
		}
		if approved && tt.HasText(uiE2EToolLine) && tt.HasText(stubmodel.Answer) {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("等待界面回流超时: %v", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}

	// ④ 断言：输入框清空、发送槽回到「发送」、工具行与正文都在、这一轮已结束。
	send, ok := tt.Access("发送")
	if !ok {
		t.Fatalf("这一轮结束后发送槽没有回到「发送」: %s", strings.Join(tt.Texts(), " | "))
	}
	if !send.Disabled {
		t.Fatalf("草稿没有清空：发送槽仍可点（disabled=%t）", send.Disabled)
	}
	if input, ok := tt.Access("消息"); !ok || strings.TrimSpace(input.Value) != "" {
		t.Fatalf("输入框没有清空: value=%q ok=%t", input.Value, ok)
	}
	if txt := strings.Join(tt.Texts(), "\n"); !strings.Contains(txt, "执行") {
		t.Fatalf("工具行没有上屏:\n%s", txt)
	}
	final, ok := w.Session()
	if !ok {
		t.Fatal("结束时没有会话快照")
	}
	if !final.LastTurnEnded || final.LastTurnOutcome != "completed" {
		t.Fatalf("这一轮没有以 completed 结束: ended=%t outcome=%q",
			final.LastTurnEnded, final.LastTurnOutcome)
	}
	t.Logf("断言: 输入框已清空=true 发送槽=发送(disabled)=%t 工具行=%q 正文=%q 轮=%d/%s 替身请求=%d",
		send.Disabled, uiE2EToolLine, stubmodel.Answer, final.LastTurnIndex,
		final.LastTurnOutcome, stub.Requests.Load())
	t.Logf("界面形状序列: %s", strings.Join(shapes, " -> "))
	t.Logf("最终文本: %s", compactTexts(tt.Texts()))
}

// composerShape 是输入位的形状：审批卡 / 输入卡 / 发送槽是否可用。
func composerShape(tt *gui.Tester) string {
	if _, ok := tt.Find("审批卡"); ok {
		return "审批卡"
	}
	if info, ok := tt.Access("发送"); ok {
		if info.Disabled {
			return "发送(禁用)"
		}
		return "发送"
	}
	if _, ok := tt.Access("停止"); ok {
		return "停止"
	}
	if _, ok := tt.Find("输入卡"); ok {
		return "输入卡"
	}
	return "其它"
}

// compactTexts 把整屏文本压成一行，便于在证据里对照。
func compactTexts(texts []string) string {
	trimmed := make([]string, 0, len(texts))
	for _, text := range texts {
		text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
		if text != "" {
			trimmed = append(trimmed, text)
		}
	}
	return strings.Join(trimmed, " | ")
}
