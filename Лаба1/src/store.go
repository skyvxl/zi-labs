package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type Store struct {
	mu        sync.Mutex
	users     map[string]User
	storePath string
}

func validUsername(name string) bool {
	return name != "" && utf8.ValidString(name) && strings.TrimSpace(name) == name && !strings.ContainsFunc(name, unicode.IsControl)
}

func validateUser(user User) error {
	if !validUsername(user.Username) {
		return errors.New("имя должно быть непустым и не содержать управляющих символов или внешних пробелов")
	}
	if user.Username == AdminUsername && user.Blocked {
		return errors.New("блокировка ADMIN запрещена")
	}
	if user.PasswordHash != "" {
		if _, _, err := decodeHash(user.PasswordHash); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) OpenStore(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// #nosec G304 -- The local user selects the account file through the -data flag.
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		users := map[string]User{AdminUsername: {Username: AdminUsername, RestrictionsEnabled: true}}
		if err := saveUsers(path, users); err != nil {
			return fmt.Errorf("создание файла пользователей: %w", err)
		}
		s.users, s.storePath = users, path
		return nil
	}
	if err != nil {
		return fmt.Errorf("чтение файла пользователей: %w", err)
	}
	var users map[string]User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("повреждённый файл пользователей: %w", err)
	}
	if _, ok := users[AdminUsername]; !ok {
		return errors.New("в файле пользователей отсутствует ADMIN")
	}
	for key, user := range users {
		if key != user.Username {
			return fmt.Errorf("несовпадение ключа и имени записи %q", key)
		}
		if err := validateUser(user); err != nil {
			return fmt.Errorf("запись %q: %w", key, err)
		}
	}
	s.users, s.storePath = users, path
	return nil
}

func saveUsers(path string, users map[string]User) (err error) {
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".users-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := os.Remove(temp.Name()); cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			err = errors.Join(err, cleanupErr)
		}
	}()
	if _, err := temp.Write(append(data, '\n')); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func (s *Store) Get(username string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[username]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (s *Store) All() []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	sort.Slice(users, func(i, j int) bool {
		if users[i].Username == AdminUsername {
			return true
		}
		if users[j].Username == AdminUsername {
			return false
		}
		return users[i].Username < users[j].Username
	})
	return users
}

func (s *Store) List() []string {
	users := s.All()
	names := make([]string, len(users))
	for i, user := range users {
		names[i] = user.Username
	}
	return names
}

// Commit a copy only after the disk replacement succeeds.
func (s *Store) persist(user User) error {
	if err := validateUser(user); err != nil {
		return err
	}
	next := make(map[string]User, len(s.users)+1)
	for k, v := range s.users {
		next[k] = v
	}
	next[user.Username] = user
	if err := saveUsers(s.storePath, next); err != nil {
		return fmt.Errorf("сохранение пользователей: %w", err)
	}
	s.users = next
	return nil
}

func (s *Store) AddUser(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[user.Username]; ok {
		return ErrUserAlreadyExists
	}
	return s.persist(user)
}

func (s *Store) UpdateUser(username string, changes User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[username]; !ok {
		return ErrUserNotFound
	}
	if username != changes.Username {
		return errors.New("нельзя изменять имя учётной записи")
	}
	return s.persist(changes)
}
