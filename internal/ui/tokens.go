package ui

// 区域尺寸：取自 docs/domains/U1-ia.md §1、§7（U0 §12 的数值）。
// 这组是布局契约，不随视觉规范变化。
const (
	// WindowWidth 与 WindowHeight 是首片骨架的默认窗口尺寸。
	WindowWidth  = 1440
	WindowHeight = 900

	// R1 左栏。
	SidebarWidth     = 256
	SidebarRailWidth = 12
	SidebarMinWidth  = 200
	SidebarMaxWidth  = 360

	// R2 标题行。
	HeaderHeight = 48

	// R3 主区内容上限。
	ContentMaxWidth = 784

	// R4 输入位：卡片 60–300，上沿可带一行队列（33）。
	ComposerMinHeight = 60
	ComposerMaxHeight = 300
	ComposerHeight    = 124
	ApprovalHeight    = 150
	QueueHeight       = 33

	// R3 的宽度下限：右栏打开时主区不缩到 400 以下。
	MainMinWidth = 400

	// R5 右栏：280–（容器−400）。
	RightPanelMinWidth       = 280
	RightPanelPreferredWidth = 320
	RightPanelMainMinWidth   = 400
)

// 视觉数值占位：docs/domains/U2-visual.md 的首片 token 表已产出（draft，待评审），
// 评审通过后按表替换本组数值；在那之前这里只保证结构可走查。
// 颜色不在这张表里：一律取 c.Theme() 的中性灰阶，强调色取 Theme.Accent，
// 失败取 Theme.Danger；U2 的语义色表落地后改为按 token 取值。
const (
	spaceXXS = 2
	spaceXS  = 4
	spaceS   = 6
	spaceM   = 8
	spaceL   = 12
	spaceXL  = 16
	spaceXXL = 20

	radiusCard = 12
	radiusRow  = 8
	radiusChip = 6

	hairline = 1

	// 首片局部占位：等 U2 的 token 表定稿后并进统一标尺。
	composerPadding       = 7
	sidebarHeaderHeight   = 32
	userBubbleMaxWidth    = 520
	composerTextMinHeight = 44
	opNumberWidth         = 16
	opActionWidth         = 36
	opResultWidth         = 56
	opDurationWidth       = 44
	railMarkWidth         = 6
	railMarkHeight        = 2

	// smallFontSteps 是次要文字相对主题字号缩小的档数（占位）。
	smallFontSteps = 2
)
