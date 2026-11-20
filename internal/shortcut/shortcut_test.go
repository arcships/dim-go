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

// TestActionsCoverTwoGroups：两组 8 行的出厂键位都在，ID 不重复，默认值能规范化。
func TestActionsCoverTwoGroups(t *testing.T) {
	actions := Actions()
	if len(actions) != 8 {
		t.Fatalf("出厂动作 %d 条，想要 8 条", len(actions))
	}
	seen := map[string]bool{}
	for _, action := range actions {
		if seen[action.ID] {
			t.Fatalf("动作 ID 重复：%s", action.ID)
		}
		seen[action.ID] = true
		normalized, err := Normalize(action.Default)
		if err != nil {
			t.Fatalf("%s 的出厂键位 %q 不合法：%v", action.ID, action.Default, err)
		}
		if normalized != action.Default {
			t.Fatalf("%s 的出厂键位 %q 不是规范写法（应为 %q）", action.ID, action.Default, normalized)
		}
		if got := DefaultBinding(action.ID); got != action.Default {
			t.Fatalf("DefaultBinding(%s) = %q，想要 %q", action.ID, got, action.Default)
		}
	}
	if DefaultBinding("no-such-action") != "" {
		t.Fatal("未知动作应当没有出厂键位")
	}
}

// TestNormalizeBindingsDropsJunk：未知动作、认不出的键位、与出厂值相同的项都不落盘。
func TestNormalizeBindingsDropsJunk(t *testing.T) {
	got := NormalizeBindings(map[string]string{
		"toggle-sidebar": "⌘⌥B",         // 改过，留下规范写法
		"open-settings":  "CmdOrCtrl+,", // 与出厂值相同，不存
		"no-such-action": "Cmd+K",       // 未知动作
		"new-thread":     "N",           // 没有修饰键，认不出
	})
	if len(got) != 1 || got["toggle-sidebar"] != "Cmd+Alt+B" {
		t.Fatalf("NormalizeBindings = %v，想要只留 toggle-sidebar=Cmd+Alt+B", got)
	}
	if all := NormalizeBindings(map[string]string{"open-settings": "CmdOrCtrl+,"}); all != nil {
		t.Fatalf("全部没改时应当返回 nil，得到 %v", all)
	}
	if none := NormalizeBindings(nil); none != nil {
		t.Fatalf("空表应当返回 nil，得到 %v", none)
	}
}

// TestBindingFallsBackToDefault：没有改过的动作取出厂值；改过的取改后的值。
func TestBindingFallsBackToDefault(t *testing.T) {
	overrides := map[string]string{"toggle-terminal": "Cmd+Alt+J"}
	if got := Binding(overrides, "toggle-terminal"); got != "Cmd+Alt+J" {
		t.Fatalf("改过的键位 = %q，想要 Cmd+Alt+J", got)
	}
	if got := Binding(overrides, "toggle-sidebar"); got != "CmdOrCtrl+B" {
		t.Fatalf("没改过的键位 = %q，想要出厂值 CmdOrCtrl+B", got)
	}
	// 偏好里混进坏值时按出厂值走，不把坏值写到屏上。
	if got := Binding(map[string]string{"toggle-sidebar": "不是键位"}, "toggle-sidebar"); got != "CmdOrCtrl+B" {
		t.Fatalf("坏键位回落 = %q，想要出厂值 CmdOrCtrl+B", got)
	}
}
