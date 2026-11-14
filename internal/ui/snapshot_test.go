package ui

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件渲染首片界面的截图，存到 docs/domains/screens：五个状态各一张，
// 另有三张变体（摘要展开、右栏展开、左栏收起）。
//
//	go test ./internal/ui -run TestScreens -update   # 重写 PNG
//	go test ./internal/ui -run TestScreens          # 只比对
//
// 渲染走框架的无窗口软件渲染（ui.NewTester + Tester.Image），不建窗口、不连
// App Server：图里只有样例，没有事实。
//
// 像素比对只在 darwin 上做：文字栅格化由系统排版引擎负责（Core Text /
// Pango / DirectWrite），不同平台的字形与抗锯齿本就不同，跨平台比像素没有
// 意义。所有平台都跑 TestScreensFitWindow 的布局检查。

// updateScreens 让测试重写截图；默认只比对。
var updateScreens = flag.Bool("update", false, "重写 docs/domains/screens 下的界面截图")

// screenDir 是截图目录，相对本包目录。
const screenDir = "../../docs/domains/screens"

// screenCase 是一张截图：在哪个状态下、先设好哪个变体、窗口要不要留出
// 系统窗口控件的位置。
type screenCase struct {
	name  string
	state State
	dark  bool
	// titleBar 是窗口控件（红绿灯/最小化最大化关闭）占据的空间：非 nil 表示
	// 窗口带系统控件、没有系统标题栏（UI-19）。值由平台的 c.TitleBar() 给，
	// 截图里用手报的实测值（macOS：高 28、左侧 78）。
	titleBar *gui.TitleBar
	setup    func(*App)
}

// screenCases 是截图表：五个状态 + 三个变体。
// windowTitleBar 是 macOS 窗口控件的占位（高 28、左侧 78）：截图都按"无系统
// 标题栏的窗口"出（UI-19），顶部那条由 c.TitleBar() 给出，不是写死的布局常量。
var windowTitleBar = gui.TitleBar{Height: 28, Left: 78}

var screenCases = []screenCase{
	{name: "state-default", state: StateDefault, titleBar: &windowTitleBar},
	{name: "state-running", state: StateRunning, titleBar: &windowTitleBar},
	{name: "state-waiting", state: StateWaiting, titleBar: &windowTitleBar},
	{name: "state-failed", state: StateFailed, titleBar: &windowTitleBar},
	{name: "state-empty", state: StateEmpty, titleBar: &windowTitleBar},
	{name: "variant-summary-open", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		*a.openOf(0) = true
	}},
	{name: "variant-right-panel", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.rightOpen = true
	}},
	{name: "variant-sidebar-collapsed", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.sidebarCollapsed = true
	}},
	// UI-15 / F8：上下文用量在输入卡内，将满时的同一处形态变化与明细。
	{name: "variant-context-near", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.contextUsed = 180_000
		a.contextOpen = true
	}},
	// UI-23：输入框里只有光标时没有占位文案；有草稿时文字落在内容列上。
	{name: "variant-draft", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.draft = "把日志也开上"
	}},
	// 分支：标题行的分支标记与它的原生菜单（非默认分支才出现，UI-9）。
	{name: "variant-branch", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.branch, a.branches = branchSamples()
		a.branchMenuOpen = true
	}},
	// 会话目录动作：行末菜单、行内重命名、删除确认、搜索面。
	{name: "variant-session-menu", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		*a.menuOpenPtr(0) = true
	}},
	{name: "variant-rename", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.renaming, a.renameDraft = 0, "修发送路径"
	}},
	{name: "variant-session-delete", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.deleting = 0
	}},
	{name: "variant-search", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.searchOpen = true
	}},
	// 终端三态（右栏第五个标签）：空闲 / 运行中 / 已退出。
	{name: "variant-term", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.rightOpen, a.rightTab = true, 4
	}},
	{name: "variant-term-running", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.rightOpen, a.rightTab = true, 4
		a.termState = termRunning
		a.termLines = termSamples(termRunning)
	}},
	{name: "variant-term-exited", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.rightOpen, a.rightTab = true, 4
		a.termState = termExited
		a.termLines = termSamples(termExited)
	}},
	// 设置 · 扩展：四个页签 + 列表（插件 8 行）。
	{name: "variant-ext", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.screen, a.settingsSection, a.extTab = ScreenSettings, 3, 1
	}},
	// 设置 · 扩展：读取失败态。
	{name: "variant-ext-failed", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.screen, a.settingsSection, a.extState = ScreenSettings, 3, extFailed
	}},
	// 富文本：Markdown 的块级样式（标题、列表、行内代码、代码块、引用、表格、链接）。
	{name: "variant-rich", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.rounds = []round{{
			user:   "把排队的实现讲清楚",
			reply:  richReply,
			status: roundDone,
			ops:    []op{{action: "读取", target: "internal/ui/composer.go", result: "完成"}},
		}}
	}},
	// 分支 · 结束分支的确认态（AC-08 两步）。
	{name: "variant-branch-end", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.branch, a.branches = branchSamples()
		a.branchMenuOpen = true
		a.endBranchConfirm = true
	}},
	// 目标：输入区上沿的目标行进入设置态。
	{name: "variant-goal", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.goalEditing = true
	}},
	// 设置屏（P2）：左导航 + 三个分节；默认停在命令规则（含危险动作）。
	{name: "variant-settings", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.screen = ScreenSettings
		a.settingsSection = 0
	}},
	// 设置 · 命令规则的清空确认态（危险动作两步）。
	{name: "variant-settings-rules", state: StateDefault, titleBar: &windowTitleBar, setup: func(a *App) {
		a.screen = ScreenSettings
		a.settingsSection = 2
		a.clearRulesConfirm = true
	}},
	// 后台任务：右栏「后台」标签下的运行中任务。
	{name: "variant-tasks", state: StateRunning, titleBar: &windowTitleBar, setup: func(a *App) {
		a.rightOpen = true
		a.rightTab = 2
	}},
	// UI-19：无系统标题栏的窗口，红绿灯落在左栏顶部上方的那条里；标题行是自绘的。
	{name: "variant-titlebar", state: StateDefault, titleBar: &gui.TitleBar{Height: 28, Left: 78}},
	// 暗色不是另一个状态，是同一批 token 的另一套值（U2 §3.3）。
	{name: "variant-dark", state: StateDefault, dark: true, titleBar: &windowTitleBar},
}

// trafficLightDiameter 与 trafficLightGap 是截图里三个示意圆点的尺寸与间距：
// 与 macOS 默认窗口控件的观感一致，只用于截图标注。
const (
	trafficLightDiameter = 12
	trafficLightGap      = 20
)

// drawTrafficLights 在标题栏让位区里画三个圆点，标出系统窗口控件的位置。
//
// 这三个点**不是界面的绘制**：红绿灯由 macOS 画在窗口上，软件渲染的截图里没有。
// 这里画出来只是让截图能看出"控件落在自绘标题行的让位条里"，颜色取系统默认值。
// 界面代码不画它们（UI-19：窗口控件归平台，界面只让位）。
func drawTrafficLights(img *image.RGBA, bar gui.TitleBar) {
	colors := [3]color.RGBA{
		{R: 0xff, G: 0x5f, B: 0x57, A: 0xff},
		{R: 0xfe, G: 0xbc, B: 0x2e, A: 0xff},
		{R: 0x28, G: 0xc8, B: 0x40, A: 0xff},
	}
	cy := int(bar.Height / 2)
	for i, c := range colors {
		cx := 20 + i*trafficLightGap
		for y := cy - trafficLightDiameter/2; y < cy+trafficLightDiameter/2; y++ {
			for x := cx - trafficLightDiameter/2; x < cx+trafficLightDiameter/2; x++ {
				dx, dy := x-cx, y-cy
				if dx*dx+dy*dy <= (trafficLightDiameter/2)*(trafficLightDiameter/2) {
					img.SetRGBA(x, y, c)
				}
			}
		}
	}
}

// screenTesterAt 按指定窗口宽度渲染一张截图（三档对照用）。
func screenTesterAt(c screenCase, width int) *gui.Tester {
	a := NewApp(Options{State: c.state})
	if c.setup != nil {
		c.setup(a)
	}
	tt := gui.NewTesterWith(a.View, width, WindowHeight, gui.TesterOptions{Dark: c.dark})
	return tt
}

// screenTester 渲染一张截图，返回其 Tester，供取图与量元素用。
func screenTester(c screenCase) *gui.Tester {
	a := NewApp(Options{State: c.state})
	if c.setup != nil {
		c.setup(a)
	}
	tt := gui.NewTesterWith(a.View, WindowWidth, WindowHeight, gui.TesterOptions{Dark: c.dark})
	if c.titleBar != nil {
		tt.SetTitleBar(*c.titleBar)
	}
	return tt
}

// TestScreens 比对（或重写）截图。默认只比对；-update 重写。
func TestScreens(t *testing.T) {
	dir := filepath.Join(screenDir)
	if *updateScreens {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("建截图目录：%v", err)
		}
	}
	// 文字栅格化随平台走，像素只在 darwin 上可比。
	compare := runtime.GOOS == "darwin"
	if !compare {
		t.Logf("%s：跳过像素比对（截图是 darwin 的参考图），只检查能否渲染", runtime.GOOS)
	}
	for _, c := range screenCases {
		tt := screenTester(c)
		img := tt.Image()
		if c.titleBar != nil {
			drawTrafficLights(img, *c.titleBar)
		}
		if w, h := img.Bounds().Dx(), img.Bounds().Dy(); w != WindowWidth || h != WindowHeight {
			t.Fatalf("%s：图 %d×%d，想要 %d×%d", c.name, w, h, WindowWidth, WindowHeight)
		}
		path := filepath.Join(dir, c.name+".png")
		if *updateScreens {
			if err := writePNG(path, img); err != nil {
				t.Fatalf("%s：写图：%v", c.name, err)
			}
			continue
		}
		if !compare {
			continue
		}
		want, err := readPNG(path)
		if err != nil {
			t.Fatalf("%s：读参考图：%v（先跑 go test ./internal/ui -run TestScreens -update）", c.name, err)
		}
		if diff, total := diffPixels(want, img); diff > total/500 {
			t.Fatalf("%s：与参考图差 %d/%d 像素（>0.2%%）；确认是要改图后跑 -update", c.name, diff, total)
		} else if diff > 0 {
			t.Logf("%s：与参考图差 %d/%d 像素（在 0.2%% 容差内，可能是字体版本差异）", c.name, diff, total)
		}
	}
}

// TestScreensFitWindow 检查每张图里的文字都没有越出窗口、也没有零尺寸：
// 裁切与溢出的自动检查，任何平台都跑。
func TestScreensFitWindow(t *testing.T) {
	for _, c := range screenCases {
		tt := screenTester(c)
		for _, text := range tt.Texts() {
			r, ok := tt.Find(text)
			if !ok {
				continue // 同名的只记第一个
			}
			if r.W <= 0 || r.H <= 0 {
				continue // 滚出视口的行没有可见面积，属正常（长列表在视口外）
			}
			if r.X < 0 || r.Y < 0 ||
				r.X+r.W > WindowWidth || r.Y+r.H > WindowHeight {
				t.Fatalf("%s：%q 越出窗口（%+v）", c.name, text, r)
			}
		}
		// 设置屏没有对话流与输入位（独立成屏），跳过这条重叠检查。
		if _, ok := tt.Find("设置"); ok {
			continue
		}
		// 对话流与输入位不重叠：时间线在下沿给输入位让位。
		feed, ok := tt.Find("对话流")
		if !ok {
			t.Fatalf("%s：找不到对话流", c.name)
		}
		composer := "消息"
		if c.state == StateWaiting {
			composer = "rm -rf ./tmp/*"
		}
		box, ok := tt.Find(composer)
		if !ok {
			t.Fatalf("%s：找不到输入位（%q）", c.name, composer)
		}
		if feed.Y+feed.H > box.Y {
			t.Fatalf("%s：对话流（%+v）压到输入位（%+v）", c.name, feed, box)
		}
	}
}

// titleBarCloseupHeight 是标题行特写的高度：标题行 48 加左右两侧的一段内容，
// 够看清红绿灯让位与标题行的边界。
const titleBarCloseupHeight = 120

// TestTitleBarCloseup 出一张标题行特写（UI-19）：窗口顶部含系统控件让位、
// 标题行是自绘的。裁的是同一帧的软件渲染图，不是另画的示意图。
func TestTitleBarCloseup(t *testing.T) {
	c := screenCase{name: "variant-titlebar", state: StateDefault,
		titleBar: &gui.TitleBar{Height: 28, Left: 78}}
	img := screenTester(c).Image()
	drawTrafficLights(img, *c.titleBar)
	crop := image.NewRGBA(image.Rect(0, 0, WindowWidth, titleBarCloseupHeight))
	for y := 0; y < titleBarCloseupHeight; y++ {
		copy(crop.Pix[y*crop.Stride:], img.Pix[y*img.Stride:y*img.Stride+WindowWidth*4])
	}
	path := filepath.Join(screenDir, "titlebar-closeup.png")
	if *updateScreens {
		if err := os.MkdirAll(screenDir, 0o755); err != nil {
			t.Fatalf("建截图目录：%v", err)
		}
		if err := writePNG(path, crop); err != nil {
			t.Fatalf("写图：%v", err)
		}
		return
	}
	if runtime.GOOS != "darwin" {
		t.Logf("%s：跳过像素比对（截图是 darwin 的参考图）", runtime.GOOS)
		return
	}
	want, err := readPNG(path)
	if err != nil {
		t.Fatalf("读参考图：%v（先跑 go test ./internal/ui -run TestTitleBarCloseup -update）", err)
	}
	if diff, total := diffPixels(want, crop); diff > total/500 {
		t.Fatalf("标题行特写与参考图差 %d/%d 像素（>0.2%%）；确认是要改图后跑 -update", diff, total)
	}
}

// TestWordlessCheck 出遮字测试图（§9.14 第 0 层）：把界面上仍然保留的界面语
// （发送/停止的按钮文字、审批的三个决定词）用中性灰块盖住，检查"读历史、提交、
// 处理一次审批"三件事是否还认得出。审批准则上的三个决定词是 UI-23 的显式例外，
// 这里仍然遮住它们，逼着靠位置与层次辨认。
func TestWordlessCheck(t *testing.T) {
	c := screenCase{name: "wordless-check", state: StateDefault, titleBar: &windowTitleBar,
		setup: func(a *App) { a.draft = "把日志也开上" }}
	tt := screenTester(c)
	img := tt.Image()
	drawTrafficLights(img, *c.titleBar)
	mask := color.RGBA{R: 0xd9, G: 0xd9, B: 0xde, A: 0xff}
	for _, word := range []string{"发送", "停止"} {
		if r, ok := tt.Find(word); ok {
			for y := int(r.Y) - 1; y < int(r.Y+r.H)+1; y++ {
				for x := int(r.X) - 2; x < int(r.X+r.W)+2; x++ {
					img.SetRGBA(x, y, mask)
				}
			}
		}
	}
	path := filepath.Join(screenDir, "wordless-check.png")
	if *updateScreens {
		if err := writePNG(path, img); err != nil {
			t.Fatalf("写图：%v", err)
		}
	}
}

// writePNG 把图写到 path。
func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// readPNG 读参考图。
func readPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s：%w", path, err)
	}
	return img, nil
}

// diffPixels 返回两图不同像素数与比较的像素总数；尺寸不同时全部算差异。
func diffPixels(want, got image.Image) (diff, total int) {
	wb, gb := want.Bounds(), got.Bounds()
	if wb.Dx() != gb.Dx() || wb.Dy() != gb.Dy() {
		return wb.Dx() * wb.Dy(), wb.Dx() * wb.Dy()
	}
	for y := 0; y < gb.Dy(); y++ {
		for x := 0; x < gb.Dx(); x++ {
			wr, wg, wb2, _ := want.At(wb.Min.X+x, wb.Min.Y+y).RGBA()
			gr, gg, gb2, _ := got.At(gb.Min.X+x, gb.Min.Y+y).RGBA()
			if wr != gr || wg != gg || wb2 != gb2 {
				diff++
			}
			total++
		}
	}
	return diff, total
}
