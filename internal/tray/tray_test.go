package tray

import (
	"bytes"
	"image/png"
	"strings"
	"testing"
)

// TestMenuShape 核对托盘菜单的项与顺序：显示/隐藏 → 新建会话 → 最近会话（前 5）
// → 关窗偏好 → 退出，且分隔线只在分组的边界。
func TestMenuShape(t *testing.T) {
	state := State{
		WindowVisible: true,
		CloseToTray:   true,
		Recent: []RecentSession{
			{ID: "s1", Title: "修发送路径"},
			{ID: "s2", Title: "补托盘"},
			{ID: "s3", Title: "排查连接"},
			{ID: "s4", Title: "整理文档"},
			{ID: "s5", Title: "跑回归"},
			{ID: "s6", Title: "不该出现"},
			{ID: "s7", Title: "也不该出现"},
		},
	}
	items := Menu(state)
	var labels []string
	for _, item := range items {
		if item.Separator {
			labels = append(labels, "---")
			continue
		}
		labels = append(labels, item.Label)
	}
	want := []string{
		"隐藏主窗口", "新建会话", "---", "最近会话",
		"修发送路径", "补托盘", "排查连接", "整理文档", "跑回归",
		"---", "关闭窗口时最小化到托盘", "---", "退出",
	}
	if strings.Join(labels, "/") != strings.Join(want, "/") {
		t.Fatalf("菜单项:\n got=%v\nwant=%v", labels, want)
	}

	var opened []string
	for _, item := range items {
		if item.Action == ActionOpenSession {
			opened = append(opened, item.SessionID)
		}
	}
	if strings.Join(opened, ",") != "s1,s2,s3,s4,s5" {
		t.Fatalf("最近会话应取前 5 条：%v", opened)
	}
	if last := items[len(items)-1]; last.Action != ActionQuit || last.Label != "退出" {
		t.Fatalf("最后一项应是退出：%+v", last)
	}
	var toggle *Item
	for i := range items {
		if items[i].Action == ActionToggleCloseToTray {
			toggle = &items[i]
		}
	}
	if toggle == nil || !toggle.Checked {
		t.Fatalf("关窗偏好为真时应勾选：%+v", toggle)
	}
}

// TestMenuWindowToggle 核对第一项随窗口可见性在显示/隐藏之间切换。
func TestMenuWindowToggle(t *testing.T) {
	hidden := Menu(State{WindowVisible: false})[0]
	if hidden.Action != ActionShowWindow || hidden.Label != "显示主窗口" {
		t.Fatalf("窗口不可见时应给「显示主窗口」：%+v", hidden)
	}
	shown := Menu(State{WindowVisible: true})[0]
	if shown.Action != ActionHideWindow || shown.Label != "隐藏主窗口" {
		t.Fatalf("窗口可见时应给「隐藏主窗口」：%+v", shown)
	}
}

// TestMenuWithoutRecent 核对没有最近会话时不摆「最近会话」标题与分隔线。
func TestMenuWithoutRecent(t *testing.T) {
	items := Menu(State{WindowVisible: true})
	for _, item := range items {
		if item.Label == "最近会话" {
			t.Fatalf("没有最近会话时不该有标题项：%+v", items)
		}
		if !item.Separator && item.Action == ActionOpenSession {
			t.Fatalf("没有最近会话时不该有会话项：%+v", items)
		}
	}
}

// TestStatusOf 核对状态优先级：等待审批优先于运行中，二者都没有才是空闲。
func TestStatusOf(t *testing.T) {
	cases := []struct {
		state State
		want  Status
	}{
		{State{}, StatusIdle},
		{State{Running: true}, StatusRunning},
		{State{WaitingApproval: true}, StatusWaiting},
		{State{Running: true, WaitingApproval: true}, StatusWaiting},
	}
	for _, c := range cases {
		if got := StatusOf(c.state); got != c.want {
			t.Fatalf("StatusOf(%+v)=%v, want %v", c.state, got, c.want)
		}
	}
	if Tooltip(StatusIdle) != "DimAgent" ||
		Tooltip(StatusRunning) != "DimAgent · 运行中" ||
		Tooltip(StatusWaiting) != "DimAgent · 等待你的回应" {
		t.Fatal("悬停文字不符合三种状态")
	}
}

// TestIconVariants 核对三种状态的图标都能解码、尺寸一致，并且状态标记真的画上
// 去了（右下角像素不同）。
func TestIconVariants(t *testing.T) {
	var previous []byte
	for _, status := range []Status{StatusIdle, StatusRunning, StatusWaiting} {
		data, err := Icon(status)
		if err != nil {
			t.Fatalf("生成图标（状态 %v）：%v", status, err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("图标不是合法 PNG（状态 %v）：%v", status, err)
		}
		if b := img.Bounds(); b.Dx() != 32 || b.Dy() != 32 {
			t.Fatalf("图标尺寸：%v，想要 32×32", b)
		}
		if previous != nil && bytes.Equal(previous, data) {
			t.Fatalf("状态 %v 的图标与上一状态相同：标记没有画出来", status)
		}
		previous = data
	}
	running, _ := Icon(StatusRunning)
	img, err := png.Decode(bytes.NewReader(running))
	if err != nil {
		t.Fatalf("解码运行中图标：%v", err)
	}
	if _, _, _, a := img.At(26, 26).RGBA(); a == 0 {
		t.Fatal("运行中的圆点没有落在右下角 (26,26)")
	}
}

// TestSignatureTracksState 核对指纹：勾选与标签变化都会变，同一份菜单稳定。
func TestSignatureTracksState(t *testing.T) {
	base := Menu(State{WindowVisible: true})
	if Signature(base) != Signature(Menu(State{WindowVisible: true})) {
		t.Fatal("同一状态应得到同一份指纹")
	}
	if Signature(base) == Signature(Menu(State{WindowVisible: true, CloseToTray: true})) {
		t.Fatal("关窗偏好变化应改变指纹")
	}
	if Signature(base) == Signature(Menu(State{WindowVisible: false})) {
		t.Fatal("窗口可见性变化应改变指纹")
	}
}
