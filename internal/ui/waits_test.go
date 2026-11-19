package ui

import (
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是等后台结果的一组测试工具：唯一的目的是把"异步"从断言里拿掉。
//
// 写入意图（App.runIntent）在后台协程里跑，结果交回渲染线程，在下一次渲染时才落到
// 视图上（applyIntentResults）。所以"点一下 + 画一帧就断言"必然有一个窗口：协程还没
// 跑完时，视图仍是执行中的样子（按钮写着「删除中...」、弹窗还没关）。这个窗口在快
// 机器上通常为零，在忙的机器上就变成失败——同一命令连跑三次失败数不同，就是它。
//
// 正确做法不是加 sleep（把窗口缩成"通常够用"），而是一直画帧直到**要断言的观察值**
// 成立：条件成立即返回，条件永不成立才超时失败。超时值只用来把"卡死"报成失败，不
// 参与等待语义，宽一点不影响结论。

// waitFrameTimeout 是"一直不成立"的判定上限。
const waitFrameTimeout = 3 * time.Second

// waitFrame 反复渲染，直到 cond 成立；what 是要等的事（失败时写在报错里）。
func waitFrame(t *testing.T, tt *gui.Tester, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(waitFrameTimeout)
	for {
		tt.Frame()
		if cond() {
			return
		}
		if !time.Now().Before(deadline) {
			t.Fatalf("等不到「%s」（%v 内条件一直不成立），屏上：\n%s",
				what, waitFrameTimeout, p0WindowText(tt))
		}
		time.Sleep(time.Millisecond)
	}
}
