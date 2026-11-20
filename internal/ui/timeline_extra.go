package ui

import (
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件补时间线上此前缺的元素：用户消息轨道与标记、虚拟化占位、恢复遮罩、
// 用户中断、子代理通知 chip、终端错误详情、站点图标批次，以及用户消息正文里的
// 行内提及（技能 / 文件 / 提交 / 链接）。
//
// 事实口径：能读到的（行种类、状态、时长、工具参数与结果、技能清单、装载阶段）
// 一律照事实画；读不到的结构化提及、favicon 与终端调试详情不编造，按
// unavailable.go 的口径给出空态与原因。

const (
	// railWidth 是用户消息轨道列的宽度（参考细竖轨 + 标记，取时间线左内边距）。
	railWidth = timelinePadX
	// railMarker 是轨道上每个用户消息的标记边长（参考 6–8px）。
	railMarker = 7
	// mentionChipH 是行内提及 chip 的高度（取行档 18）。
	mentionChipH = rowCompact - 3
	// siteIconBox 是站点图标盒的边长（16×16 图标网格）。
	siteIconBox = iconInline
	// siteIconMax 是一行里最多画几个站点图标。
	siteIconMax = 3
)

// mentionOr 取第一个非空值；两个都空时返回 fallback。
func mentionOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// iconZap 是服务档 priority 的标记（lucide Zap，同一 24 网格、1.5 描边）。
var iconZap = mustSVG(`<path d="M13 2 4.5 13.5h6L11 22l8.5-11.5h-6L13 2Z"/>`)

// ---- 用户消息轨道（TL-04 / TL-05） ----

// railAvailable 报告轨道是否出现：窄窗里让位给正文，轨道的计数为 0。
func railAvailable(c *gui.Context) bool { return !narrowWindow(c) }

// userRail 是每条消息左侧的轨道段：细竖轨 + 可选标记。它是装饰，不进读屏。
// TL-04 用户消息轨道
func (a *App) userRail(c *gui.Context, marker bool) {
	if !railAvailable(c) {
		return
	}
	t := c.Theme()
	line := gui.Box(c).Absolute().Left(-railWidth).Top(0).Bottom(0).
		Width(railWidth).Label("用户消息轨道").AccessHidden(true)
	line.DrawOver(func(p *gui.Painter, r gui.Rect) {
		x := r.X + r.W/2
		p.Fill(hairlineRect(gui.Rect{X: x, Y: r.Y, W: strokeHairline, H: r.H},
			p.Scale(), false), t.Border, 0)
	})
	if !marker {
		return
	}
	// TL-05 每条用户消息的轨道标记
	box := gui.Box(c).Absolute().Left(-railWidth/2 - railMarker/2 + 1).Top(0).
		Width(railMarker).Height(railMarker).Label("用户消息标记").AccessHidden(true)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		square := gui.Rect{X: r.X, Y: r.Y, W: railMarker, H: railMarker}
		p.Fill(square, t.TextMuted, 1)
	})
}

// ---- 虚拟化占位（TL-06） ----

// virtualTopPlaceholder 是视口上方还有未渲染的行时的等高占位：无文字、无边框、
// 无骨架条，也不进读屏；滚入视口时被真实行替换。
// TL-06 虚拟化上占位
func (a *App) virtualTopPlaceholder(c *gui.Context) {
	if snap, ok := a.liveSession(); ok && snap.HasOlder {
		gui.Box(c).Shrink(0).Height(timelineRowHeight).WidthPercent(100).
			Label("更早的行占位").AccessHidden(true)
	}
}

// ---- 恢复遮罩（TL-71） ----

// restoreMask 是会话切换 / 历史位置恢复中的遮罩：品牌标记淡入 + 一行说明，
// 恢复期间不铺内容，也不把「还没有消息」的空态当成结论。
// TL-71 恢复遮罩
func (a *App) restoreMask(c *gui.Context) {
	t := c.Theme()
	gui.Box(c).Grow(1).Center().Label("恢复遮罩").Children(func() {
		gui.Column(c).AlignItems(gui.Center).Gap(space3).Children(func() {
			box := gui.Box(c).Width(brandMarkW).Height(brandMarkH).Shrink(0).
				Label("品牌：DimAgent").AccessHidden(true)
			progress := box.AnimateTo("restore-brand", 1, gui.Animation{
				Duration: 300 * time.Millisecond,
				Ease:     gui.CubicBezier(0.4, 0, 0.2, 1),
			})
			box = box.Opacity(0.4 + 0.6*progress)
			box.DrawOver(func(p *gui.Painter, r gui.Rect) {
				paintIcon(p, iconBrandWide, r, colorBrandEmpty)
			})
			typeSecondary.style(t, gui.Text(c, noteRestore)).TextColor(t.TextMuted)
		})
	})
}

// ---- 工作过程变体（TL-45 / TL-47 / TL-48 / TL-49） ----

// workProcessTitle 是轮首「工作过程」一行的文案，按这一轮的状态与事实选一种：
// 完成「工作过程 · actions」、被停止「已停止 · duration」、失败分「duration 后
// 失败」与「出错前的工作过程」两种（有终端输出时才算拿到了出错前的现场）。
// TL-45 工作过程 / 工作过程 · {actions}
func workProcessTitle(r round, summary string) string {
	switch r.status {
	case roundRunning:
		return runningSummaryText(r)
	case roundStopped:
		// TL-47 已停止 / 已停止 · {duration}
		if r.duration != "" {
			return "已停止 · " + r.duration
		}
		return "已停止"
	case roundFailed:
		if roundTerminalFailure(r) && r.duration != "" {
			// TL-48 {duration} 后失败 / · {actions}
			title := r.duration + " 后失败"
			if summary != "" {
				title += " · " + summary
			}
			return title
		}
		// TL-49 出错前的工作过程 / · {actions}
		if summary != "" {
			return "出错前的工作过程 · " + summary
		}
		return "出错前的工作过程"
	default:
		if summary == "" {
			return "工作过程"
		}
		return "工作过程 · " + summary
	}
}

// roundTerminalFailure 报告这一轮有没有「终端现场」：执行类工具失败且留下了
// 输出。它只读事实，不推断失败原因。
func roundTerminalFailure(r round) bool {
	for _, o := range r.ops {
		if o.action != "执行" {
			continue
		}
		if o.failed && (o.output != "" || o.cause != "") {
			return true
		}
	}
	return false
}

// userInterrupted 是运行被用户中断时的一行通知（TL-60）：它不冒充失败，
// 也不写原因；停止按钮本身就是原因。
// TL-60 用户中断
func (a *App) userInterrupted(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Grow(1).Gap(space2).AlignItems(gui.Center).Label("用户中断").Children(func() {
		iconGlyph(c, iconStop, t.TextMuted, "interrupted")
		typeSecondary.style(t, gui.Text(c, "用户中断")).TextColor(t.TextMuted)
	})
}

// ---- 子代理通知 chip（TL-56） ----

// subagentTool 判断一次工具调用是不是子代理（工具名没有统一口径，按常见名字
// 认：subagent / 子代理 / task / agent / delegate）。
func subagentTool(action string) bool {
	name := strings.ToLower(strings.TrimSpace(action))
	switch name {
	case "subagent", "task", "agent", "delegate", "子代理":
		return true
	}
	return strings.Contains(name, "subagent") || strings.Contains(name, "子代理")
}

// subagentChip 是子代理通知行：Bot 图标 + 子代理名 + 通知原文；点开右栏的子代理页。
// TL-56 子代理通知 chip
func (a *App) subagentChip(c *gui.Context, name, notice string) {
	t := c.Theme()
	label := "子代理：" + mentionOr(name, "子代理")
	if notice != "" {
		label += "，通知：" + notice
	}
	btn := gui.ButtonBase(c).Shrink(0).Height(timelineRowHeight).Gap(space1).
		Padding(0, space1).Radius(radiusSmall).Background(t.SurfaceHover).
		AlignItems(gui.Center).Cursor(gui.CursorPointer).Label(label).
		Tooltip(noteSubagentFacts)
	btn.Children(func() {
		iconGlyph(c, iconBot, t.TextMuted, "subagent")
		typeSecondaryStrong.style(t, gui.Text(c, mentionOr(name, "子代理"))).
			TextColor(t.Text).SingleLine()
		if notice != "" {
			typeMicro.style(t, gui.Text(c, notice)).TextColor(t.TextMuted).SingleLine()
		}
	})
	if btn.Clicked() {
		a.rightOpen = true
		a.rpOpen(tabSubagent)
	}
}

// ---- 终端错误详情（TL-59） ----

// terminalErrorDetail 是终端类工具失败后的详情折叠区：标题 + 复制 + 输出原文；
// 没有输出时写清缺的是哪一样，不把空块画成「没有错误」。
// TL-59 终端错误详情
func (a *App) terminalErrorDetail(c *gui.Context, o op) {
	t := c.Theme()
	body := strings.TrimSpace(o.output)
	if body == "" {
		body = strings.TrimSpace(o.cause)
	}
	gui.Column(c).Gap(space1).Children(func() {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			typeMicro.style(t, gui.Text(c, "终端错误详情")).TextColor(t.Danger).
				Label("工具详情：终端错误详情")
			gui.Spacer(c)
			if body == "" {
				return
			}
			if iconButton(c, iconButtonSpec{
				icon: iconCopy, label: "复制终端错误详情", tip: "复制终端错误详情",
				size: rowCompact,
			}) {
				c.WriteClipboard(body)
			}
		})
		if body == "" {
			typeSecondary.style(t, gui.Text(c, noteTerminalDetail)).TextColor(t.TextMuted)
			return
		}
		a.codeBlock(c, body)
	})
}

// ---- 站点图标批次（TL-76） ----

// webHosts 从工具参数里取网页类工具的主机名：只认带 scheme 的 URL，按出现
// 顺序去重；本版不联网取 favicon，图标用首字母占位并说明原因。
func webHosts(target string) []string {
	out := make([]string, 0, siteIconMax)
	for _, field := range strings.Fields(target) {
		value := strings.Trim(field, "()[]<>,;\"'")
		lower := strings.ToLower(value)
		index := strings.Index(lower, "://")
		if index < 0 {
			continue
		}
		rest := value[index+3:]
		if end := strings.IndexAny(rest, "/?#"); end >= 0 {
			rest = rest[:end]
		}
		host := strings.TrimSpace(rest)
		if host == "" || containsString(out, host) {
			continue
		}
		out = append(out, host)
		if len(out) >= siteIconMax {
			break
		}
	}
	return out
}

// siteIconBatch 是网页工具行左侧的站点图标批次：每个主机一个 16×16 的首字母盒，
// 悬停给出主机名与本版不取 favicon 的原因。
// TL-76 站点图标批次
func (a *App) siteIconBatch(c *gui.Context, hosts []string) {
	t := c.Theme()
	for _, host := range hosts {
		host := host
		box := gui.Box(c).Width(siteIconBox).Height(siteIconBox).Shrink(0).
			Radius(radiusSmall).Background(t.SurfacePressed).
			Label("站点图标：" + host).Tooltip(host + "（" + noteSiteIcon + "）")
		box.DrawOver(func(p *gui.Painter, r gui.Rect) {
			paintIcon(p, iconGlobe, r, t.TextMuted)
		})
	}
}

// ---- 行内提及（TL-27 / TL-28 / TL-29 / TL-30） ----

// mentionKind 是正文里一段提及的种类。
type mentionKind int

const (
	mentionText mentionKind = iota
	mentionSkill
	mentionFile
	mentionCommit
	mentionLink
)

// mention 是正文按提及切开后的一段：text 段是原文，其余段带解析出的名称与
// 完整值（完整路径 / 完整哈希 / 完整链接）。
type mention struct {
	kind mentionKind
	text string
	full string
}

// parseMentions 把用户消息正文切成原文与四类提及。语法只认无歧义的写法：
// `$技能`、`@路径`、`#7–40 位十六进制`、`http(s)://` 链接；识别不了的留在原文里。
func parseMentions(text string) []mention {
	out := make([]mention, 0, 4)
	var plain strings.Builder
	flush := func() {
		if plain.Len() > 0 {
			out = append(out, mention{kind: mentionText, text: plain.String()})
			plain.Reset()
		}
	}
	for i := 0; i < len(text); {
		switch text[i] {
		case '$':
			if name, n := scanMentionName(text[i+1:], isSkillRune, 1); n > 0 {
				flush()
				out = append(out, mention{kind: mentionSkill, text: name, full: name})
				i += 1 + n
				continue
			}
		case '@':
			if path, n := scanMentionName(text[i+1:], isPathRune, 1); n > 0 {
				flush()
				out = append(out, mention{kind: mentionFile, text: path, full: path})
				i += 1 + n
				continue
			}
		case '#':
			if hash, n := scanMentionName(text[i+1:], isHashRune, 7); n >= 7 && n <= 40 {
				flush()
				out = append(out, mention{kind: mentionCommit, text: hash, full: hash})
				i += 1 + n
				continue
			}
		case 'h', 'H':
			if url, n := scanURL(text[i:]); n > 0 {
				flush()
				host := url
				if index := strings.Index(url, "://"); index >= 0 {
					host = url[index+3:]
				}
				if end := strings.IndexAny(host, "/?#"); end >= 0 {
					host = host[:end]
				}
				out = append(out, mention{kind: mentionLink, text: host, full: url})
				i += n
				continue
			}
		}
		plain.WriteByte(text[i])
		i++
	}
	flush()
	return out
}

// scanMentionName 读一段满足 ok 的连续字符；min 是名字的最小长度。它返回名字
// 与吃掉的字节数（不含前缀符号）。
func scanMentionName(s string, ok func(byte) bool, min int) (string, int) {
	n := 0
	for n < len(s) && ok(s[n]) {
		n++
	}
	// 尾部标点不属于名字（`@internal/ui/a.go，` 里的逗号）。
	for n > min && strings.ContainsRune(".,;:!?，。；：！？）)", rune(s[n-1])) {
		n--
	}
	if n < min {
		return "", 0
	}
	return s[:n], n
}

func isSkillRune(b byte) bool {
	return b == '-' || b == '_' || b == '.' ||
		(b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isPathRune(b byte) bool {
	return isSkillRune(b) || b == '/'
}

func isHashRune(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// scanURL 读一段 http(s) 链接；遇到空白或成对括号结束。
func scanURL(s string) (string, int) {
	lower := strings.ToLower(s)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return "", 0
	}
	n := 0
	for n < len(s) && !strings.ContainsRune(" \t\r\n<>\"')]", rune(s[n])) {
		n++
	}
	if n == 0 {
		return "", 0
	}
	return s[:n], n
}

// skillKnown 报告一条技能是否在当前读数里；读数拿不到时按「不知道」处理，
// 不把技能标成已卸载。
func (a *App) skillKnown(name string) (bool, bool) {
	skills, readErr := a.skillEntries()
	if readErr != "" || len(skills) == 0 {
		return false, false
	}
	for _, skill := range skills {
		if skill.Name == name {
			return true, true
		}
	}
	return false, true
}

// mentionChip 是一条行内提及：技能（chip）、文件（可点，打开右栏预览）、提交
// （不可点，悬停给完整哈希）、链接（本版没有外部打开通道，置灰写原因）。
func (a *App) mentionChip(c *gui.Context, m mention) {
	t := c.Theme()
	switch m.kind {
	case mentionSkill:
		// TL-27 技能提及 chip
		known, read := a.skillKnown(m.text)
		color := t.Text
		aria := "技能：" + m.text
		if read && !known {
			color = t.TextMuted
			aria += "（不可用）"
		}
		gui.Row(c).Shrink(0).Height(mentionChipH).Gap(space1).
			Padding(0, space1).Radius(radiusSmall).Background(t.SurfacePressed).
			AlignItems(gui.Center).Label(aria).Children(func() {
			iconGlyph(c, iconChip, t.TextMuted, "skill")
			typeSecondary.style(t, gui.Text(c, m.text)).TextColor(color).SingleLine()
		})
	case mentionFile:
		// TL-28 文件 / 目录提及
		name := m.text
		if index := strings.LastIndex(name, "/"); index >= 0 && index+1 < len(name) {
			name = name[index+1:]
		}
		btn := gui.ButtonBase(c).Shrink(0).Height(mentionChipH).Gap(space1).
			Padding(0, space1).Radius(radiusSmall).Background(t.SurfacePressed).
			AlignItems(gui.Center).Cursor(gui.CursorPointer).
			Label("打开文件：" + m.text).Tooltip(m.text)
		btn.Children(func() {
			iconGlyph(c, iconDocument, t.TextMuted, "mention-file")
			typeSecondary.style(t, gui.Text(c, name)).TextColor(t.Text).SingleLine()
		})
		if btn.Clicked() {
			a.rightOpen = true
			a.rpOpenPreview(m.text)
		}
	case mentionCommit:
		// TL-29 提交提及 chip：不可点，悬停给完整哈希。
		short := m.text
		if len(short) > 8 {
			short = short[:8]
		}
		gui.Row(c).Shrink(0).Height(mentionChipH).Gap(space1).
			Padding(0, space1).Radius(radiusSmall).Background(t.SurfacePressed).
			AlignItems(gui.Center).Label("提交 " + short).Tooltip(m.full).Children(func() {
			iconGlyph(c, iconGitCommit, t.TextMuted, "commit")
			typeCode.style(t, gui.Text(c, short)).TextColor(t.Text).SingleLine()
		})
	case mentionLink:
		// TL-30 链接提及：站点图标 + 主机名；外部打开没有通道。
		btn := gui.ButtonBase(c).Shrink(0).Height(mentionChipH).Gap(space1).
			Padding(0, space1).Radius(radiusSmall).Background(t.SurfacePressed).
			AlignItems(gui.Center).Disabled(true).
			Label("在浏览器打开：" + m.text + "（" + noteOpenWith + "）").
			Tooltip(m.full + "（" + noteOpenWith + "）").Opacity(noteDisabledAlpha)
		btn.Children(func() {
			iconGlyph(c, iconGlobe, t.TextMuted, "mention-link")
			typeSecondary.style(t, gui.Text(c, m.text)).TextColor(t.Text).SingleLine()
		})
	}
}

// userBody 是用户消息正文：有行内提及时按提及切开画 chip / 按钮，没有时按原文
// 画。折叠态仍走原来的纯文本路径（MaxLines 对整段生效），展开后给出提及。
// TL-27 / TL-28 / TL-29 / TL-30 用户消息里的行内提及
func (a *App) userBody(c *gui.Context, i int, text string, overflow, expanded bool) gui.Element {
	if overflow && !expanded {
		return a.searchText(c, searchKeyUser(i), text, typeBody, c.Theme().Text).
			MaxLines(bubbleCollapsedLines)
	}
	// 会话内搜索有命中时走纯文本路径：高亮区间按整段原文算，切段后区间会错位。
	if len(a.searchRanges(searchKeyUser(i))) > 0 {
		return a.searchText(c, searchKeyUser(i), text, typeBody, c.Theme().Text)
	}
	mentions := parseMentions(text)
	hasMention := false
	for _, m := range mentions {
		if m.kind != mentionText {
			hasMention = true
			break
		}
	}
	if !hasMention {
		return a.searchText(c, searchKeyUser(i), text, typeBody, c.Theme().Text)
	}
	body := gui.Row(c).Wrap().Gap(space1).AlignItems(gui.Center)
	body.Children(func() {
		for _, m := range mentions {
			if m.kind == mentionText {
				a.searchText(c, searchKeyUser(i), m.text, typeBody, c.Theme().Text)
				continue
			}
			a.mentionChip(c, m)
		}
	})
	return body
}
