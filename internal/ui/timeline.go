package ui

import (
	"fmt"
	"math"
	"slices"
	"strings"

	gui "github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// timeline 是 R3 对话流：内容左缘对齐，纵向滚动（§9.0 A3：对齐由内容推出，
// 不用居中）。空态只给一行说明，复用同一区域，不拆页面（U1-ia §4）。
func (a *App) timeline(c *gui.Context) {
	// 连着 Source 时，时间线只画它给的快照（不掺样例）。
	if snap, ok := a.liveSession(); ok {
		a.liveTimeline(c, snap)
		return
	}
	if a.wired() {
		// 还没有打开会话：空态，不给样例。
		a.emptyTimeline(c)
		return
	}
	t := c.Theme()
	if a.state == StateEmpty {
		// 空态：形状 + 一句事实（UI-23）。不写「写点什么开始」「拖进来」这类
		// 引导语——输入区就在下面，形状与位置已经说明下一步。
		gui.Box(c).Grow(1).Center().Label("对话流").Role(gui.RoleLog).Children(func() {
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Label("空态").Children(func() {
				iconGlyph(c, iconChat, t.TextMuted, "chat")
				typeBody.style(t, gui.Text(c, "还没有消息")).TextColor(t.TextMuted)
			})
		})
		return
	}

	gui.Scroll(c).TrackScroll(&a.feed).Grow(1).
		Label("对话流").Role(gui.RoleLog).Focusable().TabStop(true).Children(func() {
		gui.Column(c).Padding(space6, space4).Gap(space6).
			Label("对话内容").Children(func() {
			// 顶部「读更早」是一个动作入口：位置（时间线顶部）已经表达"更早"，
			// 所以只给图标，不写「读更早」这类界面语（UI-23）。样例里恒有更早。
			gui.Row(c).Shrink(0).Children(func() {
				iconButton(c, iconButtonSpec{icon: iconChevronUp, label: "读取更早的记录", size: rowCompact})
			})
			for i := range a.rounds {
				a.round(c, i)
			}
		})
	})
}

// round 画一轮：用户消息右对齐，助手正文铺开，下面是这一轮的组头。
// 轮次之间 21，轮内块之间 10.5（U2 §2.3）。
func (a *App) round(c *gui.Context, i int) {
	t := c.Theme()
	r := a.rounds[i]
	if r.unknown {
		// 未知提交：只有占位（原文还在本地）与说明行；没有组头、没有操作行，
		// 因为"这一轮是否存在"本身还没有事实。
		gui.Column(c).Gap(space2).Children(func() {
			gui.Row(c).Children(func() {
				gui.Spacer(c)
				gui.Box(c).MaxWidth(userBubbleMaxWidth).
					Padding(space2, space3).Radius(radiusCard).
					Background(t.Surface).Children(func() {
					typeBody.style(t, gui.Text(c, r.user)).Selectable()
				})
			})
			a.note(c, r.note)
		})
		return
	}
	gui.Column(c).Gap(space3).Children(func() {
		gui.Row(c).Children(func() {
			gui.Spacer(c)
			// 气泡外观画在自绘 Box 上；文本只负责字号与可选。
			gui.Box(c).MaxWidth(userBubbleMaxWidth).
				Padding(space2, space3).Radius(radiusCard).
				Background(t.Surface).Children(func() {
				typeBody.style(t, gui.Text(c, r.user)).Selectable()
			})
		})
		gui.Column(c).Gap(space2).Children(func() {
			a.reply(c, r)
			if r.note != "" {
				a.note(c, r.note)
			}
			a.roundHeader(c, i)
		})
	})
}

// reply 是助手正文。运行中在行尾给 2×14 的 accent-ink 光标（U2 §4.1、§5.3）。
func (a *App) reply(c *gui.Context, r round) {
	t := c.Theme()
	// 助手正文是 Markdown（U1 B2）：完成的回复用 View，正在写的用 Stream（逐字）。
	// 用宿主框架自带的 ui/markdown，不引入第三方（C4 §5 的决策）。
	if r.status != roundRunning {
		markdown.View(c, r.reply, &a.md)
		return
	}
	gui.RichText(c,
		gui.Span{Text: r.reply, Size: typeBody.sizeIn(t), Weight: typeBody.weight, Color: t.Text},
		gui.Span{Text: "\u258d", Size: typeBody.sizeIn(t), Color: colorAccentInk.in(t)},
	).Selectable()
}

// note 是本轮下方的一行说明（结果不确定）：行首 16px 状态位 + 一行正文，
// 说明属于本轮，不是容器（U2 §4.1）。
func (a *App) note(c *gui.Context, note string) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Label("结果不确定行").Children(func() {
		iconGlyph(c, iconWarn, t.Warning, "exclamation")
		typeBody.style(t, gui.Text(c, note)).Selectable()
	})
}

// roundHeader 是轮次组头（UI-11、§9.0 A3）：表头写这一组做了什么——动作与
// 对象；状态由行首图标、颜色与形态表达，不写状态词、不写计数、不写时长。
// 没有可说的动作（纯对话轮）就不出组头。展开后才是逐条「动作 · 对象 · 结果」。
//
// 入口是内容本身（UI-16）：可点区是内容块加上下各 space-1，方向指示紧跟内容，
// 不用空白把它顶到行末（§9.0 取向 4）。
func (a *App) roundHeader(c *gui.Context, i int) {
	t := c.Theme()
	r := a.rounds[i]
	phrase := roundHeaderPhrase(r)
	if phrase == "" {
		return
	}
	openPtr := a.openOf(i)

	parts := gui.CollapsibleBase(c, openPtr)
	// 指针形状与悬停底是"这能点"的两条线索（U2 §4.4）；文字与图标颜色不变。
	full := phrase
	shown := fitPhrase(c, t, phrase, a.sidebarCollapsed, a.rightOpen)
	tr := parts.Trigger.Gap(space1).AlignItems(gui.Center).AlignSelf(gui.Start).
		Padding(space1, space2).Radius(radius).Cursor(gui.CursorPointer).
		Tooltip(full)
	// 自绘可交互元素的态（§9.14 第 4 层、X17）：默认透明、悬停 surface-hover、
	// 按下 surface-pressed、聚焦由框架画环；组头不设禁用态（每一轮都可展开）。
	switch {
	case tr.Pressed():
		tr.Background(t.SurfacePressed)
	case tr.Hovered():
		tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		statusMark(c, t, r.status)
		typeSecondaryStrong.style(t, gui.Text(c, shown)).
			TextColor(headerColor(t, r.status)).SingleLine()
		chevron(c, t, parts.Progress())
	})
	parts.Panel(func() {
		gui.Column(c).Gap(space1).PaddingStart(space4).Children(func() {
			typeSecondary.style(t, gui.Text(c, roundMeta(r))).TextColor(t.TextMuted)
			for n, o := range r.ops {
				opRow(c, n+1, o)
			}
		})
	})
}

// chevron 是行末的方向指示：16×16 盒、单笔尖角，折叠时指向右、展开时指向下，
// 随开合旋转（U2 §4.4）。它不是独立按钮、不单独承担命中，也不进读屏（U3 §1.1
// L5）；标签留给测试与检查器定位，不参与读屏与行的可读名称。
func chevron(c *gui.Context, t *gui.Theme, progress float32) {
	box := gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0).
		Label("方向指示").AccessHidden(true)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.StrokePolyline(chevronPoints(r, chevronAngle(progress)), strokeChevron, t.TextMuted)
	})
}

// chevronAngle 把开合进度换成方向指示的旋转角：折叠 0°（指向右），展开 90°
// （指向下）。动效关闭时框架的 Progress 直接给 0 或 1，角度也就瞬时到位（UI-5）。
func chevronAngle(progress float32) float32 { return 90 * progress }

// chevronPoints 是盒内的一条尖角折线，绕盒中心顺时针转 deg 度。折叠时尖角朝右，
// 转 90° 后朝下；纯函数，便于逐点核对方向（U2 §4.4）。
func chevronPoints(r gui.Rect, deg float32) [][2]float32 {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	d := r.W / 4
	local := [3][2]float32{{-d * 0.5, -d}, {d * 0.8, 0}, {-d * 0.5, d}}
	sin, cos := float32(math.Sin(float64(deg)*math.Pi/180)), float32(math.Cos(float64(deg)*math.Pi/180))
	pts := make([][2]float32, len(local))
	for i, p := range local {
		pts[i] = [2]float32{cx + p[0]*cos - p[1]*sin, cy + p[0]*sin + p[1]*cos}
	}
	return pts
}


// opRow 是展开后的一行操作：序号、动作、对象、结果（数据给什么就写什么，
// 不带时长：时长不进界面，§9.0 取向 2）。失败行只把结果词标成 danger，
// 其余文字保持正文色（U2 §4.1：失败不把正文染红）。
func opRow(c *gui.Context, n int, o op) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
		// 行内编号用极小级（U2 §1.1）。
		typeMicro.style(t, gui.Text(c, fmt.Sprint(n))).Width(opNumberWidth).
			TextColor(t.TextMuted)
		// 动作前给对象类型的图标（UI-21）：读/改/跑各一个形状。
		iconBox(c, actionIcon(o.action), t.TextMuted, "")
		typeSecondary.style(t, gui.Text(c, o.action)).Width(opActionWidth).
			TextColor(t.TextMuted)
		typeSecondary.style(t, gui.Text(c, o.target)).Grow(1).SingleLine().
			TextColor(t.Text)
		typeSecondary.style(t, gui.Text(c, o.result)).Width(opResultWidth).
			TextColor(resultColor(t, o))
	})
	if o.cause != "" {
		// 原因紧跟失败条目并向内缩进：位置与应用说清它是原因，不再写「原因：」。
		typeSecondary.style(t, gui.Text(c, o.cause)).TextColor(t.TextMuted).
			PaddingStart(space4)
	}
}

// statusMark 是过程摘要行的行首状态位（U2 §4.1、§6.4、校准 C5）：
// 固定 16×16 盒，尺寸不随字号变化；运行中是居中的 8px 实心圆，其余是
// Heroicons outline 图标（24 网格 1.5 描边 → 16px 时 1.0px）。
func statusMark(c *gui.Context, t *gui.Theme, s roundStatus) {
	if s == roundRunning {
		box := gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0).
			Label("状态：" + statusWord(s))
		box.DrawOver(func(p *gui.Painter, r gui.Rect) {
			p.Fill(gui.Rect{
				X: r.X + (iconInline-dotStatus)/2,
				Y: r.Y + (iconInline-dotStatus)/2,
				W: dotStatus, H: dotStatus,
			}, colorAccentInk.in(t), dotStatus/2)
		})
		return
	}
	switch s {
	case roundWaiting:
		iconBox(c, iconClock, colorAccentInk.in(t), "状态："+statusWord(s))
	case roundFailed:
		iconBox(c, iconXMark, t.Danger, "状态："+statusWord(s))
	default:
		iconBox(c, iconCheck, t.TextMuted, "状态："+statusWord(s))
	}
}

// resultColor 只给失败的结果词上 danger，其余结果词取正文色。
func resultColor(t *gui.Theme, o op) gui.Color {
	if o.failed {
		return t.Danger
	}
	return t.Text
}

// statusWord 只用于读屏名称：屏幕上不显示状态词（§9.0 A2「能画出来的不写」），
// 组头的可读名称由「内容短语 + 状态」合成（§9.0 可读下限）。
func statusWord(s roundStatus) string {
	switch s {
	case roundRunning:
		return "正在运行"
	case roundWaiting:
		return "等待你同意"
	case roundFailed:
		return "失败"
	default:
		return "完成"
	}
}

// headerColor 让失败的一轮组头文字取 text-secondary（弱化），其余取正文色；
// 状态本身由行首图标与颜色表达（U2 §4.1）。
func headerColor(t *gui.Theme, s roundStatus) gui.Color {
	if s == roundFailed {
		return t.TextMuted
	}
	return t.Text
}

// roundHeaderPhrase 是本组的内容总览：做了什么——动作与代表对象，按首次出现
// 的顺序；不写状态词、计数与时长（§9.0 A3、D5）。对象多时取前面几个并收成
// 省略号，完整清单在展开里。没有动作的一组返回空串，调用方据此不出组头。
func roundHeaderPhrase(r round) string {
	if len(r.ops) == 0 {
		return ""
	}
	type group struct {
		action  string
		targets []string
	}
	var groups []group
	index := map[string]int{}
	for _, o := range r.ops {
		i, ok := index[o.action]
		if !ok {
			i = len(groups)
			index[o.action] = i
			groups = append(groups, group{action: o.action})
		}
		if t := shortTarget(o.target); !slices.Contains(groups[i].targets, t) {
			groups[i].targets = append(groups[i].targets, t)
		}
	}
	if len(groups) > headerGroupsMax {
		groups = groups[:headerGroupsMax]
		groups[headerGroupsMax-1].targets = append(groups[headerGroupsMax-1].targets, "…")
	}
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		targets := g.targets
		joined := strings.Join(targets, "、")
		if len(targets) > headerTargetsMax {
			joined += " …"
		}
		parts = append(parts, g.action+" "+joined)
	}
	return strings.Join(parts, " · ")
}

// headerAvailableWidth 是上面那条式子本身（纯函数，便于核对）。
func headerAvailableWidth(size func() (float32, float32), sidebarCollapsed, rightOpen bool) float32 {
	w, _ := size()
	sidebar := float32(SidebarWidth)
	if sidebarCollapsed {
		sidebar = SidebarRailWidth
	}
	panel := panelWidth(w, sidebar, rightOpen)
	return w - sidebar - panel - 2*space4 - iconInline - space1 - space1 - iconInline - space2 - space2
}

// fitPhrase 把组头收到可用的行宽内：先量（MeasureText），超宽才截；截断以对象
// 为单位（R7）——动作与对象的第一段一定留住，其余收成「…」。宽窗下不截。
func fitPhrase(c *gui.Context, t *gui.Theme, phrase string, sidebarCollapsed, rightOpen bool) string {
	width := headerAvailableWidth(c.Size, sidebarCollapsed, rightOpen)
	if phraseWidth(c, t, phrase) <= width {
		return phrase
	}
	shown := phrase
	for len([]rune(shown)) > 6 {
		shown = shortenPhrase(shown)
		if phraseWidth(c, t, shown) <= width {
			return shown
		}
	}
	return shown
}

// phraseWidth 量一句组头在组头字级下占的宽度。
func phraseWidth(c *gui.Context, t *gui.Theme, s string) float32 {
	w, _ := c.MeasureText(0, gui.Span{
		Text: s, Size: typeSecondaryStrong.sizeIn(t), Weight: typeSecondaryStrong.weight,
	})
	return w
}

// shortenPhrase 把一句组头按对象收一档：先去掉最后一个动作组，再把对象截到
// 第一段，最后才按字符收。
func shortenPhrase(phrase string) string {
	if i := strings.LastIndex(phrase, " · "); i > 0 {
		return phrase[:i] + " …"
	}
	if i := strings.Index(phrase, " "); i > 0 {
		head, obj := phrase[:i], phrase[i+1:]
		if first := firstSegment(strings.TrimSuffix(obj, " …")); first != "" &&
			first != strings.TrimSuffix(obj, " …") {
			return head + " " + first + " …"
		}
	}
	runes := []rune(strings.TrimSuffix(phrase, "…"))
	if len(runes) > 4 {
		runes = runes[:len(runes)-3]
	}
	return string(runes) + "…"
}

// truncatePhrase 以**对象**为单位截断组头（R7）：动作与对象的第一段一定留住，
// 多出来的在最后一个对象上收成「…」。整句超长时，逐个动作去掉尾部对象；仍超长
// 就把最后一个对象截到第一段（命令取第一个词、路径取末段），最后才收句。
func truncatePhrase(phrase string) string {
	if len([]rune(phrase)) <= headerPhraseMaxRunes {
		return phrase
	}
	// 先在最后一个「 · 」处收句，保留至少一个完整动作组。
	if i := strings.LastIndex(phrase, " · "); i > 0 {
		return truncatePhrase(phrase[:i] + " …")
	}
	// 只剩一个动作组：把对象截到第一段。
	if i := strings.Index(phrase, " "); i > 0 {
		head, obj := phrase[:i], phrase[i+1:]
		if first := firstSegment(obj); first != "" && first != obj {
			return head + " " + first + " …"
		}
	}
	// 连第一段都放不下：按字符硬收。
	runes := []rune(phrase)
	if len(runes) > headerPhraseMaxRunes-1 {
		runes = runes[:headerPhraseMaxRunes-1]
	}
	return string(runes) + "…"
}

// firstSegment 是对象的第一段：命令取第一个词，路径取末段。
func firstSegment(obj string) string {
	if i := strings.IndexAny(obj, " \t"); i > 0 {
		return obj[:i]
	}
	if i := strings.LastIndex(obj, "/"); i >= 0 && i+1 < len(obj) {
		return obj[i+1:]
	}
	return obj
}

// shortTarget 把操作对象写成组头里够用的短形式：文件路径取末段；命令与其它
// 带空格的原文保持原样（命令的末段不是它的名字）。
func shortTarget(s string) string {
	if strings.ContainsAny(s, " \t") {
		return s
	}
	if i := strings.LastIndex(s, "/"); i >= 0 && i+1 < len(s) {
		return s[i+1:]
	}
	return s
}

// roundMeta 是展开后的第一行：计数只在这里出现，组头与主界面不写计数与时长
// （§9.0 取向 2、UI-11）。
func roundMeta(r round) string {
	files := map[string]bool{}
	for _, o := range r.ops {
		if o.action == "改动" {
			files[o.target] = true
		}
	}
	meta := fmt.Sprintf("%d 个操作", len(r.ops))
	if len(files) > 0 {
		meta += fmt.Sprintf(" · 改了 %d 个文件", len(files))
	}
	return meta
}

// liveTimeline 只画 Source 给的快照：每条行一段文字，没有组头展开（后续批次接
// 完整的行渲染；首片先保证"事实进来、样例不掺"）。
func (a *App) liveTimeline(c *gui.Context, snap SessionSnapshot) {
	t := c.Theme()
	if len(snap.Rows) == 0 {
		a.emptyTimeline(c)
		return
	}
	gui.Scroll(c).TrackScroll(&a.feed).Grow(1).
		Label("对话流").Role(gui.RoleLog).Focusable().TabStop(true).Children(func() {
		gui.Column(c).Padding(space6, space4).Gap(space3).
			Label("对话内容").Children(func() {
			for _, row := range snap.Rows {
				row := row
				gui.Column(c).Shrink(0).Gap(space1).Children(func() {
					switch row.Kind {
					case "user":
						gui.Row(c).Children(func() {
							gui.Spacer(c)
							gui.Box(c).MaxWidth(userBubbleMaxWidth).Padding(space2, space3).
								Radius(radiusCard).Background(t.Surface).Children(func() {
								typeBody.style(t, gui.Text(c, row.Text)).Selectable()
							})
						})
					default:
						typeBody.style(t, gui.Text(c, row.Text)).Selectable()
					}
				})
			}
		})
	})
}

// emptyTimeline 是"没有消息"的空态（与样例空态同一形状）。
func (a *App) emptyTimeline(c *gui.Context) {
	t := c.Theme()
	gui.Box(c).Grow(1).Center().Label("对话流").Role(gui.RoleLog).Children(func() {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Label("空态").Children(func() {
			iconGlyph(c, iconChat, t.TextMuted, "chat")
			typeBody.style(t, gui.Text(c, "还没有消息")).TextColor(t.TextMuted)
		})
	})
}
