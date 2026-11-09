package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// modelFixtureState is the fixture with the model facts the caller read from the
// App Server: one current pair and a catalog with an available and an
// unavailable model.
func modelFixtureState() *State {
	s := fixture()
	s.ModelStatus = ModelStatus{Available: true}
	s.ModelCatalog = ModelCatalog{
		Current:    "anthropic/claude-3",
		HasCurrent: true,
		Choices: []ModelChoice{
			{
				ProviderID:   "anthropic",
				ProviderName: "Anthropic",
				ModelID:      "claude-3",
				DisplayName:  "Claude 3",
				Enabled:      true,
				Current:      true,
			},
			{
				ProviderID:   "local",
				ProviderName: "Local",
				ModelID:      "tiny",
				DisplayName:  "Tiny",
				Enabled:      false,
				Reason:       "missing context limits",
			},
		},
	}
	return s
}

// TestComposerModelChipShowsTheCurrentPair checks the chip's real fact: the
// current provider/model replaces the old unconfigured label, and a click opens
// the read-only catalog with each model's availability and reason.
func TestComposerModelChipShowsTheCurrentPair(t *testing.T) {
	s := modelFixtureState()
	tt := ui.NewTester(s.View, 900, 700)

	if !tt.HasText("anthropic/claude-3") {
		t.Errorf("the chip does not show the current pair, texts %q", tt.Texts())
	}
	chip, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyComposerModelChip))
	if !ok {
		t.Fatalf("the chip is not a named button, texts %q", tt.Texts())
	}
	if chip.Description != "" {
		t.Errorf("a configured model still carries a reason %q", chip.Description)
	}
	if err := tt.Click(tr(LocaleEN, keyComposerModelChip)); err != nil {
		t.Fatalf("open the model list: %v", err)
	}
	tt.Frame()
	if !s.ModelListOpen {
		t.Fatal("the chip did not open the read-only model list")
	}
	for _, want := range []string{
		tr(LocaleEN, keyModelTitle),
		"Claude 3",
		"Tiny",
		"local/tiny",
		tr(LocaleEN, keyModelCurrent),
		tr(LocaleEN, keyModelAvailable),
		tr(LocaleEN, keyModelUnavailable) + ": missing context limits",
		tr(LocaleEN, keyModelReadOnly),
	} {
		if !tt.HasText(want) {
			t.Errorf("the model list is missing %q, texts %q", want, tt.Texts())
		}
	}
	// Escape closes the list and sends nothing.
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if s.ModelListOpen {
		t.Error("Escape did not close the model list")
	}
}

// TestComposerModelChipExplainsAnAbsentModel checks the degraded state: with no
// current pair and a capability read failure, the chip states the reason and the
// list carries it instead of inventing a model.
func TestComposerModelChipExplainsAnAbsentModel(t *testing.T) {
	s := fixture()
	s.ModelStatus = ModelStatus{
		Available: false,
		Reason:    "appserver: capability unavailable: provider.management",
	}
	tt := ui.NewTester(s.View, 900, 700)

	if !tt.HasText("appserver: capability unavailable: provider.management") {
		t.Errorf("the chip does not show the capability reason, texts %q", tt.Texts())
	}
	chip, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyComposerModelChip))
	if !ok {
		t.Fatalf("the chip is not a named button, texts %q", tt.Texts())
	}
	if !strings.Contains(chip.Description, "provider.management") {
		t.Errorf("the chip description %q does not name the missing capability", chip.Description)
	}
	if err := tt.Click(tr(LocaleEN, keyComposerModelChip)); err != nil {
		t.Fatalf("open the model list: %v", err)
	}
	tt.Frame()
	if !tt.HasText("appserver: capability unavailable: provider.management") {
		t.Errorf("the model list hides the capability reason, texts %q", tt.Texts())
	}
	if !tt.HasText(tr(LocaleEN, keyModelEmpty)) {
		t.Errorf("an empty catalog is not stated, texts %q", tt.Texts())
	}
	if err := tt.Click(tr(LocaleEN, keyModelClose)); err != nil {
		t.Fatalf("close the model list: %v", err)
	}
	tt.Frame()
	if s.ModelListOpen {
		t.Error("Close did not close the model list")
	}
}

// TestComposerModelChipWIPWithoutAnAction checks the region contract: a caller
// that wires no OnToggleModelList keeps the chip visible and explained rather
// than opening a list it cannot toggle.
func TestComposerModelChipWIPWithoutAnAction(t *testing.T) {
	s := fixture()
	view := s.composerView()
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, Actions{})
	}, 900, 700)

	chip, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyComposerModelChip))
	if !ok {
		t.Fatalf("the chip is not a named button, texts %q", tt.Texts())
	}
	if !chip.Disabled {
		t.Error("the chip is available without an OnToggleModelList action")
	}
	if chip.Description == "" {
		t.Error("the unavailable chip does not explain the gap")
	}
}
