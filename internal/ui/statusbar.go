// This file owns the bottom status bar: the selected session's git branch, the
// current context usage, the model, the account's plan usage and the App Server
// connection. It reads StatusBarView only (see statusbar.go's own contract
// below) and draws no fact of its own.
package ui

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// The status bar's geometry in DIPs, from docs/ui-layout-spec.md §7: the bar is
// dim-electron's `.chat-status-bar` with h-10, px-5 and text-xs
// (A/styles/global.css:39-41). At the 14px root font size the spec fixes those
// rem steps at 35, 17.5 and 10.5 DIPs; ui-parity.md §2.4's 40 / 20 / 12 is the
// same bar read at a 16px root and is noted there for correction. As with the
// shell's other converted rem steps (see tokens.go), the values are kept at the
// 14px reference instead of scaling with the user's font-size step.
//
// The meter's track is dim-electron's `h-1 w-10 rounded bg-muted`
// (ContextUsageMeter.vue:47-51): 0.25rem × 2.5rem, which is 3.5 × 35 DIPs at the
// same root size.
const (
	statusBarH      = 35
	statusBarPadX   = 17.5
	statusBarMeterW = 35
	statusBarMeterH = 3.5
)

// The widths at which the bar gives up its middle items rather than squeezing
// them. The shell cannot read the window's content width from a ui.Context, so
// StatusBarView.Width carries the room the bar was given; a zero width means
// the caller has not measured one yet, and the bar draws every item.
//
// The thresholds are the sum of the parts at the 14px root size: branch, the
// meter and its label, the model, the account line and the connection label,
// plus the bar's 35 DIPs of padding and a gap between each. Below the wider
// one the context meter goes first, as the middle item; below the narrower one
// the model and account go too, leaving the branch and the connection.
const (
	statusBarCompactW = 640
	statusBarNarrowW  = 420
)

// The reasons a status item shows while its data is missing. Each names the
// field the view did not fill, so the visible WIP badge explains itself instead
// of a fabricated branch, percentage or account appearing to be a fact. They
// stay view-level statements: the App Server owns the facts, and this shell has
// not passed them to the bar yet.
const (
	reasonStatusBranch  = "Unavailable: StatusBarView.Branch is empty; the window has not passed the session's git branch to the status bar yet."
	reasonStatusContext = "Unavailable: StatusBarView.Context has no reported usage; App Server context usage is not passed to the status bar yet."
	reasonStatusModel   = "Unavailable: StatusBarView.Model is empty; App Server model facts are not passed to the status bar yet."
	reasonStatusAccount = "Unavailable: StatusBarView.Account is empty; the account is not passed to the status bar yet."
)

// ContextUsage is the current turn's context window usage as the App Server
// reported it. A zero value is unreported, not "no tokens used": the drawing
// only calls it known when Reported is set, so a missing reading never turns
// into a 0% bar.
type ContextUsage struct {
	// Used and Limit are the tokens the current turn holds and the model's
	// context window, as the App Server reported them. Limit is 0 when the
	// window size is itself unknown, which draws the used count without a bar.
	Used, Limit int64
	// Reported says the App Server reported usage for this turn. Without it the
	// item is WIP even though Used and Limit are zero values.
	Reported bool
	// Estimated marks the reading as the App Server's estimate, which the bar
	// shows with a leading "~" instead of presenting a guess as exact.
	Estimated bool
}

// fraction is how much of the context window is used, between 0 and 1, and
// whether both numbers are known. It is the only source of the meter's width,
// so an unknown limit cannot produce a bar.
func (u ContextUsage) fraction() (float64, bool) {
	if !u.Reported || u.Limit <= 0 {
		return 0, false
	}
	value := float64(u.Used) / float64(u.Limit)
	return math.Min(1, math.Max(0, value)), true
}

// PlanUsage is the account's plan usage, as the account service reported it.
// A zero value is no reading, not 0% used: Reported tells them apart.
type PlanUsage struct {
	// Percent is the plan's usage from 0 to 100.
	Percent float64
	// Reported says the account service reported a percentage. Without it the
	// account line says so instead of drawing 0%.
	Reported bool
}

// StatusBarView is what the status bar draws: the selected session's branch and
// context usage, the model, the signed-in account's plan usage, and the
// supervisor's connection. Every field is a display copy of what the window was
// given; an empty one is drawn as a visible WIP badge and never as a stand-in
// value.
type StatusBarView struct {
	// Width is the room the bar has, in DIPs, for choosing what to omit when the
	// window is narrow. Zero means unmeasured: every item is drawn.
	Width float32
	// Branch is the selected session's current git branch. Empty is WIP.
	Branch string
	// Context is the current turn's context usage. Unreported is WIP; a known
	// usage with an unknown limit draws no bar.
	Context ContextUsage
	// Model is the model the selected session uses, as the App Server named it.
	// Empty is WIP.
	Model string
	// Account is the signed-in account's display name. Empty is WIP.
	Account string
	// Plan is the account's plan usage. An unreported percentage is shown as
	// WIP when the account itself is known.
	Plan PlanUsage
	// Connection is the supervisor's last connection report. Its zero value is
	// the supervisor's own "Not connected" state, not a missing fact.
	Connection Connection
}

// renderStatusBar draws the bottom status bar: the git branch on the left, the
// context meter in the middle, and the model, account and connection on the
// right. It calls no action: every item is a reading, not a control.
//
// The bar is 35 DIPs high and 17.5 DIPs from either edge, from
// docs/ui-layout-spec.md §7. When StatusBarView.Width is below the compact
// threshold the context meter is omitted, and below the narrow one the model
// and account follow, so the remaining items keep their place instead of
// overlapping in a window too small for them.
func renderStatusBar(c *ui.Context, k tokens, v StatusBarView) {
	showContext := v.Width <= 0 || v.Width >= statusBarCompactW
	showDetails := v.Width <= 0 || v.Width >= statusBarNarrowW

	ui.Row(c).FillWidth().Height(statusBarH).PaddingX(statusBarPadX).
		AlignItems(ui.Center).Role(ui.RoleGroup).Label("Status bar").Children(func() {
		statusBranch(c, k, v.Branch)
		ui.Box(c).Grow(1)
		if showContext {
			statusContext(c, k, v.Context)
			ui.Box(c).Grow(1)
		}
		ui.Row(c).Gap(k.gapWide).AlignItems(ui.Center).Children(func() {
			if showDetails {
				statusModel(c, k, v.Model)
				statusAccount(c, k, v.Account, v.Plan)
			}
			statusConnection(c, k, v.Connection)
		})
	})
}

// statusBranch draws the branch name, or a WIP badge while the window has none
// to pass.
func statusBranch(c *ui.Context, k tokens, branch string) {
	if branch == "" {
		statusWIP(c, k, "Branch", reasonStatusBranch)
		return
	}
	statusReading(c, k, "Branch: "+branch)
}

// statusContext draws the context meter and its reading, or a WIP badge while
// the App Server has reported none. A known used count with an unknown limit
// shows the count and marks the limit WIP: the bar is drawn only from a known
// fraction.
func statusContext(c *ui.Context, k tokens, usage ContextUsage) {
	if !usage.Reported {
		statusWIP(c, k, "Context", reasonStatusContext)
		return
	}
	tilde := ""
	if usage.Estimated {
		tilde = "~"
	}
	fraction, hasFraction := usage.fraction()
	if !hasFraction {
		statusReading(c, k, fmt.Sprintf("Context: %s%s tokens · limit WIP",
			tilde, formatTokenCount(usage.Used)))
		return
	}
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		contextMeter(c, k, fraction)
		statusReading(c, k, fmt.Sprintf("Context: %s%s / %s (%s%d%%)",
			tilde, formatTokenCount(usage.Used), formatTokenCount(usage.Limit),
			tilde, int(math.Round(fraction*100))))
	})
}

// contextMeter draws the context usage as a filled track. It only runs when the
// fraction is known; the track is the progress element assistive technology
// reads, and its fill is a share of the track's width, so the drawing and the
// reported value cannot disagree.
func contextMeter(c *ui.Context, k tokens, fraction float64) {
	trackW, trackH := float32(statusBarMeterW), float32(statusBarMeterH)
	radius := trackH / 2
	ui.Row(c).Size(trackW, trackH).Radius(radius).Background(k.muted).Clip().
		AlignItems(ui.Stretch).Role(ui.RoleProgress).Label("Context usage").
		Range(0, 1, fraction).Children(func() {
		ui.Box(c).Width(float32(fraction) * trackW).Height(trackH).Radius(radius).
			Background(contextUsageColor(k, fraction)).Unselectable().AccessHidden(true)
	})
}

// contextUsageColor is the meter's fill, at dim-electron's thresholds: under
// half the context window is success, under four fifths warning, and past that
// danger (ContextUsageMeter.vue:18). The colors are the shell's semantic
// tokens, so the meter follows the theme.
func contextUsageColor(k tokens, fraction float64) ui.Color {
	switch {
	case fraction < 0.5:
		return k.success
	case fraction < 0.8:
		return k.warning
	default:
		return k.danger
	}
}

// statusModel draws the model name, or a WIP badge while the window has none.
func statusModel(c *ui.Context, k tokens, model string) {
	if model == "" {
		statusWIP(c, k, "Model", reasonStatusModel)
		return
	}
	statusReading(c, k, "Model: "+model)
}

// statusAccount draws the account and its plan usage together, so the two never
// read as separate facts. A known account with no plan reading says the usage
// is WIP; a missing account is WIP and shows no invented plan.
func statusAccount(c *ui.Context, k tokens, account string, plan PlanUsage) {
	if account == "" {
		statusWIP(c, k, "Account", reasonStatusAccount)
		return
	}
	reading := "Account: " + account
	if plan.Reported {
		reading += fmt.Sprintf(" · %d%% used", int(math.Round(clampPercent(plan.Percent))))
	} else {
		reading += " · usage WIP"
	}
	statusReading(c, k, reading)
}

// statusConnection draws the supervisor's connection as a dot and its label in
// a live region, as the toolbar's status row does. The zero Connection is the
// supervisor's "Not connected" state, a real reading rather than a WIP gap.
func statusConnection(c *ui.Context, k tokens, conn Connection) {
	ui.Row(c).Role(ui.RoleStatus).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(k.statusDot, k.statusDot).Radius(k.statusDot / 2).
			Background(k.connectionColor(conn)).Unselectable()
		statusReading(c, k, "App Server: "+conn.String())
	})
}

// statusReading draws one value-bearing status item: a single line of small
// secondary text at the bar's size, kept on one line so a long branch or model
// name cannot grow the bar past its height.
func statusReading(c *ui.Context, k tokens, text string) {
	ui.Text(c, text).FontSize(k.smallSize).TextColor(k.mutedForeground).
		Unselectable().NoWrap()
}

// statusWIP draws the label of one status item whose data is missing: the label
// stays where its value would be, and the visible WIP badge says why there is
// no value. The reason is attached as the item's description, so assistive
// technology reads the gap without a fabricated value.
func statusWIP(c *ui.Context, k tokens, label, reason string) {
	ui.Text(c, label+": "+wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
		Description(reason).Unselectable().NoWrap()
}

// formatTokenCount formats a token count as dim-electron's status bar does
// (context-usage-presentation.ts:25-29): one decimal for millions and
// thousands, and the plain number below a thousand.
func formatTokenCount(n int64) string {
	switch {
	case n >= 1_000_000:
		return strconv.FormatFloat(math.Floor(float64(n)/100_000)/10, 'f', 1, 64) + "M"
	case n >= 1_000:
		return strconv.FormatFloat(math.Floor(float64(n)/100)/10, 'f', 1, 64) + "K"
	default:
		return strconv.FormatInt(n, 10)
	}
}

// clampPercent keeps a reported plan percentage inside 0–100, so a provider
// that reports past its own limit still draws a sane reading.
func clampPercent(percent float64) float64 {
	return math.Min(100, math.Max(0, percent))
}
