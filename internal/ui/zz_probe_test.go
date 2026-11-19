package ui

import (
	"fmt"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func focusedNames(tt *gui.Tester) []string {
	var out []string
	for _, s := range tt.Texts() {
		if tt.Focused(s) {
			out = append(out, s)
		}
	}
	return out
}

func TestProbeFocusNames(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	seen := map[string]bool{}
	for i := 0; i < 60; i++ {
		tt.Key(0, gui.KeyTab)
		names := focusedNames(tt)
		key := fmt.Sprint(names)
		if i > 0 && key == "" {
			t.Logf("stop %d: NO NAME", i)
			continue
		}
		if seen[key] && i > 0 {
			t.Logf("cycle at %d: %v", i, names)
			break
		}
		seen[key] = true
		t.Logf("stop %d: %v", i, names)
	}
}
