package main

import (
	"strings"
	"testing"
)

func TestVariant15Matrix(t *testing.T) {
	cases := []struct {
		pass  string
		valid bool
	}{
		{"aя+", true},
		{"AЯ-", true},
		{"qЖ*", true},
		{"bю/", true},
		{"mф%", true},
		{"+яa", true},
		{"aaяя++", true},
		{"abcЯ123+!", true},
		{"abcЯ+ ", true},
		{"", false},
		{"abc", false},
		{"пароль", false},
		{"123", false},
		{"abc+", false},
		{"пароль+", false},
		{"abcЯ", false},
		{"abcЯ!", false},
		{"abcЯ=", false},
		{"abcЯ^", false},
		{"abcЯ×", false},
		{"abcЯ÷", false},
		{"abcЯ−", false},
		{"Aa+", false},
		{"Аа+", false},
		{"aа+", true},
		{"AaАа1!", false},
		{"aЯ++", true},
		{"aя+б", true},
		{"aЁ+", true},
		{"aё+", true},
		{"aЇ+", true},
		{"éя+", true},
		{"aα+", false},
		{"a\u0483+", false},
	}
	for _, c := range cases {
		t.Run(c.pass, func(t *testing.T) {
			err := ValidatePolicy(User{RestrictionsEnabled: true}, c.pass)
			if (err == nil) != c.valid {
				t.Fatalf("password %q: %v", c.pass, err)
			}
			if err := ValidatePolicy(User{}, c.pass); err != nil {
				t.Fatal("disabled restriction:", err)
			}
		})
	}
	for _, c := range []struct{ pass, missing string }{{"я+", "латинская"}, {"a+", "кириллицы"}, {"aя!", "+ - * / %"}} {
		err := ValidatePolicy(User{RestrictionsEnabled: true}, c.pass)
		if err == nil || !strings.Contains(err.Error(), c.missing) {
			t.Fatalf("missing explanation %s: %v", c.missing, err)
		}
	}
}
