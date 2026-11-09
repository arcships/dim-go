// This file owns the window's search palette: the centered dialog that Cmd+G
// and the sidebar's Search entry open, with an input, the recent-session
// results and the shortcut that opens one.
//
// The reference is docs/evidence/electron-reference/window-search.png and
// window-search-results.png: a 448 DIP panel rounded by 12, lifted by a strong
// shadow over a black 80% scrim, a 56 DIP input row with a magnifier and the
// "搜索聊天" placeholder that takes the focus as the dialog opens, then the
// "最近聊天" heading and 42 DIP rows. Each row is a 32 DIP icon tile that
// follows the status its caller reports, the session's title over its project,
// and the shortcut that opens it (Cmd+1 to Cmd+8 on macOS, Ctrl+1 to Ctrl+8
// elsewhere). The query matches a title, a project or a session id; the pointer
// moves the highlight to its row as it enters it. The panel fades and zooms
// over the reference's duration-200 as it opens and closes. Escape and a click
// on the scrim close it, and the focus returns to the control that opened it.
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
	"time"

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

// searchMotion is the reference dialog's duration-200: the panel fades in and
// out over 200 ms and zooms from 95%, and the scrim fades with it. It is the
// dialog's own step, not the 250 ms layout step (tokens.go) the reference gives
// panels that take room in the layout.
const searchMotion = 200 * time.Millisecond

// searchScrim is the reference's bg-black/80 behind the panel: the window shows
// through dimmed, and a click on it closes the palette.
var searchScrim = ui.RGBA(0, 0, 0, 0.8)

// searchShadow is the reference's shadow-2xl under the panel: a wide, soft,
// downward shadow that lifts the dialog off the scrim.
var searchShadow = ui.RGBA(0, 0, 0, 0.45)

// searchFade is the scrim's own motion: it starts and ends transparent, as the
// reference's overlay fade-in-0/fade-out-0 does. searchZoom is the panel's: it
// starts and ends at 95% of its size with no opacity, the reference's
// zoom-in-95/zoom-out-95 over the same 200 ms.
var (
	searchFade = ui.Motion{Opacity: 0}
	searchZoom = ui.Motion{Opacity: 0, Scale: 0.95}
)

// searchPaletteTransition is the dialog's enter and exit: 200 ms on the shell's
// layout curve, with the reduce-motion path the other motion helpers take. It
// is declared here because the reference gives this dialog duration-200 rather
// than the 250 ms panel step in tokens.go. The colors flag lets the panel follow
// a theme switch without animating its size as the results change.
func searchPaletteTransition(c *ui.Context, enter, exit ui.Motion) ui.ElementTransition {
	t := ui.ElementTransition{
		Duration: searchMotion,
		Ease:     motionEaseInOut,
		Colors:   true,
		Enter:    &enter,
		Exit:     &exit,
	}
	if c.Preferences().ReduceMotion {
		t.Duration = -1
		t.ExitDuration = -1
	}
	return t
}

// searchChatIcon is the speech-bubble mark of a result row's tile, drawn in
// the same 16×16 outline frame as the sidebar's glyphs.
var searchChatIcon = sidebarIcon(`<path d="M2 4.2A1.7 1.7 0 0 1 3.7 2.5h8.6A1.7 1.7 0 0 1 14 4.2v5.6a1.7 1.7 0 0 1-1.7 1.7H7.6L4.3 14v-2.5H3.7A1.7 1.7 0 0 1 2 9.8Z"/>`)

// searchErrorIcon is the tile's failed mark: the same 16×16 outline frame with
// an exclamation in a circle, so a row whose caller reported a failure reads
// apart from a chat still loading.
var searchErrorIcon = sidebarIcon(`<circle cx="8" cy="8" r="5.6"/><path d="M8 4.9v3.7"/><circle cx="8" cy="11.2" r=".8" fill="currentColor" stroke="none"/>`)

// searchShortcutKey is the shortcut that opens the palette, and
// searchShortcutMods its modifier: Cmd+G on macOS, Ctrl+G elsewhere, as the
// reference's sidebar entry opens it.
const searchShortcutKey = ui.KeyG

var searchShortcutMods = ui.Cmd

// SearchResultStatus is the state a result's icon tile draws: the ordinary
// chat mark, a spinner while the caller reports work in flight, or the failed
// mark. It is the caller's report, never the palette's inference: the catalog
// this shell holds carries no per-session run state, so searchResults leaves
// every row normal rather than inventing unread or pinned facts the shell does
// not have.
type SearchResultStatus uint8

const (
	// SearchResultNormal is a session with no reported work: the chat mark.
	SearchResultNormal SearchResultStatus = iota
	// SearchResultLoading is a session the caller reports as still working: a
	// spinner in the tile.
	SearchResultLoading
	// SearchResultError is a session the caller reports as failed: the failed
	// mark in the danger color.
	SearchResultError
)

// SearchResult is one row of the palette: the session the window would open,
// its title and the project label the catalog carries as Group. Every field is
// a display copy of a catalog row; the palette adds no session and no project.
type SearchResult struct {
	SessionID string
	Title     string
	Group     string
	// Status is the state the row's 32×32 tile draws. The zero value is the
	// ordinary chat mark; a caller with a verified loading or failed state may
	// set it, and the palette adds no state of its own.
	Status SearchResultStatus
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
// chats" list. Title, Group and SessionID are matched case-insensitively, so a
// pasted session id finds its chat as well as a title or project does.
func searchResults(sessions []Session, query string) []SearchResult {
	query = strings.ToLower(strings.TrimSpace(query))
	results := make([]SearchResult, 0, min(len(sessions), searchMaxResults))
	for _, session := range sessions {
		if query != "" && !searchSessionMatches(session, query) {
			continue
		}
		title := session.Title
		results = append(results, SearchResult{SessionID: session.ID, Title: title, Group: session.Group})
		if len(results) == searchMaxResults {
			break
		}
	}
	return results
}

// searchSessionMatches reports whether the lower-cased query occurs in the
// session's title, its project label or its id.
func searchSessionMatches(session Session, query string) bool {
	return strings.Contains(strings.ToLower(session.Title), query) ||
		strings.Contains(strings.ToLower(session.Group), query) ||
		strings.Contains(strings.ToLower(session.ID), query)
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

// searchPalette draws the centered palette over the window's content. It is
// an overlay of two children, both direct children of the window's overlay
// layer: the scrim, and a centered backdrop holding the panel. They are not
// DialogBase's backdrop-with-panel because the dialog has to keep a copy on
// the frame it closes: mygo keeps a copy of an element that goes with an Exit
// while its parent still is, and DialogBase's panel leaves with the backdrop
// that holds it. As siblings, the scrim fades and the backdrop fades and zooms
// with the panel it centers, each over the reference's 200 ms.
//
// The backdrop carries the modal scope and the dialog's ways out, as
// DialogBase's does: Escape and a click on it outside the panel set Open false,
// the focus enters the panel as it opens and returns to the control that opened
// it as it goes, and the panel absorbs a click inside it. The panel is the
// reference's: 448 DIPs wide up to 90% of a narrow window, rounded 12, a
// hairline edge and the strong shadow, with the input over the grouped results.
func searchPalette(c *ui.Context, k tokens, v SearchPaletteView, a Actions) {
	if v.Open == nil || !*v.Open || v.Query == nil || v.Selected == nil {
		return
	}
	ui.Overlay(c, func() {
		// The scrim is its own overlay child so it fades even though the panel's
		// backdrop is a sibling: a click lands on the backdrop above it.
		ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(searchScrim).AccessHidden(true).
			Transition(searchPaletteTransition(c, searchFade, searchFade))
		backdrop := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Center().Modal().Transition(searchPaletteTransition(c, searchZoom, searchZoom))
		if backdrop.Clicked() || backdrop.OverlayShortcut(0, ui.KeyEscape) {
			*v.Open = false
		}
		backdrop.Children(func() {
			panel := ui.Box(c).Role(ui.RoleDialog).Width(searchPanelW).MaxWidthPercent(90).
				Radius(searchPanelRadius).Background(k.background).
				BorderWidth(k.hairline).BorderColor(k.border).
				Shadow(0, 24, 48, 0, searchShadow).Clip()
			// The pointer's row moves the highlight when the pointer enters it,
			// not on every frame: with the pointer resting over a row, the arrow
			// keys still move the highlight between rows.
			hover := ui.Local(backdrop, searchHoverState{}, func() int { return -1 })
			// A click inside the panel is not a click on the scrim. DialogBase
			// marks its panel clickable for the same reason.
			panel.OnClick(func() {})
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
							searchResultRows(c, k, v, a, hover)
						})
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

// searchHoverState is the palette's own record of the row under the pointer,
// kept on the backdrop across frames with ui.Local. A pointer that stays over a
// row does not keep the highlight there, so the arrow keys still move it.
type searchHoverState struct{}

// searchResultRows builds the result rows and moves the highlight to a row the
// pointer just entered. All rows are built before the highlight moves, so one
// frame never paints two highlighted rows: the pointer's row and the row the
// keyboard had. When the pointer leaves, the last row it touched stays
// selected, so Enter opens what the user last pointed at.
func searchResultRows(c *ui.Context, k tokens, v SearchPaletteView, a Actions, hover *int) {
	rows := make([]ui.Element, len(v.Results))
	for i, result := range v.Results {
		rows[i] = searchResultRow(c, k, v, a, i, result)
	}
	if hover != nil {
		at := -1
		for i, row := range rows {
			if row.Hovered() {
				at = i
				break
			}
		}
		if at >= 0 && at != *hover {
			*v.Selected = at
		}
		*hover = at
	}
	for i, row := range rows {
		if i == *v.Selected {
			row.Background(k.muted)
		}
	}
}

// searchResultRow draws one session: its tile, its title over its project, and
// the shortcut that opens it. searchResultRows paints the highlighted row with
// the reference's bg-muted face. A click opens the row immediately, as the
// reference's rows do.
func searchResultRow(c *ui.Context, k tokens, v SearchPaletteView, a Actions, index int, result SearchResult) ui.Element {
	label := sessionLabel(result.Title, v.Locale)
	row := ui.ButtonBase(c).Key("search-result:" + result.SessionID).
		Height(searchRowH).FillWidth().PaddingX(searchRowPadX).Gap(searchRowGap).
		Radius(radiusMD).AlignItems(ui.Center).TabStop(false).
		Label(label)
	row.OnClick(func() {
		openSearchResult(v, a, index)
	})
	row.Children(func() {
		searchResultTile(c, k, result.Status)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Children(func() {
			ui.Text(c, label).FontSize(k.bodySize).TextColor(k.foreground).
				FontWeight(500).SingleLine()
			if result.Group != "" {
				ui.Text(c, result.Group).FontSize(k.smallSize).
					TextColor(k.mutedForeground).SingleLine()
			}
		})
		ui.Text(c, searchShortcutHint(index)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Shrink(0).SingleLine()
	})
	return row
}

// searchResultTile draws the reference's 32×32 icon block by the row's status:
// the ordinary chat mark, a spinner for a row the caller reports as working, or
// the failed mark on the danger wash. The tile is decoration; the row's label
// names the chat, so nothing here is announced twice.
func searchResultTile(c *ui.Context, k tokens, status SearchResultStatus) {
	tile := ui.Row(c).Size(searchResultIcon, searchResultIcon).Shrink(0).
		Radius(radiusMD).Background(k.muted).Justify(ui.Center).
		AlignItems(ui.Center).AccessHidden(true)
	switch status {
	case SearchResultLoading:
		tile.Children(func() {
			ui.Spinner(c).AccessHidden(true)
		})
	case SearchResultError:
		tile.Background(k.wash(k.danger)).Children(func() {
			ui.Icon(c, searchErrorIcon).Size(sidebarIconSize, sidebarIconSize).
				TextColor(k.danger)
		})
	default:
		tile.Children(func() {
			ui.Icon(c, searchChatIcon).Size(sidebarIconSize, sidebarIconSize).
				TextColor(k.mutedForeground)
		})
	}
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
