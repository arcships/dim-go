package ui

// 临时出图：只用于本轮的截图比对，跑完即删（不属于交付的三个文件）。

import (
	"image/png"
	"os"
	"strconv"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestShotTemp(t *testing.T) {
	cases := []struct {
		name    string
		section settingsSectionID
		render  func(*gui.Context)
	}{
		{"notifications", secNotifications, renderNotifications},
		{"pets", secPets, renderPets},
		{"permissions", secPermissions, renderPermissions},
	}
	sizes := [][2]int{{1030, 725}, {720, 600}}
	for _, tc := range cases {
		for _, size := range sizes {
			section, render := tc.section, tc.render
			w, h := size[0], size[1]
			app := NewApp(Options{State: StateDefault})
			view := func(c *gui.Context) {
				app.screen, app.settingsSection = ScreenSettings, section
				c.SetTheme(theme(false, c.Preferences()))
				tm := c.Theme()
				gui.Row(c).Fill().AlignItems(gui.Stretch).Background(tm.Background).Label("设置").
					Children(func() {
						app.settingsNav(c)
						gui.Column(c).Grow(1).MinWidth(MainMinWidth).Children(func() {
							gui.Column(c).Grow(1).MaxWidth(SettingsContentMaxWidth).
								Padding(space8, space8).Gap(space4).Children(func() {
									render(c)
								})
						})
					})
			}
			tt := gui.NewTester(view, w, h)
			path := "/tmp/shot-" + tc.name + "-" + strconv.Itoa(w) + ".png"
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, tt.Image()); err != nil {
				t.Fatal(err)
			}
			_ = f.Close()
			t.Logf("%s @%d → %s", tc.name, w, path)
		}
	}
}
