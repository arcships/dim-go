package uiwire

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/ui"
)

// 右栏终端的 pty 接线：界面读的快照来自真实伪终端，输入原样写回，停止后状态
// 变回「没在跑」。用假服务端只给会话目录里的 cwd，pty 是本机真实的 shell。

// newTerminalWire 起一个连着假服务端、会话 cwd 指向真实临时目录的适配器。
func newTerminalWire(t *testing.T) (*App, context.Context) {
	t.Helper()
	root := t.TempDir()
	server := startE2EServer(t, panelScript(root))
	w := New(newRowsApp(t, server, root))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	return w, ctx
}

// TestTerminalRunsRealShell 断言「新建终端」起的是真 shell：输出从 pty 读到，
// 写进去的命令能跑出结果，停止后不再运行。
func TestTerminalRunsRealShell(t *testing.T) {
	w, ctx := newTerminalWire(t)
	if err := w.StartTerminal(ctx); err != nil {
		t.Fatalf("启动终端：%v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = w.StopTerminal(stopCtx)
	})

	snapshot := ui.TerminalSnapshot{}
	waitFor(t, "pty 输出", func() bool {
		snapshot = w.Terminal()
		return snapshot.Active && snapshot.Running && len(snapshot.Lines) > 0
	})
	// 真实 shell 起来就会打提示符：这里断言「读到了字节并切成了行」。
	if strings.TrimSpace(strings.Join(snapshot.Lines, "")) == "" {
		t.Fatalf("pty 有输出但切出来的行是空的：%+v", snapshot.Lines)
	}

	if err := w.WriteTerminal(ctx, "echo dimwire-terminal-ok\r"); err != nil {
		t.Fatalf("写入终端：%v", err)
	}
	waitFor(t, "命令输出", func() bool {
		return strings.Contains(strings.Join(w.Terminal().Lines, "\n"), "dimwire-terminal-ok")
	})

	if err := w.StopTerminal(ctx); err != nil {
		t.Fatalf("停止终端：%v", err)
	}
	waitFor(t, "终端停止", func() bool {
		after := w.Terminal()
		return after.Active && !after.Running
	})
	// 停止后回写失败，不会假装写进了已有的终端。
	if err := w.WriteTerminal(ctx, "echo x\r"); err == nil {
		t.Fatal("终端停止后写入仍成功了")
	}
}

// TestTerminalStreamSplitsLines 断言字节流按换行切行、CRLF 去掉行尾回车、半行
// 也留着（提示符通常不带换行）。
func TestTerminalStreamSplitsLines(t *testing.T) {
	stream := &terminalStream{}
	stream.append("第一行\r\n第二")
	stream.append("行\n提示符> ")
	got := stream.snapshot()
	want := []string{"第一行", "第二行", "提示符> "}
	if len(got) != len(want) {
		t.Fatalf("切出来的行 = %q，想要 %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("第 %d 行 = %q，想要 %q", i+1, got[i], want[i])
		}
	}
}

// scriptedReader 先给出预置的字节，再给出一个固定的错误（err 为空表示 EOF）。
type scriptedReader struct {
	chunks []string
	err    error
}

func (r *scriptedReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		if r.err == nil {
			return 0, io.EOF
		}
		return 0, r.err
	}
	n := copy(p, r.chunks[0])
	r.chunks = r.chunks[1:]
	return n, nil
}

// TestTerminalReadFailureIsAFact 断言读 pty 得到的 EOF 算正常收尾，其它读错误
// 登记成「读不到输出」的事实（界面据此收起输入行）；已读到的行照常保留。
func TestTerminalReadFailureIsAFact(t *testing.T) {
	eof := &terminalStream{}
	if err := pumpTerminal(&scriptedReader{chunks: []string{"$ ls\n"}}, eof, func() {}); !errors.Is(err, io.EOF) {
		t.Fatalf("读到 EOF 的返回 = %v，想要 io.EOF", err)
	}
	if eof.failed() {
		t.Fatal("EOF 被当成了读失败（进程正常退出不是读不到输出）")
	}

	broken := &terminalStream{}
	readErr := errors.New("设备读失败")
	err := pumpTerminal(&scriptedReader{chunks: []string{"$ make\n", "building…\n"}, err: readErr}, broken, func() {})
	if !errors.Is(err, readErr) {
		t.Fatalf("读失败的返回 = %v，想要 %v", err, readErr)
	}
	if !broken.failed() {
		t.Fatal("读到非 EOF 错误却没有登记成读失败")
	}
	if got := broken.snapshot(); len(got) != 2 || got[1] != "building…" {
		t.Fatalf("读失败前读到的行没有保留：%q", got)
	}
}

// TestTerminalStartRequiresSession 断言没有会话 / 没有工作目录时明确拒绝启动，
// 不猜一个目录。
func TestTerminalStartRequiresSession(t *testing.T) {
	w := New(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := w.startTerminalFor(ctx, "")
	if err == nil || !strings.Contains(err.Error(), "没有打开的会话") {
		t.Fatalf("没有会话时启动终端 = %v，想要明确拒绝", err)
	}
	err = w.writeTerminalFor(ctx, "", "x")
	if err == nil || !strings.Contains(err.Error(), "没有打开的会话") {
		t.Fatalf("没有会话时写入终端 = %v，想要明确拒绝", err)
	}
}
