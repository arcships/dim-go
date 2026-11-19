package uiwire

import (
	"context"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// TestAppearancePreferencesCrossLayer：设置 › 外观的本地偏好在三层之间原样穿行
// ——集成层落盘的 app.Preferences.Appearance 读成界面快照，界面写回来的快照又
// 落进 preferences.json。任何一层漏字段，这里都会失败。
func TestAppearancePreferencesCrossLayer(t *testing.T) {
	base, err := app.New(app.Config{DataDir: t.TempDir(), ClientName: "appearance-wire-test"})
	if err != nil {
		t.Fatalf("创建集成层：%v", err)
	}
	wire := New(base)
	ctx := context.Background()

	want := ui.AppearancePreferences{
		UIFont: "PingFang SC", CodeFont: "Menlo", UIScale: 1.08, CodeSize: 18,
		CodeTheme: "One Dark Pro",
	}
	prefs := wire.Preferences()
	prefs.Appearance = want
	if err := wire.SetPreferences(ctx, prefs); err != nil {
		t.Fatalf("保存偏好：%v", err)
	}

	if got := base.Preferences().Appearance; got != (app.AppearancePreferences{
		UIFont: want.UIFont, CodeFont: want.CodeFont, UIScale: want.UIScale,
		CodeSize: want.CodeSize, CodeTheme: want.CodeTheme,
	}) {
		t.Fatalf("写入没有落进本地偏好：%+v", got)
	}
	if got := wire.Preferences().Appearance; got != want {
		t.Fatalf("偏好读回界面快照不一致：\n got=%+v\nwant=%+v", got, want)
	}
}
