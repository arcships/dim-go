// This file owns the context panel: the changed-file tree above the selected
// file's diff or preview. It is this shell's counterpart of dim-electron's
// context-panel/ directory, and it draws only what the caller passes in: no
// changed files, diff or preview is read here.
//
// The panel reads ContextPanelView and Actions only. A selection and a fold are
// view state the caller owns: a click writes the selected path through
// ContextPanelView.Selected and the directory's fold through
// ContextPanelView.Expanded, exactly as the sidebar writes SidebarView.Selected.
// Actions carries no context-panel request today, because this shell wires no
// read for a file, a diff or a preview; the parameter stays so this entry keeps
// the region contract (see protocols.go and docs/ui-component-protocols.md §1)
// if one is added.
//
// Protocol note: shell.View draws this entry beside the conversation while
// State.ShowContextPanel is on, through the State.contextPanelColumn helper,
// and protocols.go records the entry. The shell still reads no changed files,
// diffs or previews of its own, so State.ContextPanel stays a caller-supplied
// display copy and starts as a zero view, which the panel draws as its WIP
// statement. docs/ui-component-protocols.md §2 does not list the view yet;
// that page is owned elsewhere and is reported as a follow-up rather than
// edited here.
//
// Geometry: docs/ui-layout-spec.md §7 puts the context panel out of its
// measured scope, so the numbers below are this component's own. Where
// dim-electron has a step, they are converted from the 14px root font size the
// spec fixes: a tree row is px-3 py-1.5 with text-sm (FileTreeNode.vue:116,134;
// DiffView.vue:558,568), the indent is ml-4 (FileTreeNode.vue:144), the panel's
// header is h-11 px-3 (ContextPanel.vue:442), a group header is h-8 px-3 text-xs
// (DiffView.vue:520), and the empty state is py-12 (DiffView.vue:498-506). The
// diff gutter is this file's own, because dim-electron draws the diff with
// markstream rather than with sized rows.

package ui

import (
	"path"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// The context panel's geometry in DIPs. See the file comment for where each
// step comes from.
const (
	contextPanelHeaderH  = 38.5 // h-11: 2.75rem at the 14px root size
	contextPanelPadX     = 10.5 // px-3: the header, the tree rows and the detail
	contextPanelRowY     = 5.25 // py-1.5: above and below a tree row
	contextPanelRowH     = 28   // py-1.5 plus text-sm's leading-5 (1.25rem)
	contextPanelIndent   = 14   // ml-4: one directory level
	contextPanelEmptyY   = 42   // py-12: above and below the empty state
	contextPanelStatusW  = 10.5 // a file row's status letter cell
	contextPanelLineNoW  = 28   // the diff's line-number gutter, one column
	contextPanelSignW    = 10.5 // the diff's +/− cell
	contextPanelDiffRowH = 21   // one diff line, at the code size
)

// contextPanelNarrowW is the width, in DIPs, below which the detail drops the
// diff's line-number gutters so its code keeps the room. It is this component's
// own threshold, not a measured spec value: the panel cannot read the window's
// content width from a ui.Context, so ContextPanelView.Width carries the room
// it was given, and a zero Width means unmeasured and draws every column.
const contextPanelNarrowW = 280

// The context panel's own width in DIPs, as the layout gives it beside the
// conversation: 320 is the reference's right panel, and the bounds keep a
// future resize or preference from taking the conversation's room. The shell
// draws the panel at contextPanelW; the row shrinks it below that when the
// window is too narrow to show both, rather than pushing the card past the
// window's edge.
const (
	contextPanelW    = 320
	contextPanelMinW = 240
	contextPanelMaxW = 400
)

// clampContextPanelWidth keeps a width the layout asks for between the panel's
// bounds, as the sidebar's clamp does for the session list.
func clampContextPanelWidth(w float32) float32 {
	if w < contextPanelMinW {
		return contextPanelMinW
	}
	if w > contextPanelMaxW {
		return contextPanelMaxW
	}
	return w
}

// contextPanelTreeShare caps the file tree's height as a percentage of the
// panel, so the detail below it keeps its place in a short window. The tree
// scrolls; the detail takes the rest.
const contextPanelTreeShare = 45

// The reasons the panel shows while the data it needs is missing. Each names
// the view field that was not filled, so the WIP badge explains itself instead
// of an empty tree or a fabricated file appearing to be a fact. The panel never
// reads a workspace, a daemon or a file itself.
const (
	reasonContextFiles  = "Unavailable: ContextPanelView.FilesReported is false; the window has not passed changed-file facts to the context panel yet."
	reasonContextDetail = "Unavailable: the window has passed no diff or preview for the selected file yet."
)

// ContextPanelStatus is a changed file's status, as the caller's tree reports
// it. The zero value is a changed file: dim-electron's "modified" status, which
// this panel names "changed".
type ContextPanelStatus uint8

const (
	// ContextPanelChanged is a file whose content changed.
	ContextPanelChanged ContextPanelStatus = iota
	// ContextPanelAdded is a file the change created.
	ContextPanelAdded
	// ContextPanelDeleted is a file the change removed.
	ContextPanelDeleted
)

// String is the status as the detail header reads it.
func (s ContextPanelStatus) String() string {
	switch s {
	case ContextPanelAdded:
		return "added"
	case ContextPanelDeleted:
		return "deleted"
	default:
		return "changed"
	}
}

// Letter is the status letter the tree shows, as git's.
func (s ContextPanelStatus) Letter() string {
	switch s {
	case ContextPanelAdded:
		return "A"
	case ContextPanelDeleted:
		return "D"
	default:
		return "M"
	}
}

// contextPanelStatusColor is the color of a status letter and of a diff
// gutter, using the shell's semantic tokens so it follows the theme.
func (k tokens) contextPanelStatusColor(s ContextPanelStatus) ui.Color {
	switch s {
	case ContextPanelAdded:
		return k.success
	case ContextPanelDeleted:
		return k.danger
	default:
		return k.warning
	}
}

// ContextPanelFile is one node of the changed-file tree: its path, its status
// and, for a directory, its children. The caller builds the tree from its own
// facts; the panel neither groups paths itself nor invents a child.
//
// A node with children is drawn as a directory: its status letter is not drawn
// and its children show while its fold is open. A node without children is a
// file and is selectable. Paths are the key of the caller's fold map and of the
// selection, so a tree with two equal paths would fold and select both.
type ContextPanelFile struct {
	Path     string
	Status   ContextPanelStatus
	Children []ContextPanelFile
}

// Directory reports whether the node has children, and so is drawn as a
// directory rather than a file.
func (f ContextPanelFile) Directory() bool { return len(f.Children) > 0 }

// Name is the node's display text: the last element of its path, as
// dim-electron's FileTreeNode.name is. A path with no element is drawn as it
// was given rather than replaced with a made-up name.
func (f ContextPanelFile) Name() string {
	if f.Path == "" {
		return ""
	}
	name := path.Base(f.Path)
	if name == "." || name == "/" {
		return f.Path
	}
	return name
}

// ContextPanelDiffKind decides how one diff line is drawn.
type ContextPanelDiffKind uint8

const (
	// ContextPanelDiffContext is an unchanged line beside the change.
	ContextPanelDiffContext ContextPanelDiffKind = iota
	// ContextPanelDiffAdded is a line the change added.
	ContextPanelDiffAdded
	// ContextPanelDiffRemoved is a line the change removed.
	ContextPanelDiffRemoved
)

// Sign is the line's gutter mark: "+" added, "-" removed, a space for context.
func (l ContextPanelDiffLine) Sign() string {
	switch l.Kind {
	case ContextPanelDiffAdded:
		return "+"
	case ContextPanelDiffRemoved:
		return "-"
	default:
		return " "
	}
}

// ContextPanelDiffLine is one line of the selected file's diff, as the caller
// read it. OldLine and NewLine are the line's number on each side, 0 when the
// line has none there (an added line has no old number and a removed line no
// new one); the cell stays in place so the columns line up.
type ContextPanelDiffLine struct {
	Kind    ContextPanelDiffKind
	Text    string
	OldLine int
	NewLine int
}

// ContextPanelView is what the context panel draws: the changed-file tree, the
// selected file's diff or preview, the room it was given, and the view state
// the caller keeps across frames.
//
// Every file, diff line and preview line is a display copy of what the caller
// read. An empty view is not an empty workspace: FilesReported tells the panel
// whether the caller has facts at all, so a missing read draws a WIP statement
// instead of "No changed files".
type ContextPanelView struct {
	// Files is the changed-file tree in the caller's order. The panel reorders
	// nothing and adds no node.
	Files []ContextPanelFile
	// FilesReported says the caller has changed-file facts for the workspace
	// it is showing. Without it, an empty Files is "no facts", not "no
	// changes", and the whole panel reads as WIP.
	FilesReported bool
	// Selected is the selected file's path, or nil while the caller keeps no
	// selection. The panel writes a clicked file's path here; it is the only
	// field besides Expanded the region writes. The detail shows only a path
	// that is selected; a stale path keeps its heading and draws whatever the
	// caller passed for it rather than being cleared locally.
	Selected *string
	// Expanded is the caller's fold state, keyed by a directory's path. A path
	// with no entry starts expanded and a nil pointer for a path means expanded.
	// The panel creates an entry when a directory is first drawn, so a caller
	// that wants folds to survive a frame passes a non-nil map. With a nil map
	// every directory starts expanded each frame and a click cannot record the
	// change.
	Expanded map[string]*bool
	// Diff is the selected file's diff lines. It is drawn when DiffReported is
	// set; a reported diff with no lines says so instead of falling back to the
	// preview.
	Diff []ContextPanelDiffLine
	// DiffReported says the caller read a diff for the selection. Without it an
	// empty Diff is unknown, not "no changes".
	DiffReported bool
	// Preview is the selected file's preview text, drawn when the diff was not
	// reported and PreviewReported is set.
	Preview string
	// PreviewReported says the caller read a preview for the selection. Without
	// either read the detail section keeps a WIP statement.
	PreviewReported bool
	// Width is the room the panel has, in DIPs, for choosing what to omit when
	// it is narrow. Zero means unmeasured: every column is drawn.
	Width float32
}

// fileCount is the number of files under the view's tree, counted from the
// caller's nodes; it names the header's count and holds no fact of its own.
func (v ContextPanelView) fileCount() int {
	var count func(nodes []ContextPanelFile) int
	count = func(nodes []ContextPanelFile) int {
		total := 0
		for _, node := range nodes {
			if node.Directory() {
				total += count(node.Children)
				continue
			}
			total++
		}
		return total
	}
	return count(v.Files)
}

// find returns the selected path's node, so the detail can name its status
// without the panel guessing one. A selection the tree no longer holds is not
// found and draws no status.
func (v ContextPanelView) find(p string) (ContextPanelFile, bool) {
	if p == "" {
		return ContextPanelFile{}, false
	}
	var walk func(nodes []ContextPanelFile) (ContextPanelFile, bool)
	walk = func(nodes []ContextPanelFile) (ContextPanelFile, bool) {
		for _, node := range nodes {
			if !node.Directory() {
				if node.Path == p {
					return node, true
				}
				continue
			}
			if found, ok := walk(node.Children); ok {
				return found, true
			}
		}
		return ContextPanelFile{}, false
	}
	return walk(v.Files)
}

// isSelected reports whether the node at p is the selected file. A nil Selected
// means the caller keeps no selection and no row is marked.
func (v ContextPanelView) isSelected(p string) bool {
	return v.Selected != nil && p != "" && *v.Selected == p
}

// selectFile records a click on a file. A nil Selected is a caller that keeps no
// selection; the click then changes nothing rather than inventing a field for it.
func (v ContextPanelView) selectFile(p string) {
	if v.Selected != nil && p != "" {
		*v.Selected = p
	}
}

// currentNode is the selected path, or "" while none is selected.
func (v ContextPanelView) currentNode() string {
	if v.Selected == nil {
		return ""
	}
	return *v.Selected
}

// fold returns the fold state of a directory at p. A caller that passed a map
// gets an entry created for a path it has not named yet, starting expanded, so
// its first click closes the directory. A nil map cannot keep the state and
// each frame starts over expanded.
func (v ContextPanelView) fold(p string) *bool {
	if v.Expanded != nil {
		if open, ok := v.Expanded[p]; ok && open != nil {
			return open
		}
		open := true
		v.Expanded[p] = &open
		return &open
	}
	open := true
	return &open
}

// renderContextPanel draws the context panel: the changed-file tree, then the
// selected file's diff or preview. It writes ContextPanelView.Selected on a
// file click and the directory's fold through ContextPanelView.Expanded; it
// calls no Actions field, because the panel starts no App Server request.
//
// With FilesReported false the whole panel is a WIP statement, not an empty
// tree: the panel cannot tell a clean workspace from a read it never received.
// With facts and no file it reads "No changed files"; it draws no invented row.
func renderContextPanel(c *ui.Context, k tokens, v ContextPanelView, a Actions) {
	// The region contract keeps Actions in every entry, but no context-panel
	// request exists to send yet (see the file comment).
	_ = a

	ui.Column(c).FillHeight().FillWidth().Clip().Background(k.background).
		BorderStartWidth(k.hairline).BorderColor(k.border).
		Role(ui.RoleRegion).Label("Context panel").Children(func() {
		if !v.FilesReported {
			contextPanelWIP(c, k)
			return
		}
		contextPanelHeader(c, k, v)
		if len(v.Files) == 0 {
			contextPanelEmpty(c, k)
			return
		}
		contextPanelTree(c, k, v)
		contextPanelDetail(c, k, v)
	})
}

// contextPanelWIP draws the panel while the caller has no changed-file facts.
// It names the missing field and offers no control, so nothing on the panel can
// look like a workspace it was never given.
func contextPanelWIP(c *ui.Context, k tokens) {
	ui.Column(c).Padding(k.space(3)).Gap(k.gap).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Context panel").Bold().FontSize(k.titleSize)
			ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
				Unselectable().AccessHidden(true)
		})
		ui.Text(c, reasonContextFiles).FontSize(k.smallSize).TextColor(k.mutedForeground)
	})
}

// contextPanelHeader names the region and counts the files the caller's tree
// holds. The count is derived from those nodes and adds no file.
func contextPanelHeader(c *ui.Context, k tokens, v ContextPanelView) {
	ui.Row(c).Height(contextPanelHeaderH).PaddingX(contextPanelPadX).Gap(k.gap).
		AlignItems(ui.Center).BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).
		Children(func() {
			ui.Text(c, "Changes").Bold().FontSize(k.labelSize).SingleLine()
			ui.Spacer(c)
			ui.Text(c, contextPanelCount(v.fileCount())).FontSize(k.smallSize).
				TextColor(k.mutedForeground).SingleLine()
		})
}

// contextPanelCount is the header's count text.
func contextPanelCount(n int) string {
	if n == 1 {
		return "1 file"
	}
	return strconv.Itoa(n) + " files"
}

// contextPanelEmpty draws the empty state of a reported, unchanged workspace.
// The text is dim-electron's contextPanel.noUncommittedChanges equivalent; the
// panel adds no row and no fabricated file to fill the room.
func contextPanelEmpty(c *ui.Context, k tokens) {
	ui.Column(c).Grow(1).Padding(k.space(3), contextPanelPadX, contextPanelEmptyY, contextPanelPadX).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "No changed files").FontSize(k.bodySize).TextColor(k.mutedForeground)
	})
}

// contextPanelTree draws the changed-file tree. It is one Tab stop: Up and Down
// move between the items shown, Right opens a directory or moves to its first
// child, Left closes it or moves to its parent, and Enter or Space on a file
// selects it. A click on a directory toggles it; a click on a file selects it.
func contextPanelTree(c *ui.Context, k tokens, v ContextPanelView) {
	ui.Scroll(c).Shrink(1).MaxHeightPercent(contextPanelTreeShare).Children(func() {
		tree := ui.TreeBase(c)
		tree.Root.Label("Changed files").Gap(k.gap).
			Padding(k.gap, contextPanelPadX, k.gap, contextPanelPadX)
		tree.Root.Children(func() {
			for _, node := range v.Files {
				contextPanelNode(c, k, v, tree, node, 0)
			}
		})
	})
}

// contextPanelNode draws one tree node and, while its fold is open, its
// children. The row's level is the tree's; the indent follows it, as
// dim-electron's nested lists do.
func contextPanelNode(c *ui.Context, k tokens, v ContextPanelView, tree ui.TreeParts, node ContextPanelFile, depth int) {
	directory := node.Directory()
	var open *bool
	if directory {
		open = v.fold(node.Path)
	}
	item := tree.Item(open)
	selected := !directory && v.isSelected(node.Path)
	row := item.Row.Height(contextPanelRowH).FillWidth().
		PaddingY(contextPanelRowY).
		PaddingStart(contextPanelPadX + float32(depth)*contextPanelIndent).
		PaddingEnd(contextPanelPadX).Gap(k.gap).AlignItems(ui.Center).
		Label(node.Name()).AccessSelected(selected)
	switch {
	case selected:
		row.Background(k.accent)
	case row.Hovered():
		row.Background(k.sidebarHover)
	}
	if !directory {
		row.Description(node.Status.String())
	}
	row.Children(func() {
		if directory {
			progress := float32(0)
			if open != nil && *open {
				progress = 1
			}
			chevron(c, k, k.mutedForeground, progress)
		} else {
			ui.Text(c, node.Status.Letter()).Bold().FontSize(k.smallSize).
				Width(contextPanelStatusW).TextColor(k.contextPanelStatusColor(node.Status)).
				AccessHidden(true)
		}
		ui.Text(c, node.Name()).FontSize(k.bodySize).TextColor(k.foreground).
			SingleLine().Grow(1)
	})
	if row.Clicked() {
		if directory && open != nil {
			*open = !*open
		} else if !directory {
			v.selectFile(node.Path)
		}
	}
	item.Group(func() {
		for _, child := range node.Children {
			contextPanelNode(c, k, v, tree, child, depth+1)
		}
	})
}

// contextPanelDetail draws what is below the tree: the selected file's path and
// status, then its diff, or its preview when no diff was reported. A selection
// with neither read keeps a WIP statement; an empty panel asks for a selection
// instead of pretending a file is open.
func contextPanelDetail(c *ui.Context, k tokens, v ContextPanelView) {
	ui.Column(c).Grow(1).MinHeight(contextPanelRowH*2).Clip().
		Padding(k.gap, contextPanelPadX, k.gap, contextPanelPadX).Gap(k.gap).
		Role(ui.RoleGroup).Label("Selected file").Children(func() {
		selected := v.currentNode()
		if selected == "" {
			ui.Text(c, "Select a file to see its diff.").FontSize(k.smallSize).
				TextColor(k.mutedForeground)
			return
		}
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, selected).Bold().FontSize(k.bodySize).SingleLine().Grow(1)
			if file, ok := v.find(selected); ok {
				ui.Text(c, file.Status.String()).FontSize(k.smallSize).
					TextColor(k.contextPanelStatusColor(file.Status)).SingleLine()
			}
		})
		switch {
		case v.DiffReported:
			contextPanelDiff(c, k, v)
		case v.PreviewReported:
			contextPanelPreview(c, k, v)
		default:
			ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
				ui.Text(c, reasonContextDetail).FontSize(k.smallSize).
					TextColor(k.mutedForeground)
				ui.Text(c, wipBadge).FontSize(k.smallSize).
					TextColor(k.mutedForeground).Unselectable().AccessHidden(true)
			})
		}
	})
}

// contextPanelDiff draws the reported diff, one row per line: the two
// line-number gutters while the panel has room, the sign, then the line. A
// reported diff with no lines says so rather than falling back to a preview,
// because the caller reported the diff as this file's detail.
func contextPanelDiff(c *ui.Context, k tokens, v ContextPanelView) {
	if len(v.Diff) == 0 {
		ui.Text(c, "No diff lines.").FontSize(k.smallSize).TextColor(k.mutedForeground)
		return
	}
	showNumbers := v.Width <= 0 || v.Width >= contextPanelNarrowW
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(0).Children(func() {
			for _, line := range v.Diff {
				contextPanelDiffRow(c, k, line, showNumbers)
			}
		})
	})
}

// contextPanelDiffRow draws one diff line. The gutter mark and the line numbers
// are decorative to assistive technology; the line's text is what it reads.
func contextPanelDiffRow(c *ui.Context, k tokens, line ContextPanelDiffLine, showNumbers bool) {
	color := k.foreground
	switch line.Kind {
	case ContextPanelDiffAdded:
		color = k.success
	case ContextPanelDiffRemoved:
		color = k.danger
	}
	row := ui.Row(c).FillWidth().Height(contextPanelDiffRowH).
		PaddingX(k.gap).Gap(k.gap).AlignItems(ui.Center)
	switch line.Kind {
	case ContextPanelDiffAdded:
		row.Background(k.wash(k.success))
	case ContextPanelDiffRemoved:
		row.Background(k.wash(k.danger))
	}
	row.Children(func() {
		if showNumbers {
			contextPanelLineNo(c, k, line.OldLine)
			contextPanelLineNo(c, k, line.NewLine)
		}
		ui.Text(c, line.Sign()).Bold().FontSize(k.codeFontSize).TextColor(color).
			Width(contextPanelSignW).TextAlign(ui.Center).AccessHidden(true)
		ui.Text(c, line.Text).FontSize(k.codeFontSize).TextColor(k.foreground).
			SingleLine().Grow(1)
	})
}

// contextPanelLineNo draws one line-number cell. A line with no number on that
// side keeps the cell empty, so the code columns stay aligned.
func contextPanelLineNo(c *ui.Context, k tokens, n int) {
	if n <= 0 {
		ui.Box(c).Width(contextPanelLineNoW).Shrink(0).AccessHidden(true)
		return
	}
	ui.Text(c, strconv.Itoa(n)).FontSize(k.smallSize).
		TextColor(k.mutedForeground).Width(contextPanelLineNoW).
		TextAlign(ui.End).SingleLine().Shrink(0).AccessHidden(true)
}

// contextPanelPreview draws the reported preview text. The text scrolls both
// ways because a preview line can be longer than the panel; it is drawn as
// given, as one string, and the panel adds no line number it does not have.
func contextPanelPreview(c *ui.Context, k tokens, v ContextPanelView) {
	if v.Preview == "" {
		ui.Text(c, "The preview is empty.").FontSize(k.smallSize).
			TextColor(k.mutedForeground)
		return
	}
	ui.ScrollBoth(c).Grow(1).Children(func() {
		ui.Text(c, v.Preview).FontSize(k.codeFontSize).TextColor(k.foreground).NoWrap()
	})
}
