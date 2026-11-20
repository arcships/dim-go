package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「添加模型」对话框的回归：标题、添加自定义模型、使用模板创建与空态。
// 需求 ID 写在各个用例注释里。

// openModelDraft 打开添加模型对话框并返回 tester。
func openModelDraft(t *testing.T) (*App, *gui.Tester) {
	t.Helper()
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: providerDetailSource()})
	tt := openSettingsSection(t, app, secProviders)
	if err := tt.Click("添加模型"); err != nil {
		t.Fatalf("点不到添加模型：%v", err)
	}
	tt.Frame()
	if configScreen.draft == nil {
		t.Fatal("添加模型对话框没有打开")
	}
	return app, tt
}

// TestAddModelDialogCopy 断言对话框的标题与两段说明逐字取自参考（S-configuration-78/79）。
func TestAddModelDialogCopy(t *testing.T) {
	_, tt := openModelDraft(t)

	for _, want := range []string{
		"添加模型", "添加自定义模型", configCopyAddModelHint,
		"使用模板创建", configCopyTemplateDesc,
	} {
		if !tt.HasText(want) {
			t.Fatalf("对话框缺少 %q：%v", want, tt.Texts())
		}
	}
}

// TestAddModelCustomPrefillsID 断言「添加自定义模型」把搜索词预填成模型标识
// （S-configuration-78）。
func TestAddModelCustomPrefillsID(t *testing.T) {
	app, tt := openModelDraft(t)

	configScreen.draft.query = "my-model-7"
	tt.Frame()
	if err := tt.Click("添加自定义模型"); err != nil {
		t.Fatalf("点不到添加自定义模型：%v", err)
	}
	tt.Frame()
	if got := configScreen.draft.rows; len(got) == 0 || got[len(got)-1] != "my-model-7" {
		t.Fatalf("草稿行 = %v，想要末行 my-model-7", got)
	}
	if _, ok := tt.AccessByRole(gui.RoleTextField, "草稿模型 2"); !ok {
		t.Fatalf("预填的模型标识没有输入框：%v", tt.Texts())
	}
	_ = app
}

// TestAddModelTemplatePick 断言模板按来源供应商分组，点一个模板把它的标识加进草稿
// （S-configuration-79）。
func TestAddModelTemplatePick(t *testing.T) {
	_, tt := openModelDraft(t)

	if !tt.HasText("Acme") || !tt.HasText("Model-X") {
		t.Fatalf("模板分组缺少来源供应商或模型：%v", tt.Texts())
	}
	if err := tt.Click("模板模型 Model-X"); err != nil {
		t.Fatalf("点不到模板模型：%v", err)
	}
	tt.Frame()
	found := false
	for _, row := range configScreen.draft.rows {
		if row == "model-x" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("模板没有加成草稿行：%v", configScreen.draft.rows)
	}
}

// TestAddModelNoMatches 断言搜索词没有命中时写「没有匹配的模型」（S-configuration-80）。
func TestAddModelNoMatches(t *testing.T) {
	_, tt := openModelDraft(t)

	configScreen.draft.query = "zzz-nothing"
	tt.Frame()
	if !tt.HasText("没有匹配的模型") {
		t.Fatalf("空搜索结果没有写空态：%v", tt.Texts())
	}
}
