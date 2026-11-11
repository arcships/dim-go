package catalog

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/projection"
)

// TestRealDaemonExportTurns 在真实 App Server 上跑两轮对话，再用会话投影的按轮
// 来源导出，核对文件轮次等于真实轮次、两轮内容都在、每页一轮时确实翻了多页。
//
// 用法：
//
//	DIM_APP_SERVER_BIN=<app-server-bin> \
//	  go test -run TestRealDaemonExportTurns -v ./internal/catalog/
func TestRealDaemonExportTurns(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实 daemon 证据 blocked（缺 App Server 可执行文件）")
	}
	var serverArgs []string
	if args := os.Getenv("DIM_APP_SERVER_ARGS"); args != "" {
		serverArgs = append(serverArgs, args)
	}

	dataDir := evidenceDataDir(t)
	workDir := filepath.Join(dataDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	supervisor, err := daemon.New(evidenceMockDaemonConfig(dataDir, serverPath, serverArgs))
	if err != nil {
		t.Fatalf("创建监督者: %v", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	t.Cleanup(func() { supervisor.Close(nil) })
	t.Logf("服务端 %s（protocol %d，profile %s，owned=%v）",
		conn.Hello.ServerVersion, conn.Hello.ProtocolVersion, conn.Hello.DaemonProfile.Kind, conn.Owned)

	if !conn.HasFeature("session.history.latestTurn") {
		cleanupDaemon(t, supervisor)
		t.Skipf("服务端没有 session.history.latestTurn：真实导出证据 blocked")
	}

	harness := &appServerHarness{client: conn.Client}
	writes := newWriteManager(t, harness)
	creator := newTestCatalog(t, Config{Reader: harness, Writes: writes})
	created, err := creator.Create(ctx, CreateRequest{Cwd: workDir})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("新建会话: %v", err)
	}

	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("创建会话流管理器: %v", err)
	}
	t.Cleanup(manager.Close)

	turnEnded := make(chan struct{}, 4)
	snapshot, lease, err := manager.Acquire(ctx, created.ID, session.Listener{
		OnEvent: func(event session.Event) {
			if isTurnEndNotice(event) {
				select {
				case turnEnded <- struct{}{}:
				default:
				}
			}
		},
	})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("打开会话: %v", err)
	}
	t.Cleanup(lease.Release)
	t.Logf("首屏: head=%d events=%d", snapshot.Head, len(snapshot.History.Events))

	for turn := 1; turn <= 2; turn++ {
		submitOneTurn(t, ctx, conn.Client, created.ID, fmt.Sprintf("第 %d 轮：用一句话说明当前目录", turn), turnEnded)
	}
	// 等事实落定再读历史；mock 模型不会拖长，给一小段余量。
	time.Sleep(300 * time.Millisecond)

	source, err := projection.NewTurnHistory(created.ID, lease.Subscription())
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("创建按轮历史来源: %v", err)
	}
	exporter := newTestCatalog(t, Config{Reader: harness, Writes: writes, History: source})

	path := filepath.Join(dataDir, "export.json")
	result, err := exporter.Export(ctx, ExportRequest{
		SessionID: created.ID,
		Path:      path,
		PageLimit: 1, // 每页一轮，强制走多页
	})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("导出: %v", err)
	}
	if result.Turns != 2 {
		cleanupDaemon(t, supervisor)
		t.Fatalf("导出轮次不符：got %d，want 2", result.Turns)
	}
	if result.Pages != 2 {
		cleanupDaemon(t, supervisor)
		t.Fatalf("每页一轮时页数不符：got %d，want 2", result.Pages)
	}
	t.Logf("导出：%d 轮，%d 页，%d 字节", result.Turns, result.Pages, result.Bytes)

	raw, err := os.ReadFile(path)
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("读取导出文件: %v", err)
	}
	var document struct {
		Format    string `json:"format"`
		TurnCount int    `json:"turnCount"`
		Turns     []struct {
			Index int64 `json:"index"`
			Data  struct {
				Rows []struct {
					Kind string `json:"kind"`
					Text string `json:"text"`
				} `json:"rows"`
			} `json:"data"`
		} `json:"turns"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("导出文件不是合法 JSON: %v", err)
	}
	if document.Format != ExportFormat || document.TurnCount != 2 || len(document.Turns) != 2 {
		cleanupDaemon(t, supervisor)
		t.Fatalf("文件头不符：format=%q，turnCount=%d，turns=%d", document.Format, document.TurnCount, len(document.Turns))
	}
	texts := make(map[string]bool)
	for _, turn := range document.Turns {
		if turn.Index <= 0 {
			cleanupDaemon(t, supervisor)
			t.Fatalf("轮次序号非正：%d", turn.Index)
		}
		for _, row := range turn.Data.Rows {
			if row.Kind == "user_message" {
				texts[row.Text] = true
			}
		}
	}
	for turn := 1; turn <= 2; turn++ {
		want := fmt.Sprintf("第 %d 轮：用一句话说明当前目录", turn)
		if !texts[want] {
			cleanupDaemon(t, supervisor)
			t.Fatalf("导出文件缺少第 %d 轮的输入：%v", turn, texts)
		}
	}
	t.Logf("导出文件轮次与内容一致：%d 轮", document.TurnCount)

	lease.Release()
	cleanupDaemon(t, supervisor)
}

// submitOneTurn 提交一轮并等 turn/end。
func submitOneTurn(
	t *testing.T,
	ctx context.Context,
	client *protocol.Client,
	sessionID, text string,
	turnEnded <-chan struct{},
) {
	t.Helper()
	if err := client.CallInto(ctx, "session.submit", map[string]any{
		"sessionId":    sessionID,
		"branchId":     "main",
		"submissionId": newUUID(t),
		"mode":         "queue",
		"text":         text,
	}, &struct {
		Event protocol.RawEvent `json:"event"`
	}{}); err != nil {
		t.Fatalf("提交一轮: %v", err)
	}
	select {
	case <-turnEnded:
	case <-ctx.Done():
		t.Fatalf("等待 turn/end 超时")
	}
}

// isTurnEndNotice 判断通知是不是一轮结束。
func isTurnEndNotice(event session.Event) bool {
	if event.Method != "session.event" {
		return false
	}
	var params protocol.SessionEventParams
	if err := json.Unmarshal(event.Params, &params); err != nil {
		return false
	}
	return params.Event.Type == "turn" && params.Event.Name == "end"
}

// newUUID 造一个 v4 UUID：服务端要求提交标识是 UUID。
func newUUID(t *testing.T) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatalf("生成提交标识: %v", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}
