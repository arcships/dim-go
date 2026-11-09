// This file owns the session sidebar: the brand row, the global entries, the
// grouped thread list, the read-only branch list, the thread actions and the WIP
// capability cards. It reads SidebarView and Actions only (see protocols.go).

package ui

import (
	"runtime"
	"strconv"

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

// sidebar draws the P0 sidebar: the brand header, the global entries, the
// grouped thread list and the footer of actions the shell cannot carry out
// yet. Its width comes from the split divider around it; the shell clamps it
// to the sidebar's own bounds before every frame.
func sidebar(c *ui.Context, k tokens, v SidebarView, a Actions) {
	ui.Column(c).FillHeight().Clip().Background(k.sidebar).
		Role(ui.RoleNavigation).Label("Session sidebar").Children(func() {
		sidebarHeader(c, k)
		sidebarEntries(c, k, v, a)
		threadListHeader(c, k, v)
		threadList(c, k, v, a)
		if len(v.Branches) > 0 {
			branchList(c, k, v)
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
// the search entry. Search opens a dialog in dim-electron; this shell has no
// search, so the entry keeps its place, its shortcut hint and the reason it
// waits on rather than disappearing.
func sidebarEntries(c *ui.Context, k tokens, v SidebarView, a Actions) {
	ui.Column(c).Padding(k.sidebarPadX).Gap(k.sidebarGap).Children(func() {
		newSessionEntry(c, k, v, a)
		sidebarEntry(c, k, "Search sessions", searchShortcutText(), reasonSearch, false, false, nil)
	})
}

// newSessionEntry offers session.create once the window wires it. Without the
// callback the entry stays visible and unavailable rather than disappearing,
// and it is unavailable again while one creation is in flight, so a second
// click cannot ask for a second session.
func newSessionEntry(c *ui.Context, k tokens, v SidebarView, a Actions) {
	if a.OnCreateSession == nil {
		sidebarEntry(c, k, "New session", "", reasonCreateSession, false, false, nil)
		return
	}
	sidebarEntry(c, k, "New session", "", "", true, v.Creating, func() {
		createSession(a, v.Creating)
	})
}

// sidebarEntry draws one global entry as a menu row: its label, an optional
// shortcut hint and, when the shell cannot carry the action out, the visible
// WIP badge with the reason it waits on. An unavailable entry registers no
// click, so the shell cannot report an action it did not send.
func sidebarEntry(c *ui.Context, k tokens, label, hint, reason string, enabled, disabled bool, on func()) {
	row := ui.ButtonBase(c).Height(k.sidebarEntryH).FillWidth().
		PaddingX(k.sidebarPadX).Radius(radiusMD).Gap(k.sidebarGap).AlignItems(ui.Center)
	row.Label(label)
	if enabled {
		row.OnClick(on)
		row.Disabled(disabled)
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

// threadListHeader names the thread list and its size. Grouping by project or
// date is a dim-electron view mode fed by catalog facts this shell does not
// receive, so the control stays visible and unavailable.
func threadListHeader(c *ui.Context, k tokens, v SidebarView) {
	ui.Row(c).Height(k.sidebarSectionH).PaddingX(k.sidebarSectionPadX).Gap(k.sidebarGap).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Sessions").FontSize(k.smallSize).TextColor(k.mutedForeground).Bold().SingleLine()
		ui.Text(c, strconv.Itoa(len(v.Sessions))).FontSize(k.smallSize).TextColor(k.mutedForeground)
		ui.Spacer(c)
		wipAction(c, k, "Group", reasonGroup)
	})
}

// threadList draws the grouped thread rows in a keyed, virtualized list with
// one scroll container. The list's choice is the shell's selected session:
// a click or an arrow key moves the choice, and chooseSession publishes it.
func threadList(c *ui.Context, k tokens, v SidebarView, a Actions) {
	rows := threadRows(v.Sessions)
	v.state.rows = rows
	if *v.Selected != v.state.synced {
		v.state.choice = rowIndex(rows, *v.Selected)
		v.state.synced = *v.Selected
	}
	if len(rows) == 0 {
		ui.Column(c).Grow(1).Padding(k.sidebarPadX).Children(func() {
			ui.Text(c, "No sessions yet").FontSize(k.smallSize).TextColor(k.mutedForeground).Role(ui.RoleStatus)
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

// branchList shows the branch metadata supplied by the App Server as read-only
// rows. Branch switching and cut point remain disabled, so the list offers no
// switch or side-chat action.
func branchList(c *ui.Context, k tokens, v SidebarView) {
	ui.Column(c).Padding(k.gap, k.sidebarPadX).Gap(k.gap).
		Role(ui.RoleGroup).Label("Branches (read-only)").Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Branches").FontSize(k.smallSize).TextColor(k.mutedForeground).Bold().AccessHidden(true)
			ui.Box(c).Grow(1)
			ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
		})
		for _, branch := range v.Branches {
			label := branch.Label
			if label == "" {
				label = branch.ID
			}
			if branch.Selected || branch.ID == v.SelectedBranch {
				label += " (current)"
			}
			ui.Text(c, label).FontSize(k.smallSize).TextColor(k.mutedForeground).NoWrap()
		}
		ui.Text(c, reasonBranches).FontSize(k.smallSize).TextColor(k.mutedForeground)
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
