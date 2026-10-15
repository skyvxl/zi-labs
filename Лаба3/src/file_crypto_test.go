package main

import (
	"bytes"
	"testing"
)

func TestFileCryptoRoundTrip(t *testing.T) {
	for _, plaintext := range [][]byte{nil, {0}, bytes.Repeat([]byte("яA%"), 257)} {
		c, err := newFileCipher("фраза с пробелами 🔐")
		if err != nil {
			t.Fatal(err)
		}
		data, err := c.seal(plaintext)
		if err != nil {
			t.Fatal(err)
		}
		_, got, err := decryptFile("фраза с пробелами 🔐", data)
		if err != nil || !bytes.Equal(got, plaintext) {
			t.Fatalf("round trip: %v", err)
		}
		if _, _, err := decryptFile("фраза с пробелами 🔐 ", data); err == nil {
			t.Fatal("passphrase was normalized")
		}
		other, err := c.seal(plaintext)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(data, other) {
			t.Fatal("IV reused")
		}
		second, err := newFileCipher("фраза с пробелами 🔐")
		if err != nil {
			t.Fatal(err)
		}
		if c.salt == second.salt || c.key == second.key {
			t.Fatal("salt does not change the key")
		}
	}
}

func TestFileCryptoRejectsTamperingAndTruncation(t *testing.T) {
	c, err := newFileCipher("test phrase")
	if err != nil {
		t.Fatal(err)
	}
	data, err := c.seal([]byte(`{"ADMIN":{"username":"ADMIN"}}`))
	if err != nil {
		t.Fatal(err)
	}
	for i := range data {
		damaged := bytes.Clone(data)
		damaged[i] ^= 1
		if _, _, err := decryptFile("test phrase", damaged); err == nil {
			t.Fatalf("accepted modified byte %d", i)
		}
		if _, _, err := decryptFile("test phrase", data[:i]); err == nil {
			t.Fatalf("accepted truncated length %d", i)
		}
	}
	if _, _, err := decryptFile("wrong", data); err == nil {
		t.Fatal("wrong passphrase accepted")
	}
	if _, _, err := decryptFile("", data); err == nil {
		t.Fatal("empty passphrase accepted")
	}
	if _, err := newFileCipher(""); err == nil {
		t.Fatal("empty passphrase creates a key")
	}
}
