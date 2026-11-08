// Package ui builds the DimAgent native view tree from plain Go view state.
//
// It is the AppFrame spike of Milestone 0: a session sidebar, a conversation
// placeholder and a composer. It holds no session, run or approval facts; the
// real catalog and timeline come from the App Server projection, so everything
// here is throwaway view state.
package ui

import "github.com/egoist/mygo/ui"

// Session is a sidebar row. The fixture fills it by hand until the App Server
// session catalog lands.
type Session struct {
	ID    string
	Title string
}

// State is the view state of the shell spike.
type State struct {
	Sessions []Session
	Selected string
	// Messages are placeholder conversation rows, not session facts.
	Messages []string
	Draft    string
}

// View builds one frame of the AppFrame: sidebar, conversation and composer.
func (s *State) View(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Width(220).Shrink(0).PaddingY(8).Background(t.Surface).Children(func() {
			ui.Sidebar(c, &s.Selected, func() {
				for _, session := range s.Sessions {
					ui.SidebarItem(c, session.ID, nil, session.Title)
				}
			}).Grow(1).Label("Sessions")
		})
		ui.Column(c).Grow(1).FillHeight().Children(func() {
			s.header(c, t)
			s.conversation(c, t)
			s.composer(c)
		})
	})
}

// header shows the selected session's title in a toolbar row.
func (s *State) header(c *ui.Context, t *ui.Theme) {
	ui.Row(c).Gap(12).Padding(8, 12).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		ui.Text(c, s.selectedTitle()).Bold()
		ui.Box(c).Grow(1)
		ui.Text(c, "App Server not connected").FontSize(12).TextColor(t.TextMuted)
	})
}

// conversation is the placeholder for the TimelineFold-driven ui.List.
func (s *State) conversation(c *ui.Context, t *ui.Theme) {
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Padding(16).Gap(12).Children(func() {
			for i, message := range s.Messages {
				ui.Textf(c, "%d", i).FontSize(12).TextColor(t.TextMuted)
				ui.Text(c, message).FontSize(14)
			}
		})
	})
}

// composer is the placeholder for the TextArea/TextBuffer composer fixture.
func (s *State) composer(c *ui.Context) {
	ui.Column(c).Padding(12).Gap(6).Children(func() {
		ui.TextArea(c.Key("composer"), &s.Draft).
			Height(72).Placeholder("Message DimAgent").Label("Composer")
	})
}

// selectedTitle returns the title of the selected session, or a placeholder.
func (s *State) selectedTitle() string {
	for _, session := range s.Sessions {
		if session.ID == s.Selected {
			return session.Title
		}
	}
	return "No session"
}
