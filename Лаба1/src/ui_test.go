package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func newTestUI(t *testing.T) *UI {
	t.Helper()
	app := test.NewApp()
	app.Settings().SetTheme(lightTheme{Theme: theme.DefaultTheme()})
	t.Cleanup(app.Quit)
	ui := NewUI(app, newTestService(t), "test-users.json")
	ui.window.Resize(fyne.NewSize(860, 600))
	ui.ShowLogin()
	return ui
}

func TestUIFirstLoginMenusAndAdmin(t *testing.T) {
	ui := newTestUI(t)
	if !ui.usersItem.Disabled || !ui.changeItem.Disabled {
		t.Fatal("guest menu grants access")
	}
	test.Type(ui.loginName, AdminUsername)
	test.Tap(ui.loginSubmit)
	if ui.service.State() != StateSetup || !ui.usersItem.Disabled {
		t.Fatal("setup permissions incorrect")
	}
	test.Type(ui.setupPassword, "abcЯ!")
	test.Type(ui.setupConfirm, "abcЯ!")
	test.Tap(ui.setupSubmit)
	if ui.service.State() != StateSetup || ui.message.Text == "" {
		t.Fatal("bad setup accepted")
	}
	test.Type(ui.setupPassword, "aя+")
	test.Type(ui.setupConfirm, "aя+")
	test.Tap(ui.setupSubmit)
	if ui.service.State() != StateAuthenticated || ui.usersItem.Disabled {
		t.Fatal("admin menu not enabled")
	}
	ui.showNewUser()
	test.Type(ui.newName, "student")
	test.Tap(ui.newSubmit)
	if _, err := ui.service.store.Get("student"); err != nil {
		t.Fatal(err)
	}
	ui.showUsers()
	if len(ui.visibleUsers) != 2 {
		t.Fatal("users missing")
	}
	ui.service.Logout()
	ui.ShowLogin()
	test.Type(ui.loginName, "student")
	test.Tap(ui.loginSubmit)
	test.Type(ui.setupPassword, "qЖ*")
	test.Type(ui.setupConfirm, "qЖ*")
	test.Tap(ui.setupSubmit)
	if !ui.usersItem.Disabled || !ui.newItem.Disabled || ui.changeItem.Disabled {
		t.Fatal("ordinary permissions incorrect")
	}
	ui.showChangePassword()
	test.Type(ui.oldPassword, "qЖ*")
	test.Type(ui.newPassword, "mф%")
	test.Type(ui.passwordConfirm, "mф%")
	test.Tap(ui.passwordSubmit)
	ui.service.Logout()
	if err := ui.service.Login("student", "mф%"); err != nil {
		t.Fatal(err)
	}
}

func TestUIThirdErrorQuits(t *testing.T) {
	ui := newTestUI(t)
	loginAdmin(t, ui.service)
	ui.service.Logout()
	ui.ShowLogin()
	calls := 0
	ui.quit = func() { calls++ }
	for i := 0; i < 3; i++ {
		ui.loginName.SetText(AdminUsername)
		test.Type(ui.loginPassword, "wrong")
		test.Tap(ui.loginSubmit)
	}
	if ui.service.State() != StateStopped || calls != 0 {
		t.Fatal("terminal message not shown")
	}
	if !ui.loginSubmit.Disabled() || !ui.changeItem.Disabled || !ui.usersItem.Disabled {
		t.Fatal("stopped ui allows actions")
	}
	test.Tap(ui.stopAcknowledge)
	if calls != 1 {
		t.Fatal("third error does not quit app")
	}
}

// Optional reproducible images for visual review and the lab report.
func TestUIScreenshots(t *testing.T) {
	dir := os.Getenv("LAB1_SCREENSHOTS")
	if dir == "" {
		t.Skip("set LAB1_SCREENSHOTS to export forms")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ui := newTestUI(t)
	capture := func(name string) {
		t.Helper()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, ui.window.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
	capture("01-login")
	test.Type(ui.loginName, AdminUsername)
	test.Tap(ui.loginSubmit)
	test.Type(ui.setupPassword, "aя+")
	test.Type(ui.setupConfirm, "aя+")
	capture("02-setup")
	ui.setupPassword.Clear()
	ui.setupConfirm.Clear()
	test.Type(ui.setupPassword, "aя+")
	test.Type(ui.setupConfirm, "aя+")
	test.Tap(ui.setupSubmit)
	_ = ui.service.AddUser("student")
	_ = ui.service.AddUser("blocked")
	_ = ui.service.SetBlocked("blocked", true)
	capture("03-admin")
	ui.showUsers()
	capture("04-users")
	ui.activeDialog.Hide()
	ui.showChangePassword()
	test.Type(ui.oldPassword, "aя+")
	test.Type(ui.newPassword, "abcЯ!")
	test.Type(ui.passwordConfirm, "abcЯ!")
	test.Tap(ui.passwordSubmit)
	capture("05-policy-error")
	ui.activeDialog.Hide()
	ui.showAbout()
	capture("06-about")
	ui.activeDialog.Hide()
	ui.service.Logout()
	ui.ShowLogin()
	test.Type(ui.loginName, "student")
	test.Tap(ui.loginSubmit)
	test.Type(ui.setupPassword, "mф%")
	test.Type(ui.setupConfirm, "mф%")
	test.Tap(ui.setupSubmit)
	capture("07-user")
}
