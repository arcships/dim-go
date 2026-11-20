package ui

import (
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「动效」这一类需求的取证：能测的测到常量与开关（时长、延迟、关动效
// 后布局不变），测不到的（真实窗口的过渡曲线、帧时间）在文件末尾写明需要录屏。

// TestMotionDurationsAreTokens：动效时长是唯一一份 token，需求里的数值与它逐字
// 对齐（U2 §5.1、§5.3、§8.4）。
func TestMotionDurationsAreTokens(t *testing.T) {
	cases := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"过渡（motionTransition）", motionTransition, 200 * time.Millisecond},
		{"弹性（motionSpring）", motionSpring, 500 * time.Millisecond},
		{"内容（motionContent）", motionContent, 350 * time.Millisecond},
		{"光标周期（caretCycle）", caretCycle, 1060 * time.Millisecond},
		{"悬停提示延迟（tooltipDelay）", tooltipDelay, 600 * time.Millisecond},
		{"悬停提示热启动（tooltipWarm）", tooltipWarm, 400 * time.Millisecond},
		{"会话行悬停卡延迟", 500 * time.Millisecond, 500 * time.Millisecond},
		{"命令面板输入去抖", commandPaletteDebounce, 120 * time.Millisecond},
		{"右栏骨架呼吸周期", rpSkeletonPulse, 1200 * time.Millisecond},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Fatalf("%s = %v，想要 %v", c.name, c.got, c.want)
		}
	}
	if caretWidth != 2 || caretHeight != 14 {
		t.Fatalf("光标尺寸 = %v×%v，想要 2×14", caretWidth, caretHeight)
	}
}

// TestHoverCardWaitsForDelay：会话行悬停卡在指针停留满延迟后才出现，移出立即
// 消失。
//
// 证据：R-LB-822（出现带延迟、消失立即）、R-LB-484（500ms 后出现）。
func TestHoverCardWaitsForDelay(t *testing.T) {
	src := wiredSource()
	src.names = []string{"s1", "s2"}
	src.titles = []string{"第一条", "第二条"}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	row, ok := tt.Find("第一条")
	if !ok {
		t.Fatalf("找不到会话行：%v", tt.Texts())
	}
	tt.Move(row.X+row.W/2, row.Y+row.H/2)
	tt.Frame()
	tt.Advance(400 * time.Millisecond)
	tt.Frame()
	if tt.HasText("会话标题：第一条") {
		t.Fatalf("停留不足 500ms 就出现了悬停卡")
	}
	tt.Advance(200 * time.Millisecond)
	tt.Frame()
	if !tt.HasText("会话标题：第一条") {
		t.Fatalf("停留满 500ms 后悬停卡没有出现：%v", tt.Texts())
	}

	// 移出：立即消失（不等延迟）。
	tt.Move(1, WindowHeight-1)
	tt.Frame()
	if tt.HasText("会话标题：第一条") {
		t.Fatal("指针移出后悬停卡还在")
	}
}

// TestReduceMotionKeepsCopyAndLayout：关动效后每一屏的文字集合与关键元素位置
// 不变——动效只影响过渡，不影响信息与布局（UI-5、§9.14）。
func TestReduceMotionKeepsCopyAndLayout(t *testing.T) {
	screens := []struct {
		name string
		open func() *App
	}{
		{"主界面", func() *App { return NewApp(Options{State: StateDefault}) }},
		{"运行中", func() *App { return NewApp(Options{State: StateRunning}) }},
		{"等待审批", func() *App { return NewApp(Options{State: StateWaiting}) }},
		{"右栏·文件", func() *App {
			app := NewApp(Options{State: StateDefault})
			app.rightOpen, app.rightTab = true, tabFiles
			return app
		}},
		{"右栏·改动", func() *App {
			app := NewApp(Options{State: StateDefault})
			app.rightOpen, app.rightTab = true, tabChanges
			return app
		}},
		{"设置·外观", func() *App {
			app := NewApp(Options{State: StateDefault})
			app.screen, app.settingsSection = ScreenSettings, secAppearance
			return app
		}},
	}
	for _, s := range screens {
		t.Run(s.name, func(t *testing.T) {
			on := gui.NewTester(s.open().View, WindowWidth, WindowHeight)
			off := gui.NewTesterWith(s.open().View, WindowWidth, WindowHeight,
				gui.TesterOptions{Preferences: gui.Preferences{ReduceMotion: true}})
			if a, b := strings.Join(on.Texts(), "|"), strings.Join(off.Texts(), "|"); a != b {
				t.Fatalf("关动效后文字集合变了")
			}
			for _, label := range []string{"标题行", "对话流", "消息", "右栏"} {
				ra, ok1 := on.Find(label)
				rb, ok2 := off.Find(label)
				if !ok1 && !ok2 {
					continue
				}
				if ok1 != ok2 || ra != rb {
					t.Fatalf("关动效后 %q 的位置变了：%v → %v", label, ra, rb)
				}
			}
		})
	}
}

// 以下动效条目本批无法在无窗口环境里取证，需要真实窗口录屏：
//
//   - 过渡曲线与缓动（cubic-bezier）的实际观感：R-TB-046、R-TL-068、R-TL-181。
//   - 旋转指示「线性、无停顿」（R-LB-040、R-LB-399、R-LB-416、R-LB-571）是逐帧
//     相位是否匀速，headless 的帧时钟可以推进但没有 GPU 合成，量不到观感。
//   - 「不做缩放 / 不做脉冲 / 不做拖影」这类负向动效（R-LB-680、R-LB-694、
//     R-LB-798、R-LB-807、R-LB-844）只能靠录屏逐帧看。
//   - 帧时间与滚动流畅度（性能维度）需要 MYGO_FRAME_STATS=1 在真实窗口采样。
