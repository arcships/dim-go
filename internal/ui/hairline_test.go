package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 高 DPI：1px 的线只有落在设备像素格上才是实心色。这组测试在 1×、1.5×、2×
// 三档下量像素：横线与竖线必须是 round(DPR) 个整像素、颜色正好是 t.Border，
// 相邻像素正好是背景（没有半像素的灰边）。

// linePixels 沿一条线取像素，返回「边界色」的坐标；既不是背景也不是边界色的
// 位置记成 -1-i（那就是发灰的半像素）。
func linePixels(img *image.RGBA, fixed int, horizontal bool, bg, border color.RGBA) []int {
	b := img.Bounds()
	n, span := b.Dy(), b.Dx()
	if !horizontal {
		n, span = b.Dx(), b.Dy()
	}
	out := make([]int, 0, n)
	for i := 0; i < n; i++ {
		var x, y int
		if horizontal {
			x, y = fixed, i
		} else {
			x, y = i, fixed
		}
		got := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
		switch got {
		case bg:
		case border:
			out = append(out, i)
		default:
			out = append(out, -1-i)
		}
	}
	_ = span
	return out
}

// TestHairlinePaintsWholeDevicePixels：三档 DPR 下 1 DIP 的横线与竖线都正好
// 占 round(DPR) 个实心设备像素，没有半像素的过渡色。
func TestHairlinePaintsWholeDevicePixels(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		for _, horizontal := range []bool{true, false} {
			var bg, border gui.Color
			view := func(c *gui.Context) {
				th := c.Theme()
				bg, border = th.Background, th.Border
				gui.Column(c).Fill().Background(th.Background).Children(func() {
					gui.Row(c).Height(space1).Shrink(0)
					if horizontal {
						hairlineBox(c, true, "")
						return
					}
					gui.Row(c).Grow(1).Children(func() {
						gui.Box(c).Width(space2).Shrink(0)
						hairlineBox(c, false, "").Height(24)
					})
				})
			}
			tt := gui.NewTester(view, 100, 40)
			tt.SetScale(scale)
			tt.Frame()
			img := tt.Image()
			b := img.Bounds()
			want := int(math.Round(float64(scale)))
			var got []int
			if horizontal {
				got = linePixels(img, b.Dx()/2, true, colorOf(bg), colorOf(border))
			} else {
				got = linePixels(img, b.Dy()/2, false, colorOf(bg), colorOf(border))
			}
			if len(got) != want {
				t.Fatalf("%v×（横线 %v）：线上有 %d 个像素，想要 %d 个：%v",
					scale, horizontal, len(got), want, got)
			}
			for i, v := range got {
				if v < 0 {
					t.Fatalf("%v×（横线 %v）：第 %d 个像素不是实心边界色（发灰）：%v",
						scale, horizontal, i, got)
				}
			}
			limit := b.Dy()
			if !horizontal {
				limit = b.Dx()
			}
			if got[0] < 1 || got[len(got)-1] > limit-2 {
				t.Fatalf("%v×（横线 %v）：线贴边或被裁：%v", scale, horizontal, got)
			}
		}
	}
}

// TestSnapDIPRoundsToDeviceGrid：取整函数是"就近取整到设备像素"，0 与无效
// DPR 原样返回。
func TestSnapDIPRoundsToDeviceGrid(t *testing.T) {
	cases := []struct {
		v     float32
		scale float32
		want  float32
	}{
		{3.5, 1, 4},
		{3.5, 1.5, 10.0 / 3.0}, // 5.25 → 5 设备像素
		{3.5, 2, 3.5},          // 7 设备像素
		{10.25, 2, 10.5},       // 20.5 → 21
		{10.25, 1.5, 10},       // 15.375 → 15 → 10 DIP
		{0, 2, 0},
		{7, 0, 7},
	}
	for _, tc := range cases {
		got := snapDIP(tc.v, tc.scale)
		if math.Abs(float64(got-tc.want)) > 1e-5 {
			t.Fatalf("snapDIP(%v, %v) = %v，想要 %v", tc.v, tc.scale, got, tc.want)
		}
	}
}

// TestHairlineRectKeepsOneDevicePixel：线框厚度取 max(1, round(DPR)) 个设备
// 像素，起点取整；横线只动 y，竖线只动 x。
func TestHairlineRectKeepsOneDevicePixel(t *testing.T) {
	onGrid := func(v, scale float32) bool {
		d := float64(v*scale) - math.Round(float64(v*scale))
		return math.Abs(d) < 1e-4
	}
	thickDevices := func(h, scale float32) float64 { return float64(h*scale) - math.Round(float64(scale)) }
	for _, scale := range []float32{1, 1.5, 2} {
		h := hairlineRect(gui.Rect{X: 10, Y: 3.5, W: 50, H: strokeHairline}, scale, true)
		if math.Abs(float64(h.W)-50) > 1e-4 {
			t.Fatalf("%v×：横线宽 %v，想要 50", scale, h.W)
		}
		if math.Abs(thickDevices(h.H, scale)) > 1e-4 {
			t.Fatalf("%v×：横线厚 %v DIP = %v 设备像素", scale, h.H, h.H*scale)
		}
		if !onGrid(h.Y, scale) || !onGrid(h.X, scale) || !onGrid(h.X+h.W, scale) {
			t.Fatalf("%v×：横线 %+v 不在设备像素格上", scale, h)
		}
		v := hairlineRect(gui.Rect{X: 3.5, Y: 10, W: strokeHairline, H: 40}, scale, false)
		if math.Abs(thickDevices(v.W, scale)) > 1e-4 {
			t.Fatalf("%v×：竖线厚 %v DIP = %v 设备像素", scale, v.W, v.W*scale)
		}
		if !onGrid(v.X, scale) || !onGrid(v.Y, scale) || !onGrid(v.Y+v.H, scale) {
			t.Fatalf("%v×：竖线 %+v 不在设备像素格上", scale, v)
		}
	}
}

// TestSidebarIconsAlignOnDeviceColumn：左栏四个全局入口的图标是同一列——
// 左缘相同、且 2× 下落在整数设备像素上（图标对齐的判据：同一列的图标不能
// 相差半个像素）。
func TestSidebarIconsAlignOnDeviceColumn(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.SetScale(2)
	tt.Frame()
	var column float32
	seen := 0
	for _, mark := range []string{"plus", "search", "clock", "layout-grid"} {
		r, ok := tt.Find("图标：" + mark)
		if !ok {
			t.Fatalf("找不到图标 %q：%v", mark, tt.Texts())
		}
		if seen == 0 {
			column = r.X
		} else if r.X != column {
			t.Fatalf("图标 %q 的左缘 %v 与同列 %v 不同", mark, r.X, column)
		}
		seen++
		if d := float64(r.X*2) - math.Round(float64(r.X*2)); math.Abs(d) > 1e-4 {
			t.Fatalf("图标 %q 的左缘 %v 不在设备像素格上", mark, r.X)
		}
	}
}
