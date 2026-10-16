package main

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestPasswordUpdatesRequireFreshLogin(t *testing.T) {
	s := newTestService(t)
	if err := s.Login(AdminUsername, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SetInitialPassword("aя+", "aя+"); err != nil {
		t.Fatal(err)
	}
	if s.State() != StateGuest {
		t.Fatal("initial password automatically logged in")
	}
	if _, err := s.CurrentUser(); !errors.Is(err, ErrAccessDenied) {
		t.Fatal("initial setup retained identity")
	}
	if err := s.AddUser("forbidden"); !errors.Is(err, ErrAccessDenied) {
		t.Fatal("setup grants administrative access")
	}
	if err := s.Login(AdminUsername, "aя+"); err != nil {
		t.Fatal(err)
	}
	if err := s.ChangePassword("aя+", "mф%", "mф%"); err != nil {
		t.Fatal(err)
	}
	if s.State() != StateGuest {
		t.Fatal("password change retained session")
	}
	if err := s.Login(AdminUsername, "aя+"); !errors.Is(err, ErrWrongPassword) {
		t.Fatal("old password still accepted")
	}
	if err := s.Login(AdminUsername, "mф%"); err != nil || s.State() != StateAuthenticated {
		t.Fatalf("fresh login: %v", err)
	}
}

func TestEnabledPolicyForcesReplacementBeforeAccess(t *testing.T) {
	for _, username := range []string{"student", AdminUsername} {
		t.Run(username, func(t *testing.T) {
			s := newTestService(t)
			user := User{Username: username, RestrictionsEnabled: true}
			var err error
			user.PasswordHash, err = HashPassword("123")
			if err != nil {
				t.Fatal(err)
			}
			if username == AdminUsername {
				err = s.store.UpdateUser(username, user)
			} else {
				err = s.store.AddUser(user)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Login(username, "wrong"); !errors.Is(err, ErrWrongPassword) {
				t.Fatal(err)
			}
			if err := s.Login(username, "123"); err != nil {
				t.Fatal(err)
			}
			if s.State() == StateAuthenticated || s.State() == StateGuest {
				t.Fatal("noncompliant password bypassed mandatory change")
			}
			for _, err := range []error{s.AddUser("hack"), s.SetRestrictions(username, false)} {
				if !errors.Is(err, ErrAccessDenied) {
					t.Fatal("mandatory change grants admin rights", err)
				}
			}
			if _, err := s.authorizedUser(); !errors.Is(err, ErrAccessDenied) {
				t.Fatal("mandatory change grants normal access")
			}
			if err := s.ChangePassword("123", "456", "456"); err == nil {
				t.Fatal("noncompliant replacement accepted")
			}
			if err := s.ChangePassword("123", "aя+", "qЖ*"); err == nil {
				t.Fatal("confirmation ignored")
			}
			if err := s.ChangePassword("123", "aя+", "aя+"); err != nil {
				t.Fatal(err)
			}
			if s.State() != StateGuest {
				t.Fatal("mandatory change logged in")
			}
			if s.Attempts() != 1 {
				t.Fatal("policy mismatch counted as login failure")
			}
			if err := s.Login(username, "aя+"); err != nil || s.State() != StateAuthenticated {
				t.Fatalf("fresh login: %v", err)
			}
		})
	}
}

func TestUIPasswordSetupReturnsToEmptyLogin(t *testing.T) {
	ui := newTestUI(t)
	test.Type(ui.loginName, AdminUsername)
	test.Tap(ui.loginSubmit)
	test.Type(ui.setupPassword, "aя+")
	test.Type(ui.setupConfirm, "aя+")
	test.Tap(ui.setupSubmit)
	if ui.service.State() != StateGuest || ui.loginName.Text != "" || ui.loginPassword.Value() != "" || !ui.usersItem.Disabled {
		t.Fatal("setup did not return to an empty login form")
	}
}

func TestUIForcedChangeReturnsToEmptyLogin(t *testing.T) {
	ui := newTestUI(t)
	hash, err := HashPassword("123")
	if err != nil {
		t.Fatal(err)
	}
	if err := ui.service.store.AddUser(User{Username: "student", PasswordHash: hash, RestrictionsEnabled: true}); err != nil {
		t.Fatal(err)
	}
	test.Type(ui.loginName, "student")
	test.Type(ui.loginPassword, "123")
	test.Tap(ui.loginSubmit)
	if ui.setupPassword == nil || ui.service.State() == StateAuthenticated || !ui.changeItem.Disabled {
		t.Fatal("mandatory change form missing")
	}
	if ui.oldPassword == nil {
		t.Fatal("mandatory change has no old-password field")
	}
	test.Type(ui.oldPassword, "wrong")
	test.Type(ui.setupPassword, "aя+")
	test.Type(ui.setupConfirm, "aя+")
	test.Tap(ui.setupSubmit)
	if ui.service.State() != StatePasswordChange || ui.message.Text == "" {
		t.Fatal("wrong old password bypassed mandatory change")
	}
	if ui.oldPassword.Value() != "" || ui.setupPassword.Value() != "" || ui.setupConfirm.Value() != "" {
		t.Fatal("password fields were not cleared")
	}
	if dir := os.Getenv("LAB3_SCREENSHOTS"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "08-required-change.png"))
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
	test.Type(ui.oldPassword, "123")
	test.Type(ui.setupPassword, "aя+")
	test.Type(ui.setupConfirm, "aя+")
	test.Tap(ui.setupSubmit)
	if ui.service.State() != StateGuest || ui.loginName.Text != "" || ui.loginPassword.Value() != "" {
		t.Fatal("mandatory change did not reset login")
	}
	test.Type(ui.loginName, "student")
	test.Type(ui.loginPassword, "aя+")
	test.Tap(ui.loginSubmit)
	if ui.service.State() != StateAuthenticated {
		t.Fatal("fresh login failed")
	}
}

func TestTamperRestoreWorksAfterRejectedOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.enc")
	original := &Store{}
	if err := original.OpenStore(path, testPassphrase); err != nil {
		t.Fatal(err)
	}
	if err := original.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	position := len(data) / 2
	savedByte := data[position]
	data[position] ^= 1
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	rejected := &Store{}
	if err := rejected.OpenStore(path, testPassphrase); err == nil {
		t.Fatal("tamper accepted")
	}
	if err := rejected.Close(); err != nil {
		t.Fatal(err)
	}
	damaged, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	damaged[position] = savedByte
	if err := os.WriteFile(path, damaged, 0o600); err != nil {
		t.Fatal(err)
	}
	restored := &Store{}
	if err := restored.OpenStore(path, testPassphrase); err != nil {
		t.Fatalf("restoration failed: %v", err)
	}
	if _, err := restored.Get(AdminUsername); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredPasswordFailedSaveKeepsChangePending(t *testing.T) {
	s := newTestService(t)
	hash, err := HashPassword("123")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.store.AddUser(User{Username: "student", PasswordHash: hash, RestrictionsEnabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Login("student", "123"); err != nil {
		t.Fatal(err)
	}
	path := s.store.storePath
	s.store.storePath = t.TempDir()
	if err := s.ChangePassword("123", "aя+", "aя+"); err == nil {
		t.Fatal("failed save reported success")
	}
	if s.State() != StatePasswordChange {
		t.Fatal("failed save removed required change")
	}
	user, err := s.store.Get("student")
	if err != nil || user.PasswordHash != hash {
		t.Fatal("failed save changed password")
	}
	s.store.storePath = path
	if err := s.ChangePassword("123", "aя+", "aя+"); err != nil {
		t.Fatal(err)
	}
	if s.State() != StateGuest {
		t.Fatal("retry did not require fresh login")
	}
}

func TestForcedChangeRequiresOldPassword(t *testing.T) {
	s := newTestService(t)
	hash, err := HashPassword("123")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.store.AddUser(User{Username: "student", PasswordHash: hash, RestrictionsEnabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Login("student", "123"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetInitialPassword("aя+", "aя+"); !errors.Is(err, ErrAccessDenied) {
		t.Fatal("mandatory change can bypass old-password verification")
	}
	before, err := os.ReadFile(s.store.storePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, old := range []string{"", "wrong"} {
		if err := s.ChangePassword(old, "aя+", "aя+"); err == nil {
			t.Fatal("wrong old password accepted")
		}
		user, err := s.store.Get("student")
		if err != nil || user.PasswordHash != hash || s.State() != StatePasswordChange {
			t.Fatal("failed change modified account or state")
		}
		after, err := os.ReadFile(s.store.storePath)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("failed change wrote to disk")
		}
	}
	if s.Attempts() != 0 {
		t.Fatal("change failure counted as login failure")
	}
	if err := s.ChangePassword("123", "aя+", "aя+"); err != nil {
		t.Fatal(err)
	}
	if s.State() != StateGuest {
		t.Fatal("change did not require fresh login")
	}
	if err := s.Login("student", "aя+"); err != nil || s.State() != StateAuthenticated {
		t.Fatalf("fresh login: %v", err)
	}
}
