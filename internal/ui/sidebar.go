// This file owns the session sidebar: the global entries, the grouped thread
// list and the account footer. It reads SidebarView and Actions only (see
// protocols.go).
//
// The reference has no wordmark, no branch directory, no thread-action row and
// no capability cards in this column: the entries begin below the traffic
// lights, the session and workspace headings follow them, the catalog sits
// under the workspace heading, and the account row sits at the bottom. Branch
// and delete callbacks stay in State and Actions for the session header menu
// that will carry them, but this column draws neither today.

package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// ungroupedLabel heads the thread rows the caller gave no group, when other
// rows do carry one. It names the absence of a group; it is not a project.
const ungroupedLabel = "No project"

// sidebarEmptyIndent is the reference's pl-[26px] on the built-in workspace's
// "No sessions yet" row: 26 DIPs from the column's 7 DIP content edge, which
// aligns the message with the project headings' text rather than their icons.
// It is the reference's literal, not a spacing step.
const sidebarEmptyIndent = 26

// sidebarRowInset and sidebarRowEndInset are the room a session row's face
// adds on top of the list's own 7 DIP padding, in DIPs. window-conversation.png
// draws the selected row's fill from x≈11.5 to 248.5: 11.5 from the column's
// left edge and 7.5 from its right one, the right side 4 DIPs wider because the
// reference's scroll area keeps a gutter there. The row's title and relative
// stamp move with the face; the project headings keep the list's own edge, as
// the reference draws them.
const (
	sidebarRowInset    = 4.5
	sidebarRowEndInset = 0.5
)

// sidebarEntriesTop is the room above the first global entry, in multiples of
// the sidebar's own 3.5 DIP step. In window-workspace.png the first entry's
// label ink sits at y≈67-78.5 (center 72.8), while the same 28 DIP row drawn at
// the 48 DIP header's edge centres its label at 62; three steps put the entry
// where the reference draws it and carry the session heading below within a DIP.
// The extra room belongs to the entries block, not the header: the header still
// reserves the 48 DIPs the traffic lights float over.
const sidebarEntriesTop = 3

// sidebarEmptyPadTop and sidebarEmptyPadBottom are the room above the built-in
// workspace's message and below it before the section's hairline, in multiples
// of the sidebar's 3.5 DIP step. window-conversation.png measures the message's
// ink at y≈267.5-277 (center 272), under the "Workspace" heading's center at
// 237, and the hairline closing the section at y≈295.5. The message keeps 14
// DIPs over its line and 17.5 below it, so the heading, the message and the
// hairline stand where the reference draws them instead of on the 2 DIP gap
// the section used to keep.
const (
	sidebarEmptyPadTop    = 4
	sidebarEmptyPadBottom = 5
)

const (
	// mainBranchID is the branch the App Server always keeps: it can be shown,
	// but not ended.
	mainBranchID = "main"
	// branchStateEnded is session.branches' terminal state. An ended branch
	// offers no end entry, because the App Server already closed it.
	branchStateEnded = "ended"

	reasonEndBranch       = "Unavailable: the shell has no session.branch_end callback yet."
	reasonRefreshBranches = "Unavailable: the shell has no session.branches refresh callback yet."
)

// sidebarRow is one row of the sidebar's thread list: a group heading or a
// session. Every field is derived from SidebarView.Sessions each frame, so the
// row holds no fact of its own and cannot outlive the catalog it came from.
type sidebarRow struct {
	// group marks a heading row, which the list shows sticky and never
	// chooses.
	group bool
	// label is the heading's text.
	label string
	// sessionID identifies a session row, and keys the list.
	sessionID string
	// updatedAt is the row's catalog time in epoch milliseconds; zero draws
	// no time.
	updatedAt int64
	// title is the session row's display text; empty when the catalog has
	// none.
	title string
}

// accessLabel is what assistive technology reads for the row: the heading's
// text, or the session's title falling back to its ID.
func (r sidebarRow) accessLabel() string {
	if r.group {
		return r.label
	}
	if r.title != "" {
		return r.title
	}
	return r.sessionID
}

// threadRows lays the catalog out for the sidebar. When no session carries a
// Group label the list stays flat; once one does, every heading is drawn and
// the ungrouped sessions fall under the locale's "no project" label. The
// catalog's order is kept inside each heading, and no row is invented.
func threadRows(sessions []Session, locale Locale) []sidebarRow {
	grouped := false
	for _, session := range sessions {
		if session.Group != "" {
			grouped = true
			break
		}
	}
	rows := make([]sidebarRow, 0, len(sessions))
	if !grouped {
		for _, session := range sessions {
			rows = append(rows, sidebarRow{sessionID: session.ID, title: session.Title, updatedAt: session.UpdatedAt})
		}
		return rows
	}
	var order []string
	buckets := make(map[string][]sidebarRow, len(sessions))
	for _, session := range sessions {
		label := session.Group
		if label == "" {
			label = tr(locale, keySidebarUngrouped)
		}
		if _, ok := buckets[label]; !ok {
			order = append(order, label)
		}
		buckets[label] = append(buckets[label], sidebarRow{sessionID: session.ID, title: session.Title, updatedAt: session.UpdatedAt})
	}
	for _, label := range order {
		rows = append(rows, sidebarRow{group: true, label: label})
		rows = append(rows, buckets[label]...)
	}
	return rows
}

// relativeStamp is the sidebar row's relative time, as the reference shows it
// ("5m", "2h", "3d"). The value is session.catalog's updatedAt: epoch
// milliseconds when it looks like one, epoch seconds otherwise; zero or a
// future time draws nothing rather than a made-up stamp.
func relativeStamp(updatedAt int64, now time.Time) string {
	if updatedAt <= 0 {
		return ""
	}
	stamp := updatedAt
	if stamp < 1_000_000_000_000 {
		stamp *= 1000
	}
	delta := now.Sub(time.UnixMilli(stamp))
	switch {
	case delta < 0:
		return ""
	case delta < time.Minute:
		return "now"
	case delta < time.Hour:
		return strconv.Itoa(int(delta/time.Minute)) + "m"
	case delta < 24*time.Hour:
		return strconv.Itoa(int(delta/time.Hour)) + "h"
	default:
		return strconv.Itoa(int(delta/(24*time.Hour))) + "d"
	}
}

// rowIndex returns the index of the session's row, or -1 when the catalog no
// longer holds it.
func rowIndex(rows []sidebarRow, sessionID string) int {
	for i, row := range rows {
		if !row.group && row.sessionID == sessionID {
			return i
		}
	}
	return -1
}

// searchShortcutKey is the shortcut that opens the search box, and
// searchShortcutMods its modifier: Cmd+G on macOS, Ctrl+G elsewhere. The
// reference's search row shows no hint for it, so the shortcut stays a
// behavior only and the row stays a plain label.
const searchShortcutKey = ui.KeyG

var searchShortcutMods = ui.Cmd

// filterSessions returns the sessions the query leaves, in the catalog's
// order. It filters a display copy: the catalog itself is never written and no
// read is requested. An empty or blank query matches every session, and the
// title and the id are both searched, case-insensitively: a session whose
// title is unknown is still found by its id.
func filterSessions(sessions []Session, query string) []Session {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return sessions
	}
	visible := make([]Session, 0, len(sessions))
	for _, session := range sessions {
		if sessionMatches(session, query) {
			visible = append(visible, session)
		}
	}
	return visible
}

// sessionMatches reports whether the lower-cased query occurs in the session's
// title or id. Group labels are not searched: they head rows, they are not
// sessions.
func sessionMatches(session Session, query string) bool {
	return strings.Contains(strings.ToLower(session.Title), query) ||
		strings.Contains(strings.ToLower(session.ID), query)
}

// adoptSearch takes the caller's SidebarView.Searching on the first frame the
// sidebar builds, as the shell takes the caller's selection: the shell copies
// State into a fresh SidebarView every frame, so the search box's open state
// lives in the region's own sidebarState. After that the region owns it,
// because a click, the shortcut and Escape all toggle it.
func (v SidebarView) adoptSearch() {
	if v.state.searchTaken {
		return
	}
	v.state.searchTaken = true
	v.state.searching = v.Searching
}

// searchQuery is the storage the search field edits. A caller that keeps the
// text itself passes SearchQuery; otherwise the region keeps it in its own
// state, which is how the window runs it while it has no field for the text.
func (v SidebarView) searchQuery() *string {
	if v.SearchQuery != nil {
		return v.SearchQuery
	}
	return &v.state.query
}

// visibleSessions is the display copy the thread list lays out: the catalog
// filtered by the open search box. It reorders nothing and adds no session.
func (v SidebarView) visibleSessions() []Session {
	if !v.state.searching {
		return v.Sessions
	}
	return filterSessions(v.Sessions, *v.searchQuery())
}

// toggleSearch opens the search box, or closes it and clears its filter.
func (v SidebarView) toggleSearch() {
	if v.state.searching {
		v.closeSearch()
		return
	}
	v.state.searching = true
}

// closeSearch closes the box and clears the filter, so reopening it starts
// from the whole catalog rather than the last query.
func (v SidebarView) closeSearch() {
	v.state.searching = false
	*v.searchQuery() = ""
}

// sidebar draws the reference's sidebar: the global entries, the grouped thread
// list and the account footer. Its width comes from the split divider around
// it; the shell clamps it to the sidebar's own bounds before every frame.
func sidebar(c *ui.Context, k tokens, v SidebarView, a Actions) {
	v.adoptSearch()
	locale := v.Locale
	// The search box is the sidebar's own display state, so its shortcut and
	// Escape are read here: the shell has no callback for either, and a
	// filter needs none. Both write the view state and send nothing.
	if c.Shortcut(searchShortcutMods, searchShortcutKey) {
		v.toggleSearch()
	}
	if v.state.searching && c.Shortcut(0, ui.KeyEscape) {
		v.closeSearch()
	}
	sessions := v.visibleSessions()
	ui.Column(c).FillHeight().Clip().Background(k.sidebar).
		Transition(motionLayoutColors(c)).
		Role(ui.RoleNavigation).Label(tr(locale, keySidebarNavigation)).Children(func() {
		sidebarEntries(c, k, v, a)
		threadListHeader(c, k, locale)
		workspaceHeader(c, k, locale)
		threadList(c, k, v, sessions, a)
		sidebarAccount(c, k, v, a)
	})
}

// sidebarEntries are the sidebar's global entries, in dim-electron's order:
// the session action, search, scheduled tasks and skills/plugins. The column
// starts below the traffic lights, which float over it in the reference's
// hidden-title-bar window; the wordmark and the update entry are not drawn.
// Search is local: the entry expands a filter box over the catalog the sidebar
// already draws, and nothing it does reaches the App Server. The two
// management entries have no client or page in this shell yet, so they keep
// their place as unavailable entries whose tooltip carries the reason.
func sidebarEntries(c *ui.Context, k tokens, v SidebarView, a Actions) {
	ui.Column(c).Padding(k.sidebarHeaderH+sidebarEntriesTop*k.sidebarGap, k.sidebarPadX, k.sidebarPadX, k.sidebarPadX).
		Gap(k.sidebarGap).Children(func() {
		newSessionEntry(c, k, v, a)
		searchEntry(c, k, v)
		scheduledEntry(c, k, v)
		skillsEntry(c, k, v)
	})
}

// scheduledEntry offers dim-electron's scheduled-task page. The shell has no
// scheduled-task client, no page and no App Server method to read one, so the
// entry stays visible, unavailable and explained rather than opening an empty
// page.
func scheduledEntry(c *ui.Context, k tokens, v SidebarView) {
	locale := v.Locale
	sidebarEntry(c, k, sidebarIcons.clock, tr(locale, keySidebarScheduled), reasonScheduled, false, false, false, nil)
}

// skillsEntry offers dim-electron's skills and plugins page. The shell has no
// skills or plugins client, no marketplace page and no App Server method to
// read either, so the entry stays visible, unavailable and explained.
func skillsEntry(c *ui.Context, k tokens, v SidebarView) {
	locale := v.Locale
	sidebarEntry(c, k, sidebarIcons.grid, tr(locale, keySidebarSkills), reasonSkills, false, false, false, nil)
}

// searchEntry toggles the sidebar's local session search. It needs no caller
// callback, because filtering the display copy sends nothing: the entry is real
// while the shell has a catalog at all. The box under it edits the query in
// place, and the thread list below narrows on the same frame.
func searchEntry(c *ui.Context, k tokens, v SidebarView) {
	locale := v.Locale
	sidebarEntry(c, k, sidebarIcons.search, tr(locale, keySidebarSearch), "", true, false, v.state.searching, func() {
		v.toggleSearch()
	})
	if !v.state.searching {
		return
	}
	ui.SearchField(c, v.searchQuery()).Label(tr(locale, keySidebarSearchLabel)).
		Placeholder(tr(locale, keySidebarSearchHint)).TextColor(k.sidebarForeground).FillWidth()
}

// newSessionEntry offers session.create once the window wires it. Without the
// callback the entry stays visible and unavailable rather than disappearing,
// and it is unavailable again while one creation is in flight, so a second
// click cannot ask for a second session.
func newSessionEntry(c *ui.Context, k tokens, v SidebarView, a Actions) {
	locale := v.Locale
	if a.OnCreateSession == nil {
		sidebarEntry(c, k, sidebarIcons.plus, tr(locale, keySidebarNewSession), tr(locale, keyReasonCreateSession), false, false, false, nil)
		return
	}
	sidebarEntry(c, k, sidebarIcons.plus, tr(locale, keySidebarNewSession), "", true, v.Creating, false, func() {
		createSession(a, v.Creating)
	})
}

// reasonKeysEntry is what the keys browser said while the window had no
// snapshot to browse. The entry is gone from the sidebar, as the reference has
// none; the browser stays in the main column for a caller that sets ShowKeys.
const reasonKeysEntry = "Unavailable: the shell has no key browser callback while no session.keys snapshot is loaded."

// sidebarEntry draws one global entry as a menu row: its icon and its label,
// as the reference's clean rows have no trailing shortcut hint. An unavailable
// entry registers no click and carries its reason in the description and the
// tooltip instead of a visible badge. selected marks an entry that is showing
// its surface. The glyph is decorative: the label is the entry's accessible
// name.
func sidebarEntry(c *ui.Context, k tokens, icon *ui.SVG, label, reason string, enabled, disabled, selected bool, on func()) {
	row := ui.ButtonBase(c).Key("sidebar-entry:" + label).Height(k.sidebarEntryH).FillWidth().
		PaddingX(k.sidebarPadX).Radius(radiusMD).Gap(k.sidebarGap).AlignItems(ui.Center).
		Transition(motionFastColors(c))
	row.Label(label)
	if enabled {
		row.OnClick(on)
		row.Disabled(disabled)
		// The entry expands its surface in the main column, as a disclosure
		// button does; assistive technology reads it as expanded, not selected.
		row.Expanded(selected)
	} else {
		row.AccessDisabled(true).Description(reason).Tooltip(reason).Opacity(wipOpacity)
	}
	if row.Hovered() {
		row.Background(k.sidebarHover)
	}
	row.Children(func() {
		sidebarGlyph(c, icon, k.sidebarForeground)
		ui.Text(c, label).FontSize(k.bodySize).TextColor(k.sidebarForeground).SingleLine()
		ui.Spacer(c)
	})
}

// threadListHeader names the thread list, as the reference's session column
// does. The count, the unavailable group control and the heading's add and list
// decorations are gone: the decorations took no click, and the add mark repeated
// the new-session entry above. The heading is text alone.
func threadListHeader(c *ui.Context, k tokens, locale Locale) {
	ui.Row(c).Height(k.sidebarSectionH).PaddingX(k.sidebarSectionPadX).Gap(k.sidebarGap).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, tr(locale, keySidebarSessions)).FontSize(k.smallSize).TextColor(k.mutedForeground).Bold().SingleLine()
		ui.Spacer(c)
	})
}

// workspaceHeader names the workspace the session rows belong to, the
// reference's second heading: it follows the session heading whether the column
// holds rows or not, and the project groups or the empty message sit under it.
// It is a display heading, not a control or a project fact: the shell has no
// workspace catalog, so it draws the interface language's word for the group.
func workspaceHeader(c *ui.Context, k tokens, locale Locale) {
	ui.Row(c).Height(k.sidebarSectionH).PaddingX(k.sidebarSectionPadX).Gap(k.sidebarGap).
		AlignItems(ui.Center).Children(func() {
		sidebarGlyph(c, sidebarIcons.house, k.mutedForeground)
		ui.Text(c, tr(locale, keyComposerWorkspace)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Bold().SingleLine()
		ui.Spacer(c)
	})
}

// threadList draws the workspace section and, below it, the grouped thread
// rows in a keyed, virtualized list with one scroll container. The built-in
// workspace's message appears only while the list has no row, and the hairline
// closing the section stays under both states. The list's choice is the shell's
// selected session: a click or an arrow key moves the choice, and chooseSession
// publishes it. sessions is the display copy the search filter left; the
// catalog itself is never written.
func threadList(c *ui.Context, k tokens, v SidebarView, sessions []Session, a Actions) {
	locale := v.Locale
	rows := threadRows(sessions, locale)
	v.state.rows = rows
	// The choice maps from the selection anew when the selection or the filter
	// changed, so a row the filter hides cannot keep the old index and a row the
	// filter leaves cannot be marked by one. An unchanged pair keeps the list's
	// own choice, which the arrow keys moved.
	if *v.Selected != v.state.synced || *v.searchQuery() != v.state.filter {
		v.state.choice = rowIndex(rows, *v.Selected)
		v.state.synced = *v.Selected
		v.state.filter = *v.searchQuery()
	}
	ui.Column(c).Grow(1).Children(func() {
		workspaceEmpty(c, k, v, len(rows))
		if len(rows) == 0 {
			return
		}
		list := &v.state.list
		list.Selected = &v.state.choice
		list.Label = func(row int) string { return rows[row].accessLabel() }
		list.Header = func(row int) bool { return rows[row].group }
		list.Key = func(row int) any {
			if rows[row].group {
				return "group:" + rows[row].label
			}
			return rows[row].sessionID
		}
		ui.List(c, list, len(rows), func(row int) {
			// A click on a session row publishes the choice itself: the list's own
			// click path moves its cursor, and with only Selected bound it does not
			// raise OnChange on every change. The row's click keeps the two in step,
			// and a heading has no click.
			var on func()
			if !rows[row].group {
				index := row
				on = func() {
					v.state.choice = index
					chooseSession(v, a)
				}
			}
			threadRow(c, k, v, rows[row], on)
		}).Grow(1).MinHeight(k.sidebarRowH*2).
			// The list paints its own face over the column, so it takes the sidebar's
			// tint instead of the window background. Its rows are inset by the
			// sidebar's padding plus the reference's row inset, so a session row's
			// selected or hover face, and the list's own choice face under it, stop
			// inside the column instead of running edge to edge.
			Background(k.sidebar).Transition(motionLayoutColors(c)).
			Padding(k.sidebarGap, k.sidebarPadX+sidebarRowEndInset, k.sidebarGap, k.sidebarPadX+sidebarRowInset).
			Label(tr(locale, keySidebarSessions)).OnChange(func() {
			chooseSession(v, a)
		})
	})
}

// workspaceEmpty draws the built-in workspace's message while the list has no
// row, and the hairline that closes the section. The message appears only then:
// above session rows it contradicted the rows under it. A filter that leaves no
// row swaps in the no-match text, because then the list itself is empty too. The
// hairline stays in both states, as the reference closes the section with it.
func workspaceEmpty(c *ui.Context, k tokens, v SidebarView, rows int) {
	locale := v.Locale
	ui.Column(c).Children(func() {
		if rows == 0 {
			message := tr(locale, keySidebarEmpty)
			if v.state.searching && len(v.Sessions) > 0 {
				message = tr(locale, keySidebarEmptyFiltered)
			}
			ui.Column(c).Padding(
				sidebarEmptyPadTop*k.sidebarGap, k.sidebarPadX+sidebarEmptyIndent,
				sidebarEmptyPadBottom*k.sidebarGap, k.sidebarPadX+sidebarEmptyIndent).
				Children(func() {
					ui.Text(c, message).FontSize(k.smallSize).TextColor(k.mutedForeground).
						Role(ui.RoleStatus)
				})
		}
		// The hairline is inset like the reference's, so it closes the section
		// without running under the sidebar's edges. With rows under it the
		// message's room is gone, so a small step keeps the rule off both the
		// workspace heading and the first group.
		padY := k.sidebarGap
		if rows == 0 {
			padY = 0
		}
		ui.Row(c).Padding(padY, k.sidebarPadX*2, padY, k.sidebarPadX*2).Children(func() {
			ui.Box(c).FillWidth().Height(k.hairline).Background(k.border).AccessHidden(true)
		})
	})
}

// threadRow draws a group heading or a session row. A session row shows the
// selected and hovered faces from the sidebar's own tokens; the heading stays
// sticky over the rows it names, so it paints the sidebar's face. The group
// heading keeps the reference's project row: py-1 around its text, 28 DIPs
// high, rather than the section heading's h-9, so the session rows under it sit
// where window-conversation.png draws them.
func threadRow(c *ui.Context, k tokens, v SidebarView, row sidebarRow, on func()) {
	if row.group {
		// The list insets every row by sidebarRowInset, but the reference keeps
		// the project heading's glyph at the column's own 7 DIP edge, so the
		// heading takes that inset back in its padding. Its face is the column's,
		// so only the ink's place changes.
		ui.Row(c).Key("thread-group:" + row.label).Height(k.sidebarEntryH).PaddingX(k.sidebarPadX - sidebarRowInset).Gap(k.sidebarGap).
			Background(k.sidebar).Transition(motionLayoutColors(c)).AlignItems(ui.Center).Children(func() {
			sidebarGlyph(c, sidebarIcons.folder, k.mutedForeground)
			ui.Text(c, row.label).FontSize(k.smallSize).TextColor(k.mutedForeground).Bold().SingleLine()
			ui.Spacer(c)
		})
		return
	}
	selected := row.sessionID == *v.Selected
	line := ui.Row(c).Key("thread:" + row.sessionID).Height(k.sidebarRowH).
		PaddingX(k.sidebarGap).Gap(k.sidebarGap).
		Radius(radiusMD).AlignItems(ui.Center).Transition(motionFastColors(c))
	if on != nil {
		line.OnClick(on)
	}
	switch {
	case selected:
		line.Background(k.sidebarAccent)
	case line.Hovered():
		line.Background(k.sidebarHover)
	}
	line.Children(func() {
		// The reference's session row carries its title and relative stamp only:
		// no status dot, because the shell has no run fact for a session row.
		ui.Text(c, row.title).FontSize(k.bodySize).TextColor(k.sidebarForeground).
			SingleLine().Grow(1)
		if stamp := relativeStamp(row.updatedAt, time.Now()); stamp != "" {
			ui.Text(c, stamp).FontSize(k.smallSize).TextColor(k.sidebarForeground).
				SingleLine()
		}
	})
}

// chooseSession publishes the list's choice. A heading is never chosen, and a
// row already selected asks for nothing, so the callback fires only for a new
// session.
func chooseSession(v SidebarView, a Actions) {
	if v.state.choice < 0 || v.state.choice >= len(v.state.rows) {
		return
	}
	row := v.state.rows[v.state.choice]
	if row.group || row.sessionID == "" || row.sessionID == *v.Selected {
		return
	}
	*v.Selected = row.sessionID
	v.state.synced = row.sessionID
	selectSession(a, row.sessionID)
}

// sidebarAccount is the footer pinned to the sidebar's bottom, as dim-electron's
// AccountEntry is: the account row under the session content, with the settings
// entry at its tail. The account is WIP — this shell has no account client, so
// it has no identity, session or sign-in fact to show — while settings is
// desktop-local and opens the same appearance surface the toolbar's entry
// toggles.
func sidebarAccount(c *ui.Context, k tokens, v SidebarView, a Actions) {
	locale := v.Locale
	ui.Row(c).MinHeight(k.sidebarFooterH).
		Padding(0, k.sidebarPadX, k.sidebarPadX, k.sidebarPadX).
		Gap(k.sidebarGap).AlignItems(ui.Center).Children(func() {
		accountEntry(c, k, locale)
		ui.Spacer(c)
		if a.OnOpenSettings == nil {
			return
		}
		// The reference's footer carries the gear alone: the label stays the
		// button's accessible name and tooltip, but no visible word sits beside
		// the glyph.
		label := tr(locale, keySidebarSettingsEntry)
		settings := ui.ButtonBase(c).Key("sidebar-settings").Height(k.sidebarFooterH).
			PaddingX(k.sidebarPadX).AlignItems(ui.Center).
			Label(label).Tooltip(label).OnClick(a.OnOpenSettings).Shrink(1).
			Transition(motionFastColors(c))
		settings.Children(func() {
			sidebarGlyph(c, sidebarIcons.gear, k.sidebarForeground)
		})
	})
}

// accountEntry is the account button on the footer row: the account glyph
// before its label. The shell has no account client, so the entry is
// unavailable and carries the reason in its description and tooltip rather than
// a visible badge. The label is the accessible name, so the glyph is skipped by
// assistive technology.
func accountEntry(c *ui.Context, k tokens, locale Locale) {
	label := tr(locale, keySidebarAccount)
	button := ui.ButtonBase(c).Key("sidebar-account").Height(k.sidebarFooterH).
		PaddingX(k.sidebarPadX).Gap(k.sidebarGap).AlignItems(ui.Center).
		Label(label).AccessDisabled(true).Description(reasonAccount).Tooltip(reasonAccount).
		Opacity(wipOpacity).Shrink(1).Transition(motionFastColors(c))
	button.Children(func() {
		sidebarGlyph(c, sidebarIcons.user, k.sidebarForeground)
		ui.Text(c, label).FontSize(k.bodySize).TextColor(k.sidebarForeground).SingleLine()
	})
}
