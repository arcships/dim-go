// This file owns the timeline region: the keyed conversation list that follows
// its end, the loading and empty states, and the per-author row dispatch. It
// reads TimelineView and Actions only (see protocols.go).

package ui

import "github.com/egoist/mygo/ui"

// timeline shows the projection's rows in a keyed, virtualized list that
// follows its end, the shape the TimelineFold-driven conversation takes: only
// the rows in view are built, a row keeps its state by message ID, and a
// streaming append stays in view until the user scrolls away from the end.
func timeline(c *ui.Context, k tokens, v TimelineView, a Actions) {
	switch {
	case v.Loading:
		loadingConversation(c, k)
	case len(v.Messages) == 0:
		emptyConversation(c, k)
	default:
		list := v.list
		list.FollowEnd = true
		list.Key = func(row int) any { return v.Messages[row].ID }
		ui.List(c, list, len(v.Messages), func(row int) {
			messageRow(c, k, v, a, v.Messages[row])
		}).Grow(1).Label("Conversation")
	}
}

// loadingConversation is what shows while history is still coming. The row is a
// busy status region, so assistive technology waits for the rows rather than
// reading the placeholder, and reads the one label of the work.
func loadingConversation(c *ui.Context, k tokens) {
	ui.Column(c).Grow(1).Padding(k.panelPad).Gap(k.gap).Children(func() {
		ui.Row(c).Role(ui.RoleStatus).Busy(true).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).AccessHidden(true) // the text beside it says what the work is
			ui.Text(c, "Loading conversation…").FontSize(k.bodySize).TextColor(k.mutedForeground)
		})
	})
}

// brandMark is the official DimAgent mark, the 45:32 filled path the reference's
// BrandMark.vue draws (its d attribute, from brand-mark.ts). ui.SVG is parsed
// once at start-up and drawn every frame through ui.Icon, which paints the
// shapes in the text color, so the mark takes the shell's primary fill in both
// appearances instead of carrying a color of its own.
var brandMark = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="45" height="32" viewBox="0 0 45 32"><path fill="currentColor" d="M30.5753 0.061063C24.084 -0.350713 17.6572 1.3463 12.0353 4.35976C5.50548 7.916 -1.94527 14.9536 0.463184 22.9957C2.3951 28.848 9.11815 30.9755 14.8817 30.9755C17.8697 31.0254 18.1981 26.8827 15.223 26.5021C14.3923 26.4085 13.2267 26.3523 12.4088 26.1839C6.3362 25.1045 3.08415 21.1802 6.96729 15.6524C11.0179 10.1808 18.1273 7.04878 24.889 6.15036C32.3526 5.07724 42.463 8.11566 35.7206 16.6569C34.1557 18.5723 32.1079 20.1944 29.8926 21.5046C29.2487 21.8852 28.4244 21.4422 28.4244 20.7123V16.8378C28.4244 15.2718 27.1171 14.0053 25.5008 14.0053C23.8844 14.0053 22.5771 15.2718 22.5771 16.8378V29.1599C22.5771 30.7259 23.8844 31.9924 25.5008 31.9924C26.8145 31.9924 27.9221 31.1564 28.2892 30.0022C28.4308 29.5654 28.7335 29.1973 29.1714 29.0164C29.6544 28.8168 30.1309 28.6046 30.6075 28.3863C36.5835 25.4976 42.4114 20.9868 44.1502 14.3297C46.1336 5.66371 38.6249 0.52899 30.5753 0.061063Z"/></svg>`))

// renderHero draws the empty conversation's hero: the official brand mark and
// the heading, centered in the room above the composer, as window-workspace.png
// lays out the empty main area. The mark is the reference's filled 45:32 shape
// in the primary green, not a letter tile; the heading is the single line the
// reference keeps, with no subtitle.
//
// The title is a status region: assistive technology hears the end of loading
// rather than finding the busy region gone and nothing said. The locale is the
// frame's, so the hero follows the interface language like every other region.
func renderHero(c *ui.Context, k tokens, locale Locale) {
	ui.Column(c).Grow(1).Center().Gap(k.heroGap).Padding(k.heroPad).Children(func() {
		// The mark is decorative: the heading beneath it carries the words.
		ui.Icon(c, brandMark).Size(k.heroMarkW, k.heroMarkH).TextColor(k.primary).
			AccessHidden(true)
		ui.Text(c, tr(locale, keyHeroTitle)).FontSize(k.heroTitleSize).
			FixedLineHeight(k.heroTitleLineH).FontWeight(600).
			TextAlign(ui.Center).Role(ui.RoleStatus)
	})
}

// emptyConversation is what shows once history has loaded and holds no row
// while the window keeps a state the hero must not speak over, such as a
// terminal session.closed. Its one line is a status region, so assistive
// technology hears the end of loading rather than finding the busy region gone
// and nothing said.
func emptyConversation(c *ui.Context, k tokens) {
	ui.Column(c).Grow(1).Padding(k.panelPad).Gap(k.gap).Children(func() {
		ui.Text(c, "No messages yet").FontSize(k.bodySize).TextColor(k.mutedForeground).Role(ui.RoleStatus)
		ui.Text(c, "The conversation fills in when the App Server projection arrives.").
			FontSize(k.smallSize).TextColor(k.mutedForeground)
	})
}

// messageRow builds one row by its author, the way docs/ui-layout-spec.md §4
// splits dim-electron's timeline: a user bubble, an assistant turn with its
// blocks in order, a tool card, or an error/retry notice. A pending approval is
// not a row: it takes the composer's slot (renderApprovalPanel). A row's own text
// stands in for an unknown author, so a caller cannot lose text to a row type the
// shell does not know.
func messageRow(c *ui.Context, k tokens, v TimelineView, a Actions, m Message) {
	switch m.Author {
	case AuthorUser:
		userRow(c, k, m)
	case AuthorAssistant:
		assistantRow(c, k, v, m)
	case AuthorTool:
		toolRow(c, k, v.folds, m)
	case AuthorError:
		errorRow(c, k, m)
	case AuthorRetry:
		retryRow(c, k, m)
	default:
		ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Children(func() {
			ui.Text(c, m.Content).FontSize(k.bodySize)
		})
	}
}
