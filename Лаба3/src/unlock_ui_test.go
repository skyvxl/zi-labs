package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func newLockedTestUI(t *testing.T, path string, firstRun bool) *UI {
	t.Helper()
	app := test.NewApp()
	app.Settings().SetTheme(lightTheme{Theme: theme.DefaultTheme()})
	t.Cleanup(app.Quit)
	ui := NewUI(app, NewService(&Store{}), path)
	ui.window.Resize(fyne.NewSize(860, 600))
	ui.ShowUnlock(firstRun)
	return ui
}

func TestUICreateUnlockAndCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.enc")
	ui := newLockedTestUI(t, path, true)
	if !ui.usersItem.Disabled || !ui.changeItem.Disabled {
		t.Fatal("locked menu grants access")
	}
	test.Type(ui.passphrase, "phrase")
	test.Type(ui.passphraseConfirm, "different")
	test.Tap(ui.unlockSubmit)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("confirmation mismatch created file")
	}
	if ui.message.Text == "" {
		t.Fatal("confirmation mismatch not explained")
	}
	test.Type(ui.passphrase, testPassphrase)
	test.Type(ui.passphraseConfirm, testPassphrase)
	test.Tap(ui.unlockSubmit)
	if ui.loginName == nil || ui.service.store.cipher == nil {
		t.Fatal("first unlock did not open login")
	}
	if ui.passphrase.Value() != "" || ui.passphraseConfirm.Value() != "" {
		t.Fatal("phrase retained in fields")
	}
	calls := 0
	ui.quit = func() { calls++ }
	ui.finish()
	if calls != 1 {
		t.Fatal("close did not quit")
	}
	again := newLockedTestUI(t, path, false)
	if again.passphraseConfirm != nil {
		t.Fatal("existing file asks to create a phrase")
	}
	test.Type(again.passphrase, testPassphrase)
	test.Tap(again.unlockSubmit)
	if again.loginName == nil {
		t.Fatal("existing file did not unlock")
	}
	if err := again.service.Login(AdminUsername, ""); err != nil {
		t.Fatal(err)
	}
	cancelPath := filepath.Join(t.TempDir(), "cancelled.enc")
	cancelled := newLockedTestUI(t, cancelPath, true)
	cancelled.quit = func() { calls++ }
	cancelled.finish()
	if calls != 1 || cancelled.service.State() != StateStopped {
		t.Fatal("cancel did not show a terminal message")
	}
	if cancelled.passphrase.Value() != "" || cancelled.stopAcknowledge == nil {
		t.Fatal("cancel retains phrase or lacks acknowledgement")
	}
	test.Tap(cancelled.stopAcknowledge)
	if calls != 2 {
		t.Fatal("cancel acknowledgement did not quit")
	}
	if _, err := os.Stat(cancelPath); !os.IsNotExist(err) {
		t.Fatal("cancel created a file")
	}
}

func TestUIWrongPhraseStopsWithoutChangingFile(t *testing.T) {
	s := openTestStore(t)
	before, err := os.ReadFile(s.storePath)
	if err != nil {
		t.Fatal(err)
	}
	ui := newLockedTestUI(t, s.storePath, false)
	calls := 0
	ui.quit = func() { calls++ }
	test.Type(ui.passphrase, "wrong")
	test.Tap(ui.unlockSubmit)
	if ui.service.State() != StateStopped || ui.loginName != nil || calls != 0 {
		t.Fatal("failed unlock did not display terminal message")
	}
	if ui.passphrase.Value() != "" || !ui.unlockSubmit.Disabled() {
		t.Fatal("failed unlock retains phrase or allows retry")
	}
	test.Tap(ui.stopAcknowledge)
	after, err := os.ReadFile(s.storePath)
	if calls != 1 || err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed unlock changed data or did not quit")
	}
}

func TestUIUnlockScreenshots(t *testing.T) {
	dir := os.Getenv("LAB3_SCREENSHOTS")
	if dir == "" {
		t.Skip("set LAB3_SCREENSHOTS to export forms")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "users.enc")
	ui := newLockedTestUI(t, path, true)
	capture := func(name string) {
		t.Helper()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, ui.window.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	capture("00-create-file")
	test.Type(ui.passphrase, testPassphrase)
	test.Type(ui.passphraseConfirm, testPassphrase)
	test.Tap(ui.unlockSubmit)
	ui = newLockedTestUI(t, path, false)
	capture("00-open-file")
	test.Type(ui.passphrase, "wrong")
	test.Tap(ui.unlockSubmit)
	capture("00-wrong-phrase")
}
