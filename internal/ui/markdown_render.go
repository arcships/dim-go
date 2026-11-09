package ui

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
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
	// OnLink handles a click on a link, given its destination. Nil lets
	// mygo open http, https and mailto links itself, and scroll to a link to
	// #a-heading.
	OnLink func(url string)
}

// renderMarkdownContent renders text as the content layer: the Markdown view
// of one message or block, themed with the shell's tokens and with the code
// block head below.
//
// Build it once a frame, in the same place in the tree: the Markdown view keeps
// what it does not parse again, which code block was copied, and each table's
// column widths in local state keyed to that place.
//
// The options are built here rather than kept by the caller because they hold
// no state of their own; mygo's own Options would (its image cache and reveal
// state), so a caller that needs images or streaming passes text through View
// and Stream itself instead of handing them to this layer.
func renderMarkdownContent(c *ui.Context, k tokens, text string, opts MarkdownOptions) ui.Element {
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
	return markdown.View(c, text, o)
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
