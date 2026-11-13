package ui

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「与官方控件并排对照」的取证工具（U0 §control-baseline）：
//   - TestBaselineSheet 渲染官方控件的对照表（1x / 2x）并量出几何；
//   - TestBaselineScreens 渲染三屏的"官方控件版"参照屏（参照物，不是我们的界面）。
//
// 取值只看这里量到的数字与框架源码（ui/widgets.go、ui/theme.go 的默认值），
// 不按印象调。

// controlSheet 把官方控件按同一列摆出来，每行一个控件，方便量外框。
func controlSheet(c *gui.Context) {
	t := c.Theme()
	s := "输入"
	sel := "选项"
	on := false
	off := false
	gui.Column(c).Padding(space4).Gap(space3).Children(func() {
		gui.Text(c, "Button").Label("capButton")
		gui.Button(c, "按钮").Label("button")
		gui.Text(c, "PrimaryButton").Label("capPrimary")
		gui.PrimaryButton(c, "主按钮").Label("primary")
		gui.Button(c, "禁用按钮").Label("disabled").Disabled(true)
		gui.Text(c, "TextInput").Label("capInput")
		gui.TextInput(c, &s).Label("input")
		gui.Text(c, "Select").Label("capSelect")
		gui.Select(c, &sel, []string{"选项", "另一项"}).Label("select")
		gui.Text(c, "Switch").Label("capSwitch")
		gui.Switch(c, &on).Label("switch")
		gui.Switch(c, &off).Label("switchOff")
		gui.Text(c, "Divider").Label("capDivider")
		gui.Divider(c).Label("divider")
		gui.Text(c, "Text 正文").Label("capText")
		gui.Text(c, "正文 14px").Label("text")
		gui.Text(c, "行高由主题给").Label("text2")
		_ = t
	})
}

// TestBaselineSheet 出两张对照表（1x / 2x）并打印几何。
func TestBaselineSheet(t *testing.T) {
	dir := filepath.Join("..", "..", "docs", "domains", "screens", "baseline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("建目录：%v", err)
	}
	for _, scale := range []int{1, 2} {
		tt := gui.NewTester(controlSheet, 320, 620)
		img := tt.Image()
		if scale == 2 {
			img = scaleImage(img, 2)
		}
		path := filepath.Join(dir, fmt.Sprintf("controls-%dx.png", scale))
		if err := writePNG(path, img); err != nil {
			t.Fatalf("写 %s：%v", path, err)
		}
	}
	tt := gui.NewTester(controlSheet, 320, 620)
	for _, name := range []string{"button", "primary", "disabled", "input", "select", "switch", "divider", "text"} {
		info, ok := tt.Access(name)
		if !ok {
			t.Fatalf("量不到 %s", name)
		}
		t.Logf("%-8s X=%.1f Y=%.1f W=%.1f H=%.1f", name, info.Bounds.X, info.Bounds.Y, info.Bounds.W, info.Bounds.H)
	}
}

// scaleImage 把图放大 n 倍（最近邻），用于出 2x 对照。
func scaleImage(src *image.RGBA, n int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, src.Bounds().Dx()*n, src.Bounds().Dy()*n))
	for y := 0; y < dst.Bounds().Dy(); y++ {
		for x := 0; x < dst.Bounds().Dx(); x++ {
			dst.Set(x, y, src.At(x/n, y/n))
		}
	}
	return dst
}
