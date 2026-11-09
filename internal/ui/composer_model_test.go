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
	s.ModelStatus = ModelStatus{Available: true, Selectable: true}
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

// TestComposerModelListSelectsOnce checks the wired list: an available row
// sends exactly one OnSelectModel pair and closes the dialog before the request,
// an unavailable row sends nothing, and Escape still closes without a request.
func TestComposerModelListSelectsOnce(t *testing.T) {
	s := modelFixtureState()
	s.ModelCatalog.Choices = append(s.ModelCatalog.Choices, ModelChoice{
		ProviderID:   "local",
		ProviderName: "Local",
		ModelID:      "small",
		DisplayName:  "Small",
		Enabled:      true,
	})
	var selections [][2]string
	s.OnSelectModel = func(providerID, modelID string) {
		selections = append(selections, [2]string{providerID, modelID})
	}
	tt := ui.NewTester(s.View, 900, 700)

	if err := tt.Click(tr(LocaleEN, keyComposerModelChip)); err != nil {
		t.Fatalf("open the model list: %v", err)
	}
	tt.Frame()

	// An unavailable row is a disabled control: its click sends nothing.
	if err := tt.Click("Tiny"); err != nil {
		t.Fatalf("click the unavailable row: %v", err)
	}
	tt.Frame()
	if len(selections) != 0 {
		t.Fatalf("an unavailable model sent %#v", selections)
	}
	if !s.ModelListOpen {
		t.Fatal("an unavailable row closed the list")
	}

	if err := tt.Click("Small"); err != nil {
		t.Fatalf("select a model: %v", err)
	}
	tt.Frame()
	if len(selections) != 1 || selections[0] != [2]string{"local", "small"} {
		t.Fatalf("selections = %#v, want one local/small request", selections)
	}
	if s.ModelListOpen {
		t.Fatal("the selected row did not close the list")
	}
	// A second click after the dialog closed cannot send the same selection.
	if err := tt.Click("Small"); err == nil {
		tt.Frame()
	}
	if len(selections) != 1 {
		t.Fatalf("selections = %#v after a second click, want one", selections)
	}
}

// TestComposerModelListEscapeSendsNothing checks the keyboard path: Escape
// closes the list and never calls the selection action.
func TestComposerModelListEscapeSendsNothing(t *testing.T) {
	s := modelFixtureState()
	calls := 0
	s.OnSelectModel = func(string, string) { calls++ }
	tt := ui.NewTester(s.View, 900, 700)

	if err := tt.Click(tr(LocaleEN, keyComposerModelChip)); err != nil {
		t.Fatalf("open the model list: %v", err)
	}
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if s.ModelListOpen {
		t.Fatal("Escape did not close the list")
	}
	if calls != 0 {
		t.Fatalf("Escape sent %d selections, want 0", calls)
	}
}

// TestComposerModelRowsStayDisabledWithoutASelectionAction checks that the rows
// are not newly clickable when the caller wires no selection: they stay visible,
// disabled and paired with the read-only note.
func TestComposerModelRowsStayDisabledWithoutASelectionAction(t *testing.T) {
	s := modelFixtureState()
	s.ModelStatus = ModelStatus{Available: true, Selectable: true}
	tt := ui.NewTester(s.View, 900, 700)
	if err := tt.Click(tr(LocaleEN, keyComposerModelChip)); err != nil {
		t.Fatalf("open the model list: %v", err)
	}
	tt.Frame()
	row, ok := tt.AccessByRole(ui.RoleButton, "Claude 3")
	if !ok {
		t.Fatalf("the current model row is not a named button, texts %q", tt.Texts())
	}
	if !row.Disabled {
		t.Fatalf("model row %#v is enabled without OnSelectModel", row)
	}
	if !tt.HasText(tr(LocaleEN, keyModelReadOnly)) {
		t.Errorf("the list does not state the read-only gap, texts %q", tt.Texts())
	}
}
