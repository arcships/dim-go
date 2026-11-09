package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// indexOfText returns the index in texts of the first text containing s, or -1.
func indexOfText(texts []string, s string) int {
	for i, text := range texts {
		if strings.Contains(text, s) {
			return i
		}
	}
	return -1
}

// TestConversationRowsInOrder checks that the keyed list builds every fixture
// row, once each, in message order.
func TestConversationRowsInOrder(t *testing.T) {
	s := fixture()
	s.Messages = []Message{
		{ID: "m1", Author: AuthorUser, Content: "first row"},
		{ID: "m2", Author: AuthorAssistant, Content: "second row"},
		{ID: "m3", Author: AuthorUser, Content: "third row"},
	}
	tt := ui.NewTester(s.View, 900, 600)

	texts := tt.Texts()
	prev := -1
	for _, want := range []string{"first row", "second row", "third row"} {
		at := indexOfText(texts, want)
		if at < 0 {
			t.Fatalf("row %q is missing, texts %q", want, texts)
		}
		if at <= prev {
			t.Fatalf("row %q at %d, not after the row before it at %d, texts %q", want, at, prev, texts)
		}
		prev = at
	}

	first, last := s.conversation.Visible()
	if first != 0 || last != len(s.Messages)-1 {
		t.Fatalf("visible rows %d..%d, want 0..%d", first, last, len(s.Messages)-1)
	}
}

// TestConversationFollowsAppend checks that a streaming append reaches the view
// while the list follows its end, and that scrolling away from the end holds
// the view where the user left it, the shape a session's live rows take.
func TestConversationFollowsAppend(t *testing.T) {
	s := fixture()
	s.Messages = nil
	for i := range 80 {
		s.Messages = append(s.Messages, Message{
			ID:      fmt.Sprintf("m%02d", i),
			Author:  AuthorAssistant,
			Content: fmt.Sprintf("message-%02d", i),
		})
	}
	tt := ui.NewTester(s.View, 420, 320)
	tt.Frame()
	if !s.conversation.AtEnd() {
		first, last := s.conversation.Visible()
		t.Fatalf("list is not at its end: visible=%d..%d", first, last)
	}
	if first, _ := s.conversation.Visible(); first == 0 {
		t.Fatalf("the list built every row of 80, so it is not the virtualized list")
	}
	if !tt.HasText("message-79") {
		t.Fatalf("newest fixture row is not visible, texts %q", tt.Texts())
	}

	s.Messages = append(s.Messages, Message{ID: "m80", Author: AuthorAssistant, Content: "message-80"})
	tt.Frame()
	if !s.conversation.AtEnd() {
		first, last := s.conversation.Visible()
		t.Fatalf("list stopped following its end after the append: visible=%d..%d", first, last)
	}
	if !tt.HasText("message-80") {
		t.Fatalf("appended row is not visible, texts %q", tt.Texts())
	}

	// Scrolling up, in the conversation's area, leaves the end. The list's own
	// rect is used so the point stays inside it whatever the other rows take
	// (the status bar changed the available height).
	rect, ok := tt.Find("Conversation")
	if !ok {
		t.Fatalf("conversation list is not drawn, texts %q", tt.Texts())
	}
	tt.Scroll(rect.X+rect.W/2, rect.Y+rect.H/2, 0, -300)
	if s.conversation.AtEnd() {
		t.Fatalf("the list still shows its end after scrolling away from it")
	}
	held, _ := s.conversation.Visible()
	s.Messages = append(s.Messages, Message{ID: "m81", Author: AuthorAssistant, Content: "message-81"})
	tt.Frame()
	if first, _ := s.conversation.Visible(); first != held {
		t.Fatalf("row %d at the top after the append, want the row %d the user left", first, held)
	}
	if tt.HasText("message-81") {
		t.Fatalf("the appended row pulled the view back to the end, texts %q", tt.Texts())
	}

	// Scrolling back to the end follows it again.
	tt.Scroll(rect.X+rect.W/2, rect.Y+rect.H/2, 0, 3000)
	if !s.conversation.AtEnd() {
		first, last := s.conversation.Visible()
		t.Fatalf("the list does not follow its end again: visible=%d..%d", first, last)
	}
	if !tt.HasText("message-81") {
		t.Fatalf("the newest row is not visible again, texts %q", tt.Texts())
	}
}

// TestAssistantRowRendersMarkdown checks that an assistant row goes through the
// Markdown view: its heading, emphasis and code show as text, without their
// syntax, while a user row stays literal.
func TestAssistantRowRendersMarkdown(t *testing.T) {
	s := fixture()
	// The sidebar's rows are list items too; this test reads the conversation
	// list's items, so the fixture leaves the catalog out.
	s.Sessions = nil
	s.Selected = ""
	s.Messages = []Message{
		{ID: "m1", Author: AuthorAssistant, Content: "# Result\n\nSome **bold** and `inline code` text.\n\n```go\nfmt.Println(\"hi\")\n```\n"},
		{ID: "m2", Author: AuthorUser, Content: "plain **not** markdown"},
	}
	tt := ui.NewTester(s.View, 900, 600)

	for _, want := range []string{"Result", "Some bold and inline code text.", `fmt.Println("hi")`} {
		if !tt.HasText(want) {
			t.Errorf("markdown row is missing %q, texts %q", want, tt.Texts())
		}
	}
	for _, syntax := range []string{"# Result", "**bold**", "`inline code`", "```go"} {
		if tt.HasText(syntax) {
			t.Errorf("markdown syntax %q shows as text, texts %q", syntax, tt.Texts())
		}
	}
	if !tt.HasText("plain **not** markdown") {
		t.Errorf("user row is not literal, texts %q", tt.Texts())
	}

	// Assistive technology reads the rows and their rendered text, not the
	// Markdown source.
	if _, ok := tt.AccessByRole(ui.RoleList, "Conversation"); !ok {
		t.Errorf("conversation list is not exposed to accessibility, texts %q", tt.Texts())
	}
	row, ok := tt.AccessByRole(ui.RoleListItem, "")
	if !ok {
		t.Fatalf("assistant row is not exposed to accessibility, texts %q", tt.Texts())
	}
	if !strings.HasPrefix(row.Name, "DimAgent Result Some bold and inline code text.") {
		t.Errorf("assistant row name %q is not its rendered text", row.Name)
	}
	if strings.ContainsAny(row.Name, "#`*") {
		t.Errorf("assistant row name %q keeps its Markdown syntax", row.Name)
	}
}

// TestEmptyConversation checks the hero shown once history has loaded and holds
// no row: the heading and subtitle, with the heading as the status region so
// assistive technology hears the end of loading. The old placeholder row is
// gone from this state.
func TestEmptyConversation(t *testing.T) {
	s := fixture()
	s.Messages = nil
	tt := ui.NewTester(s.View, 900, 600)

	title := tr(LocaleEN, keyHeroTitle)
	subtitle := tr(LocaleEN, keyHeroSubtitle)
	for _, want := range []string{title, subtitle} {
		if !tt.HasText(want) {
			t.Fatalf("empty state is missing %q, texts %q", want, tt.Texts())
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleStatus, title); !ok {
		t.Fatalf("the hero heading is not a status region, texts %q", tt.Texts())
	}
	if tt.HasText("No messages yet") {
		t.Errorf("the placeholder row still shows under the hero, texts %q", tt.Texts())
	}
	if first, last := s.conversation.Visible(); last >= first {
		t.Fatalf("an empty conversation shows rows %d..%d", first, last)
	}
}

// TestEmptyConversationFollowsTheLocale checks that the hero is the shell's own
// text and follows the interface language, not a fixed English string.
func TestEmptyConversationFollowsTheLocale(t *testing.T) {
	s := fixture()
	s.Messages = nil
	s.Settings = SettingsView{Locale: LocaleZHCN}
	tt := ui.NewTester(s.View, 900, 600)

	for _, want := range []string{tr(LocaleZHCN, keyHeroTitle), tr(LocaleZHCN, keyHeroSubtitle)} {
		if !tt.HasText(want) {
			t.Fatalf("Chinese hero is missing %q, texts %q", want, tt.Texts())
		}
	}
	if tt.HasText(tr(LocaleEN, keyHeroTitle)) {
		t.Errorf("the English heading still shows in a Chinese frame, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleStatus, tr(LocaleZHCN, keyHeroTitle)); !ok {
		t.Errorf("the Chinese hero heading is not a status region, texts %q", tt.Texts())
	}
}

// TestHeroHidesWithRows checks that the hero belongs to the empty state alone: a
// session with rows draws them instead.
func TestHeroHidesWithRows(t *testing.T) {
	tt := ui.NewTester(fixture().View, 900, 600)
	for _, want := range []string{tr(LocaleEN, keyHeroTitle), tr(LocaleEN, keyHeroSubtitle)} {
		if tt.HasText(want) {
			t.Errorf("the hero shows over a conversation with rows: %q, texts %q", want, tt.Texts())
		}
	}
}

// TestHeroFitsANarrowWindow checks that the hero wraps inside a narrow main
// column instead of pushing the composer or the status bar out of the window.
func TestHeroFitsANarrowWindow(t *testing.T) {
	const width, height = 560, 420
	s := fixture()
	s.Messages = nil
	tt := ui.NewTester(s.View, width, height)

	title := textBox(t, tt, tr(LocaleEN, keyHeroTitle))
	subtitle := textBox(t, tt, tr(LocaleEN, keyHeroSubtitle))
	for _, r := range []ui.Rect{title, subtitle} {
		if r.X < 0 || r.Y < 0 || r.X+r.W > width || r.Y+r.H > height {
			t.Errorf("hero text at %.1f,%.1f %.1fx%.1f leaves the %dx%d window", r.X, r.Y, r.W, r.H, width, height)
		}
	}
	if !tt.HasText("Composer") {
		t.Errorf("the composer is missing from the narrow frame, texts %q", tt.Texts())
	}
}

// TestComposerKeepsDraftAcrossFrames checks that the draft survives a frame
// that rebuilds the whole tree, as an appended row does.
func TestComposerKeepsDraftAcrossFrames(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Type("draft survives")

	s.Messages = append(s.Messages, Message{ID: "m3", Author: AuthorAssistant, Content: "another row"})
	tt.Frame()

	if s.Draft != "draft survives" {
		t.Fatalf("draft %q after the rebuild, want %q", s.Draft, "draft survives")
	}
	info, ok := tt.AccessByRole(ui.RoleTextField, "Composer")
	if !ok {
		t.Fatalf("composer is not exposed to accessibility, texts %q", tt.Texts())
	}
	if info.Value != "draft survives" {
		t.Fatalf("composer value %q after the rebuild, want %q", info.Value, "draft survives")
	}
	tt.Type("!")
	if s.Draft != "draft survives!" {
		t.Fatalf("draft %q after typing, want %q", s.Draft, "draft survives!")
	}
}
