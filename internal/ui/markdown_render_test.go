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

// TestMarkdownFilePathReadsAsAFileChip checks the reading that makes a word a
// file chip: a word of path bytes whose last part names a known extension,
// with an optional :line or :line:column on the end. The reading is
// conservative on purpose, since a wrong chip names a file the author did not,
// so prose that carries a dot but no known extension stays text.
func TestMarkdownFilePathReadsAsAFileChip(t *testing.T) {
	cases := []struct {
		word string
		want bool
	}{
		{"internal/ui/shell.go:42", true},
		{"docs/x.md", true},
		{"foo.go", true},
		{"README.md", true},
		{"config.yaml", true},
		{"pkg/a/b_test.go", true},
		{"./docs/x.md", true},
		{"../docs/x.md", true},
		{"/tmp/x.go", true},
		{"src/main.go:42:7", true},

		{"", false},
		{"internal/ui", false},               // a directory, no extension
		{"foo", false},                       // a word
		{"README", false},                    // a name, not a path to a file
		{"e.g.", false},                      // prose
		{"v1.2.3", false},                    // a version
		{"1.2", false},                       // a number
		{"example.com", false},               // a host
		{"example.com/x.md", false},          // an address, not a path
		{"https://example.com/x.md", false},  // a URI
		{"npm install", false},               // two words
		{"foo.bar", false},                   // an extension no file here has
		{"fmt.Println", false},               // Go prose
		{"a@b.com", false},                   // an address
		{".bashrc", false},                   // a dotfile, no extension
		{"foo.go.", false},                   // a sentence's full stop
		{"foo.md#L1", false},                 // a fragment
		{"internal/ui/shell.go:four", false}, // a line reference that is no number
		{strings.Repeat("a", fileChipMax) + ".go", false}, // past a line's room
	}
	for _, tc := range cases {
		if got := isFilePath(tc.word); got != tc.want {
			t.Errorf("isFilePath(%q) = %v, want %v", tc.word, got, tc.want)
		}
	}
}

// TestMarkdownFileChipPathsCutTheWordFromTheLine checks which bytes of a line
// of text a chip takes: the path's own, without the punctuation around it, the
// line reference included, and none of a URI's.
func TestMarkdownFileChipPathsCutTheWordFromTheLine(t *testing.T) {
	const text = "see internal/ui/shell.go:42, then docs/x.md. Not /a/dir, e.g. or https://example.com/x.md"
	var got []string
	for _, r := range fileChipPaths(text) {
		got = append(got, text[r[0]:r[1]])
	}
	want := []string{"internal/ui/shell.go:42", "docs/x.md"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("the line reads as the chips %q, want %q", got, want)
	}
}

// chipPixels counts the pixels of face within r, as faceShare counts a share:
// a chip paints its face behind the word, so a line with a chip has some, and
// a line of plain text none of that exact color.
func chipPixels(tt *ui.Tester, r ui.Rect, face ui.Color) int {
	img := tt.Image()
	hits := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
				continue
			}
			if p := img.RGBAAt(x, y); p.R == face.R && p.G == face.G && p.B == face.B && p.A == face.A {
				hits++
			}
		}
	}
	return hits
}

// TestMarkdownFileChipPaintsAPathAsCode checks a path in a line of text end to
// end: the view shows the path itself with the inline code face behind it (the
// muted face, monospace, which is the chip mygo's public theme can draw), the
// line selects and copies as the path, and the view's own Markdown copy shows
// the chip as the code span it is.
func TestMarkdownFileChipPaintsAPathAsCode(t *testing.T) {
	const sentence = "See internal/ui/shell.go:42 and docs/x.md."
	k := newTokens(ui.LightTheme())
	tt := markdownTester(t, 640, 200, sentence+"\n", MarkdownOptions{})

	markdownKind(t, tt, sentence, "`")
	line, ok := tt.Find(sentence)
	if !ok {
		t.Fatalf("the line is missing, texts %q", tt.Texts())
	}
	if hits := chipPixels(tt, line, k.muted); hits < 100 {
		t.Errorf("%d pixels of the chip face behind the paths, want the words tinted", hits)
	}

	// A line with no path is not tinted, so the face comes from the chip and
	// not from the text around it.
	const plainSentence = "See the shell and the docs."
	plain := markdownTester(t, 640, 200, plainSentence+"\n", MarkdownOptions{})
	plainLine, ok := plain.Find(plainSentence)
	if !ok {
		t.Fatalf("the plain line is missing, texts %q", plain.Texts())
	}
	if hits := chipPixels(plain, plainLine, k.muted); hits > 20 {
		t.Errorf("%d pixels of the chip face behind text with no path, want the face to come from a chip", hits)
	}

	// The chip is part of the text: selecting the line copies the path as it
	// reads, without the code span around it...
	tt.ClickAt(line.X+2, line.Y+line.H/2)
	tt.Command("selectAll")
	tt.Command("copy")
	if got := tt.Clipboard(); got != sentence {
		t.Errorf("the text copy is %q, want %q", got, sentence)
	}
	// ...and Copy as Markdown shows the chip as the code span the view draws,
	// which is the rendering mygo copies, not the author's source.
	tt.Key(ui.Cmd|ui.Shift, ui.KeyC)
	if got, want := tt.Clipboard(), "See `internal/ui/shell.go:42` and `docs/x.md`."; got != want {
		t.Errorf("the Markdown copy is %q, want %q", got, want)
	}
}

// TestMarkdownFileChipLeavesCodeAlone checks that the reading never reaches
// text it cannot rewrite: a fenced block, a code span, a link's destination and
// an escaped word keep their bytes, so a code block still copies exactly the
// code it shows.
func TestMarkdownFileChipLeavesCodeAlone(t *testing.T) {
	cases := []struct {
		name        string
		src         string
		linked      bool
		want        string
		wantDemoted bool
	}{
		{"a path in text", "see docs/x.md\n", false, "see `docs/x.md`\n", false},
		{"a path in a code span", "see `docs/x.md`\n", false, "see `docs/x.md`\n", false},
		{"a path in a fenced block", "```\nsee docs/x.md\n```\n", false, "```\nsee docs/x.md\n```\n", false},
		{"a path in a link's destination", "[docs](https://e.com/docs/x.md)\n", true, "[docs](https://e.com/docs/x.md)\n", false},
		{"a path in a link's text", "[see docs/x.md](https://e.com)\n", true, "[see `docs/x.md`](https://e.com)\n", false},
		{"an escaped path", "see docs/x\\.md\n", false, "see docs/x\\.md\n", false},
		{"a word that is no path", "see docs/x, e.g. and v1.2.3\n", false, "see docs/x, e.g. and v1.2.3\n", false},
		{"a link without a callback", "see [docs](https://e.com)\n", false, "see docs\n", true},
		{"a link with a callback", "see [docs](https://e.com)\n", true, "see [docs](https://e.com)\n", false},
		{"an anchor link without a callback", "see [docs](#docs)\n", false, "see [docs](#docs)\n", false},
	}
	for _, tc := range cases {
		got, demoted := contentSource(tc.src, tc.linked)
		if got != tc.want {
			t.Errorf("%s: the view parses %q, want %q", tc.name, got, tc.want)
		}
		if demoted != tc.wantDemoted {
			t.Errorf("%s: demoted is %v, want %v", tc.name, demoted, tc.wantDemoted)
		}
	}

	// The code block's copy button copies the block's source, so a path the
	// reading had rewritten would show on the clipboard.
	const code = "see internal/ui/shell.go:42 here\n"
	tt := markdownTester(t, 640, 240, "```\n"+code+"```\n", MarkdownOptions{})
	if err := tt.Click("Copy"); err != nil {
		t.Fatalf("the copy button: %v", err)
	}
	if got := tt.Clipboard(); got != code {
		t.Errorf("the block copied %q, want %q", got, code)
	}
}

// TestMarkdownLinkCallsTheHandler checks a link where the layer has somewhere
// to hand it: the click reaches the callback with the destination, the view
// opens nothing itself, and mygo draws the link as a link, which the pointer
// cursor and the link role record.
func TestMarkdownLinkCallsTheHandler(t *testing.T) {
	var opened []string
	tt := markdownTester(t, 640, 240, "See [the guide](https://example.com/guide) now.\n", MarkdownOptions{
		OnLink: func(url string) { opened = append(opened, url) },
	})

	info, ok := tt.AccessByRole(ui.RoleLink, "the guide")
	if !ok {
		t.Fatalf("the link is not exposed as a link, texts %q", tt.Texts())
	}
	tt.Move(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if got := tt.Cursor(); got != ui.CursorPointer {
		t.Errorf("the pointer over the link is %v, want the hand %v", got, ui.CursorPointer)
	}
	if err := tt.Click("the guide"); err != nil {
		t.Fatalf("the link: %v", err)
	}
	if len(opened) != 1 || opened[0] != "https://example.com/guide" {
		t.Errorf("the link handed %q to the handler, want https://example.com/guide", opened)
	}
	if urls := tt.OpenedURLs(); len(urls) != 0 {
		t.Errorf("the link opened %q, want the handler to decide", urls)
	}
	if tt.HasText(wipBadge) {
		t.Errorf("a wired link still shows the WIP note, texts %q", tt.Texts())
	}
}

// TestMarkdownLinkWithoutCallbackShowsTextAndWIP checks a link where the layer
// has nowhere to hand it: it shows as its text, nothing opens, and the view
// says why with the shell's WIP badge rather than leaving a control that opens
// nothing and reads as a link.
func TestMarkdownLinkWithoutCallbackShowsTextAndWIP(t *testing.T) {
	const sentence = "See the guide now."
	tt := markdownTester(t, 640, 240, "See [the guide](https://example.com/guide) now.\n", MarkdownOptions{})

	markdownKind(t, tt, sentence, "[the guide](https://example.com/guide)")
	if _, ok := tt.AccessByRole(ui.RoleLink, "the guide"); ok {
		t.Errorf("the link is still a link without a callback, texts %q", tt.Texts())
	}
	line, ok := tt.Find(sentence)
	if !ok {
		t.Fatalf("the sentence is missing, texts %q", tt.Texts())
	}
	// Not a control: the pointer over the link's text is the text cursor, and
	// a click opens nothing and does nothing.
	tt.Move(line.X+line.W/2, line.Y+line.H/2)
	if got := tt.Cursor(); got == ui.CursorPointer {
		t.Errorf("the pointer over the link's text is the hand, as if it were clickable")
	}
	if err := tt.Click(sentence); err != nil {
		t.Fatalf("the sentence: %v", err)
	}
	if urls := tt.OpenedURLs(); len(urls) != 0 {
		t.Errorf("a click opened %q, want nothing", urls)
	}

	// The gap is visible: the badge and the line it explains, with the reason
	// for assistive technology.
	if !tt.HasText(wipBadge) {
		t.Errorf("the view shows no %q badge, texts %q", wipBadge, tt.Texts())
	}
	if !tt.HasText(linkWipLabel) {
		t.Errorf("the view does not say %q, texts %q", linkWipLabel, tt.Texts())
	}
	note, ok := tt.AccessByRole(ui.RoleGroup, linkWipLabel)
	if !ok {
		t.Fatalf("the note is not named %q to assistive technology, texts %q", linkWipLabel, tt.Texts())
	}
	if !strings.Contains(note.Description, reasonLink) {
		t.Errorf("the note describes %q, want the reason %q", note.Description, reasonLink)
	}

	// A link to a heading still works without a callback: it stays a link and
	// the view scrolls to the heading itself.
	anchor := markdownTester(t, 640, 240, "Jump to [the end](#the-end).\n\n## The end\n\nLast words.\n", MarkdownOptions{})
	if _, ok := anchor.AccessByRole(ui.RoleLink, "the end"); !ok {
		t.Errorf("an anchor link is shown as text, texts %q", anchor.Texts())
	}
	if anchor.HasText(wipBadge) {
		t.Errorf("an anchor link shows the WIP note, texts %q", anchor.Texts())
	}
}
