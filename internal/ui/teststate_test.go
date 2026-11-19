package ui

import "testing"

// 本文件是"包级渲染状态"的测试侧开关：这类状态每帧被 applyTokens 按本地偏好覆写，
// 谁都不会替用例复位，于是用例之间互相串味——单独跑绿、整包跑红，或反过来。
//
// 目前的包级渲染状态有两份（见 tokens.go 的 currentCodeLook 与 markdown.go 的
// currentCodeTheme）：代码字体栈 + 字号倍率、代码配色名。它们全局一份（各窗口同值，
// 行为上就该如此），所以测试要么显式钉成自己断言的那份，要么在自己改过之后还原。

// pinCodeLook 把代码字级与代码配色钉成 look，跑完还原成用例开始前的值。
// 断言"默认那套"的用例显式传零值 AppearancePreferences，不靠测试顺序。
func pinCodeLook(t *testing.T, look AppearancePreferences) {
	t.Helper()
	prevLook, prevTheme := currentCodeLook, currentCodeTheme
	applyCodeLook(look)
	t.Cleanup(func() {
		currentCodeLook, currentCodeTheme = prevLook, prevTheme
	})
}
