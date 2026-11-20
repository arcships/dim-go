package ui

// 本文件逐态验收「导入 Chrome Cookie」与「清理浏览器数据」两个对话框
// （G-H6-03 / 06 / 07 / 08 / 09a / 09b / 10、G-H7-03 / 04）。
//
// 通道没有接：默认态由 settings_general_browser_test.go 验收（只写缺什么、
// 动作置灰）。这里注入视图状态，把每个态都画出来并按文案、角色与禁用原因
// 验收；不把注入的数据说成通道读数。

import (
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// openBrowserStateDialog 打开设置 › 通用，并把两个对话框的状态换成给的态。
func openBrowserStateDialog(t *testing.T, st generalState) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: StateDefault})
	tt := openSettingsSection(t, app, secGeneral)
	generalScreen = st
	t.Cleanup(func() { generalScreen = generalState{} })
	tt.Frame()
	return app, tt
}

// TestGeneralImportScanningState：扫描行是旋转图标 + 逐字文案 + 进度条，最小高
// 96；扫描中不出现空态；「导入」置灰并写原因（G-H6-03）。
func TestGeneralImportScanningState(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen:    true,
		browserImport: browserImportState{phase: browserImportScanning, progress: 0.4},
	})
	if !tt.HasText(browserImportScanningText) {
		t.Fatalf("扫描行没有逐字文案：%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(gui.RoleProgress, "扫描 Chrome 配置"); !ok {
		t.Fatalf("扫描行没有进度条：%v", tt.Texts())
	}
	if _, ok := tt.Find(browserImportEmptyText); ok {
		t.Fatal("扫描中出现了空态（还没有结论）")
	}
	if info, ok := tt.Access("导入"); !ok || !info.Disabled {
		t.Fatalf("扫描中「导入」没有置灰：%+v（找到 %v）", info, ok)
	}
	if info, ok := tt.Access("导入"); ok && info.Description != noteBrowserNoImport {
		t.Fatalf("「导入」的禁用原因 = %q，想要 %q", info.Description, noteBrowserNoImport)
	}
}

// TestGeneralImportNoProfilesState：无配置空态逐字「未找到 Google Chrome
// 配置。」、居中；不画分组标签；「导入」置灰（G-H6-06）。
func TestGeneralImportNoProfilesState(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen:    true,
		browserImport: browserImportState{phase: browserImportNoProfiles},
	})
	if _, ok := tt.Find(browserImportEmptyText); !ok {
		t.Fatalf("没有空态逐字文案：%v", tt.Texts())
	}
	if _, ok := tt.Find("Chrome 配置"); ok {
		t.Fatal("无配置时不该出现分组标签")
	}
	if info, ok := tt.Access("导入"); !ok || !info.Disabled {
		t.Fatalf("无配置时「导入」没有置灰：%+v（找到 %v）", info, ok)
	}
}

// TestGeneralImportReadingState：读取行逐字「正在读取所选配置…」；读取中不画
// 计数块，避免旧计数闪在屏上（G-H6-07）。
func TestGeneralImportReadingState(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen: true,
		browserImport: browserImportState{
			phase:  browserImportReading,
			counts: browserCookieCounts{importable: 128, session: 96, expired: 20, partitioned: 12},
		},
	})
	if !tt.HasText(browserImportReadingText) {
		t.Fatalf("读取行没有逐字文案：%v", tt.Texts())
	}
	if tt.HasText("可导入 128") {
		t.Fatalf("读取中仍画了旧计数：%v", tt.Texts())
	}
	if info, ok := tt.Access("导入"); !ok || !info.Disabled {
		t.Fatalf("读取中「导入」没有置灰：%+v（找到 %v）", info, ok)
	}
}

// TestGeneralImportReadyShowsProfilesAndCounts：配置单选行与四段计数逐字；
// 说明句在可导入数非 0 时出现（G-H6-02 / 04 / 05 / 08）。
func TestGeneralImportReadyShowsProfilesAndCounts(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen: true,
		browserImport: browserImportState{
			phase: browserImportReady,
			profiles: []browserChromeProfile{
				{name: "Default", capability: "可读"},
				{name: "Profile 1", capability: "需要解锁"},
			},
			selected: 0,
			counts:   browserCookieCounts{importable: 128, session: 96, expired: 20, partitioned: 12},
		},
	})
	if _, ok := tt.Find("Chrome 配置"); !ok {
		t.Fatalf("缺分组标签：%v", tt.Texts())
	}
	for _, want := range []string{"Default", "Profile 1", "可读", "需要解锁",
		"可导入 128 · 会话 96 · 已过期 20 · 分区 12"} {
		if !tt.HasText(want) {
			t.Fatalf("配置态缺 %q：%v", want, tt.Texts())
		}
	}
	if !tt.HasText("此操作会替换内置浏览器中的全部非分区 Cookie") {
		t.Fatalf("可导入数非 0 时缺说明句：%v", tt.Texts())
	}
	if info, ok := tt.Access("导入"); !ok || !info.Disabled {
		t.Fatalf("没有通道时「导入」没有置灰：%+v（找到 %v）", info, ok)
	}
	// 配置行是单选项：选中项读成 Selected，其余不选中（G-H6-05）。
	if info, ok := tt.AccessByRole(gui.RoleRadio, "Default 可读"); !ok || !info.Checked {
		t.Fatalf("选中的配置没有读成单选项：%+v（找到 %v）", info, ok)
	}
	if info, ok := tt.AccessByRole(gui.RoleRadio, "Profile 1 需要解锁"); !ok || info.Checked {
		t.Fatalf("未选中的配置读成了选中：%+v（找到 %v）", info, ok)
	}
}

// TestGeneralImportZeroCountsHidesDescription：可导入数为 0 时不画说明句
// （没有内容可替换），计数块仍写出四段 0（G-H6-02 / G-H6-08）。
func TestGeneralImportZeroCountsHidesDescription(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen: true,
		browserImport: browserImportState{
			phase:    browserImportReady,
			profiles: []browserChromeProfile{{name: "Default", capability: "可读"}},
			counts:   browserCookieCounts{},
		},
	})
	if tt.HasText("此操作会替换内置浏览器中的全部非分区 Cookie") {
		t.Fatalf("可导入数为 0 仍画了说明句：%v", tt.Texts())
	}
	if !tt.HasText("可导入 0 · 会话 0 · 已过期 0 · 分区 0") {
		t.Fatalf("计数块没有照写 0：%v", tt.Texts())
	}
}

// TestGeneralImportPartialFailureState：部分失败是告警行 + 计数 + 12px 说明；
// 主按钮转「关闭」，不再给「导入」（G-H6-09b）。
func TestGeneralImportPartialFailureState(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen: true,
		browserImport: browserImportState{
			phase:      browserImportPartial,
			profiles:   []browserChromeProfile{{name: "Default", capability: "可读"}},
			counts:     browserCookieCounts{importable: 128, session: 96, expired: 20, partitioned: 12},
			imported:   120,
			failed:     8,
			diagnostic: "",
		},
	})
	want := "已导入 120 · 失败 8 · 已跳过 20 个过期 Cookie 和 12 个分区 Cookie。"
	if !tt.HasText(want) {
		t.Fatalf("部分失败缺计数句 %q：%v", want, tt.Texts())
	}
	if !tt.HasText(browserImportPartialText) {
		t.Fatalf("部分失败缺警示说明：%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "关闭导入 Chrome Cookie"); !ok {
		t.Fatalf("回执后主按钮没有转「关闭」：%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "导入"); ok {
		t.Fatal("回执后仍给了「导入」")
	}
}

// TestGeneralImportSuccessState：失败数为 0 走成功行（对勾）；不出现警示说明。
func TestGeneralImportSuccessState(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen: true,
		browserImport: browserImportState{
			phase:    browserImportDone,
			profiles: []browserChromeProfile{{name: "Default", capability: "可读"}},
			counts:   browserCookieCounts{importable: 128, session: 96, expired: 20, partitioned: 12},
			imported: 128,
		},
	})
	if !tt.HasText("已导入 128 · 失败 0 · 已跳过 20 个过期 Cookie 和 12 个分区 Cookie。") {
		t.Fatalf("成功回执不对：%v", tt.Texts())
	}
	if tt.HasText(browserImportPartialText) {
		t.Fatal("失败数为 0 仍画了部分失败说明")
	}
}

// TestGeneralImportErrorRetryState：错误块是危险色消息 + 等宽诊断行 + 置灰的
// 「重试」（缺通道写原因）；读屏按 alert 播报（G-H6-10）。
func TestGeneralImportErrorRetryState(t *testing.T) {
	const message = "无法访问 Chrome 加密密钥。"
	const diagnostic = "chrome_key_access_denied: keychain locked"
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen: true,
		browserImport: browserImportState{
			phase:      browserImportFailed,
			message:    message,
			diagnostic: diagnostic,
		},
	})
	if !tt.HasText(message) || !tt.HasText(diagnostic) {
		t.Fatalf("错误块缺消息或诊断行：%v", tt.Texts())
	}
	if info, ok := tt.AccessByRole(gui.RoleAlert, message); !ok {
		t.Fatalf("错误块没有按 alert 播报：%+v（找到 %v）", info, ok)
	}
	info, ok := tt.Access("重试")
	if !ok || !info.Disabled {
		t.Fatalf("「重试」不存在或没有置灰：%+v（找到 %v）", info, ok)
	}
	if info.Description != noteBrowserNoImport {
		t.Fatalf("「重试」的原因 = %q，想要 %q", info.Description, noteBrowserNoImport)
	}
}

// TestGeneralClearSuccessState：成功行逐字、对勾；次按钮转「关闭」，危险按钮
// 不再出现（G-H7-03 / G-H7-05）。
func TestGeneralClearSuccessState(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		clearOpen:    true,
		browserClear: browserClearState{phase: browserClearSuccess},
	})
	if !tt.HasText(browserClearSuccessText) {
		t.Fatalf("没有成功行：%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "关闭清理浏览器数据"); !ok {
		t.Fatalf("成功行后次按钮没有转「关闭」：%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "清理浏览器数据"); ok {
		t.Fatal("成功后仍给了「清理浏览器数据」")
	}
	if tt.HasText(noteBrowserNoClear) {
		t.Fatal("成功回执仍写着缺通道原因")
	}
}

// TestGeneralClearFailureState：失败块是危险色消息 + 等宽诊断行；按钮恢复但
// 仍因缺通道置灰，取消仍在（G-H7-04 / G-H7-06）。
func TestGeneralClearFailureState(t *testing.T) {
	const message = "浏览器数据未能完全清理，请重试。"
	const diagnostic = "clear_failed: browser pages still open"
	_, tt := openBrowserStateDialog(t, generalState{
		clearOpen: true,
		browserClear: browserClearState{
			phase:      browserClearFailed,
			message:    message,
			diagnostic: diagnostic,
		},
	})
	if !tt.HasText(message) || !tt.HasText(diagnostic) {
		t.Fatalf("失败块缺消息或诊断行：%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(gui.RoleAlert, message); !ok {
		t.Fatal("失败块没有按 alert 播报")
	}
	info, ok := tt.AccessByRole(gui.RoleButton, "清理浏览器数据")
	if !ok || !info.Disabled || info.Description != noteBrowserNoClear {
		t.Fatalf("失败后清理按钮没有按缺通道置灰：%+v（找到 %v）", info, ok)
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "取消清理浏览器数据"); !ok {
		t.Fatalf("失败后缺「取消」：%v", tt.Texts())
	}
	if tt.HasText(browserClearSuccessText) {
		t.Fatal("失败态画了成功行")
	}
}

// TestGeneralClearClearingState：清理中是执行文案 + 两个按钮都禁用；说明句
// 保留，不换成进度文案（G-H7-02 / G-H7-06）。
func TestGeneralClearClearingState(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		clearOpen:    true,
		browserClear: browserClearState{phase: browserClearClearing},
	})
	if !tt.HasText("正在清理浏览器数据…") {
		t.Fatalf("清理中没有执行文案：%v", tt.Texts())
	}
	if !tt.HasText(generalClearDesc) {
		t.Fatalf("清理中说明句被换掉了：%v", tt.Texts())
	}
	if info, ok := tt.AccessByRole(gui.RoleButton, "取消"); !ok || !info.Disabled {
		t.Fatalf("清理中「取消」没有禁用：%+v（找到 %v）", info, ok)
	}
}

// TestGeneralImportReadingShowsActivity：读取中同样有不确定进度条；不带百分比的
// 扫描用不确定进度条（没有比例读数就不写编出来的百分比）（G-H6-03 / G-H6-07）。
func TestGeneralImportReadingShowsActivity(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen:    true,
		browserImport: browserImportState{phase: browserImportReading},
	})
	if _, ok := tt.AccessByRole(gui.RoleProgress, "读取所选 Chrome 配置"); !ok {
		t.Fatalf("读取中没有活动进度条：%v", tt.Texts())
	}
}

// TestGeneralImportScanningUnknownProgress：没有进度读数时画不确定进度条，
// 不显示任何百分比文案（G-H6-03）。
func TestGeneralImportScanningUnknownProgress(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen:    true,
		browserImport: browserImportState{phase: browserImportScanning, progress: 0},
	})
	if _, ok := tt.AccessByRole(gui.RoleProgress, "扫描 Chrome 配置"); !ok {
		t.Fatalf("扫描中没有进度条：%v", tt.Texts())
	}
	for _, text := range tt.Texts() {
		if strings.Contains(text, "%") {
			t.Fatalf("没有比例读数却写了百分比 %q", text)
		}
	}
}

// TestGeneralImportRetryInFlight：重试进行中「重试」同样禁用，原因写执行态
// （G-H6-10）。
func TestGeneralImportRetryInFlight(t *testing.T) {
	_, tt := openBrowserStateDialog(t, generalState{
		importOpen: true,
		browserImport: browserImportState{
			phase:    browserImportFailed,
			message:  "无法读取 Chrome Cookie，请关闭 Chrome 后重试。",
			retrying: true,
		},
	})
	info, ok := tt.Access("重试")
	if !ok || !info.Disabled {
		t.Fatalf("重试进行中「重试」没有禁用：%+v（找到 %v）", info, ok)
	}
	if info.Description != "正在重试" {
		t.Fatalf("重试进行中的原因 = %q，想要「正在重试」", info.Description)
	}
}
