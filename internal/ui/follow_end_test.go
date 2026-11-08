package ui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestFollowEndFixture is a native, headless contract for the chat list shape:
// keyed rows begin at the newest item, remain there as streaming rows arrive,
// and a list scrolled away from its end keeps the row the user left when older
// rows load above it. It deliberately owns only throwaway view state, not
// timeline facts.
func TestFollowEndFixture(t *testing.T) {
	messages := make([]string, 40)
	for i := range messages {
		messages[i] = fmt.Sprintf("message-%02d", i)
	}
	list := ui.ListState{
		FollowEnd: true,
		Key:       func(i int) any { return messages[i] },
	}
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
	if first, _ := list.Visible(); first == 0 {
		t.Fatalf("the list built every row of 40, so it is not the virtualized list")
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

	// Scrolling up leaves the end; loading older rows above must keep the row
	// at the top, which the keyed list follows by its item rather than its index.
	tt.Scroll(160, 60, 0, -1000)
	if list.AtEnd() {
		t.Fatalf("the list still shows its end after scrolling away from it")
	}
	_, first := list.Visible()
	top := messages[first]
	older := make([]string, 5)
	for i := range older {
		older[i] = fmt.Sprintf("older-%02d", i)
	}
	messages = append(older, messages...)
	tt.Frame()
	if _, first := list.Visible(); messages[first] != top {
		t.Fatalf("top row is %q after older rows loaded, want %q", messages[first], top)
	}
	if !tt.HasText(top) {
		t.Fatalf("the held top row %q is not built, texts=%q", top, tt.Texts())
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
