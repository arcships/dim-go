package ui

import (
	"fmt"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「屏上只说用户能读的话」的批量验收：遍历主要屏，断言开发口径与内部
// 术语一个字都不出现。
//
// 这些词只该留在代码注释与决策记录里，供实现与排查引用：
//
//   - 等事实 / 进快照 / 事实未到：投影字段还没覆盖的说明，用户读不懂也不需要；
//   - 缺宿主 / 缺协议 / 缺少事实：缺口的技术分类，界面只说「当前版本不支持」；
//   - 本版不做 / 读数未到：开发排期与读数状态，界面写用户看到的结果。
//
// 元素缺失时正确的做法是整块不画或给用户能理解的空态；确实要解释为什么不能做，
// 就写用户视角的原因（「当前版本不支持外部打开」而不是「缺宿主能力」）。
var devSpeakWords = []string{
	"等事实", "进快照", "快照", "缺少事实", "事实未",
	"缺宿主", "缺协议",
	"本版不做", "本版不", "本版没有", "本版只",
	"读数未到", "读数由",
	"服务端没有", "服务端未",
}

// TestScreensShowNoDevSpeak：主要屏（主界面各状态、连上 Source 的主界面、右栏各
// 标签页、设置全部小节、走查态、两个浮层）都不出现开发口径。
func TestScreensShowNoDevSpeak(t *testing.T) {
	resetSettingsScreenState()
	type screen struct {
		name string
		open func(t *testing.T) (*App, *gui.Tester)
	}
	var screens []screen

	// 主界面五态：样例与连线各一遍（未接入口的原因只在连线态出现）。
	screenStates := []State{StateDefault, StateRunning, StateWaiting, StateFailed, StateEmpty}
	for _, state := range screenStates {
		screens = append(screens, screen{
			fmt.Sprintf("主界面·状态 %d", state),
			a11yMainScreen(state),
		})
		screens = append(screens, screen{
			fmt.Sprintf("主界面·连线·状态 %d", state),
			func(t *testing.T) (*App, *gui.Tester) {
				t.Helper()
				app := NewApp(Options{State: state, Source: wiredSource()})
				return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
			},
		})
	}

	// 右栏各标签页：样例与连线各一遍。
	for _, tab := range []int{tabFiles, tabChanges, tabTasks, tabGoal, tabTerminal, tabSideChat, tabSubagent} {
		screens = append(screens, screen{fmt.Sprintf("右栏·标签 %d", tab), a11yRightPanel(tab)})
		screens = append(screens, screen{
			fmt.Sprintf("右栏·连线·标签 %d", tab),
			func(t *testing.T) (*App, *gui.Tester) {
				t.Helper()
				app := NewApp(Options{State: StateDefault, Source: wiredSource()})
				app.rightOpen, app.rightTab = true, tab
				return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
			},
		})
	}

	// 设置：全部小节，样例与连线各一遍。
	for _, def := range settingsSectionDefs {
		screens = append(screens, screen{"设置·" + def.slug, a11ySettings(def.id)})
		screens = append(screens, screen{
			"设置·连线·" + def.slug,
			func(t *testing.T) (*App, *gui.Tester) {
				t.Helper()
				app := NewApp(Options{State: StateDefault, Source: wiredSource()})
				app.screen, app.settingsSection = ScreenSettings, def.id
				return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
			},
		})
	}

	// 走查态与浮层。
	for _, v := range []Variant{VariantPalette, VariantSearch, VariantSessionMenu, VariantSessionRename,
		VariantSessionDelete, VariantBranch, VariantBranchEnd, VariantGoal} {
		screens = append(screens, screen{fmt.Sprintf("走查态 %d", v), a11yVariant(v)})
	}
	screens = append(screens, screen{"浮层·会话内搜索", a11ySessionSearch()})

	for _, s := range screens {
		t.Run(s.name, func(t *testing.T) {
			_, tt := s.open(t)
			tt.Frame()
			for _, word := range devSpeakWords {
				if tt.HasText(word) {
					t.Fatalf("%s：屏上出现了开发口径 %q\n屏上文字：\n%s",
						s.name, word, negScreenText(tt))
				}
			}
		})
	}
}

// TestAttachmentAreaDrawsNothingWithoutFacts：用户消息的附件与引用字段还没进
// RowSnapshot（DEC-07），所以真实消息没有这些内容时这一区整块不画——不摆占位，
// 也不写一句「附件与引用还没有进快照」给用户看。
func TestAttachmentAreaDrawsNothingWithoutFacts(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.source = &fakeSource{
		conn:    Connection{Phase: "connected"},
		names:   []string{"s1"},
		session: liveFactSession(),
		hasSess: true,
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	// 输入区的「附件」按钮还在（那是要有的），这里查的是消息气泡里那一区。
	for _, text := range []string{"附件与引用", "等事实", "占位"} {
		if tt.HasText(text) {
			t.Fatalf("没有附件事实却画了 %q：\n%s", text, negScreenText(tt))
		}
	}
}
