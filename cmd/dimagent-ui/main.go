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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/app"
	appui "github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/uiwire"
)

func main() {
	stateFlag := flag.String("state", "default",
		"起始状态：default、running、waiting、failed、empty（也可写 1–5）")
	connectFlag := flag.Bool("connect", false,
		"连接真实 App Server（app.New + 连接阶段）；失败时界面显示原因。默认 -demo 用样例数据")
	dataDirFlag := flag.String("data-dir", "", "-connect 模式下的客户端数据目录")
	serverFlag := flag.String("server", "", "-connect 模式下的 App Server 可执行文件")
	mockFlag := flag.Bool("mock-model", false, "-connect 模式下用 mock 模型拉起服务端")
	variantFlag := flag.String("variant", "",
		"额外的走查态：branch（分支菜单）、goal（目标编辑）、tasks（后台任务）、settings（设置）、settings-rules（清空确认）、ext / ext-failed / ext-empty / ext-loading（扩展四态）")
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

	variant, err := appui.ParseVariant(*variantFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// -connect：从真实集成层读事实；-demo（默认）：样例数据（截图与走查用）。
	var source appui.Source
	var intents appui.Intents
	var wire *uiwire.App
	var appBase *app.App
	if *connectFlag {
		if *dataDirFlag == "" {
			fmt.Fprintln(os.Stderr, "-connect 需要 -data-dir（客户端数据目录）")
			os.Exit(2)
		}
		base, err := app.New(app.Config{
			DataDir:    *dataDirFlag,
			ServerPath: *serverFlag,
			MockModel:  *mockFlag,
			ClientName: "dimagent-ui",
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "创建集成层：", err)
			os.Exit(1)
		}
		appBase = base
		wire = uiwire.New(base)
		source = wire
		intents = wire
	}

	app := appui.NewApp(appui.Options{
		Source:  source,
		Intents: intents,
		State:   state,
		Variant: variant,
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
		width, height, x, y := *widthFlag, *heightFlag, 0, 0
		hasPosition := false
		if appBase != nil {
			prefs := appBase.Preferences()
			if prefs.Window.Width > 0 {
				width = prefs.Window.Width
			}
			if prefs.Window.Height > 0 {
				height = prefs.Window.Height
			}
			x, y, hasPosition = prefs.Window.X, prefs.Window.Y, prefs.Window.HasPosition
		}
		options := mygo.WindowOptions{
			Title:     title(state),
			Width:     width,
			Height:    height,
			MinWidth:  720,
			MinHeight: 600,
			// 无系统标题栏，保留系统阴影、圆角与边缘缩放；红绿灯由系统画在我们
			// 自绘标题行的上方（UI-19、A4 平台律）。
			TitleBarStyle: mygo.TitleBarHidden,
			Content:       ui.View(app.View),
		}
		if hasPosition {
			options.X, options.Y = x, y
		}
		win = mygo.NewWindow(options)
		if appBase != nil {
			win.OnClose(func(*mygo.CloseEvent) {
				bounds := win.Bounds()
				_ = appBase.UpdateWindowBounds(bounds.X, bounds.Y, bounds.Width, bounds.Height)
			})
		}
		if wire != nil {
			// 订阅收到事件后重绘一帧；视图每帧从 Source 读事实。
			wire.OnChange = func() { win.Invalidate() }
			app.SetChanged(func() { win.Invalidate() })
			// 集成层目前没有"事实变化"回调：用一帧轮询把收到的流式事件画出来。
			// 这是显示刷新，不是第二份事实——每帧仍然只从 Source 读。
			go func() {
				ticker := time.NewTicker(250 * time.Millisecond)
				defer ticker.Stop()
				for range ticker.C {
					win.Invalidate()
				}
			}()
			// 连接在后台走；失败时视图显示原因（Connected/Connection 来自事实）。
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				if err := wire.Connect(ctx); err != nil {
					fmt.Fprintln(os.Stderr, "连接失败：", err)
				}
				win.Invalidate()
			}()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
