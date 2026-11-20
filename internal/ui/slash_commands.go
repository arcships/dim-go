package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 斜杠命令弹层（requirements-global §3.1，G-B1-01–G-B1-20）。
//
// 输入框行首键入 `/` 弹出命令清单：四组（常用 / 会话 / 设置 / 调试），每行
// 图标 + 名称 + 说明 + 真实键位；↑↓ 选择、↵ 执行、Esc 关闭、继续输入按前缀
// 与模糊过滤；没有实现路径的命令置灰，原因写在行内（G-B1-05 的行内说明位）。
//
// 命令清单是**客户端静态表**：协议 v3 没有「命令清单」方法，清单不依赖服务端。
// 涉及写入的两条（`/compact`、`/reload-skills`）走既有 Intents：写入仍然只有
// 一个入口，不另开协议通道。其余命令只改可丢的界面状态（设置屏路由、模式位、
// 右栏标签），不写事实。
//
// 视图状态（可丢）：开合、当前前缀、Esc 关闭时的前缀、选中行与「执行反馈」。
// 选中行默认是 -1：参考 C37 实测 10 行全为普通底，首行并不默认高亮（G-B1-07）。

const (
	// slashMenuWidth 是弹层宽：参考 C37 实测 327（= min(24rem, 100vw-2rem)）。
	slashMenuWidth = 327
	// slashMenuRowHeight 是候选行高（C37：327×28）。
	slashMenuRowHeight = rowRegular
	// slashMenuHeaderHeight 是分组标题行高（C37 实测高 25，取行档 24）。
	slashMenuHeaderHeight = rowCompact
	// slashMenuVisibleRows 是一屏最多露出的候选行数：更多就在弹层内滚动。
	slashMenuVisibleRows = 8
	// slashMenuMaxHeight 是滚动区的高度上限：8 行候选（224），不遮住输入行。
	slashMenuMaxHeight = slashMenuRowHeight * slashMenuVisibleRows
	// slashMenuPad / slashMenuGap 是弹层内边距与行间距（与候选行文字对齐）。
	slashMenuPad = space2
	slashMenuGap = space1
	// slashMenuDescMax 是说明列的最大宽：命令名完整，说明在右端截断（G-B1-11）。
	slashMenuDescMax = 168
	// slashBranchNoChannel 是 /branch 在连接模式下的原因：侧聊还没有事实通道，
	// 打开它只会本地回声——不装作用。
	slashBranchNoChannel = "侧边对话当前不可用，先在这里继续"
)

// slashGroup 是命令的分组，顺序即渲染顺序（C37 的分组标题只有一组时也成立）。
type slashGroup int

const (
	slashGroupCommon slashGroup = iota
	slashGroupSession
	slashGroupSettings
	slashGroupDebug
	slashGroupCount
)

// slashGroupLabels 是四个分组标题，逐字上屏。
var slashGroupLabels = [slashGroupCount]string{"常用", "会话", "设置", "调试"}

// slashMenuState 是弹层的可丢视图状态：不承载事实，切屏 / 重开后重新开始。
type slashMenuState struct {
	open bool
	// prefix 是当前用于过滤的行首前缀（不含 `/`）。
	prefix string
	// dismissed / dismissedFor 是 Esc 关闭时停下的位置：前缀不变就不再自动
	// 弹出（G-B1-01 的「Esc 关闭，`/` 与前缀保留在输入框」）。用一个独立的
	// 布尔量而不是空串判断，`/` 后面为空（前缀是空串）时同样成立。
	dismissed    bool
	dismissedFor string
	// selected 是选中行的下标（按过滤后的命令顺序计数）；-1 表示没有选中。
	selected int
	// reveal 为真表示这一帧要把选中行滚进可视区（键盘移动后跟随，G-B1-07）。
	reveal bool
	// notice 是执行反馈（成功一句 / 不可用的原因），下一次打开弹层时清空。
	notice string
	// pointerInside 表示这一帧指针在弹层上（悬停或按下）。指针在弹层上时，
	// 焦点离开输入框不算「点击弹层外」——否则点一行会先把弹层关掉，点击落空。
	pointerInside bool
	// hadFocus 表示弹层打开之后输入框确实拿过焦点。刚打开的那一帧焦点还在
	// 上一棵树上（新树的焦点差一帧才到位），这时不算「焦点离开」，否则弹层
	// 会自己关掉。
	hadFocus bool
}

// reset 收起弹层并清掉本次的全部视图状态（不含 Esc 记住的前缀）。
func (s *slashMenuState) reset() {
	s.open = false
	s.prefix = ""
	s.selected = -1
	s.reveal = false
	s.hadFocus = false
}

// dismiss 是 Esc：收起弹层，记住当前前缀，草稿不动（不吞字）。
func (s *slashMenuState) dismiss() {
	s.dismissed = true
	s.dismissedFor = s.prefix
	s.reset()
}

// slashCommand 是清单里的一条命令。
//
// run 为 nil 表示本版没有实现路径：行置灰、不可选中，原因写在行内说明位。
// shortcut 只在确有键位时填（例如打开设置的 ⌘,）；没有绑定就留空，不画占位符。
//
// receipt 给「动作确实发生了，但结果不能写成一句已成功」的命令用：它自己完成
// 动作并返回逐字上屏的回执（取代默认的「已执行：/name」）。默认回执只用在
// 「执行了就真的生效」的命令上。
type slashCommand struct {
	name     string
	desc     string
	group    slashGroup
	icon     *gui.SVG
	shortcut string
	run      func()
	receipt  func() string
	reason   string
}

// runnable 表示这条命令有实现路径（可执行、可被选中）。
func (c slashCommand) runnable() bool { return c.run != nil || c.receipt != nil }

// execute 执行命令并返回这次执行的逐字回执。
func (c slashCommand) execute() string {
	if c.receipt != nil {
		return c.receipt()
	}
	if c.run != nil {
		c.run()
	}
	return "已执行：/" + c.name
}

// slashMenuPrefix 取行首的斜杠前缀：草稿第一行以 `/` 开头、且后面没有空白时
// 成立（`/settings providers` 这类带参数的写法不进弹层，本版不解析参数）。
func slashMenuPrefix(draft string) (string, bool) {
	line := draft
	if at := strings.IndexByte(draft, '\n'); at >= 0 {
		line = draft[:at]
	}
	if !strings.HasPrefix(line, "/") {
		return "", false
	}
	rest := line[1:]
	if strings.ContainsAny(rest, " \t") {
		return "", false
	}
	return rest, true
}

// syncSlashMenu 每帧按草稿决定弹层开合。草稿是唯一的触发来源：没有另一个
// 「打开弹层」的入口，也就不存在弹层与输入不一致的状态。
func (a *App) syncSlashMenu() {
	prefix, ok := slashMenuPrefix(a.draft)
	if !ok {
		a.slash.reset()
		return
	}
	if a.slash.open {
		if prefix == a.slash.prefix {
			return
		}
		// 继续输入：重新过滤；选中行回到「没有选中」，锚点不动（G-B1-01）。
		a.slash.prefix = prefix
		a.slash.selected = -1
		a.slash.reveal = false
		return
	}
	if a.slash.dismissed && prefix == a.slash.dismissedFor {
		// Esc 关掉之后，同样的前缀不再自动弹出；再键入字符才会回来。
		return
	}
	a.slash.open = true
	a.slash.prefix = prefix
	a.slash.dismissed = false
	a.slash.dismissedFor = ""
	a.slash.selected = -1
	a.slash.reveal = false
	a.slash.notice = ""
	a.slash.hadFocus = false
}

// slashFocus 处理「点击弹层外」：焦点离开输入区（落到别的按钮 / 列表 / 窗口
// 之外）就收起弹层，草稿不动。刚打开的那一帧不算：新树的焦点差一帧才到位，
// 先看一帧再判（与命令面板的宽限期同一处理）。
func (a *App) slashFocus(c *gui.Context) {
	if !a.slash.open {
		return
	}
	if c.FocusTakesText() {
		a.slash.hadFocus = true
		return
	}
	if a.slash.pointerInside {
		return
	}
	if a.slash.hadFocus {
		// 与 Esc 同一处理：记住当前前缀，草稿不动，不因为下一次重绘又弹回来。
		a.slash.dismiss()
	}
}

// slashCommands 是客户端静态命令表（13 条）。不可用的命令也在表里：入口常在，
// 只是置灰并写原因（G-B1-18 的「快速模式不可用 → 命令不出现」按参考产品处理：
// 本版保留入口，原因写明，不做「按能力隐藏」的第二套判定）。
func (a *App) slashCommands() []slashCommand {
	hasSession := true
	if a.wired() {
		_, hasSession = a.liveSession()
	}
	noIntents := a.intents == nil
	sessionReason := "还没有打开会话"
	if noIntents {
		sessionReason = "未连接 App Server，无法执行"
	}

	compact := slashCommand{
		// G-B1-17 「/compact」「压缩当前上下文」
		name: "compact", desc: "压缩当前上下文", group: slashGroupCommon,
		icon: iconChevronUp, run: nil, reason: sessionReason,
	}
	if !noIntents && hasSession {
		compact.run = func() {
			a.dispatch("压缩上下文", a.intents.CompactContext, nil)
		}
	}

	branch := slashCommand{
		// G-B1-19 「/branch」「侧边对话」
		name: "branch", desc: "侧边对话", group: slashGroupCommon,
		icon: iconChat, run: nil, reason: sessionReason,
	}
	switch {
	case a.wired():
		// 侧聊还没有事实通道（发不出去、历史读不回来）：在通道接通前不把用户送进
		// 一个只会本地回声的面板，按不可用画并写原因。
		branch.reason = slashBranchNoChannel
	case hasSession:
		// -demo 的样例面板：与「从消息处开始侧边对话」同一落点。
		branch.run = func() {
			a.rightOpen = true
			a.rpOpen(tabSideChat)
			a.panels.sideChatFromMessage = false
		}
	}

	goal := slashCommand{
		// G-B1-09 「/goal」「切换至 Goal 模式」
		name: "goal", desc: "切换至 Goal 模式", group: slashGroupSession,
		icon: iconTarget,
		// 与输入卡模式菜单同一落点（switchMode）；模式位本版只改界面标记，
		// 回执如实写明这一点，并写清是否打开了目标编辑框。
		receipt: func() string { return a.switchMode("Goal") },
	}
	if a.wired() && !hasSession {
		goal.receipt = nil
		goal.reason = "还没有打开会话，暂时设定不了目标"
	}

	reloadSkills := slashCommand{
		// G-B1-15 「/reload-skills」「重新扫描磁盘上的技能」
		name: "reload-skills", desc: "重新扫描磁盘上的技能", group: slashGroupDebug,
		icon: iconUndo, reason: "未连接 App Server，没有可重扫的技能目录",
	}
	if !noIntents {
		reloadSkills.run = func() {
			a.dispatch("重新扫描技能", a.intents.RefreshSkills, nil)
		}
	}

	return []slashCommand{
		compact,
		branch,

		{
			name: "agent", desc: "切换至 Agent 模式", group: slashGroupSession,
			icon: iconBot,
			// 与输入卡模式菜单同一落点；模式位本版只改界面标记，回执写明未生效范围。
			receipt: func() string { return a.switchMode("Agent") },
		},
		goal,
		{
			// G-B1-10 「/pet」「唤醒或收起 Pet」
			name: "pet", desc: "唤醒或收起 Pet", group: slashGroupSession,
			icon: iconPaw, reason: notePetShown,
		},

		{
			// G-B1-11 「/settings」「打开设置」
			name: "settings", desc: "打开设置", group: slashGroupSettings,
			icon: iconGear, shortcut: "⌘,",
			run: func() { a.screen, a.settingsSection = ScreenSettings, secAccount },
		},
		{
			// G-B1-12 「/models」「打开模型设置」
			name: "models", desc: "打开模型设置", group: slashGroupSettings,
			icon: iconChip,
			run:  func() { a.screen, a.settingsSection = ScreenSettings, secProviders },
		},
		{
			// G-B1-13 「/mcp」「打开 MCP 设置」
			name: "mcp", desc: "打开 MCP 设置", group: slashGroupSettings,
			icon: iconWrench,
			run:  func() { a.screen, a.settingsSection = ScreenSettings, secMCP },
		},
		{
			// G-B1-14 「/skills」「打开技能」
			name: "skills", desc: "打开技能", group: slashGroupSettings,
			icon: iconSparkles, run: a.openSkillsPage,
		},

		reloadSkills,
		{
			// G-B1-16 「/reload-plugins」「重新扫描已安装插件」
			name: "reload-plugins", desc: "重新扫描已安装插件", group: slashGroupDebug,
			icon: iconPlug, reason: notePluginReload,
		},
		{
			name: "fast", desc: "Toggle fast mode", group: slashGroupDebug,
			icon: iconGauge, reason: noteFastMode,
		},
		{
			// G-B1-20 「/workflow」「Browse workflows」
			name: "workflow", desc: "Browse workflows", group: slashGroupDebug,
			icon: iconCommand, reason: noteWorkflow,
		},
	}
}

// 命中档位：名称前缀 > 名称模糊（子序列）> 说明子串。有前缀命中时只留前缀
// 命中，短查询不会被模糊匹配带出一堆（`/ag` 只出 /agent，不出 reload-plugins）。
const (
	slashMatchNone = iota
	slashMatchPrefix
	slashMatchFuzzy
	slashMatchDesc
)

// slashCommandRank 判断一条命令命中的档位。大小写不敏感（G-B1-03 的 `/AG`）。
func slashCommandRank(cmd slashCommand, prefix string) int {
	q := strings.ToLower(strings.TrimSpace(prefix))
	if q == "" {
		return slashMatchPrefix
	}
	name := strings.ToLower(cmd.name)
	switch {
	case strings.HasPrefix(name, q):
		return slashMatchPrefix
	case slashSubsequence(name, q):
		return slashMatchFuzzy
	case strings.Contains(strings.ToLower(cmd.desc), q):
		return slashMatchDesc
	}
	return slashMatchNone
}

// slashSubsequence 判断 query 是不是 text 的子序列（按顺序出现，可不连续）。
func slashSubsequence(text, query string) bool {
	at := 0
	for _, r := range query {
		next := strings.IndexRune(text[at:], r)
		if next < 0 {
			return false
		}
		at += next + len(string(r))
	}
	return true
}

// slashMenuGroups 是过滤后的四组命令：组内保持静态表里的顺序，空组不渲染
// （G-B1-04 的「组内项全部被过滤掉 → 该组标题一起消失」）。
func (a *App) slashMenuGroups() [slashGroupCount][]slashCommand {
	commands := a.slashCommands()
	ranks := make([]int, len(commands))
	best := slashMatchNone
	for i, cmd := range commands {
		ranks[i] = slashCommandRank(cmd, a.slash.prefix)
		if ranks[i] != slashMatchNone && (best == slashMatchNone || ranks[i] < best) {
			best = ranks[i]
		}
	}
	var groups [slashGroupCount][]slashCommand
	for i, cmd := range commands {
		if ranks[i] == slashMatchNone || ranks[i] != best {
			continue
		}
		groups[cmd.group] = append(groups[cmd.group], cmd)
	}
	return groups
}

// slashMenuCommands 是过滤后的命令（跨组摊平）：键盘选择与执行都按这个顺序，
// 与渲染顺序一致。
func (a *App) slashMenuCommands() []slashCommand {
	groups := a.slashMenuGroups()
	var out []slashCommand
	for g := 0; g < int(slashGroupCount); g++ {
		out = append(out, groups[g]...)
	}
	return out
}

// slashSelectable 是当前可执行命令的下标（按过滤后的顺序）：键盘选择只在这些
// 行之间移动，置灰项不被选中。
func (a *App) slashSelectable() []int {
	commands := a.slashMenuCommands()
	selectable := make([]int, 0, len(commands))
	for i, cmd := range commands {
		if cmd.runnable() {
			selectable = append(selectable, i)
		}
	}
	return selectable
}

// slashMoveSelection 上下移动选中行：跳过置灰项，越界回卷；没有选中时按方向
// 落到第一个 / 最后一个可执行项（G-B1-07）。没有可执行项时不动。
func (a *App) slashMoveSelection(delta int) {
	selectable := a.slashSelectable()
	if len(selectable) == 0 {
		return
	}
	pos := -1
	for k, i := range selectable {
		if i == a.slash.selected {
			pos = k
		}
	}
	switch {
	case pos < 0 && delta > 0:
		pos = 0
	case pos < 0 && delta < 0:
		pos = len(selectable) - 1
	default:
		pos = (pos + delta + len(selectable)) % len(selectable)
	}
	a.slash.selected = selectable[pos]
	a.slash.reveal = true
}

// slashExecuteSelected 执行当前选中项：不可用的只把原因写回反馈行，不执行；
// 可执行的把命令文本从输入框里清掉并收起弹层（G-B1-08 的「执行后输入框清空
// 该命令文本」）。反馈行取命令自己的回执，不是一句无差别的「已执行」。
func (a *App) slashExecuteSelected() {
	commands := a.slashMenuCommands()
	if a.slash.selected < 0 || a.slash.selected >= len(commands) {
		return
	}
	cmd := commands[a.slash.selected]
	if !cmd.runnable() {
		a.slash.notice = "不可用：" + cmd.reason
		return
	}
	a.slash.notice = cmd.execute()
	a.draft = ""
	a.slash.reset()
}

// slashTextKey 接管输入框里的键：↑↓ 选择、↵ 执行、Esc 关闭（G-B1-01、G-B1-07）。
// 返回 true 表示这一键已被弹层消费，输入框不再处理它。
func (a *App) slashTextKey(k gui.TextKey) bool {
	if !a.slash.open {
		return false
	}
	switch {
	case k.Mods == 0 && k.Key == gui.KeyEscape:
		a.slash.dismiss()
		return true
	case k.Mods == 0 && k.Key == gui.KeyUp:
		a.slashMoveSelection(-1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyDown:
		a.slashMoveSelection(1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyEnter:
		if a.slash.selected < 0 {
			// 没有选中项（C37 的初始态）：回车仍是发送，不被弹层吞掉。
			return false
		}
		a.slashExecuteSelected()
		return true
	}
	return false
}

// slashMenu 画弹层：贴在输入卡上沿、与输入卡同宽的中心列里靠左对齐（C37 的
// 弹层 x = 输入区 x）。列表超过 8 行时在弹层内滚动。
func (a *App) slashMenu(c *gui.Context) {
	if !a.slash.open {
		// 不画弹层时留一个零尺寸占位：同一层里后面的元素下标不变，输入框不会
		// 因为弹层出现而换到新的节点（换节点就会丢焦点，弹层自己把自己关掉）。
		gui.Row(c).Shrink(0).Height(0)
		return
	}
	t := c.Theme()
	a.slash.pointerInside = false
	groups := a.slashMenuGroups()
	total := 0
	for g := 0; g < int(slashGroupCount); g++ {
		total += len(groups[g])
	}
	// IN-03 斜杠命令菜单浮层
	gui.Column(c).Shrink(0).Width(slashMenuWidth).Padding(slashMenuPad).
		Gap(slashMenuGap).Background(t.Surface).Border(strokeHairline, t.Border).
		Radius(radiusCard).Clip().
		Shadow(0, 4, 16, 0, colorShadow.in(t)).Label("斜杠命令").
		Children(func() {
			if total == 0 {
				// 没有匹配：给一句空态，不留空白浮层（G-B1-03）。
				gui.Row(c).Shrink(0).Padding(space2, space2).Children(func() {
					typeSecondary.style(t, gui.Text(c, "未找到命令或技能")).
						TextColor(t.TextMuted).SingleLine()
				})
				return
			}
			gui.Scroll(c).Shrink(0).MaxHeight(slashMenuMaxHeight).Children(func() {
				gui.Column(c).Gap(slashMenuGap).Children(func() {
					index := 0
					for g := 0; g < int(slashGroupCount); g++ {
						group := groups[g]
						if len(group) == 0 {
							continue
						}
						slashGroupHeader(c, slashGroupLabels[g])
						for _, cmd := range group {
							a.slashCommandRow(c, index, cmd)
							index++
						}
					}
				})
			})
		})
	// 键盘移动后把选中行滚进可视区一次；之后的手动滚动不被改写（G-B1-06）。
	a.slash.reveal = false
}

// slashGroupHeader 是分组标题：12px 次要色，左缘与命令名对齐（G-B1-04）。
func slashGroupHeader(c *gui.Context, label string) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Height(slashMenuHeaderHeight).Padding(0, space2).
		AlignItems(gui.Center).Label("命令分组：" + label).Children(func() {
		typeSecondary.style(t, gui.Text(c, label)).TextColor(t.TextMuted).SingleLine()
	})
}

// slashCommandRow 是一行命令：图标 + 名称 + 右端说明（不可用时换成原因）。
// 选中行用悬停面；置灰行不可选中、不可点（G-B1-05、G-B1-18）。
func (a *App) slashCommandRow(c *gui.Context, index int, cmd slashCommand) {
	t := c.Theme()
	selected := index == a.slash.selected
	label := "/" + cmd.name + "，" + cmd.desc
	note := cmd.desc
	if !cmd.runnable() {
		note = "不可用：" + cmd.reason
		label += "，" + note
	}
	row := gui.Row(c).Shrink(0).Height(slashMenuRowHeight).Padding(0, space2).
		Gap(slashMenuGap).Radius(radius).AlignItems(gui.Center).
		Label(label).Role(gui.RoleOption).AccessSelected(selected)
	if row.Hovered() {
		// 指针在弹层上：这一帧不按「点击弹层外」收起（见 slashFocus）。
		a.slash.pointerInside = true
	}
	if selected {
		row = row.Background(t.SurfaceHover)
	} else if row.Hovered() {
		row = row.Background(t.SurfaceHover)
	}
	if !cmd.runnable() {
		row = row.Opacity(noteDisabledAlpha)
	} else {
		row = row.OnClick(func() {
			a.slash.notice = cmd.execute()
			a.draft = ""
			a.slash.reset()
		})
	}
	if selected && a.slash.reveal {
		row = row.ScrollIntoView()
	}
	row.Children(func() {
		iconGlyph(c, cmd.icon, t.TextMuted, "command")
		typeBody.style(t, gui.Text(c, "/"+cmd.name)).TextColor(t.Text).SingleLine()
		typeSecondary.style(t, gui.Text(c, note)).TextColor(t.TextMuted).
			Grow(1).MaxWidth(slashMenuDescMax).TextAlign(gui.End).SingleLine()
		if cmd.shortcut != "" {
			typeMicro.style(t, gui.Text(c, cmd.shortcut)).TextColor(t.TextMuted).
				SingleLine()
		}
	})
}

// slashNotice 是执行反馈行：贴在输入卡上沿的一行小字（成功一句 / 不可用原因）。
// 下一次打开弹层时清空，不挂计时器。
func (a *App) slashNotice(c *gui.Context) {
	if a.slash.notice == "" {
		// 同 slashMenu：反馈行不画时也占一个零尺寸的位置。
		gui.Row(c).Shrink(0).Height(0)
		return
	}
	t := c.Theme()
	gui.Row(c).Shrink(0).Justify(gui.End).Padding(0, space2).Children(func() {
		typeMicro.style(t, gui.Text(c, a.slash.notice)).TextColor(t.TextMuted).
			SingleLine()
	})
}
