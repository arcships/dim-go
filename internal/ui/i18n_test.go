package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestLocaleDefaultsToEnglish checks the language a window opens in and the
// fallback a missing or unknown preference takes: English text, with the
// language chooser naming both languages in their own words.
func TestLocaleDefaultsToEnglish(t *testing.T) {
	if got := resolveLocale(""); got != LocaleEN {
		t.Errorf("an empty locale resolves to %q, want %q", got, LocaleEN)
	}
	if got := resolveLocale("fr"); got != LocaleEN {
		t.Errorf("an unknown locale resolves to %q, want the default %q", got, LocaleEN)
	}
	if got := LocaleEN.Name(); got != "English" {
		t.Errorf("English names itself %q", got)
	}
	if got := LocaleZHCN.Name(); got != "简体中文" {
		t.Errorf("Simplified Chinese names itself %q", got)
	}

	if got := tr("", keySidebarNewSession); got != tr(LocaleEN, keySidebarNewSession) {
		t.Errorf("an empty locale draws %q, want the English %q", got, tr(LocaleEN, keySidebarNewSession))
	}
	if got := tr(Locale("fr"), keyComposerSend); got != "Send" {
		t.Errorf("an unknown locale draws %q, want the English %q", got, "Send")
	}

	// The shell itself opens in English: the fixture saves no language.
	tt := ui.NewTester(fixture().View, 900, 600)
	for _, want := range []string{"New session", "Search sessions", "Session keys", "Send", "Attach file"} {
		if !tt.HasText(want) {
			t.Errorf("the default frame is missing the English %q, texts %q", want, tt.Texts())
		}
	}
}

// TestChineseLocaleDrawsChinese checks that a saved zh-CN preference reaches
// every region: the sidebar entries and headings, the toolbar, the composer,
// the WIP reasons and the capability cards. Nothing on the frame is a session
// fact in this fixture, so the comparison is only the shell's own words.
func TestChineseLocaleDrawsChinese(t *testing.T) {
	s := fixture()
	s.Settings = SettingsView{Locale: LocaleZHCN}
	s.Sessions = append(s.Sessions, Session{ID: "s3", Title: "Third session", Group: "alpha"})
	s.SessionClosed = true
	s.SessionClosedReason = "done"
	tt := ui.NewTester(s.View, 900, 600)

	for _, want := range []string{
		// Sidebar entries, headings and the search box.
		"新建会话", "搜索会话", "定时任务", "技能/插件", "会话键", "会话", "分组", "无项目",
		// Toolbar and banner.
		"设置", "会话已关闭",
		// The account footer.
		"登录 Dim",
		// Composer.
		"添加附件", "模式", "模型", "发送",
		// WIP reasons and the capability list.
		"会话操作（不可用）", "不可用的能力", "创建会话", "会话键",
	} {
		if !tt.HasText(want) {
			t.Errorf("the zh-CN frame is missing %q, texts %q", want, tt.Texts())
		}
	}
	// The shortcut pair is the composer's description, not a visible row, so it
	// is read by assistive technology rather than drawn.
	if info, ok := tt.Access("输入框"); !ok || info.Description != tr(LocaleZHCN, keyComposerHint) {
		t.Errorf("the composer's localized shortcut help is %q, drawn %v", info.Description, ok)
	}
	for _, gone := range []string{"New session", "Search sessions", "Scheduled tasks", "Skills and plugins", "Sign in to Dim", "Session keys", "Attach file", "Unavailable capabilities"} {
		if tt.HasText(gone) {
			t.Errorf("the zh-CN frame still shows the English %q, texts %q", gone, tt.Texts())
		}
	}
	// Session titles and bridge facts are not the shell's words.
	if !tt.HasText("First session") {
		t.Error("the zh-CN frame dropped the caller's session title")
	}
	if !tt.HasText("session.create") {
		t.Error("the zh-CN frame translated the wire method session.create")
	}
	// The sidebar's own entry and the capability card both carry "会话键"; the
	// entry must still be a button, the card read-only text.
	if _, ok := tt.AccessByRole(ui.RoleButton, "会话键"); !ok {
		t.Error("the zh-CN sidebar lost its keys entry")
	}
	if _, ok := tt.AccessByRole(ui.RoleText, "创建会话"); !ok {
		t.Error("the zh-CN capability card is not read-only text")
	}
}

// TestMissingKeyFallsBackToEnglish checks the lookup directly: a key a
// language does not carry draws the English text, and a key neither table
// carries draws the key itself rather than an empty row.
func TestMissingKeyFallsBackToEnglish(t *testing.T) {
	var empty Strings
	if got, want := empty.text(keySidebarNewSession), "New session"; got != want {
		t.Errorf("a missing key draws %q, want the English %q", got, want)
	}
	blank := Strings{keySidebarNewSession: ""}
	if got, want := blank.text(keySidebarNewSession), "New session"; got != want {
		t.Errorf("a blank entry draws %q, want the English %q", got, want)
	}
	if got, want := empty.text("no.such.key"), "no.such.key"; got != want {
		t.Errorf("an unknown key draws %q, want the key itself %q", got, want)
	}
	if got := stringsFor(Locale("fr")); got == nil || got.text(keyComposerSend) != "Send" {
		t.Errorf("an unknown locale does not fall back to English")
	}
}

// TestProtocolNamesStayLiteral checks the boundary of the translation: a WIP
// reason may say what is missing in the reader's language, but the method it
// names is a wire name and stays literal. Capability cards keep the method in
// every language, and every translated reason differs from its English entry,
// so a key added without a translation does not pass as done.
func TestProtocolNamesStayLiteral(t *testing.T) {
	wire := []struct {
		key  string
		name string
	}{
		{keyReasonCreateSession, "session.create"},
		{keyReasonApproval, "approval.respond"},
		{keyReasonAttachment, "attachment.put"},
		{keyReasonReconnect, "supervisor"},
		{keyReasonStop, "session.stop"},
		{keyReasonKeysEntry, "session.keys"},
		{keyReasonEndBranch, "session.branch_end"},
		{keyReasonRefreshBranches, "session.branches"},
		{keyReasonRename, "session.rename"},
	}
	for _, tc := range wire {
		got := tr(LocaleZHCN, tc.key)
		if !strings.Contains(got, tc.name) {
			t.Errorf("the zh-CN reason %q no longer names %q: %q", tc.key, tc.name, got)
		}
		if !strings.HasPrefix(got, "不可用：") {
			t.Errorf("the zh-CN reason %q is not translated: %q", tc.key, got)
		}
	}

	translated := []string{
		keyReasonCreateSession, keyReasonApproval, keyReasonAttachment, keyReasonReconnect,
		keyReasonStop, keyReasonBranches, keyReasonUpdate, keyReasonSearch, keyReasonGroup,
		keyReasonPin, keyReasonRename, keyReasonArchive, keyReasonKeysEntry, keyReasonEndBranch,
		keyReasonRefreshBranches, keyReasonModel, keyReasonMode, keyReasonSkin, keyReasonChrome,
	}
	for _, key := range translated {
		if tr(LocaleZHCN, key) == tr(LocaleEN, key) {
			t.Errorf("the zh-CN entry for %q is the English text, so it was never translated", key)
		}
	}

	s := fixture()
	s.Settings = SettingsView{Locale: LocaleZHCN}
	tt := ui.NewTester(s.View, 900, 600)
	for _, capability := range wipCapabilities(s) {
		if !tt.HasText(capability.method) {
			t.Errorf("the zh-CN card for %q is missing the literal method %q, texts %q",
				capability.action, capability.method, tt.Texts())
		}
		if got := tr(LocaleZHCN, capability.actionKey); got == capability.action {
			t.Errorf("the zh-CN card for %q is still English: %q", capability.action, got)
		}
	}
	if !tt.HasText(tr(LocaleZHCN, keyCapabilityCreateGap)) {
		t.Errorf("the zh-CN cards are missing the translated gap %q, texts %q",
			tr(LocaleZHCN, keyCapabilityCreateGap), tt.Texts())
	}
}

// TestSettingsLanguageChoiceShowsOnTheNextFrame checks the settings row end to
// end: the page names the current language, a choice reports the preference
// and the same window draws the next frame in it, without the caller
// rebuilding the state.
func TestSettingsLanguageChoiceShowsOnTheNextFrame(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 900)

	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Interface language", "Current: English", "English", "简体中文"} {
		if !tt.HasText(want) {
			t.Fatalf("the language row is missing %q, texts %q", want, tt.Texts())
		}
	}

	if err := tt.Click("简体中文"); err != nil {
		t.Fatal(err)
	}
	if s.Settings.Locale != LocaleZHCN {
		t.Fatalf("the surface reported locale %q, want %q", s.Settings.Locale, LocaleZHCN)
	}
	for _, want := range []string{"外观", "语言", "界面语言", "当前：简体中文"} {
		if !tt.HasText(want) {
			t.Errorf("the next frame is missing %q, texts %q", want, tt.Texts())
		}
	}
	if tt.HasText("Interface language") {
		t.Errorf("the next frame still draws the English language row, texts %q", tt.Texts())
	}

	// Another preference carries the language along: choosing a font size must
	// not silently switch the page back to English.
	if err := tt.Click("12"); err != nil {
		t.Fatal(err)
	}
	if s.Settings.Locale != LocaleZHCN || s.Settings.FontSize != 12 {
		t.Fatalf("after choosing 12 the settings are %+v, want zh-CN at 12", s.Settings)
	}
	if !tt.HasText("界面语言") {
		t.Errorf("choosing a font size dropped the Chinese language, texts %q", tt.Texts())
	}

	// The control carries each language's own name, so the way back needs no
	// translation: the English segment stays "English" on a Chinese page.
	if err := tt.Click("English"); err != nil {
		t.Fatal(err)
	}
	if s.Settings.Locale != LocaleEN {
		t.Fatalf("the surface reported locale %q after choosing English, want %q", s.Settings.Locale, LocaleEN)
	}
	if !tt.HasText("Appearance") || !tt.HasText("Interface language") {
		t.Errorf("the frame did not return to English, texts %q", tt.Texts())
	}
}

// TestEnglishTableMatchesConstants pins the English table to the shell's
// constants, so the default frame keeps exactly the text the rest of the tests
// assert and a constant cannot drift from its message entry.
func TestEnglishTableMatchesConstants(t *testing.T) {
	for _, tc := range []struct{ key, want string }{
		{keyNoSession, "No session"},
		{keyComposerPlaceholder, composerPlaceholder},
		{keyComposerHint, composerHint},
		{keySidebarUngrouped, ungroupedLabel},
		{keyReasonCreateSession, reasonCreateSession},
		{keyReasonApproval, reasonApproval},
		{keyReasonAttachment, reasonAttachment},
		{keyReasonReconnect, reasonReconnect},
		{keyReasonStop, reasonStop},
		{keyReasonBranches, reasonBranches},
		{keyReasonUpdate, reasonUpdate},
		{keyReasonSearch, reasonSearch},
		{keyReasonGroup, reasonGroup},
		{keyReasonPin, reasonPin},
		{keyReasonRename, reasonRename},
		{keyReasonArchive, reasonArchive},
		{keyReasonKeysEntry, reasonKeysEntry},
		{keyReasonEndBranch, reasonEndBranch},
		{keyReasonRefreshBranches, reasonRefreshBranches},
		{keyReasonModel, reasonModel},
		{keyReasonMode, reasonMode},
		{keyReasonSkin, reasonSkin},
		{keyReasonChrome, reasonChrome},
		{keySettingsLocalNote, settingsLocalNote},
		{keySettingsSkinNote, settingsSkinNote},
	} {
		if got := stringsEN.text(tc.key); got != tc.want {
			t.Errorf("the English entry %q is %q, want the constant %q", tc.key, got, tc.want)
		}
	}
	if wipBadge != "WIP" {
		t.Errorf("the WIP badge is %q; it stays literal in every language", wipBadge)
	}
}

// TestLocaleTablesStayInStep checks the two tables key by key: every English
// key has a non-empty translation, and no translation names a key the English
// table does not define. A missing key still falls back at run time, but this
// keeps the fallback a safety net rather than the normal path.
func TestLocaleTablesStayInStep(t *testing.T) {
	for key := range stringsEN {
		if !stringsZHCN.hasText(key) {
			t.Errorf("zh-CN has no translation for %q", key)
		}
	}
	for key := range stringsZHCN {
		if !stringsEN.hasText(key) {
			t.Errorf("zh-CN defines %q, which English does not", key)
		}
	}
	for key, text := range stringsEN {
		if strings.TrimSpace(text) == "" {
			t.Errorf("the English entry %q is empty", key)
		}
		if text == key {
			t.Errorf("the English entry %q draws its own key", key)
		}
	}
}
