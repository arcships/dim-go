package ui

import (
	"flag"
	"fmt"
	"image"
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

// screenCase 是一张截图：在哪个状态下、先设好哪个变体。
type screenCase struct {
	name  string
	state State
	dark  bool
	setup func(*App)
}

// screenCases 是截图表：五个状态 + 三个变体。
var screenCases = []screenCase{
	{name: "state-default", state: StateDefault},
	{name: "state-running", state: StateRunning},
	{name: "state-waiting", state: StateWaiting},
	{name: "state-failed", state: StateFailed},
	{name: "state-empty", state: StateEmpty},
	{name: "variant-summary-open", state: StateDefault, setup: func(a *App) {
		a.summaryOpen[0] = true
	}},
	{name: "variant-right-panel", state: StateDefault, setup: func(a *App) {
		a.rightOpen = true
	}},
	{name: "variant-sidebar-collapsed", state: StateDefault, setup: func(a *App) {
		a.sidebarCollapsed = true
	}},
	// 暗色不是另一个状态，是同一批 token 的另一套值（U2 §3.3）。
	{name: "variant-dark", state: StateDefault, dark: true},
}

// screenTester 渲染一张截图，返回其 Tester，供取图与量元素用。
func screenTester(c screenCase) *gui.Tester {
	a := NewApp(Options{State: c.state})
	if c.setup != nil {
		c.setup(a)
	}
	return gui.NewTesterWith(a.View, WindowWidth, WindowHeight, gui.TesterOptions{Dark: c.dark})
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
		img := screenTester(c).Image()
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
				t.Fatalf("%s：%q 没有可见面积（%+v）", c.name, text, r)
			}
			if r.X < 0 || r.Y < 0 ||
				r.X+r.W > WindowWidth || r.Y+r.H > WindowHeight {
				t.Fatalf("%s：%q 越出窗口（%+v）", c.name, text, r)
			}
		}
		// 对话流与输入位不重叠：时间线在下沿给输入位让位。
		feed, ok := tt.Find("对话流")
		if !ok {
			t.Fatalf("%s：找不到对话流", c.name)
		}
		composer := "消息"
		if c.state == StateWaiting {
			composer = "需要你同意：跑一条命令"
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
