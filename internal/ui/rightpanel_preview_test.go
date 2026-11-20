package ui

import (
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 文件预览页（P-01）：文件页点一行就打开「打开文件」页，内容读 PanelFacts 的
// Preview；读态、二进制、截断与失败都来自快照，-demo 时用样例。

// previewTextSource 造一个带一份文本预览读数的源。
func previewTextSource() *fakePanelSource {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree(
		FileEntrySnapshot{Name: "notes.md"},
		FileEntrySnapshot{Name: "blob.bin"},
	)
	src.previews = map[string]PreviewSnapshot{
		"notes.md": {Kind: PreviewText, Encoding: "utf-8", Size: 12, ReadBytes: 12,
			Content: "hello\nworld\n", Phase: PanelReadReady},
		"blob.bin": {Kind: PreviewBinary, Size: 7, ReadBytes: 7,
			Phase: PanelReadReady, Note: "内容含 NUL 字节，按二进制处理"},
	}
	return src
}

// TestFileRowOpensPreview 断言点文件行打开预览页，并按同一路径读预览读数。
func TestFileRowOpensPreview(t *testing.T) {
	src := previewTextSource()
	app, tt := rightWiredTester(t, tabFiles, src)

	if err := tt.Click("文件，notes.md"); err != nil {
		t.Fatalf("点不到文件行：%v", err)
	}
	tt.Frame()
	if app.rightTab != tabFilePreview {
		t.Fatalf("点文件行之后停在第 %d 页，想要预览页", app.rightTab)
	}
	if src.lastPreview.Path != "notes.md" || src.lastPreview.Cwd != "/work" {
		t.Fatalf("预览读的请求 = %+v，想要 /work 下的 notes.md", src.lastPreview)
	}
	if got := app.rpTabLabel(tabFilePreview); got != "notes.md" {
		t.Fatalf("预览标签文案 = %q，想要文件名 notes.md（RP-A-12）", got)
	}
	if _, ok := tt.Find("打开文件"); ok {
		t.Fatal("选了文件后标签仍写着「打开文件」")
	}
	if got := joinedText(tt); !strings.Contains(got, "hello\nworld\n") {
		t.Fatalf("预览页没有显示文件内容：\n%s", got)
	}
	if _, ok := tt.Find("12 B · utf-8"); !ok {
		t.Fatal("页头没有画大小与编码")
	}
}

// TestPreviewBinaryAndFailure 断言二进制给说明、读取失败给原因与重试。
func TestPreviewBinaryAndFailure(t *testing.T) {
	src := previewTextSource()
	app, tt := rightWiredTester(t, tabFiles, src)

	if err := tt.Click("文件，blob.bin"); err != nil {
		t.Fatalf("点不到文件行：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("二进制或未识别编码，不显示内容。"); !ok {
		t.Fatal("二进制文件没有给说明")
	}
	if _, ok := tt.Find("内容含 NUL 字节，按二进制处理"); !ok {
		t.Fatal("二进制文件的读取说明没有上屏")
	}

	// 读取失败：危险色原因 + 重试入口，点重试让下一次读带上 Refresh。
	src.previews["notes.md"] = PreviewSnapshot{Phase: PanelReadFailed, Error: "读取文件: permission denied"}
	app.right.previewPath = "notes.md"
	app.rightTab = tabFilePreview
	tt.Frame()
	if _, ok := tt.Find("读取文件: permission denied"); !ok {
		t.Fatal("读取失败没有给出原因")
	}
	// Tester.Click 自己跑一帧：刷新标记要在这一帧的读里出现。
	if err := tt.Click("重试读取文件"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	if !src.lastPreview.Refresh {
		t.Fatal("点重试之后的一次预览读没有要求刷新")
	}
}

// TestPreviewDemoKeepsSample 断言 -demo（没有 Source）时预览页给样例，而不是空白。
func TestPreviewDemoKeepsSample(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.rightOpen, app.rightTab = true, tabFiles
	// 样例树很长，根下的 main.go 在视口外：直接落在预览页上验样例正文。
	app.right.previewPath = "main.go"
	app.rightTab = tabFilePreview
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if app.rightTab != tabFilePreview {
		t.Fatalf("点样例文件行之后停在第 %d 页，想要预览页", app.rightTab)
	}
	if got := joinedText(tt); !strings.Contains(got, "hello from the preview sample") {
		t.Fatalf("样例模式下预览页没有内容：\n%s", got)
	}
	if _, ok := tt.Find("样例内容：连着 App Server 时这里读磁盘上的真实文件。"); !ok {
		t.Fatal("样例内容没有说明它是样例")
	}
}

// TestPreviewEmptyPath 断言没有选文件时预览页给空态，不发任何读。
func TestPreviewEmptyPath(t *testing.T) {
	src := previewTextSource()
	_, tt := rightWiredTester(t, tabFilePreview, src)
	if _, ok := tt.Find("没有打开的文件"); !ok {
		t.Fatal("没有选文件时没有空态")
	}
	if src.lastPreview.Path != "" {
		t.Fatalf("空态时仍发了预览读：%+v", src.lastPreview)
	}
}
