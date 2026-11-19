package tray

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sync"
)

// 托盘图标取参考产品的模板图标（黑字透明底，macOS 会按菜单栏明暗重新着色），
// 状态标记画在右下角：运行中是一个圆点，等待回应是一个感叹号。模板图标只有
// 明暗两色，所以标记靠形状区分，不靠颜色。
//
// 资源是参考产品的 icon-trayTemplate@2x.png（32×32 像素，即 16pt 的 2x 表示）。
//
//go:embed assets/tray-template.png
var templatePNG []byte

var (
	iconOnce  sync.Once
	iconBase  image.Image
	iconErr   error
	iconCache = map[Status][]byte{}
)

// Icon 返回某个状态下的托盘图标（PNG）。同一个状态只编码一次。
func Icon(status Status) ([]byte, error) {
	iconOnce.Do(func() {
		iconBase, iconErr = png.Decode(bytes.NewReader(templatePNG))
	})
	if iconErr != nil {
		return nil, iconErr
	}
	if data, ok := iconCache[status]; ok {
		return data, nil
	}
	bounds := iconBase.Bounds()
	canvas := image.NewRGBA(bounds)
	draw.Draw(canvas, bounds, iconBase, bounds.Min, draw.Src)
	ink := color.RGBA{A: 0xff}
	switch status {
	case StatusRunning:
		fillCircle(canvas, 25, 26, 4, ink)
	case StatusWaiting:
		fillRect(canvas, 24, 17, 27, 25, ink)
		fillRect(canvas, 24, 27, 27, 30, ink)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, err
	}
	iconCache[status] = buf.Bytes()
	return iconCache[status], nil
}

// fillCircle 在画布上填一个实心圆（坐标含边界）。
func fillCircle(canvas *image.RGBA, cx, cy, r int, ink color.Color) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				canvas.Set(x, y, ink)
			}
		}
	}
}

// fillRect 填一个矩形（右下边界不含）。
func fillRect(canvas *image.RGBA, x0, y0, x1, y1 int, ink color.Color) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			canvas.Set(x, y, ink)
		}
	}
}
