package main

import (
	"testing"

	"github.com/egoist/mygo"

	"github.com/arcships/dimagent-mygo/internal/tray"
	appui "github.com/arcships/dimagent-mygo/internal/ui"
)

// TestTrayMenuMapping 核对托盘模型到 mygo 菜单的转接：普通项、分隔线、勾选项与
// 禁用项都按模型来，菜单里没有空标签。
func TestTrayMenuMapping(t *testing.T) {
	state := tray.State{
		WindowVisible: true,
		CloseToTray:   true,
		Recent: []tray.RecentSession{
			{ID: "sess-a", Title: "修发送路径"},
			{ID: "sess-b", Title: ""},
		},
	}
	host := &shell{}
	menu := host.trayMenu(tray.Menu(state))
	items := menu.Items()
	if len(items) == 0 {
		t.Fatal("托盘菜单是空的")
	}

	var labels []string
	var checkbox *mygo.MenuItem
	var separators int
	for _, item := range items {
		if item.Type == mygo.MenuItemSeparator {
			separators++
			continue
		}
		if item.Label == "" {
			t.Fatalf("菜单项没有标签：%+v", item)
		}
		labels = append(labels, item.Label)
		if item.Type == mygo.MenuItemCheckbox {
			checkbox = item
		}
	}
	if separators != 3 {
		t.Fatalf("分隔线数量：%d，想要 3", separators)
	}
	if checkbox == nil || checkbox.Label != "关闭窗口时最小化到托盘" || !checkbox.Checked {
		t.Fatalf("关窗偏好项应是勾选态的复选框：%+v", checkbox)
	}
	want := []string{"隐藏主窗口", "新建会话", "最近会话", "修发送路径", "sess-b",
		"关闭窗口时最小化到托盘", "退出"}
	if len(labels) != len(want) {
		t.Fatalf("菜单项：%v，想要 %v", labels, want)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("第 %d 项：%q，想要 %q", i, labels[i], want[i])
		}
	}
}

// TestRecentSessionsTruncated 核对托盘只拿前 5 条最近会话，空列表也不出错。
func TestRecentSessionsTruncated(t *testing.T) {
	items := []appui.HostSession{
		{ID: "a", Title: "A"}, {ID: "b", Title: "B"}, {ID: "c", Title: "C"},
		{ID: "d", Title: "D"}, {ID: "e", Title: "E"}, {ID: "f", Title: "F"},
	}
	got := recentSessions(items)
	if len(got) != tray.RecentLimit || got[tray.RecentLimit-1].ID != "e" {
		t.Fatalf("最近会话没有截到前 %d 条：%+v", tray.RecentLimit, got)
	}
	if got := recentSessions(nil); len(got) != 0 {
		t.Fatalf("没有最近会话时应为空：%+v", got)
	}
}
