package ui

import (
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是左栏重建的度量、配色、图标与几个小件（状态槽、标签、键帽、折叠箭头）。
//
// 度量取自 design-baseline 的常量档：侧栏内容左右内边距 7（p-2）；列表行 ·
// 侧栏菜单 28；项目行 21；会话行 28（DOM 实测 main/session.json 的 7,255 346×28）；
// 账户行 30；图标 16 / 14；账户头像 18。这些是左栏自己的尺寸，写在左栏文件里，
// 不进全局 token 表（tokens.go 归 U2）。
const (
	// sidebarRowHeight 是入口行（新建 / 搜索 / 定时 / 技能）与会话行的高度。
	sidebarRowHeight = 28
	// sidebarGroupRowHeight 是工作区行的高度（DOM 实测 288×18 的内容盒，
	// 行按钮 21 高）。
	sidebarGroupRowHeight = 21
	// sidebarHeadingHeight 是「会话」「已置顶」这类分组标题行的高度。
	sidebarHeadingHeight = 25
	// sidebarActionSize 是行内小动作（置顶 / 归档 / 更多 / 工作区动作）的命中区。
	sidebarActionSize = 21
	// sidebarStatusSize 是会话行左端状态槽的边长；图标 16 居中。
	sidebarStatusSize = 21
	// sidebarObjectIcon 是工作区行的对象图标（🏠 / 文件夹）边长。
	sidebarObjectIcon = 14
	// sidebarAvatarSize 是账户行头像的边长。
	sidebarAvatarSize = 18
	// sidebarAccountHeight 是账户行的高度。
	sidebarAccountHeight = 30
	// sidebarBadgeHeight 是「等待操作」标签的高度。
	sidebarBadgeHeight = 18
	// sidebarKeycapHeight / sidebarKeycapWidth 是「⌘G」键帽的字面尺寸。
	sidebarKeycapHeight = 18
	sidebarKeycapWidth  = 32
	// sidebarEmptyIndent 是「暂无会话」相对分组行的缩进（与行文字同一列）。
	sidebarEmptyIndent = space8
	// sidebarHoverCardWidth 是会话行悬停卡的宽：小于栏宽（256），长标题折行。
	sidebarHoverCardWidth = 216
	// radiusFull 是胶囊与头像的整圆半径。
	radiusFull = 999
)

// 侧栏配色：底取 sidebar（明 #f6fdff / 暗 #090c0d），悬停与选中取
// --sidebar-hover / --sidebar-accent（12% 黑）。分组标题取
// text-sidebar-foreground/45 的合成值；「暂无会话」再淡一档（design-baseline §5）。
var (
	colorSidebarHover    = colorSet{gui.Hex("#e2e9ed"), gui.Hex("#282c2b")}
	colorSidebarSelected = colorSet{gui.Hex("#d8dce2"), gui.Hex("#262626")}
	colorSidebarHeading  = colorSet{gui.Hex("#8f9294"), gui.Hex("#8a8a8a")}
	colorSidebarFaint    = colorSet{gui.Hex("#9aa0a4"), gui.Hex("#7f7f7f")}
)

// 左栏图标：Heroicons v2.2.0 outline（24 网格、1.5 描边）的同形路径；参考产品
// 用 lucide 的地方按同一网格自绘（钉住、外部打开、面板收起）。命名带 Sidebar
// 前缀，避免与 icons.go 的通用图标混用。
var (
	iconSidebarHome       = mustSVG(`<path d="M2.25 12 11.54 2.71a.75.75 0 0 1 1.06 0L21.75 12M4.5 9.75v10.125c0 .621.504 1.125 1.125 1.125H9.75v-4.875c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21h4.125c.621 0 1.125-.504 1.125-1.125V9.75"/>`)
	iconSidebarFolder     = mustSVG(`<path d="M2.25 12.75V12A2.25 2.25 0 0 1 4.5 9.75h15A2.25 2.25 0 0 1 21.75 12v.75m-8.69-6.44-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z"/>`)
	iconSidebarFolderOpen = mustSVG(`<path d="M3.75 9.776c.112-.017.227-.026.344-.026h15.812c.117 0 .232.009.344.026m-16.5 0a2.25 2.25 0 0 0-1.883 2.542l.857 6a2.25 2.25 0 0 0 2.227 1.932H19.05a2.25 2.25 0 0 0 2.227-1.932l.857-6a2.25 2.25 0 0 0-1.883-2.542m-16.5 0V6A2.25 2.25 0 0 1 6 3.75h3.879a1.5 1.5 0 0 1 1.06.44l2.122 2.12a1.5 1.5 0 0 0 1.06.44H18A2.25 2.25 0 0 1 20.25 9v.776"/>`)
	iconSidebarFolderPlus = mustSVG(`<path d="M12 10.5v6m3-3H9m4.06-7.19-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z"/>`)
	iconSidebarList       = mustSVG(`<path d="M3.75 6.75h16.5M3.75 12h16.5M3.75 17.25h16.5"/>`)
	iconSidebarGrid       = mustSVG(`<path d="M3.75 3.75h6.5v6.5h-6.5zM13.75 3.75h6.5v6.5h-6.5zM3.75 13.75h6.5v6.5h-6.5zM13.75 13.75h6.5v6.5h-6.5z"/>`)
	iconSidebarPin        = mustSVG(`<path d="M9.75 3h4.5M12 3v6.75M8.25 9.75h7.5l1.125 4.5H7.125l1.125-4.5ZM12 14.25V21"/>`)
	iconSidebarPinOff     = mustSVG(`<path d="M9.75 3h4.5M12 3v6.75M8.25 9.75h7.5l1.125 4.5H7.125l1.125-4.5ZM12 14.25V21M4.5 4.5l15 15"/>`)
	iconSidebarDownload   = mustSVG(`<path d="M12 3.75v10.5m0 0 3.75-3.75M12 14.25 8.25 10.5M3.75 18.75h16.5"/>`)
	iconSidebarExternal   = mustSVG(`<path d="M13.5 6h4.5v4.5M18 6l-7.5 7.5M6.75 6.75h4.5M6.75 6.75v10.5h10.5V13.5"/>`)
	iconSidebarTrash      = mustSVG(`<path d="M4.5 6.75h15M9 6.75V4.5h6v2.25M6.75 6.75l.75 12.75h9l.75-12.75M10.5 10.5v6M13.5 10.5v6"/>`)
	iconSidebarPanelClose = mustSVG(`<rect x="3.75" y="3.75" width="16.5" height="16.5" rx="2.25"/><path d="M9.75 3.75v16.5"/><path d="m16.5 9.75-2.25 2.25 2.25 2.25"/>`)
	iconSidebarGauge      = mustSVG(`<path d="M3.75 12a8.25 8.25 0 0 1 16.5 0"/><path d="M12 12l4.5-3"/>`)
	iconSidebarSmile      = mustSVG(`<circle cx="12" cy="12" r="8.25"/><path d="M9 14.25c.75 1.25 1.75 1.875 3 1.875s2.25-.625 3-1.875M9.75 9.75h.008v.008h-.008zM14.25 9.75h.008v.008h-.008z"/>`)
	iconSidebarSend       = mustSVG(`<path d="M3.75 11.25 20.25 3.75l-7.5 16.5-2.25-6.75-6.75-2.25Z"/>`)
	iconSidebarLogout     = mustSVG(`<path d="M15.75 9V5.25A1.5 1.5 0 0 0 14.25 3.75H6A1.5 1.5 0 0 0 4.5 5.25v13.5A1.5 1.5 0 0 0 6 20.25h8.25a1.5 1.5 0 0 0 1.5-1.5V15M12 12h8.25m0 0-3-3m3 3-3 3"/>`)
	iconSidebarUser       = mustSVG(`<circle cx="12" cy="8.25" r="3.75"/><path d="M4.5 20.25a7.5 7.5 0 0 1 15 0"/>`)
	iconSidebarSpinner    = mustSVG(`<path d="M12 3.75a8.25 8.25 0 1 0 8.25 8.25"/>`)
	iconSidebarAlert      = mustSVG(`<circle cx="12" cy="12" r="8.25"/><path d="M12 8.25v4.5M12 15.75h.008v.008H12v-.008Z"/>`)
)

// sidebarRowPaint 给一行自绘入口上默认 / 悬停 / 按下三态底色：默认透明，悬停
// 侧栏悬停色，按下再深一档（并集里选中态由调用方先判，选中行不再叠悬停）。
func sidebarRowPaint(row gui.Element, t *gui.Theme, selected bool) gui.Element {
	switch {
	case selected:
		return row.Background(colorSidebarSelected.in(t))
	case row.Pressed():
		return row.Background(t.SurfacePressed)
	case row.Hovered():
		return row.Background(colorSidebarHover.in(t))
	}
	return row
}

// sidebarStatusSlot 是会话行左端的状态槽（21×21）：运行中给旋转的加载环，
// 等待操作给时钟，出错给红色警示，置顶的会话给钉住标记（LB-69）；没有标记时
// 留同宽空槽，让所有会话行的标题从同一列开始。
func sidebarStatusSlot(c *gui.Context, t *gui.Theme, marker string, pinned bool) {
	slot := gui.Row(c).Width(sidebarStatusSize).Height(sidebarStatusSize).Shrink(0).
		AlignItems(gui.Center).Justify(gui.Center)
	slot.Children(func() {
		switch {
		case marker == "●":
			spin := gui.Icon(c, iconSidebarSpinner).Size(iconInline, iconInline).
				TextColor(t.TextMuted).Label("状态：正在运行")
			spin.Rotate(spin.Loop("sidebar-running", time.Second, gui.Linear) * 360)
		case marker == "◷":
			gui.Icon(c, iconClock).Size(iconInline, iconInline).TextColor(t.TextMuted).
				Label("状态：等待你同意")
		case marker == "✕":
			gui.Icon(c, iconSidebarAlert).Size(iconInline, iconInline).TextColor(t.Danger).
				Label("状态：失败")
		case pinned:
			gui.Icon(c, iconSidebarPin).Size(iconInline, iconInline).TextColor(t.TextMuted).
				AccessHidden(true)
		}
	})
}

// sidebarBadge 是「等待操作」这类行内标签：10.5px、胶囊、warning 15% 底 + warning 字。
func sidebarBadge(c *gui.Context, t *gui.Theme, text string) {
	gui.Row(c).Height(sidebarBadgeHeight).Shrink(0).Padding(0, space1).
		Radius(radiusFull).Background(t.Warning.Alpha(0.16)).
		AlignItems(gui.Center).Children(func() {
		typeMicro.style(t, gui.Text(c, text)).TextColor(t.Warning).SingleLine()
	})
}

// sidebarKeycap 是行尾的快捷键键帽（⌘G）：10.5px、小圆角、更淡的填充。
func sidebarKeycap(c *gui.Context, t *gui.Theme, text string) {
	gui.Row(c).Width(sidebarKeycapWidth).Height(sidebarKeycapHeight).Shrink(0).
		Padding(0, space1).Radius(radiusSmall).Background(t.SurfacePressed).
		AlignItems(gui.Center).Justify(gui.Center).Children(func() {
		typeMicro.style(t, gui.Text(c, text)).TextColor(t.TextMuted).TextAlign(gui.Center)
	})
}

// sidebarDisclosure 是工作区行的图标槽（14×14）：悬停时换成折叠箭头，展开时
// 旋转 90°（参考产品：150ms、cubic-bezier(0.4, 0, 0.2, 1)，这里用框架的
// Animate 默认缓动）。
func sidebarDisclosure(c *gui.Context, t *gui.Theme, ws sidebarWorkspace, open, hovered bool) {
	if hovered {
		arrow := gui.Icon(c, iconChevronR).Size(sidebarObjectIcon, sidebarObjectIcon).
			TextColor(t.TextMuted)
		turn := float32(0)
		if open {
			turn = 90
		}
		arrow.Rotate(arrow.Animate("sidebar-open", turn, motionTransition/2))
		return
	}
	icon := iconSidebarFolder
	if ws.builtin {
		icon = iconSidebarHome
	} else if open {
		icon = iconSidebarFolderOpen
	}
	gui.Icon(c, icon).Size(sidebarObjectIcon, sidebarObjectIcon).TextColor(t.TextMuted)
}

// sortThreads 是排序口径的落点：最近的活跃时间 / 创建时间不在目录快照里，
// 因此两种口径目前同序（会话数不变、位置不变）；留成一处，接上时间事实后只改
// 这里。
func sortThreads(threads []sidebarThread, byCreated bool) []sidebarThread {
	_ = byCreated
	return threads
}
