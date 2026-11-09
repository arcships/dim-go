// This file owns the session sidebar: the brand row, the global entries, the
// grouped thread list, the branch list, the thread actions and the WIP
// capability cards. It reads SidebarView and Actions only (see protocols.go).

package ui

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// ungroupedLabel heads the thread rows the caller gave no group, when other
// rows do carry one. It names the absence of a group; it is not a project.
const ungroupedLabel = "No project"

// sidebarRow is one row of the sidebar's thread list: a group heading or a
// session. Every field is derived from SidebarView.Sessions each frame, so the
// row holds no fact of its own and cannot outlive the catalog it came from.
type sidebarRow struct {
	// group marks a heading row, which the list shows sticky and never
	// chooses.
	group bool
	// label and count are the heading's text and the rows under it.
	label string
	count int
	// sessionID identifies a session row, and keys the list.
	sessionID string
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
// the ungrouped sessions fall under ungroupedLabel. The catalog's order is
// kept inside each heading, and no row is invented.
func threadRows(sessions []Session) []sidebarRow {
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
			rows = append(rows, sidebarRow{sessionID: session.ID, title: session.Title})
		}
		return rows
	}
	var order []string
	buckets := make(map[string][]sidebarRow, len(sessions))
	for _, session := range sessions {
		label := session.Group
		if label == "" {
			label = ungroupedLabel
		}
		if _, ok := buckets[label]; !ok {
			order = append(order, label)
		}
		buckets[label] = append(buckets[label], sidebarRow{sessionID: session.ID, title: session.Title})
	}
	for _, label := range order {
		rows = append(rows, sidebarRow{group: true, label: label, count: len(buckets[label])})
		rows = append(rows, buckets[label]...)
	}
	return rows
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

// searchShortcutKey is the shortcut the search entry's hint names, and
// searchShortcutMods its modifier: Cmd+G on macOS, Ctrl+G elsewhere, as
// searchShortcutText writes it.
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

// sidebar draws the P0 sidebar: the brand header, the global entries, the
// grouped thread list and the footer of actions the shell cannot carry out
// yet. Its width comes from the split divider around it; the shell clamps it
// to the sidebar's own bounds before every frame.
func sidebar(c *ui.Context, k tokens, v SidebarView, a Actions) {
	v.adoptSearch()
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
		Role(ui.RoleNavigation).Label("Session sidebar").Children(func() {
		sidebarHeader(c, k)
		sidebarEntries(c, k, v, a)
		threadListHeader(c, k, len(sessions))
		threadList(c, k, v, sessions, a)
		if len(v.Branches) > 0 {
			branchList(c, k, v, a)
		}
		sidebarFooter(c, k)
		capabilityCards(c, k, v.Capabilities)
	})
}

// sidebarHeader is the brand row, 48 DIP high as the titlebar it sits beside.
// The update entry stays visible and unavailable until the shell is wired to
// the app's updater.
func sidebarHeader(c *ui.Context, k tokens) {
	ui.Row(c).Height(k.sidebarHeaderH).Margin(k.sidebarHeaderTop, 0, 0, 0).
		PaddingX(k.sidebarPadX).Gap(k.sidebarGap).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "DimAgent").Bold().FontSize(k.titleSize).TextColor(k.sidebarForeground).SingleLine()
		ui.Spacer(c)
		wipAction(c, k, "Update", reasonUpdate)
	})
}

// sidebarEntries are the sidebar's global entries: the session action, then
// the search entry, then the keys browser. Search is local: the entry expands a
// filter box over the catalog the sidebar already draws, and nothing it does
// reaches the App Server.
func sidebarEntries(c *ui.Context, k tokens, v SidebarView, a Actions) {
	ui.Column(c).Padding(k.sidebarPadX).Gap(k.sidebarGap).Children(func() {
		newSessionEntry(c, k, v, a)
		searchEntry(c, k, v)
		keysEntry(c, k, v, a)
	})
}

// searchEntry toggles the sidebar's local session search. It needs no caller
// callback, because filtering the display copy sends nothing: the entry is real
// while the shell has a catalog at all. The box under it edits the query in
// place, and the thread list below narrows on the same frame.
func searchEntry(c *ui.Context, k tokens, v SidebarView) {
	sidebarEntry(c, k, "Search sessions", searchShortcutText(), "", true, false, v.state.searching, func() {
		v.toggleSearch()
	})
	if !v.state.searching {
		return
	}
	ui.SearchField(c, v.searchQuery()).Label("Filter sessions").
		Placeholder("Title or id").TextColor(k.sidebarForeground).FillWidth()
}

// newSessionEntry offers session.create once the window wires it. Without the
// callback the entry stays visible and unavailable rather than disappearing,
// and it is unavailable again while one creation is in flight, so a second
// click cannot ask for a second session.
func newSessionEntry(c *ui.Context, k tokens, v SidebarView, a Actions) {
	if a.OnCreateSession == nil {
		sidebarEntry(c, k, "New session", "", reasonCreateSession, false, false, false, nil)
		return
	}
	sidebarEntry(c, k, "New session", "", "", true, v.Creating, false, func() {
		createSession(a, v.Creating)
	})
}

// keysEntry offers the session.keys browser once the window has a snapshot.
// The shell wires its toggle exactly then, so a missing toggle means there is
// nothing to browse: the entry keeps its place, dimmed and explained, instead
// of opening a panel that can only say it has no read.
func keysEntry(c *ui.Context, k tokens, v SidebarView, a Actions) {
	if a.OnToggleKeys == nil {
		sidebarEntry(c, k, "Session keys", "", reasonKeysEntry, false, false, false, nil)
		return
	}
	sidebarEntry(c, k, "Session keys", "", "", true, false, v.KeysOpen, func() {
		a.OnToggleKeys()
	})
}

// reasonKeysEntry is what the keys entry says while the window has no snapshot
// to browse. The helper and the panel exist; what is missing is the wiring that
// would hand this window a session.keys read.
const reasonKeysEntry = "Unavailable: the shell has no key browser callback while no session.keys snapshot is loaded."

// sidebarEntry draws one global entry as a menu row: its label, an optional
// shortcut hint and, when the shell cannot carry the action out, the visible
// WIP badge with the reason it waits on. An unavailable entry registers no
// click, so the shell cannot report an action it did not send. selected marks
// an entry that is showing its surface, as the keys browser does.
func sidebarEntry(c *ui.Context, k tokens, label, hint, reason string, enabled, disabled, selected bool, on func()) {
	row := ui.ButtonBase(c).Height(k.sidebarEntryH).FillWidth().
		PaddingX(k.sidebarPadX).Radius(radiusMD).Gap(k.sidebarGap).AlignItems(ui.Center)
	row.Label(label)
	if enabled {
		row.OnClick(on)
		row.Disabled(disabled)
		// The entry expands its surface in the main column, as a disclosure
		// button does; assistive technology reads it as expanded, not selected.
		row.Expanded(selected)
	} else {
		row.AccessDisabled(true).Description(reason).Opacity(wipOpacity)
	}
	if row.Hovered() {
		row.Background(k.sidebarHover)
	}
	row.Children(func() {
		ui.Text(c, label).FontSize(k.bodySize).TextColor(k.sidebarForeground).SingleLine()
		ui.Spacer(c)
		if hint != "" {
			ui.Text(c, hint).FontSize(k.smallSize).TextColor(k.mutedForeground).
				Unselectable().AccessHidden(true)
		}
		if !enabled {
			badge := ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
				Unselectable().AccessHidden(true)
			row.DescribedBy(badge)
		}
	})
}

// searchShortcutText is the shortcut the search entry opens its dialog with,
// written for the platform's keyboard: Cmd+G or Ctrl+G.
func searchShortcutText() string {
	if runtime.GOOS == "darwin" {
		return "⌘G"
	}
	return "Ctrl+G"
}

// threadListHeader names the thread list and its size. The count is the rows
// the search filter leaves. Grouping by project or date is a dim-electron view
// mode fed by catalog facts this shell does not receive, so the control stays
// visible and unavailable.
func threadListHeader(c *ui.Context, k tokens, count int) {
	ui.Row(c).Height(k.sidebarSectionH).PaddingX(k.sidebarSectionPadX).Gap(k.sidebarGap).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Sessions").FontSize(k.smallSize).TextColor(k.mutedForeground).Bold().SingleLine()
		ui.Text(c, strconv.Itoa(count)).FontSize(k.smallSize).TextColor(k.mutedForeground)
		ui.Spacer(c)
		wipAction(c, k, "Group", reasonGroup)
	})
}

// threadList draws the grouped thread rows in a keyed, virtualized list with
// one scroll container. The list's choice is the shell's selected session:
// a click or an arrow key moves the choice, and chooseSession publishes it.
// sessions is the display copy the search filter left; the catalog itself is
// never written.
func threadList(c *ui.Context, k tokens, v SidebarView, sessions []Session, a Actions) {
	rows := threadRows(sessions)
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
	if len(rows) == 0 {
		empty := "No sessions yet"
		if v.state.searching && len(v.Sessions) > 0 {
			empty = "No matching sessions"
		}
		ui.Column(c).Grow(1).Padding(k.sidebarPadX).Children(func() {
			ui.Text(c, empty).FontSize(k.smallSize).TextColor(k.mutedForeground).Role(ui.RoleStatus)
		})
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
		threadRow(c, k, v, rows[row])
	}).Grow(1).MinHeight(k.sidebarRowH*2).
		Padding(k.sidebarGap, k.sidebarPadX).
		Label("Sessions").OnChange(func() {
		chooseSession(v, a)
	})
}

// threadRow draws a group heading or a session row. A session row shows the
// selected and hovered faces from the sidebar's own tokens; the heading stays
// sticky over the rows it names, so it paints the sidebar's face.
func threadRow(c *ui.Context, k tokens, v SidebarView, row sidebarRow) {
	if row.group {
		ui.Row(c).Height(k.sidebarSectionH).PaddingX(k.sidebarPadX).Gap(k.sidebarGap).
			Background(k.sidebar).AlignItems(ui.Center).Children(func() {
			ui.Text(c, row.label).FontSize(k.smallSize).TextColor(k.mutedForeground).Bold().SingleLine()
			ui.Text(c, strconv.Itoa(row.count)).FontSize(k.smallSize).TextColor(k.mutedForeground)
			ui.Spacer(c)
		})
		return
	}
	selected := row.sessionID == *v.Selected
	line := ui.Row(c).Height(k.sidebarRowH).PaddingX(k.sidebarGap).Gap(k.sidebarGap).
		Radius(radiusMD).AlignItems(ui.Center)
	switch {
	case selected:
		line.Background(k.sidebarAccent)
	case line.Hovered():
		line.Background(k.sidebarHover)
	}
	line.Children(func() {
		// The status cell is 21 DIPs square, as dim-electron's 24px cell at
		// the 14px root size. The marker inside it is decorative: the shell
		// has no run fact for a session row, so it must not claim one.
		ui.Box(c).Size(k.sidebarCell, k.sidebarCell).Center().Children(func() {
			dot := k.mutedForeground
			if selected {
				dot = k.sidebarPrimary
			}
			ui.Box(c).Size(k.sidebarGap*2, k.sidebarGap*2).Radius(k.sidebarGap).
				Background(dot).AccessHidden(true)
		})
		ui.Text(c, row.title).FontSize(k.bodySize).TextColor(k.sidebarForeground).
			SingleLine().Grow(1)
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

// branchList draws the branch directory the App Server supplied. With
// Actions.OnSelectBranch wired each row asks the window to open that branch's
// history; without the callback the list keeps the read-only metadata form and
// the WIP reason, so an unwired window offers no switch it cannot perform. The
// region marks the displayed branch and writes nothing: a switch shows only
// after the window publishes the branch the App Server returned.
func branchList(c *ui.Context, k tokens, v SidebarView, a Actions) {
	selectable := a.OnSelectBranch != nil
	label := "Branches (read-only)"
	if selectable {
		label = "Branches"
	}
	ui.Column(c).Padding(k.gap, k.sidebarPadX).Gap(k.gap).
		Role(ui.RoleGroup).Label(label).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Branches").FontSize(k.smallSize).TextColor(k.mutedForeground).Bold().AccessHidden(true)
			ui.Box(c).Grow(1)
			if !selectable {
				ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
			}
		})
		for _, branch := range v.Branches {
			current := branchCurrent(branch, v.SelectedBranch)
			if selectable {
				branchEntry(c, k, branch, current, a)
				continue
			}
			ui.Text(c, branchLabel(branch, current)).FontSize(k.smallSize).TextColor(k.mutedForeground).NoWrap()
		}
		if !selectable {
			ui.Text(c, reasonBranches).FontSize(k.smallSize).TextColor(k.mutedForeground)
		}
	})
}

// branchLabel is a row's text: the caller's label, or the branch's id when the
// App Server gave none, with the displayed branch named.
func branchLabel(branch Branch, current bool) string {
	label := branch.Label
	if label == "" {
		label = branch.ID
	}
	if current {
		label += " (current)"
	}
	return label
}

// branchCurrent reports whether the branch is the one the shell displays. Both
// inputs are App Server facts the shell copied into the view; neither is
// written here.
func branchCurrent(branch Branch, selectedBranch string) bool {
	return branch.Selected || branch.ID == selectedBranch
}

// branchEntry draws one selectable branch row. A click on the displayed branch
// asks for nothing; any other asks the window to open that branch, and the
// row's selected face follows the view, so a failed read never looks switched.
func branchEntry(c *ui.Context, k tokens, branch Branch, current bool, a Actions) {
	label := branchLabel(branch, false)
	row := ui.ButtonBase(c).Height(k.sidebarEntryH).FillWidth().
		PaddingX(k.sidebarGap).Radius(radiusMD).Gap(k.sidebarGap).
		AlignItems(ui.Center).Role(ui.RoleListItem).Label(label).Selected(current)
	row.OnClick(func() {
		selectBranch(a, branch.ID, current)
	})
	if !current && row.Hovered() {
		row.Background(k.sidebarHover)
	}
	row.Children(func() {
		ui.Text(c, label).FontSize(k.smallSize).TextColor(k.sidebarForeground).
			SingleLine().Grow(1)
		if current {
			ui.Text(c, "current").FontSize(k.smallSize).TextColor(k.mutedForeground).AccessHidden(true)
		}
	})
}

// sidebarFooter keeps the thread actions dim-electron offers in a row's tail
// — pin, rename and archive — visible and unavailable: the App Server has no
// verified pin or archive RPC for them, session.rename is reached through
// session.methods, and this shell keeps no local list for any of the three.
func sidebarFooter(c *ui.Context, k tokens) {
	ui.Row(c).MinHeight(k.sidebarFooterH).
		Padding(0, k.sidebarPadX, k.sidebarPadX, k.sidebarPadX).
		Gap(k.sidebarGap).Wrap().AlignItems(ui.Center).
		Role(ui.RoleGroup).Label("Thread actions (unavailable)").Children(func() {
		var actions []ui.Element
		for _, action := range []struct{ label, reason string }{
			{"Pin", reasonPin},
			{"Rename", reasonRename},
			{"Archive", reasonArchive},
		} {
			actions = append(actions, ui.Button(c, action.label).AccessDisabled(true).
				Description(action.reason).Opacity(wipOpacity))
		}
		// One visible badge after the three entries describes them all, so the
		// row reads "Pin Rename Archive WIP" rather than repeating the badge.
		badge := ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
			Unselectable().AccessHidden(true)
		for _, action := range actions {
			action.DescribedBy(badge)
		}
	})
}
