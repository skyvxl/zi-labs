package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

func main() {
	dataPath := flag.String("data", filepath.Join("data", "users.enc"), "путь к файлу учётных записей")
	flag.Parse()
	if err := run(*dataPath); err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка запуска:", err)
		os.Exit(1)
	}
}

func run(path string) (err error) {
	lock, err := LockStore(path)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, lock.Close())
	}()
	store := &Store{}
	_, statErr := os.Stat(path)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	defer func() { err = errors.Join(err, store.Close()) }()
	application := app.NewWithID("ru.stankin.nilov.security.lab3")
	application.Settings().SetTheme(lightTheme{Theme: theme.DefaultTheme()})
	ui := NewUI(application, NewService(store), path)
	ui.ShowUnlock(errors.Is(statErr, os.ErrNotExist))
	ui.window.ShowAndRun()
	return nil
}
