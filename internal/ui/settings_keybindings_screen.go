package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 快捷键」的重建屏：元素按
// docs/domains/requirements/inventory-settings-rt.md §6（S-keybindings-01 … 27，
// 27 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md（快捷键行距 60、键帽高 32、
// 卡片圆角 8、搜索框高 32）。
//
// 两组 8 行与默认键位取自截图 runtime/settings/快捷键-top.png：
//
//	通用：打开设置 ⌘, / 新建线程 ⌘N / 切换侧边栏 ⌘B / 搜索对话 ⌘G
//	对话：聚焦输入框 ⌘L / 切换更改面板 ⌘⇧B / 切换终端 ⌘J / 提交更改 ⌘⇧↵
//
// 只保留可丢的界面状态（keybindingsScreen）：搜索词、正在录音的行、改过的
// 键位。写事实（保存绑定）需要设置领域与宿主按键捕获，本屏只改界面状态。

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

// renderKeybindings 是设置 › 快捷键这一屏（设置壳的一个分节）。
func renderKeybindings(c *gui.Context) {
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(screenSpace5).Children(func() {
				screenTitle(c, "快捷键")
				keybindingSearch(c)
				keybindingGroups(c)
			})
		})
	})
}

// keybindingSearch 是搜索框：只有占位「搜索快捷键」，无前置图标、无省略号，宽 392。
func keybindingSearch(c *gui.Context) {
	t := c.Theme()
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
	keybindingGroupStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Label(text)
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
		Padding(0, screenSpace5).Gap(screenSpace25).Children(func() {
		keybindingNameStyle.style(t, gui.Text(c, def.name)).TextColor(t.Text).SingleLine()
		gui.Spacer(c)
		keybindingButton(c, def)
		if keybindingBinding(def) != def.binding {
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

// keybindingButton 是键位按钮：描边小按钮、高 32、最小宽 45；录音态显示
// 「录音中…」，再次点击退出并保留原值。
func keybindingButton(c *gui.Context, def keybindingDef) {
	t := c.Theme()
	recording := keybindingsScreen.recording == def.id
	text := keybindingBinding(def)
	if recording {
		text = "录音中…"
	}
	minWidth := float32(0)
	if !recording {
		minWidth = screenKeyMinW
	}
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
