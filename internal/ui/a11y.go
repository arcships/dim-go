package ui

import gui "github.com/egoist/mygo/ui"

// 本文件是读屏语义的公共入口：只补可访问名、角色与状态，不改外观与布局。
//
// 框架的 gui.Modal 自带一个没有名字的 dialog 角色；窗口里的对话框都是「一块
// 自绘面板 + 一句标题」，标题与角色由面板自己给。modalShell 收掉外壳那个无名
// dialog，读屏不会先遇到「对话框（无名）」再遇到真正的那个。
func modalShell(m gui.Element) gui.Element { return m.Role(gui.RoleNone) }
