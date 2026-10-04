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
	dataPath := flag.String("data", filepath.Join("data", "users.json"), "путь к файлу учётных записей")
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
	if err := store.OpenStore(path); err != nil {
		return err
	}
	application := app.NewWithID("ru.stankin.nilov.security.lab1")
	application.Settings().SetTheme(lightTheme{Theme: theme.DefaultTheme()})
	ui := NewUI(application, NewService(store), path)
	ui.ShowLogin()
	ui.window.ShowAndRun()
	return nil
}
