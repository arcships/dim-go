package ui

// 扩展的详情与批量选择所需的动作（数据侧的几条小函数）。

// openExtDetail 打开某一项的详情（按当前页签定位）。
func (a *App) openExtDetail(name string) {
	items := a.extItems(extKind(a.extTab))
	for i := range items {
		if items[i].name == name {
			a.extDetail = i
			a.extUninstallConfirm = false
			return
		}
	}
}

// currentExt 是详情页当前指向的那一项。
func (a *App) currentExt() *extItem {
	items := a.extItems(extKind(a.extTab))
	if a.extDetail < 0 || a.extDetail >= len(items) {
		return nil
	}
	return &items[a.extDetail]
}

// removeExt 卸载（样例：从当前页签的列表里移除）。
func (a *App) removeExt(i int) {
	items := a.extItems(extKind(a.extTab))
	if i < 0 || i >= len(items) {
		return
	}
	items = append(items[:i], items[i+1:]...)
	a.setExtItems(extKind(a.extTab), items)
}

// setExtItems 把某一类的列表写回。
func (a *App) setExtItems(kind extKind, items []extItem) {
	switch kind {
	case extSkills:
		a.skills = items
	case extPlugins:
		a.plugins = items
	case extMCP:
		a.mcps = items
	default:
		a.hooks = items
	}
}

// setSelected 批量启停。可逆（再点一次就能改回），因此不做确认——理由见
// U3-actions 的 AC-33：影响面是多行，但每行都可以立刻改回，且批量本身是用户的
// 明确选择；加确认只会让"选错一行"更难纠正。
func (a *App) setSelected(on bool) {
	items := a.extItems(extKind(a.extTab))
	for i := range items {
		if a.extSelected[items[i].name] {
			items[i].on = on
		}
	}
	a.setExtItems(extKind(a.extTab), items)
	a.selectMode = false
	a.extSelected = map[string]bool{}
}
