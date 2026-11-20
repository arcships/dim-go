// 临时诊断程序（不进仓库）。复制到 cmd/ 下再 build。
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/projection"
)

type repeatedFlag []string

func (r *repeatedFlag) String() string { return fmt.Sprint(*r) }

func (r *repeatedFlag) Set(v string) error { *r = append(*r, v); return nil }

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "tmpqueuediag: %v\n", err)
		os.Exit(1)
	}
}

func emit(label string, value any) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Printf("%s: %v\n", label, err)
		return
	}
	fmt.Printf("== %s ==\n%s\n", label, raw)
}

func run() error {
	var serverArgs repeatedFlag
	dataDir := flag.String("data-dir", "", "客户端数据目录")
	serverPath := flag.String("server", os.Getenv("DIMCODE_APP_SERVER_BIN"), "App Server 路径")
	sessionID := flag.String("session", "", "已有会话标识")
	mock := flag.Bool("mock", false, "mock 模型")
	modelConfig := flag.String("model-config", "", "模型配置文件")
	submits := flag.Int("submits", 0, "提交几条输入")
	text := flag.String("text", "排队探针输入", "提交正文")
	gap := flag.Duration("gap", 700*time.Millisecond, "提交间隔")
	wait := flag.Duration("wait", 6*time.Second, "提交后等待")
	branchID := flag.String("branch", "main", "分支标识")
	cycles := flag.Int("cycles", 1, "接入/释放的轮数（模拟重连）")
	flag.Var(&serverArgs, "server-arg", "追加服务端参数")
	flag.Parse()

	if *dataDir == "" || *serverPath == "" {
		return errors.New("需要 -data-dir 与 -server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	supervisor, err := daemon.New(daemon.Config{
		DataDir: *dataDir, ServerPath: *serverPath, ServerArgs: serverArgs,
		MockModel: *mock, ModelConfig: *modelConfig,
	})
	if err != nil {
		return fmt.Errorf("监督者: %w", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		return fmt.Errorf("连接: %w", err)
	}
	fmt.Printf("server=%s profile=%s\n", conn.Hello.ServerVersion, conn.Hello.DaemonProfile.Kind)

	id := *sessionID
	if id == "" {
		workDir := filepath.Join(*dataDir, "workspace")
		if err := os.MkdirAll(workDir, 0o700); err != nil {
			return err
		}
		var created protocol.SessionCreateResult
		if err := conn.Client.CallInto(ctx, "session.create", map[string]any{
			"cwd": workDir, "tools": []string{},
		}, &created); err != nil {
			return fmt.Errorf("创建会话: %w", err)
		}
		id = created.SessionID
	}
	fmt.Printf("session=%s\n", id)

	manager, err := session.NewManager(session.Config{
		Opener: supervisor, ReconnectAttempts: 2, ReconnectDelay: 200 * time.Millisecond,
	})
	if err != nil {
		return err
	}
	defer manager.Close()

	listener := session.Listener{OnEvent: func(session.Event) {}}
	for i := 1; i <= *cycles; i++ {
		_, lease, err := manager.Acquire(ctx, id, listener)
		if err != nil {
			return fmt.Errorf("第 %d 次接入: %w", i, err)
		}
		query := protocol.PendingQuery{SessionID: id, BranchID: *branchID}
		raw, callErr := lease.Subscription().Call(ctx, protocol.MethodSessionInputPending, query)
		if callErr != nil {
			fmt.Printf("cycle %d pending error: %v\n", i, callErr)
		} else {
			var page protocol.SessionInputPendingResult
			_ = json.Unmarshal(raw, &page)
			fmt.Printf("cycle %d attach ok, pending=%d raw=%s\n", i, len(page.Inputs), string(raw))
		}
		time.Sleep(300 * time.Millisecond)
		lease.Release()
		time.Sleep(200 * time.Millisecond)
	}

	_, lease, err := manager.Acquire(ctx, id, listener)
	if err != nil {
		return fmt.Errorf("最终打开会话: %w", err)
	}
	sub := lease.Subscription()

	for i := 0; i < *submits; i++ {
		_, err := sub.Call(ctx, "session.submit", map[string]any{
			"sessionId": id, "branchId": *branchID, "submissionId": uuid(),
			"text": fmt.Sprintf("%s #%d", *text, i+1),
		})
		fmt.Printf("submit %d err=%v\n", i+1, err)
		time.Sleep(*gap)
	}
	time.Sleep(*wait)

	query := protocol.PendingQuery{SessionID: id, BranchID: *branchID}
	raw, rawErr := sub.Call(ctx, protocol.MethodSessionInputPending, query)
	if rawErr != nil {
		fmt.Printf("pending call error: %v\n", rawErr)
	} else {
		fmt.Printf("== raw session.input.pending ==\n%s\n", string(raw))
	}

	limit := 500
	params := protocol.SessionHistoryParams{SessionID: id, BranchID: *branchID, Limit: &limit}
	var history protocol.SessionHistoryResult
	historyRaw, err := sub.Call(ctx, "session.history", params)
	if err != nil {
		fmt.Printf("history error: %v\n", err)
	} else if err := json.Unmarshal(historyRaw, &history); err != nil {
		fmt.Printf("history decode: %v\n", err)
	}
	inputFacts := []map[string]any{}
	for _, event := range history.Events {
		kind := event.Type + "/" + event.Name
		if len(kind) >= 6 && kind[:6] == "input/" {
			inputFacts = append(inputFacts, map[string]any{
				"type": kind, "id": event.ID, "sequence": event.Sequence,
				"data": json.RawMessage(event.Data),
			})
		}
	}
	emit("history input/* facts", inputFacts)

	folder, err := projection.New(projection.Config{SessionID: id})
	if err != nil {
		return err
	}
	folder.ApplyHistory(*branchID, history)
	if rawErr == nil {
		var page protocol.SessionInputPendingResult
		if err := json.Unmarshal(raw, &page); err == nil {
			items := make([]projection.PendingInput, 0, len(page.Inputs))
			for _, input := range page.Inputs {
				items = append(items, projection.PendingInput{
					InputID: input.InputID, Sequence: input.Sequence, Mode: input.Mode,
					Text: pendingText(input), Attachments: len(input.Attachments),
					Applied: input.Applied,
				})
			}
			folder.SeedPendingInputs(*branchID, page.Through, items)
		}
	}
	view := folder.View()
	emit("projection pending inputs", view.Pending.Inputs)
	emit("projection head", map[string]any{"head": folder.Head(*branchID), "through": history.Through})

	lease.Release()
	_, err = supervisor.Shutdown(ctx)
	return err
}

func uuid() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// pendingText 复制 app.pendingInputText 的取正文规则。
func pendingText(input protocol.PendingInput) string {
	if input.User != nil {
		for _, block := range input.User.Content {
			var plain string
			if err := json.Unmarshal(block, &plain); err == nil {
				if plain != "" {
					return plain
				}
				continue
			}
			var parsed struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(block, &parsed); err != nil {
				continue
			}
			switch parsed.Type {
			case "", "text", "input_text":
				if parsed.Text != "" {
					return parsed.Text
				}
			}
		}
	}
	if input.Message != "" {
		return input.Message
	}
	return input.Preview
}
