// This file owns the session keys panel: a read-only browser for the
// session.keys snapshot the App Server returned when the session opened. The
// snapshot is the latest live keyed block per key on the branch, and the panel
// draws it as a list, with a local filter over the entries the caller passed.
//
// The panel reads KeysPanelView and Actions only. Every entry is a display
// copy of what the caller read; the panel derives no key fact, sends no App
// Server request, and writes nothing except the selected key through
// KeysPanelView.Selected and the filter text through KeysPanelView.Query.
// Both are view state the caller owns, like the sidebar's selection: a click
// marks a row, it never changes a session.
//
// The shell is the caller: State.Keys carries the snapshot and
// State.ShowKeys says whether the browser takes the main column, and
// protocols.go's renderKeysPanel is the frozen entry. The panel has no
// callback of its own, because session.keys is read once, when the session
// opens, and the snapshot is what the caller supplies.
//
// Geometry: docs/ui-layout-spec.md does not measure this panel, so the numbers
// below are this component's own. They follow the context panel's steps, which
// are converted from dim-electron's 14px root font size: the header is h-11
// (38.5 DIPs) and the row padding is px-3 (10.5 DIPs).
//
// A row is a section rather than one line: its key heading and the note that
// names where the block came from sit on the first line, and the value takes
// the second. The two lines are 18 DIPs each, with a 2 DIP gap and 6 DIPs of
// padding above and below, so a row is 50 DIPs high: tall enough to read the
// key and its value apart, short enough that a browser of many keys keeps a
// useful density. The value is monospaced, because a keyed block is a code
// body, not prose; the key is monospaced and bold so it reads as the heading.
//
// Narrow windows: no column has a fixed width. The key heading grows and the
// note, the value and the header's texts each shrink to their room, so a narrow
// main column clips them with an ellipsis instead of pushing the panel past the
// window's edge. The panel keeps no width of its own to switch columns on: it
// draws the same three parts at every width and lets the layout bound them.

package ui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// The keys panel's geometry in DIPs. See the file comment for where each step
// comes from.
const (
	keysPanelHeaderH    = 38.5 // h-11: the header
	keysPanelPadX       = 10.5 // px-3: the header, the search field and the rows
	keysPanelRowY       = 6    // above and below a row's two lines
	keysPanelRowH       = 50   // keysPanelRowY, two 18 DIP lines and their 2 DIP gap
	keysPanelLineH      = 18   // one line of a row: the heading and the value
	keysPanelLineGap    = 2    // between a row's heading and its value
	keysPanelSearchPadY = 6    // above and below the search field
)

// keysPanelTextRunes bounds the block text a row shows. A keyed block can be
// the whole body of an input, and a list row is one line; the rest is cut with
// an ellipsis rather than wrapped, so every row keeps its place. The same
// input always gives the same text, which a headless fixture checks.
const keysPanelTextRunes = 80

// The reasons the panel shows while the snapshot it needs is missing. The
// reason names the view field that was not filled, so the WIP badge explains
// itself instead of an empty list appearing to be a fact.
const reasonKeysUnloaded = "Unavailable: KeysPanelView.Loaded is false; the window has not passed the session.keys snapshot to the keys panel yet."

// KeysPanelEntry is one keyed block as the panel draws it: a display copy of
// the App Server's SessionKeyEntry. Key is the key's name on the branch,
// Sequence is the sequence of the block the snapshot kept, InputID is the
// input that wrote it, and Text is the block body.
type KeysPanelEntry struct {
	Key      string
	Sequence int64
	InputID  string
	Text     string
}

// KeysPanelView is what the keys panel draws: the session.keys snapshot, the
// through the caller requested, the selected key and the local filter.
//
// An empty Loaded view is not "no keys": Loaded tells the panel whether the
// caller read the snapshot at all, so a missing read draws a WIP statement
// instead of "No keys yet".
type KeysPanelView struct {
	// Entries is the snapshot's entries in the App Server's order. The panel
	// reorders nothing and adds no key. The caller copies them from the
	// session.keys reply; Entries is nil when the read failed.
	Entries []KeysPanelEntry
	// Through is the sequence the snapshot was read through, as the caller
	// requested it. It is drawn only with Loaded, so it never stands in for a
	// snapshot that was not read.
	Through int64
	// Loaded says the caller has a session.keys snapshot for the session and
	// branch it is showing. Without it the panel cannot tell a branch with no
	// keys from a read it never received, and the whole panel reads as WIP.
	Loaded bool
	// Selected is the selected key, or nil while the caller keeps no
	// selection. A click or Enter on a row writes that row's key here; it is
	// the only session-shaped field the panel writes, and it is display state,
	// not a key fact.
	Selected *string
	// Query is the local filter text. The panel writes it while the user types
	// in the search field; the caller passes it back on the next frame. The
	// filter is applied to the display copy of Entries and never asks the App
	// Server for a different window.
	Query string
}

// selected reports whether entry is the caller's selection. A nil Selected
// means the caller keeps no selection and no row is marked.
func (v KeysPanelView) selected(entry KeysPanelEntry) bool {
	return v.Selected != nil && entry.Key != "" && *v.Selected == entry.Key
}

// choose records a click or Enter on a row. A nil Selected is a caller that
// keeps no selection; the click then changes nothing rather than inventing a
// field for it.
func (v *KeysPanelView) choose(entry KeysPanelEntry) {
	if v.Selected != nil && entry.Key != "" {
		*v.Selected = entry.Key
	}
}

// visible returns the entries the filter leaves, in the caller's order. It
// filters a display copy and never mutates Entries or asks for another read.
// An empty or blank Query matches every entry.
func (v KeysPanelView) visible() []KeysPanelEntry {
	query := strings.ToLower(strings.TrimSpace(v.Query))
	if query == "" {
		return v.Entries
	}
	out := make([]KeysPanelEntry, 0, len(v.Entries))
	for _, entry := range v.Entries {
		if keysPanelMatches(entry, query) {
			out = append(out, entry)
		}
	}
	return out
}

// keysPanelMatches reports whether the lower-cased query occurs in the entry's
// key, input id or text. Sequence numbers are not searched: they are positions,
// not names.
func keysPanelMatches(entry KeysPanelEntry, query string) bool {
	return strings.Contains(strings.ToLower(entry.Key), query) ||
		strings.Contains(strings.ToLower(entry.InputID), query) ||
		strings.Contains(strings.ToLower(entry.Text), query)
}

// keysPanelText makes a block body a single bounded line: every run of
// whitespace becomes one space, so a multi-line block stays one row, and text
// past keysPanelTextRunes runes ends in an ellipsis.
func keysPanelText(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if len(runes) <= keysPanelTextRunes {
		return s
	}
	return string(runes[:keysPanelTextRunes]) + "…"
}

// keysPanelNote is a row's note: which block the snapshot kept. It names the
// input that wrote the block and the block's sequence, in that order, and the
// sequence alone when the block names no input. The note derives nothing new:
// both fields are the caller's, and the panel only joins them for display.
func keysPanelNote(entry KeysPanelEntry) string {
	note := "seq " + strconv.FormatInt(entry.Sequence, 10)
	if entry.InputID != "" {
		note = entry.InputID + " · " + note
	}
	return note
}

// keysPanel draws the session keys browser: its title and through, the local
// filter, and one row per key the filter leaves. It writes
// KeysPanelView.Query while the filter is edited and KeysPanelView.Selected on
// a row click or Enter; it calls no Actions field, because the snapshot is a
// read and the panel starts no request.
//
// v is a pointer because the search field edits Query in place, so the filter
// survives the next frame without the panel keeping a second copy. With
// Loaded false the whole panel is a WIP statement, not an empty list: the
// panel cannot tell a branch with no keys from a read it never received.
func keysPanel(c *ui.Context, k tokens, v *KeysPanelView, a Actions) {
	// The region contract keeps Actions in every entry, but there is no key
	// request to send: session.keys is read once, when the session opens.
	_ = a

	ui.Column(c).FillHeight().FillWidth().Clip().Background(k.background).
		BorderStartWidth(k.hairline).BorderColor(k.border).
		Role(ui.RoleRegion).Label("Keys panel").Children(func() {
		if !v.Loaded {
			keysPanelWIP(c, k)
			return
		}
		keysPanelHeader(c, k, *v)
		keysPanelSearch(c, k, v)
		entries := v.visible()
		switch {
		case len(entries) > 0:
			keysPanelList(c, k, v, entries)
		case len(v.Entries) == 0:
			keysPanelEmpty(c, k, "No keys yet")
		default:
			keysPanelEmpty(c, k, "No keys match "+strconv.Quote(strings.TrimSpace(v.Query))+".")
		}
	})
}

// keysPanelWIP draws the panel while the caller has no snapshot. It keeps the
// header's frame and names the missing field, and offers no filter or row, so
// nothing on the panel can look like a branch it was never given.
func keysPanelWIP(c *ui.Context, k tokens) {
	ui.Column(c).Children(func() {
		ui.Row(c).Height(keysPanelHeaderH).PaddingX(keysPanelPadX).Gap(k.gap).
			AlignItems(ui.Center).BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).
			Children(func() {
				ui.Text(c, "Keys").Bold().FontSize(k.labelSize).SingleLine().Shrink(0)
				ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
					Unselectable().AccessHidden(true).Shrink(0)
				ui.Spacer(c)
			})
		ui.Column(c).Padding(k.space(3), keysPanelPadX, k.space(3), keysPanelPadX).
			Gap(k.gap).Children(func() {
			ui.Text(c, reasonKeysUnloaded).FontSize(k.smallSize).
				TextColor(k.mutedForeground)
		})
	})
}

// keysPanelHeader names the region, counts the entries the caller passed and
// shows the through of the snapshot. The count is derived from Entries and
// adds no key. The header is one line like a table's: nothing in it grows, so
// at a narrow width its texts clip in place rather than wrapping.
func keysPanelHeader(c *ui.Context, k tokens, v KeysPanelView) {
	ui.Row(c).Height(keysPanelHeaderH).PaddingX(keysPanelPadX).Gap(k.gap).
		AlignItems(ui.Center).BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).
		Children(func() {
			ui.Text(c, "Keys").Bold().FontSize(k.labelSize).SingleLine().Shrink(0)
			ui.Text(c, keysPanelCount(len(v.Entries))).FontSize(k.smallSize).
				TextColor(k.mutedForeground).SingleLine().Shrink(1).MinWidth(0)
			ui.Spacer(c)
			ui.Text(c, "through "+strconv.FormatInt(v.Through, 10)).
				FontSize(k.smallSize).TextColor(k.mutedForeground).SingleLine().
				Shrink(1).MinWidth(0).TextAlign(ui.End)
		})
}

// keysPanelCount is the header's count text.
func keysPanelCount(n int) string {
	if n == 1 {
		return "1 key"
	}
	return strconv.Itoa(n) + " keys"
}

// keysPanelSearch draws the local filter over its own rule, so the list below
// it reads as the panel's body rather than as another row. It edits
// KeysPanelView.Query in place; the panel filters the display copy on the same
// frame the caller passes the new query back, so no read is repeated.
func keysPanelSearch(c *ui.Context, k tokens, v *KeysPanelView) {
	ui.Row(c).Padding(keysPanelSearchPadY, keysPanelPadX, keysPanelSearchPadY, keysPanelPadX).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).Children(func() {
		ui.SearchField(c, &v.Query).Label("Search keys").
			Placeholder("Filter keys").TextColor(k.foreground).FillWidth()
	})
}

// keysPanelEmpty draws the empty state of a loaded snapshot, or of a filter
// that leaves no row, centered in the room the list would take. The panel adds
// no invented entry to fill the room.
func keysPanelEmpty(c *ui.Context, k tokens, text string) {
	ui.Column(c).Grow(1).Padding(k.space(3), keysPanelPadX, k.space(3), keysPanelPadX).
		Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Text(c, text).FontSize(k.bodySize).TextColor(k.mutedForeground).
			TextAlign(ui.Center)
	})
}

// keysPanelList draws one row per visible entry in one scroll container. The
// list is one Tab stop per row: Tab reaches the next row and Enter or Space
// selects it, as a button does. The container around the focus takes the
// scrolling keys, so Page Up and Page Down (and the arrows) move the list while
// a row holds the keyboard; the panel binds no list keys of its own, because a
// virtualized list's keys belong to its own state and this panel keeps none.
func keysPanelList(c *ui.Context, k tokens, v *KeysPanelView, entries []KeysPanelEntry) {
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(0).Role(ui.RoleList).Label("Keys").Children(func() {
			for _, entry := range entries {
				keysPanelRow(c, k, v, entry)
			}
		})
	})
}

// keysPanelRow draws one keyed block as a section. The first line is the key
// heading, monospaced and bold, with the note that names the block the
// snapshot kept at the end; the second line is the block's value, monospaced
// because it is a code body. Both lines are single lines: a long key, note or
// value ends in an ellipsis, so a narrow panel clips its own text instead of
// widening the row. The row's accessible name is the key, so assistive
// technology can address the row without reading its three parts.
func keysPanelRow(c *ui.Context, k tokens, v *KeysPanelView, entry KeysPanelEntry) {
	selected := v.selected(entry)
	row := ui.ButtonBase(c).Key("keys-row:" + entry.Key).Height(keysPanelRowH).
		FillWidth().PaddingY(keysPanelRowY).PaddingX(keysPanelPadX).Radius(k.radiusSM).
		AlignItems(ui.Center).
		Role(ui.RoleListItem).Label(entry.Key).Selected(selected)
	switch {
	case selected:
		row.Background(k.accent)
	case row.Hovered():
		row.Background(k.sidebarHover)
	}
	row.OnClick(func() {
		v.choose(entry)
	})
	row.Children(func() {
		ui.Column(c).FillWidth().Gap(keysPanelLineGap).Justify(ui.Center).Children(func() {
			ui.Row(c).FillWidth().Gap(k.gap).AlignItems(ui.Center).Children(func() {
				ui.Text(c, entry.Key).Font(monospaceFamily).Bold().
					FontSize(k.codeFontSize).TextColor(k.foreground).
					FixedLineHeight(keysPanelLineH).SingleLine().
					Grow(1).Shrink(1).MinWidth(0)
				ui.Text(c, keysPanelNote(entry)).FontSize(k.smallSize).
					TextColor(k.mutedForeground).FixedLineHeight(keysPanelLineH).
					SingleLine().Shrink(1).MinWidth(0).TextAlign(ui.End)
			})
			ui.Text(c, keysPanelText(entry.Text)).Font(monospaceFamily).
				FontSize(k.bodySize).TextColor(k.foreground).
				FixedLineHeight(keysPanelLineH).SingleLine().
				Grow(1).Shrink(1).MinWidth(0)
		})
	})
}
