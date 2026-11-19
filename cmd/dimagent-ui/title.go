package main

// 窗口标题：样例模式按界面状态取，连接模式按连接阶段取。
//
// 连接模式的标题必须跟着阶段变化刷新，不能只在启动协程里设一次：启动时连不上会写
// 「连接失败」，装上服务端重连成功后要回到产品标题，运行中断线又得改回失败。
// 标题只从 Source 的连接阶段推出来，不另存一份连接状态。

import (
	"sync"

	"github.com/egoist/mygo"

	appui "github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/uiwire"
)

const (
	// titleProduct 是连接模式的基准标题；连上之后就是这个。
	titleProduct = "DimAgent"
	// titleConnectFailed 是断线时的标题：窗口不能看起来像「白屏」。
	titleConnectFailed = "DimAgent — 连接失败"
)

// 连接阶段名，取值与 internal/ui 的 Connection.Phase 一致。
const (
	connectionPhaseConnected    = "connected"
	connectionPhaseDisconnected = "disconnected"
)

// windowTitle 是窗口标题的取词处：连接模式看连接阶段，样例模式看界面状态。
type windowTitle struct {
	mu sync.Mutex
	// mode 是启动模式：样例模式的标题带「界面骨架」，截图脚本按它认窗口。
	mode runMode
	// hint 是连接模式的启动说明（「未找到 App Server」之类），连上之后不再算数。
	hint   string
	window *mygo.Window
	source *uiwire.App
	// log 记录标题变更，便于从日志判断标题何时跟着连接变了；可为 nil。
	log *startupLog
	// last 是最后写进窗口的标题：没变就不动窗口。
	last string
	// frozen 表示退出已经开始：此后不再改标题，别在退出途中翻成「连接失败」。
	frozen bool
}

// newWindowTitle 建标题取词处；事实来源在连接模式里随后补上（见 useSource）。
func newWindowTitle(mode runMode, log *startupLog) *windowTitle {
	return &windowTitle{mode: mode, log: log}
}

// useSource 接上事实来源：连接阶段从它读。
func (t *windowTitle) useSource(source *uiwire.App) {
	t.mu.Lock()
	t.source = source
	t.mu.Unlock()
}

// setHint 记下启动说明：连接模式找不到 App Server 时，标题要说明原因。
func (t *windowTitle) setHint(hint string) {
	t.mu.Lock()
	t.hint = hint
	t.mu.Unlock()
}

// base 是还没有连接阶段可依据时的标题（启动第一帧与样例模式都用它）。
func (t *windowTitle) base(s appui.State) string {
	if t == nil {
		return titleProduct
	}
	t.mu.Lock()
	mode, hint := t.mode, t.hint
	t.mu.Unlock()
	if mode == modeDemo {
		return titleProduct + " — " + s.String() + "（界面骨架）"
	}
	if hint != "" {
		return titleProduct + " — " + hint
	}
	return titleProduct
}

// forConnection 按连接阶段取标题：连上回到产品标题，断线点明连接失败，
// 连接中保留启动标题（不提前说失败）。
func (t *windowTitle) forConnection(conn appui.Connection) string {
	switch conn.Phase {
	case connectionPhaseConnected:
		// 连上之后，启动时那条「未找到 App Server」的说明就不成立了。
		return titleProduct
	case connectionPhaseDisconnected:
		return titleConnectFailed
	default:
		return t.base(appui.StateDefault)
	}
}

// attach 记下窗口与它的初始标题（NewWindow 的 Title）。
func (t *windowTitle) attach(win *mygo.Window, initial string) {
	t.mu.Lock()
	t.window = win
	t.last = initial
	t.mu.Unlock()
}

// freeze 之后不再改标题：退出已经开始了。
func (t *windowTitle) freeze() {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.frozen = true
	t.mu.Unlock()
}

// apply 按当前连接阶段写一次标题；标题没变时不动窗口，也不写日志。
//
// 失败路径（连接/重连失败）不广播事实变化，所以除了 OnChange 之外，首次连接结束
// 与每次重连结束都显式调用一次；250ms 的重绘帧再补一次兜底。
func (t *windowTitle) apply() {
	if t == nil {
		return
	}
	t.mu.Lock()
	win, source, frozen := t.window, t.source, t.frozen
	t.mu.Unlock()
	if frozen || win == nil || source == nil {
		return
	}
	conn := source.Connection()
	wanted := t.forConnection(conn)
	t.mu.Lock()
	changed := wanted != t.last && !t.frozen
	if changed {
		t.last = wanted
	}
	t.mu.Unlock()
	if !changed {
		return
	}
	win.SetTitle(wanted)
	t.log.printf("窗口标题：%s（连接阶段 %s）", wanted, conn.Phase)
}
