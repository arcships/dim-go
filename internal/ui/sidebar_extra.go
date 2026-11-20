package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是左栏残留的补齐：栏头的品牌位与更新入口（LB-02–LB-09）、列表尾部的
// 工作区分页（LB-58/LB-59）、平铺视图的加载行与分页（LB-63–LB-65）、工作区
// 拖拽落点线（LB-60）与整栏拖入提示（LB-105/LB-106）。
//
// 事实口径：
//   - 更新相位与错误取可选的 UpdateFacts（与设置 › 关于同一份读数）；没有实现
//     它的 Source 不画更新按钮（不假装「已是最新」）。
//   - 「重启安装」需要宿主安装能力：Installable 为假时置灰，原因写在 tooltip。
//   - 会话目录一次读入（Source.SessionNames），没有分页读通道，所以平铺视图的
//     加载行只有在将来接上分页时才会出现；分页本身是本端按批展示。
//   - 拖入的路径是不是文件夹要读磁盘（os.Stat）；只有文件夹能登记成工作区。

// 左栏分页与拖入的度量。
const (
	// sidebarGroupPageSize 是分组视图一页放几个工作区分组（LB-58）。
	sidebarGroupPageSize = 8
	// sidebarFlatPageSize 是平铺视图默认展示的会话行数（LB-64）。
	sidebarFlatPageSize = 20
	// sidebarDropLineHeight 是工作区拖拽落点线的命中高度（线本身画 2）。
	sidebarDropLineHeight = 6
	// sidebarBrandMark 是栏头品牌图形的边长（参考品牌位 ≤80×52 内居中）。
	sidebarBrandMark = 16
	// sidebarUpdateHeight 是栏头更新按钮的行高（取行档 row-compact）。
	sidebarUpdateHeight = rowCompact
	// sidebarHintHeight 是拖入提示的主副两行高。
	sidebarHintHeight = 52
)

// sidebarProductName 是栏头品牌图形右侧的产品名（LB-03，取自参考产品的
// DESKTOP_PRODUCT_NAME）。
const sidebarProductName = "DimAgent"

// sidebarChromeIsMacOS 报告宿主是否 macOS：macOS 上栏头左端是窗口控件让位带
// （LB-01），其余平台画品牌图形与产品名（LB-02/LB-03）。做成变量是为了让测试
// 能切到非 macOS 分支——渲染结果仍只由平台决定。
var sidebarChromeIsMacOS = func() bool { return runtime.GOOS == "darwin" }

// sidebarBrand 是栏头左端的品牌图形与产品名（LB-02、LB-03）：只在非 macOS
// 出现；图形是单一标记（不带外框、不带文字），取当前前景色——浅色主题下是深色
// 标记，深色主题下是浅色标记。
func (a *App) sidebarBrand(c *gui.Context) {
	if sidebarChromeIsMacOS() {
		return
	}
	t := c.Theme()
	gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).Children(func() {
		// LB-02 品牌图形：不单独播报（产品名已经在旁边）。
		mark := gui.Box(c).Width(sidebarBrandMark).Height(sidebarBrandMark).Shrink(0).
			Label("品牌图形").AccessHidden(true)
		mark.DrawOver(func(p *gui.Painter, r gui.Rect) {
			paintIcon(p, iconBrand, r, t.Text)
		})
		// LB-03 产品名：静态文本，不是按钮。
		typeSecondaryStrong.style(t, gui.Text(c, sidebarProductName)).
			TextColor(t.Text).SingleLine().Role(gui.RoleText).Label(sidebarProductName)
	})
}

// sidebarUpdateAction 是栏头更新按钮按下后要发的动作。
type sidebarUpdateAction int

const (
	sidebarUpdateCheck sidebarUpdateAction = iota
	sidebarUpdateDownload
	sidebarUpdateInstall
)

// sidebarUpdateSpec 是栏头更新按钮的一帧：文案、图标、禁用与浮层内容。
type sidebarUpdateSpec struct {
	show     bool
	label    string
	icon     *gui.SVG
	spin     bool
	disabled bool
	danger   bool
	tip      string
	action   sidebarUpdateAction
}

// sidebarUpdate 按一帧更新读数收出按钮（LB-04–LB-09）。
//
// 相位到按钮：available →「更新」；downloading →「{pct}%」禁用；preparing →
// 「正在准备…」禁用并转圈；downloaded →「重启安装」（宿主没有安装能力时置灰）；
// 有错误消息时（下载或安装失败）→「重试更新」，取警示色。idle / checking /
// up_to_date 不在栏头挂按钮——「已是最新版本」属于设置 › 关于。
func (a *App) sidebarUpdate() sidebarUpdateSpec {
	facts, ok := a.source.(UpdateFacts)
	if !ok {
		return sidebarUpdateSpec{}
	}
	snap := facts.Update()
	if snap.ConfigError != "" {
		return sidebarUpdateSpec{}
	}
	version := snap.AvailableVersion
	if version == "" {
		version = aboutScreen.currentVersion
	}
	spec := sidebarUpdateSpec{show: true}
	status := ""
	switch snap.Phase {
	case "available":
		status = "有新版本可用"
		spec.label, spec.icon, spec.action = "更新", iconSidebarDownload, sidebarUpdateDownload
	case "downloading":
		status = fmt.Sprintf("下载中 %d%%", snap.Percent)
		spec.label, spec.disabled = fmt.Sprintf("%d%%", snap.Percent), true
	case "preparing":
		status, spec.label, spec.spin, spec.disabled = "正在准备", "正在准备…", true, true
	case "downloaded":
		status = "已下载，待重启"
		spec.label, spec.icon, spec.action = "重启安装", iconChevronDown, sidebarUpdateInstall
		spec.disabled = !snap.Installable
	default:
		return sidebarUpdateSpec{}
	}
	// 失败重试：有错误消息时按钮换成「重试更新」，重发失败的那一个动作。
	if snap.Err != "" && !spec.spin && snap.Phase != "downloading" {
		status = "更新失败"
		spec.label, spec.icon, spec.danger = "重试更新", iconSidebarDownload, true
		spec.action = sidebarUpdateDownload
		if snap.Phase == "downloaded" {
			spec.action = sidebarUpdateInstall
		}
	}
	// LB-09 浮层：状态 · 版本 · 错误消息（错误只在这里，不撑高栏头）。
	tip := status
	if version != "" {
		tip += " · v" + version
	}
	if snap.Err != "" {
		tip += " · " + snap.Err
	}
	if spec.disabled && spec.action == sidebarUpdateInstall {
		// 宿主没有安装能力时把原因一并写进浮层，不把按钮画成「点了会装」。
		tip += " · " + NoteUpdateInstall
	}
	spec.tip = tip
	return spec
}

// sidebarUpdateButton 画栏头更新入口（LB-04–LB-08）；浮层内容同时是按钮的
// 提示与描述（LB-09、LB-062）。
func (a *App) sidebarUpdateButton(c *gui.Context) {
	spec := a.sidebarUpdate()
	if !spec.show {
		return
	}
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(sidebarUpdateHeight).
		Padding(0, space2).Radius(radiusSmall).Gap(space1).AlignItems(gui.Center).
		Label(spec.label).Tooltip(spec.tip).Description(spec.tip).
		Cursor(gui.CursorPointer)
	if spec.disabled {
		b = b.Disabled(true).Opacity(noteDisabledAlpha)
	}
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	ink := t.Text
	if spec.danger {
		ink = t.Danger
	}
	if b.Clicked() && !spec.disabled {
		a.runSidebarUpdate(spec.action)
	}
	b.Children(func() {
		switch {
		case spec.spin:
			// 准备态用旋转图标：读屏只读文案，图标不单独播报。
			spinner := gui.Icon(c, iconSidebarSpinner).Size(iconInline, iconInline).
				TextColor(t.TextMuted).AccessHidden(true)
			spinner.Rotate(spinner.Loop("sidebar-update-spin", time.Second, gui.Linear) * 360)
		case spec.icon != nil:
			iconGlyph(c, spec.icon, ink, "更新入口")
		}
		typeSecondary.style(t, gui.Text(c, spec.label)).TextColor(ink).SingleLine()
	})
}

// runSidebarUpdate 把栏头更新按钮的交点交给更新动作；动作在后台跑，失败原因
// 由 UpdateFacts 的下一帧读数写回按钮与浮层，不在这里写假回执。
func (a *App) runSidebarUpdate(action sidebarUpdateAction) {
	update, ok := a.intents.(UpdateIntents)
	if !ok || a.intents == nil {
		return
	}
	switch action {
	case sidebarUpdateDownload:
		a.dispatch("下载更新", update.DownloadUpdate, nil)
	case sidebarUpdateInstall:
		a.dispatch("重启安装", update.InstallUpdate, nil)
	default:
		a.dispatch("检查更新", update.CheckUpdate, nil)
	}
}

// sidebarTextAction 是一行全宽文字按钮（LB-58 / LB-59 / LB-64 / LB-65）：28 高、
// 文本居中、无边框；没有图标时只画文字。
func (a *App) sidebarTextAction(c *gui.Context, label, tip string, icon *gui.SVG) bool {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(sidebarRowHeight).Grow(1).
		Radius(radiusSmall).Gap(space1).Justify(gui.Center).AlignItems(gui.Center).
		Label(label).Tooltip(tip).Cursor(gui.CursorPointer)
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	pressed := b.Clicked()
	b.Children(func() {
		if icon != nil {
			iconGlyph(c, icon, t.TextMuted, label)
		}
		typeSecondary.style(t, gui.Text(c, label)).TextColor(t.TextMuted).
			SingleLine().TextAlign(gui.Center)
	})
	return pressed
}

// sidebarGroupWindow 把分组视图的工作区分组裁成一页（LB-58 / LB-59）。
// 返回这一页的组、当前页（已夹到有效范围）与总页数。
func (a *App) sidebarGroupWindow(groups []sidebarWorkspace) ([]sidebarWorkspace, int, int) {
	pages := (len(groups) + sidebarGroupPageSize - 1) / sidebarGroupPageSize
	if pages <= 0 {
		pages = 1
	}
	page := a.sidebarView.groupPage
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	a.sidebarView.groupPage = page
	start := page * sidebarGroupPageSize
	end := start + sidebarGroupPageSize
	if end > len(groups) {
		end = len(groups)
	}
	return groups[start:end], page, pages
}

// sidebarGroupPager 是列表尾部的工作区分页（LB-58「显示更多」/ LB-59「上一页」）：
// 两处并存时同排，「显示更多」在后。没有更多页时两项都不出现。
func (a *App) sidebarGroupPager(c *gui.Context, page, pages int) {
	if pages <= 1 {
		return
	}
	gui.Row(c).Shrink(0).Gap(space2).AlignItems(gui.Center).Children(func() {
		if page > 0 {
			if a.sidebarTextAction(c, "上一页", "工作区列表：上一页", iconChevronL) {
				a.sidebarView.groupPage = page - 1
			}
		}
		if page < pages-1 {
			if a.sidebarTextAction(c, "显示更多", "工作区列表：显示更多", nil) {
				a.sidebarView.groupPage = page + 1
			}
		}
	})
}

// sidebarFlatLimit 是平铺视图当前展示的会话行数（LB-64 / LB-65）：默认阈值起，
// 点「显示更多」按批增加，点「显示更少」回到阈值。
func (a *App) sidebarFlatLimit(total int) int {
	limit := a.sidebarView.flatLimit
	if limit <= 0 {
		limit = sidebarFlatPageSize
	}
	if limit > total {
		limit = total
	}
	return limit
}

// sidebarFlatPager 是平铺视图尾部的分页：还有未展示的行时给「显示更多」，
// 全部展示且已经超过阈值时给「显示更少」（LB-64 / LB-65）。
func (a *App) sidebarFlatPager(c *gui.Context, limit, total int) {
	switch {
	case limit < total:
		if a.sidebarTextAction(c, "显示更多", "平铺视图：显示更多会话", iconChevronR) {
			a.sidebarView.flatLimit = limit + sidebarFlatPageSize
		}
	case limit > sidebarFlatPageSize:
		if a.sidebarTextAction(c, "显示更少", "平铺视图：显示更少会话", iconChevronDown) {
			a.sidebarView.flatLimit = sidebarFlatPageSize
		}
	}
}

// sidebarFlatLoadingRow 是平铺视图的加载行（LB-63）：居中一个旋转图标，没有
// 文字。会话目录一次读入、没有分页读通道，所以这一行只有在将来接上分页读取、
// 把 flatLoading 置真时才会出现。
func (a *App) sidebarFlatLoadingRow(c *gui.Context) {
	if !a.sidebarView.flatLoading {
		return
	}
	t := c.Theme()
	gui.Row(c).Shrink(0).Height(sidebarRowHeight).Justify(gui.Center).
		AlignItems(gui.Center).Label("正在加载更多").Children(func() {
		spinner := gui.Icon(c, iconSidebarSpinner).Size(iconInline, iconInline).
			TextColor(t.TextMuted).AccessHidden(true)
		spinner.Rotate(spinner.Loop("sidebar-flat-loading", time.Second, gui.Linear) * 360)
	})
}

// sidebarGroupDrag 是工作区行拖动的值：只带分组标识（拖动不搬目录）。
type sidebarGroupDrag struct{ id string }

// sidebarDropLine 是工作区拖拽的落点线（LB-60）：列表最前一条、最后一条。
// 平时透明（只有命中区，没有像素），拖着工作区停在上面时画 2px 强调色横线；
// 松手时把该组排到最前 / 最后。线不覆盖分组标题，也不进读屏。
func (a *App) sidebarDropLine(c *gui.Context, atTop bool) {
	t := c.Theme()
	label := "工作区拖拽落点：列表末尾"
	if atTop {
		label = "工作区拖拽落点：列表开头"
	}
	line := gui.Box(c).Shrink(0).Height(sidebarDropLineHeight).Grow(1).
		Label(label).AccessHidden(true)
	_, over := gui.DragOver[sidebarGroupDrag](line)
	line.DrawOver(func(p *gui.Painter, r gui.Rect) {
		if !over {
			return
		}
		p.Fill(gui.Rect{X: r.X, Y: r.Y + (r.H-2)/2, W: r.W, H: 2}, t.Accent, 0)
	})
	if drag, ok := gui.Drop[sidebarGroupDrag](line); ok {
		a.moveGroupTo(drag.id, atTop)
	}
}

// sidebarDroppedFiles 处理拖进左栏的文件（LB-105 / LB-106）：只有文件夹能登记
// 成工作区，登记后在该组下新建一个会话（副文案说的「同时新建会话」）。拖进来的
// 都不是文件夹时给一行失败原因，不留下半截提示。
func (a *App) sidebarDroppedFiles(files []string) {
	if len(files) == 0 {
		return
	}
	added := 0
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		// 先建会话再登记目录：样例模式下「新建会话」会把样例状态重建一遍，
		// 反过来的话刚登记的目录会被那次重建清掉。
		a.createSessionAt(filepath.Clean(path))
		a.addWorkspacePath(filepath.Clean(path))
		added++
	}
	if added == 0 {
		a.sidebarView.dragError = "只能拖入文件夹（文件不会被登记成工作区）"
		return
	}
	a.sidebarView.dragError = ""
}

// sidebarDragHint 是整栏拖入提示（LB-105 / LB-106）：文件夹拖到左栏列表上时
// 覆盖列表区，主文案「松开以添加工作区」+ 副文案「仅限文件夹 · 同时新建会话」；
// 提示不遮账户行（账户行在列表之外）。
func (a *App) sidebarDragHint(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Grow(1).MinHeight(sidebarHintHeight).Center().
		Padding(space3, space2).Gap(space1).Radius(radius).
		Background(t.SurfaceHover).
		Role(gui.RoleStatus).Label("松开以添加工作区").
		Children(func() {
			gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).Children(func() {
				iconGlyph(c, iconSidebarFolderPlus, t.TextMuted, "拖入工作区")
				typeSecondaryStrong.style(t, gui.Text(c, "松开以添加工作区")).
					TextColor(t.Text).SingleLine()
			})
			typeMicro.style(t, gui.Text(c, "仅限文件夹 · 同时新建会话")).
				TextColor(t.TextMuted).SingleLine().TextAlign(gui.Center)
		})
}

// sidebarDragLabel 是拖入失败原因的文本；空串表示没有失败。
func (a *App) sidebarDragLabel() string { return strings.TrimSpace(a.sidebarView.dragError) }
