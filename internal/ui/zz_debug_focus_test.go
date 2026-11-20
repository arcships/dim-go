package ui

import (
	"fmt"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestDebugApprovalFocus(t *testing.T) {
	app, tt := newTester(t, StateWaiting)
	names := []string{"审批卡", approvalCopyReject, approvalCopyAllowOnce, approvalCopyAlwaysCommand, approvalInputLabel, "DimAgent 想要：执行"}
	report := func(step string) {
		out := ""
		for _, n := range names {
			if tt.Focused(n) {
				out += fmt.Sprintf(" [label:%q]", n)
			}
			if focusOn(tt, n) {
				out += fmt.Sprintf(" [self:%q]", n)
			}
		}
		info, ok := tt.Access(approvalCopyReject)
		t.Logf("%s ring=%d focus=%s access=%+v ok=%v", step, len(app.approvalRing), out, info, ok)
	}
	report("initial")
	for i := 0; i < 5; i++ {
		tt.Key(0, gui.KeyTab)
		report(fmt.Sprintf("tab%d", i+1))
	}
	tt.Key(gui.Shift, gui.KeyTab)
	report("shiftTab")
}
