package interaction_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// 真实模型审批证据。默认跳过；设置 DIM_REAL_MODEL_FIXTURE 指向 /tmp 夹具后运行：
//
//	DIM_REAL_MODEL_FIXTURE=/tmp/dimagent-real-model \
//	  go test -run TestRealModelApproval -v -timeout 12m ./internal/interaction/
//
// 夹具只被复制到临时目录，不修改来源，也不提交任何夹具内容。输出只含会话标识、
// 审批请求标识、决定与布尔结果，不打印会话正文或凭据。
const realApprovalPrompt = "Use the shell tool to run exactly: printf DIMAGENT_APPROVAL_OK. You must call the shell tool; do not answer from memory. Then report the tool output."

const realApprovalMarker = "DIMAGENT_APPROVAL_OK"

// TestRealModelApproval 验证真实模型下的审批链路：同意后执行、拒绝后不执行、
// 长期允许先写命令规则再应答，且同一命令第二次不再询问。
func TestRealModelApproval(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("设置 DIM_REAL_MODEL_FIXTURE 后运行真实模型审批证据")
	}
	if _, err := os.Stat(filepath.Join(fixture, "bin", "dimcode-server")); err != nil {
		t.Fatalf("夹具缺少 daemon: %v", err)
	}
	home := prepareApprovalHome(t, fixture)
	base := startRealApp(t, home, fixture)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	sessionID := createRealSession(t, ctx, base)
	t.Cleanup(func() {
		if err := base.Close(context.Background()); err != nil {
			t.Logf("关闭集成层: %v", err)
		}
	})

	approveScenario(t, ctx, base, sessionID)
	if os.Getenv("DIM_APPROVAL_ONLY") == "approve" {
		return
	}
	denyScenario(t, ctx, base, sessionID)
	longTermScenario(t, ctx, base, sessionID)
}

func approveScenario(t *testing.T, ctx context.Context, base *app.App, sessionID string) {
	t.Helper()
	if err := setWorkspaceWrite(ctx, base, sessionID); err != nil {
		t.Fatalf("设置 workspace-write: %v", err)
	}
	submitRealPrompt(t, ctx, base, sessionID)
	approval := waitApproval(t, ctx, base, sessionID)
	t.Logf("approve: approval=%s toolCommand=%t pendingApprovals=%d",
		approval.RequestID, strings.HasPrefix(approval.Action, "exec"), approval.Position+1)

	outcome, err := base.Respond(ctx, sessionID, approval.RequestID, interaction.Response{
		Decision: interaction.DecisionApprove,
	})
	if err != nil {
		t.Fatalf("同意审批: %v", err)
	}
	if outcome.Response == nil || outcome.Response.Status != write.StatusSucceeded {
		t.Fatalf("同意应答应写入成功: %+v", outcome.Response)
	}
	waitToolResult(t, ctx, base, sessionID, true)
	t.Logf("approve: decided=allow toolResult=true")
}

func denyScenario(t *testing.T, ctx context.Context, base *app.App, sessionID string) {
	t.Helper()
	submitRealPrompt(t, ctx, base, sessionID)
	approval := waitApproval(t, ctx, base, sessionID)
	t.Logf("deny: approval=%s pendingApprovals=%d", approval.RequestID, approval.Position+1)

	outcome, err := base.Respond(ctx, sessionID, approval.RequestID, interaction.Response{
		Decision: interaction.DecisionReject,
	})
	if err != nil {
		t.Fatalf("拒绝审批: %v", err)
	}
	if outcome.Response == nil || outcome.Response.Status != write.StatusSucceeded {
		t.Fatalf("拒绝应答应写入成功: %+v", outcome.Response)
	}
	// 拒绝后模型可能再要一次；继续拒绝，直到工具以错误结束或本轮结束。
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		view, _ := base.View(sessionID)
		if ok, failed := toolOutcome(view); failed {
			if ok {
				t.Fatal("拒绝后仍出现了成功执行结果")
			}
			t.Logf("deny: decided=deny toolResult=false")
			return
		}
		if !view.HasCurrent && turnEnded(view) {
			t.Logf("deny: decided=deny toolResult=false")
			return
		}
		if view.HasCurrent && view.Current.Kind == interaction.PendingApproval {
			_, _ = base.Respond(ctx, sessionID, view.Current.RequestID, interaction.Response{
				Decision: interaction.DecisionReject,
			})
		}
		select {
		case <-ctx.Done():
			t.Fatalf("等待拒绝结果超时: %v", ctx.Err())
		case <-time.After(80 * time.Millisecond):
		}
	}
	t.Fatal("拒绝后没有等到结束")
}

func longTermScenario(t *testing.T, ctx context.Context, base *app.App, sessionID string) {
	t.Helper()
	submitRealPrompt(t, ctx, base, sessionID)
	approval := waitApproval(t, ctx, base, sessionID)
	if !approval.AllowLongTerm {
		t.Fatalf("简单 exec 命令应可长期允许: action=%q", approval.Action)
	}
	t.Logf("long-term: approval=%s allowLongTerm=true", approval.RequestID)

	outcome, err := base.Respond(ctx, sessionID, approval.RequestID, interaction.Response{
		Decision: interaction.DecisionAllowLongTerm,
	})
	if err != nil {
		t.Fatalf("长期允许: %v", err)
	}
	if outcome.LongTerm == nil || outcome.LongTerm.Status != write.StatusSucceeded {
		t.Fatalf("长期规则应写入成功: %+v", outcome.LongTerm)
	}
	if outcome.Response == nil || outcome.Response.Status != write.StatusSucceeded {
		t.Fatalf("长期允许后的应答应写入成功: %+v", outcome.Response)
	}
	waitToolResult(t, ctx, base, sessionID, true)
	t.Logf("long-term: ruleWritten=true firstToolResult=true")

	// 同一命令再来一次：规则应让服务端不再发审批请求。
	submitRealPrompt(t, ctx, base, sessionID)
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		view, _ := base.View(sessionID)
		if view.HasCurrent && view.Current.Kind == interaction.PendingApproval {
			// 规则没有生效；拒绝掉以免留下悬空审批，然后失败。
			_, _ = base.Respond(ctx, sessionID, view.Current.RequestID, interaction.Response{
				Decision: interaction.DecisionReject,
			})
			t.Fatalf("长期允许后同一命令仍触发审批: %s", view.Current.RequestID)
		}
		if ok, _ := toolOutcome(view); ok {
			t.Logf("long-term: secondApproval=false secondToolResult=true")
			return
		}
		if !view.HasCurrent && turnEnded(view) {
			t.Fatal("第二次提交结束但没有成功执行结果")
		}
		select {
		case <-ctx.Done():
			t.Fatalf("等待第二次执行超时: %v", ctx.Err())
		case <-time.After(80 * time.Millisecond):
		}
	}
	t.Fatal("第二次提交没有在时限内执行")
}

// describeView 只打印结构与状态，不打印任何会话正文。
func describeView(t *testing.T, ctx context.Context, base *app.App, sessionID string) {
	t.Helper()
	view, ok := base.View(sessionID)
	if !ok {
		t.Logf("view: missing session")
		return
	}
	t.Logf("view: ok=%t current=%t approvals=%d questions=%d turns=%d",
		ok, view.HasCurrent, view.PendingCounts.Approvals, view.PendingCounts.Questions, len(view.Turns))
	for i, turn := range view.Turns {
		t.Logf("view: turn[%d] status=%s", i, turn.Status)
	}
	for i, row := range view.Rows {
		tool := ""
		isError := false
		resultLen := 0
		if row.Tool != nil {
			tool = row.Tool.Name
			isError = row.Tool.IsError
			resultLen = len(row.Tool.Result)
		}
		t.Logf("view: row[%d] kind=%s status=%s tool=%s isError=%t resultLen=%d",
			i, row.Kind, row.Status, tool, isError, resultLen)
	}
}

func waitApproval(t *testing.T, ctx context.Context, base *app.App, sessionID string) interaction.Item {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		view, _ := base.View(sessionID)
		if view.HasCurrent && view.Current.Kind == interaction.PendingApproval {
			if view.PendingCounts.Approvals != 1 {
				t.Fatalf("待决审批应只有一条，实际 %d", view.PendingCounts.Approvals)
			}
			return view.Current
		}
		select {
		case <-ctx.Done():
			describeView(t, ctx, base, sessionID)
			t.Fatalf("等待审批请求超时: %v", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	describeView(t, ctx, base, sessionID)
	t.Fatal("等待审批请求超时")
	return interaction.Item{}
}

func waitToolResult(t *testing.T, ctx context.Context, base *app.App, sessionID string, wantOK bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		view, _ := base.View(sessionID)
		ok, failed := toolOutcome(view)
		if wantOK && ok {
			return
		}
		if !wantOK && failed {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("等待工具结果超时: %v", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatalf("等待工具结果超时")
}

// toolOutcome 只看工具有没有执行成功或失败，不返回任何工具输出正文。
func toolOutcome(view app.SessionView) (ok bool, failed bool) {
	for _, row := range view.Rows {
		if row.Tool == nil {
			continue
		}
		if row.Tool.IsError {
			failed = true
			continue
		}
		if strings.Contains(row.Tool.Result, realApprovalMarker) {
			ok = true
		}
	}
	return ok, failed
}

func turnEnded(view app.SessionView) bool {
	if len(view.Turns) == 0 {
		return false
	}
	return view.Turns[len(view.Turns)-1].Status != "running"
}

func submitRealPrompt(t *testing.T, ctx context.Context, base *app.App, sessionID string) {
	t.Helper()
	view, err := base.Submit(ctx, app.SubmitRequest{
		SessionID: sessionID,
		Text:      realApprovalPrompt,
	})
	if err != nil {
		t.Fatalf("提交提示词: %v", err)
	}
	if _, err := base.Wait(ctx, view.ID); err != nil {
		t.Fatalf("等待提交结果: %v", err)
	}
}

func setWorkspaceWrite(ctx context.Context, base *app.App, sessionID string) error {
	return base.SetPermissionTier(ctx, sessionID, interaction.PermissionWorkspaceWrite)
}

func createRealSession(t *testing.T, ctx context.Context, base *app.App) string {
	t.Helper()
	workspace := t.TempDir()
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatalf("创建工作区: %v", err)
	}
	entry, err := base.Catalog().Create(ctx, catalog.CreateRequest{
		Cwd: workspace,
		Key: fmt.Sprintf("approval-evidence-%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("新建会话: %v", err)
	}
	if _, err := base.OpenSession(ctx, entry.ID); err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	t.Logf("session: id=%s opened=true", entry.ID)
	return entry.ID
}

func startRealApp(t *testing.T, home, fixture string) *app.App {
	t.Helper()
	base, err := app.New(app.Config{
		DataDir:      home,
		ServerPath:   filepath.Join(fixture, "bin", "dimcode-server"),
		ClientName:   "dimagent-approval-evidence",
		Version:      "0.1.0",
		HistoryLimit: 20,
		SendTimeout:  90 * time.Second,
		Verify: write.VerifyPolicy{
			MaxAttempts: 3,
			Interval:    time.Second,
			Timeout:     10 * time.Second,
		},
		RequestTimeout: 30 * time.Second,
		CloseTimeout:   20 * time.Second,
		Logf: func(format string, args ...any) {
			t.Logf("app: "+format, args...)
		},
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := base.Connect(ctx); err != nil {
		t.Fatalf("连接 daemon: %v", err)
	}
	if err := base.Subscribe(ctx); err != nil {
		t.Fatalf("订阅: %v", err)
	}
	if err := base.FirstHistory(ctx); err != nil {
		t.Fatalf("首轮历史: %v", err)
	}
	if err := base.AllowWrites(ctx); err != nil {
		t.Fatalf("打开写闸门: %v", err)
	}
	return base
}

// prepareApprovalHome 复制夹具 home 到临时目录，并把权限改成需要审批。
func prepareApprovalHome(t *testing.T, fixture string) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "dimagent-approval-")
	if err != nil {
		t.Fatalf("创建临时夹具目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	home := filepath.Join(root, "home")
	if err := copyDir(filepath.Join(fixture, "home"), home); err != nil {
		t.Fatalf("复制夹具 home: %v", err)
	}
	for _, name := range []string{"daemon.sock", "daemon.sock.lock", "daemon.start.lock"} {
		_ = os.Remove(filepath.Join(home, "rust", name))
	}
	configPath := filepath.Join(home, "config.json")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取夹具配置: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("解析夹具配置: %v", err)
	}
	config["toolApproval"] = "ask"
	config["permission"] = "workspace-write"
	updated, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("编码夹具配置: %v", err)
	}
	if err := os.WriteFile(configPath, updated, 0o600); err != nil {
		t.Fatalf("写夹具配置: %v", err)
	}
	t.Logf("fixture: copied=true toolApproval=ask permission=workspace-write")
	return home
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
