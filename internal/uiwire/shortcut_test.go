package uiwire

import (
	"context"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/app"
)

// TestSetGlobalShortcutPersistsAndNormalizes：设置 › 快捷键的写入先规范化再落
// 本地偏好，并马上反映在 Source 读到的状态里。
func TestSetGlobalShortcutPersistsAndNormalizes(t *testing.T) {
	base, err := app.New(app.Config{DataDir: t.TempDir(), ClientName: "shortcut-wire-test"})
	if err != nil {
		t.Fatalf("创建集成层：%v", err)
	}
	wire := New(base)
	ctx := context.Background()

	if err := wire.SetGlobalShortcut(ctx, false, "⌘⇧K"); err != nil {
		t.Fatalf("保存全局快捷键：%v", err)
	}
	prefs := base.Preferences()
	if !prefs.GlobalShortcutOff || prefs.GlobalShortcutAccelerator != "Cmd+Shift+K" {
		t.Fatalf("偏好没有按规范写法落盘：%+v", prefs)
	}
	snapshot := wire.GlobalShortcut()
	if snapshot.Enabled || snapshot.Accelerator != "Cmd+Shift+K" || snapshot.Label == "" {
		t.Fatalf("状态没有跟上偏好：%+v", snapshot)
	}

	// 认不出的键位不落盘：用户的意图不该被一个拼错的值覆盖。
	before := base.Preferences()
	if err := wire.SetGlobalShortcut(ctx, true, "K"); err == nil {
		t.Fatal("裸键应当被拒绝")
	} else if !strings.Contains(err.Error(), "键位无法识别") {
		t.Fatalf("错误里没说清原因：%v", err)
	}
	if got := base.Preferences(); got.GlobalShortcutAccelerator != before.GlobalShortcutAccelerator ||
		got.GlobalShortcutOff != before.GlobalShortcutOff {
		t.Fatalf("被拒绝的键位改动了偏好：%+v", got)
	}

	if err := wire.SetGlobalShortcut(ctx, true, "CmdOrCtrl+Shift+Space"); err != nil {
		t.Fatalf("恢复默认键位：%v", err)
	}
	if got := wire.GlobalShortcut(); !got.Enabled || got.Accelerator != "CmdOrCtrl+Shift+Space" {
		t.Fatalf("重新打开后的状态 = %+v", got)
	}
}
