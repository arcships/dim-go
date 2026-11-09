// This file owns the conversation's find bar: the toolbar's search field, the
// "n/m" count beside it, the entries that step through the matches, and the
// highlight the timeline draws for the current one.
//
// The bar is a nested region of the toolbar, as the approval row is one of the
// timeline: header.go draws it at the end of the bar while ThreadSearchView says
// it is open, and thread_search.go reads that view and Actions only (see
// protocols.go). Its matches are a display copy of what the window already
// shows: threadSearchMatches counts the query in the rows the timeline draws and
// starts no App Server read, so nothing here becomes a second source of
// timeline facts.
//
// The reference is dim-electron's ThreadSearchBar.vue: Cmd+F opens the field,
// Enter and Shift+Enter step through the matches and wrap around, Escape closes
// the bar, and the count reads "n/m", "0/0" while nothing matches. The reference
// stacks its field over a second row of entries; this bar is one row, so it fits
// the 41 DIP toolbar and shrinks with the window instead of running past it.

package ui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// The bar's own steps, in DIPs. The row keeps the toolbar's 28 DIP entry height;
// the field takes the room that is left between the count and the three entries,
// each a 20 DIP square with the toolbar's own small gap. The count keeps a fixed
// share so a query that matches nothing, "0/0", does not move the entries.
const (
	threadSearchW      = 320
	threadSearchEntry  = 20
	threadSearchGap    = 4
	threadSearchPadX   = 8
	threadSearchCountW = 44
)

// threadSearchHighlightAlpha is the strength of the face behind the run the
// current match highlights: the brand color as a wash strong enough to read as a
// mark over the text and light enough to keep the words legible in both
// appearances. A banner's wash (tokens.go) is lighter and fills a whole row.
const threadSearchHighlightAlpha = 0.28

// The keys of the bar's own text. i18n.go does not carry them yet, so
// threadSearchLabel falls back to the English text below rather than drawing a
// raw key; the entries go away as the table picks the keys up.
const (
	keyThreadSearchRegion      = "threadSearch.region"
	keyThreadSearchPlaceholder = "threadSearch.placeholder"
	keyThreadSearchLabel       = "threadSearch.label"
	keyThreadSearchPrevious    = "threadSearch.previous"
	keyThreadSearchNext        = "threadSearch.next"
	keyThreadSearchClose       = "threadSearch.close"
	keyThreadSearchResults     = "threadSearch.results"
	keyThreadSearchNoResults   = "threadSearch.noResults"
)

// threadSearchFallbacks is the English text of the keys above, as the reference
// draws them. It is a stand-in for the message table, not a second language
// table: threadSearchLabel consults it only when tr reports a missing key.
var threadSearchFallbacks = map[string]string{
	keyThreadSearchRegion:      "Conversation search",
	keyThreadSearchPlaceholder: "Find in conversation",
	keyThreadSearchLabel:       "Find in conversation",
	keyThreadSearchPrevious:    "Previous match",
	keyThreadSearchNext:        "Next match",
	keyThreadSearchClose:       "Close search",
	keyThreadSearchResults:     "%d of %d matches",
	keyThreadSearchNoResults:   "No matches",
}

// threadSearchLabel resolves one of the bar's labels. The lookup goes through the
// locale table first, so a translated entry wins as soon as i18n.go carries the
// key; a missing entry draws the English fallback instead of the key.
func threadSearchLabel(locale Locale, key string) string {
	if text := tr(locale, key); text != key {
		return text
	}
	if text, ok := threadSearchFallbacks[key]; ok {
		return text
	}
	return key
}

// The bar's glyphs, in the sidebar's 16×16 outline frame: the two arrows that
// step through the matches and the cross that closes the bar. Each strokes
// currentColor, so the theme and the muted ink carry through.
var (
	threadSearchUpIcon    = sidebarIcon(`<path d="M8 12.5V3.5"/><path d="M4.6 7 8 3.5 11.4 7"/>`)
	threadSearchDownIcon  = sidebarIcon(`<path d="M8 3.5v9"/><path d="M4.6 9 8 12.5 11.4 9"/>`)
	threadSearchCloseIcon = sidebarIcon(`<path d="M4.6 4.6 11.4 11.4"/><path d="M11.4 4.6 4.6 11.4"/>`)
)

// threadSearchBar draws the find bar at the end of the toolbar while the window
// says it is open. The field edits ThreadSearchView.Query in place, so the next
// frame counts from the same text; Enter, Shift+Enter, Escape and the three
// entries ask the window through Actions, and every step puts the focus back in
// the field, so a stepped match can be stepped again with Enter.
func threadSearchBar(c *ui.Context, k tokens, v ThreadSearchView, a Actions) {
	if !v.IsOpen() || v.Query == nil {
		return
	}
	locale := v.Locale
	hasMatches := v.Total > 0
	bar := ui.Row(c).Key("thread-search").Width(threadSearchW).MaxWidth(threadSearchW).
		Shrink(1).MinWidth(0).Height(k.sidebarEntryH).AlignItems(ui.Center).
		Gap(threadSearchGap).Padding(0, threadSearchPadX, 0, threadSearchPadX).
		Radius(radiusMD).Background(k.muted).BorderWidth(k.hairline).BorderColor(k.border).
		Role(ui.RoleGroup).Label(threadSearchLabel(locale, keyThreadSearchRegion)).
		Transition(motionLayoutColors(c))
	bar.Children(func() {
		threadSearchField(c, k, v, a)
		threadSearchCount(c, k, v)
		threadSearchStep(c, k, v, a, -1, threadSearchUpIcon, threadSearchLabel(locale, keyThreadSearchPrevious), hasMatches)
		threadSearchStep(c, k, v, a, 1, threadSearchDownIcon, threadSearchLabel(locale, keyThreadSearchNext), hasMatches)
		threadSearchButton(c, k, "thread-search-close", threadSearchCloseIcon,
			threadSearchLabel(locale, keyThreadSearchClose), false, func() {
				if a.OnThreadSearchClose != nil {
					a.OnThreadSearchClose()
				}
			})
	})
	// The bar takes Escape for its own subtree, so that the entry the keyboard
	// focus is on closes the bar rather than only dismissing the tooltip that its
	// focus brought: an overlay's key comes before the window's, and a tooltip is
	// one. The field takes Escape itself (threadSearchField), and with the focus
	// outside the bar the window's own shortcut closes it (shell.go).
	if a.OnThreadSearchClose != nil {
		bar.OnShortcut(0, ui.KeyEscape, a.OnThreadSearchClose)
	}
}

// threadSearchField draws the bar's text field. It takes the room the bar has
// left, so a narrow toolbar narrows the field rather than pushing the count or
// the entries out of the window. Enter steps forward, Shift+Enter back and
// Escape closes the bar, as the reference's input does.
func threadSearchField(c *ui.Context, k tokens, v ThreadSearchView, a Actions) {
	placeholder := threadSearchLabel(v.Locale, keyThreadSearchPlaceholder)
	field := ui.TextInputBase(c.Key("thread-search-field"), v.Query).
		Grow(1).Shrink(1).MinWidth(0).Label(threadSearchLabel(v.Locale, keyThreadSearchLabel)).
		Placeholder(placeholder).TextColor(k.foreground)
	if v.Focus != nil {
		field.Bind(v.Focus)
	}
	field.OnTextKey(func(key ui.TextKey) bool {
		switch {
		case key.Key == ui.KeyEscape && key.Mods == 0:
			if a.OnThreadSearchClose == nil {
				return false
			}
			a.OnThreadSearchClose()
			return true
		case key.Key == ui.KeyEnter && key.Mods == 0:
			if a.OnThreadSearchNext == nil {
				return false
			}
			a.OnThreadSearchNext()
			return true
		case key.Key == ui.KeyEnter && key.Mods == ui.Shift:
			if a.OnThreadSearchPrevious == nil {
				return false
			}
			a.OnThreadSearchPrevious()
			return true
		}
		// Tab and everything else fall through: Tab moves the focus on, as it
		// does out of every other field.
		return false
	})
}

// threadSearchCount draws the "n/m" counter: the number of the match the
// timeline highlights over how many the query found, and "0/0" while it finds
// none. The compact pair is the counter's accessible name, and its description
// spells it out, as a screen reader reads a live count.
func threadSearchCount(c *ui.Context, k tokens, v ThreadSearchView) {
	ui.Text(c, threadSearchCountText(v.Current, v.Total)).FontSize(k.smallSize).
		TextColor(k.mutedForeground).SingleLine().Shrink(0).MinWidth(threadSearchCountW).
		TextAlign(ui.End).Role(ui.RoleStatus).
		Description(threadSearchCountLabel(v.Locale, v.Current, v.Total))
}

// threadSearchCountText is the counter's drawn text.
func threadSearchCountText(current, total int) string {
	return fmt.Sprintf("%d/%d", current, total)
}

// threadSearchCountLabel is what the counter reads after its drawn text: the
// spelled-out count, or that nothing matched.
func threadSearchCountLabel(locale Locale, current, total int) string {
	if total == 0 {
		return threadSearchLabel(locale, keyThreadSearchNoResults)
	}
	return fmt.Sprintf(threadSearchLabel(locale, keyThreadSearchResults), current, total)
}

// threadSearchStep draws one of the two entries that step through the matches.
// A query that matches nothing leaves them unavailable; they explain themselves
// with their label rather than pretending a move. A click steps and puts the
// focus back in the field, so Enter carries on from there.
func threadSearchStep(c *ui.Context, k tokens, v ThreadSearchView, a Actions, delta int, icon *ui.SVG, label string, enabled bool) {
	key := "thread-search-previous"
	step := func() {
		if v.Focus != nil {
			v.Focus.Focus()
		}
		if delta > 0 && a.OnThreadSearchNext != nil {
			a.OnThreadSearchNext()
		}
		if delta < 0 && a.OnThreadSearchPrevious != nil {
			a.OnThreadSearchPrevious()
		}
	}
	if delta > 0 {
		key = "thread-search-next"
	}
	threadSearchButton(c, k, key, icon, label, !enabled, step)
}

// threadSearchButton draws one square entry of the bar. Its label is its
// accessible name and its tooltip; the glyph repeats what the label says, so
// assistive technology skips it.
func threadSearchButton(c *ui.Context, k tokens, key string, icon *ui.SVG, label string, disabled bool, click func()) {
	button := ui.ButtonBase(c).Key(key).Size(threadSearchEntry, threadSearchEntry).Shrink(0).
		Radius(radiusSM).AlignItems(ui.Center).Justify(ui.Center).
		Label(label).Tooltip(label).Transition(motionFastColors(c))
	if disabled {
		button.Disabled(true)
	}
	if button.Hovered() {
		button.Background(k.sidebarHover)
	}
	button.OnClick(click).Children(func() {
		sidebarGlyph(c, icon, k.mutedForeground)
	})
}

// threadMatch is one occurrence of the query in the timeline's text: the row and
// the block it is in, and the byte range of that text it matched. Block is -1
// when the match is in the row's own text.
type threadMatch struct {
	Row   int
	Block int
	Start int
	End   int
}

// threadSearchRun is one run of a row that the bar searches: the row's own text
// (Block -1), or one of the turn's blocks.
type threadSearchRun struct {
	Block int
	Text  string
}

// threadSearchRuns lists the text the bar searches in one row, in the order the
// timeline draws it: what the user wrote, an assistant turn's text, thinking and
// tool calls, or a tool row's call. A file-changes block carries paths rather
// than prose, so the bar leaves it out instead of counting paths the row draws as
// a list.
func threadSearchRuns(m Message) []threadSearchRun {
	switch m.Author {
	case AuthorUser, AuthorError, AuthorRetry:
		return []threadSearchRun{{Block: -1, Text: m.Content}}
	case AuthorTool:
		return []threadSearchRun{{Block: -1, Text: toolSearchText(m.Tool)}}
	case AuthorAssistant:
		if len(m.Blocks) == 0 {
			return []threadSearchRun{{Block: -1, Text: m.Content}}
		}
		texts := make([]threadSearchRun, 0, len(m.Blocks))
		for i, block := range m.Blocks {
			switch block.Kind {
			case BlockText, BlockThinking:
				texts = append(texts, threadSearchRun{Block: i, Text: block.Content})
			case BlockTool:
				texts = append(texts, threadSearchRun{Block: i, Text: toolSearchText(block.Tool)})
			}
		}
		return texts
	}
	return nil
}

// toolSearchText is the text of one tool call the bar searches: its name, its
// params and its result, as the projection kept them. The card draws them as
// bounded previews, so a match in a tool call marks its row rather than one run
// of the preview.
func toolSearchText(tool Tool) string {
	return strings.Join([]string{toolName(tool), tool.Params, tool.Result}, "\n")
}

// threadSearchMatches counts the query's occurrences in the rows the timeline
// shows, in reading order: the order of the rows and, within a row, the order of
// its text or blocks. The query is trimmed and matched case-insensitively over
// runes; an empty query matches nothing.
//
// The count is a display copy, so it is made afresh on the frame that draws it:
// the shell keeps no index of the timeline's text of its own, and a row that
// leaves the projection takes its matches with it.
func threadSearchMatches(rows []Message, query string) []threadMatch {
	needle, _ := foldRunes(strings.TrimSpace(query))
	if len(needle) == 0 {
		return nil
	}
	var matches []threadMatch
	for row, m := range rows {
		for _, text := range threadSearchRuns(m) {
			for _, run := range threadMatchRuns(text.Text, needle) {
				matches = append(matches, threadMatch{Row: row, Block: text.Block, Start: run[0], End: run[1]})
			}
		}
	}
	return matches
}

// threadMatchRuns finds every non-overlapping occurrence of the folded needle in
// text, case-insensitively, and returns each one as a byte range of text, so the
// highlight cuts the text the row draws rather than a folded copy of it.
func threadMatchRuns(text string, needle []rune) [][2]int {
	hay, offsets := foldRunes(text)
	var runs [][2]int
	for at := 0; at+len(needle) <= len(hay); {
		if !runesMatchAt(hay, needle, at) {
			at++
			continue
		}
		runs = append(runs, [2]int{offsets[at], offsets[at+len(needle)]})
		at += len(needle)
	}
	return runs
}

// runesMatchAt reports whether hay holds needle at index at.
func runesMatchAt(hay, needle []rune, at int) bool {
	for i, r := range needle {
		if hay[at+i] != r {
			return false
		}
	}
	return true
}

// foldRunes lower-cases s one rune at a time and returns the folded runes with
// the byte offset in s of each one, plus len(s) as the offset of the end. Simple
// per-rune folding keeps the runes in step with the drawn text, so a match found
// in the folded runes maps back to a byte range of the text itself.
func foldRunes(s string) ([]rune, []int) {
	folded := make([]rune, 0, len(s))
	offsets := make([]int, 0, len(s)+1)
	for i, r := range s {
		offsets = append(offsets, i)
		folded = append(folded, unicode.ToLower(r))
	}
	return folded, append(offsets, len(s))
}

// textRun is a piece of a text the find bar highlights: the run itself, or the
// text around it.
type textRun struct {
	Text      string
	Highlight bool
}

// textRuns cuts text at the highlighted run, in byte offsets the bar's own
// matching produced. A range outside the text, or an empty one, leaves the text
// as one plain run rather than dropping it.
func textRuns(text string, hl TextHighlight) []textRun {
	start, end := max(0, hl.Start), min(len(text), hl.End)
	if start >= end {
		return []textRun{{Text: text}}
	}
	runs := make([]textRun, 0, 3)
	if before := text[:start]; before != "" {
		runs = append(runs, textRun{Text: before})
	}
	runs = append(runs, textRun{Text: text[start:end], Highlight: true})
	if after := text[end:]; after != "" {
		runs = append(runs, textRun{Text: after})
	}
	return runs
}

// highlightText draws text with the find bar's current match marked, as the
// timeline does where it draws the text itself: the user's bubble and a thinking
// block. The run keeps its color and takes the highlight face behind it, so the
// words read the same as before.
func highlightText(c *ui.Context, k tokens, text string, hl TextHighlight, size float32, color ui.Color) ui.Element {
	runs := textRuns(text, hl)
	spans := make([]ui.Span, 0, len(runs))
	for _, run := range runs {
		span := ui.Span{Text: run.Text, Size: size, Color: color}
		if run.Highlight {
			span.Background = k.primary.Alpha(threadSearchHighlightAlpha)
		}
		spans = append(spans, span)
	}
	return ui.RichText(c, spans...)
}
