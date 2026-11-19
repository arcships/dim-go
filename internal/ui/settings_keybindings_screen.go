package ui

import (
	"context"
	"strings"

	gui "github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/shortcut"
)

// 本文件是「设置 › 快捷键」的重建屏：元素按
// docs/domains/requirements/inventory-settings-rt.md §6（S-keybindings-01 … 27，
// 27 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md（快捷键行距 60、键帽高 32、
// 卡片圆角 8、搜索框高 32）。
//
// 三组：
//
//	全局：全局唤起 ⌘⇧Space（开关 + 可改键 + 注册失败原因）
//	通用：打开设置 ⌘, / 新建线程 ⌘N / 切换侧边栏 ⌘B / 搜索对话 ⌘G
//	对话：聚焦输入框 ⌘L / 切换更改面板 ⌘⇧B / 切换终端 ⌘J / 提交更改 ⌘⇧↵
//
// 后两组的 8 行取自截图 runtime/settings/快捷键-top.png，仍然只改界面状态
// （keybindingsScreen）：保存这些绑定需要宿主按键捕获与设置领域的写路径。
//
// 分组「全局」是接线的：开关、键位与注册结果都来自 Source.GlobalShortcut，
// 改动走 Intents.SetGlobalShortcut（落本地偏好并重新注册系统级快捷键）。
// 录制按键要宿主捕获键盘，本批用文本输入改键（输入 ⌘⇧Space 或 Cmd+Shift+Space
// 都认）。

// keybindingStyle 是键帽文字：11、半粗（design-baseline §2「键帽 / 计数」）。
var keybindingStyle = fontStep{size: 11, line: 18, weight: 600}

// keybindingNameStyle / keybindingGroupStyle 是行名与分组标题：13、半粗
// （R-KEY-034 / R-KEY-026）。
var (
	keybindingNameStyle  = fontStep{size: 13, line: 19.5, weight: 600}
	keybindingGroupStyle = fontStep{size: 13, line: 19.5, weight: 600}
)

// keybindingDef 是一条快捷键：分组、名称、默认绑定。
type keybindingDef struct {
	id      string
	scope   string
	name    string
	binding string
}

// keybindingDefs 是两组 8 条，顺序与截图一致。
var keybindingDefs = []keybindingDef{
	{"open-settings", "app", "打开设置", "⌘,"},
	{"new-thread", "app", "新建线程", "⌘N"},
	{"toggle-sidebar", "app", "切换侧边栏", "⌘B"},
	{"search-chat", "app", "搜索对话", "⌘G"},
	{"focus-composer", "chat", "聚焦输入框", "⌘L"},
	{"toggle-changes", "chat", "切换更改面板", "⌘⇧B"},
	{"toggle-terminal", "chat", "切换终端", "⌘J"},
	{"submit-changes", "chat", "提交更改", "⌘⇧↵"},
}

// keybindingScopes 是两个分组标题，顺序与截图一致。
var keybindingScopes = []struct {
	id    string
	title string
}{
	{"app", "通用"},
	{"chat", "对话"},
}

// keybindingsState 是本屏可丢的界面状态：搜索词、录音中的行、改过的键位。
type keybindingsState struct {
	query     string
	recording string
	bindings  map[string]string
}

var keybindingsScreen = keybindingsState{
	bindings: map[string]string{},
}

// 全局唤起这一行的名称、说明与输入宽（分组「全局」）。
const (
	globalShortcutName = "全局唤起"
	globalShortcutHint = "在任意应用里按下，把窗口带到前台并聚焦输入区。"
	// globalShortcutInputW 是改键输入框的宽：够放 18 个键位字符。
	globalShortcutInputW = 180
)

// globalShortcutEdit 是本屏全局键位的编辑态（可丢的界面状态）：是否在编辑、
// 输入框草稿、就地错误（键位认不出来）。
type globalShortcutEditState struct {
	editing bool
	draft   string
	err     string
}

var globalShortcutEdit = globalShortcutEditState{}

// renderKeybindings 是设置 › 快捷键这一屏（设置壳的一个分节）。
func (a *App) renderKeybindings(c *gui.Context) {
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(screenSpace5).Children(func() {
				// S-keybindings-01 分节标题「快捷键」
				screenTitle(c, "快捷键")
				keybindingSearch(c)
				a.keybindingGlobalGroup(c)
				keybindingGroups(c)
			})
		})
	})
}

// keybindingGlobalGroup 是分组「全局」：全局唤起这一行接的是本端偏好与系统级
// 注册（Source.GlobalShortcut / Intents.SetGlobalShortcut），开关、键位与注册
// 失败原因都是真实读数，不是样例。
func (a *App) keybindingGlobalGroup(c *gui.Context) {
	t := c.Theme()
	query := strings.TrimSpace(keybindingsScreen.query)
	if query != "" && !strings.Contains(globalShortcutName, query) {
		return
	}
	status := a.globalShortcut()
	gui.Column(c).Shrink(0).Gap(screenSpace25).Children(func() {
		keybindingGroupTitle(c, "全局")
		screenCard(c, func() {
			gui.Column(c).Padding(screenSpace35, screenSpace5).Gap(screenSpace15).
				Children(func() {
					gui.Row(c).AlignItems(gui.Center).Gap(screenSpace25).Children(func() {
						gui.Column(c).Grow(1).MinWidth(0).Gap(screenSpace05).Children(func() {
							keybindingNameStyle.style(t, gui.Text(c, globalShortcutName)).
								TextColor(t.Text).SingleLine()
							screenHint(c, globalShortcutHint)
						})
						if globalShortcutEdit.editing {
							a.globalShortcutEditor(c, status)
						} else {
							a.globalShortcutKey(c, status)
						}
						on := status.Enabled
						sw := gui.Switch(c, &on).Size(screenSwitchW, screenSwitchH).
							Radius(screenSwitchH / 2).Label("全局唤起开关")
						if sw.Changed() {
							a.applyGlobalShortcut(on, status.Accelerator)
						}
					})
					if note := globalShortcutNote(status); note != "" {
						globalShortcutNoteView(c, note, globalShortcutEdit.err != "")
					}
				})
		})
	})
}

// globalShortcutKey 是键位按钮：显示当前键位，点击进入改键（文本输入）。
func (a *App) globalShortcutKey(c *gui.Context, status GlobalShortcutSnapshot) {
	t := c.Theme()
	text := status.Label
	if strings.TrimSpace(text) == "" {
		text = "未设置"
	}
	b := gui.ButtonBase(c).Shrink(0).Height(screenControlH).MinWidth(screenKeyMinW).
		Padding(controlPadV, screenSpace25).Radius(screenRadiusCtl).
		Background(t.Background).Label("全局唤起快捷键")
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	if b.Clicked() {
		globalShortcutEdit = globalShortcutEditState{editing: true, draft: status.Accelerator}
	}
	b.Children(func() {
		keybindingStyle.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
	})
}

// globalShortcutEditor 是改键的输入态：文本输入 +「应用」「取消」。回车等同
// 应用（输入框自己带提交）。
func (a *App) globalShortcutEditor(c *gui.Context, status GlobalShortcutSnapshot) {
	input := screenInput(c, &globalShortcutEdit.draft, "例如 ⌘⇧Space", "全局快捷键键位").
		Width(globalShortcutInputW)
	input.OnSubmit(func() { a.commitGlobalShortcut(status) })
	if screenButton(c, "应用", "应用全局快捷键", "", false, screenControlH) {
		a.commitGlobalShortcut(status)
	}
	if screenButton(c, "取消", "取消修改全局快捷键", "", false, screenControlH) {
		globalShortcutEdit = globalShortcutEditState{}
	}
}

// commitGlobalShortcut 把输入框里的键位按当前开关交给写路径；认不出来的键位
// 留在原地并就地写原因，不改开关。
func (a *App) commitGlobalShortcut(status GlobalShortcutSnapshot) {
	a.applyGlobalShortcut(status.Enabled, globalShortcutEdit.draft)
	if globalShortcutEdit.err == "" {
		globalShortcutEdit = globalShortcutEditState{}
	}
}

// applyGlobalShortcut 保存开关与键位：接上 Intents 时走 SetGlobalShortcut
// （落本地偏好并重新注册），样例模式只改本地的开关与键位。
func (a *App) applyGlobalShortcut(enabled bool, accelerator string) {
	normalized, err := shortcut.Normalize(accelerator)
	if err != nil {
		globalShortcutEdit.err = "键位无法识别：" + err.Error()
		return
	}
	globalShortcutEdit.err = ""
	if a.intents == nil {
		a.globalShortcutOn = enabled
		a.globalShortcutAccel = normalized
		return
	}
	a.dispatch("保存全局快捷键", func(ctx context.Context) error {
		return a.intents.SetGlobalShortcut(ctx, enabled, normalized)
	}, nil)
}

// globalShortcutNote 是这一项下面的说明行：就地错误 → 注册失败原因 → 关闭说明；
// 正常注册时不占行。
func globalShortcutNote(status GlobalShortcutSnapshot) string {
	if globalShortcutEdit.err != "" {
		return globalShortcutEdit.err
	}
	if !status.Enabled {
		return "已关闭：在其它应用里按下不会唤起。"
	}
	if status.Reason != "" {
		return "无法注册 " + status.Label + "：" + status.Reason
	}
	return ""
}

// globalShortcutNoteView 是说明行本身：错误用危险色，其余用次级前景。
func globalShortcutNoteView(c *gui.Context, text string, danger bool) {
	t := c.Theme()
	ink := t.TextMuted
	if danger {
		ink = t.Danger
	}
	screenHintStyle.style(t, gui.Text(c, text)).TextColor(ink)
}

// keybindingSearch 是搜索框：只有占位「搜索快捷键」，无前置图标、无省略号，宽 392。
func keybindingSearch(c *gui.Context) {
	t := c.Theme()
	// S-keybindings-03 搜索快捷键…
	box := gui.Row(c).Shrink(0).Width(screenSearchW).Height(screenControlH).
		AlignItems(gui.Center).Gap(screenSpace15).
		Padding(screenSpace15, screenSpace25).Radius(screenRadiusCtl).
		Background(fieldFill(t)).Label("搜索快捷键")
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	box.Children(func() {
		gui.TextInputBase(c, &keybindingsScreen.query).Grow(1).
			Label("搜索快捷键").Placeholder("搜索快捷键")
	})
}

// keybindingGroups 是两组：标题 + 卡片，命中的行保留，整组无命中就不渲染。
// S-keybindings-04 / S-keybindings-14 分组标题「通用」「对话」
func keybindingGroups(c *gui.Context) {
	for _, scope := range keybindingScopes {
		scope := scope
		rows := keybindingRows(scope.id)
		if len(rows) == 0 {
			continue
		}
		gui.Column(c).Shrink(0).Gap(screenSpace25).Children(func() {
			keybindingGroupTitle(c, scope.title)
			keybindingCard(c, rows)
		})
	}
}

// keybindingRows 返回某个分组里命中搜索词的行。
func keybindingRows(scope string) []keybindingDef {
	out := make([]keybindingDef, 0, len(keybindingDefs))
	query := strings.TrimSpace(keybindingsScreen.query)
	for _, def := range keybindingDefs {
		if def.scope != scope {
			continue
		}
		if query != "" && !strings.Contains(def.name, query) {
			continue
		}
		out = append(out, def)
	}
	return out
}

// keybindingGroupTitle 是分组标题：13、半粗、次级前景。
func keybindingGroupTitle(c *gui.Context, text string) {
	t := c.Theme()
	keybindingGroupStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted).
		Role(gui.RoleHeading).Level(3).Label(text)
}

// keybindingCard 是一组的卡：每行 60 高，行间 1px 细线。
func keybindingCard(c *gui.Context, rows []keybindingDef) {
	screenCard(c, func() {
		for i, def := range rows {
			if i > 0 {
				screenRule(c)
			}
			keybindingRow(c, def)
		}
	})
}

// keybindingRow 是一行：名称 + 键位按钮 + 条件出现的「重置」。
func keybindingRow(c *gui.Context, def keybindingDef) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Height(screenKeyRowH).AlignItems(gui.Center).
		Role(gui.RoleListItem).Label(def.name).
		Padding(0, screenSpace5).Gap(screenSpace25).Children(func() {
		// S-keybindings-05/08/10/12/15/17/19/21 快捷键名称（打开设置、新建线程、切换侧边栏、搜索对话、聚焦输入框、切换更改面板、切换终端、提交更改）
		keybindingNameStyle.style(t, gui.Text(c, def.name)).TextColor(t.Text).SingleLine()
		gui.Spacer(c)
		keybindingButton(c, def)
		if keybindingBinding(def) != def.binding {
			// S-keybindings-07 重置
			if screenButton(c, "重置", "重置 "+def.name, "", false, rowRegular) {
				delete(keybindingsScreen.bindings, def.id)
				if keybindingsScreen.recording == def.id {
					keybindingsScreen.recording = ""
				}
			}
		}
	})
}

// keybindingBinding 返回当前绑定：改过的取界面状态，否则取默认。
func keybindingBinding(def keybindingDef) string {
	if binding, ok := keybindingsScreen.bindings[def.id]; ok {
		return binding
	}
	return def.binding
}

// S-keybindings-25 冲突键位（按钮描边转主色）
// keybindingButton 是键位按钮：描边小按钮、高 32、最小宽 45；录音态显示
// 「录音中…」，再次点击退出并保留原值。
func keybindingButton(c *gui.Context, def keybindingDef) {
	t := c.Theme()
	recording := keybindingsScreen.recording == def.id
	text := keybindingBinding(def)
	if recording {
		// S-keybindings-23 录音中…
		text = "录音中…"
	}
	minWidth := float32(0)
	if !recording {
		minWidth = screenKeyMinW
	}
	// S-keybindings-06/09/11/13/16/18/20/22 键位按钮（⌘, / ⌘N / ⌘B / ⌘G / ⌘L / ⌘⇧B / ⌘J / ⌘⇧↵）
	b := gui.ButtonBase(c).Shrink(0).Height(screenControlH).MinWidth(minWidth).
		Padding(controlPadV, screenSpace25).Radius(screenRadiusCtl).
		Background(t.Background).Label("快捷键：" + def.name)
	if recording {
		b = b.Background(t.Surface).Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Accent, screenRadiusCtl, screenStroke)
		})
	} else {
		b = b.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
		})
		switch {
		case b.Pressed():
			b = b.Background(t.SurfacePressed)
		case b.Hovered():
			b = b.Background(t.SurfaceHover)
		}
	}
	if b.Clicked() {
		if recording {
			keybindingsScreen.recording = ""
		} else {
			keybindingsScreen.recording = def.id
		}
	}
	b.Children(func() {
		keybindingStyle.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
	})
}
