// Command dimagent-ui 是客户端入口：默认按连接模式启动，不带参数也能起来。
//
//	DimAgent.app                                 双击：默认数据目录 + 自动发现 App Server
//	dimagent-ui -server <可执行文件> -mock-model  指定服务端（验收用）
//	dimagent-ui -demo                            样例数据，不连服务端（截图与走查）
//
// 连接模式的事实全部来自 App Server（经 internal/uiwire 的 Source）；样例模式只用
// 静态样例画界面骨架。两种模式共用同一棵视图，差别仅在有没有 Source。
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

// retryingIntents 把「重新连接」接到完整启动序列上。
//
// 启动阶段就失败时（找不到服务端、服务端没起来），集成层还没有连接监督者，
// 单纯重连只会得到「未连接服务端」；这时补一次连接与订阅才是真正的重试。
// 其余意图原样交给适配器，不新增任何协议通道。
type retryingIntents struct {
	*uiwire.App
	connect func(context.Context) error
}

// Reconnect 先走常规重连；连不上时重走一遍连接与订阅。
func (r retryingIntents) Reconnect(ctx context.Context) error {
	if err := r.App.Reconnect(ctx); err == nil {
		return nil
	}
	return r.connect(ctx)
}

// 启动时序。连接模式第一次运行要等 App Server 拉起并握手，超时按启动失败处理。
const (
	// demoShotDelay 是样例模式截图的等待：静态样例第一帧就够。
	demoShotDelay = 900 * time.Millisecond
	// shotSettle 是连接模式截图在连接结束后的等待：让首屏事实先画上来。
	shotSettle = 1500 * time.Millisecond
	// connectTimeout 是连接（含自启服务端）的上限。
	connectTimeout = 90 * time.Second
	// shutdownTimeout 是退出时回收自启实例的上限。
	shutdownTimeout = 10 * time.Second
)

// defaultVersion 是兜底版本号；打包时用 `-ldflags -X main.version=` 覆盖。
const defaultVersion = "0.1.0"

// version 只用于诊断日志。
var version = defaultVersion

// runMode 是两种启动模式：连接模式（默认）与样例模式（-demo）。
type runMode int

const (
	modeConnect runMode = iota
	modeDemo
)

func main() {
	demoFlag := flag.Bool("demo", false,
		"样例数据模式：不连 App Server，只用静态样例画界面骨架（截图与走查用）")
	// -connect 是旧参数：连接模式已是默认，接受但不改变行为。
	_ = flag.Bool("connect", false, "兼容旧参数：连接模式已是默认，可省略")
	stateFlag := flag.String("state", "default",
		"样例模式的起始状态：default、running、waiting、failed、empty（也可写 1–5）")
	dataDirFlag := flag.String("data-dir", "",
		"客户端数据目录；默认 DIM_DATA_DIR，否则 macOS 的产品目录 ~/Library/Application Support/DimAgent")
	serverFlag := flag.String("server", "",
		"App Server 可执行文件；省略时按 DIM_APP_SERVER → 可执行文件同目录 → PATH → 已有端点 的顺序发现")
	mockFlag := flag.Bool("mock-model", false, "连接模式下用 mock 模型拉起服务端（验收用）")
	variantFlag := flag.String("variant", "",
		"额外的走查态：branch（分支菜单）、goal（目标编辑）、tasks（后台任务）、settings（设置）、settings-rules（清空确认）、ext / ext-failed / ext-empty / ext-loading（扩展四态）")
	widthFlag := flag.Int("width", appui.WindowWidth, "窗口宽度（逻辑像素，用于三档宽度对照截图）")
	heightFlag := flag.Int("height", appui.WindowHeight, "窗口高度（逻辑像素）")
	noFocusFlag := flag.Bool("no-focus", false,
		"后台运行（accessory 策略）：窗口可见但不抢焦点、不进 Dock 与菜单栏；自动截图与调试用")
	shotFlag := flag.String("screenshot", "",
		"渲染一帧后把窗口截到该 PNG 路径并退出；配合 -no-focus 可在不抢焦点的情况下取图")
	logPathFlag := flag.String("log", "",
		"客户端启动日志路径；默认 <数据目录>/logs/dimagent.log")
	flag.Parse()

	state, err := appui.ParseState(*stateFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	variant, err := appui.ParseVariant(*variantFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	mode := modeConnect
	if *demoFlag {
		mode = modeDemo
	}
	// 样例状态只属于样例模式：连接模式的事实由服务端给。
	if mode == modeConnect {
		state = appui.StateDefault
	}

	dataDir := ""
	if mode == modeConnect {
		resolved, err := resolveDataDir(*dataDirFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "解析数据目录：", err)
			os.Exit(1)
		}
		dataDir = resolved
	}

	logPath := *logPathFlag
	if logPath == "" && dataDir != "" {
		logPath = defaultLogPath(dataDir)
	}
	var startup *startupLog
	if logPath != "" {
		startup = openStartupLog(logPath)
		defer startup.close()
	}

	// 标题：样例模式保留「界面骨架」字样（截图脚本按它认窗口），连接模式用产品名。
	serverHint := ""
	title := func(s appui.State) string {
		if mode == modeDemo {
			return "DimAgent — " + s.String() + "（界面骨架）"
		}
		if serverHint != "" {
			return "DimAgent — " + serverHint
		}
		return "DimAgent"
	}

	// 连接模式（默认）从真实集成层读事实；-demo 用样例数据。
	var source appui.Source
	var intents appui.Intents
	var wire *uiwire.App
	var appBase *app.App
	if mode == modeConnect {
		startup.printf("启动：版本 %s 模式 连接 数据目录 %s", version, dataDir)
		plan, planErr := findServer(*serverFlag, dataDir)
		switch {
		case planErr != nil && plan.Explicit:
			// 用户指了地方却不可用：立即失败并写清原因，不悄悄换成别的服务端。
			startup.printf("App Server 不可用：%v", planErr)
			fmt.Fprintln(os.Stderr, "启动失败：", planErr)
			os.Exit(1)
		case planErr != nil:
			// 自动发现没结果：仍然起窗口，让断线说明在原位给原因与重试入口。
			serverHint = "未找到 App Server"
			startup.printf("%v", planErr)
		default:
			startup.printf("App Server：%s", plan.describe())
			startup.printf("服务端点：%s", plan.Endpoint)
		}
		base, err := app.New(app.Config{
			DataDir:    dataDir,
			ServerPath: plan.Path,
			MockModel:  *mockFlag,
			ClientName: "dimagent-ui",
			Version:    version,
		})
		if err != nil {
			startup.printf("创建集成层失败：%v", err)
			fmt.Fprintln(os.Stderr, "创建集成层：", err)
			os.Exit(1)
		}
		appBase = base
		wire = uiwire.New(base)
		source = wire
		intents = retryingIntents{App: wire, connect: wire.Connect}
		startup.printf("日志：%s", startup.Path())
	}

	var win *mygo.Window
	app := appui.NewApp(appui.Options{
		Source:  source,
		Intents: intents,
		State:   state,
		Variant: variant,
		// 断线卡的「查看日志」用它打开客户端日志；样例模式没有日志时为空串。
		LogPath: startup.Path(),
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
		// -no-focus：切到 accessory 策略，窗口仍可见但不激活应用、不占 Dock 与菜单栏，
		// 供自动截图与后台调试使用（人还要在前台干别的）。必须在 ready 之后调用。
		if *noFocusFlag {
			mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
		}
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

		// 连接完成的信号：截图与标题都等它，免得截到"连接中"的一帧。
		connectDone := make(chan struct{})
		if wire == nil {
			close(connectDone)
		} else {
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
			// 连接在后台走；失败时视图在自己的位置显示原因（Connected/Connection
			// 来自事实），标题同时点明，不让窗口看起来像"白屏"。
			go func() {
				defer close(connectDone)
				ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
				defer cancel()
				started := time.Now()
				if err := wire.Connect(ctx); err != nil {
					startup.printf("连接失败（%.0f ms）：%v", elapsedMS(started), err)
					win.SetTitle("DimAgent — 连接失败")
				} else {
					startup.printf("连接就绪（%.0f ms）", elapsedMS(started))
					win.SetTitle(title(appui.StateDefault))
				}
				win.Invalidate()
			}()
		}

		if *shotFlag != "" {
			capture := func() {
				data, err := win.CapturePage()
				if err != nil {
					startup.printf("截图失败：%v", err)
				} else if err := os.WriteFile(*shotFlag, data, 0o644); err != nil {
					startup.printf("写截图失败：%v", err)
				}
				mygo.App.Quit()
			}
			if mode == modeDemo {
				time.AfterFunc(demoShotDelay, capture)
			} else {
				go func() {
					select {
					case <-connectDone:
					case <-time.After(connectTimeout + 2*time.Second):
					}
					time.Sleep(shotSettle)
					capture()
				}()
			}
		}

		if appBase != nil {
			win.OnClose(func(*mygo.CloseEvent) {
				bounds := win.Bounds()
				_ = appBase.UpdateWindowBounds(bounds.X, bounds.Y, bounds.Width, bounds.Height)
				// 双击启动的应用关掉窗口就该退出：留着没有窗口的进程只会让自启的
				// App Server 落到没人回收的地方。
				mygo.App.Quit()
			})
		}
	})

	// 退出时按退出四阶段收尾并回收自己拉起的服务端；只回收空闲实例，别的客户端
	// 还在用就原样留下。
	if appBase != nil {
		var quitting bool
		mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
			if quitting {
				return
			}
			quitting = true
			e.PreventDefault()
			go func() {
				shutdown(appBase, startup)
				mygo.App.Quit()
			}()
		})
	}

	if err := mygo.App.Run(); err != nil {
		startup.printf("事件循环退出：%v", err)
		log.Fatal(err)
	}
}

// shutdown 按退出四阶段收尾：停意图 → 结束交互 → 关连接 → 回收自启实例。
//
// 每一步的失败都只记日志：退出路径不因为某一步失败而卡住。
func shutdown(base *app.App, startup *startupLog) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	started := time.Now()
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"停意图", base.StopIntents},
		{"结束交互", base.EndInteraction},
		{"关连接", base.CloseConnection},
		{"回收服务", base.ReclaimService},
	}
	for _, step := range steps {
		if err := step.fn(ctx); err != nil {
			startup.printf("退出 %s 失败：%v", step.name, err)
			continue
		}
		startup.printf("退出 %s 完成", step.name)
	}
	startup.printf("退出完成（%.0f ms）", elapsedMS(started))
}

// elapsedMS 是毫秒耗时，只用于日志。
func elapsedMS(since time.Time) float64 {
	return float64(time.Since(since).Microseconds()) / 1000
}
