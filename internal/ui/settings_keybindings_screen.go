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
// # 接了什么
//
//   - 分组「全局」接的是本端偏好与系统级注册：开关、键位与注册结果都来自
//     Source.GlobalShortcut，改动走 Intents.SetGlobalShortcut（落本地偏好并重新
//     注册系统级快捷键）。
//   - 后两组的 8 行（S-keybindings-05 … 22）接的是本地偏好
//     Preferences.Keybindings：改键、重置都落盘，重启后按改过的键渲染；按键判定
//     在 App.keybindingHit（keymap.go），主界面上真的按得动。
//   - 「提交更改」（submit-changes）本版没有落点：那一行保留形态、按钮置灰并写
//     原因（S-keybindings-22 的形态在，动作不在），不改键。
//
// # 与本文件原先的差异
//
// 原先后两组只改屏内状态，重启就丢；现在按本地偏好落盘。原先点击键帽进入
// 「录音中…」而不会真的捕获按键——那是录制态的样子货。本版改为**文本改键**：
// 点键帽展开一个输入框，输入 ⌘⇧B 或 Cmd+Shift+B 都认，规范化失败就地写原因并
// 保留原值，所以不放「录音中…」这句（没有任何东西在录音）。改键保存失败时回滚
// 到保存前的值，原因写在行下（R-KEY-203/205 的回滚口径）。

// keybindingStyle 是键帽文字：11、半粗（design-baseline §2「键帽 / 计数」）。
var keybindingStyle = fontStep{size: 11, line: 18, weight: 600}

// keybindingNameStyle / keybindingGroupStyle 是行名与分组标题：13、半粗
// （R-KEY-034 / R-KEY-026）。
var (
	keybindingNameStyle  = fontStep{size: 13, line: 19.5, weight: 600}
	keybindingGroupStyle = fontStep{size: 13, line: 19.5, weight: 600}
)

// keybindingDef 是一条快捷键的行：分组、动作 ID、名称。键位不在这里——改过的
// 取本地偏好，没改过的取出厂值（internal/shortcut 的 Actions）。
type keybindingDef struct {
	id    string
	scope string
	name  string
}

// keybindingDefs 是两组 8 条，顺序与截图一致。
var keybindingDefs = []keybindingDef{
	{"open-settings", "app", "打开设置"},
	{"new-thread", "app", "新建线程"},
	{"toggle-sidebar", "app", "切换侧边栏"},
	{"search-chat", "app", "搜索对话"},
	{"focus-composer", "chat", "聚焦输入框"},
	{"toggle-changes", "chat", "切换更改面板"},
	{"toggle-terminal", "chat", "切换终端"},
	{"submit-changes", "chat", "提交更改"},
}

// keybindingNoAction 是「这一行的动作本版没有落点」的用户可见原因：保留入口、
// 置灰、写清为什么。不出现协议方法名与「本版不做」这类内部口径。
const keybindingNoAction = "当前版本不支持这个快捷键"

// keybindingScopes 是两个分组标题，顺序与截图一致。
var keybindingScopes = []struct {
	id    string
	title string
}{
	{"app", "通用"},
	{"chat", "对话"},
}

// keybindingErrors 是行内错误的几种写法：认不出的键位、与别的行冲突、落盘失败。
const (
	keybindingErrInvalid  = "这个键位不可用：至少需要一个修饰键与一个主键"
	keybindingErrConflict = "与「%s」冲突，仍然保存了这个键位"
	keybindingErrSave     = "这个键位没有保存成功"
)

// keybindingsState 是本屏可丢的界面状态：搜索词、正在改键的行与其草稿、行内错误。
// 键位本身不在这里（那是本地偏好，见 App.keybindings）。
type keybindingsState struct {
	query string
	// editing 是正在改键的动作 ID；空表示没有在改。
	editing string
	draft   string
	// errors 是行内错误（动作 ID → 一句原因），落盘成功后清掉。
	errors map[string]string
}

var keybindingsScreen = keybindingsState{}

// 全局唤起这一行的名称、说明与输入宽（分组「全局」）。
const (
	globalShortcutName = "全局唤起"
	globalShortcutHint = "在任意应用里按下，把窗口带到前台并聚焦输入框。"
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
				a.keybindingGroups(c)
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
func (a *App) keybindingGroups(c *gui.Context) {
	for _, scope := range keybindingScopes {
		rows := keybindingRows(scope.id)
		if len(rows) == 0 {
			continue
		}
		gui.Column(c).Shrink(0).Gap(screenSpace25).Children(func() {
			keybindingGroupTitle(c, scope.title)
			a.keybindingCard(c, rows)
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
func (a *App) keybindingCard(c *gui.Context, rows []keybindingDef) {
	screenCard(c, func() {
		for i, def := range rows {
			if i > 0 {
				screenRule(c)
			}
			a.keybindingRow(c, def)
		}
	})
}

// keybindingRow 是一行：名称 + 键位按钮（或改键输入框）+ 条件出现的「重置」，
// 行下是行内错误。
func (a *App) keybindingRow(c *gui.Context, def keybindingDef) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Children(func() {
		gui.Row(c).Shrink(0).Height(screenKeyRowH).AlignItems(gui.Center).
			Role(gui.RoleListItem).Label(def.name).
			Padding(0, screenSpace5).Gap(screenSpace25).Children(func() {
			// S-keybindings-05/08/10/12/15/17/19/21 快捷键名称（打开设置、新建线程、切换侧边栏、搜索对话、聚焦输入框、切换更改面板、切换终端、提交更改）
			keybindingNameStyle.style(t, gui.Text(c, def.name)).TextColor(t.Text).SingleLine()
			gui.Spacer(c)
			if keybindingsScreen.editing == def.id {
				a.keybindingEditor(c, def)
			} else {
				a.keybindingButton(c, def)
			}
			if !a.keybindingIsDefault(def.id) && keybindingsScreen.editing != def.id {
				// S-keybindings-07 重置
				if screenButton(c, "重置", "重置 "+def.name, "", false, rowRegular) {
					a.resetKeybinding(def)
				}
			}
		})
		if err := keybindingsScreen.errors[def.id]; err != "" {
			// S-keybindings-24/26 行内错误（无效组合 / 保存失败）
			rowIndent(c, func() {
				screenHintStyle.style(t, gui.Text(c, err)).TextColor(t.Danger).
					Label(def.name + "：" + err)
			})
		}
	})
}

// rowIndent 让行内错误与行名左缘对齐（缩进一个行内边距）。
func rowIndent(c *gui.Context, body func()) {
	gui.Row(c).Shrink(0).Padding(0, screenSpace5).Children(body)
}

// keybindingDefault 是某一行没改过时的键位（内部写法）。
func keybindingDefault(id string) string { return shortcut.DefaultBinding(id) }

// keybindingValue 是这一行当前的键位（内部写法）：改过的取本地偏好，否则出厂值。
func (a *App) keybindingValue(id string) string { return a.keybindingAccel(id) }

// keybindingIsDefault 判这一行有没有改过：与出厂值的规范写法不同即改过。
func (a *App) keybindingIsDefault(id string) bool {
	return shortcut.Binding(a.keybindings, id) == keybindingDefault(id)
}

// keybindingHasAction 判这一行的动作本版有没有落点：没有落点的行置灰并写原因。
func keybindingHasAction(id string) bool { return id != "submit-changes" }

// keybindingButton 是键位按钮：显示当前键位。没有落点的动作置灰（原因在 tooltip
// 与读屏名里），点击不进入改键。
// S-keybindings-25 冲突键位（按钮描边转主色）
func (a *App) keybindingButton(c *gui.Context, def keybindingDef) {
	t := c.Theme()
	act := keybindingHasAction(def.id)
	b := gui.ButtonBase(c).Shrink(0).Height(screenControlH).MinWidth(screenKeyMinW).
		Padding(controlPadV, screenSpace25).Radius(screenRadiusCtl).
		Background(t.Background).Label("快捷键：" + def.name)
	if !act {
		b = b.Disabled(true).Opacity(noteDisabledAlpha).Tooltip(keybindingNoAction)
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
		if b.Clicked() {
			keybindingsScreen.editing = def.id
			keybindingsScreen.draft = shortcut.Label(a.keybindingValue(def.id))
		}
	}
	b.Children(func() {
		ink := t.Text
		if !act {
			ink = t.TextMuted
		}
		// S-keybindings-06/09/11/13/16/18/20/22 键位按钮（⌘, / ⌘N / ⌘B / ⌘G / ⌘L / ⌘⇧B / ⌘J / ⌘⇧↵）
		keybindingStyle.style(t, gui.Text(c, shortcut.Label(a.keybindingValue(def.id)))).
			TextColor(ink).SingleLine()
	})
}

// keybindingEditor 是改键输入态：文本输入 +「应用」「取消」；回车等同应用。
func (a *App) keybindingEditor(c *gui.Context, def keybindingDef) {
	input := screenInput(c, &keybindingsScreen.draft, "例如 ⌘⇧B", def.name+"快捷键").
		Width(globalShortcutInputW)
	input.OnSubmit(func() { a.commitKeybinding(def) })
	if screenButton(c, "应用", "应用"+def.name+"快捷键", "", false, rowRegular) {
		a.commitKeybinding(def)
	}
	if screenButton(c, "取消", "取消修改"+def.name+"快捷键", "", false, rowRegular) {
		keybindingsScreen.editing = ""
		delete(keybindingsScreen.errors, def.id)
	}
}

// commitKeybinding 校验并保存一次改键：
//
//   - 认不出的键位不保存，原值保留，行下写原因；
//   - 与别的行撞键不阻断保存（按钮保留新值），行下写冲突提示；
//   - 落盘失败回滚到保存前的值，行下写失败原因。
func (a *App) commitKeybinding(def keybindingDef) {
	errs := keybindingsScreen.errs()
	normalized, err := shortcut.Normalize(keybindingsScreen.draft)
	if err != nil {
		keybindingsScreen.editing = ""
		errs[def.id] = keybindingErrInvalid
		return
	}
	keybindingsScreen.editing = ""
	if conflict := a.keybindingConflict(def.id, normalized); conflict != "" {
		errs[def.id] = strings.Replace(keybindingErrConflict, "%s", conflict, 1)
	} else {
		delete(errs, def.id)
	}
	a.setKeybinding(def.id, normalized)
}

// keybindingConflict 返回与新键位撞车的另一行名称；没有撞车时为空。
//
// 比对按「实际按下的键」算，不按字面：⌘B 与 CmdOrCtrl+B 在 macOS 上是同一个键，
// 字面不同也要算冲突（parseAccelerator 把两者换成同一组修饰键与主键）。
func (a *App) keybindingConflict(id, accelerator string) string {
	mods, key, ok := parseAccelerator(accelerator)
	if !ok {
		return ""
	}
	for _, other := range keybindingDefs {
		if other.id == id {
			continue
		}
		otherMods, otherKey, otherOK := parseAccelerator(a.keybindingValue(other.id))
		if !otherOK {
			continue
		}
		if otherMods == mods && otherKey == key {
			return other.name
		}
	}
	return ""
}

// resetKeybinding 把一行恢复出厂值并落盘。
func (a *App) resetKeybinding(def keybindingDef) {
	delete(keybindingsScreen.errors, def.id)
	if keybindingsScreen.editing == def.id {
		keybindingsScreen.editing = ""
	}
	a.setKeybinding(def.id, keybindingDefault(def.id))
}

// setKeybinding 写本地镜像并落盘：与出厂值相同就是删掉这一项（重置）。没有
// Intents 时（-demo）只改本地镜像；落盘失败时回滚并把原因挂到行内错误。
func (a *App) setKeybinding(id, accelerator string) {
	previous := copyBindingMap(a.keybindings)
	next := make(map[string]string, len(a.keybindings)+1)
	for other, binding := range a.keybindings {
		next[other] = binding
	}
	if accelerator == "" || accelerator == keybindingDefault(id) {
		delete(next, id)
	} else {
		next[id] = accelerator
	}
	if len(next) == 0 {
		next = nil
	}
	a.keybindings = next
	a.persistKeybindings(previous, id)
}

// persistKeybindings 落盘一次快捷键偏好；失败回滚镜像并把原因交给行内错误。
func (a *App) persistKeybindings(previous map[string]string, id string) {
	if a.intents == nil {
		return
	}
	prefs := a.preferenceSnapshot()
	if samePreferences(prefs, a.lastPrefs) {
		return
	}
	a.lastPrefs = prefs
	a.dispatchQuiet("保存快捷键", func(ctx context.Context) error {
		return a.intents.SetPreferences(ctx, prefs)
	}, nil, func(string) {
		a.keybindings = previous
		a.lastPrefs.Keybindings = copyBindingMap(previous)
		errs := keybindingsScreen.errs()
		errs[id] = keybindingErrSave
	})
}

// errs 取行内错误表（按需创建）。
func (s *keybindingsState) errs() map[string]string {
	if s.errors == nil {
		s.errors = map[string]string{}
	}
	return s.errors
}
