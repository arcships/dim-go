package ui

import (
	"context"
	"fmt"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › MCP」的重建屏。元素按
// docs/domains/requirements/inventory-settings-rt.md §10（S-mcp-01 … 51，
// 51 行；可见 15）与 inventory-settings.md §S-mcp（源码穷举 138 个）对齐，
// 行为对应 docs/domains/requirements/requirements-settings.md 的 R-MCP-001 … 232，
// 数值取 docs/domains/requirements/design-baseline.md 与运行时 JSON 的包围盒：
//
//	页面标题 17.5（y=33，h=25）、说明 12（y=61，h=18）、
//	分组标题 13/600（y=113，h=18）、动作按钮 32（y=106）、
//	范围标签 11（y=158，h=12）、范围下拉 140×32（y=148）、
//	服务器卡 741×64（y=190–254）、卡内左右 14 / 上下 14、
//	名称 12.25（y=205）、元信息 11（y=226，段间 6）、
//	工具数 783–818（图标 14 + 计数）、三个图标按钮 32×32（x=829/871/913）、
//	开关 28×16（x=955）、右端按钮与开关间距 10。
//
// 版式：列表视图（S-mcp-01 … 29）是首屏；编辑器（S-mcp-30 … 50）是可从
// 「添加服务器」「配置服务器」进入的第二视图。导入面板（S-mcp-07）点「导入」
// 在列表上方展开。
//
// 只保留可丢的界面状态（mcpScreen）：当前视图、范围筛选、导入面板开合、
// 每个服务器的开关位置、编辑器草稿。事实（服务器清单、状态、工具数、
// 警告、认证、插件来源）归 App Server：连着 Source 时清单每帧从
// Extensions().MCP 读，没有 Source（-demo）时用运行时实测的那一条做样例；
// 服务器开关走 SetExtensionEnabled("mcp")。没有单独的「重新加载」Intents：
// 清单随扩展读数重读，所以本屏不摆刷新按钮。
//
// # 未实现分支（清单里条件成立才出现的行）
//
//   - S-mcp-10…13 配置问题横幅与明细、加载骨架、错误 + 重试：要 App Server 的
//     配置问题与加载态事实；本屏只画读数里的 ReadError / 能力缺失两行。
//   - S-mcp-14/15 两种空态有渲染（无服务器 / 范围筛选空）；读数为空时走「无服务器」。
//   - S-mcp-18 认证提示、S-mcp-20/21 认证与取消认证：OAuth 流程要宿主能力，
//     本屏不摆认证按钮（该服务器无需认证）。
//   - S-mcp-26…29 来自插件的服务器（发现、只读、测试结果、三个图标按钮）：
//     插件来源的连接测试与只读标记要 App Server 的 inspect 能力。
//   - S-mcp-32/33/34 编辑器的加载态、已存储值隐藏提示、移动作用域：编辑已有
//     条目才出现，需要条目快照与 move 能力。
//   - S-mcp-48/49 表单校验错误与测试结果条：要写入与连接测试的返回值。
//   - S-mcp-51 删除服务器确认，S-mcp-100…106 查看连接弹窗：都要服务端的
//     写入 / inspect 能力；本版一律不做，保存、查看连接、断开连接按禁用态
//     渲染并写原因（DEC-04）。
//   - 导入面板的发现结果、预览、应用（S-mcp-113 … 138）：要服务端的
//     discover / preview / apply 能力，本屏只渲染面板骨架，两个入口为禁用态
//     并写原因（DEC-04）。
//
// 差异：参考产品的「了解更多。」与「文档」在悬停时加下划线，框架的文本没有
// 下划线装饰，本屏只保留链接色（登记为可接受偏差）。

// 几何：运行时 JSON 的包围盒（CSS px）与 design-baseline §1–§4 的档位。
const (
	mcpSectionGap = space8        // 说明下沿 79 → 组头上沿 106（27）
	mcpStackGap   = space3        // 组头 138 → 范围 148 → 卡 190（10）
	mcpHeadGap    = space1        // 标题 58 → 说明 61（3.5）
	mcpInlineGap  = screenSpace15 // 说明与链接、行内元信息段之间（6）
	mcpClusterGap = screenSpace25 // 工具数 / 图标按钮 / 开关之间（10）
	mcpRowPadX    = screenSpace35 // 卡左 257 → 名称 272
	mcpRowPadY    = screenSpace35 // 卡上 190 → 名称 205
	mcpRowNameGap = space1        // 名称 223 → 元信息 226
	mcpFieldGap   = screenSpace25 // 字段标签 → 控件（10）
	mcpPanelPadX  = screenSpace5  // 面板与空态卡的左右内边距（20）
	mcpPanelPadY  = screenSpace35 // 面板与空态卡的上下内边距（14）
	mcpEditorGap  = screenSpace5  // 编辑器块间 20
	mcpScopeW     = 140           // 范围下拉 140×32
	mcpActionSize = screenControlH
	mcpTinyIcon   = 12
)

// 字阶（design-baseline §2）：分组标题 13/600、名称 12.25/500、元信息 11、
// 字段标签 13/500、说明 12、链接 12.25、计数 12.25。
var (
	mcpGroupStyle = fontStep{size: 13, line: 18, weight: 600}
	mcpNameStyle  = fontStep{size: 12.25, line: 18, weight: 500}
	mcpMetaStyle  = fontStep{size: 11, line: 14, weight: 400}
	mcpFieldStyle = fontStep{size: 13, line: 18, weight: 500}
	mcpHintStyle  = fontStep{size: 12, line: 14, weight: 400}
	mcpLinkStyle  = fontStep{size: 12.25, line: 18, weight: 400}
	mcpCountStyle = fontStep{size: 12.25, line: 18, weight: 400}
)

// 本屏新增的图标：Heroicons v2 outline（24 网格、1.5 描边、MIT）的内联路径，
// 与 icons.go 同一套约定。
var (
	mcpIconDownload = mustSVG(`<path d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3"/>`)
	mcpIconExternal = mustSVG(`<path d="M13.5 6H5.25A2.25 2.25 0 0 0 3 8.25v10.5A2.25 2.25 0 0 0 5.25 21h10.5A2.25 2.25 0 0 0 18 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25"/>`)
	mcpIconActivity = mustSVG(`<path d="M3 12h3.75l2.25-6 4.5 12 2.25-6H21"/>`)
	mcpIconPower    = mustSVG(`<path d="M5.636 5.636a9 9 0 1 0 12.728 0M12 3v9"/>`)
	mcpIconWrench   = mustSVG(`<path d="M21.75 6.75a4.5 4.5 0 0 1-4.884 4.484c-1.076-.091-2.264.071-2.95.904l-7.152 8.684a2.548 2.548 0 1 1-3.586-3.586l8.684-7.152c.833-.686.995-1.874.904-2.95a4.5 4.5 0 0 1 6.336-4.486l-3.276 3.276a3.004 3.004 0 0 0 2.25 2.25l3.276-3.276c.256.565.398 1.192.398 1.852Z"/><path d="M4.867 19.125h.008v.008h-.008v-.008Z"/>`)
)

// 本屏文案：逐字取自参考产品的 zh-CN（i18n:toolsSettings.*、common.*）。
const (
	mcpCopyTitle       = "MCP 服务器"
	mcpCopyDesc        = "连接外部工具和数据源。"
	mcpCopyLearnMore   = "了解更多。"
	mcpCopyServers     = "服务器"
	mcpCopyImport      = "导入"
	mcpCopyAddServer   = "添加服务器"
	mcpCopyScope       = "范围"
	mcpCopyScopeAll    = "全部范围"
	mcpCopyScopeUser   = "用户"
	mcpCopyScopeItem   = "项目"
	mcpCopyNoServers   = "还没有配置 MCP 服务器"
	mcpCopyScopeEmpty  = "此范围内没有服务器。"
	mcpCopyInspect     = "查看连接"
	mcpCopyDisconnect  = "断开连接"
	mcpCopyConfigure   = "配置服务器"
	mcpCopySseDepr     = "SSE（已弃用）"
	mcpCopyImportTitle = "导入 MCP 服务器"
	mcpCopyImportDesc  = "从其他本地客户端发现 MCP 服务器，或选择 JSON 文件。界面只显示字段名；导入和替换的条目默认禁用。"
	mcpCopyImportDest  = "导入到：用户配置"
	mcpCopyClose       = "关闭"
	mcpCopyDiscover    = "发现"
	mcpCopyChooseFile  = "选择 JSON 文件"
	mcpCopyImportHint  = "当前版本不支持发现与预览 MCP 服务器。"

	mcpCopyConnectTitle = "连接自定义 MCP"
	mcpCopyDocs         = "文档"
	mcpCopyName         = "名称"
	mcpCopyNameExample  = "my-server"
	mcpCopyTransport    = "传输类型"
	mcpCopyStdio        = "STDIO"
	mcpCopyHTTP         = "Streamable HTTP"
	mcpCopySseNotice    = "SSE 已弃用，仅用于兼容旧版 MCP 服务器，不推荐用于新配置。请优先使用 Streamable HTTP。"
	mcpCopyCommand      = "启动命令"
	mcpCopyCommandEx    = "openai-dev-mcp serve-sqlite"
	mcpCopyArguments    = "参数"
	mcpCopyArgValue     = "参数值"
	mcpCopyAddArg       = "添加参数"
	mcpCopyRemoveArg    = "删除该参数"
	mcpCopyEnv          = "环境变量"
	mcpCopyKey          = "键"
	mcpCopyValue        = "值"
	mcpCopyAddEnv       = "添加环境变量"
	mcpCopyRemoveEnv    = "删除该环境变量"
	mcpCopyPassThrough  = "环境变量透传"
	mcpCopyPassValue    = "透传变量名"
	mcpCopyAddVar       = "添加变量"
	mcpCopyRemoveVar    = "删除该透传变量"
	mcpCopyCwd          = "工作目录"
	mcpCopyCwdEx        = "~/code"
	mcpCopyURL          = "URL"
	mcpCopyURLEx        = "https://mcp.example.com/mcp"
	mcpCopyBearer       = "Bearer 令牌"
	mcpCopyHeaders      = "请求头"
	mcpCopyAddHeader    = "添加请求头"
	mcpCopyRemoveHeader = "删除该请求头"
	mcpCopyEnvHeaders   = "来自环境变量的请求头"
	mcpCopyEnvHeader    = "ENV_VAR"
	mcpCopyRemoveRow    = "删除该行"
	mcpCopyCancel       = "取消"
	mcpCopySave         = "保存"
	mcpCopySaveOff      = noteMCPWrite
)

// 视图与传输类型：两组常量分开声明，各自的 iota 从 0 起。
const (
	mcpViewList = iota
	mcpViewEdit
)

const (
	mcpTransportStdio = iota
	mcpTransportHTTP
	mcpTransportSse
)

// mcpTransports 是三段传输类型，顺序与参考产品一致。
var mcpTransports = []string{mcpCopyStdio, mcpCopyHTTP, mcpCopySseDepr}

// mcpServer 是一台 MCP 服务器（一行视图数据）。事实行由 Source 给，
// 样例行只给 -demo；字段对应参考实现的 server 对象。
type mcpServer struct {
	// id 是服务端标识（启停写入要它）；样例行没有。
	id         string
	name       string
	scope      string
	status     string
	tools      int
	warning    string
	auth       string
	sse        bool
	enabled    bool
	connected  bool
	inspect    bool
	disconnect bool
	// command / args / url / transport 是编辑器带入的原始配置（事实）；
	// 样例行只有名字与范围。
	command   string
	args      []string
	url       string
	transport string
}

// mcpSampleServers 是运行时实测的那一条（MCP.json：dim-desktop-browser / user /
// disconnected / 0 个工具）。它是样例，不是事实：只在 -demo（没有 Source）时用。
var mcpSampleServers = []mcpServer{{
	id:   "dim-desktop-browser",
	name: "dim-desktop-browser", scope: "user", status: "disconnected",
	tools: 0, enabled: true, connected: false, inspect: true, disconnect: true,
}}

// mcpServers 取这一帧的 MCP 清单：连着 Source 时读 Extensions().MCP，
// 否则用样例。第二个返回值是读数本身的读态（ReadError / CapabilityMissing）。
func (a *App) mcpServers() ([]mcpServer, ExtListSnapshot) {
	if a.source == nil {
		return mcpSampleServers, ExtListSnapshot{}
	}
	list := a.source.Extensions().MCP
	rows := make([]mcpServer, 0, len(list.Items))
	for _, item := range list.Items {
		rows = append(rows, mcpServer{
			id: item.ID, name: item.Name, scope: item.Source, status: item.Status,
			tools: item.Tools, enabled: item.Enabled, connected: item.Connected,
			warning: item.Error, sse: item.Transport == "sse",
			command: item.Command, args: item.Args, url: item.URL,
			transport: item.Transport,
			// 查看连接要 inspect 能力、断开要 disconnect 能力：两份都没有，按钮
			// 照既有形态渲染，点击不写事实。
			inspect: true, disconnect: true,
		})
	}
	return rows, list
}

// mcpKeyValue 是编辑器里的一行键值。
type mcpKeyValue struct{ key, value string }

// mcpDraft 是编辑器草稿（可丢的界面状态）。
type mcpDraft struct {
	name        string
	scope       int
	transport   int
	command     string
	args        []string
	env         []mcpKeyValue
	passthrough []string
	cwd         string
	url         string
	bearer      string
	headers     []mcpKeyValue
	envHeaders  []string
}

// mcpState 是本屏可丢的界面状态：视图、范围、导入面板、开关、草稿。
type mcpState struct {
	view       int
	scope      int
	importOpen bool
	editing    bool // 编辑器打开的是已有条目（名称不可改）
	enabled    map[string]*bool
	draft      mcpDraft
}

var mcpScreen = mcpState{view: mcpViewList, enabled: map[string]*bool{}}

// renderMCP 是设置 › MCP 这一屏（设置壳的一个分节）。内容自己滚。
func renderMCP(c *gui.Context, a *App) {
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Label(mcpCopyTitle).Children(func() {
			gui.Column(c).Shrink(0).Gap(mcpSectionGap).Children(func() {
				mcpTitleBlock(c)
				if mcpScreen.view == mcpViewEdit {
					mcpEditor(c)
				} else {
					a.mcpListBlock(c)
				}
			})
		})
	})
}

// mcpTitleBlock 是页首：列表视图是标题 + 说明 + 链接；编辑器是标题 + 文档。
func mcpTitleBlock(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(mcpHeadGap).Children(func() {
		if mcpScreen.view == mcpViewEdit {
			// S-mcp-30 「连接自定义 MCP」
			gui.Row(c).AlignItems(gui.Center).Gap(mcpInlineGap).Children(func() {
				screenTitle(c, mcpCopyConnectTitle)
				// S-mcp-31 「文档」
				gui.Spacer(c)
				mcpLink(c, mcpCopyDocs)
			})
			return
		}
		// S-mcp-01 「MCP 服务器」
		screenTitle(c, mcpCopyTitle)
		gui.Row(c).AlignItems(gui.Center).Gap(mcpInlineGap).Children(func() {
			// S-mcp-02 「连接外部工具和数据源。」
			mcpHintStyle.style(t, gui.Text(c, mcpCopyDesc)).TextColor(t.TextMuted).
				Label(mcpCopyDesc).SingleLine()
			// S-mcp-03 「了解更多。」
			mcpLink(c, mcpCopyLearnMore)
		})
	})
}

// mcpLink 是一条外链：链接色文字 + 外链图标，读屏名同文案。打开浏览器要宿主
// 能力，本屏没有入口，点击不改任何状态（登记为未实现的动作）。
func mcpLink(c *gui.Context, text string) {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Radius(screenRadiusCtl).Label(text).
		Cursor(gui.CursorPointer)
	if b.Hovered() {
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			mcpLinkStyle.style(t, gui.Text(c, text)).TextColor(colorAccentInk.in(t)).SingleLine()
			screenIcon(c, mcpIconExternal, mcpTinyIcon, colorAccentInk.in(t), "external-link")
		})
	})
}

// mcpListBlock 是列表视图：组头、导入面板、范围行、服务器列表。
func (a *App) mcpListBlock(c *gui.Context) {
	rows, list := a.mcpServers()
	gui.Column(c).Shrink(0).Gap(mcpStackGap).Children(func() {
		mcpServerBar(c)
		if mcpScreen.importOpen {
			mcpImportPanel(c)
		}
		mcpScopeRow(c)
		mcpReadState(c, list)
		a.mcpServerList(c, rows)
	})
}

// mcpReadState 画读数本身的两种读态：能力缺失与最近一次读取失败。
// 两者都不清空上一次确认的清单（行照画），只在清单上方给原因。
func mcpReadState(c *gui.Context, list ExtListSnapshot) {
	t := c.Theme()
	// S-mcp-12 读数的两种读态（能力缺失 / 读取失败）
	if list.CapabilityMissing {
		mcpMetaStyle.style(t, gui.Text(c, "当前版本读不到完整的 MCP 清单，下面列出的是已确认的部分。")).
			TextColor(t.TextMuted).Role(gui.RoleStatus).Selectable()
	}
	if list.ReadError != "" {
		mcpMetaStyle.style(t, gui.Text(c, "读取 MCP 服务器失败："+list.ReadError)).
			TextColor(t.Danger).Role(gui.RoleAlert).Selectable()
	}
}

// S-mcp-13 读取 MCP 服务器失败 + 原因
// mcpServerBar 是「服务器」组头与右端两个动作（S-mcp-04 … 06）。
func mcpServerBar(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).AlignItems(gui.Center).Gap(mcpInlineGap).Children(func() {
		// S-mcp-04 组头「服务器」
		mcpGroupStyle.style(t, gui.Text(c, mcpCopyServers)).TextColor(t.Text).
			Role(gui.RoleHeading).Level(3).Label(mcpCopyServers).SingleLine()
		gui.Spacer(c)
		// S-mcp-05 「导入」
		if mcpButton(c, mcpIconDownload, "download", mcpCopyImport, false) {
			mcpScreen.importOpen = !mcpScreen.importOpen
		}
		// S-mcp-06 「添加服务器」
		if mcpButton(c, iconPlus, "plus", mcpCopyAddServer, true) {
			mcpScreen.openEditor(mcpServer{})
		}
	})
}

// mcpButton 是「图标 + 文字」按钮：outline 或主色实心，高 32。
func mcpButton(c *gui.Context, icon *gui.SVG, mark, text string, primary bool) bool {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(screenControlH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Label(text).Tooltip(text).Cursor(gui.CursorPointer)
	ink := t.Text
	if primary {
		b = b.Background(t.Accent)
		ink = t.AccentText
	} else {
		b = b.Background(t.Background)
		b.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
		})
	}
	switch {
	case primary && b.Pressed():
		b = b.Background(t.AccentPressed)
	case primary && b.Hovered():
		b = b.Background(t.AccentHover)
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		gui.Row(c).Gap(mcpInlineGap).AlignItems(gui.Center).Children(func() {
			screenIcon(c, icon, screenIconInline, ink, mark)
			screenValueStyle.style(t, gui.Text(c, text)).TextColor(ink).SingleLine()
		})
	})
	return b.Clicked()
}

// mcpImportPanel 是导入面板（S-mcp-07、107 … 112）：标题、说明、目标与两个
// 入口。发现 / 预览 / 应用要 App Server 的导入能力，两个入口按禁用态渲染。
// S-mcp-07 / S-mcp-08 / S-mcp-09 导入面板（发现 / 选择 JSON 文件 / 导入到用户配置）
func mcpImportPanel(c *gui.Context) {
	t := c.Theme()
	panel := gui.Column(c).Shrink(0).Background(t.Surface).Radius(screenRadiusCard)
	panel.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
	})
	panel.Children(func() {
		gui.Column(c).Padding(mcpPanelPadY, mcpPanelPadX).Gap(mcpFieldGap).Children(func() {
			gui.Row(c).AlignItems(gui.Center).Gap(mcpInlineGap).Children(func() {
				screenStrongStyle.style(t, gui.Text(c, mcpCopyImportTitle)).TextColor(t.Text).
					Role(gui.RoleHeading).Level(3).Label(mcpCopyImportTitle).SingleLine()
				gui.Spacer(c)
				if mcpIconAction(c, iconXMark, mcpCopyClose, "", false) {
					mcpScreen.importOpen = false
				}
			})
			mcpHintStyle.style(t, gui.Text(c, mcpCopyImportDesc)).TextColor(t.TextMuted).
				Selectable()
			mcpHintStyle.style(t, gui.Text(c, mcpCopyImportDest)).TextColor(t.TextMuted).
				Label(mcpCopyImportDest).SingleLine()
			gui.Row(c).Gap(screenSpace2).Children(func() {
				mcpDisabledButton(c, mcpCopyDiscover)
				mcpDisabledButton(c, mcpCopyChooseFile)
			})
			mcpHintStyle.style(t, gui.Text(c, mcpCopyImportHint)).TextColor(t.TextMuted).
				Selectable()
		})
	})
}

// mcpDisabledButton 是导入面板里未接能力的入口：28 高的描边按钮，禁用。
func mcpDisabledButton(c *gui.Context, text string) {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(rowRegular).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Background(t.Background).Label(text).Tooltip(noteMCPWrite).Disabled(true)
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	b.Children(func() {
		screenValueStyle.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
	})
}

// mcpScopeRow 是范围筛选行（S-mcp-08/09）：标签 11 + 140×32 下拉。
// S-mcp-07 范围行（全部范围 / 用户 / 项目）
func mcpScopeRow(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).AlignItems(gui.Center).Gap(mcpFieldGap).Children(func() {
		mcpMetaStyle.style(t, gui.Text(c, mcpCopyScope)).TextColor(t.TextMuted).
			Label(mcpCopyScope).SingleLine()
		mcpSelect(c, &mcpScreen.scope,
			[]string{mcpCopyScopeAll, mcpCopyScopeUser, mcpCopyScopeItem},
			mcpCopyScope, mcpScopeW)
	})
}

// mcpSelect 是自绘下拉：触发区固定宽，弹出面板逐项带对勾（选中项）。
func mcpSelect(c *gui.Context, selected *int, options []string, label string, width float32) {
	t := c.Theme()
	parts := gui.SelectBase[int](c, selected)
	trigger := parts.Trigger.Shrink(0).AlignSelf(gui.Start).Width(width).Height(screenControlH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Background(t.Background).Justify(gui.SpaceBetween).Gap(screenSpace1).Label(label)
	trigger.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	trigger.Children(func() {
		screenValueStyle.style(t, gui.Text(c, options[*selected])).TextColor(t.Text).SingleLine()
		screenIcon(c, screenIconChevronDown, screenIconMicro, t.TextMuted, "chevron-down")
	})
	parts.Popup(func(panel gui.Element) {
		panel.Padding(space1).Radius(screenRadiusCard).Background(t.Background)
		panel.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
		})
		for i, option := range options {
			item := parts.Item(i).Height(rowRegular).Padding(screenSpace15, screenSpace25).
				Radius(screenRadiusCtl).Label(option)
			color := t.Text
			switch {
			case item.Highlighted():
				item = item.Background(t.Accent)
				color = t.AccentText
			case i == *selected:
				item = item.Background(t.Surface)
			}
			item.Children(func() {
				gui.Row(c).Gap(mcpInlineGap).AlignItems(gui.Center).Children(func() {
					if i == *selected {
						screenIcon(c, iconCheck, screenIconMicro, color, "check")
					} else {
						gui.Box(c).Width(screenIconMicro).Shrink(0)
					}
					screenValueStyle.style(t, gui.Text(c, option)).TextColor(color).SingleLine()
				})
			})
		}
	})
}

// mcpServerList 是服务器列表（S-mcp-16 … 25）与两种空态。
// S-mcp-14 / S-mcp-15 服务器列表
func (a *App) mcpServerList(c *gui.Context, rows []mcpServer) {
	if len(rows) == 0 {
		mcpEmpty(c, mcpCopyNoServers)
		return
	}
	gui.Column(c).Shrink(0).Gap(mcpStackGap).Children(func() {
		visible := 0
		for i := range rows {
			if !mcpInScope(rows[i], mcpScreen.scope) {
				continue
			}
			visible++
			a.mcpServerCard(c, rows[i])
		}
		if visible == 0 {
			mcpEmpty(c, mcpCopyScopeEmpty)
		}
	})
}

// mcpInScope 判一条服务器是否落在当前范围筛选里。
func mcpInScope(server mcpServer, scope int) bool {
	switch scope {
	case 1:
		return server.scope == "user"
	case 2:
		return server.scope == "project"
	}
	return true
}

// mcpEmpty 是空态：虚线框在框架里没有对应描边，用面差与居中说明表达。
// S-mcp-14 / S-mcp-15 空态（还没有配置 MCP 服务器 / 此范围内没有服务器）
func mcpEmpty(c *gui.Context, text string) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Padding(mcpPanelPadY, mcpPanelPadX).
		Radius(screenRadiusCard).Background(t.Surface).
		Justify(gui.Center).Label(text).Children(func() {
		mcpHintStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted).
			TextAlign(gui.Center)
	})
}

// mcpServerCard 是一台服务器的卡（S-mcp-16 … 25）。点卡片（不是卡上的开关与
// 图标按钮）进入编辑器，编辑器带入这一条的真实配置。
// S-mcp-26 / S-mcp-27 / S-mcp-28 / S-mcp-29 服务器卡（插件来源、只读信息、连接状态、连接动作）
func (a *App) mcpServerCard(c *gui.Context, server mcpServer) {
	t := c.Theme()
	card := gui.Column(c).Shrink(0).Background(t.Surface).Radius(screenRadiusCard)
	card.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
	})
	// childHit 记录这一帧有没有点在卡上的交互控件：有则卡片本身不再算被点。
	childHit := false
	card.Children(func() {
		gui.Row(c).Padding(mcpRowPadY, mcpRowPadX).Gap(mcpClusterGap).
			AlignItems(gui.Center).Children(func() {
			gui.Column(c).Grow(1).MinWidth(0).Gap(mcpRowNameGap).Children(func() {
				mcpNameStyle.style(t, gui.Text(c, server.name)).TextColor(t.Text).
					Label(server.name).SingleLine()
				gui.Row(c).AlignItems(gui.Center).Gap(mcpInlineGap).Children(func() {
					if server.scope != "" {
						mcpMetaStyle.style(t, gui.Text(c, server.scope)).TextColor(t.TextMuted).
							SingleLine()
					}
					if server.sse {
						mcpMetaStyle.style(t, gui.Text(c, mcpCopySseDepr)).TextColor(t.TextMuted).
							SingleLine()
					}
					if server.status != "" {
						mcpMetaStyle.style(t, gui.Text(c, server.status)).TextColor(t.TextMuted).
							SingleLine()
					}
					if server.warning != "" {
						mcpMetaStyle.style(t, gui.Text(c, server.warning)).TextColor(t.Danger).
							SingleLine().Grow(1)
					}
					// S-mcp-11 warning.scope · serverId · message
					if server.auth != "" {
						mcpMetaStyle.style(t, gui.Text(c, server.auth)).TextColor(t.Danger).
							SingleLine().Grow(1)
					}
				})
			})
			mcpToolCount(c, server.tools)
			if server.inspect {
				// 查看连接要服务端的 inspect 能力，本版没有：禁用并写原因（DEC-04）。
				if mcpIconAction(c, mcpIconActivity, mcpCopyInspect, noteMCPWrite, true) {
					childHit = true
				}
			}
			if server.disconnect {
				// 断开连接同样没有通道：禁用并写原因（DEC-04）。
				if mcpIconAction(c, mcpIconPower, mcpCopyDisconnect, noteMCPWrite, true) {
					childHit = true
				}
			}
			if mcpIconAction(c, iconGear, mcpCopyConfigure, "", false) {
				mcpScreen.openEditor(server)
				childHit = true
			}
			toggle := mcpEnabled(server)
			sw := gui.Switch(c, toggle).Size(screenSwitchW, screenSwitchH).
				Radius(screenSwitchH / 2).Label("启用服务器 " + server.name)
			if sw.Changed() {
				a.mcpToggle(server, *toggle)
				childHit = true
			}
		})
	})
	if card.Clicked() && !childHit {
		mcpScreen.openEditor(server)
	}
}

// mcpEnabled 取一台服务器的开关位（本屏的可丢状态，初值来自这一帧的读数）。
func mcpEnabled(server mcpServer) *bool {
	key := server.id
	if key == "" {
		key = server.name
	}
	if on, ok := mcpScreen.enabled[key]; ok {
		return on
	}
	on := server.enabled
	mcpScreen.enabled[key] = &on
	return &on
}

// mcpToggle 把开关落到服务端（SetExtensionEnabled("mcp")）：失败原因由设置壳
// 的错误行给出，成功由扩展读数重读带回。样例行与没有 Intents 时只改界面状态。
func (a *App) mcpToggle(server mcpServer, enabled bool) {
	if a.intents == nil || server.id == "" {
		return
	}
	id := server.id
	a.dispatch("启停 MCP 服务器", func(ctx context.Context) error {
		return a.intents.SetExtensionEnabled(ctx, "mcp", id, enabled)
	}, nil)
}

// mcpToolCount 是行右端的工具数（S-mcp-19）：扳手图标 + 数字，读屏名为
// 「{count} 个工具」。
func mcpToolCount(c *gui.Context, tools int) {
	t := c.Theme()
	label := fmt.Sprintf("%d 个工具", tools)
	gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).Label(label).Children(func() {
		screenIcon(c, mcpIconWrench, screenIconMicro, t.TextMuted, "wrench")
		mcpCountStyle.style(t, gui.Text(c, fmt.Sprintf("%d", tools))).
			TextColor(t.TextMuted).SingleLine()
	})
}

// mcpIconAction 是一个 32×32 的图标入口：命中区 32、悬停与按下有底色、
// 禁用时半透明。tip 非空时作为 tooltip（禁用入口用它写原因）。
func mcpIconAction(c *gui.Context, icon *gui.SVG, label, tip string, disabled bool) bool {
	t := c.Theme()
	if tip == "" {
		tip = label
	}
	b := gui.ButtonBase(c).Shrink(0).Width(mcpActionSize).Height(mcpActionSize).
		Radius(screenRadiusCtl).Center().Label(label).Tooltip(tip).
		Cursor(gui.CursorPointer)
	if disabled {
		b = b.Disabled(true)
	}
	ink := t.TextMuted
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
		ink = t.Text
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
		ink = t.Text
	}
	b.Children(func() {
		screenIcon(c, icon, screenIconInline, ink, label)
	})
	return b.Clicked()
}

// ---- 编辑器（S-mcp-30 … 50） ----

// openEditor 进入编辑器：新建时草稿清空，配置某一条时带入这一条的真实配置
// （名称、范围、传输方式与启动命令；读不出来就不填，不猜）。
func (s *mcpState) openEditor(server mcpServer) {
	s.view = mcpViewEdit
	s.editing = server.name != ""
	s.draft = mcpDraft{}
	if !s.editing {
		return
	}
	s.draft.name = server.name
	if server.scope == "project" {
		s.draft.scope = 1
	}
	switch server.transport {
	case "http":
		s.draft.transport = mcpTransportHTTP
	case "sse":
		s.draft.transport = mcpTransportSse
	default:
		s.draft.transport = mcpTransportStdio
	}
	s.draft.command = server.command
	s.draft.args = append([]string(nil), server.args...)
	s.draft.url = server.url
}

// mcpEditor 是「连接自定义 MCP」表单：名称、范围、传输类型与两组字段。
func mcpEditor(c *gui.Context) {
	gui.Column(c).Shrink(0).Gap(mcpEditorGap).Children(func() {
		// S-mcp-35 名称
		mcpField(c, mcpCopyName, func() {
			name := screenInput(c, &mcpScreen.draft.name, mcpCopyNameExample, mcpCopyName)
			if mcpScreen.editing {
				name = name.Disabled(true)
			}
		})
		// S-mcp-36 范围（用户 / 项目）
		mcpField(c, mcpCopyScope, func() {
			mcpSelect(c, &mcpScreen.draft.scope,
				[]string{mcpCopyScopeUser, mcpCopyScopeItem}, mcpCopyScope, mcpScopeW)
		})
		mcpTransportSegments(c)
		if mcpScreen.draft.transport == mcpTransportStdio {
			mcpStdioFields(c)
		} else {
			if mcpScreen.draft.transport == mcpTransportSse {
				mcpNotice(c, mcpCopySseNotice)
			}
			mcpHTTPFields(c)
		}
		mcpHintStyle.style(c.Theme(), gui.Text(c, noteMCPWrite)).
			TextColor(c.Theme().TextMuted).Selectable()
		gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
			// S-mcp-50 保存 / 测试连接 / 取消
			if screenButton(c, mcpCopyCancel, mcpCopyCancel+"编辑 MCP 服务器", "", false, screenControlH) {
				mcpScreen.view = mcpViewList
			}
			mcpSaveButton(c, mcpCopySave)
		})
	})
}

// mcpField 是一个字段：13/500 标签 + 控件，标签与控件相距 10。
func mcpField(c *gui.Context, label string, control func()) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(mcpFieldGap).Children(func() {
		mcpFieldStyle.style(t, gui.Text(c, label)).TextColor(t.Text).Label(label).SingleLine()
		control()
	})
}

// mcpTransportSegments 是传输类型的三段切换（S-mcp-37）：选中档主色描边。
func mcpTransportSegments(c *gui.Context) {
	t := c.Theme()
	parts := gui.SegmentedBase(c, &mcpScreen.draft.transport, len(mcpTransports))
	track := parts.Track.AlignItems(gui.Stretch).Gap(screenStroke).
		Radius(screenRadiusCtl).Background(t.Border).Padding(screenStroke).
		Label(mcpCopyTransport)
	track.Children(func() {
		for i, name := range mcpTransports {
			seg := parts.Segment(i).Grow(1).Height(screenSegHLg).AlignItems(gui.Stretch).
				Padding(screenStroke).Radius(screenRadiusCtl).Background(t.Border).
				Label(name)
			if i == mcpScreen.draft.transport {
				seg = seg.Background(t.Accent)
			}
			seg.Children(func() {
				gui.Box(c).Grow(1).Radius(screenRadiusCtl).Background(t.Background).
					Center().Children(func() {
					ink := t.TextMuted
					if i == mcpScreen.draft.transport {
						ink = t.Text
					}
					mcpHintStyle.style(t, gui.Text(c, name)).TextColor(ink).SingleLine()
				})
			})
		}
	})
}

// mcpStdioFields 是 stdio 的五组字段（S-mcp-38 … 42）。
func mcpStdioFields(c *gui.Context) {
	mcpField(c, mcpCopyCommand, func() {
		screenInput(c, &mcpScreen.draft.command, mcpCopyCommandEx, mcpCopyCommand)
	})
	mcpStringList(c, mcpCopyArguments, &mcpScreen.draft.args,
		mcpCopyArgValue, mcpCopyAddArg, mcpCopyRemoveArg)
	mcpKeyValueList(c, mcpCopyEnv, &mcpScreen.draft.env,
		mcpCopyAddEnv, mcpCopyRemoveEnv)
	mcpStringList(c, mcpCopyPassThrough, &mcpScreen.draft.passthrough,
		mcpCopyPassValue, mcpCopyAddVar, mcpCopyRemoveVar)
	mcpField(c, mcpCopyCwd, func() {
		screenInput(c, &mcpScreen.draft.cwd, mcpCopyCwdEx, mcpCopyCwd)
	})
}

// mcpHTTPFields 是 http / sse 的四组字段（S-mcp-44 … 47）。
func mcpHTTPFields(c *gui.Context) {
	mcpField(c, mcpCopyURL, func() {
		screenInput(c, &mcpScreen.draft.url, mcpCopyURLEx, mcpCopyURL)
	})
	mcpField(c, mcpCopyBearer, func() {
		screenInput(c, &mcpScreen.draft.bearer, "", mcpCopyBearer)
	})
	mcpKeyValueList(c, mcpCopyHeaders, &mcpScreen.draft.headers,
		mcpCopyAddHeader, mcpCopyRemoveHeader)
	mcpStringList(c, mcpCopyEnvHeaders, &mcpScreen.draft.envHeaders,
		mcpCopyEnvHeader, mcpCopyAddVar, mcpCopyRemoveRow)
}

// mcpNotice 是一条警示说明（SSE 弃用提示，S-mcp-43）：警示底 + 图标 + 说明。
func mcpNotice(c *gui.Context, text string) {
	t := c.Theme()
	box := gui.Row(c).Shrink(0).Padding(mcpPanelPadY, mcpPanelPadX).
		Radius(screenRadiusCtl).Background(t.Surface).Gap(mcpInlineGap).
		AlignItems(gui.Start).Label(text)
	box.Children(func() {
		screenIcon(c, iconWarn, screenIconInline, t.Warning, "warning")
		mcpHintStyle.style(t, gui.Text(c, text)).TextColor(t.Text).Selectable()
	})
}

// mcpStringList 是一组单值行（参数 / 透传变量 / 环境变量请求头）。
func mcpStringList(c *gui.Context, label string, items *[]string, placeholder, addLabel, removeLabel string) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(mcpFieldGap).Children(func() {
		mcpFieldStyle.style(t, gui.Text(c, label)).TextColor(t.Text).Label(label).SingleLine()
		gui.Column(c).Shrink(0).Gap(screenSpace1).Children(func() {
			for i := range *items {
				index, removed := i, false
				gui.Row(c).Gap(screenSpace1).AlignItems(gui.Center).Children(func() {
					screenInput(c, &(*items)[index], placeholder, placeholder).Grow(1)
					if mcpIconAction(c, screenIconTrash, removeLabel, "", false) {
						removed = true
					}
				})
				if removed {
					mcpRemoveAt(items, index)
					break
				}
			}
			mcpAddButton(c, addLabel, func() {
				*items = append(*items, "")
			})
		})
	})
}

// mcpKeyValueList 是一组键值行（环境变量 / 请求头）。
func mcpKeyValueList(c *gui.Context, label string, items *[]mcpKeyValue, addLabel, removeLabel string) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(mcpFieldGap).Children(func() {
		mcpFieldStyle.style(t, gui.Text(c, label)).TextColor(t.Text).Label(label).SingleLine()
		gui.Column(c).Shrink(0).Gap(screenSpace1).Children(func() {
			for i := range *items {
				index, removed := i, false
				gui.Row(c).Gap(screenSpace1).AlignItems(gui.Center).Children(func() {
					screenInput(c, &(*items)[index].key, mcpCopyKey, mcpCopyKey).Grow(1)
					screenInput(c, &(*items)[index].value, mcpCopyValue, mcpCopyValue).Grow(1)
					if mcpIconAction(c, screenIconTrash, removeLabel, "", false) {
						removed = true
					}
				})
				if removed {
					*items = append((*items)[:index], (*items)[index+1:]...)
					break
				}
			}
			mcpAddButton(c, addLabel, func() {
				*items = append(*items, mcpKeyValue{})
			})
		})
	})
}

// mcpRemoveAt 删掉单值行里的一行。
func mcpRemoveAt(items *[]string, index int) {
	*items = append((*items)[:index], (*items)[index+1:]...)
}

// mcpAddButton 是整宽的「添加 …」按钮（secondary + 加号，32 高）。
func mcpAddButton(c *gui.Context, label string, run func()) {
	if mcpButton(c, iconPlus, "plus", label, false) {
		run()
	}
}

// mcpSaveButton 是保存：写入归 App Server，未接写入前按禁用态渲染（不留
// 看起来能存的按钮）。
func mcpSaveButton(c *gui.Context, label string) {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(screenControlH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Background(t.Accent).Label(label).Tooltip(mcpCopySaveOff).Disabled(true)
	b.Children(func() {
		screenValueStyle.style(t, gui.Text(c, label)).TextColor(t.AccentText).SingleLine()
	})
}
