// Command dimagent-conncheck 在真实 App Server 上跑一次握手与打开会话，
// 作为连接领域的验收证据。它不是产品入口，不做投影，也不保存任何事实。
//
// 用法（mock 模型、临时数据目录）：
//
//	dimagent-conncheck -data-dir /tmp/d1-home \
//	  -server /path/to/dimcode-server -server-arg app-server -mock
//
// 用已配置模型启动（产品/configured profile）可加 `-model-config <文件>`；
// `-probe` 会逐项读取 v3 wire：方法清单、四类未完成事项、提交核对与附件写入。
//
// 期望输出若干 JSON 行：hello、session、session.opened、probe.*、shutdown。
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

type repeatedFlag []string

func (r *repeatedFlag) String() string { return strings.Join(*r, " ") }

func (r *repeatedFlag) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "conncheck: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var serverArgs repeatedFlag
	dataDir := flag.String("data-dir", "", "客户端数据目录（必须显式给出，建议临时目录）")
	serverPath := flag.String("server", os.Getenv("DIMCODE_APP_SERVER_BIN"), "App Server 可执行文件路径")
	sessionID := flag.String("session", "", "打开已有会话；省略则新建一个")
	mockModel := flag.Bool("mock", false, "用 mock 模型启动服务端（验收用）")
	modelConfig := flag.String("model-config", "", "用模型配置启动服务端（configured profile）")
	probe := flag.Bool("probe", false, "逐项读取 v3 wire：方法清单、四类未完成事项、提交核对、附件写入")
	keepDaemon := flag.Bool("keep", false, "不回收自己拉起的服务端")
	noticeSeconds := flag.Int("notice-seconds", 1, "打开会话后等待通知的秒数")
	flag.Var(&serverArgs, "server-arg", "追加给服务端的参数，可重复（例如 dimcode-server 需要 app-server）")
	flag.Parse()

	if *dataDir == "" {
		return errors.New("必须给出 -data-dir")
	}
	if *serverPath == "" {
		return errors.New("必须给出 -server 或 DIMCODE_APP_SERVER_BIN")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	supervisor, err := daemon.New(daemon.Config{
		DataDir:     *dataDir,
		ServerPath:  *serverPath,
		ServerArgs:  serverArgs,
		MockModel:   *mockModel,
		ModelConfig: *modelConfig,
	})
	if err != nil {
		return fmt.Errorf("创建监督者: %w", err)
	}

	conn, err := supervisor.Connect(ctx)
	if err != nil {
		return fmt.Errorf("连接服务端: %w", err)
	}
	emit("hello", map[string]any{
		"endpoint":        conn.Endpoint.String(),
		"instanceId":      conn.Hello.InstanceID,
		"serverVersion":   conn.Hello.ServerVersion,
		"protocolVersion": conn.Hello.ProtocolVersion,
		"profile":         conn.Hello.DaemonProfile.Kind,
		"owned":           conn.Owned,
		"features":        conn.Hello.Features,
	})

	id := *sessionID
	if id == "" {
		workDir := filepath.Join(*dataDir, "workspace")
		if mkErr := os.MkdirAll(workDir, 0o700); mkErr != nil {
			return fmt.Errorf("创建工作目录: %w", mkErr)
		}
		var created protocol.SessionCreateResult
		if err := conn.Client.CallInto(ctx, "session.create", map[string]any{
			"cwd":   workDir,
			"tools": []string{},
		}, &created); err != nil {
			return fmt.Errorf("创建会话: %w", err)
		}
		id = created.SessionID
		emit("session.created", map[string]any{"sessionId": id, "cwd": workDir})
	}

	manager, err := session.NewManager(session.Config{
		Opener:            supervisor,
		ReconnectAttempts: 3,
		ReconnectDelay:    200 * time.Millisecond,
	})
	if err != nil {
		return fmt.Errorf("创建会话流管理器: %w", err)
	}
	defer manager.Close()

	var events int64
	var resyncs int64
	var fatal atomic.Value
	snapshot, lease, err := manager.Acquire(ctx, id, session.Listener{
		OnEvent: func(event session.Event) {
			atomic.AddInt64(&events, 1)
			if event.Method != "session.event" {
				emit("notification", map[string]any{"method": event.Method})
			}
		},
		OnResync: func(resync session.Resync) {
			atomic.AddInt64(&resyncs, 1)
		},
		OnFatal: func(err error) {
			fatal.Store(err.Error())
		},
	})
	if err != nil {
		return fmt.Errorf("打开会话: %w", err)
	}
	emit("session.opened", map[string]any{
		"sessionId":      snapshot.SessionID,
		"head":           snapshot.Head,
		"historyEvents":  len(snapshot.History.Events),
		"historyThrough": snapshot.History.Through,
		"historyDone":    snapshot.History.Done,
		"streamFeatures": snapshot.Features,
	})

	if *probe {
		probeWire(ctx, lease.Subscription())
	}

	time.Sleep(time.Duration(*noticeSeconds) * time.Second)
	lease.Release()
	emit("session.released", map[string]any{
		"events":  atomic.LoadInt64(&events),
		"resyncs": atomic.LoadInt64(&resyncs),
		"fatal":   fatal.Load(),
	})

	if *keepDaemon {
		supervisor.Close(errors.New("conncheck done"))
		emit("shutdown", map[string]any{"skipped": true})
		return nil
	}
	result, err := supervisor.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("回收自启服务端: %w", err)
	}
	emit("shutdown", map[string]any{
		"stopped": result.Stopped,
		"reason":  result.Reason,
	})
	return nil
}

// probeWire 逐项读取连接领域冻结的 v3 wire 形状。单项失败不中断其余项目。
func probeWire(ctx context.Context, sub *session.Subscription) {
	methods, err := sub.Methods(ctx)
	if err != nil {
		emit("probe.methods", map[string]any{"error": err.Error()})
	} else {
		names := make([]string, 0, len(methods.Methods))
		for _, method := range methods.Methods {
			names = append(names, method.Name)
		}
		emit("probe.methods", map[string]any{"methods": names})
	}

	branchID := "main"
	lookup, err := sub.InputLookup(ctx, protocol.InputLookupParams{
		BranchID:     branchID,
		SubmissionID: randomUUID(),
	})
	if err != nil {
		emit("probe.input_lookup.unknown", map[string]any{"error": err.Error()})
	} else {
		emit("probe.input_lookup.unknown", map[string]any{"status": lookup.Status})
	}

	submissionID := randomUUID()
	if _, err := sub.Call(ctx, "session.submit", map[string]any{
		"sessionId":    sub.SessionID(),
		"branchId":     branchID,
		"submissionId": submissionID,
		"text":         "conncheck lookup probe",
	}); err != nil {
		emit("probe.submit", map[string]any{"error": err.Error()})
	} else {
		accepted, err := sub.InputLookup(ctx, protocol.InputLookupParams{
			BranchID:     branchID,
			SubmissionID: submissionID,
		})
		if err != nil {
			emit("probe.input_lookup.accepted", map[string]any{"error": err.Error()})
		} else {
			fields := map[string]any{"status": accepted.Status}
			if accepted.Event != nil {
				fields["sequence"] = accepted.Event.Sequence
				fields["eventId"] = accepted.Event.ID
			}
			emit("probe.input_lookup.accepted", fields)
		}
	}

	query := protocol.PendingQuery{BranchID: branchID}
	if inputs, err := sub.PendingInputs(ctx, query); err != nil {
		emit("probe.input.pending", map[string]any{"error": err.Error()})
	} else {
		emit("probe.input.pending", map[string]any{"through": inputs.Through, "inputs": inputs.Inputs, "next": inputs.Next})
	}
	if tasks, err := sub.PendingExecTasks(ctx, query); err != nil {
		emit("probe.exec.pending", map[string]any{"error": err.Error()})
	} else {
		emit("probe.exec.pending", map[string]any{"through": tasks.Through, "tasks": tasks.Tasks, "next": tasks.Next})
	}
	if questions, err := sub.PendingQuestions(ctx, query); err != nil {
		emit("probe.question.pending", map[string]any{"error": err.Error()})
	} else {
		emit("probe.question.pending", map[string]any{"through": questions.Through, "batches": questions.Batches, "next": questions.Next})
	}
	if collaboration, err := sub.PendingCollaboration(ctx, query); err != nil {
		emit("probe.collaboration.pending", map[string]any{"error": err.Error()})
	} else {
		emit("probe.collaboration.pending", map[string]any{"through": collaboration.Through, "requests": collaboration.Requests, "next": collaboration.Next})
	}

	put, err := sub.PutAttachment(ctx, protocol.AttachmentPutParams{Name: "conncheck.txt", Text: "conncheck probe"})
	if err != nil {
		emit("probe.attachment.put", map[string]any{"error": err.Error()})
	} else {
		emit("probe.attachment.put", map[string]any{
			"attachmentId": put.AttachmentID,
			"path":         put.Path,
			"bytes":        put.Bytes,
		})
	}
}

func randomUUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	text := hex.EncodeToString(buf)
	return text[0:8] + "-" + text[8:12] + "-" + text[12:16] + "-" + text[16:20] + "-" + text[20:32]
}

func emit(name string, fields map[string]any) {
	fields["event"] = name
	raw, err := json.Marshal(fields)
	if err != nil {
		fmt.Printf("{\"event\":%q,\"error\":%q}\n", name, err.Error())
		return
	}
	fmt.Println(string(raw))
}
