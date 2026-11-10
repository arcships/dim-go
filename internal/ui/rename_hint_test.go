package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestRenameDialogExplainsBlankTitleInBothLocales(t *testing.T) {
	for _, tc := range []struct {
		locale          Locale
		blank, fallback string
	}{
		{LocaleEN, "whitespace", "default title"},
		{LocaleZHCN, "空白", "默认标题"},
	} {
		for _, draft := range []string{"", "   ", " New title "} {
			t.Run(string(tc.locale)+"/"+draft, func(t *testing.T) {
				s := fixture()
				s.Settings.Locale = tc.locale
				var got []string
				s.OnRename = func(title string) { got = append(got, title) }
				old := s.Sessions[0].Title
				tt := ui.NewTester(s.View, 900, 700)
				if err := tt.Click(tr(tc.locale, keyHeaderMenu)); err != nil {
					t.Fatal(err)
				}
				clickMenuItem(t, tt, menuText(tc.locale, keySidebarRename))
				s.RenameDraft = draft
				tt.Frame()
				message := tr(tc.locale, keyHeaderRenameMessage)
				if !tt.HasText(message) || !strings.Contains(message, tc.blank) || !strings.Contains(message, tc.fallback) {
					t.Fatalf("missing visible blank-title explanation: %q, texts=%q", message, tt.Texts())
				}
				if err := tt.Click(tr(tc.locale, keyHeaderRenameConfirm)); err != nil {
					t.Fatal(err)
				}
				tt.Frame()
				if len(got) != 1 || got[0] != draft {
					t.Fatalf("callback=%q want unchanged input %q once", got, draft)
				}
				if s.RenameConfirm || s.Sessions[0].Title != old || s.ActionStatus != "" || s.ActionError != "" {
					t.Fatal("dialog invented rename result or stayed open")
				}
			})
		}
	}
}
