package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
)

// TestConnectWithoutServerPathReportsHint 核对「没有可执行文件」时给界面的原因。
//
// 发现链的说明（试过哪些位置）就是界面原位显示的那句；拉起服务的内部细节只
// 留在 unwrap 里，供日志与分类使用。
func TestConnectWithoutServerPathReportsHint(t *testing.T) {
	dir := shortDataDir(t)
	const hint = "按顺序都没找到 App Server，试过：PATH 里的 dim-app-server → <data>/rust/daemon.sock"
	base, err := New(Config{
		DataDir:    dir,
		ServerHint: hint,
		ClientName: "dimagent-hint-test",
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx := context.Background()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	connectErr := base.Connect(ctx)
	if connectErr == nil {
		t.Fatal("没有可执行文件时连接应失败")
	}
	if !strings.Contains(connectErr.Error(), hint) {
		t.Fatalf("原因应带上发现链的说明: %v", connectErr)
	}
	if strings.Contains(connectErr.Error(), "可执行文件路径") {
		t.Fatalf("界面原因不该是拉起服务的内部细节: %v", connectErr)
	}
	var missing error
	if unwrapped := errors.Unwrap(connectErr); unwrapped != nil {
		missing = unwrapped
	}
	if missing == nil {
		t.Fatal("原始错误应仍可 unwrap 出来")
	}
	if !failure.Is(connectErr, failure.KindNonRetryable) && !failure.Is(connectErr, failure.KindConnectionMissing) {
		t.Fatalf("原始错误的分类应仍可达: %v", connectErr)
	}
}

// TestSetServerPathFeedsNextConnect 核对重跑发现链之后，新的可执行文件路径真的
// 用在下一次连接上：启动时没有路径不是定局。
func TestSetServerPathFeedsNextConnect(t *testing.T) {
	dir := shortDataDir(t)
	base, err := New(Config{DataDir: dir, ClientName: "dimagent-hint-test"})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx := context.Background()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	base.SetServerPath("", "没有可执行文件")
	if err := base.Connect(ctx); err == nil {
		t.Fatal("没有可执行文件时连接应失败")
	}
	// 换成一个存在但跑不起来的路径：错误里应出现这个路径，说明新路径确实被用了。
	broken := dir + "/not-an-executable"
	if err := os.WriteFile(broken, []byte("not executable"), 0o644); err != nil {
		t.Fatalf("写占位文件: %v", err)
	}
	base.SetServerPath(broken, "")
	err = base.Connect(ctx)
	if err == nil {
		t.Fatal("不可执行的文件应连接失败")
	}
	if !strings.Contains(err.Error(), broken) {
		t.Fatalf("新的可执行文件路径应被用上: %v", err)
	}
}

// shortDataDir 建一个短路径的临时数据目录：Unix socket 路径有长度上界。
func shortDataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "dim-app-hint")
	if err != nil {
		t.Fatalf("建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
