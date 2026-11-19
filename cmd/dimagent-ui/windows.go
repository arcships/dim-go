package main

// 多窗口与深链的落点：哪个窗口开在哪个会话、第二个窗口怎么开、深链来了落到
// 哪个窗口，都只在这里。
//
// 分工：
//
//	windowManager   窗口清单与路由（复用 / 聚焦 / 关闭 / 几何记忆）
//	windowSlot      一个窗口连同它自己的会话视图与标题取词处
//	mygo.NewWindow  只在 realCreate 里出现一次
//
// 一个窗口一份界面（appui.App）与会话视图（uiwire.Session）：选中哪个会话是
// 窗口自己的可丢状态；连接、目录、设置与缓存仍只有一份（见 internal/uiwire）。

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/deeplink"
	appui "github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/uiwire"
)

// windowSpec 是开一个窗口要看什么。
type windowSpec struct {
	// sessionID 是连接模式要打开的会话；空表示按默认（目录首项或当前会话）。
	// 样例模式里这里是样例会话名。
	sessionID string
	// settings 非空时直接进设置屏的该分节（深链 dimagent://settings/<分节>）。
	settings string
}

// windowSlot 是一个窗口连同它的视图。
type windowSlot struct {
	// index 是窗口索引：窗口偏好按它记忆几何，0 是主窗口。
	index int
	win   *mygo.Window
	view  *appui.App
	// session 是连接模式下本窗口的会话视图；样例模式为 nil。
	session *uiwire.Session
	titles  *windowTitle
	// sessionID 是样例模式（没有会话视图）下本窗口的会话名。
	sessionID string
}

// pinned 是本窗口当前看的会话：连接模式读会话视图（窗口内切换后跟着变），
// 样例模式读开窗时定的那一个。
func (s *windowSlot) pinned() string {
	if s.session != nil {
		return s.session.ID()
	}
	return s.sessionID
}

// windowManagerOptions 是窗口管理的依赖；全部由入口给，窗口管理自己不推导。
type windowManagerOptions struct {
	mode    runMode
	state   appui.State
	variant appui.Variant
	log     *startupLog
	// base 是集成层（连接模式）；样例模式为 nil。
	base *app.App
	// wire 是共享连接上的适配器；样例模式为 nil。
	wire *uiwire.App
	// rescan 是重连前重跑发现链；可为 nil。
	rescan func()
	// hint 是连接模式的启动说明（「未找到 App Server」之类）。
	hint string
	// logPath 是客户端日志路径（断线卡的「查看日志」要它）。
	logPath string
	// noFocus 表示窗口可见但不抢焦点（-no-focus，截图与后台调试用）。
	noFocus bool
	// width / height 是偏好里没有几何时的初始尺寸。
	width, height int
	// quitting 表示退出已经开始：关窗回调据此不再拦成最小化到托盘。
	quitting *atomic.Bool
	// onLastClosed 在最后一个窗口关闭时调用（入口按退出四阶段收尾）。
	onLastClosed func()
}

// windowManager 管所有窗口。窗口清单只在主线程上改（菜单、深链、关窗回调都在
// 主线程），锁只保护跨线程的读（托盘、通知、重绘轮询）。
type windowManager struct {
	opts windowManagerOptions

	mu     sync.Mutex
	slots  []*windowSlot
	active *windowSlot
	// pending 是还没能落窗的深链：窗口就绪之前到达的（系统拉起应用时先到
	// URL、后到 ready），以及目录还没读上来时先等着的。
	pending []string
	ready   bool
	// settled 表示连接阶段已经结束（成功或失败）：目录此时可用了。
	settled bool
	// deferredOpen 是启动参数里要「在新窗口中打开」的会话：目录还没读上来时先
	// 记下，连接结算之后再开。
	deferredOpen []string

	// host 是宿主外壳（托盘与通知），窗口就绪之后才接上；可为 nil。
	host *shell

	// create 建一个窗口；测试里换成不碰真实窗口的假实现。
	create func(spec windowSpec, index int) *windowSlot
}

// newWindowManager 建窗口管理。
func newWindowManager(opts windowManagerOptions) *windowManager {
	m := &windowManager{opts: opts}
	m.create = m.realCreate
	return m
}

// setHost 接上宿主外壳：托盘与通知的落点经过窗口管理。
func (m *windowManager) setHost(host *shell) { m.host = host }

// ---- 窗口清单 ----

// slotsSnapshot 是窗口清单的副本（从别的线程读时用）。
func (m *windowManager) slotsSnapshot() []*windowSlot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*windowSlot(nil), m.slots...)
}

// activeSlot 是最近聚焦的窗口。
func (m *windowManager) activeSlot() *windowSlot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active
}

// firstSlot 是索引最小的窗口：托盘与系统通知读它的会话（主窗口优先）。
func (m *windowManager) firstSlot() *windowSlot {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.slots) == 0 {
		return nil
	}
	return m.slots[0]
}

// mainSlot 是主窗口（索引 0）；它已经关掉时返回 nil。
func (m *windowManager) mainSlot() *windowSlot {
	for _, slot := range m.slotsSnapshot() {
		if slot.index == 0 {
			return slot
		}
	}
	return nil
}

// count 是打开着的窗口数。
func (m *windowManager) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.slots)
}

// nextIndex 是下一个窗口的索引：优先补关掉的窗口留下的槽位，让窗口偏好
// 按窗口索引继续用，而不是每开一次都往后长。
func (m *windowManager) nextIndex() int {
	used := map[int]bool{}
	for _, slot := range m.slotsSnapshot() {
		used[slot.index] = true
	}
	for index := 0; ; index++ {
		if !used[index] {
			return index
		}
	}
}

// open 打开一个新窗口并把它记进清单。
func (m *windowManager) open(spec windowSpec) *windowSlot {
	slot := m.create(spec, m.nextIndex())
	m.mu.Lock()
	m.slots = append(m.slots, slot)
	sort.SliceStable(m.slots, func(i, j int) bool { return m.slots[i].index < m.slots[j].index })
	m.active = slot
	m.mu.Unlock()
	return slot
}

// remove 把关掉的窗口移出清单。
func (m *windowManager) remove(slot *windowSlot) {
	m.mu.Lock()
	out := m.slots[:0]
	for _, item := range m.slots {
		if item != slot {
			out = append(out, item)
		}
	}
	m.slots = out
	if m.active == slot {
		m.active = nil
		if len(m.slots) > 0 {
			m.active = m.slots[0]
		}
	}
	m.mu.Unlock()
}

// focus 把一个窗口带到前台（还原、显示、聚焦应用）。
func (m *windowManager) focus(slot *windowSlot) {
	if slot == nil {
		return
	}
	m.mu.Lock()
	m.active = slot
	m.mu.Unlock()
	if slot.win == nil {
		return
	}
	if slot.win.IsDestroyed() {
		return
	}
	if slot.win.IsMinimized() {
		slot.win.Restore()
	}
	slot.win.Show()
	mygo.App.Focus()
	slot.win.Focus()
	slot.win.Invalidate()
}

// focusMain 把主窗口带到前台：托盘「显示主窗口」与全局快捷键的落点。
// 主窗口已经关掉时落到剩下的第一个窗口。
func (m *windowManager) focusMain() {
	if slot := m.mainSlot(); slot != nil {
		m.focus(slot)
		return
	}
	m.focus(m.firstSlot())
}

// hideMain 隐藏主窗口：托盘「隐藏主窗口」的落点。
func (m *windowManager) hideMain() {
	slot := m.mainSlot()
	if slot == nil || slot.win == nil || slot.win.IsDestroyed() {
		return
	}
	slot.win.Hide()
}

// invalidateAll 让所有窗口重绘：连接变化与流式事实都从这里上屏。
func (m *windowManager) invalidateAll() {
	for _, slot := range m.slotsSnapshot() {
		if slot.win != nil && !slot.win.IsDestroyed() {
			slot.win.Invalidate()
		}
	}
}

// applyTitles 让所有窗口按连接阶段刷新标题。
func (m *windowManager) applyTitles() {
	for _, slot := range m.slotsSnapshot() {
		slot.titles.apply()
	}
}

// freezeTitles 在退出一开始冻结所有窗口的标题。
func (m *windowManager) freezeTitles() {
	for _, slot := range m.slotsSnapshot() {
		slot.titles.freeze()
	}
}

// saveBounds 记下所有窗口的几何：退出与逐个关窗都写，下次按索引还原。
func (m *windowManager) saveBounds() {
	if m.opts.base == nil {
		return
	}
	for _, slot := range m.slotsSnapshot() {
		m.saveSlotBounds(slot)
	}
}

// saveSlotBounds 记下一个窗口的几何。
func (m *windowManager) saveSlotBounds(slot *windowSlot) {
	if m.opts.base == nil || slot.win == nil || slot.win.IsDestroyed() {
		return
	}
	bounds := slot.win.Bounds()
	if err := m.opts.base.UpdateWindowBoundsFor(slot.index, bounds.X, bounds.Y, bounds.Width, bounds.Height); err != nil {
		m.printf("窗口几何保存失败（第 %d 个窗口）：%v", slot.index+1, err)
	}
}

// ---- 会话窗口路由 ----

// findSession 找已经显示该会话的窗口。
func (m *windowManager) findSession(sessionID string) *windowSlot {
	if sessionID == "" {
		return nil
	}
	for _, slot := range m.slotsSnapshot() {
		if slot.pinned() == sessionID {
			return slot
		}
	}
	return nil
}

// knownSession 报告会话是否在目录里：深链与开窗之前先认一下。
func (m *windowManager) knownSession(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	if m.opts.wire != nil {
		for _, name := range m.opts.wire.SessionNames() {
			if name == sessionID {
				return true
			}
		}
		return m.findSession(sessionID) != nil
	}
	for _, name := range appui.SampleSessionNames(m.opts.state) {
		if name == sessionID {
			return true
		}
	}
	return false
}

// showSession 让一个已有的窗口换到某个会话（不新开窗口）。
func (m *windowManager) showSession(slot *windowSlot, sessionID string) {
	if slot == nil {
		return
	}
	if slot.session != nil {
		slot.session.SetPin(sessionID)
	}
	slot.sessionID = sessionID
	if slot.view != nil {
		slot.view.FocusSession(sessionID)
	}
	if slot.win != nil && !slot.win.IsDestroyed() {
		slot.win.Invalidate()
	}
}

// OpenSessionFromMenu 是「在新窗口中打开」的落点：已经有窗口显示该会话就切过去，
// 否则开一个新窗口。返回值是给用户看的回执。
func (m *windowManager) OpenSessionFromMenu(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("没有打开的会话")
	}
	if slot := m.findSession(sessionID); slot != nil {
		m.focus(slot)
		return "已切到打开该会话的窗口", nil
	}
	if !m.knownSession(sessionID) {
		return "", fmt.Errorf("会话不在目录里")
	}
	slot := m.open(windowSpec{sessionID: sessionID})
	m.focus(slot)
	m.printf("多窗口：新窗口打开会话 %s（第 %d 个窗口）", sessionID, slot.index+1)
	return "已在新窗口中打开", nil
}

// newSessionMain 在主窗口新建会话：托盘「新建会话」的落点。
func (m *windowManager) newSessionMain() {
	slot := m.mainSlot()
	if slot == nil {
		slot = m.firstSlot()
	}
	if slot == nil || slot.view == nil {
		return
	}
	slot.view.NewSession()
	m.focus(slot)
}

// FocusSession 是托盘「最近会话」与通知点开的落点：已经有窗口显示它就切过去，
// 否则在最近聚焦的窗口里换过去（没有窗口时开一个）。
func (m *windowManager) FocusSession(sessionID string) {
	if slot := m.findSession(sessionID); slot != nil {
		m.focus(slot)
		return
	}
	if slot := m.activeSlot(); slot != nil {
		m.showSession(slot, sessionID)
		m.focus(slot)
		return
	}
	m.focus(m.open(windowSpec{sessionID: sessionID}))
}

// ---- 深链 ----

// handleURL 处理一条深链（mygo.App.OnOpenURL 的落点）：切到主线程再处理，
// 窗口动作与视图状态都只在主线程上改。
func (m *windowManager) handleURL(raw string) {
	mygo.RunOnMain(func() { m.handleURLNow(raw) })
}

// handleURLNow 在主线程上处理一条深链。窗口还没就绪时先排队：系统拉起应用时
// URL 可能先于 ready 到达。
func (m *windowManager) handleURLNow(raw string) {
	m.mu.Lock()
	if !m.ready {
		m.pending = append(m.pending, raw)
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
	m.routeURL(raw)
}

// markReady 建好主窗口之后调用：把排队中的深链按到达顺序处理掉。
func (m *windowManager) markReady() {
	m.mu.Lock()
	m.ready = true
	pending := m.pending
	m.pending = nil
	m.mu.Unlock()
	for _, raw := range pending {
		m.routeURL(raw)
	}
}

// connectionSettled 在连接阶段结束（成功或失败）后调用：目录此时可用，排队中
// 的深链（含启动参数里的第二个窗口）再落窗。失败路径也要调：迟迟连不上时
// 「会话不在目录里」才是真的原因。
func (m *windowManager) connectionSettled() {
	m.mu.Lock()
	m.settled = true
	pending := m.pending
	m.pending = nil
	deferred := m.deferredOpen
	m.deferredOpen = nil
	m.mu.Unlock()
	for _, raw := range pending {
		m.routeURL(raw)
	}
	for _, sessionID := range deferred {
		m.openSessionInNewWindow(sessionID)
	}
	m.applyTitles()
	m.invalidateAll()
}

// openSessionInNewWindow 是启动参数 -new-window 的落点：按「在新窗口中打开」的
// 语义落窗（已经开在该会话的窗口切过去，不重复开）。目录还没读上来时先记下，
// 连接结算之后再开。
func (m *windowManager) openSessionInNewWindow(sessionID string) {
	if sessionID == "" {
		return
	}
	if m.waitingForCatalog() {
		m.mu.Lock()
		m.deferredOpen = append(m.deferredOpen, sessionID)
		m.mu.Unlock()
		m.printf("第二个窗口：目录还没读上来，等连接结算后再开（%s）", sessionID)
		return
	}
	receipt, err := m.OpenSessionFromMenu(sessionID)
	if err != nil {
		m.printf("第二个窗口：%v", err)
		if slot := m.activeSlot(); slot != nil && slot.view != nil {
			slot.view.ShowNotice("无法在新窗口中打开：" + err.Error())
		}
		return
	}
	m.printf("第二个窗口：%s（%s）", receipt, sessionID)
}

// waitingForCatalog 报告会话目录是不是还没读上来：连接模式的连接阶段还没
// 结算，且目录为空。这时深链先排队，不当成未知会话。
func (m *windowManager) waitingForCatalog() bool {
	if m.opts.wire == nil {
		return false
	}
	m.mu.Lock()
	settled := m.settled
	m.mu.Unlock()
	if settled {
		return false
	}
	return len(m.opts.wire.SessionNames()) == 0
}

// deferURL 把一条深链放回队列，等连接结算之后再落窗。
func (m *windowManager) deferURL(raw string) {
	m.mu.Lock()
	m.pending = append(m.pending, raw)
	m.mu.Unlock()
	m.printf("深链：%s → 目录还没读上来，等连接结算后再处理", raw)
}

// routeURL 把一条深链落到窗口上。
func (m *windowManager) routeURL(raw string) {
	target, err := deeplink.Parse(raw)
	if err != nil {
		m.report(err.Error())
		return
	}
	switch target.Kind {
	case deeplink.KindSession:
		if !m.knownSession(target.SessionID) {
			// 目录还没读上来时先别判成「会话不存在」：连接结算之后再处理。
			if m.waitingForCatalog() {
				m.deferURL(raw)
				return
			}
			m.report("链接里的会话不在目录里：" + target.SessionID)
			return
		}
		if slot := m.findSession(target.SessionID); slot != nil {
			m.focus(slot)
			m.printf("深链：%s → 切到已打开它的窗口", raw)
			return
		}
		if slot := m.activeSlot(); slot != nil {
			m.showSession(slot, target.SessionID)
			m.focus(slot)
			m.printf("深链：%s → 在现有窗口打开", raw)
			return
		}
		slot := m.open(windowSpec{sessionID: target.SessionID})
		m.focus(slot)
		m.printf("深链：%s → 新窗口", raw)
	case deeplink.KindSettings:
		slug := target.Section
		if slug == "" {
			slug = appui.DefaultSettingsSectionSlug
		}
		if !appui.SettingsSectionKnown(slug) {
			m.report("链接里的设置分节不存在：" + slug)
			return
		}
		if slot := m.activeSlot(); slot != nil && slot.view != nil {
			if err := slot.view.ShowSettingsSection(slug); err == nil {
				m.focus(slot)
				if slot.win != nil && !slot.win.IsDestroyed() {
					slot.win.Invalidate()
				}
				m.printf("深链：%s → 现有窗口进设置 › %s", raw, slug)
				return
			}
		}
		slot := m.open(windowSpec{settings: slug})
		m.focus(slot)
		m.printf("深链：%s → 新窗口进设置 › %s", raw, slug)
	}
}

// report 把深链的失败原因写给用户看：最近聚焦的窗口（没有窗口就开一个）的
// 回执行上写一行，同时写日志。
func (m *windowManager) report(reason string) {
	m.printf("深链处理失败：%s", reason)
	slot := m.activeSlot()
	if slot == nil {
		slot = m.open(windowSpec{})
		m.focus(slot)
	}
	if slot.view != nil {
		slot.view.ShowNotice(reason)
	}
	if slot.win != nil && !slot.win.IsDestroyed() {
		slot.win.Invalidate()
	}
}

// ---- 截图 ----

// capture 把每个窗口截一张图：第一个窗口用给定路径，其余按 `-2.png`、`-3.png`
// 顺序编号。返回实际写出的文件。
func (m *windowManager) capture(path string) ([]string, error) {
	var written []string
	for i, slot := range m.slotsSnapshot() {
		if slot.win == nil || slot.win.IsDestroyed() {
			continue
		}
		target := path
		if i > 0 {
			target = numberedPath(path, i+1)
		}
		data, err := slot.win.CapturePage()
		if err != nil {
			return written, fmt.Errorf("截图第 %d 个窗口：%w", i+1, err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return written, fmt.Errorf("写第 %d 个窗口的截图：%w", i+1, err)
		}
		written = append(written, target)
	}
	if len(written) == 0 {
		return nil, fmt.Errorf("没有可截图的窗口")
	}
	return written, nil
}

// numberedPath 给第 n 个窗口的截图加序号：out.png → out-2.png。
func numberedPath(path string, n int) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + fmt.Sprintf("-%d", n) + ext
}

// printf 写一行启动日志（nil 日志时静默）。
func (m *windowManager) printf(format string, args ...any) {
	if m.opts.log == nil {
		return
	}
	m.opts.log.printf(format, args...)
}

// ---- 真正的窗口 ----

// realCreate 建一个窗口。它是本文件里唯一调用 mygo.NewWindow 的地方。
func (m *windowManager) realCreate(spec windowSpec, index int) *windowSlot {
	slot := &windowSlot{index: index, sessionID: spec.sessionID}

	// 连接模式：一个窗口一份会话视图，接同一条连接与同一份缓存。
	var source appui.Source
	var intents appui.Intents
	if m.opts.wire != nil {
		slot.session = m.opts.wire.Pin(spec.sessionID)
		source = slot.session
		intents = retryingIntents{
			Session: slot.session,
			rescan:  m.opts.rescan,
			after:   m.applyTitles,
			log:     m.opts.log,
		}
	}

	titles := newWindowTitle(m.opts.mode, m.opts.log)
	if m.opts.wire != nil {
		titles.useSource(m.opts.wire)
		titles.setHint(m.opts.hint)
	}
	slot.titles = titles

	view := appui.NewApp(appui.Options{
		State:           m.opts.state,
		Variant:         m.opts.variant,
		Source:          source,
		Intents:         intents,
		LogPath:         m.opts.logPath,
		OpenInNewWindow: m.OpenSessionFromMenu,
		SaveFile: func(suggested string) (string, error) {
			return saveFileWithDialog(slot.win, suggested)
		},
		OnFrame: func(snapshot appui.HostSnapshot) {
			// 托盘与通知读索引最小的那个窗口的会话：多窗口时不会两帧互相顶掉。
			if slot != m.firstSlot() || m.host == nil {
				return
			}
			m.host.frame(snapshot)
		},
		StateChanged: func(s appui.State) {
			if m.opts.mode == modeConnect {
				// 连接模式的标题由连接阶段决定（见 windowTitle）。
				titles.apply()
				return
			}
			if slot.win != nil && !slot.win.IsDestroyed() {
				slot.win.SetTitle(titles.base(s))
			}
		},
	})
	slot.view = view
	// 深链与「在新窗口中打开」给的会话：开窗之前就定位到它（下一帧打开）。
	if spec.sessionID != "" {
		view.FocusSession(spec.sessionID)
	}
	if spec.settings != "" {
		if err := view.ShowSettingsSection(spec.settings); err != nil {
			m.printf("窗口设置分节：%v", err)
		}
	}

	width, height, x, y, hasPosition := m.windowGeometry(index)
	options := mygo.WindowOptions{
		Title:         titles.base(m.opts.state),
		Width:         width,
		Height:        height,
		MinWidth:      720,
		MinHeight:     600,
		TitleBarStyle: mygo.TitleBarHidden,
		Content:       ui.View(view.View),
	}
	if hasPosition {
		options.X, options.Y = x, y
	}
	slot.win = mygo.NewWindow(options)
	titles.attach(slot.win, options.Title)

	slot.win.OnClose(func(e *mygo.CloseEvent) {
		if m.quitting() {
			return
		}
		m.saveSlotBounds(slot)
		if m.count() == 1 && slot.index == 0 && m.closeToTray() {
			e.PreventDefault()
			slot.win.Hide()
			if m.host != nil {
				m.host.refresh()
			}
			m.printf("关窗：已按偏好最小化到托盘")
		}
	})
	slot.win.OnClosed(func() { m.windowClosed(slot) })
	slot.win.OnFocus(func() {
		m.mu.Lock()
		m.active = slot
		m.mu.Unlock()
	})

	if m.opts.noFocus {
		// -no-focus：窗口可见但不激活应用（截图与后台调试用）。
		slot.win.ShowInactive()
	}
	return slot
}

// saveFileWithDialog 是导出面板的系统保存面板：按建议路径的扩展名给过滤器，
// 窗口在时挂在窗口上（macOS 上是 sheet）。取消返回空路径、无错误。
func saveFileWithDialog(parent *mygo.Window, suggested string) (string, error) {
	opts := mygo.SaveDialogOptions{
		Parent:            parent,
		Title:             "导出对话",
		DefaultPath:       suggested,
		ButtonLabel:       "导出",
		CreateDirectories: true,
	}
	if ext := strings.TrimPrefix(filepath.Ext(suggested), "."); ext != "" {
		opts.Filters = []mygo.FileFilter{{Name: "会话导出（." + ext + "）", Extensions: []string{ext}}}
	}
	return mygo.Dialog.Save(opts)
}

// windowClosed 把一个窗口的关闭落到清单上：关一个不影响其他窗口；最后一个
// 关掉时按退出处理（没有窗口的进程留着只会让自启的 App Server 没人回收）。
func (m *windowManager) windowClosed(slot *windowSlot) {
	m.remove(slot)
	m.printf("关窗：第 %d 个窗口关闭（还剩 %d 个）", slot.index+1, m.count())
	if m.count() == 0 && !m.quitting() && m.opts.onLastClosed != nil {
		m.opts.onLastClosed()
	}
}

// windowGeometry 取某个窗口索引的初始几何：偏好里有的用偏好，没有的按主窗口
// 偏移（第二个窗口叠在主窗口右下方，截图与日常使用都不完全盖住）。
func (m *windowManager) windowGeometry(index int) (width, height, x, y int, hasPosition bool) {
	width, height = m.opts.width, m.opts.height
	hasPosition = false
	if m.opts.base != nil {
		if bounds := m.opts.base.Preferences().WindowFor(index); bounds.Width > 0 || bounds.HasPosition {
			if bounds.Width > 0 {
				width = bounds.Width
			}
			if bounds.Height > 0 {
				height = bounds.Height
			}
			x, y, hasPosition = bounds.X, bounds.Y, bounds.HasPosition
		}
	}
	if !hasPosition && index > 0 {
		if main := m.mainSlot(); main != nil && main.win != nil && !main.win.IsDestroyed() {
			bounds := main.win.Bounds()
			x, y = bounds.X+windowCascadeOffset, bounds.Y+windowCascadeOffset
			hasPosition = true
		}
	}
	return width, height, x, y, hasPosition
}

// windowCascadeOffset 是第二个窗口相对主窗口的偏移（逻辑像素）。
const windowCascadeOffset = 48

// quitting 报告退出是否已经开始。
func (m *windowManager) quitting() bool {
	return m.opts.quitting != nil && m.opts.quitting.Load()
}

// closeToTray 读关窗偏好；没有集成层（样例模式）时按关处理。
func (m *windowManager) closeToTray() bool {
	if m.opts.base == nil {
		return false
	}
	return m.opts.base.Preferences().CloseToTray
}
