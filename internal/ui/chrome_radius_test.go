package ui

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// chromeRadiusProbeView draws one component from the shell's own view code on a
// single background. Two frames of the same probe differ only where the
// component's corners are drawn, so comparing them checks that the component
// reads its radius from the frame's tokens rather than from a package
// constant.
func chromeRadiusProbeView(k tokens, component func(c *ui.Context, k tokens)) func(c *ui.Context) {
	return func(c *ui.Context) {
		ui.Column(c).Fill().Padding(k.panelPad).Background(k.background).Children(func() {
			component(c, k)
		})
	}
}

// TestChromeRadiusReachesTheViews renders a button, a card and a user bubble
// under two radius ladders: the default chrome's, and the square ladder a
// Windows 98 chrome asks for. The faces, type and spacing come from the same
// tokens in both frames, so a differing frame means the chrome's corner
// reached that component. Before the views read k.radius*, the package's
// fixed ladder drew all three identically.
func TestChromeRadiusReachesTheViews(t *testing.T) {
	rounded := newTokensFor(ui.LightTheme(), SettingsView{Chrome: ChromeDefault})
	win98 := newTokensFor(ui.LightTheme(), SettingsView{Chrome: ChromeWindows98})
	square := rounded
	square.radiusSM, square.radiusMD = win98.radiusSM, win98.radiusMD
	square.radiusLG, square.radiusXL = win98.radiusLG, win98.radiusXL
	square.cardRadius, square.frameRadius = win98.cardRadius, win98.frameRadius
	square.timelineBubbleR = win98.timelineBubbleR
	if square.radiusSM == rounded.radiusSM || square.cardRadius == rounded.cardRadius ||
		square.timelineBubbleR == rounded.timelineBubbleR {
		t.Fatalf("Windows 98 and the default chrome share a corner: %.2f/%.2f/%.2f",
			square.radiusSM, square.cardRadius, square.timelineBubbleR)
	}

	components := []struct {
		name  string
		draw  func(c *ui.Context, k tokens)
		label string
	}{
		{
			name: "button",
			draw: func(c *ui.Context, k tokens) {
				modelChoiceRow(c, k, ComposerView{}, Actions{}, ModelChoice{
					ProviderID: "anthropic", ModelID: "claude", DisplayName: "Claude", Enabled: true,
				})
			},
			label: "Claude",
		},
		{
			name: "card",
			draw: func(c *ui.Context, k tokens) {
				capabilityCard(c, k, LocaleEN, wipCapabilityOf(keyCapabilityCreate, "session.create", keyCapabilityCreateGap))
			},
			label: "Session creation",
		},
		{
			name: "bubble",
			draw: func(c *ui.Context, k tokens) {
				userRow(c, k, TimelineView{}, Actions{}, Message{ID: "m1", Author: AuthorUser, Content: "hello"})
			},
			label: "hello",
		},
	}

	for _, tc := range components {
		t.Run(tc.name, func(t *testing.T) {
			roundedTT := ui.NewTester(chromeRadiusProbeView(rounded, tc.draw), 420, 180)
			squareTT := ui.NewTester(chromeRadiusProbeView(square, tc.draw), 420, 180)
			if !roundedTT.HasText(tc.label) {
				t.Fatalf("the %s probe does not draw %q, texts %q", tc.name, tc.label, roundedTT.Texts())
			}
			roundedImg, squareImg := roundedTT.Image(), squareTT.Image()
			if roundedImg == nil || squareImg == nil {
				t.Fatalf("the %s probe drew no frame", tc.name)
			}
			if bytes.Equal(roundedImg.Pix, squareImg.Pix) {
				t.Errorf("the %s's corners did not change with the chrome's radius", tc.name)
			}
		})
	}
}

// TestViewsReadRadiiFromTokens guards the same rule for every view file: no
// view may draw a corner from the package's fixed ladder again. tokens.go keeps
// the fallback ladder and derives the tokens from it, so it is the one file
// allowed to name the constants.
func TestViewsReadRadiiFromTokens(t *testing.T) {
	banned := map[string]bool{
		"radiusSM": true, "radiusMD": true, "radiusLG": true, "radiusXL": true,
		"mainFrameRadius": true, "timelineBubbleR": true,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list the view files: %v", err)
	}
	checked := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") || name == "tokens.go" {
			continue
		}
		checked++
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		// A selector's field name is a token read (k.radiusSM), not the
		// package constant; the view rule is about bare identifiers.
		fields := map[token.Pos]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				fields[sel.Sel.Pos()] = true
			}
			return true
		})
		ast.Inspect(file, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok || !banned[id.Name] || fields[id.Pos()] {
				return true
			}
			t.Errorf("%s reads %s directly; draw the corner from the frame's tokens", name, id.Name)
			return true
		})
	}
	if checked == 0 {
		t.Fatal("no view files were checked")
	}
}
