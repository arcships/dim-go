package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「高 DPI」这一类需求的取证：2×（以及 1.5×）下 1 DIP 的线必须正好占
// round(DPR) 个实心设备像素，相邻像素是底色——半像素的过渡色就是截图里「1px
// 分隔线发灰」。量的是真实屏上画出来的像素，不是单独渲染一根线的夹具。

// rgbaAt 取图像上一点的 sRGB 值。
func rgbaAt(img *image.RGBA, x, y int) color.RGBA {
	return color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
}

// paintedRun 在布局矩形附近量一段画出来的线：返回线色占的设备像素个数、是否
// 整段同色，以及线两侧紧邻像素是否与线色不同（没有半像素过渡）。
//
// 起点按画出来的第一颗实心像素算：布局矩形可能落在半像素上（1.5×），绘制侧
// 才把它对到整像素——这里要证明的正是「画出来的那段是实心的」。
func paintedRun(t *testing.T, img *image.RGBA, r gui.Rect, horizontal bool, scale float32) (n int, uniform, edged bool) {
	t.Helper()
	at := func(i, fixed int) color.RGBA {
		if horizontal {
			return rgbaAt(img, fixed, i)
		}
		return rgbaAt(img, i, fixed)
	}
	// 横线量厚度：固定横坐标（线中点），沿 y 扫；竖线量厚度：固定纵坐标，沿 x 扫。
	var fixed, origin int
	if horizontal {
		fixed = int(math.Round(float64((r.X + r.W/2) * scale)))
		origin = int(math.Round(float64(r.Y * scale)))
	} else {
		fixed = int(math.Round(float64((r.Y + r.H/2) * scale)))
		origin = int(math.Round(float64(r.X * scale)))
	}
	limit := img.Bounds().Dx()
	if horizontal {
		limit = img.Bounds().Dy()
	}
	from, to := origin-3, origin+int(math.Max(2, math.Round(float64(scale))))+3
	if from < 0 {
		from = 0
	}
	if to >= limit {
		to = limit - 1
	}
	// 绘制侧把 1 DIP 的线对到设备像素格：线从 snapDIP(起点) 那一颗开始。
	line := at(origin, fixed)
	start, end := -1, -1
	for i := from; i <= to; i++ {
		if at(i, fixed) != line {
			continue
		}
		if start < 0 {
			start = i
		}
		end = i
	}
	if start < 0 {
		return 0, false, false
	}
	uniform = true
	for i := start; i <= end; i++ {
		if at(i, fixed) != line {
			uniform = false
		}
	}
	if start-1 >= from && end+1 <= to {
		edged = at(start-1, fixed) != line && at(end+1, fixed) != line
	}
	return end - start + 1, uniform, edged
}

// TestHairlineOnRealScreensLandsOnDevicePixels：标题行的竖向分隔线与窄窗设置的
// 横向分隔线在三档 DPR 下都是 round(DPR) 个实心设备像素，两侧是底色。
//
// 证据：R-TB-293（TB-38 标题行竖分隔线）。
func TestHairlineOnRealScreensLandsOnDevicePixels(t *testing.T) {
	cases := []struct {
		name  string
		open  func() *App
		width int
		label string
		horiz bool
	}{
		{
			name:  "标题行·竖向分隔线",
			open:  func() *App { return NewApp(Options{State: StateDefault, Source: wiredSource()}) },
			width: WindowWidth,
			label: "分隔线",
		},
		{
			name: "设置·导航与内容的分隔（窄窗横向）",
			open: func() *App {
				app := NewApp(Options{State: StateDefault})
				app.screen, app.settingsSection = ScreenSettings, secAccount
				return app
			},
			width: narrowWidth,
			label: "导航与内容的分隔",
			horiz: true,
		},
	}
	for _, c := range cases {
		for _, scale := range []float32{1, 1.5, 2} {
			t.Run(c.name+"/"+scaleLabel(scale), func(t *testing.T) {
				tt := gui.NewTester(c.open().View, c.width, WindowHeight)
				tt.SetScale(scale)
				tt.Frame()
				r, ok := tt.Find(c.label)
				if !ok {
					t.Fatalf("找不到分隔线 %q：%v", c.label, tt.Texts())
				}
				n, uniform, edged := paintedRun(t, tt.Image(), r, c.horiz, scale)
				want := int(math.Max(1, math.Round(float64(scale))))
				if n != want {
					t.Fatalf("%s：线占 %d 个设备像素，想要 %d（布局矩形 %+v，DPR %v）",
						c.name, n, want, r, scale)
				}
				if !uniform {
					t.Fatalf("%s：线内有过渡色（发灰）（%+v，DPR %v）", c.name, r, scale)
				}
				if !edged {
					t.Fatalf("%s：线两侧不是底色（另有相邻色）", c.name)
				}
			})
		}
	}
}

// scaleLabel 把 DPR 写成用例名里的短标签。
func scaleLabel(scale float32) string {
	switch scale {
	case 1:
		return "1x"
	case 1.5:
		return "1.5x"
	case 2:
		return "2x"
	}
	return "dpr"
}

// TestSnappedIconBoxesLandOnDevicePixels：图标盒按 DPR 取整后四缘都落在整设备
// 像素上、厚度至少一个设备像素；半像素的盒缘会让 16×16 的线性图标糊一档。
func TestSnappedIconBoxesLandOnDevicePixels(t *testing.T) {
	boxes := []gui.Rect{
		{X: 421.99194, Y: 16, W: 16, H: 16},
		{X: 12.5, Y: 7.25, W: 14, H: 14},
		{X: 0, Y: 0, W: 18, H: 18},
		{X: 1390.0001, Y: 17, W: 16, H: 16},
	}
	for _, scale := range []float32{1, 1.5, 2} {
		for _, box := range boxes {
			got := snapIconRect(box, scale)
			for _, v := range []float32{got.X, got.Y, got.X + got.W, got.Y + got.H} {
				if !onDevicePixel(v, scale) {
					t.Fatalf("DPR %v：%+v 取整后边 %v 不在整像素上（%+v）", scale, box, v, got)
				}
			}
			if got.W*scale < 1-1e-3 || got.H*scale < 1-1e-3 {
				t.Fatalf("DPR %v：%+v 取整后不到一个设备像素（%+v）", scale, box, got)
			}
		}
	}
}

// onDevicePixel 报告一条 DIP 边在 DPR 下是否正好落在整设备像素上。
func onDevicePixel(v, scale float32) bool {
	dev := float64(v) * float64(scale)
	return math.Abs(dev-math.Round(dev)) < 1e-3
}
