package main

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	VariantNumber      = 15
	VariantDescription = "Наличие латинских букв, символов кириллицы и знаков арифметических операций"
	PolicyHint         = "Нужны латинская буква, символ кириллицы и один из знаков: + - * / %"
)

func ValidatePolicy(user User, password string) error {
	if !user.RestrictionsEnabled {
		return nil
	}
	var latin, cyrillic, arithmetic bool
	for _, char := range password {
		latin = latin || (unicode.IsLetter(char) && unicode.In(char, unicode.Latin))
		cyrillic = cyrillic || (unicode.IsLetter(char) && unicode.In(char, unicode.Cyrillic))
		arithmetic = arithmetic || strings.ContainsRune("+-*/%", char)
	}
	var missing []string
	if !latin {
		missing = append(missing, "латинская буква")
	}
	if !cyrillic {
		missing = append(missing, "символ кириллицы")
	}
	if !arithmetic {
		missing = append(missing, "один из знаков + - * / %")
	}
	if len(missing) != 0 {
		return fmt.Errorf("пароль не соответствует ограничениям. Не хватает: %s", strings.Join(missing, ", "))
	}
	return nil
}
