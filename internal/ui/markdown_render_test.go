package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// markdownTester builds a window whose only content is src rendered as the
// content layer, at width by height DIPs, and returns the headless tester of
// its first frame.
func markdownTester(t *testing.T, width, height int, src string, opts MarkdownOptions) *ui.Tester {
	t.Helper()
	return ui.NewTester(func(c *ui.Context) {
		k := newTokens(c.Theme())
		ui.Column(c).Children(func() {
			renderMarkdownContent(c, k, src, opts)
		})
	}, width, height)
}

// markdownKind asserts that a text is shown, and that a piece of Markdown
// syntax is not: the renderer parses the source into text, it does not print
// it.
func markdownKind(t *testing.T, tt *ui.Tester, shown, syntax string) {
	t.Helper()
	if !tt.HasText(shown) {
		t.Errorf("the content is missing %q, texts %q", shown, tt.Texts())
	}
	if syntax != "" && tt.HasText(syntax) {
		t.Errorf("the Markdown syntax %q shows as text, texts %q", syntax, tt.Texts())
	}
}

// TestMarkdownContentThemeUsesShellTokens checks that the content layer is
// themed with the shell's semantic colors rather than mygo's own palette: the
// faces and colors the renderer names in its --markdown-* tokens map onto the
// tokens the rest of the window uses.
func TestMarkdownContentThemeUsesShellTokens(t *testing.T) {
	k := newTokens(ui.LightTheme())
	th := contentTheme(k)

	cases := []struct {
		name string
		got  ui.Color
		want ui.Color
	}{
		{"text", th.Text, k.foreground},
		{"muted", th.Muted, k.mutedForeground},
		{"link", th.Link, k.primary},
		{"strong", th.Strong, k.foreground},
		{"inline code text", th.Code, k.foreground},
		{"inline code face", th.CodeBackground, k.muted},
		{"quote bar", th.QuoteBar, k.border},
		{"quote text", th.QuoteText, k.mutedForeground},
		{"rule", th.Rule, k.border},
		{"code block face", th.CodeBlockBackground, k.card},
		{"code block border", th.CodeBlockBorder, k.border},
		{"table border", th.TableBorder, k.border},
		{"table header", th.TableHeader, k.card.Mix(k.foreground, tableHeaderRise)},
		{"task list check", th.Checkbox, k.primary},
	}
	for _, tt := range cases {
		if tt.got != tt.want {
			t.Errorf("the %s color is %v, want the shell token %v", tt.name, tt.got, tt.want)
		}
	}
	if th.FontSize != k.bodySize {
		t.Errorf("the body size is %v, want the shell's %v", th.FontSize, k.bodySize)
	}
	if th.CodeSize != inlineCodeSize {
		t.Errorf("the inline code size is %vem, want %vem", th.CodeSize, inlineCodeSize)
	}
	if th.CodeBlockRadius != k.cardRadius {
		t.Errorf("the code block radius is %v, want the shell card's %v", th.CodeBlockRadius, k.cardRadius)
	}

	// The layer must follow a dark window too, and not keep the light colors:
	// the tokens it derives from change with the theme.
	dark := contentTheme(newTokens(ui.DarkTheme()))
	if dark.Text == th.Text || dark.CodeBlockBackground == th.CodeBlockBackground {
		t.Errorf("the dark theme keeps the light text %v and code face %v", dark.Text, dark.CodeBlockBackground)
	}
}

// TestMarkdownCodeBlockHeadNamesItsLanguage checks the head of a fenced block:
// its language as a label and the copy button beside it, and no language
// invented for a fence that names none.
func TestMarkdownCodeBlockHeadNamesItsLanguage(t *testing.T) {
	const src = "```go\nfmt.Println(\"hi\")\n```\n\n```\nplain\n```\n"
	tt := markdownTester(t, 640, 480, src, MarkdownOptions{})

	if !tt.HasText("go") {
		t.Errorf("the Go block's head is missing its language, texts %q", tt.Texts())
	}
	// "text" is found exactly: the second block's body also says "plain",
	// which must not stand in for the label.
	if _, ok := tt.Find(noCodeLanguage); !ok {
		t.Errorf("a fence with no language is not labeled %q, texts %q", noCodeLanguage, tt.Texts())
	}
	for _, lang := range []string{"go", noCodeLanguage} {
		if _, ok := tt.AccessByRole(ui.RoleGroup, codeGroupLabel(lang)); !ok {
			t.Errorf("the %q block is not named %q to assistive technology", lang, codeGroupLabel(lang))
		}
	}
	// No language was guessed: the second block is not called after the word
	// in its body or after a made-up language.
	for _, guessed := range []string{"plain", "text/plain", "txt"} {
		if _, ok := tt.AccessByRole(ui.RoleGroup, codeGroupLabel(guessed)); ok {
			t.Errorf("the unlabeled fence claims the language %q", guessed)
		}
	}

	// A copy button belongs to each block: one per head.
	if n := countText(tt.Texts(), "Copy"); n != 2 {
		t.Errorf("%d copy buttons for two code blocks, texts %q", n, tt.Texts())
	}

	// The code itself is still shown, highlighted as text without its fence.
	markdownKind(t, tt, `fmt.Println("hi")`, "```")
	if !tt.HasText("plain") {
		t.Errorf("the unlabeled block's code is missing, texts %q", tt.Texts())
	}
}

// countText counts the texts equal to s.
func countText(texts []string, s string) int {
	n := 0
	for _, text := range texts {
		if text == s {
			n++
		}
	}
	return n
}

// TestMarkdownCodeCopyButtonCopiesTheBlock checks that pressing a code block's
// copy button puts exactly the block's source on the clipboard, with the "did
// it" feedback mygo's button shows and its two-second life, and that a second
// block copies its own code rather than another view's.
func TestMarkdownCodeCopyButtonCopiesTheBlock(t *testing.T) {
	tt := markdownTester(t, 640, 480, "```go\nfmt.Println(\"hi\")\n```\n", MarkdownOptions{})

	if got := tt.Clipboard(); got != "" {
		t.Fatalf("the clipboard starts with %q", got)
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatalf("the copy button: %v", err)
	}
	if got, want := tt.Clipboard(), "fmt.Println(\"hi\")\n"; got != want {
		t.Errorf("copying the block put %q on the clipboard, want %q", got, want)
	}
	if !tt.HasText("Copied") {
		t.Errorf("the copy button does not say it copied, texts %q", tt.Texts())
	}

	// The feedback is the button's own for two seconds, not forever.
	tt.Advance(3 * time.Second)
	tt.Frame()
	if tt.HasText("Copied") {
		t.Errorf("the copy button still says Copied after its feedback, texts %q", tt.Texts())
	}
	if !tt.HasText("Copy") {
		t.Errorf("the copy button is gone after its feedback, texts %q", tt.Texts())
	}

	// A fence with no language copies its code too, and its head says "text".
	other := markdownTester(t, 640, 480, "```\nnpm install\n```\n", MarkdownOptions{})
	if _, ok := other.Find(noCodeLanguage); !ok {
		t.Errorf("the unlabeled block's head is not %q, texts %q", noCodeLanguage, other.Texts())
	}
	if err := other.Click("Copy"); err != nil {
		t.Fatalf("the unlabeled block's copy button: %v", err)
	}
	if got, want := other.Clipboard(), "npm install\n"; got != want {
		t.Errorf("copying the unlabeled block put %q on the clipboard, want %q", got, want)
	}
}

// TestMarkdownCodeCopyCannotSeeARefusedWrite records a gap rather than a
// feature: mygo's copy button copies with Context.WriteClipboard, so a refused
// write leaves the clipboard empty while the button still reports that it
// copied. A refused write is not a WIP capability — the clipboard exists — but
// the content layer cannot tell the user it failed through this API.
func TestMarkdownCodeCopyCannotSeeARefusedWrite(t *testing.T) {
	tt := markdownTester(t, 640, 480, "```go\nfmt.Println(\"hi\")\n```\n", MarkdownOptions{})
	tt.FailClipboard(errors.New("the clipboard is held by another app"))
	if err := tt.Click("Copy"); err != nil {
		t.Fatalf("the copy button: %v", err)
	}
	if got := tt.Clipboard(); got != "" {
		t.Errorf("a refused write put %q on the clipboard", got)
	}
	// The gap: the button cannot tell, so it says it copied. This assertion
	// is the documented gap; the clipboard above shows nothing was copied.
	if !tt.HasText("Copied") {
		t.Errorf("mygo's copy button changed to report failures; drop the gap note in markdown_render.go")
	}
}

// TestMarkdownContentRendersInlineCodeLinkTableAndTasks checks the content
// layer's other blocks end to end: inline code and links as text, a table with
// its header and cells exposed as a table, a task list's checked and unchecked
// boxes, and a thematic break between them.
func TestMarkdownContentRendersInlineCodeLinkTableAndTasks(t *testing.T) {
	const src = "Some `inline code` and a [docs](https://example.com) link.\n\n" +
		"| Name | Value |\n| --- | --- |\n| alpha | 1 |\n| beta | 2 |\n\n" +
		"- [x] done\n- [ ] open\n\n---\n\nAfter the rule.\n"
	var opened string
	tt := markdownTester(t, 720, 640, src, MarkdownOptions{
		OnLink: func(url string) { opened = url },
	})

	markdownKind(t, tt, "Some inline code and a docs link.", "`inline code`")
	markdownKind(t, tt, "After the rule.", "---")
	for _, cell := range []string{"Name", "Value", "alpha", "1", "beta", "2"} {
		if !tt.HasText(cell) {
			t.Errorf("the table is missing the cell %q, texts %q", cell, tt.Texts())
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleTable, ""); !ok {
		t.Errorf("the table is not exposed as a table, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleColumnHeader, "Name"); !ok {
		t.Errorf("the table header cell is not exposed as a column header, texts %q", tt.Texts())
	}
	// mygo names a cell's box, not its text, so the cells are found by role;
	// their text is the text tree's, checked above.
	if _, ok := tt.AccessByRole(ui.RoleCell, ""); !ok {
		t.Errorf("a table cell is not exposed as a cell, texts %q", tt.Texts())
	}

	checked, ok := tt.AccessByRole(ui.RoleCheckBox, "")
	if !ok {
		t.Fatalf("the task list has no check box, texts %q", tt.Texts())
	}
	if !checked.Checked {
		t.Errorf("the checked task's box is unchecked")
	}
	unchecked := markdownTester(t, 720, 320, "- [ ] open\n", MarkdownOptions{})
	box, ok := unchecked.AccessByRole(ui.RoleCheckBox, "")
	if !ok {
		t.Fatalf("the task list has no check box, texts %q", unchecked.Texts())
	}
	if box.Checked {
		t.Errorf("the unchecked task's box is checked")
	}
	for _, task := range []string{"done", "open"} {
		if !tt.HasText(task) {
			t.Errorf("the task %q is missing, texts %q", task, tt.Texts())
		}
	}

	// The link's color is the shell's primary (contentTheme), and pressing it
	// hands its destination to the region: it is not a dead label.
	if err := tt.Click("docs"); err != nil {
		t.Fatalf("the link: %v", err)
	}
	if opened != "https://example.com" {
		t.Errorf("the link opened %q, want https://example.com", opened)
	}
}

// TestMarkdownCodeBlockScrollsLongLines checks that code wider than the view
// scrolls sideways instead of being cut off, and that with WrapCode it wraps
// instead and the wheel moves nothing.
func TestMarkdownCodeBlockScrollsLongLines(t *testing.T) {
	line := `const veryLongLine = "` + strings.Repeat("x", 200) + `"`
	src := "```ts\n" + line + "\n```\n"

	tt := markdownTester(t, 360, 240, src, MarkdownOptions{})
	before, ok := tt.Find(line)
	if !ok {
		t.Fatalf("the code line is missing, texts %q", tt.Texts())
	}
	if before.W >= 360 {
		t.Fatalf("the code line shows %v wide in a 360 DIP window: it is not clipped to its block", before.W)
	}
	// A positive dx moves the content forward in a left-to-right window:
	// the code scrolls toward its end, which is off the left edge.
	tt.Scroll(before.X+before.W/2, before.Y+before.H/2, 100, 0)
	after, ok := tt.Find(line)
	if !ok {
		t.Fatalf("the code line is missing after scrolling, texts %q", tt.Texts())
	}
	if after.X >= before.X {
		t.Errorf("scrolling the block sideways moved the code from %v to %v, want it further left", before.X, after.X)
	}

	// Wrapped, the same line takes more than one line of room, and sideways
	// scrolling has nothing to move.
	wrapped := markdownTester(t, 360, 320, src, MarkdownOptions{WrapCode: true})
	first, ok := wrapped.Find(line)
	if !ok {
		t.Fatalf("the wrapped code line is missing, texts %q", wrapped.Texts())
	}
	oneLine := float32(14 * 1.4)
	if first.H <= oneLine {
		t.Errorf("the wrapped code line is %v high, no taller than one line of %v", first.H, oneLine)
	}
	wrapped.Scroll(first.X+first.W/2, first.Y+first.H/2, 100, 0)
	moved, ok := wrapped.Find(line)
	if !ok {
		t.Fatalf("the wrapped code line is missing after scrolling, texts %q", wrapped.Texts())
	}
	if moved.X != first.X {
		t.Errorf("scrolling wrapped code moved it from %v to %v, want it in place", first.X, moved.X)
	}
}
