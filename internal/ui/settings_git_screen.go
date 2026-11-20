package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › Git」的重建屏：元素按
// docs/domains/requirements/inventory-settings-rt.md §14（S-git-01 … 25，
// 25 行；可见 23 + 两个下拉展开项）逐行对应，行为对应
// docs/domains/requirements/requirements-settings.md 的 R-GIT-001 … 200，
// 数值取 docs/domains/requirements/design-baseline.md 与运行时 JSON 的包围盒：
//
//	页面标题 17.5（y=33，h=25）、字段标签 13/500（y=85）、说明 12（h=14）、
//	单行输入 280×32（y=113）与 703×32（y=1067）、
//	下拉 77×32（origin）/ 72×32（main）、开关 28×16（x=970）、
//	多行输入 741×100（y=680 / 882）、分组间 1px 分隔线（组后 27、组前 29）。
//
// 版式：八个分组，组间一条 1px 细线（screenRule 用面差表达，不新增描边）；
// 组内「标签 → 说明 4、说明 → 控件 8」，单标签组「标签 → 控件 10」。
//
// 只保留可丢的界面状态（gitScreen）：分支前缀、远程与基线分支的选中项、
// 两个开关、两段指令草稿、Worktree 路径。事实（仓库设置）归 App Server。
//
// 连着 Source 时这一屏的控件形态照留但全部置灰（noteGitSettings 写在原因条、
// 行下与 tooltip）：仓库设置的读写都没有通道，所以不画样例远程 / 分支清单。
// 样例状态只走 -demo（没有 Source）。
//
// # 未实现分支
//
//   - S-git-07 / S-git-11 两个下拉的展开面板已实现（行高 32、选中项对勾），
//     但远程与分支清单是样例；真实清单要 App Server 的仓库事实。
//   - R-GIT-021/077/189 的保存期校验（非法字符、无分支、路径不存在）要写入
//     路径；本屏只在进程内保留输入值，不显示假的错误。
//   - R-GIT-194 的系统目录选择器要宿主能力：路径改为手动输入（占位「手动
//     输入路径」），「选择目录」按钮禁用并写原因（DEC-10）。
//   - R-GIT-197 的无权限提示同属宿主能力。
//
// 差异：参考产品的每段说明可选中复制，本屏照做；输入框的聚焦环取主题的
// 默认焦点环（参考是主色 3px，框架是 2px，登记为可接受偏差）。

// 几何：运行时 JSON 的包围盒（CSS px）与 design-baseline §1–§4 的档位。
const (
	gitSectionGap   = space8        // 组与组之间（上一组下沿 → 分隔线 27，分隔线 → 下一组 29）
	gitLabelGap     = screenSpace25 // 单标签组：标签 → 控件（85+18 → 113）
	gitHintGap      = screenSpace1  // 标签 → 说明（202+18 → 223）
	gitControlGap   = screenSpace2  // 说明 → 控件（237 → 247）
	gitSwitchGap    = screenSpace05 // 开关行：标签 → 说明（470+18 → 489）
	gitPrefixW      = 280           // 分支前缀输入 280×32
	gitActionSize   = screenControlH
	gitTextareaH    = 100
	gitTextareaPad  = screenSpace25
	gitSelectPadH   = controlPadH
	gitSelectPadV   = controlPadV
	gitCodeSize     = 12
	gitCodeLine     = 18
	gitRuleH        = screenRuleH
	gitNotesCodeGap = screenSpace1
)

// 字阶（design-baseline §2）：字段标签 13/500、说明 12、值 12.25、代码 12。
var (
	gitLabelStyle = fontStep{size: 13, line: 18, weight: 500}
	gitHintStyle  = fontStep{size: 12, line: 14, weight: 400}
	gitValueStyle = fontStep{size: 12.25, line: 18, weight: 400}
	gitCodeStyle  = fontStep{size: gitCodeSize, line: gitCodeLine, weight: 400, family: codeFont}
)

// 本屏文案：逐字取自参考产品的 zh-CN（i18n:gitSettings.*）。
const (
	gitCopyTitle         = "Git"
	gitCopyPrefix        = "分支前缀"
	gitCopyPrefixEx      = "feature/"
	gitCopyRemote        = "默认远程仓库"
	gitCopyRemoteHint    = "commit 和 push 的默认远程目标"
	gitCopyBase          = "默认基线分支"
	gitCopyBaseHint      = "对比/合并的基准分支"
	gitCopyForce         = "始终强制推送"
	gitCopyForceHint     = "默认使用 force push（需谨慎使用）"
	gitCopyAutoPush      = "提交后自动推送"
	gitCopyCommitNotes   = "Commit 指令补充"
	gitCopyCommitHint    = "AI 生成 commit message 时的额外提示"
	gitCopyCommitEx      = "例如：使用中文，遵循 Conventional Commits 规范..."
	gitCopyPRNotes       = "PR 指令补充"
	gitCopyPRHint        = "AI 生成 PR description 时的额外提示"
	gitCopyPREx          = "例如：包含测试步骤，说明破坏性变更..."
	gitCopyWorktree      = "Worktree 根目录"
	gitCopyChooseDirA11y = "选择目录，按钮"
	gitCopyRemoteA11y    = "默认远程仓库"
	gitCopyBaseA11y      = "默认基线分支"
	// gitCopyUnread 是连接模式下两个下拉的取值位置：没有清单可列，也不摆样例。
	gitCopyUnread = "未读取"
)

// 样例数据：远程与分支清单（运行时状态是 origin / main）。
var (
	gitRemotes  = []string{"origin", "upstream"}
	gitBranches = []string{"main", "develop", "release"}
)

// gitState 是本屏可丢的界面状态。不是事实。
type gitState struct {
	prefix      string
	remote      int
	base        int
	forcePush   bool
	autoPush    bool
	commitNotes string
	prNotes     string
	worktree    string
}

var gitScreen = gitState{}

// renderGit 是设置 › Git 这一屏（设置壳的一个分节）。内容自己滚。
//
// 连着 Source 时没有任何仓库设置事实（远程、基线分支、推送策略、指令补充、
// Worktree 根目录都读不到也写不了）：参考图里的控件形态照留，但每一个都置灰并
// 在 tooltip 与卡片下方写原因（noteGitSettings）——看得见"这一屏有什么"，点不动，
// 也不摆样例远程 / 分支清单当现状。样例只在 -demo（没有 Source）出现。
func renderGit(c *gui.Context, a *App) {
	live := a != nil && a.wired()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Label(gitCopyTitle).Children(func() {
			gui.Column(c).Shrink(0).Gap(gitSectionGap).Children(func() {
				// S-git-01 页面标题「Git」
				screenTitle(c, gitCopyTitle)
				if live {
					gitNoFactsBar(c)
				}
				gitPrefixGroup(c, live)
				gitRule(c)
				gitSelectGroup(c, gitCopyRemote, gitCopyRemoteHint, &gitScreen.remote, gitRemotes, gitCopyRemoteA11y, live)
				gitRule(c)
				gitSelectGroup(c, gitCopyBase, gitCopyBaseHint, &gitScreen.base, gitBranches, gitCopyBaseA11y, live)
				gitRule(c)
				gitSwitchGroup(c, gitCopyForce, gitCopyForceHint, &gitScreen.forcePush, live)
				gitRule(c)
				gitSwitchGroup(c, gitCopyAutoPush, "", &gitScreen.autoPush, live)
				gitRule(c)
				gitNotesGroup(c, gitCopyCommitNotes, gitCopyCommitHint, &gitScreen.commitNotes, gitCopyCommitEx, live)
				gitRule(c)
				gitNotesGroup(c, gitCopyPRNotes, gitCopyPRHint, &gitScreen.prNotes, gitCopyPREx, live)
				gitRule(c)
				gitWorktreeGroup(c, live)
			})
		})
	})
}

// gitNoFactsBar 是连接模式下的原因条：整屏的每一项都读不到也写不了，原因写在
// 这里一次，每个控件的 tooltip 再写一次。
func gitNoFactsBar(c *gui.Context) {
	noteBar(c, noteGitSettings)
}

// noteBar 是设置屏共用的一行原因条：次要面 + 1px 描边，文字可选中复制。
// 它只说明"这一屏读不到什么"，不冒充控件。
func noteBar(c *gui.Context, text string) {
	t := c.Theme()
	box := gui.Column(c).Shrink(0).Radius(screenRadiusCtl).Background(t.SurfaceHover).
		Role(gui.RoleStatus).Label(text)
	box.BorderWidth(strokeHairline).BorderColor(t.Border)
	box.Children(func() {
		gui.Row(c).Shrink(0).Padding(screenSpace15, screenSpace25).Children(func() {
			typeMicro.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Selectable()
		})
	})
}

// gitRule 是分组之间的 1px 细线：1px 高的面差，不新增自绘描边。
func gitRule(c *gui.Context) {
	t := c.Theme()
	gui.Box(c).Shrink(0).Height(gitRuleH).Background(t.Border).
		Label("分隔线").AccessHidden(true)
}

// gitPrefixGroup 是分支前缀（S-git-02/03）：标签 + 280×32 输入。
func gitPrefixGroup(c *gui.Context, live bool) {
	gui.Column(c).Shrink(0).Gap(gitLabelGap).Children(func() {
		gitLabel(c, gitCopyPrefix)
		input := screenInput(c, &gitScreen.prefix, gitCopyPrefixEx, gitCopyPrefix).
			Width(gitPrefixW)
		if live {
			input = input.Disabled(true).Tooltip(noteGitSettings).Opacity(noteDisabledAlpha)
		}
	})
}

// gitSelectGroup 是一个「标签 + 说明 + 下拉」分组（S-git-04 … 11）。
func gitSelectGroup(c *gui.Context, label, hint string, selected *int, options []string,
	a11y string, live bool) {
	gui.Column(c).Shrink(0).Gap(gitControlGap).Children(func() {
		gitLabelHint(c, label, hint)
		if live {
			// 没有远程 / 分支事实：下拉照画，但取值位置写"未读取"，不摆样例清单。
			gitSelectDisabled(c, label, a11y)
			return
		}
		gitSelect(c, selected, options, a11y)
	})
}

// gitSelectDisabled 是连接模式下没有可选项的下拉：形态照参考，整钮置灰，
// 取值位置写"未读取"，原因进 tooltip。
func gitSelectDisabled(c *gui.Context, label, a11y string) {
	t := c.Theme()
	trigger := gui.ButtonBase(c).Shrink(0).AlignSelf(gui.Start).Height(screenControlH).
		Padding(gitSelectPadV, gitSelectPadH).Radius(screenRadiusCtl).
		Background(t.Background).Gap(screenSpace1).Justify(gui.Start).
		Label(a11y).Tooltip(noteGitSettings).Disabled(true).Opacity(noteDisabledAlpha)
	trigger.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	trigger.Children(func() {
		gitValueStyle.style(t, gui.Text(c, gitCopyUnread)).TextColor(t.TextMuted).SingleLine()
		screenIcon(c, screenIconChevronDown, screenIconMicro, t.TextMuted, "chevron-down")
	})
}

// gitSwitchGroup 是一个开关行（S-git-12 … 16）：左侧标签与说明、右侧开关。
//
// live 时开关照画并置灰（形态优先），原因进 tooltip 与行下说明——开关的位置
// 不代表现状，说明行写了"读不到"。
func gitSwitchGroup(c *gui.Context, label, hint string, on *bool, live bool) {
	gui.Row(c).AlignItems(gui.Center).Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(gitSwitchGap).Children(func() {
			gitLabel(c, label)
			if hint != "" {
				gitHint(c, hint)
			}
			if live {
				gitHint(c, noteGitSettings)
			}
		})
		sw := gui.Switch(c, on).Size(screenSwitchW, screenSwitchH).
			Radius(screenSwitchH / 2).Label(label)
		if live {
			sw = sw.Disabled(true).Tooltip(noteGitSettings).Opacity(noteDisabledAlpha)
		}
	})
}

// gitNotesGroup 是一段指令补充（S-git-17 … 22）：标签、说明与 100 高的
// 等宽多行输入。
func gitNotesGroup(c *gui.Context, label, hint string, value *string, placeholder string,
	live bool) {
	gui.Column(c).Shrink(0).Gap(gitNotesCodeGap).Children(func() {
		gitLabelHint(c, label, hint)
		area := gitCodeStyle.style(c.Theme(), gui.TextArea(c, value)).
			Label(label).Placeholder(placeholder).Height(gitTextareaH).
			Padding(gitTextareaPad, gitTextareaPad)
		if live {
			area = area.Disabled(true).Tooltip(noteGitSettings).Opacity(noteDisabledAlpha)
		}
	})
}

// gitWorktreeGroup 是 Worktree 根目录（S-git-23 … 25）：标签、路径输入与
// 右端「选择目录」图标按钮（输入 703 + 间距 6 + 按钮 32）。
//
// 系统目录选择器要宿主能力，本版没有：路径改为手动输入（占位「手动输入路径」），
// 右侧按钮置灰并写原因（DEC-10）。
func gitWorktreeGroup(c *gui.Context, live bool) {
	gui.Column(c).Shrink(0).Gap(gitControlGap).Children(func() {
		gitLabel(c, gitCopyWorktree)
		gui.Row(c).AlignItems(gui.Center).Gap(gitInlineGap).Children(func() {
			input := screenInput(c, &gitScreen.worktree, "手动输入路径", gitCopyWorktree).Grow(1)
			if live {
				input = input.Disabled(true).Tooltip(noteGitSettings).Opacity(noteDisabledAlpha)
			}
			if gitIconButton(c, screenIconFolderOpen, gitCopyChooseDirA11y,
				gitPickerTip(live), true) {
				// 禁用入口不会走到这里；保留分支只为了让点击语义与其余入口一致。
			}
		})
		gitHint(c, "手动输入路径（当前版本没有目录选择器）")
	})
}

// gitPickerTip 是「选择目录」的 tooltip：连着 Source 时先写仓库设置读不到，
// 再写这条入口本身要手动输入。
func gitPickerTip(live bool) string {
	if live {
		return noteGitSettings + "；" + noteDirPicker
	}
	return noteDirPicker
}

// gitInlineGap 是输入与右端按钮之间（959 → 966）。
const gitInlineGap = screenSpace15

// gitLabel 是一个字段标签：13/500 前景色。
func gitLabel(c *gui.Context, text string) {
	t := c.Theme()
	gitLabelStyle.style(t, gui.Text(c, text)).TextColor(t.Text).Label(text).SingleLine()
}

// gitHint 是一句说明：12 号次级前景，可选中复制。
func gitHint(c *gui.Context, text string) {
	t := c.Theme()
	gitHintStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Selectable()
}

// gitLabelHint 是「标签 + 说明」两行（相距 4）。
func gitLabelHint(c *gui.Context, label, hint string) {
	gui.Column(c).Shrink(0).Gap(gitHintGap).Children(func() {
		gitLabel(c, label)
		if hint != "" {
			gitHint(c, hint)
		}
	})
}

// gitSelect 是取值下标的自绘下拉：触发区宽随内容，弹出面板选中项带对勾
// （S-git-07 / 11：行高 32、字号 13、对勾一个且为主色）。
func gitSelect(c *gui.Context, selected *int, options []string, label string) {
	t := c.Theme()
	parts := gui.SelectBase[int](c, selected)
	trigger := parts.Trigger.Shrink(0).AlignSelf(gui.Start).Height(screenControlH).
		Padding(gitSelectPadV, gitSelectPadH).Radius(screenRadiusCtl).
		Background(t.Background).Gap(screenSpace1).Justify(gui.Start).Label(label)
	trigger.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	trigger.Children(func() {
		gitValueStyle.style(t, gui.Text(c, options[*selected])).TextColor(t.Text).SingleLine()
		screenIcon(c, screenIconChevronDown, screenIconMicro, t.TextMuted, "chevron-down")
	})
	parts.Popup(func(panel gui.Element) {
		panel.Padding(space1).Radius(screenRadiusCard).Background(t.Background)
		panel.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
		})
		for i, option := range options {
			item := parts.Item(i).Height(screenControlH).
				Padding(gitSelectPadV, screenSpace25).Radius(screenRadiusCtl).Label(option)
			color := t.Text
			switch {
			case item.Highlighted():
				item = item.Background(t.Accent)
				color = t.AccentText
			case i == *selected:
				item = item.Background(t.Surface)
			}
			item.Children(func() {
				gui.Row(c).Gap(screenSpace1).AlignItems(gui.Center).Children(func() {
					if i == *selected {
						screenIcon(c, iconCheck, screenIconMicro, color, "check")
					} else {
						gui.Box(c).Width(screenIconMicro).Shrink(0)
					}
					gitValueStyle.style(t, gui.Text(c, option)).TextColor(color).SingleLine()
				})
			})
		}
	})
}

// gitIconButton 是一个 32×32 的描边图标入口（S-git-25）。tip 非空时写进
// tooltip；disabled 时整钮置灰，点击不返回真。
func gitIconButton(c *gui.Context, icon *gui.SVG, label, tip string, disabled bool) bool {
	t := c.Theme()
	if tip == "" {
		tip = label
	}
	b := gui.ButtonBase(c).Shrink(0).Width(gitActionSize).Height(gitActionSize).
		Radius(screenRadiusCtl).Background(t.Background).Center().
		Label(label).Tooltip(tip).Cursor(gui.CursorPointer).Disabled(disabled)
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	ink := t.TextMuted
	switch {
	case disabled:
		b = b.Opacity(0.5)
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
