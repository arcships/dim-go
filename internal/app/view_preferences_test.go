package app

import (
	"reflect"
	"testing"
)

// TestViewPreferencesRoundTrip 视图状态（左栏收起、视图口径、分组折叠、右栏开合
// 与标签页、两层筛选）随偏好落盘，重建后原样读回。
func TestViewPreferencesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	first, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	want := Preferences{
		Theme: "明",
		View: ViewPreferences{
			SidebarCollapsed:    true,
			SidebarFlat:         true,
			SidebarByCreated:    true,
			CollapsedGroups:     []string{"builtin", "local:/work/dim-agent"},
			RightOpen:           true,
			RightTab:            1,
			RightTabs:           []int{0, 1},
			RightExpanded:       true,
			RightWidth:          420,
			FilesFilter:         "comp",
			ChangesScope:        1,
			ChangesGroupsClosed: []string{"staged"},
		},
	}
	if err := first.UpdatePreferences(want); err != nil {
		t.Fatalf("写入偏好：%v", err)
	}

	second, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("重建 App：%v", err)
	}
	got := second.Preferences()
	if !reflect.DeepEqual(got.View, want.View) {
		t.Fatalf("视图状态往返不一致:\n got=%+v\nwant=%+v", got.View, want.View)
	}
	// 本端登记的目录与视图状态互不覆盖。
	if len(got.Workspaces) != 0 {
		t.Fatalf("视图状态写坏了工作区登记：%+v", got.Workspaces)
	}
}

// TestViewPreferencesNormalized：负的标签与宽度按 0，重复的分组标识只留一份。
func TestViewPreferencesNormalized(t *testing.T) {
	prefs := normalizePreferences(Preferences{View: ViewPreferences{
		RightTab:        -1,
		RightWidth:      -8,
		RightTabs:       []int{2, 2, -1, 0},
		CollapsedGroups: []string{"builtin", "builtin", " "},
	}})
	if prefs.View.RightTab != 0 || prefs.View.RightWidth != 0 {
		t.Fatalf("负值没有归一：%+v", prefs.View)
	}
	if !reflect.DeepEqual(prefs.View.RightTabs, []int{2, 0}) {
		t.Fatalf("标签页去重结果 = %v，想要 [2 0]", prefs.View.RightTabs)
	}
	if !reflect.DeepEqual(prefs.View.CollapsedGroups, []string{"builtin"}) {
		t.Fatalf("分组标识去重结果 = %v，想要 [builtin]", prefs.View.CollapsedGroups)
	}
}
