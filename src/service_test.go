package main

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"
)

func newTestService(t *testing.T) *Service { t.Helper(); return NewService(openTestStore(t)) }
func loginAdmin(t *testing.T, s *Service) {
	t.Helper()
	if err := s.Login(AdminUsername, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SetInitialPassword("aя+", "aя+"); err != nil {
		t.Fatal(err)
	}
}

func TestServiceFirstLoginAndPasswordChange(t *testing.T) {
	s := newTestService(t)
	if err := s.Login("unknown", ""); !errors.Is(err, ErrUserNotFound) {
		t.Fatal(err)
	}
	if s.Attempts() != 0 {
		t.Fatal("unknown name counted")
	}
	if err := s.Login(AdminUsername, ""); err != nil {
		t.Fatal(err)
	}
	if s.State() != StateSetup {
		t.Fatal("first login bypassed setup")
	}
	if err := s.AddUser("forbidden"); !errors.Is(err, ErrAccessDenied) {
		t.Fatal(err)
	}
	if err := s.SetInitialPassword("abcЯ!", "abcЯ!"); err == nil {
		t.Fatal("bad composition accepted")
	}
	if err := s.SetInitialPassword("aя+", "aя-"); err == nil {
		t.Fatal("confirmation ignored")
	}
	admin, _ := s.store.Get(AdminUsername)
	if admin.PasswordHash != "" {
		t.Fatal("failed setup saved")
	}
	if err := s.SetInitialPassword("aя+", "aя+"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetInitialPassword("qЖ*", "qЖ*"); !errors.Is(err, ErrAccessDenied) {
		t.Fatal("stale setup allowed", err)
	}
	for _, c := range []struct{ old, next, confirm string }{{"wrong", "qЖ*", "qЖ*"}, {"aя+", "abcЯ!", "abcЯ!"}, {"aя+", "qЖ*", "bю/"}} {
		if err := s.ChangePassword(c.old, c.next, c.confirm); err == nil {
			t.Fatal("invalid change accepted")
		}
	}
	if s.Attempts() != 0 {
		t.Fatal("change errors counted as login errors")
	}
	if err := s.ChangePassword("aя+", "qЖ*", "qЖ*"); err != nil {
		t.Fatal(err)
	}
	s.Logout()
	if err := s.Login(AdminUsername, "aя+"); !errors.Is(err, ErrWrongPassword) {
		t.Fatal(err)
	}
	if err := s.Login(AdminUsername, "qЖ*"); err != nil {
		t.Fatal(err)
	}
}

func TestServiceAdminAndRestrictionWorkflow(t *testing.T) {
	s := newTestService(t)
	loginAdmin(t, s)
	if err := s.AddUser(" student "); err != nil {
		t.Fatal(err)
	}
	if err := s.AddUser("student"); !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatal(err)
	}
	if err := s.AddUser("\n"); err == nil {
		t.Fatal("blank name accepted")
	}
	if err := s.SetBlocked(AdminUsername, true); err == nil {
		t.Fatal("ADMIN locked out")
	}
	if err := s.SetBlocked("student", true); err != nil {
		t.Fatal(err)
	}
	s.Logout()
	if err := s.Login("student", ""); !errors.Is(err, ErrBlocked) {
		t.Fatal(err)
	}
	if s.Attempts() != 0 {
		t.Fatal("blocked login counted")
	}
	if err := s.Login(AdminUsername, "aя+"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetBlocked("student", false); err != nil {
		t.Fatal(err)
	}
	if err := s.SetRestrictions("student", false); err != nil {
		t.Fatal(err)
	}
	s.Logout()
	if err := s.Login("student", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SetInitialPassword("", ""); err == nil {
		t.Fatal("empty new password accepted")
	}
	if err := s.SetInitialPassword("123", "456"); err == nil {
		t.Fatal("disabled flag removed confirmation")
	}
	if err := s.SetInitialPassword("123", "123"); err != nil {
		t.Fatal(err)
	}
	for _, err := range []error{s.AddUser("hack"), s.SetBlocked(AdminUsername, true), s.SetRestrictions("student", false)} {
		if !errors.Is(err, ErrAccessDenied) {
			t.Fatal("ordinary user acquired admin rights", err)
		}
	}
	if _, err := s.Users(); !errors.Is(err, ErrAccessDenied) {
		t.Fatal(err)
	}
	s.Logout()
	_ = s.Login(AdminUsername, "aя+")
	_ = s.SetRestrictions("student", true)
	s.Logout()
	if err := s.Login("student", "123"); err != nil {
		t.Fatal("existing password rejected", err)
	}
	if err := s.ChangePassword("123", "456", "456"); err == nil {
		t.Fatal("new restriction ignored")
	}
	if err := s.ChangePassword("123", "mф%", "mф%"); err != nil {
		t.Fatal(err)
	}
	reopened := &Store{}
	if err := reopened.OpenStore(s.store.storePath); err != nil {
		t.Fatal(err)
	}
	again := NewService(reopened)
	if err := again.Login("student", "mф%"); err != nil {
		t.Fatal(err)
	}
}

func TestServiceThreeErrorsStopProcessState(t *testing.T) {
	s := newTestService(t)
	loginAdmin(t, s)
	s.Logout()
	for i := 1; i <= 3; i++ {
		if err := s.Login(AdminUsername, "wrong"); !errors.Is(err, ErrWrongPassword) {
			t.Fatal(err)
		}
		if s.Attempts() != i {
			t.Fatalf("attempts: %d", s.Attempts())
		}
		if i == 1 {
			if err := s.Login(AdminUsername, "aя+"); err != nil {
				t.Fatal(err)
			}
			s.Logout()
		}
	}
	if s.State() != StateStopped {
		t.Fatal("third wrong password did not stop app")
	}
	if err := s.Login(AdminUsername, "aя+"); !errors.Is(err, ErrStopped) {
		t.Fatal(err)
	}
	s.Logout()
	if s.State() != StateStopped {
		t.Fatal("logout revived stopped app")
	}
	admin, _ := s.store.Get(AdminUsername)
	if admin.Blocked {
		t.Fatal("third error permanently blocked account")
	}
	fresh := NewService(s.store)
	if err := fresh.Login(AdminUsername, "aя+"); err != nil {
		t.Fatal(err)
	}
}

func TestServiceFailedSaveAndCurrentRestriction(t *testing.T) {
	s := newTestService(t)
	loginAdmin(t, s)
	before := s.store.All()
	disk, _ := os.ReadFile(s.store.storePath)
	original := s.store.storePath
	s.store.storePath = t.TempDir()
	if err := s.ChangePassword("aя+", "qЖ*", "qЖ*"); err == nil {
		t.Fatal("failed save reported success")
	}
	if !reflect.DeepEqual(before, s.store.All()) {
		t.Fatal("password changed despite failed save")
	}
	got, _ := os.ReadFile(original)
	if !bytes.Equal(got, disk) {
		t.Fatal("disk changed")
	}
	s.store.storePath = original
	_ = s.AddUser("student")
	_ = s.SetRestrictions("student", false)
	s.Logout()
	_ = s.Login("student", "")
	// Simulate changed data after opening a setup form; submission must re-read the flag.
	user, _ := s.store.Get("student")
	user.RestrictionsEnabled = true
	_ = s.store.UpdateUser("student", user)
	if err := s.SetInitialPassword("123", "123"); err == nil {
		t.Fatal("stale restriction used")
	}
	user.Blocked = true
	_ = s.store.UpdateUser("student", user)
	if err := s.SetInitialPassword("aя+", "aя+"); !errors.Is(err, ErrBlocked) {
		t.Fatal(err)
	}
}
