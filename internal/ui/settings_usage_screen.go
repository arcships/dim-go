package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 用量（首屏）的重建实现：结构、文案、控件与间距按
// docs/domains/requirements/inventory-settings-rt.md §2（S-usage-01…74）与
// docs/domains/requirements/requirements-settings.md 的 R-USG-* 条目对齐，
// 视觉常量取 docs/domains/requirements/design-baseline.md。
//
// 只读：本屏不写任何事实，也没有可丢的界面状态。
//
// # 可达分支
//
// 参考实现 AccountActivity.vue 的分支链是
// appServerRead → usageUnavailable → loading → error → 数据区。本客户端是
// App Server 的宿主界面，"设置读 App Server 的用量"这一形态恒成立，因此
// **只读模式分支（appServerRead）恒真**，其余四个分支在本产品里不可达：
//
//	S-usage-06…09   需连接卡        （条件 !appServerRead）
//	S-usage-10…14   加载中 / 失败卡  （条件 !appServerRead）
//	S-usage-15…74   后台导入提示、
//	                 概览指标卡、Token 活跃度热力图、活跃度指标、Token 指标、
//	                 Cache 指标、最常用 skills、模型用量（条件 !appServerRead）
//
// 这些分支不是漏做：它们的前提（本机自己聚合用量）在本产品里不成立 ——
// 事实归 App Server，Go 侧不读它的库表、也不自行聚合（AGENTS.md §3）。
// 判据见清单 §2 末段：首屏只有 5 个可见元素（标题、卡、图标、分组标题、说明）。
func renderSettingsUsageScreen(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(usPageGap).Children(func() {
				usPageTitle(c, t)
				usUnavailableCard(c, t)
			})
		})
	})
}

// usPageTitle 是页面标题「用量」（S-usage-01）：与「账户」「个性化」同级的
// 二级标题，跟随内容滚出，不吸顶。
func usPageTitle(c *gui.Context, t *gui.Theme) {
	typePageTitle.style(t, gui.Text(c, usCopyTitle)).TextColor(t.Text).
		Role(gui.RoleHeading).Level(2)
}

// usUnavailableCard 是只读模式提示卡（S-usage-02…05）：警示卡内只有图标、
// 分组标题与一句说明，没有按钮；点击不响应。
//
// 参考实现：rounded-lg border p-6，内层 gap-3，图标 mt-0.5 h-5 w-5 取次文字色，
// 标题 text-sm font-medium，说明 text-sm 次文字色，二者 space-y-1。
func usUnavailableCard(c *gui.Context, t *gui.Theme) {
	card := accCardFrame(c, gui.Row(c).Shrink(0).Wrap().AlignItems(gui.Start).
		Gap(usIconGap).Padding(usCardPad))
	card.Role(gui.RoleStatus).Label(usCopyUnavailable)
	card.Children(func() {
		// 图标下移半档（mt-0.5）与标题首行对齐，不单独作为读屏节点。
		gui.Box(c).Shrink(0).Margin(usIconNudge, 0, 0, 0).Children(func() {
			accIcon(c, accIconTriangleAlert, usIconBlock, t.TextMuted, "triangle-alert")
		})
		gui.Column(c).Grow(1).MinWidth(0).Gap(usTextGap).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, usCopyUnavailable)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3)
			// 说明可选中复制（R-USG-037）。
			typeSecondary.style(t, gui.Text(c, usCopyAppServerOnly)).TextColor(t.TextMuted).
				Selectable()
		})
	})
}

// 本屏几何：取参考实现的 Tailwind 确定值，换算成 design-baseline §1.1 的
// 3.5 网格档位（p-6 = 21、gap-3 = 10.5、space-y-1 = 3.5、space-y-7 = 24.5）。
// 圆角与配色取本仓库的 token：卡框走 accCardFrame（Theme.Radius 6、t.Surface、
// 1px t.Border），与设置 › 账户同一条边界。
const (
	usPageGap   = 24.5 // space-y-7：标题与内容之间
	usCardPad   = 21   // p-6：卡内边距
	usIconGap   = 10.5 // gap-3：图标与文字块之间
	usTextGap   = 3.5  // space-y-1：分组标题与说明之间
	usIconBlock = 20   // h-5 w-5：警示图标
	usIconNudge = 2    // mt-0.5：图标相对首行下移半档
)

// 本屏文案：逐字取自参考产品的 zh-CN（i18n:usageSettings.title、
// i18n:plugins.unavailable、i18n:wipCapabilities.global_usage.reason）。
const (
	usCopyTitle         = "用量"
	usCopyUnavailable   = "不可用"
	usCopyAppServerOnly = "目前只有按会话的模型/用量；聚合结果不完整且不是权威数据。"
)
