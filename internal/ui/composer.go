package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 输入区（参考产品的 InputArea + NewThread）：空态在输入卡上方给品牌标记与
// 「开始构建」，会话态只留同一个输入卡；审批卡整体替换输入区。
//
// 几何取 design-baseline：卡片圆角 12.75、控件高 25、行内间距 3.5 / 7 / 14。
// 文案取 requirements-main 的 IN-*：占位「发送消息...」、工具行「+」「»」「技能」、
// 右侧「高」+ 模型名 + 发送、底部控制行「工作区」「默认权限」。
//
// 来源差异（空态）：参考产品的空态只有品牌标记、「开始构建」与输入卡，首次运行
// 由一次性欢迎卡（inventory-global K5）承接，没有「登录账号」「添加供应商」这两条
// 入口。这两条是我们的首次运行引导，只在**未登录且没有供应商**时出现（已登录 /
// 已有供应商时与参考一致，不画），点了直接跳设置对应分节。
//
// 来源差异（断线卡）：「查看日志」也是我们加的：参考产品在这一位只给原因与重试。
// 它打开入口交来的客户端日志（Options.LogPath），打开失败时把路径留在原位供复制。

const (
	// composerMaxWidth 是输入卡的最大宽（参考 max-w-3xl = 672）。
	composerMaxWidth = 672
	// composerMinHeight 是输入卡的最小高：正文一行 + 工具行 + 内边距。
	composerMinHeight = 110
	// composerControlHeight 是空态底部控制行的高度（参考 40）。
	composerControlHeight = 40
	// composerToolSize 是工具行图标按钮的边长（参考 h-7 = 25）。
	composerToolSize = 25
	// composerInnerHeight 是审批卡里命令面板的最小高。
	composerInnerHeight = 80
	// brandMarkW / brandMarkH 是空态品牌标记的墨迹尺寸（参考实测 68×49，
	// 与标记自身 34:24.5 的比例一致）。
	brandMarkW = 68
	brandMarkH = 49
	// composerMenuWidth 是标题栏 ⋯ 菜单的最小宽（参考 159；中文括注留余量）。
	composerMenuWidth = 190
	// composerControlTriggerHeight 是控制行按钮的高度（参考 21）。
	composerControlTriggerHeight = 24
	// 浮层与行内定宽（参考产品的采集值）。
	composerEffortMenuWidth     = 220
	composerModelMenuWidth      = 305
	composerRecentWidth         = 150
	composerSkillsMenuWidth     = 260
	composerModeMenuWidth       = 200
	composerWorkspaceMenuWidth  = 271
	composerPermissionMenuWidth = 160
	composerSliderHeight        = 34
	composerSearchRowHeight     = 25
)

// 重建输入区 / 标题栏新增的两个颜色：控制行底（参考 #fafafa）与卡片投影。
var (
	colorControlBar = colorSet{gui.Hex("#fafafa"), gui.Hex("#151515")}
	colorShadow     = colorSet{gui.RGBA(0, 0, 0, 0.08), gui.RGBA(0, 0, 0, 0.45)}
)

// 空态标题：参考 text-2xl = 21 / 28，字重 700。
var typeEmptyTitle = fontStep{size: 21, line: 28, weight: 700}

// sampleWorkspaceTitle 是样例里的工作区名；接入 Source 后由会话标题替换。
const sampleWorkspaceTitle = "dim-agent"

// sampleWorkdir 是样例审批卡里的工作目录；接真实审批时由审批事实替换。
const sampleWorkdir = "~/dim-agent"

// sampleModel 是样例里的当前模型；接入 Source 后读设置快照。
const sampleModel = "seed-2.1-pro"

// composer 是输入位：断线说明、审批卡、提问卡或输入卡，同一位置只放一件事。
func (a *App) composer(c *gui.Context) {
	gui.Column(c).Shrink(0).Padding(space3, space4).Children(func() {
		if a.disconnected() {
			a.disconnectedCard(c)
			return
		}
		if a.state == StateWaiting {
			gui.Row(c).Justify(gui.Center).Children(func() { a.approvalCard(c) })
			return
		}
		if _, ok := a.liveApproval(); ok {
			gui.Row(c).Justify(gui.Center).Children(func() { a.approvalCard(c) })
			return
		}
		if question, ok := a.liveQuestion(); ok {
			a.questionCard(c, question)
			return
		}
		// 接上事实且已打开会话时保留目标行（参考产品把它做成输入区上沿的活动条）。
		_, hasSession := a.liveSession()
		if hasSession {
			a.goalRow(c)
		}
		gui.Row(c).Justify(gui.Center).Children(func() {
			// 连上事实但没有打开会话时也是空态：输入卡底下带工作区与权限两处入口
			// （-demo 的空态走 emptyThread，由它带控制行）。
			a.inputBlock(c, a.wired() && !hasSession)
		})
	})
}

// emptyThread 是空态（NewThread）：品牌标记 +「开始构建」+ 输入卡 + 控制行；
// 还没登录且没有供应商时，卡片下方多两条首次运行入口。
// 它占满主区并整体居中，时间线不在空态里重复写一句「还没有消息」。
func (a *App) emptyThread(c *gui.Context) {
	t := c.Theme()
	gui.Box(c).Grow(1).Center().Padding(space4).Label("空态").Children(func() {
		gui.Column(c).WidthPercent(100).MaxWidth(composerMaxWidth).Gap(space8).
			Children(func() {
				gui.Column(c).AlignItems(gui.Center).Gap(space2).Children(func() {
					brandMark(c, brandMarkW, brandMarkH, t)
					typeEmptyTitle.style(t, gui.Text(c, "开始构建")).
						FontWeight(700).Label("开始构建")
				})
				a.inputBlock(c, true)
				if a.firstRunNeeded() {
					a.firstRunGuide(c)
				}
			})
	})
}

// firstRunNeeded 是首次运行入口的出现条件：**未登录且没有供应商**。
// 已登录或已有供应商时，参考产品的空态没有这两条入口。
func (a *App) firstRunNeeded() bool {
	return !a.accountLoggedIn() && !a.hasProviders()
}

// accountLoggedIn 取账号是否已登录：连着 Source 时看服务端确认的状态，
// 样例（-demo）里有账号首字母就是已登录（样例账号与设置 › 账户同一份）。
func (a *App) accountLoggedIn() bool {
	if a.source == nil {
		return a.accountInitial != ""
	}
	return a.source.Account().Status == "signed_in"
}

// hasProviders 取是否读到任何供应商：管理清单、模型选择器的供应商表或
// 当前供应商任一非空即算已有供应商；样例（-demo）没有供应商读数，按没有算。
func (a *App) hasProviders() bool {
	if a.source == nil {
		return false
	}
	settings := a.source.Settings()
	return len(settings.ProviderRecords) > 0 || len(settings.Providers) > 0 ||
		settings.ProviderID != ""
}

// firstRunGuide 是空态的两条首次运行入口（我们的差异项，见文件头）：
// 「登录账号」进设置 · 账户、「添加供应商」进设置 · 模型供应。
// 它只切屏与分节，不发请求、不新开对话框：设置屏里的入口才是真正的写路径。
func (a *App) firstRunGuide(c *gui.Context) {
	gui.Row(c).Justify(gui.Center).Gap(space2).Label("首次运行引导").Children(func() {
		if actionButton(c, "登录账号", "登录账号：打开设置里的账户分节", false, false).Clicked() {
			a.screen, a.settingsSection = ScreenSettings, secAccount
		}
		if actionButton(c, "添加供应商", "添加供应商：打开设置里的模型供应分节", false, false).Clicked() {
			a.screen, a.settingsSection = ScreenSettings, secProviders
		}
	})
}

// brandMark 是空态的品牌图形：墨迹 68×49（标记自身的比例），取品牌亮绿；
// 只画形状，读屏名给产品名。
func brandMark(c *gui.Context, width, height float32, t *gui.Theme) {
	box := gui.Box(c).Width(width).Height(height).Shrink(0).Label("品牌：DimAgent")
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Icon(iconBrandWide, r, colorBrandEmpty)
	})
}

// inputBlock 是输入卡：正文 + 工具行；空态在下面多一条控制行（工作区 / 权限）。
func (a *App) inputBlock(c *gui.Context, withControls bool) {
	t := c.Theme()
	gui.Column(c).WidthPercent(100).MaxWidth(composerMaxWidth).MinHeight(composerMinHeight).
		Background(t.Surface).Border(strokeHairline, t.Border).
		Radius(radiusWindowCard).Clip().
		Shadow(0, 2, 10, 0, colorShadow.in(t)).Label("输入卡").Children(func() {
		gui.Column(c).Padding(space4).Gap(space2).Children(func() {
			if len(a.attachments) > 0 {
				gui.Row(c).Gap(space2).Wrap().Children(func() {
					for i := range a.attachments {
						a.attachChip(c, i)
					}
				})
			}
			if a.attachmentEditing {
				a.attachmentEditor(c)
			}
			ph, name := "发送消息...", "消息"
			if a.shellMode() {
				ph, name = "输入命令...", "命令"
			}
			input := gui.TextAreaBase(c, &a.draft).Placeholder(ph).Grow(1).MinHeight(composerTextMinHeight).
				Label(name)
			// 全局唤起（⌘⇧Space）后把焦点落到输入区：Focus 只在请求被消费的
			// 那一帧调用一次，用户之后点别处不会再被抢回来（AutoFocus 只对
			// 当帧新建的元素生效，输入区早就存在，用不上）。
			if a.focusComposer.Swap(false) {
				input = input.Focus()
			}
			a.composerToolRow(c)
		})
		if withControls {
			a.composerControls(c)
		}
	})
}

// composerToolRow 是输入卡里的工具行：左侧附件 / 模式 / 技能，右侧推理强度 /
// 模型 / 发送。所有入口都是 25×25 的图标位（文字入口按内容放宽）。
func (a *App) composerToolRow(c *gui.Context) {
	gui.Row(c).Height(composerToolSize).Shrink(0).AlignItems(gui.Center).Children(func() {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
			if iconButton(c, iconButtonSpec{
				icon: iconPlus, label: "附件", tip: "添加附件", size: composerToolSize,
			}) {
				a.attachmentEditing = !a.attachmentEditing
			}
			a.modeSelector(c)
			a.skillsSelector(c)
		})
		gui.Spacer(c)
		gui.Row(c).Gap(space4).AlignItems(gui.Center).Children(func() {
			a.effortSelector(c)
			a.modelSelector(c)
			a.sendSlot(c)
		})
	})
}

// composerControls 是空态底部控制行：工作区 + 权限，灰底、圆角随卡片下沿。
func (a *App) composerControls(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Height(composerControlHeight).Shrink(0).Padding(space2, space3).
		Background(colorControlBar.in(t)).AlignItems(gui.Center).Gap(space3).
		Label("输入区控制行").Children(func() {
		a.workspaceSelector(c)
		a.permissionSelector(c)
		gui.Spacer(c)
	})
}

// sendSlot 是发送 / 停止槽：无草稿时禁用（浅色主色仍是主色），运行中换成停止。
func (a *App) sendSlot(c *gui.Context) {
	icon, label, tip := iconArrowUp, "发送", "发送消息... · Enter"
	disabled := strings.TrimSpace(a.draft) == "" || a.disconnected()
	switch {
	case a.shellMode():
		icon, label, tip = iconPlay, "运行命令", "运行命令 · Enter"
		disabled = strings.TrimSpace(a.shellCommand()) == ""
	case a.liveRunning():
		icon, label, tip = iconStop, "停止", "停止回复 · Enter"
		disabled = false
	}
	if iconAction(c, iconButtonSpec{
		icon: icon, label: label, tip: tip, size: composerToolSize, disabled: disabled,
	}, true) {
		switch {
		case a.liveRunning():
			if a.intents != nil {
				a.dispatch("停止", a.intents.Stop, nil)
			}
		case a.shellMode():
			a.runShellCommand()
		case a.intents != nil:
			a.submitDraft()
		}
	}
}

// submitDraft 发送当前草稿；附件块沿用原提交路径（样例只清空草稿）。
func (a *App) submitDraft() {
	text := a.draft
	blocks := a.attachmentBlocks()
	a.dispatch("发送", func(ctx context.Context) error {
		return a.intents.SubmitWithContent(ctx, text, blocks)
	}, func() {
		a.draft = ""
		a.attachments = nil
	})
}

// attachmentBlocks 把待发附件变成提交内容块。
func (a *App) attachmentBlocks() []ContentBlockSnapshot {
	blocks := make([]ContentBlockSnapshot, 0, len(a.attachments))
	for _, chip := range a.attachments {
		if chip.kind == "image" {
			source := &ImageBlockSnapshot{MediaType: chip.mediaType, Data: chip.data}
			if chip.fileID != "" {
				source.FileID = chip.fileID
				source.SourceType = "file"
			} else {
				source.SourceType = "base64"
			}
			blocks = append(blocks, ContentBlockSnapshot{Kind: "image", Image: source})
			continue
		}
		text := chip.text
		if chip.path != "" {
			lines := strings.Count(text, "\n") + 1
			text = fmt.Sprintf("[Pasted text: %s (%d lines)]", chip.path, lines)
		}
		blocks = append(blocks, ContentBlockSnapshot{Kind: "text", Text: text})
	}
	return blocks
}

// attachmentChipData 是一个待发附件：文本与图片两种。
type attachmentChipData struct {
	name      string
	kind      string // text / image
	text      string
	mediaType string
	data      string
	fileID    string
	path      string
}

// disconnectedCard 是断线时的输入位说明：原因写在原位，给重新连接与查看日志入口。
// 「查看日志」打开入口交来的客户端日志（Options.LogPath）；打不开时把路径留在
// 卡片第二行供复制，不静默失败。
func (a *App) disconnectedCard(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).MinHeight(ComposerMinHeight).Padding(space4).Gap(space2).
		Background(t.Surface).Border(strokeHairline, t.Border).
		Radius(radiusWindowCard).Label("连接断开").Children(func() {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
			iconGlyph(c, iconWarn, t.Danger, "exclamation")
			typeBody.style(t, gui.Text(c, "连接已断开："+a.liveConnection().Reason)).
				TextColor(t.Danger).SingleLine()
			gui.Spacer(c)
			if actionButton(c, "查看日志", "查看日志：用系统默认程序打开客户端日志", false, false).Clicked() {
				a.openLog()
			}
			if actionButton(c, "重新连接", "重新连接服务端", false, false).Clicked() && a.intents != nil {
				a.dispatch("重新连接", a.intents.Reconnect, nil)
			}
		})
		if a.logNote != "" {
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Label("日志路径").Children(func() {
				typeCode.style(t, gui.Text(c, a.logNote)).TextColor(t.TextMuted).SingleLine()
				if iconButton(c, iconButtonSpec{
					icon: iconClipboard, label: "复制日志路径", tip: "复制日志路径", size: rowCompact,
				}) {
					c.WriteClipboard(a.logNote)
				}
			})
		}
	})
}

// openLog 打开客户端日志：成功不留提示，失败把路径（或缺失原因）留在第二行。
// 打开是本地动作，不经过 Intents：它不是对会话的写入，也不碰 App Server。
func (a *App) openLog() {
	if a.logPath == "" {
		a.logNote = "入口没有记录日志路径（-log 为空）"
		return
	}
	if err := openWithSystem(a.logPath); err != nil {
		a.logNote = a.logPath
		return
	}
	a.logNote = ""
}

// approvalCard 是审批卡：整体替换输入区，标题、命令面板与三个决定。
func (a *App) approvalCard(c *gui.Context) {
	t := c.Theme()
	cmd := a.approvalCommand
	if ap, ok := a.liveApproval(); ok {
		cmd = ap.Command
	}
	if cmd == "" {
		cmd = "ls -la"
	}
	gui.Column(c).WidthPercent(100).MaxWidth(composerMaxWidth).MinHeight(ApprovalHeight).
		Padding(space4).Gap(space2).Background(t.Surface).
		Border(strokeHairline, t.Border).Radius(radiusWindowCard).Label("审批卡").Children(func() {
		gui.Column(c).Gap(space2).Children(func() {
			typeBodyStrong.style(t, gui.Text(c, "DimAgent 想要：执行")).
				FontWeight(600).Label("DimAgent 想要：执行")
			gui.Column(c).MinHeight(composerInnerHeight).Padding(space2, space3).Gap(space1).
				Border(strokeHairline, t.Border).Radius(radiusCard).Children(func() {
				typeMicro.style(t, gui.Text(c, "输入")).TextColor(t.TextMuted).Label("输入")
				gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
					typeCode.style(t, gui.Text(c, "$")).TextColor(t.TextMuted)
					typeCode.style(t, gui.Text(c, cmd)).TextColor(t.Text)
				})
				typeCode.style(t, gui.Text(c, "cwd: "+sampleWorkdir)).TextColor(t.TextMuted)
			})
		})
		gui.Spacer(c)
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
			gui.Spacer(c)
			if gui.Button(c, "拒绝").Label("拒绝").Clicked() {
				a.respondApproval("deny")
			}
			if gui.PrimaryButton(c, "仅这次").Label("仅这次").Clicked() {
				a.respondApproval("allow")
			}
			if gui.Button(c, "始终允许此命令").Label("始终允许此命令").Clicked() {
				a.respondApproval("allow-always")
			}
		})
	})
}

// respondApproval 把审批决定发出去：接上事实时走 Intents，否则只记样例反馈。
func (a *App) respondApproval(decision string) {
	if a.intents == nil {
		a.approvalDecision = decision
		return
	}
	a.dispatch("审批", func(ctx context.Context) error {
		return a.intents.RespondApproval(ctx, decision)
	}, nil)
}

// questionCard 是提问卡：出现时占据输入位，批次只提交一次（U3 §5.5）。
func (a *App) questionCard(c *gui.Context, question QuestionSnapshot) {
	t := c.Theme()
	gui.Column(c).MinHeight(ApprovalHeight).Padding(space4).Gap(space2).
		Background(t.Surface).Border(strokeHairline, t.Border).
		Radius(radiusWindowCard).Label("需要你回答").Children(func() {
		iconGlyph(c, iconClock, colorAccentInk.in(t), "clock")
		typeBody.style(t, gui.Text(c, question.Text)).TextColor(t.Text)
		for _, option := range question.Options {
			option := option
			if actionButton(c, option, "选项："+option, false, false).Clicked() {
				a.questionAnswer = option
			}
		}
		gui.TextInputBase(c, &a.questionAnswer).Label("回答").Grow(1).
			Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
			gui.Spacer(c)
			answer := strings.TrimSpace(a.questionAnswer)
			if actionButton(c, "回答", "提交回答", true, answer == "").Clicked() && a.intents != nil {
				a.dispatch("回答", func(ctx context.Context) error {
					return a.intents.RespondQuestion(ctx, answer)
				}, func() { a.questionAnswer = "" })
			}
			if actionButton(c, "跳过", "跳过这个问题", false, false).Clicked() && a.intents != nil {
				a.dispatch("跳过提问", func(ctx context.Context) error {
					return a.intents.SkipQuestion(ctx)
				}, func() { a.questionAnswer = "" })
			}
		})
	})
}

// ---- 选择器：推理强度 / 模型 / 技能 / 模式 / 工作区 / 权限 ----

// effortLabel 是当前推理强度的短标签。
func (a *App) effortLabel() string {
	switch int(a.effort + 0.5) {
	case 0:
		return "低"
	case 1:
		return "中"
	case 3:
		return "最高"
	default:
		return "高"
	}
}

// effortSelector 是工具行右侧的推理强度按钮与滑杆浮层。
func (a *App) effortSelector(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.effortOpen)
	tr := m.Trigger.Height(composerToolSize).Padding(0, space1).Radius(radius).
		Label("思考强度：" + a.effortLabel()).Tooltip("思考强度").
		Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		iconBox(c, iconEffort, colorAccentInk.in(t), "effort")
		typeSecondary.style(t, gui.Text(c, a.effortLabel())).TextColor(t.Text)
	})
	m.Popup(func(panel gui.Element) {
		panel.Width(composerEffortMenuWidth).Padding(space3).Gap(space2).Radius(radiusCard).
			Background(t.Surface).Border(strokeHairline, t.Border).
			Shadow(0, 4, 16, 0, colorShadow.in(t))
		gui.Row(c).AlignItems(gui.Center).Children(func() {
			typeSecondary.style(t, gui.Text(c, "思考强度")).TextColor(t.Text)
			gui.Spacer(c)
			typeSecondary.style(t, gui.Text(c, a.effortLabel())).TextColor(t.Text)
		})
		// 滑杆只显示当前档：写思考强度需要协议方法，本版没有通道（DEC-15）。
		// 置灰后拖动不改值，原因写在滑杆下方。
		gui.Slider(c, &a.effort, 0, 3).Height(composerSliderHeight).
			Label("思考强度滑杆（" + noteThinking + "）").Disabled(true)
		typeMicro.style(t, gui.Text(c, noteThinking)).TextColor(t.TextMuted).Selectable()
	})
}

// modelSelector 是模型按钮与模型浮层（最近使用 + 模型列表，简化为两列）。
func (a *App) modelSelector(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.modelOpen)
	tr := m.Trigger.Height(composerToolSize).Padding(0, space1).Radius(radius).
		Label("模型：" + a.currentModel()).Tooltip("选择模型").
		Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		iconBox(c, iconProvider, colorAccent.in(t), "provider")
		typeSecondary.style(t, gui.Text(c, a.currentModel())).TextColor(t.Text).SingleLine()
	})
	m.Popup(func(panel gui.Element) {
		panel.Width(composerModelMenuWidth).Padding(space2).Gap(space2).Radius(radiusCard).
			Background(t.Surface).Border(strokeHairline, t.Border).
			Shadow(0, 4, 18, 0, colorShadow.in(t))
		gui.Row(c).Gap(space3).AlignItems(gui.Center).Children(func() {
			typeBody.style(t, gui.Text(c, "模型")).TextColor(t.Text).Label("模型页签")
			typeSecondary.style(t, gui.Text(c, "子代理")).TextColor(t.TextMuted)
			typeSecondary.style(t, gui.Text(c, "外部 agent")).TextColor(t.TextMuted)
		})
		gui.Box(c).Height(strokeHairline).Background(t.Border)
		gui.TextInputBase(c, &a.modelQuery).Placeholder("搜索模型...").Height(composerSearchRowHeight).
			Padding(0, space2).Radius(radius).Background(t.SurfacePressed).Label("搜索模型")
		options, currentID := a.modelOptions()
		gui.Row(c).Gap(space2).AlignItems(gui.Start).Children(func() {
			// 最近使用：服务端没有「最近用过哪些模型」的事实，能解析出来的只有当前
			// 选择，所以这一栏最多一项；没有当前选择时整栏不画。
			if recent, ok := a.recentModel(options, currentID); ok {
				gui.Column(c).Width(composerRecentWidth).Gap(space1).Children(func() {
					gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
						iconBox(c, iconClock, t.TextMuted, "recent")
						typeMicro.style(t, gui.Text(c, "最近使用")).TextColor(t.TextMuted)
					})
					if a.modelOpen {
						if menuItem(c, m, iconProvider, recent.Name, "最近使用："+recent.Name, "", false, true) {
							a.selectModel(recent)
						}
					}
				})
			}
			gui.Column(c).Grow(1).Gap(space1).Children(func() {
				for _, option := range options {
					option := option
					if !strings.Contains(strings.ToLower(option.Name), strings.ToLower(a.modelQuery)) {
						continue
					}
					if menuItem(c, m, nil, option.Name, "模型："+option.Name, "", false, option.ID == currentID) {
						a.selectModel(option)
					}
				}
			})
		})
	})
}

// sampleRecentProvider 是样例里「最近使用」栏的那一项（参考产品的 provider 名）。
const sampleRecentProvider = "DimAgent OAuth"

// modelOptions 是模型浮层的候选与当前选择：连上 Source 时读 Settings().Models，
// 样例里用静态清单（-demo 出图用）。
func (a *App) modelOptions() ([]ModelOption, string) {
	if st, ok := a.liveSettings(); ok {
		return st.Models, st.ModelID
	}
	return demoModels(), a.model
}

// demoModels 是 -demo 的静态模型清单：只有没有 Source 时才用；接上事实后浮层
// 一律读 Settings().Models，这个函数不会被调用。
func demoModels() []ModelOption {
	names := []string{
		"seed-2.1-pro", "GPT-5.6 Terra", "DeepSeek V4 Pro", "Qwen: Qwen3.8 27B",
		"MiniMax-M3", "GLM-5.3 Flash", "GPT-6 Astra", "DeepSeek: DeepSeek V4.1 Flash",
	}
	out := make([]ModelOption, 0, len(names))
	for _, name := range names {
		out = append(out, ModelOption{ID: name, Name: name})
	}
	return out
}

// recentModel 是「最近使用」栏里的那一项：样例里是固定的 provider 名；接上事实
// 时是当前选择（服务端没有另外的最近使用事实，不编造历史）。
func (a *App) recentModel(options []ModelOption, currentID string) (ModelOption, bool) {
	if a.source == nil {
		return ModelOption{ID: sampleModel, Name: sampleRecentProvider}, true
	}
	for _, option := range options {
		if currentID != "" && option.ID == currentID {
			return option, true
		}
	}
	return ModelOption{}, false
}

// selectModel 是模型浮层里的一次点选：连上事实时走 SelectModel（provider.select），
// 样例里只改视图里的当前模型。
func (a *App) selectModel(option ModelOption) {
	if a.source == nil || a.intents == nil {
		a.model = option.ID
		return
	}
	a.dispatch("选择模型", func(ctx context.Context) error {
		return a.intents.SelectModel(ctx, option.ProviderID, option.ID)
	}, nil)
}

// skillsSelector 是技能按钮与技能浮层（搜索 + 名称/描述列表）。
func (a *App) skillsSelector(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.skillsOpen)
	tr := m.Trigger.Width(composerToolSize).Height(composerToolSize).Radius(radius).
		Label("技能").Tooltip("选择技能").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() { iconBox(c, iconCube, t.TextMuted, "skills") })
	m.Popup(func(panel gui.Element) {
		panel.Width(composerSkillsMenuWidth).Padding(space2).Gap(space1).Radius(radiusCard).
			Background(t.Surface).Border(strokeHairline, t.Border).
			Shadow(0, 4, 18, 0, colorShadow.in(t))
		gui.Row(c).Height(composerSearchRowHeight).Padding(0, space2).Gap(space1).Radius(radius).
			Background(t.SurfacePressed).AlignItems(gui.Center).Children(func() {
			iconBox(c, iconSearch, t.TextMuted, "search")
			gui.TextInputBase(c, &a.skillQuery).Placeholder("搜索技能...").Grow(1).
				Label("搜索技能")
		})
		skills, readErr := a.skillEntries()
		if readErr != "" {
			typeMicro.style(t, gui.Text(c, readErr)).TextColor(t.TextMuted).SingleLine()
			return
		}
		if len(skills) == 0 {
			typeMicro.style(t, gui.Text(c, "还没有技能")).TextColor(t.TextMuted).
				Padding(space1, space2)
			return
		}
		for _, skill := range skills {
			skill := skill
			if !strings.Contains(strings.ToLower(skill.Name), strings.ToLower(a.skillQuery)) {
				continue
			}
			it := m.Item().Padding(space1, space2).Radius(radius).
				Label("技能：" + skill.Name)
			it.Children(func() {
				gui.Column(c).Grow(1).Children(func() {
					typeSecondaryStrong.style(t, gui.Text(c, skill.Name)).TextColor(t.Text).SingleLine()
					if skill.Description != "" {
						typeMicro.style(t, gui.Text(c, skill.Description)).TextColor(t.TextMuted).
							SingleLine()
					}
				})
			})
			if it.Highlighted() {
				it.Background(t.SurfaceHover)
			}
			if it.Clicked() {
				a.draft = strings.TrimSpace(a.draft + " $" + skill.Name)
			}
		}
	})
}

// skillEntries 是技能浮层的读数：连上 Source 时读 Extensions().Skills，样例里用
// 静态清单（-demo 出图用）。第二个返回值是最近一次读取失败的原因。
func (a *App) skillEntries() ([]ExtItemSnapshot, string) {
	if a.source == nil {
		return demoSkills(), ""
	}
	skills := a.source.Extensions().Skills
	return skills.Items, skills.ReadError
}

// demoSkills 是 -demo 的静态技能清单：只有没有 Source 时才用。
func demoSkills() []ExtItemSnapshot {
	return []ExtItemSnapshot{
		{Name: "agents-sdk", Description: "Build AI agents on Cloudflare Workers using the Agents SDK."},
		{Name: "background-check", Description: "Comprehensive OSINT background investigation."},
		{Name: "baoyu-format-markdown", Description: "Formats plain text or markdown files with frontmatter."},
		{Name: "baoyu-infographic", Description: "Generate professional infographics with layout types."},
		{Name: "deep-read", Description: "Read and summarize long documents."},
	}
}

// modeSelector 是 Agent / Goal 模式选择器。
func (a *App) modeSelector(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.modeOpen)
	tr := m.Trigger.Width(composerToolSize).Height(composerToolSize).Radius(radius).
		Label("模式：" + a.mode).Tooltip("Agent / Goal").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() { iconBox(c, iconChevronsR, t.TextMuted, "mode") })
	m.Popup(func(panel gui.Element) {
		panel.Width(composerModeMenuWidth).Padding(space1).Radius(radiusCard).Background(t.Surface).
			Border(strokeHairline, t.Border).Shadow(0, 4, 16, 0, colorShadow.in(t))
		if menuItem(c, m, iconChevronsR, "Agent", "Agent 模式", "", false, a.mode == "Agent") {
			a.mode = "Agent"
		}
		// Goal 模式的五个写入（设置 / 修改 / 暂停 / 恢复 / 清除）都已在目标行接通：
		// 这里只切模式，还没有目标时把目标行的编辑框打开，让「设定目标」走同一条
		// SetGoal 路径。目标归会话：没有打开会话时写不进去，按不可用画。
		_, hasSession := a.liveSession()
		goalUnavailable := a.wired() && !hasSession
		if menuItem(c, m, iconTarget, "Goal", "Goal 模式", "", goalUnavailable, a.mode == "Goal") {
			a.mode = "Goal"
			if _, _, exists := a.goalState(); !exists {
				a.goalDraft, a.goalEditing = "", true
			}
		}
		if goalUnavailable {
			typeMicro.style(t, gui.Text(c, "还没有打开会话，暂时设定不了目标")).
				TextColor(t.TextMuted).Padding(space1, space2)
		}
	})
}

// workspaceOption 是一个可选工作区：界面名与目录（空目录表示样例里的内置项）。
type workspaceOption struct {
	name string
	path string
}

// workspaceOptions 是工作区选择器的候选：连上 Source 时读服务端授权的工作区，
// 样例里用静态清单（-demo 出图用）。
func (a *App) workspaceOptions() []workspaceOption {
	if a.source == nil {
		return []workspaceOption{
			{name: "工作区"},
			{name: "year"}, {name: "dim-3"}, {name: "dim-2"}, {name: "dim"},
		}
	}
	workspaces := a.source.Workspaces()
	out := make([]workspaceOption, 0, len(workspaces))
	for _, workspace := range workspaces {
		if strings.TrimSpace(workspace.Path) == "" {
			continue
		}
		out = append(out, workspaceOption{
			name: workspaceBaseName(workspace.Path),
			path: workspace.Path,
		})
	}
	return out
}

// workspaceBaseName 是工作区的界面名：目录的最后一段；取不到时用原路径。
func workspaceBaseName(path string) string {
	base := filepath.Base(path)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return path
	}
	return base
}

// currentWorkspaceCwd 是「新建会话」要用的工作目录：当前会话的目录优先，其次
// 是空态选择器选中的工作区，再其次是授权清单的第一项。都没有时为空，由适配器
// 沿用目录首项。
func (a *App) currentWorkspaceCwd() string {
	if snap, ok := a.liveSession(); ok && snap.Cwd != "" {
		return snap.Cwd
	}
	if a.workspacePath != "" {
		return a.workspacePath
	}
	if options := a.workspaceOptions(); len(options) > 0 {
		return options[0].path
	}
	return ""
}

// workspaceSelector 是空态控制行的工作区入口。
func (a *App) workspaceSelector(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.workspaceOpen)
	tr := m.Trigger.Height(composerControlTriggerHeight).Padding(0, space1).Radius(radius).
		Label("选择工作区").Tooltip("选择工作区").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		iconBox(c, iconHome, t.TextMuted, "workspace")
		typeSecondary.style(t, gui.Text(c, "工作区")).TextColor(t.TextMuted)
		iconBox(c, iconChevronDown, t.TextMuted, "chevron")
	})
	m.Popup(func(panel gui.Element) {
		panel.Width(composerWorkspaceMenuWidth).Padding(space2).Gap(space1).Radius(radiusCard).Background(t.Surface).
			Border(strokeHairline, t.Border).Shadow(0, 4, 18, 0, colorShadow.in(t))
		gui.Row(c).Height(composerSearchRowHeight).Padding(0, space2).Gap(space1).Radius(radius).
			Background(t.SurfacePressed).AlignItems(gui.Center).Children(func() {
			iconBox(c, iconSearch, t.TextMuted, "search")
			gui.TextInputBase(c, &a.workspaceQuery).Placeholder("搜索工作区...").
				Grow(1).Label("搜索工作区")
		})
		options := a.workspaceOptions()
		if len(options) == 0 {
			typeMicro.style(t, gui.Text(c, "服务端没有授权的工作区")).TextColor(t.TextMuted).
				Padding(space1, space2)
		}
		for _, option := range options {
			option := option
			if !strings.Contains(strings.ToLower(option.name), strings.ToLower(a.workspaceQuery)) {
				continue
			}
			icon := iconFolderOpen
			label := "工作区：" + option.name
			checked := option.path != "" && option.path == a.workspacePath
			if option.path == "" {
				// 样例里的工作区没有目录：内置工作区是默认落点（参考产品停在 🏠），
				// 其余按屏内名字记选中。
				icon, label = iconHome, "内置工作区"
				checked = option.name == "工作区" || option.name == a.workspaceName
			}
			if menuItem(c, m, icon, option.name, label, "", false, checked) {
				a.workspaceName, a.workspacePath = option.name, option.path
			}
		}
		if menuItem(c, m, iconPlus, "添加新工作区", "添加新工作区", "", false, false) {
			a.workspaceName = "新工作区"
		}
	})
}

// permissionLevel 是一档权限：界面文案（SB-02）与协议 preset（D5 §4.8）。
type permissionLevel struct {
	label string
	tier  string
}

// permissionLevels 是权限选择器的三档。协议只有三档 preset（read-only /
// workspace-write / full-access）；参考产品的「默认权限」与「智能权限」都落在
// workspace-write 上（两者只差审批策略，而协议档位不表达审批策略），所以这两档
// 同值；「完全访问权限」对应 full-access。
//
// read-only 不在这张表里：它只在服务端回执明确给出时显示（见 permissionLabel），
// 不作为可选档位出现在菜单里。映射与理由见
// docs/domains/requirements/decisions.md 的 DEC-01/DEC-02。
var permissionLevels = []permissionLevel{
	{label: "默认权限", tier: "workspace-write"},
	{label: "智能权限", tier: "workspace-write"},
	{label: "完全访问权限", tier: "full-access"},
}

// permissionReadOnlyLabel 是回执给出 read-only 时的界面文案。它不可选，只在
// 当前档位显示处出现。
const permissionReadOnlyLabel = "只读"

// permissionTierOf 把界面文案换成协议取值；未知文案返回 false。
func permissionTierOf(label string) (string, bool) {
	for _, level := range permissionLevels {
		if level.label == label {
			return level.tier, true
		}
	}
	return "", false
}

// permissionLabelOf 把协议取值换成界面文案；三档之外的取值返回 false（原样显示，
// 不翻译成别名）。
func permissionLabelOf(tier string) (string, bool) {
	for _, level := range permissionLevels {
		if level.tier == tier {
			return level.label, true
		}
	}
	return "", false
}

// permissionLabel 是权限选择器显示的当前档位：连上事实时读会话档位，样例里用
// 视图里的值。两档文案共用 workspace-write，所以点过的那一档优先（见
// permissionChoice）；read-only 只在回执明确给出时显示为「只读」；其余未知取值
// 原样显示。
func (a *App) permissionLabel() string {
	st, ok := a.liveSettings()
	if !ok || !st.TierKnown || st.Tier == "" {
		return a.permission
	}
	if chosen, ok := permissionTierOf(a.permissionChoice); ok && chosen == st.Tier {
		return a.permissionChoice
	}
	if st.Tier == "read-only" {
		return permissionReadOnlyLabel
	}
	if label, ok := permissionLabelOf(st.Tier); ok {
		return label
	}
	return st.Tier
}

// choosePermission 是权限菜单里点一档：完全放开先确认（两步），其余直接生效。
func (a *App) choosePermission(level string) {
	if level == "完全访问权限" && level != a.permissionLabel() {
		a.permissionArmOpen = true
		a.permissionArmAck = false
		return
	}
	a.applyPermission(level)
}

// applyPermission 切到一档：连上事实时走 SetPermissionTier，样例里只改视图。
func (a *App) applyPermission(level string) {
	tier, known := permissionTierOf(level)
	if a.source == nil || a.intents == nil || !known {
		a.permission, a.permissionChoice = level, level
		return
	}
	a.dispatch("切换权限", func(ctx context.Context) error {
		return a.intents.SetPermissionTier(ctx, tier)
	}, func() { a.permissionChoice = level })
}

// permissionSelector 是空态控制行的权限入口。
func (a *App) permissionSelector(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.permissionOpen)
	tr := m.Trigger.Height(composerControlTriggerHeight).Padding(0, space1).Radius(radius).
		Label("权限：" + a.permissionLabel()).Tooltip("选择权限").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		iconBox(c, iconShield, t.TextMuted, "permission")
		typeSecondary.style(t, gui.Text(c, a.permissionLabel())).TextColor(t.TextMuted)
	})
	m.Popup(func(panel gui.Element) {
		panel.Width(composerPermissionMenuWidth).Padding(space1).Radius(radiusCard).Background(t.Surface).
			Border(strokeHairline, t.Border).Shadow(0, 4, 16, 0, colorShadow.in(t))
		for _, level := range permissionLevels {
			level := level
			if menuItem(c, m, iconShield, level.label, "权限："+level.label, "", false,
				level.label == a.permissionLabel()) {
				a.choosePermission(level.label)
			}
		}
	})
}

// menuItem 是一行自绘菜单项：16px 图标 + 名称 + 可选括注 + 末尾勾选。
// 返回值是这一项是否被选中（菜单由 MenuBase 负责关闭）。
func menuItem(c *gui.Context, m gui.MenuParts, icon *gui.SVG, text, label, note string,
	disabled, checked bool) bool {
	t := c.Theme()
	it := m.Item().MinHeight(rowRegular).Padding(space1, space2).Radius(radius).
		Gap(space2).AlignItems(gui.Center).Label(label).Disabled(disabled)
	if disabled {
		it = it.Opacity(0.5)
	}
	it.Children(func() {
		if icon != nil {
			iconBox(c, icon, t.TextMuted, "")
		}
		gui.Column(c).Grow(1).Children(func() {
			typeSecondary.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
			if note != "" {
				typeMicro.style(t, gui.Text(c, note)).TextColor(t.TextMuted).SingleLine()
			}
		})
		if checked {
			iconBox(c, iconCheck, t.Text, "checked")
		}
	})
	if it.Highlighted() && !disabled {
		it = it.Background(t.SurfaceHover)
	}
	return it.Clicked()
}

// attachChip 是一个待发附件：文件名（内容）+ 行内移除入口（可逆）。
func (a *App) attachChip(c *gui.Context, i int) {
	if i >= len(a.attachments) {
		return
	}
	chip := a.attachments[i]
	label := chip.name
	if chip.kind == "image" {
		label += " · 图片"
	}
	chipRow(c, label, func() {
		a.attachments = append(a.attachments[:i], a.attachments[i+1:]...)
	})
}

// attachmentEditor 是附件的行内编辑：文本走 attachment.put，图片按 base64 提交。
func (a *App) attachmentEditor(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(space1).Padding(space1, 0).Label("附件编辑").Children(func() {
		gui.Row(c).Gap(space2).Children(func() {
			if actionButton(c, "文本", "附件类型：文本", a.attachmentKind == "text", a.attachmentKind == "text").Clicked() {
				a.attachmentKind = "text"
			}
			if actionButton(c, "图片", "附件类型：图片", a.attachmentKind == "image", a.attachmentKind == "image").Clicked() {
				a.attachmentKind = "image"
			}
			gui.Spacer(c)
		})
		gui.TextInputBase(c, &a.attachmentName).Label("附件名").Placeholder("附件名").
			Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
		if a.attachmentKind == "image" {
			gui.TextInputBase(c, &a.attachmentMedia).Label("媒体类型").Placeholder("image/png").
				Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
			gui.TextInputBase(c, &a.attachmentData).Label("图片 base64").
				Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
		} else {
			gui.TextInputBase(c, &a.attachmentText).Label("附件内容").
				Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
		}
		gui.Row(c).Gap(space2).Children(func() {
			gui.Spacer(c)
			name, kind := a.attachmentName, a.attachmentKind
			if actionButton(c, "添加附件", "添加附件", true, false).Clicked() {
				if a.intents == nil {
					a.attachments = append(a.attachments, attachmentChipData{name: name, kind: kind})
					a.attachmentEditing = false
					return
				}
				if kind == "image" {
					a.attachments = append(a.attachments, attachmentChipData{
						name: name, kind: "image", mediaType: a.attachmentMedia, data: a.attachmentData,
					})
					a.attachmentData = ""
					a.attachmentEditing = false
					return
				}
				text := a.attachmentText
				var ref AttachmentRefSnapshot
				a.dispatch("添加附件", func(ctx context.Context) error {
					var err error
					ref, err = a.intents.WriteAttachment(ctx, name, text)
					return err
				}, func() {
					a.attachments = append(a.attachments, attachmentChipData{
						name: name, kind: "text", text: text, fileID: ref.ID, path: ref.Path,
					})
					a.attachmentText = ""
					a.attachmentEditing = false
				})
			}
			if actionButton(c, "取消", "取消添加附件", false, false).Clicked() {
				a.attachmentEditing = false
			}
		})
	})
}

// goalState 取当前目标与状态：连着 Source 时用会话快照，否则用样例。
func (a *App) goalState() (string, string, bool) {
	if snap, ok := a.liveSession(); ok {
		return snap.Goal, snap.GoalStatus, snap.Goal != ""
	}
	return a.goal, "", a.goal != ""
}

// goalRow 是输入区上沿的目标行（参考产品把它做成活动条；接事实时保留编辑入口）。
func (a *App) goalRow(c *gui.Context) {
	objective, status, exists := a.goalState()
	if !exists && !a.goalEditing && !a.wired() {
		return
	}
	t := c.Theme()
	if a.goalEditing {
		gui.Row(c).Height(QueueHeight).Shrink(0).Padding(0, space4).
			Gap(space2).AlignItems(gui.Center).Children(func() {
			iconGlyph(c, iconTarget, colorAccentInk.in(t), "target")
			gui.TextInputBase(c, &a.goalDraft).Grow(1).Label("目标").Placeholder(placeholderGoal).
				Padding(space1, space2).Radius(radius).Background(t.Surface)
			if actionButton(c, "保存", "保存目标", true, false).Clicked() {
				text := a.goalDraft
				verb := "新建目标"
				if a.intents == nil {
					a.goal = text
					a.goalEditing = false
					return
				}
				a.dispatch(verb, func(ctx context.Context) error {
					if exists {
						return a.intents.EditGoal(ctx, text)
					}
					return a.intents.SetGoal(ctx, text)
				}, func() { a.goalEditing = false })
			}
			if actionButton(c, "取消", "取消编辑", false, false).Clicked() {
				a.goalEditing = false
			}
		})
		return
	}
	gui.Row(c).Height(QueueHeight).Shrink(0).Padding(0, space4).
		Gap(space2).AlignItems(gui.Center).Label("目标行").Children(func() {
		iconGlyph(c, iconTarget, colorAccentInk.in(t), "target")
		color := t.Text
		if status == "complete" || status == "blocked" {
			color = t.TextMuted
		}
		text := objective
		if !exists {
			text, color = "设定目标", t.TextMuted
		}
		typeSecondary.style(t, gui.Text(c, text)).TextColor(color).Grow(1).SingleLine()
		if status == "complete" {
			iconBox(c, iconCheck, t.TextMuted, "目标已完成")
		}
		if iconButton(c, iconButtonSpec{icon: iconPencil, label: "修改目标", size: rowCompact}) {
			a.goalDraft = objective
			a.goalEditing = true
			if a.intents != nil {
				a.dispatch("读取目标", func(ctx context.Context) error {
					_, err := a.intents.ReadGoal(ctx)
					return err
				}, nil)
			}
		}
		if !exists {
			return
		}
		if status == "paused" {
			if iconButton(c, iconButtonSpec{icon: iconPlay, label: "恢复目标", size: rowCompact}) {
				if a.intents != nil {
					a.dispatch("恢复目标", a.intents.ResumeGoal, nil)
				}
			}
		} else {
			if iconButton(c, iconButtonSpec{icon: iconStop, label: "暂停目标", size: rowCompact}) {
				if a.intents != nil {
					a.dispatch("暂停目标", a.intents.PauseGoal, nil)
				}
			}
		}
		if a.goalClearConfirm {
			if textIconButton(c, textIconButtonSpec{
				icon: iconXMark, mark: "x-mark", text: "确认清除目标", label: "确认清除目标",
			}) {
				a.goalClearConfirm = false
				if a.intents != nil {
					a.dispatch("清除目标", a.intents.ClearGoal, nil)
				}
			}
		} else if iconButton(c, iconButtonSpec{icon: iconXMark, label: "清除目标", size: rowCompact}) {
			a.goalClearConfirm = true
		}
	})
}

// shellMode 是输入区的命令模式（`!` 前缀）。它是视图状态（草稿首字符），
// 不是另一块界面：输入卡仍是同一个，只换占位与发送槽语义。
func (a *App) shellMode() bool {
	return strings.HasPrefix(a.draft, "!")
}

// shellCommand 是去掉 `!` 前缀后的命令原文。
func (a *App) shellCommand() string {
	return strings.TrimSpace(strings.TrimPrefix(a.draft, "!"))
}

// runShellCommand 是命令模式的"运行"：把命令与一行结果追加到右栏终端。
func (a *App) runShellCommand() {
	cmd := a.shellCommand()
	if cmd == "" {
		return
	}
	if a.intents != nil {
		var runID string
		a.dispatch("运行命令", func(ctx context.Context) error {
			id, err := a.intents.RunShell(ctx, cmd)
			if err == nil {
				runID = id
			}
			return err
		}, func() {
			a.lastShellRun = runID
			a.draft = ""
			a.termState = termRunning
		})
		return
	}
	if a.wired() {
		return
	}
	a.termLines = append(a.termLines, termLine{text: "$ " + cmd, color: -1})
	a.termLines = append(a.termLines, termLine{text: "ok  " + cmd + " (样例)", color: 2})
	a.termState = termExited
	a.draft = ""
}

// exitShellMode 清掉 `!` 前缀（回到普通输入）。
func (a *App) exitShellMode() {
	a.draft = strings.TrimLeft(strings.TrimPrefix(a.draft, "!"), " ")
}
