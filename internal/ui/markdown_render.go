package ui

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/markdown/parser"
)

// The content layer is the shell's own reading surface: one Markdown view per
// assistant turn, thinking block or tool result, themed with the shell's tokens
// rather than mygo's defaults, with a code block head that names the language
// and copies the code. docs/ui-parity.md §2.2 lists the Markdown tokens the
// shell had none of, and §3.1 lists the code block head among the renderer's
// P0 gaps; docs/ui-layout-spec.md §6 places the layer in the P0 order.
//
// The layer is not wired into the timeline yet: the shell and the timeline still
// build their rows with markdown.View and their own Options (shell.go,
// timeline.go), which a later change switches over to renderMarkdownContent.
//
// Two readings of the text are the layer's own, since markdown.View has no hook
// inside a paragraph (Options.CodeBlock builds fenced blocks, not the inlines of
// text), so both go through the source the view parses (contentSource):
//
//   - A word that reads as a path to a file, as internal/ui/shell.go:42 or
//     docs/x.md, becomes a file chip: a code span, which the content theme
//     draws monospace on the muted face, and which the view selects and copies
//     as its text. The shell has no file to open, so a chip is shown and not
//     acted on; the renderer's FileChip.vue also opens the file, which this
//     layer cannot until the window has a file-open callback.
//   - A link shows as its text, followed by a WIP note, while the layer has no
//     OnLink: mygo would otherwise open http, https and mailto links itself, and
//     a control that opens what the shell cannot vouch for is worse than plain
//     text. With OnLink, links stay links and mygo draws their hover.

// noCodeLanguage is what a fenced code block's head shows when its fence names
// no language. The renderer says "text" rather than guessing one from the
// code's content, and the block's accessible name says the same, so the label
// never claims a language the author did not: see markdown_render_test.go.
const noCodeLanguage = "text"

// The content layer's own steps.
const (
	// codeHeadRise is how far a code block's head lifts off the block's
	// face, as the renderer's --code-header-bg mixes 8% of the foreground
	// into the code background (styles/timeline-markdown.css:48).
	codeHeadRise = 0.08

	// inlineCodeSize is an inline code span's size in ems, as the
	// renderer's .inline-code sets 0.875em (styles/timeline-markdown.css).
	// mygo sizes inline code and code blocks with the one Theme.CodeSize,
	// so a block here is 0.875em too where the renderer gives it the
	// separate 13px of --dimcode-code-font-size; see contentTheme's gaps.
	inlineCodeSize = 0.875

	// tableHeaderRise is how far a table's header row lifts off the table's
	// face. mygo has no table face of its own, so the header row is the one
	// cell face the theme can name.
	tableHeaderRise = 0.05
)

// MarkdownOptions are one content view's choices beyond the shell's tokens.
// The zero value is the reading surface a finished assistant turn gets: code
// that scrolls sideways rather than wrapping, and links opened as mygo opens
// them.
//
// The options are a plain value rebuilt every frame; the view keeps its parse,
// its copy state and its table widths in local state keyed to its place in the
// tree, so a caller passes the same options on every frame and changes them
// only when the view should change. What is not here is the view's own state,
// as whether a code block was copied or how far it was scrolled.
type MarkdownOptions struct {
	// WrapCode wraps the lines of code blocks rather than scrolling them
	// sideways, for a region narrow enough that scrolling would hide more
	// than wrapping does.
	WrapCode bool
	// OnLink handles a click on a link, given its destination. Without it a
	// link shows as its text, under a WIP note, rather than as a control that
	// opens nothing: mygo would otherwise open http, https and mailto links
	// itself. A link to #a-heading still scrolls to that heading, which the
	// view does without a callback.
	OnLink func(url string)
}

// renderMarkdownContent renders text as the content layer: the Markdown view
// of one message or block, themed with the shell's tokens and with the code
// block head below.
//
// Build it once a frame, in the same place in the tree: the Markdown view keeps
// what it does not parse again, which code block was copied, and each table's
// column widths in local state keyed to that place. The layer keeps the rewrite
// of the text there too, so unchanged text is not parsed twice a frame.
//
// The options are built here rather than kept by the caller because they hold
// no state of their own; mygo's own Options would (its image cache and reveal
// state), so a caller that needs images or streaming passes text through View
// and Stream itself instead of handing them to this layer.
func renderMarkdownContent(c *ui.Context, k tokens, text string, opts MarkdownOptions) ui.Element {
	box := ui.Column(c)
	cache := ui.Local(box, contentSourceKey{}, func() sourceCache { return sourceCache{} })
	src, demoted := cache.rewrite(text, opts.OnLink != nil)

	o := &markdown.Options{
		Theme:    contentTheme(k),
		WrapCode: opts.WrapCode,
		OnLink:   opts.OnLink,
		// The code block hook replaces mygo's DefaultCodeBlock, not wraps
		// it: the default already draws a head, and wrapping it would put a
		// second head inside this one.
		CodeBlock: func(c *ui.Context, b *markdown.CodeBlockView) {
			codeBlock(c, k, b)
		},
	}
	box.Children(func() {
		markdown.View(c, src, o)
		if demoted {
			linkWipNote(c, k)
		}
	})
	return box
}

// contentSourceKey keys the layer's rewrite of its text among an element's
// local state.
type contentSourceKey struct{}

// sourceCache keeps one content view's last rewrite of its text, so a frame
// that shows unchanged text does not parse it again: the view parses
// incrementally, the layer's rewrite does not.
type sourceCache struct {
	text, src string
	linked    bool
	demoted   bool
	done      bool
}

// rewrite returns the Markdown the view parses for text, rewriting it only when
// the text or whether the layer has a link callback changed.
func (s *sourceCache) rewrite(text string, linked bool) (src string, demoted bool) {
	if s.done && s.text == text && s.linked == linked {
		return s.src, s.demoted
	}
	src, demoted = contentSource(text, linked)
	*s = sourceCache{text: text, src: src, linked: linked, demoted: demoted, done: true}
	return src, demoted
}

// contentSource returns what the content view parses for text: text with every
// word that reads as a file path in a code span, and, without a link callback,
// every link with its destination removed so that it shows as its text.
// demoted reports whether a link was shown as text.
//
// The rewrite goes through the parser instead of scanning the source, so a path
// in a fenced block, in a code span or in a link's destination, and a word with
// an escape or an entity in it, keep their bytes: only an inline's own literal
// text is rewritten, and its offsets map to the source exactly because the
// bytes are the same (see the Test case for escapes in markdown_render_test.go).
//
// A chip reads back as a code span in the view's own Markdown copy, since mygo
// copies the source it parsed. That is the rendering the user sees, which is
// what Copy as Markdown promises; the author's source keeps its own bytes.
func contentSource(text string, linked bool) (src string, demoted bool) {
	d := parser.Parse(text)
	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	for i := 1; i < d.Len(); i++ {
		in := d.Inlines(i)
		for k := 0; k < len(in); k++ {
			n := &in[k]
			switch n.Kind {
			case parser.Link:
				if linked || n.Pending || strings.HasPrefix(n.Dest, "#") {
					continue
				}
				label := ""
				if n.Next > k+1 {
					label = text[in[k+1].Start:in[n.Next-1].End]
				}
				// A chip within the link's text stays a chip: the text is
				// rewritten on its own, as the paragraph would have been.
				labelSrc, _ := contentSource(label, true)
				edits = append(edits, edit{n.Start, n.End, labelSrc})
				demoted = true
				k = n.Next - 1 // the label is rewritten with the link
			case parser.Image:
				// A description is not shown as text: leave its bytes alone.
				k = n.Next - 1
			case parser.Text:
				if text[n.Start:n.End] != n.Text {
					continue // an escape or an entity: the offsets do not map
				}
				for _, r := range fileChipPaths(n.Text) {
					edits = append(edits, edit{
						start: n.Start + r[0],
						end:   n.Start + r[1],
						text:  "`" + n.Text[r[0]:r[1]] + "`",
					})
				}
			}
		}
	}
	if len(edits) == 0 {
		return text, false
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	var b strings.Builder
	last := 0
	for _, e := range edits {
		if e.start < last {
			continue // a rewrite within one already made
		}
		b.WriteString(text[last:e.start])
		b.WriteString(e.text)
		last = e.end
	}
	b.WriteString(text[last:])
	return b.String(), demoted
}

// contentTheme is the Markdown theme of the shell's tokens: the semantic faces
// and colors the renderer's --markdown-* tokens name, derived from tokens here
// so that a theme or dark-mode change carries through the content, and so that
// no view carries a color of its own.
//
// What mygo's public theme cannot express is kept as a documented gap rather
// than faked with a color that would pass for it:
//
//   - An inline code span has a color, a background, a size and mygo's own
//     corner (Size/4), but no padding and no hairline inset ring: it is a
//     tinted run, where the renderer gives it padding 0.1em 0.35em, radius
//     0.375em and a 0.5px inset ring (styles/timeline-markdown.css:154-163).
//     A file chip is that run: the renderer's FileChip.vue also sets a file
//     icon and padding, which no public option reaches inside a paragraph.
//   - A link has one color and one weight floor for every state. The
//     renderer's hover color and underline, and its link icon hook
//     (Options.LinkIcon is public, the icon a view builds is not), have no
//     slot here.
//   - A table has a border, a header face and a stripe, but no face of its
//     own and no row hover: --markdown-table-surface and
//     --markdown-table-row-hover stay unimplemented. The header takes a step
//     up from the card (tableHeaderRise) because there is no table surface to
//     tint.
//   - The rule color is shared by thematic breaks, heading rules and the
//     footnote separator, so --markdown-hr cannot differ from a heading's
//     rule.
//   - A task list's check box draws its checked fill from Theme.Checkbox but
//     its unchecked outline from Theme.Muted, which list markers and a code
//     block's language share: --markdown-checkbox-border, -bg and -ring have
//     no slots.
//   - Headings keep mygo's scale (1.6/1.35/1.15/1/0.95/0.9em), where the
//     renderer's timeline sets 1.25/1.1875/1.125/1.0625rem for h1–h4 and
//     leaves h5–h6 to its own defaults (styles/timeline-markdown.css:19-24).
//   - Code coloring is mygo's editor default, since no code theme is wired;
//     the renderer's six code themes are deferred with the appearance
//     surface (docs/ui-parity.md §5, step 4).
//
// The quote bar, its text and the strong weight follow the renderer's defaults,
// and the muted text, the list markers and a code block's language label all
// read from Theme.Muted, so they are tokens rather than mygo's own palette.
func contentTheme(k tokens) *markdown.Theme {
	return &markdown.Theme{
		FontSize: k.bodySize,

		Text:   k.foreground,
		Muted:  k.mutedForeground,
		Link:   k.primary,
		Strong: k.foreground,

		// Inline code: the renderer's --inline-code-bg is the secondary face
		// at about half strength; mygo has one color and no per-span alpha
		// compositing control, so the muted face stands in for it.
		Code:           k.foreground,
		CodeBackground: k.muted,
		CodeSize:       inlineCodeSize,

		QuoteBar:  k.border,
		QuoteText: k.mutedForeground,

		// Thematic breaks, heading rules and the footnote separator: one
		// color, --markdown-hr and --border agreeing in the renderer.
		Rule: k.border,

		// A code block is the card step of the surface ladder: the renderer's
		// --markdown-code-block-bg comes from its code theme, which is not
		// wired, and mygo's default is a wash of the text color. Its border,
		// corners and language label follow the shell's card.
		CodeBlockBackground: k.card,
		CodeBlockBorder:     k.border,
		CodeBlockRadius:     k.cardRadius,
		// 0.85em of room inside a block: mygo's default, kept, since the
		// renderer's pre padding is not a token.
		CodeBlockPadding: 0.85,

		// A table's cells are a foreground at 90% in the renderer; mygo
		// colors them with Text, so the cell text is the body text. A
		// header row is a step up from the card, and no row is striped, as
		// the renderer stripes none.
		TableBorder:      k.border,
		TableHeader:      k.card.Mix(k.foreground, tableHeaderRise),
		TableCellPadding: [2]float32{0.6, 0.4},
		TableMaxColumn:   24,

		// The renderer's --markdown-checkbox-checked is --primary.
		Checkbox: k.primary,
	}
}

// codeBlock builds one fenced code block as the content layer shows it: a head
// carrying the fence's language and mygo's copy button, over the highlighted
// code.
//
// The head is built here rather than by wrapping markdown.DefaultCodeBlock for
// two reasons: the default prints an empty label for a fence with no language,
// where the renderer says "text", and the default's head is the only place its
// face can be reached, since a wrapping hook would put a second head inside it.
//
// Copying is mygo's own: CodeBlockView.CopyButton copies the block's source
// with the window's clipboard (Context.WriteClipboard, which exists), shows
// "Copied" for two seconds, and waits for a block still being written to close.
// It cannot report a refused write, because it copies with WriteClipboard
// rather than WriteClipboardThen; a refused write leaves the clipboard empty
// while the button still says Copied, which markdown_render_test.go records.
func codeBlock(c *ui.Context, k tokens, b *markdown.CodeBlockView) {
	lang := b.Language
	if lang == "" {
		lang = noCodeLanguage
	}
	box := ui.Column(c).Direction(ui.LeftToRight).
		Background(k.card).Border(k.hairline, k.border).Radius(k.cardRadius).
		AlignSelf(ui.Stretch).Clip()
	box.Role(ui.RoleGroup).Label(codeGroupLabel(lang))
	box.Children(func() {
		ui.Row(c).AlignItems(ui.Center).AlignSelf(ui.Stretch).
			Background(k.card.Mix(k.foreground, codeHeadRise)).
			Padding(k.space(1), k.space(2)).Children(func() {
			ui.Text(c, lang).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
			ui.Spacer(c)
			b.CopyButton(c)
		})
		b.Body(c)
	})
}

// codeGroupLabel names a code block to assistive technology: the code and the
// language it is in, which is noCodeLanguage for a fence that names none. It
// matches mygo's own label shape ("Code, go").
func codeGroupLabel(lang string) string { return "Code, " + lang }

// The content layer's readings of a text it cannot act on. A link without a
// callback is shown as its text and the view says why below it, so the link is
// not a control that opens nothing (wipBadge marks it, as the shell's other WIP
// entries do: see wip.go).
const (
	// linkWipLabel is the line added below a text whose links show as text.
	linkWipLabel = "Links are not clickable here yet."
	// reasonLink is why: the window has no callback to open one with, and the
	// layer does not let mygo open links itself.
	reasonLink = "Unavailable: the content layer has no OnLink callback, so it shows a link as text instead of opening it."
)

// linkWipNote adds that line below a text whose links showed as text. It is
// unselectable, and sits beside the Markdown view rather than in it, so it is
// never selected, copied or read as part of the message: it describes the view.
func linkWipNote(c *ui.Context, k tokens) {
	row := ui.Row(c).AlignItems(ui.Center).Gap(k.gap)
	row.Role(ui.RoleGroup).Label(linkWipLabel).Description(reasonLink)
	row.Children(func() {
		ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
			Unselectable().AccessHidden(true)
		ui.Text(c, linkWipLabel).FontSize(k.smallSize).TextColor(k.mutedForeground).
			Unselectable()
	})
}

// The bounds of a word that reads as a file chip, in bytes: "a.go" at the
// shortest, and no more than a path a line of text can hold.
const (
	fileChipMin = 4
	fileChipMax = 120
)

// fileChipExts are the extensions a word must carry to read as a file chip. It
// is an allowlist of unambiguous source, document and configuration extensions
// rather than "anything with a dot": prose that reads as a path when it is not
// one, as e.g, v1.2.3, fmt.Println or example.com, stays text, since a wrong
// chip claims a file the author never named.
var fileChipExts = map[string]bool{
	"bash": true, "bat": true, "c": true, "cc": true, "cfg": true, "clj": true,
	"conf": true, "cpp": true, "cs": true, "css": true, "cjs": true, "dart": true,
	"diff": true, "erl": true, "ex": true, "exs": true, "fish": true, "go": true,
	"gql": true, "graphql": true, "h": true, "hpp": true, "hrl": true, "htm": true,
	"html": true, "ini": true, "java": true, "js": true, "json": true, "jsonc": true,
	"jsx": true, "kt": true, "kts": true, "less": true, "lock": true, "lua": true,
	"md": true, "markdown": true, "mjs": true, "mod": true, "patch": true, "php": true,
	"proto": true, "ps1": true, "py": true, "rb": true, "rs": true, "sass": true,
	"scala": true, "scss": true, "sh": true, "sql": true, "sum": true, "svelte": true,
	"swift": true, "tex": true, "toml": true, "ts": true, "tsx": true, "txt": true,
	"vue": true, "yaml": true, "yml": true, "zsh": true,
}

// isFilePath reports whether s reads as a path to a file: a word of path bytes
// whose last part names a known extension, with an optional :line or
// :line:column on the end. It answers for the word a line of text holds, not for
// a file that exists: the content layer has no file system to ask.
func isFilePath(s string) bool {
	if len(s) < fileChipMin || len(s) > fileChipMax {
		return false
	}
	base := s
	if i := strings.IndexByte(s, ':'); i >= 0 {
		if !isLineRef(s[i+1:]) {
			return false
		}
		base = s[:i]
	}
	for i := 0; i < len(base); i++ {
		if !isPathByte(base[i]) {
			return false
		}
	}
	slash := strings.LastIndexByte(base, '/')
	name := base[slash+1:]
	dot := strings.LastIndexByte(name, '.')
	if dot <= 0 || dot == len(name)-1 {
		return false // a name with no room for an extension, or .bashrc
	}
	if !fileChipExts[name[dot+1:]] {
		return false
	}
	// A host with a dot in it, as example.com/x.md, is an address, not a path.
	if slash > 0 {
		if head := base[:strings.IndexByte(base, '/')]; head != "." && head != ".." && strings.Contains(head, ".") {
			return false
		}
	}
	return true
}

// isLineRef reports whether s is a :line or :line:column suffix, as in
// internal/ui/shell.go:42:7.
func isLineRef(s string) bool {
	n := 0
	for n < len(s) && isDigitByte(s[n]) {
		n++
	}
	if n == 0 {
		return false
	}
	if n == len(s) {
		return true
	}
	if s[n] != ':' {
		return false
	}
	return isDigits(s[n+1:])
}

// fileChipPaths returns the byte ranges of text that read as file chips, in
// document order.
func fileChipPaths(text string) [][2]int {
	var paths [][2]int
	for i := 0; i < len(text); {
		if !isPathByte(text[i]) {
			i++
			continue
		}
		j := i
		for j < len(text) && isPathByte(text[j]) {
			j++
		}
		end := j
		// A :line or :line:column suffix belongs to the path.
		if end < len(text) && text[end] == ':' {
			after := end + 1
			for after < len(text) && isDigitByte(text[after]) {
				after++
			}
			if after > end+1 {
				if after < len(text) && text[after] == ':' {
					col := after + 1
					for col < len(text) && isDigitByte(text[col]) {
						col++
					}
					if col > after+1 {
						after = col
					}
				}
				j, end = after, after
			}
		}
		// A sentence's full stop, or a slash after a directory, is not the
		// path's: docs/x.md. reads as docs/x.md.
		for end > i && (text[end-1] == '.' || text[end-1] == '/') {
			end--
		}
		if end > i && isFilePath(text[i:end]) && !inURIToken(text, i) {
			paths = append(paths, [2]int{i, end})
		}
		i = j
	}
	return paths
}

// inURIToken reports whether the word around offset i is part of a URI, as the
// path of https://example.com/x.md is: a URI is not a file chip, whatever its
// last part reads as.
func inURIToken(text string, i int) bool {
	start, end := i, i
	for start > 0 && !isSpaceByte(text[start-1]) {
		start--
	}
	for end < len(text) && !isSpaceByte(text[end]) {
		end++
	}
	return strings.Contains(text[start:end], "://")
}

// isPathByte reports whether b may stand within a path: a letter, a digit, and
// the separators a relative path uses.
func isPathByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '.', b == '_', b == '-', b == '/':
		return true
	}
	return false
}

// isDigitByte reports whether b is a decimal digit.
func isDigitByte(b byte) bool { return b >= '0' && b <= '9' }

// isDigits reports whether s is one or more decimal digits.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigitByte(s[i]) {
			return false
		}
	}
	return true
}

// isSpaceByte reports whether b is white space, which ends a word of text.
func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
