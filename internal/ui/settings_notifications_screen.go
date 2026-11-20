package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 通知」的重建屏：元素按
// docs/domains/requirements/inventory-settings-rt.md §11（S-notifications-01 … 08，
// 8 个）与 inventory-settings.md §S-notifications（源码穷举 8 个，含两条 toast）
// 逐行对应；行为对应 docs/domains/requirements/requirements-settings.md 的
// R-NTF-001 … 064。
//
// 数值取 docs/domains/requirements/design-baseline.md（页面标题 17.5、开关 28×16、
// 卡片圆角 8.75、标签 13、说明 12.25）与运行时截图的实测（卡高 46、卡间距 22、
// 标题下 27、卡内左右 15 / 上下 14）。参考产品里本节没有分组标题、没有说明文案、
// 没有空态：三张卡都是「边框卡 + 标签 + 开关」，本屏照此。
//
// 三个开关是本地偏好（PreferencesSnapshot 的 NotifyTaskDone / NotifySound /
// NotifyNeedsResponse）：接上 Source 时读事实、改动经 Intents 落盘；样例模式没有
// 写入路径，只改进程内的界面状态。宿主的系统通知按同一份偏好决定发不发、带不带
// 声音（见 internal/notify 与 internal/tray）。
//
// # 未实现分支
//
//   - 加载 / 保存失败 toast（S-notifications-08）：本屏没有 toast 宿主；保存失败
//     由宿主外壳的通用错误行显示，不在这里弹一条单独的提示。
//   - 系统拒绝通知权限时的回滚（R-NTF-029）：回滚要有真实的通知授权结果；授权被
//     拒只写启动日志，不把开关拨回去。
//   - 语言包缺失的回落（R-NTF-005 / 022 / 038）：文案直接取参考产品中文，没有
//     i18n 层，因此不存在"缺键回落"这条支。
//   - 整行可点（R-NTF-020 的判据TBD）：参考实现的标签不是 label 元素，只有开关
//     可点；本屏与参考一致，标签不切换。

const (
	// 通知屏几何：design-baseline 只给常量，元素级包围盒取运行时 DOM 实测
	// （inventory-settings-rt.md §11 的 png 边框行：85 → 131.5 → 153.5 → 200）。
	notifyTitleGap = 27 // 页面标题下沿 → 首卡上沿（58 → 85）
	notifyCardH    = 46 // 一张卡的高度（85 → 131.5）
	notifyCardGap  = 22 // 卡与卡之间（131.5 → 153.5，中心距 68）
	notifyCardPadX = 15 // 卡内左右：标签 x=272 − 卡左 257
	notifyCardPadY = 14 // 卡内上下：卡高 46 = 18（行盒）+ 2×14

	notifyCopyTitle   = "通知"
	notifyCopyTask    = "任务完成或失败时通知"
	notifyCopySound   = "通知声音"
	notifyCopyNeedsMe = "需要我回应时通知"
)

// notifyLabelStyle 是开关卡的标签：R-NTF-018 判据的 13 字号、前景色、中粗。
var notifyLabelStyle = fontStep{size: 13.3, line: 18, weight: 500}

// renderNotifications 是设置 › 通知这一屏（设置壳的一个分节）。内容自己滚：
// 三张卡在矮窗口里放得下，但与其他分节保持同一条滚动契约。
func (a *App) renderNotifications(c *gui.Context) {
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Label(notifyCopyTitle).Children(func() {
			gui.Column(c).Shrink(0).Gap(notifyTitleGap).Children(func() {
				// S-notifications-01 页面标题「通知」
				screenTitle(c, notifyCopyTitle)
				gui.Column(c).Shrink(0).Gap(notifyCardGap).Children(func() {
					a.notifySwitchCard(c, notifyCopyTask, &a.notifyTaskDone)
					a.notifySwitchCard(c, notifyCopySound, &a.notifySound)
					// S-notifications-08 需要我回应时通知（开关）
					a.notifySwitchCard(c, notifyCopyNeedsMe, &a.notifyNeedsResponse)
				})
			})
		})
	})
}

// notifySwitchCard 是一张开关卡（S-notifications-02 … 07）：卡面（白 / 暗
// 0c0c0c）、1px 主题描边、卡片圆角；左侧标签，右侧 28×16 开关。卡本身是分组
// （R-NTF-013：每卡以分组播报），但卡面不可点（R-NTF-012：点卡空白无响应）。
//
// 开关变化后落盘一次本地偏好；比较由 persistPreferences 挡掉重复写入。
func (a *App) notifySwitchCard(c *gui.Context, label string, on *bool) {
	t := c.Theme()
	card := gui.Column(c).Shrink(0).Background(t.Surface).
		Radius(screenRadiusCard).Role(gui.RoleGroup).Label(label)
	card.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
	})
	card.Children(func() {
		gui.Row(c).Shrink(0).MinHeight(notifyCardH).
			Padding(notifyCardPadY, notifyCardPadX).
			AlignItems(gui.Center).Children(func() {
			notifyLabelStyle.style(t, gui.Text(c, label)).TextColor(t.Text).
				Label(label).SingleLine()
			gui.Spacer(c)
			sw := gui.Switch(c, on).Size(screenSwitchW, screenSwitchH).
				Radius(screenSwitchH / 2).Label(label + "开关")
			if sw.Changed() {
				a.persistPreferences()
			}
		})
	})
}
