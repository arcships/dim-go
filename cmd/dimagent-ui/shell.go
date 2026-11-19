package main

import (
	"fmt"
	"sync/atomic"

	"github.com/egoist/mygo"

	"github.com/arcships/dimagent-mygo/internal/app"
	appui "github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/notify"
	"github.com/arcships/dimagent-mygo/internal/tray"
)

// shell 是宿主外壳：菜单栏托盘图标与系统通知。
//
// 它只做转接：菜单项、图标状态与通知策略由 internal/tray、internal/notify 计算，
// 这里把它们接到 mygo 的托盘与 Notification 上，并把用户动作交回界面（appui.App）
// 与窗口。事实仍只从 HostSnapshot（每帧由界面从 Source 现读）与本地偏好来。
type shell struct {
	ui   *appui.App
	base *app.App
	log  *startupLog

	win  *mygo.Window
	icon *mygo.Tray

	// watcher 与 notifier 只在连接模式（有集成层）时建：样例模式没有真实事件。
	watcher  *notify.Watcher
	notifier *notify.Notifier

	// quit 是托盘「退出」的落地（入口按退出四阶段收尾后放行 Quit）。
	quit func()
	// quitting 表示退出已经开始：关窗回调不再把它拦成「最小化到托盘」。
	quitting *atomic.Bool

	// 上一次安装的菜单指纹与图标状态：只在变化时重装，免得每帧重建系统菜单。
	menuSignature string
	iconStatus    tray.Status
	tooltip       string
	// last 是最近一帧的会话快照：托盘动作改了窗口可见性后立刻重装菜单。
	last appui.HostSnapshot
	// notifiedUnsupported 让「系统通知不可用」只记一次日志。
	notifiedUnsupported atomic.Bool
}

// newShell 建宿主外壳；base 可为 nil（样例模式，只有托盘、没有通知）。
func newShell(ui *appui.App, base *app.App, log *startupLog) *shell {
	s := &shell{ui: ui, base: base, log: log, iconStatus: tray.StatusIdle, tooltip: tray.Tooltip(tray.StatusIdle)}
	if base != nil {
		s.watcher = notify.NewWatcher()
		s.notifier = notify.NewNotifier(s, s.focusSession, s.printf)
	}
	return s
}

// attachWindow 记下主窗口（托盘动作与通知点开都要把它带到前台）。
func (s *shell) attachWindow(win *mygo.Window) { s.win = win }

// createTray 建菜单栏图标。必须在应用就绪之后调用。
func (s *shell) createTray() error {
	icon, err := tray.Icon(tray.StatusIdle)
	if err != nil {
		return fmt.Errorf("托盘图标: %w", err)
	}
	item, err := mygo.NewTray(mygo.TrayOptions{
		ID:             "dimagent",
		Icon:           icon,
		IconIsTemplate: true,
		ToolTip:        s.tooltip,
	})
	if err != nil {
		return err
	}
	s.icon = item
	// 首帧之前先装一份菜单：图标一出现就能点，不用等下一次状态变化。
	s.frame(appui.HostSnapshot{})
	return nil
}

// frame 在每帧视图之后刷新托盘，并把会话状态交给通知观察器。
func (s *shell) frame(snapshot appui.HostSnapshot) {
	s.last = snapshot
	s.syncTray(snapshot)
	if s.watcher == nil || s.notifier == nil || s.base == nil {
		return
	}
	prefs := s.base.Preferences()
	events := s.watcher.Observe(notify.Session{
		ID:              snapshot.SessionID,
		Title:           snapshot.SessionTitle,
		Running:         snapshot.Running,
		Pending:         snapshot.Pending,
		PendingID:       snapshot.PendingID,
		LastTurnEnded:   snapshot.LastTurnEnded,
		LastTurnIndex:   snapshot.LastTurnIndex,
		LastTurnOutcome: snapshot.LastTurnOutcome,
	}, notify.Prefs{
		TaskDone:      prefs.NotifyTaskDone,
		Sound:         prefs.NotifySound,
		NeedsResponse: prefs.NotifyNeedsResponse,
	})
	for _, event := range events {
		event := event
		// 发通知要等系统交付，不能占住渲染这一帧。
		go func() { _ = s.notifier.Notify(event) }()
	}
}

// refresh 用最近一帧的事实重装托盘菜单：窗口显示/隐藏与偏好刚变过时立即反映，
// 不等下一次渲染。
func (s *shell) refresh() { s.syncTray(s.last) }

// syncTray 按真实状态重装菜单、图标与悬停文字；没有变化就不碰系统菜单。
func (s *shell) syncTray(snapshot appui.HostSnapshot) {
	if s.icon == nil {
		return
	}
	visible := s.win != nil && s.win.IsVisible()
	state := tray.State{
		WindowVisible:   visible,
		Running:         snapshot.Running,
		WaitingApproval: snapshot.Pending,
		Recent:          recentSessions(snapshot.Recent),
		CloseToTray:     s.closeToTray(),
	}
	items := tray.Menu(state)
	if signature := tray.Signature(items); signature != s.menuSignature {
		s.menuSignature = signature
		s.icon.SetMenu(s.trayMenu(items))
	}
	status := tray.StatusOf(state)
	if status != s.iconStatus {
		s.iconStatus = status
		if icon, err := tray.Icon(status); err != nil {
			s.printf("托盘图标切换失败：%v", err)
		} else if err := s.icon.SetIcon(icon, true); err != nil {
			s.printf("托盘图标切换失败：%v", err)
		}
	}
	if tooltip := tray.Tooltip(status); tooltip != s.tooltip {
		s.tooltip = tooltip
		s.icon.SetToolTip(tooltip)
	}
}

// trayMenu 把托盘模型转成 mygo 菜单；每个动作都在这里落到窗口或界面。
func (s *shell) trayMenu(items []tray.Item) *mygo.Menu {
	out := make([]*mygo.MenuItem, 0, len(items))
	for _, item := range items {
		if item.Separator {
			out = append(out, mygo.Separator())
			continue
		}
		entry := &mygo.MenuItem{Label: item.Label, Disabled: item.Disabled}
		if item.Action == tray.ActionToggleCloseToTray {
			entry.Type = mygo.MenuItemCheckbox
			entry.Checked = item.Checked
		}
		if item.Action != "" {
			action := item.Action
			sessionID := item.SessionID
			entry.Click = func(*mygo.MenuItem, *mygo.Window) { s.run(action, sessionID) }
		}
		out = append(out, entry)
	}
	return mygo.NewMenu(out)
}

// run 执行一个托盘动作。
func (s *shell) run(action tray.Action, sessionID string) {
	switch action {
	case tray.ActionShowWindow:
		s.showWindow()
	case tray.ActionHideWindow:
		if s.win != nil {
			s.win.Hide()
			s.refresh()
		}
	case tray.ActionNewSession:
		s.showWindow()
		if s.ui != nil {
			s.ui.NewSession()
		}
	case tray.ActionOpenSession:
		s.showWindow()
		if s.ui != nil {
			s.ui.FocusSession(sessionID)
		}
	case tray.ActionToggleCloseToTray:
		s.toggleCloseToTray()
	case tray.ActionQuit:
		if s.quitting != nil {
			s.quitting.Store(true)
		}
		if s.quit != nil {
			s.quit()
		}
	}
}

// showWindow 把主窗口带到前台：还原、显示、聚焦，应用也切到前台。
func (s *shell) showWindow() {
	if s.win == nil {
		return
	}
	if s.win.IsMinimized() {
		s.win.Restore()
	}
	s.win.Show()
	mygo.App.Focus()
	s.win.Focus()
	s.win.Invalidate()
	s.refresh()
}

// focusSession 是通知点开的落点：窗口到前台，界面定位到该会话。
func (s *shell) focusSession(sessionID string) {
	s.showWindow()
	if s.ui != nil {
		s.ui.FocusSession(sessionID)
	}
	if s.win != nil {
		s.win.Invalidate()
	}
}

// Show 实现 notify.Presenter：把一条通知交给系统的 Notification。
func (s *shell) Show(note notify.Note) error {
	if !mygo.NotificationsSupported() {
		if !s.notifiedUnsupported.Swap(true) {
			s.printf("系统通知不可用：当前运行方式不支持桌面通知（macOS 需要打包后的应用）")
		}
		return fmt.Errorf("系统通知不可用")
	}
	item := mygo.NewNotification(mygo.NotificationOptions{
		ID:       note.ID,
		Title:    note.Title,
		Subtitle: note.Subtitle,
		Body:     note.Body,
		Group:    note.Group,
		Silent:   note.Silent,
	})
	onClick := note.OnClick
	item.OnClick(func() {
		if onClick != nil {
			onClick()
		}
	})
	return item.Show()
}

// closeToTray 读关窗偏好；没有集成层（样例模式）时按关处理。
func (s *shell) closeToTray() bool {
	if s.base == nil {
		return false
	}
	return s.base.Preferences().CloseToTray
}

// toggleCloseToTray 切换并落盘关窗偏好。
func (s *shell) toggleCloseToTray() {
	if s.base == nil {
		return
	}
	prefs := s.base.Preferences()
	prefs.CloseToTray = !prefs.CloseToTray
	if err := s.base.UpdatePreferences(prefs); err != nil {
		s.printf("保存关窗偏好失败：%v", err)
		return
	}
	s.printf("关窗偏好：%s", onOff(prefs.CloseToTray))
	s.refresh()
	if s.win != nil {
		s.win.Invalidate()
	}
}

// recentSessions 把界面的最近会话换成托盘模型的形状，并截到前 5 条。
func recentSessions(items []appui.HostSession) []tray.RecentSession {
	limit := tray.RecentLimit
	if len(items) < limit {
		limit = len(items)
	}
	out := make([]tray.RecentSession, 0, limit)
	for _, item := range items[:limit] {
		out = append(out, tray.RecentSession{ID: item.ID, Title: item.Title})
	}
	return out
}

// printf 写一行启动日志（nil 日志时静默）。
func (s *shell) printf(format string, args ...any) {
	if s.log == nil {
		return
	}
	s.log.printf(format, args...)
}

// onOff 把开关写成日志里看得懂的词。
func onOff(on bool) string {
	if on {
		return "关窗进托盘"
	}
	return "关窗即退出"
}
