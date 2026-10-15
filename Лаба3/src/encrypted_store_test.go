package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreDoesNotWritePlaintext(t *testing.T) {
	s := openTestStore(t)
	if err := s.AddUser(User{Username: "private-student"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"ADMIN", "private-student", "password_hash"} {
		if bytes.Contains(data, []byte(text)) {
			t.Fatalf("account data leaked to disk: %s", text)
		}
	}
}

func TestEncryptedStoreWrongPhraseLeavesFileUnchanged(t *testing.T) {
	s := openTestStore(t)
	before, err := os.ReadFile(s.storePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{"wrong", ""} {
		rejected := &Store{}
		if err := rejected.OpenStore(s.storePath, phrase); err == nil {
			t.Fatal("invalid phrase accepted")
		}
		if err := rejected.Close(); err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(s.storePath)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("failed unlock changed encrypted file")
		}
		if len(rejected.All()) != 0 {
			t.Fatal("failed unlock exposed accounts")
		}
	}
}

func TestEncryptedStoreCloseReencryptsAndReopens(t *testing.T) {
	s := openTestStore(t)
	if err := s.AddUser(User{Username: "student", RestrictionsEnabled: true, Blocked: true}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.storePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if len(s.All()) != 0 || s.cipher != nil {
		t.Fatal("closed store retains accounts or keys")
	}
	after, err := os.ReadFile(s.storePath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after) {
		t.Fatal("exit did not reencrypt with a fresh IV")
	}
	reopened := &Store{}
	if err := reopened.OpenStore(s.storePath, testPassphrase); err != nil {
		t.Fatal(err)
	}
	user, err := reopened.Get("student")
	if err != nil || !user.Blocked || !user.RestrictionsEnabled {
		t.Fatalf("lost user parameters: %+v %v", user, err)
	}
	if err := s.AddUser(User{Username: "closed"}); err == nil {
		t.Fatal("closed store accepts writes")
	}
	files, err := os.ReadDir(filepath.Dir(s.storePath))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("temporary files left behind: %v", files)
	}
}

func TestEncryptedStoreRequiresValidAdmin(t *testing.T) {
	for _, plaintext := range []string{`{}`, `null`, `{"ADMIN":null}`, `{"ADMIN":{"username":"other"}}`, `{"ADMIN":{"username":"ADMIN","blocked":true}}`, `{"ADMIN":{"username":"ADMIN","password_hash":"broken"}}`} {
		t.Run(plaintext, func(t *testing.T) {
			c, err := newFileCipher(testPassphrase)
			if err != nil {
				t.Fatal(err)
			}
			encrypted, err := c.seal([]byte(plaintext))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "users.enc")
			if err := os.WriteFile(path, encrypted, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := (&Store{}).OpenStore(path, testPassphrase); err == nil {
				t.Fatal("invalid ADMIN accepted")
			}
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, encrypted) {
				t.Fatal("invalid file overwritten")
			}
		})
	}
}

func TestEncryptedStoreCloseFailureCanBeRetried(t *testing.T) {
	s := openTestStore(t)
	original := s.storePath
	s.storePath = t.TempDir()
	if err := s.Close(); err == nil {
		t.Fatal("failed close reported success")
	}
	if _, err := s.Get(AdminUsername); err != nil || s.cipher == nil {
		t.Fatal("failed close discarded data or key")
	}
	s.storePath = original
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
