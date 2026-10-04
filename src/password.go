package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/argon2"
)

const hashPrefix = "$argon2id$v=19$m=19456,t=2,p=1$"

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, 2, 19456, 1, 32)
	return hashPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}

// Only our fixed profile is accepted: damaged data cannot request arbitrary memory.
func decodeHash(encoded string) (salt, key []byte, err error) {
	if !strings.HasPrefix(encoded, hashPrefix) {
		return nil, nil, errors.New("неподдерживаемый формат хеша пароля")
	}
	parts := strings.Split(strings.TrimPrefix(encoded, hashPrefix), "$")
	if len(parts) != 2 || len(parts[0]) != 22 || len(parts[1]) != 43 {
		return nil, nil, errors.New("повреждённый хеш пароля")
	}
	salt, e1 := base64.RawStdEncoding.Strict().DecodeString(parts[0])
	key, e2 := base64.RawStdEncoding.Strict().DecodeString(parts[1])
	if e1 != nil || e2 != nil || len(salt) != 16 || len(key) != 32 {
		return nil, nil, errors.New("повреждённый хеш пароля")
	}
	return salt, key, nil
}

func VerifyPassword(encoded, password string) (bool, error) {
	salt, key, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, 2, 19456, 1, 32)
	return subtle.ConstantTimeCompare(key, actual) == 1, nil
}
