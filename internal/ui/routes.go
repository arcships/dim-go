// This file owns the whole-window WIP pages: the scheduled-tasks and skills
// pages the sidebar's management entries open. Both take the window like the
// appearance surface, explain the client or App Server method they wait on, and
// carry a Back entry to the conversation they replaced. They send no request and
// change no session fact; the selection, draft and conversation list state stay
// where they were. The window's notice strip is drawn on top, so an action
// outcome is never hidden behind a page.

package ui

import "github.com/egoist/mygo/ui"

// wipRouteText is one WIP page's title and the gap it explains. Both strings are
// the shell's own: the title is the sidebar entry's translated label and the
// reason names the missing client or wire method literally in every language.
func wipRouteText(route Route, locale Locale) (title, reason string) {
	switch route {
	case RouteScheduled:
		return tr(locale, keySidebarScheduled), reasonScheduled
	case RouteSkills:
		return tr(locale, keySidebarSkills), reasonSkills
	default:
		return "", ""
	}
}

// renderWIPRoute draws the scheduled-tasks or skills page over the whole
// window. Back and Escape return to the conversation through Actions.OnBackToChat;
// the page itself starts no App Server read and offers no control beyond the
// return.
func renderWIPRoute(c *ui.Context, k tokens, route Route, v HeaderView, a Actions) {
	locale := v.Locale
	title, reason := wipRouteText(route, locale)
	if c.Shortcut(0, ui.KeyEscape) && a.OnBackToChat != nil {
		a.OnBackToChat()
		return
	}
	ui.Column(c).Fill().Padding(k.framePad).Gap(k.gapWide).Background(k.background).Children(func() {
		ui.Row(c).FillWidth().Gap(k.gap).AlignItems(ui.Center).Children(func() {
			back := ui.ButtonBase(c).Key("wip-route-back").Height(k.sidebarRowH).
				PaddingX(k.sidebarPadX).Radius(k.radiusMD).Gap(k.gap).AlignItems(ui.Center).
				Role(ui.RoleButton).Label(tr(locale, keySettingsBack)).
				Transition(motionFastColors(c))
			if a.OnBackToChat != nil {
				back.OnClick(a.OnBackToChat)
			}
			if back.Hovered() {
				back.Background(k.sidebarHover)
			}
			back.Children(func() {
				sidebarGlyph(c, settingsIcons.back, k.foreground)
				ui.Text(c, tr(locale, keySettingsBack)).FontSize(k.bodySize).TextColor(k.foreground).SingleLine()
			})
			ui.Text(c, title).Bold().FontSize(k.titleSize).TextColor(k.foreground).SingleLine()
		})
		// The notice strip keeps the window's action outcomes visible on the
		// page: an error, an accepted outcome or a projection notice is not
		// hidden behind the route.
		banner(c, k, v, a)
		ui.Column(c).Key("wip-route").Grow(1).FillWidth().Padding(k.cardPadY, k.cardPadX).
			Gap(k.gap).Radius(k.frameRadius).BorderWidth(k.hairline).BorderColor(k.border).
			Background(k.card).Role(ui.RoleRegion).Label(title).
			Transition(motionPanel(c)).Children(func() {
			ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
				ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
					Unselectable().AccessHidden(true)
				ui.Text(c, title).FontSize(k.bodySize).TextColor(k.foreground).Bold()
			})
			ui.Text(c, reason).FontSize(k.smallSize).TextColor(k.mutedForeground)
		})
	})
}
