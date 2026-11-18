package ui

import (
	"image/png"
	"os"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 临时取证：把设置 › 账户屏渲染成 PNG（用完即删，不进仓库）。
func TestZZAccountShot(t *testing.T) {
	cases := []struct {
		name  string
		width int
		dark  bool
		view  func(c *gui.Context)
	}{
		{"account-wide", WindowWidth, false, accountShotView(nil)},
		{"account-narrow", 900, false, accountShotView(nil)},
		{"account-dark", WindowWidth, true, accountShotView(nil)},
		{"account-dialog", WindowWidth, false, accountShotView(func(a *App) { a.accountEmailOpen = true })},
		{"account-reset", WindowWidth, false, accountShotView(func(a *App) { a.accountResetType = "monthly" })},
		{"account-expired", WindowWidth, false, func(c *gui.Context) {
			app := NewApp(Options{State: StateDefault})
			data := accountSampleData()
			data.PlanExpired = true
			data.Depleted = true
			data.RenewalDays = -3
			gui.Column(c).Padding(space8).Gap(accSpace5).Children(func() {
				app.accountExpiryBanner(c, data)
				app.accountIdentityCard(c, data)
				app.accountPlanCard(c, data)
			})
		}},
		{"account-unlimited", WindowWidth, false, func(c *gui.Context) {
			app := NewApp(Options{State: StateDefault})
			data := accountSampleData()
			data.Unlimited = true
			data.CreditsKnown = false
			data.Features = nil
			data.Windows = []accountWindow{{Label: "5 小时窗口", Amount: "0.4万 / 3万", Percent: 12,
				ResetAt: data.RenewalAt}}
			gui.Column(c).Padding(space8).Gap(accSpace5).Children(func() {
				app.accountPlanCard(c, data)
			})
		}},
	}
	for _, tc := range cases {
		if tc.dark {
			tc.view = accountShotViewDark()
		}
		tt := gui.NewTester(tc.view, tc.width, WindowHeight)
		f, err := os.Create("/tmp/" + tc.name + ".png")
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

func accountShotViewDark() func(c *gui.Context) {
	return func(c *gui.Context) {
		app := NewApp(Options{State: StateDefault})
		app.themePref = "暗"
		app.screen, app.settingsSection = ScreenSettings, secAccount
		app.View(c)
	}
}

func accountShotView(setup func(*App)) func(c *gui.Context) {
	return func(c *gui.Context) {
		app := NewApp(Options{State: StateDefault})
		app.screen, app.settingsSection = ScreenSettings, secAccount
		if setup != nil {
			setup(app)
		}
		app.View(c)
	}
}
