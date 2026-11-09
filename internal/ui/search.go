// This file owns the window's search palette: the centered dialog that Cmd+G
// and the sidebar's Search entry open, with an input, the recent-session
// results and the shortcut that opens one.
//
// The reference is docs/evidence/electron-reference/window-search.png and
// window-search-results.png: a 448 DIP panel rounded by 12, lifted by a strong
// shadow over a black 80% scrim, a 56 DIP input row with a magnifier and the
// "搜索聊天" placeholder that takes the focus as the dialog opens, then the
// "最近聊天" heading and 42 DIP rows. Each row is a 32 DIP icon tile, the
// session's title over its project, and the shortcut that opens it (Cmd+1 to
// Cmd+8 on macOS, Ctrl+1 to Ctrl+8 elsewhere). Escape and a click on the scrim
// close it, and the focus returns to the control that opened it.
//
// The palette draws SearchPaletteView and Actions only (see protocols.go). The
// results are a display copy of the catalog the shell already holds: the
// palette derives no session fact, sends no App Server request, and writes
// nothing except its own selection through SearchPaletteView.Selected. That is
// why the pinned grouping the reference can show is not drawn: this shell has
// no pin fact, so the only heading is the recent chats it does have.

package ui

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// The palette's geometry in DIPs, from the reference at its 14px root font
// size: the panel's max-w-lg is 448, rounded-xl is 12, the input row is h-14
// (56) over a hairline, the group heading is roughly py-1.5 plus text-xs (22),
// and a result row is 42. The 8 DIP inset around a row and the room below the
// last one keep the rows off the panel's edges, as the reference's measured
// dialog does (a two-row panel is 174 DIPs high). mygo adds the panel's hairline
// to those boxes rather than drawing it inside them, so the bottom step is two
// DIPs shorter and the empty state two DIPs shorter than the measured face.
const (
	searchPanelW      = 448
	searchPanelRadius = 12
	searchInputH      = 56
	searchInputPadX   = 16
	searchInputGap    = 10
	searchGroupH      = 22
	searchGroupPadX   = 12
	searchRowH        = 42
	searchRowPadX     = 8
	searchRowGap      = 10
	searchResultIcon  = 32
	searchEmptyH      = 81
	searchBottomPad   = 10
	searchMaxResults  = 8
)

// searchScrim is the reference's bg-black/80 behind the panel: the window shows
// through dimmed, and a click on it closes the palette.
var searchScrim = ui.RGBA(0, 0, 0, 0.8)

// searchShadow is the reference's shadow-2xl under the panel: a wide, soft,
// downward shadow that lifts the dialog off the scrim.
var searchShadow = ui.RGBA(0, 0, 0, 0.45)

// searchChatIcon is the speech-bubble mark of a result row's tile, drawn in
// the same 16×16 outline frame as the sidebar's glyphs.
var searchChatIcon = sidebarIcon(`<path d="M2 4.2A1.7 1.7 0 0 1 3.7 2.5h8.6A1.7 1.7 0 0 1 14 4.2v5.6a1.7 1.7 0 0 1-1.7 1.7H7.6L4.3 14v-2.5H3.7A1.7 1.7 0 0 1 2 9.8Z"/>`)

// searchShortcutKey is the shortcut that opens the palette, and
// searchShortcutMods its modifier: Cmd+G on macOS, Ctrl+G elsewhere, as the
// reference's sidebar entry opens it.
const searchShortcutKey = ui.KeyG

var searchShortcutMods = ui.Cmd

// SearchResult is one row of the palette: the session the window would open,
// its title and the project label the catalog carries as Group. Every field is
// a display copy of a catalog row; the palette adds no session and no project.
type SearchResult struct {
	SessionID string
	Title     string
	Group     string
}

// SearchPaletteView is what the search palette draws: whether it is open, the
// text it edits, the results the shell filtered and the package's own
// selection.
type SearchPaletteView struct {
	// Open is the window's record that the palette is showing. The dialog
	// writes false here when Escape or a click on the scrim dismisses it, so a
	// nil pointer draws nothing.
	Open *bool
	// Query is the search text the input edits in place. The shell passes a
	// pointer, so the input's text survives the next frame while the results
	// are filtered from it. A nil pointer draws no palette.
	Query *string
	// Results are the catalog rows the query leaves, in the catalog's order and
	// already bounded by the shell. The palette reorders nothing and adds no
	// row; an empty list is the "no chats found" state, not a failed read.
	Results []SearchResult
	// Selected is the index of the highlighted result. The palette writes it
	// while the arrows move and before a result is opened; it is display state,
	// not a session choice, so the caller's selection changes only through
	// Actions.OnSelect.
	Selected *int
	// Locale is the language of the palette's own text: the placeholder, the
	// group heading and the empty state. Session titles and project labels are
	// catalog facts and stay as the caller sent them.
	Locale Locale
}

// searchShortcutHint is the key hint at a result's tail: Cmd+1…Cmd+8 on macOS,
// Ctrl+1…Ctrl+8 elsewhere, for the row at index i. The reference draws the
// modifier's own glyph on macOS, so the hint is a platform label rather than
// shell text the i18n table carries.
func searchShortcutHint(i int) string {
	key := strconv.Itoa(i + 1)
	if runtime.GOOS == "darwin" {
		return "⌘" + key
	}
	return "Ctrl+" + key
}

// searchDigitKeys is the direct-open row of results: the keys Cmd+1 to Cmd+8.
var searchDigitKeys = [searchMaxResults]ui.Key{
	ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8,
}

// searchResults is the display copy the palette draws: the catalog narrowed by
// query, in the caller's order, at most searchMaxResults rows. An empty or
// blank query leaves the catalog as it came, which is the reference's "recent
// chats" list. Title and Group are matched case-insensitively; the id is not
// searched, because the palette shows titles and projects, not identities.
func searchResults(sessions []Session, query string) []SearchResult {
	query = strings.ToLower(strings.TrimSpace(query))
	results := make([]SearchResult, 0, min(len(sessions), searchMaxResults))
	for _, session := range sessions {
		if query != "" && !searchSessionMatches(session, query) {
			continue
		}
		title := session.Title
		if title == "" {
			title = session.ID
		}
		results = append(results, SearchResult{SessionID: session.ID, Title: title, Group: session.Group})
		if len(results) == searchMaxResults {
			break
		}
	}
	return results
}

// searchSessionMatches reports whether the lower-cased query occurs in the
// session's title or its project label.
func searchSessionMatches(session Session, query string) bool {
	return strings.Contains(strings.ToLower(session.Title), query) ||
		strings.Contains(strings.ToLower(session.Group), query)
}

// moveSearchSelection moves the highlight by delta and keeps it inside the
// results. It stops at the first and last row rather than wrapping, so a longer
// key press never jumps from one end to the other.
func moveSearchSelection(v SearchPaletteView, delta int) {
	if v.Selected == nil || len(v.Results) == 0 {
		return
	}
	index := *v.Selected + delta
	index = max(0, min(index, len(v.Results)-1))
	*v.Selected = index
}

// openSearchResult asks the window to open the result at index and closes the
// palette. The highlight moves to the chosen row first, so the frame that
// follows shows which row was taken; the session itself changes only through
// Actions.OnSelect, so a caller without that callback keeps its selection.
func openSearchResult(v SearchPaletteView, a Actions, index int) {
	if v.Selected == nil || index < 0 || index >= len(v.Results) {
		return
	}
	*v.Selected = index
	selectSession(a, v.Results[index].SessionID)
	if v.Open != nil {
		*v.Open = false
	}
}

// searchPalette draws the centered palette over the window's content. The
// dialog's scrim takes the presses outside the panel, Escape and a click on the
// scrim close it, and the focus goes back to the control that opened it when it
// goes. The panel is the reference's: 448 DIPs wide, rounded 12, a hairline
// edge and the strong shadow, with the input over the grouped results.
func searchPalette(c *ui.Context, k tokens, v SearchPaletteView, a Actions) {
	if v.Open == nil || !*v.Open || v.Query == nil || v.Selected == nil {
		return
	}
	ui.DialogBase(c, v.Open, func(backdrop, panel ui.Element) {
		backdrop.Background(searchScrim)
		panel.Width(searchPanelW).MaxWidthPercent(90).Radius(searchPanelRadius).
			Background(k.background).BorderWidth(k.hairline).BorderColor(k.border).
			Shadow(0, 24, 48, 0, searchShadow).Clip()
		panel.Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				searchInput(c, k, v, a)
				if len(v.Results) == 0 {
					searchEmpty(c, k, v.Locale)
					return
				}
				ui.Column(c).FillWidth().Padding(0, 0, searchBottomPad, 0).Children(func() {
					searchGroupHeading(c, k, v.Locale)
					ui.Column(c).FillWidth().PaddingX(searchRowPadX).Children(func() {
						for i, result := range v.Results {
							searchResultRow(c, k, v, a, i, result)
						}
					})
				})
			})
		})
	})
}

// searchInput draws the reference's top row: the magnifier, the field and the
// hairline under it. The field takes the focus as the dialog opens, and its
// keys move the highlight, open a result or fall through to the dialog, which
// closes on Escape. Typing edits Query in place, so the next frame filters from
// the same text.
func searchInput(c *ui.Context, k tokens, v SearchPaletteView, a Actions) {
	placeholder := tr(v.Locale, keySearchPlaceholder)
	ui.Row(c).Height(searchInputH).FillWidth().PaddingX(searchInputPadX).
		Gap(searchInputGap).AlignItems(ui.Center).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).Children(func() {
		sidebarGlyph(c, sidebarIcons.search, k.mutedForeground)
		field := ui.TextInputBase(c, v.Query).Grow(1).Label(placeholder).
			Placeholder(placeholder).TextColor(k.foreground)
		field.OnTextKey(func(key ui.TextKey) bool {
			if key.Mods == ui.Cmd {
				for i, digit := range searchDigitKeys {
					if key.Key == digit {
						openSearchResult(v, a, i)
						return true
					}
				}
				return false
			}
			if key.Mods != 0 {
				return false
			}
			switch key.Key {
			case ui.KeyDown:
				moveSearchSelection(v, 1)
				return true
			case ui.KeyUp:
				moveSearchSelection(v, -1)
				return true
			case ui.KeyEnter:
				openSearchResult(v, a, *v.Selected)
				return true
			}
			// Escape and everything else fall through: the dialog on top takes
			// Escape and closes, as the reference does.
			return false
		})
	})
}

// searchGroupHeading names the results, as the reference's "最近聊天" does. The
// shell has no pin fact, so it draws no pinned group and never names one.
func searchGroupHeading(c *ui.Context, k tokens, locale Locale) {
	ui.Row(c).Height(searchGroupH).FillWidth().PaddingX(searchGroupPadX).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, tr(locale, keySearchRecent)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).FontWeight(500).SingleLine()
		ui.Spacer(c)
	})
}

// searchResultRow draws one session: its tile, its title over its project, and
// the shortcut that opens it. The highlighted row takes the reference's
// bg-muted face; the pointer's row takes the same face lighter. A click opens
// the row immediately, as the reference's rows do.
func searchResultRow(c *ui.Context, k tokens, v SearchPaletteView, a Actions, index int, result SearchResult) {
	selected := *v.Selected == index
	row := ui.ButtonBase(c).Key("search-result:" + result.SessionID).
		Height(searchRowH).FillWidth().PaddingX(searchRowPadX).Gap(searchRowGap).
		Radius(radiusMD).AlignItems(ui.Center).TabStop(false).
		Label(result.Title)
	switch {
	case selected:
		row.Background(k.muted)
	case row.Hovered():
		row.Background(k.muted.Alpha(0.7))
	}
	row.OnClick(func() {
		openSearchResult(v, a, index)
	})
	row.Children(func() {
		ui.Row(c).Size(searchResultIcon, searchResultIcon).Shrink(0).
			Radius(radiusMD).Background(k.muted).Justify(ui.Center).
			AlignItems(ui.Center).Children(func() {
			ui.Icon(c, searchChatIcon).Size(sidebarIconSize, sidebarIconSize).
				TextColor(k.mutedForeground).AccessHidden(true)
		})
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Children(func() {
			ui.Text(c, result.Title).FontSize(k.bodySize).TextColor(k.foreground).
				FontWeight(500).SingleLine()
			if result.Group != "" {
				ui.Text(c, result.Group).FontSize(k.smallSize).
					TextColor(k.mutedForeground).SingleLine()
			}
		})
		ui.Text(c, searchShortcutHint(index)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Shrink(0).SingleLine()
	})
}

// searchEmpty draws the reference's "未找到聊天" state: the panel keeps its
// input and shows the message where the results would be. It is the answer to a
// query, not a failed read, so it names no gap.
func searchEmpty(c *ui.Context, k tokens, locale Locale) {
	ui.Row(c).Height(searchEmptyH).FillWidth().PaddingX(searchInputPadX).
		Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Text(c, tr(locale, keySearchEmpty)).FontSize(k.bodySize).
			TextColor(k.mutedForeground).Role(ui.RoleStatus)
	})
}
