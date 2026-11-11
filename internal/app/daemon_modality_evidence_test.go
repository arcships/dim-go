package app

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/settings"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestRealModalityAndShellEvidence 是真实模态与命令链路的证据用例。
//
// 只有显式设置 DIMAGENT_MODALITY_EVIDENCE=1 时才运行；服务端、模型配置与图片
// 路径都由环境变量给出。证据只打印状态、标识、字节数与页数，不打印图片内容、
// 命令正文或输出。
func TestRealModalityAndShellEvidence(t *testing.T) {
	if os.Getenv("DIMAGENT_MODALITY_EVIDENCE") != "1" {
		t.Skip("未设置 DIMAGENT_MODALITY_EVIDENCE=1，跳过真实模态证据")
	}
	serverPath := os.Getenv("DIMAGENT_APP_SERVER_BIN")
	if serverPath == "" {
		serverPath = filepath.Join(os.Getenv("HOME"), ".dimcode-alpha", "rust", "dimcode-server")
	}
	modelConfig := os.Getenv("DIMAGENT_MODEL_CONFIG")
	if modelConfig == "" {
		modelConfig = "/tmp/d12n-offline.json"
	}
	imagePath := os.Getenv("DIMAGENT_MODALITY_IMAGE")
	if imagePath == "" {
		imagePath = "/tmp/dimagent-modality-ocr.png"
	}
	if _, err := os.Stat(serverPath); err != nil {
		t.Fatalf("App Server 不可用: %v", err)
	}
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("读取模态图片: %v", err)
	}

	// 单实例端点有 Unix socket 路径上限：数据目录放在 /tmp 下并保持短路径。
	dataDir, err := os.MkdirTemp("/tmp", "d12m-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	workspaceRoot := filepath.Join(dataDir, "w")
	if err := os.MkdirAll(workspaceRoot, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}
	base, err := New(Config{
		DataDir:       dataDir,
		ServerPath:    serverPath,
		ModelConfig:   modelConfig,
		ClientName:    "dimagent-mygo-modality-evidence",
		WorkspaceRoot: workspaceRoot,
		Version:       testVersion,
		SendTimeout:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	shell, err := NewHost(base, HostOptions{DataDir: dataDir, Version: testVersion})
	if err != nil {
		t.Fatalf("接到宿主外壳: %v", err)
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelStart()
	if _, err := shell.Start(startCtx); err != nil {
		shutdownAndWait(t, shell)
		t.Fatalf("启动失败（阶段 %s）: %v", shell.State().Stage, err)
	}
	defer shutdownAndWait(t, shell)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	catalogDomain := base.Catalog()
	if _, err := catalogDomain.Refresh(ctx, catalog.Query{}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}
	entry, err := base.CreateSession(ctx, workspaceRoot, "full-access")
	if err != nil {
		t.Fatalf("新建会话: %v", err)
	}
	if _, err := base.OpenSession(ctx, entry.ID); err != nil {
		t.Fatalf("打开会话: %v", err)
	}

	// 1) 命令执行：record 模式，结果写回会话历史；只记录状态与标识。
	registered, err := base.RunShell(ctx, ShellRunRequest{SessionID: entry.ID, Command: "printf DIMAGENT_SHELL_OK"})
	if err != nil {
		t.Fatalf("登记命令执行: %v", err)
	}
	settled, err := base.Wait(ctx, registered.ID)
	if err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("真实命令执行未成功: %v %+v", err, settled)
	}
	runID := settled.InputID
	t.Logf("shell.run: run=%s status=%s", runID, settled.Status)
	if kinds, err := base.HistoryKinds(ctx, entry.ID); err == nil {
		t.Logf("shell 记录事实: %s", selectedKinds(kinds))
	}

	// 2) 模态任务：提交一次 OCR 识别，轮询到终态，再分页读完结果。
	domain := base.Settings()
	caps, err := domain.RefreshModality(ctx)
	if err != nil {
		t.Fatalf("读取模态能力: %v", err)
	}
	var category, operation string
	for _, categoryItem := range caps.Categories {
		for _, operationItem := range categoryItem.Operations {
			if categoryItem.Category == "ocr" && operationItem.Operation == "recognize" && operationItem.Available {
				category, operation = categoryItem.Category, operationItem.Operation
			}
		}
	}
	if category == "" {
		t.Fatalf("服务端没有可用的 OCR 识别能力")
	}
	task, err := domain.SubmitTask(ctx, settings.ModalityTaskSubmitRequest{
		SessionID: entry.ID,
		Category:  category,
		Operation: operation,
		Params:    []byte(`{"outputMode":"text"}`),
		Files: []settings.ModalityTaskFileInput{{
			Name:       "ocr.png",
			MediaType:  "image/png",
			DataBase64: base64.StdEncoding.EncodeToString(imageBytes),
		}},
		ClientTaskID: strings.ReplaceAll(runID, "-", "")[:32],
	})
	if err != nil {
		t.Fatalf("提交模态任务: %v", err)
	}
	states := []settings.ModalityTaskState{task.State}
	deadline := time.Now().Add(90 * time.Second)
	for {
		current, err := domain.ReadTask(ctx, entry.ID, task.ID)
		if err != nil {
			t.Fatalf("读取任务状态: %v", err)
		}
		if current.State != states[len(states)-1] {
			states = append(states, current.State)
		}
		if current.State == settings.ModalityTaskSucceeded ||
			current.State == settings.ModalityTaskFailed ||
			current.State == settings.ModalityTaskCancelled ||
			current.State == settings.ModalityTaskInterrupted {
			task = current
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("任务未在期限内结束: states=%v", states)
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Logf("modality.task: id=%s states=%v", task.ID, states)
	if task.Result != nil {
		for _, file := range task.Result.Files {
			var total int64
			var pages int
			eof := false
			var offset int64
			for !eof && pages < 50 {
				chunk, err := domain.ReadTaskResult(ctx, entry.ID, settings.ModalityTaskResultQuery{
					TaskID: task.ID, Name: file.Name, Offset: offset, Length: 64 * 1024,
				})
				if err != nil {
					t.Fatalf("读取任务结果: %v", err)
				}
				data, err := base64.StdEncoding.DecodeString(chunk.DataBase64)
				if err != nil {
					t.Fatalf("结果不是合法 base64: %v", err)
				}
				total += int64(len(data))
				offset = total
				pages++
				eof = chunk.EOF
			}
			t.Logf("modality.result: file=%s bytes=%d pages=%d eof=%t", file.Name, total, pages, eof)
		}
	}

	// 3) 取消：再提交一次并立即取消，状态应收敛。
	cancelled, err := domain.SubmitTask(ctx, settings.ModalityTaskSubmitRequest{
		SessionID: entry.ID,
		Category:  category,
		Operation: operation,
		Params:    []byte(`{"outputMode":"text"}`),
		Files: []settings.ModalityTaskFileInput{{
			Name:       "ocr.png",
			MediaType:  "image/png",
			DataBase64: base64.StdEncoding.EncodeToString(imageBytes),
		}},
	})
	if err != nil {
		t.Fatalf("提交取消用任务: %v", err)
	}
	final, err := domain.CancelTask(ctx, entry.ID, cancelled.ID)
	if err != nil {
		t.Logf("modality.cancel: 取消失败（按 blocked 记录）: %v", err)
	} else {
		t.Logf("modality.cancel: id=%s state=%s", final.ID, final.State)
	}

	// 4) 实时传输：adapter 不可用（服务端要求外部传输）时按能力缺失/blocked 记录。
	connection, err := domain.OpenRealtime(ctx, settings.ModalityRealtimeOpenRequest{
		SessionID: entry.ID,
		Operation: "transcribe",
	})
	if err != nil {
		t.Logf("modality.realtime: blocked（打开失败）: %v", err)
	} else {
		t.Logf("modality.realtime: connection=%s", connection.ConnectionID)
		if closeErr := domain.CloseRealtime(ctx, entry.ID, connection.ConnectionID); closeErr != nil {
			t.Logf("modality.realtime: 关闭失败: %v", closeErr)
		}
	}

	// 确认证据结束时写闸门仍在正常状态。
	if err := base.Writable(); err != nil {
		t.Fatalf("证据结束时写闸门异常: %v", err)
	}
}

// selectedKinds 只挑与命令记录相关的事实类型，避免打印无关内容。
func selectedKinds(kinds map[string]int) string {
	wanted := []string{"input/message", "input/applied", "step/tool-use", "turn/start", "turn/end"}
	parts := make([]string, 0, len(wanted))
	for _, kind := range wanted {
		if kinds[kind] > 0 {
			parts = append(parts, kind+"="+itoa(kinds[kind]))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " ")
}
