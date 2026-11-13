// Command dimagent-ui 是首片界面骨架的入口：用静态样例把
// docs/domains/U1-ia.md 的六个区域与五种状态画出来。
//
//	go run ./cmd/dimagent-ui
//	go run ./cmd/dimagent-ui -state running
//
// 调试键：1–5 切换状态（默认 / 运行中 / 等待你同意 / 失败 / 空态），
// Tab 在区域之间移动焦点。本入口不连 App Server，也不含任何写路径。
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	appui "github.com/arcships/dimagent-mygo/internal/ui"
)

func main() {
	stateFlag := flag.String("state", "default",
		"起始状态：default、running、waiting、failed、empty（也可写 1–5）")
	widthFlag := flag.Int("width", appui.WindowWidth, "窗口宽度（逻辑像素，用于三档宽度对照截图）")
	heightFlag := flag.Int("height", appui.WindowHeight, "窗口高度（逻辑像素）")
	flag.Parse()

	state, err := appui.ParseState(*stateFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	var win *mygo.Window
	title := func(s appui.State) string {
		return "DimAgent — " + s.String() + "（界面骨架）"
	}

	app := appui.NewApp(appui.Options{
		State: state,
		StateChanged: func(s appui.State) {
			if win != nil {
				win.SetTitle(title(s))
			}
		},
	})

	// 菜单只留平台必需的最小项（UI-19）：应用菜单（退出、隐藏、关于）、编辑菜单、
	// 窗口菜单。编辑菜单不是产品功能，而是 macOS 把 ⌘C/⌘V/⌘X/⌘A/⌘Z 送进窗口的
	// 唯一通道（框架 ui/input.go 的 commandKeys 与 commandKey 注释）；文件与视图
	// 菜单没有对应能力，不提供（UI-6、UI-19）。
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleWindowMenu},
	}))

	mygo.App.WhenReady(func() {
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:     title(state),
			Width:     *widthFlag,
			Height:    *heightFlag,
			MinWidth:  720,
			MinHeight: 600,
			// 无系统标题栏，保留系统阴影、圆角与边缘缩放；红绿灯由系统画在我们
			// 自绘标题行的上方（UI-19、A4 平台律）。
			TitleBarStyle: mygo.TitleBarHidden,
			Content:       ui.View(app.View),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
