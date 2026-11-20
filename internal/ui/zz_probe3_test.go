package ui

import (
	"fmt"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestProbeTabOrder3(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	for i := 0; i < 34; i++ {
		tt.Key(0, gui.KeyTab)
		names := focusedNames(tt)
		if len(names) == 0 {
			fmt.Printf("stop %d: EMPTY\n", i+1)
			continue
		}
		fmt.Printf("stop %d: %.28s\n", i+1, names[0])
	}
}
