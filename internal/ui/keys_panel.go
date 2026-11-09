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
// (38.5 DIPs), the row padding is px-3 (10.5 DIPs), and a row is 28 DIPs high,
// py-1.5 plus text-sm's leading.

package ui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// The keys panel's geometry in DIPs. See the file comment for where each step
// comes from.
const (
	keysPanelHeaderH = 38.5 // h-11: the header
	keysPanelPadX    = 10.5 // px-3: the header, the search field and the rows
	keysPanelRowY    = 5.25 // py-1.5: above and below a row
	keysPanelRowH    = 28   // py-1.5 plus text-sm's leading
	keysPanelKeyW    = 128  // the key cell, wide enough for an ordinary key
	keysPanelSeqW    = 48   // the sequence cell, right aligned
	keysPanelInputW  = 120  // the input id cell
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

// keysPanelWIP draws the panel while the caller has no snapshot. It names the
// missing field and offers no filter or row, so nothing on the panel can look
// like a branch it was never given.
func keysPanelWIP(c *ui.Context, k tokens) {
	ui.Column(c).Padding(k.space(3)).Gap(k.gap).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Keys").Bold().FontSize(k.titleSize)
			ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
				Unselectable().AccessHidden(true)
		})
		ui.Text(c, reasonKeysUnloaded).FontSize(k.smallSize).TextColor(k.mutedForeground)
	})
}

// keysPanelHeader names the region, counts the entries the caller passed and
// shows the through of the snapshot. The count is derived from Entries and
// adds no key.
func keysPanelHeader(c *ui.Context, k tokens, v KeysPanelView) {
	ui.Row(c).Height(keysPanelHeaderH).PaddingX(keysPanelPadX).Gap(k.gap).
		AlignItems(ui.Center).BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).
		Children(func() {
			ui.Text(c, "Keys").Bold().FontSize(k.labelSize).SingleLine()
			ui.Text(c, keysPanelCount(len(v.Entries))).FontSize(k.smallSize).
				TextColor(k.mutedForeground).SingleLine()
			ui.Spacer(c)
			ui.Text(c, "through "+strconv.FormatInt(v.Through, 10)).
				FontSize(k.smallSize).TextColor(k.mutedForeground).SingleLine()
		})
}

// keysPanelCount is the header's count text.
func keysPanelCount(n int) string {
	if n == 1 {
		return "1 key"
	}
	return strconv.Itoa(n) + " keys"
}

// keysPanelSearch draws the local filter. It edits KeysPanelView.Query in
// place; the panel filters the display copy on the same frame the caller
// passes the new query back, so no read is repeated.
func keysPanelSearch(c *ui.Context, k tokens, v *KeysPanelView) {
	ui.Row(c).Padding(k.gap, keysPanelPadX, k.gap, keysPanelPadX).Children(func() {
		ui.SearchField(c, &v.Query).Label("Search keys").
			Placeholder("Filter keys").TextColor(k.foreground).FillWidth()
	})
}

// keysPanelEmpty draws the empty state of a loaded snapshot, or of a filter
// that leaves no row. The panel adds no invented entry to fill the room.
func keysPanelEmpty(c *ui.Context, k tokens, text string) {
	ui.Column(c).Grow(1).Padding(k.space(3), keysPanelPadX, k.space(3), keysPanelPadX).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, text).FontSize(k.bodySize).TextColor(k.mutedForeground)
	})
}

// keysPanelList draws one row per visible entry. The list is one Tab stop per
// row: Tab reaches the next row and Enter or Space selects it, as a button
// does. Up and Down are not bound, because a virtualized list's keys belong to
// its own state and this panel keeps none.
func keysPanelList(c *ui.Context, k tokens, v *KeysPanelView, entries []KeysPanelEntry) {
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(0).Role(ui.RoleList).Label("Keys").Children(func() {
			for _, entry := range entries {
				keysPanelRow(c, k, v, entry)
			}
		})
	})
}

// keysPanelRow draws one keyed block: its key, sequence, input id and a
// bounded text. The row's accessible name is the key, so assistive technology
// can address the row without reading the four columns.
func keysPanelRow(c *ui.Context, k tokens, v *KeysPanelView, entry KeysPanelEntry) {
	selected := v.selected(entry)
	row := ui.ButtonBase(c).Height(keysPanelRowH).FillWidth().
		PaddingY(keysPanelRowY).PaddingX(keysPanelPadX).Radius(radiusSM).
		Gap(k.gap).AlignItems(ui.Center).
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
		ui.Text(c, entry.Key).Bold().FontSize(k.codeFontSize).TextColor(k.foreground).
			Width(keysPanelKeyW).Shrink(0).SingleLine()
		ui.Text(c, strconv.FormatInt(entry.Sequence, 10)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Width(keysPanelSeqW).Shrink(0).
			TextAlign(ui.End).SingleLine()
		ui.Text(c, entry.InputID).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Width(keysPanelInputW).Shrink(0).SingleLine()
		ui.Text(c, keysPanelText(entry.Text)).FontSize(k.bodySize).
			TextColor(k.foreground).Grow(1).SingleLine()
	})
}
