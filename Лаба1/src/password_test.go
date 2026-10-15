package main

import "testing"

func TestPasswordHash(t *testing.T) {
	first, err := HashPassword("aя+ ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword("aя+ ")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("salt reused")
	}
	for _, c := range []struct {
		pass string
		want bool
	}{{"aя+ ", true}, {"aя+", false}, {"wrong", false}} {
		got, err := VerifyPassword(first, c.pass)
		if err != nil || got != c.want {
			t.Fatalf("verify %q: %v %v", c.pass, got, err)
		}
	}
	for _, data := range []string{"", "broken", "$argon2id$v=19$m=999999999,t=2,p=1$abc$abc", first + "!"} {
		if _, err := VerifyPassword(data, "aя+ "); err == nil {
			t.Fatalf("invalid hash accepted: %s", data)
		}
	}
}
