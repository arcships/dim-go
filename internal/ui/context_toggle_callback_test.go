package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestContextPanelToggleWithCallbackExecutesOnce(t *testing.T) {
	calls := 0
	tt := ui.NewTester(func(c *ui.Context) {
		contextPanelToggle(c, newTokens(c.Theme()), HeaderView{}, Actions{
			OnToggleContextPanel: func() { calls++ },
		})
	}, 200, 100)
	label := tr(LocaleEN, keyContextPanelToggle)
	info, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok || info.Disabled {
		t.Fatal("wired context-panel entry is missing or disabled")
	}
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("toggle calls = %d, want 1", calls)
	}
}

func TestContextPanelToggleWithoutCallbackIsDisabled(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		contextPanelToggle(c, newTokens(c.Theme()), HeaderView{}, Actions{})
	}, 200, 100)
	info, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyContextPanelToggle))
	if !ok {
		t.Fatal("missing context-panel entry")
	}
	if !info.Disabled {
		t.Fatal("context-panel entry is enabled without a callback")
	}
	if !strings.Contains(info.Description, "Unavailable: the window has no context-panel toggle callback.") {
		t.Fatalf("missing callback explanation: %q", info.Description)
	}
}
