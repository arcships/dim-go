package ui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestFollowEndFixture is a native, headless contract for the chat list shape:
// keyed rows begin at the newest item and remain there as streaming rows arrive.
// It deliberately owns only throwaway view state, not timeline facts.
func TestFollowEndFixture(t *testing.T) {
	messages := make([]string, 40)
	for i := range messages {
		messages[i] = fmt.Sprintf("message-%02d", i)
	}
	list := ui.ListState{FollowEnd: true}
	view := func(c *ui.Context) {
		ui.List(c, &list, len(messages), func(i int) {
			ui.Text(c, messages[i])
		}).Grow(1)
	}

	tt := ui.NewTester(view, 320, 120)
	tt.Frame()
	if !list.AtEnd() {
		first, last := list.Visible()
		t.Fatalf("initial list is not at end: visible=%d..%d", first, last)
	}
	if !tt.HasText("message-39") {
		t.Fatalf("newest row is not visible: texts=%q", tt.Texts())
	}

	messages = append(messages, "message-40")
	tt.Frame()
	if !list.AtEnd() {
		first, last := list.Visible()
		t.Fatalf("list stopped following end after append: visible=%d..%d", first, last)
	}
	if !tt.HasText("message-40") {
		t.Fatalf("appended row is not visible: texts=%q", tt.Texts())
	}
}

func TestAppFrameAccessibilityFixture(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)
	if info, ok := tt.AccessByRole(ui.RoleTextField, "Composer"); !ok || info.Value != "" {
		t.Fatalf("composer accessibility: %+v, found=%v", info, ok)
	}
	if _, ok := tt.AccessByRole(ui.RoleTreeItem, "First session"); !ok {
		t.Fatal("session sidebar item is not exposed to accessibility")
	}
}
