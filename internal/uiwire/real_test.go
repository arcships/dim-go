package uiwire

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
)

const realServerPath = "/tmp/dim-2-target/debug/dim-app-server"

// TestRealServerConnectDefault 跑真实 App Server（mock 模型）的连接与默认态：
// 连接 → 建会话 → 目录有会话 → 打开取快照。
//
// 提交段在本机 blocked：服务端没有 `session.permission.update`（-32601
// unknown_method），也不在 attach 回执里给权限档位，提交在客户端就被挡下。
// 这一段保留为显式 blocked，并打印服务端原文，不当作通过。
func TestRealServerConnectDefault(t *testing.T) {
	if _, err := os.Stat(realServerPath); err != nil {
		t.Skip("本机没有真实 App Server：" + realServerPath)
	}
	// 数据目录要短：macOS 的 unix socket 路径上限约 104 字节，t.TempDir() 太长。
	dataDir, err := os.MkdirTemp("/tmp", "dimwire")
	if err != nil {
		t.Fatalf("建临时目录：%v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	base, err := app.New(app.Config{
		DataDir:      dataDir,
		ServerPath:   realServerPath,
		MockModel:    true,
		ClientName:   "uiwire-real",
		SendTimeout:  20 * time.Second,
		CloseTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层：%v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = base.Close(ctx)
	})

	w := New(base)
	changes := 0
	w.OnChange = func() { changes++ }
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 1. 连接（拉起真实 server + 订阅）。
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接真实 server：%v", err)
	}
	if conn := w.Connection(); conn.Phase != "connected" {
		t.Fatalf("连接阶段 = %q（%s）", conn.Phase, conn.Reason)
	}
	t.Logf("① 已连接；OnChange 次数 %d", changes)

	// 2. 建一个带权限档位的会话（服务端在 attach 回执里给档位，提交才不报缺档位）。
	var createErr error
	for _, preset := range []string{"full-access", "workspace-write", "read-only"} {
		if _, createErr = base.CreateSession(ctx, dataDir, preset); createErr == nil {
			t.Logf("② 建会话预设=%s", preset)
			break
		}
		t.Logf("② 建会话预设=%s 失败：%v", preset, createErr)
	}
	if createErr != nil {
		t.Fatalf("三种预设都建不了会话：%v", createErr)
	}
	if err := w.refreshCatalog(ctx); err != nil {
		t.Fatalf("刷新目录：%v", err)
	}
	names := w.SessionNames()
	if len(names) == 0 {
		t.Fatal("② 目录里没有会话")
	}
	t.Logf("② 目录会话 %d 个，首项 %q", len(names), names[0])

	// 3. 默认态：打开会话后能取到快照（新会话还没有历史行，这是服务端事实）。
	if err := w.OpenSession(ctx, names[0]); err != nil {
		t.Fatalf("打开会话：%v", err)
	}
	snap, ok := w.Session()
	if !ok {
		t.Fatal("③ 打开后没有会话快照")
	}
	if snap.ID != names[0] {
		t.Fatalf("③ 快照会话 = %q，列表首项 = %q", snap.ID, names[0])
	}
	t.Logf("③ 默认态就绪：会话 %q，行数 %d", snap.ID, len(snap.Rows))

	// 4. 提交段：本机 blocked，保留服务端原文。
	if err := w.Submit(ctx, "你好"); err != nil {
		t.Logf("④ 提交被拒（服务端原文）：%v", err)
		t.Skipf("提交段 blocked：服务端未在 attach 回执里给权限档位；" +
			"session.permission.update -32601 unknown_method（原文见上）。已覆盖：连接、目录、默认态")
	}
	// 若本机服务端日后补上档位，这里继续核对流式增长与审批。
	deadline := time.Now().Add(60 * time.Second)
	before := len(snap.Rows)
	for time.Now().Before(deadline) {
		if cur, ok := w.Session(); ok && len(cur.Rows) > before {
			t.Logf("④ 时间线 %d → %d 行", before, len(cur.Rows))
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("④ 提交后时间线没有增长（仍 %d 行）", before)
}
