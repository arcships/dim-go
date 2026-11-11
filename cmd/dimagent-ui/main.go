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

	mygo.App.WhenReady(func() {
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:     title(state),
			Width:     appui.WindowWidth,
			Height:    appui.WindowHeight,
			MinWidth:  960,
			MinHeight: 600,
			Content:   ui.View(app.View),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
