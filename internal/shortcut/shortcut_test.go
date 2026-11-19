package shortcut

import "testing"

// TestNormalizeKeepsOneSpelling：同一组键位只有一种规范写法；字形、大小写与
// 书写顺序都不改变结果。
func TestNormalizeKeepsOneSpelling(t *testing.T) {
	cases := []struct{ input, want string }{
		{"⌘⇧Space", "Cmd+Shift+Space"},
		{"CmdOrCtrl+Shift+Space", DefaultAccelerator},
		{"cmdorctrl+shift+space", "CmdOrCtrl+Shift+Space"},
		{"Shift+CmdOrCtrl+Space", "CmdOrCtrl+Shift+Space"},
		{"⌃⌥K", "Ctrl+Alt+K"},
		{"Cmd+,", "Cmd+,"},
		{"⌘↵", "Cmd+Enter"},
		{"⌘ ⇧ Space", "Cmd+Shift+Space"},
	}
	for _, c := range cases {
		got, err := Normalize(c.input)
		if err != nil {
			t.Fatalf("Normalize(%q) 失败：%v", c.input, err)
		}
		if got != c.want {
			t.Fatalf("Normalize(%q) = %q，想要 %q", c.input, got, c.want)
		}
	}
	if _, err := Normalize("⌘"); err == nil {
		t.Fatal("只有一个修饰键的键位应当被拒绝")
	}
}

// TestNormalizeRejectsBareAndUnknown：裸键、缺主键、认不出的名字都拒绝——全局
// 快捷键至少要有一个修饰键，否则会吃掉整个系统的输入。
func TestNormalizeRejectsBareAndUnknown(t *testing.T) {
	cases := []struct{ input, why string }{
		{"", "空键位"},
		{"K", "裸键"},
		{"Shift", "只有修饰键"},
		{"Cmd+Shift", "缺主键"},
		{"Cmd+Hyper+K", "认不出的修饰键"},
		{"Cmd+Bogus", "认不出的按键"},
	}
	for _, c := range cases {
		if _, err := Normalize(c.input); err == nil {
			t.Fatalf("%s（%q）应当被拒绝", c.why, c.input)
		}
	}
}

// TestLabelUsesGlyphsOnDarwin：macOS 上用字形（⌘⇧Space），其他平台用加号写法。
func TestLabelUsesGlyphsOnDarwin(t *testing.T) {
	if got := LabelFor(DefaultAccelerator, "darwin"); got != "⌘⇧Space" {
		t.Fatalf("macOS 默认键位 = %q，想要 ⌘⇧Space", got)
	}
	if got := LabelFor("Cmd+Enter", "darwin"); got != "⌘↵" {
		t.Fatalf("macOS 回车键位 = %q，想要 ⌘↵", got)
	}
	if got := LabelFor(DefaultAccelerator, "windows"); got != "Ctrl+Shift+Space" {
		t.Fatalf("非 macOS 默认键位 = %q，想要 Ctrl+Shift+Space", got)
	}
	if got := LabelFor("", "darwin"); got != "⌘⇧Space" {
		t.Fatalf("空键位应按默认取值，得到 %q", got)
	}
}
