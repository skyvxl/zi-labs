package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testPassphrase = "учебная парольная фраза"

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s := &Store{}
	if err := s.OpenStore(filepath.Join(t.TempDir(), "data", "users.enc"), testPassphrase); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStoreFirstRunAndPersistence(t *testing.T) {
	s := openTestStore(t)
	admin, err := s.Get(AdminUsername)
	if err != nil || admin.Blocked || !admin.RestrictionsEnabled || admin.PasswordHash != "" {
		t.Fatalf("initial admin: %+v, %v", admin, err)
	}
	if len(s.All()) != 1 {
		t.Fatal("first run must contain only ADMIN")
	}
	if err := s.AddUser(User{Username: "student", RestrictionsEnabled: true}); err != nil {
		t.Fatal(err)
	}
	admin.RestrictionsEnabled = false
	if err := s.UpdateUser(AdminUsername, admin); err != nil {
		t.Fatal(err)
	}
	reopened := &Store{}
	if err := reopened.OpenStore(s.storePath, testPassphrase); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.All(), reopened.All()) {
		t.Fatal("records not persisted")
	}
	userCopy, _ := s.Get(AdminUsername)
	userCopy.Blocked = true
	if current, _ := s.Get(AdminUsername); current.Blocked {
		t.Fatal("Get leaked mutable record")
	}
	info, _ := os.Stat(s.storePath)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions: %v", info.Mode())
	}
}

func TestStoreInvalidFileNeverOverwritten(t *testing.T) {
	for _, data := range []string{"", "{", "null", "{}", `{"ADMIN":null}`, `{"ADMIN":{"username":"other"}}`, `{"ADMIN":{"username":"ADMIN","blocked":true}}`, `{"ADMIN":{"username":"ADMIN","password_hash":"broken"}}`} {
		t.Run(data, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "users.enc")
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := (&Store{}).OpenStore(path, testPassphrase); err == nil {
				t.Fatal("invalid store accepted")
			}
			got, _ := os.ReadFile(path)
			if string(got) != data {
				t.Fatal("source overwritten")
			}
		})
	}
}

func TestStoreFailuresRollback(t *testing.T) {
	s := openTestStore(t)
	if err := s.AddUser(User{Username: AdminUsername}); !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatal(err)
	}
	if err := s.UpdateUser("missing", User{Username: "missing"}); !errors.Is(err, ErrUserNotFound) {
		t.Fatal(err)
	}
	if err := s.AddUser(User{Username: " "}); err == nil {
		t.Fatal("blank user accepted")
	}
	before := s.All()
	disk, _ := os.ReadFile(s.storePath)
	oldPath := s.storePath
	// A directory as the replacement target causes rename to fail even when running as root.
	s.storePath = t.TempDir()
	if err := s.AddUser(User{Username: "failed"}); err == nil {
		t.Fatal("save should fail")
	}
	admin, _ := s.Get(AdminUsername)
	admin.RestrictionsEnabled = false
	if err := s.UpdateUser(AdminUsername, admin); err == nil {
		t.Fatal("update should fail")
	}
	if !reflect.DeepEqual(before, s.All()) {
		t.Fatal("memory changed after failed save")
	}
	got, _ := os.ReadFile(oldPath)
	if !bytes.Equal(got, disk) {
		t.Fatal("original disk changed")
	}
}

func TestStoreLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.enc")
	first, err := LockStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LockStore(path); err == nil {
		t.Fatal("second instance allowed")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := LockStore(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = next.Close()
}
