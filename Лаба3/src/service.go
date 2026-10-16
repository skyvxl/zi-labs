package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

type State int

const (
	StateGuest State = iota
	StateSetup
	StatePasswordChange
	StateAuthenticated
	StateStopped
)

type Service struct {
	mu       sync.Mutex
	store    *Store
	state    State
	username string
	attempts int
}

func NewService(store *Store) *Service { return &Service{store: store} }
func (s *Service) State() State        { s.mu.Lock(); defer s.mu.Unlock(); return s.state }
func (s *Service) Attempts() int       { s.mu.Lock(); defer s.mu.Unlock(); return s.attempts }

func (s *Service) current() (User, error) {
	if s.state == StateStopped {
		return User{}, ErrStopped
	}
	if s.state != StateSetup && s.state != StatePasswordChange && s.state != StateAuthenticated {
		return User{}, ErrAccessDenied
	}
	user, err := s.store.Get(s.username)
	if err != nil {
		return User{}, err
	}
	if user.Blocked {
		return User{}, ErrBlocked
	}
	return user, nil
}

func (s *Service) CurrentUser() (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, err := s.current()
	user.PasswordHash = ""
	return user, err
}

func (s *Service) authorize(admin bool) (User, error) {
	user, err := s.current()
	if err != nil {
		return User{}, err
	}
	if s.state != StateAuthenticated || (admin && user.Username != AdminUsername) {
		return User{}, ErrAccessDenied
	}
	return user, nil
}

func (s *Service) Login(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == StateStopped {
		return ErrStopped
	}
	if s.state != StateGuest {
		return ErrAccessDenied
	}
	username = strings.TrimSpace(username)
	user, err := s.store.Get(username)
	if err != nil {
		return err
	}
	if user.Blocked {
		return ErrBlocked
	}
	correct := password == ""
	if user.PasswordHash != "" {
		correct, err = VerifyPassword(user.PasswordHash, password)
		if err != nil {
			return err
		}
	}
	if !correct {
		s.attempts++
		if s.attempts >= 3 {
			s.state = StateStopped
			s.username = ""
			return fmt.Errorf("%w. Три ошибки входа — работа программы завершается", ErrWrongPassword)
		}
		return fmt.Errorf("%w. Осталось попыток: %d", ErrWrongPassword, 3-s.attempts)
	}
	s.username = username
	switch {
	case user.PasswordHash == "":
		s.state = StateSetup
	case ValidatePolicy(user, password) != nil:
		s.state = StatePasswordChange
	default:
		s.state = StateAuthenticated
	}
	return nil
}

func validateNewPassword(user User, password, confirmation string) error {
	if password != confirmation {
		return errors.New("новый пароль и подтверждение должны совпадать")
	}
	if password == "" {
		return errors.New("новый пароль не может быть пустым")
	}
	return ValidatePolicy(user, password)
}

func (s *Service) SetInitialPassword(password, confirmation string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateSetup {
		return ErrAccessDenied
	}
	user, err := s.current()
	if err != nil {
		return err
	}
	if user.PasswordHash != "" {
		return errors.New("пароль уже установлен; войдите повторно")
	}
	if err := validateNewPassword(user, password, confirmation); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	if err := s.store.UpdateUser(user.Username, user); err != nil {
		return err
	}
	s.state, s.username = StateGuest, ""
	return nil
}

func (s *Service) ChangePassword(old, password, confirmation string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, err := s.current()
	if err != nil {
		return err
	}
	if s.state != StateAuthenticated && s.state != StatePasswordChange {
		return ErrAccessDenied
	}
	correct, err := VerifyPassword(user.PasswordHash, old)
	if err != nil {
		return err
	}
	if !correct {
		return errors.New("неверный старый пароль")
	}
	if err := validateNewPassword(user, password, confirmation); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	if err := s.store.UpdateUser(user.Username, user); err != nil {
		return err
	}
	s.state, s.username = StateGuest, ""
	return nil
}

func (s *Service) Logout() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateStopped {
		s.state = StateGuest
		s.username = ""
	}
}

func (s *Service) Stop() { s.mu.Lock(); defer s.mu.Unlock(); s.state = StateStopped; s.username = "" }

func (s *Service) Users() ([]User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.authorize(true); err != nil {
		return nil, err
	}
	users := s.store.All()
	for i := range users {
		users[i].PasswordHash = ""
	}
	return users, nil
}

func (s *Service) AddUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.authorize(true); err != nil {
		return err
	}
	return s.store.AddUser(User{Username: strings.TrimSpace(username), RestrictionsEnabled: true})
}

func (s *Service) SetBlocked(username string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.authorize(true); err != nil {
		return err
	}
	user, err := s.store.Get(username)
	if err != nil {
		return err
	}
	user.Blocked = enabled
	return s.store.UpdateUser(username, user)
}

func (s *Service) SetRestrictions(username string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.authorize(true); err != nil {
		return err
	}
	user, err := s.store.Get(username)
	if err != nil {
		return err
	}
	user.RestrictionsEnabled = enabled
	return s.store.UpdateUser(username, user)
}

func (s *Service) authorizedUser() (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.authorize(false)
}

func (s *Service) SetParameters(username string, blocked, restrictions bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.authorize(true); err != nil {
		return err
	}
	user, err := s.store.Get(username)
	if err != nil {
		return err
	}
	user.Blocked = blocked
	user.RestrictionsEnabled = restrictions
	return s.store.UpdateUser(username, user)
}
