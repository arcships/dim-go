package ui

import (
	"math"

	gui "github.com/egoist/mygo/ui"
)

// 高 DPI（2× 与 1.5×）：1 DIP 的线在 2× 屏上是 2 个设备像素，只有整条落在
// 设备像素格上才是实心色；起点偏半个像素就会被抗锯齿摊成一条灰边——截图里
// 「1px 分隔线发灰」就是这一种。自算几何的线与图标在绘制前先按 DPR 取整，
// 取整在 DIP 里做（设备像素 = DIP × DPR），布局尺寸不变。

// snapDIP 把 v（DIP）对到最近的设备像素边界。
func snapDIP(v, scale float32) float32 {
	if scale <= 0 || v == 0 {
		return v
	}
	return float32(math.Round(float64(v*scale))) / scale
}

// deviceLength 是 n 个设备像素在 DIP 里的长度（DPR 无效时按 1×）。
func deviceLength(scale float32) float32 {
	if scale <= 0 {
		scale = 1
	}
	return 1 / scale
}

// hairlineRect 把一条 1 DIP 的线对到设备像素格：起点取整，厚度取
// max(1, round(DPR)) 个设备像素后换回 DIP。2× 上正好两行实心像素，1× 上一行。
// horizontal 为真时按横线（厚度在 y），否则按竖线（厚度在 x）。
func hairlineRect(r gui.Rect, scale float32, horizontal bool) gui.Rect {
	if scale <= 0 {
		scale = 1
	}
	thick := float32(math.Max(1, math.Round(float64(scale)))) * deviceLength(scale)
	if horizontal {
		y := snapDIP(r.Y, scale)
		x0, x1 := snapDIP(r.X, scale), snapDIP(r.X+r.W, scale)
		if x1 <= x0 {
			x1 = x0 + deviceLength(scale)
		}
		return gui.Rect{X: x0, Y: y, W: x1 - x0, H: thick}
	}
	x := snapDIP(r.X, scale)
	y0, y1 := snapDIP(r.Y, scale), snapDIP(r.Y+r.H, scale)
	if y1 <= y0 {
		y1 = y0 + deviceLength(scale)
	}
	return gui.Rect{X: x, Y: y0, W: thick, H: y1 - y0}
}

// snapIconRect 把图标盒的四缘对到设备像素格：图标的描边以内为盒缘，盒缘落在
// 半像素上时整枚图标会糊一档（16×16 的线性图标只有 1.5 DIP 描边）。
func snapIconRect(r gui.Rect, scale float32) gui.Rect {
	if scale <= 0 {
		scale = 1
	}
	x0, y0 := snapDIP(r.X, scale), snapDIP(r.Y, scale)
	x1, y1 := snapDIP(r.X+r.W, scale), snapDIP(r.Y+r.H, scale)
	if x1 <= x0 {
		x1 = x0 + deviceLength(scale)
	}
	if y1 <= y0 {
		y1 = y0 + deviceLength(scale)
	}
	return gui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// hairlineBox 是 1 DIP 的分隔线：占位仍是 1 DIP，绘制时按 DPR 落到设备像素格。
// horizontal 为真画横线，否则画竖线；label 非空时作为读屏名。
func hairlineBox(c *gui.Context, horizontal bool, label string) gui.Element {
	t := c.Theme()
	e := gui.Box(c)
	if horizontal {
		e = e.Height(strokeHairline).Shrink(0)
	} else {
		e = e.Width(strokeHairline).Shrink(0)
	}
	if label != "" {
		e = e.Label(label).AccessHidden(true)
	}
	return e.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Fill(hairlineRect(r, p.Scale(), horizontal), t.Border, 0)
	})
}

// paintIcon 画一枚图标：盒子先按 DPR 取整到设备像素格，再交给框架光栅化。
// 盒缘落在半像素上时 16×16 的线性图标（1.5 DIP 描边）会糊一档。
func paintIcon(p *gui.Painter, icon *gui.SVG, r gui.Rect, color gui.Color) {
	p.Icon(icon, snapIconRect(r, p.Scale()), color)
}
