package main

import "errors"

const AdminUsername = "ADMIN"

var (
	ErrUserAlreadyExists = errors.New("пользователь с таким именем уже существует")
	ErrUserNotFound      = errors.New("пользователь не зарегистрирован")
	ErrBlocked           = errors.New("учётная запись заблокирована")
	ErrAccessDenied      = errors.New("недостаточно прав для этой операции")
	ErrWrongPassword     = errors.New("неверный пароль")
	ErrStopped           = errors.New("программа завершена")
)

type User struct {
	Username            string `json:"username"`
	PasswordHash        string `json:"password_hash"`
	Blocked             bool   `json:"blocked"`
	RestrictionsEnabled bool   `json:"restrictions_enabled"`
}
