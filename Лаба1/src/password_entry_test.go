package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestPasswordEntryAlwaysStars(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	entry := NewMaskedEntry()
	test.Type(entry, "aя+*")
	if entry.Value() != "aя+*" || entry.Text != "****" {
		t.Fatalf("value=%q, visible=%q", entry.Value(), entry.Text)
	}
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if entry.Value() != "aя*" {
		t.Fatalf("edit: %q", entry.Value())
	}
	clipboard := app.Clipboard()
	clipboard.SetContent("Ж%")
	entry.TypedShortcut(&fyne.ShortcutPaste{Clipboard: clipboard})
	if entry.Value() != "aяЖ%*" {
		t.Fatalf("paste: %q", entry.Value())
	}
	clipboard.SetContent("unchanged")
	entry.TypedShortcut(&fyne.ShortcutCopy{Clipboard: clipboard})
	if clipboard.Content() != "unchanged" {
		t.Fatal("password copied")
	}
	if entry.ActionItem != nil || strings.ContainsAny(entry.Text, "aяЖ%") {
		t.Fatal("password disclosure")
	}
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if entry.Value() != "яЖ%*" {
		t.Fatal("delete/home failed")
	}
	entry.Clear()
	if entry.Value() != "" || entry.Text != "" {
		t.Fatal("clear failed")
	}
}

func TestPasswordDragCannotSelect(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	window := app.NewWindow("password")
	entry := NewMaskedEntry()
	window.SetContent(entry)
	window.Resize(fyne.NewSize(400, 80))
	window.Show()
	test.Type(entry, "abcdefghij")
	test.Drag(window.Canvas(), fyne.NewPos(80, 20), 60, 0)
	if selected := entry.SelectedText(); selected != "" {
		t.Fatalf("password unexpectedly selected: %q", selected)
	}
	// Mouse drag must not reveal input or desynchronise the underlying value.
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if len([]rune(entry.Value())) != 9 || len(entry.Text) != 9 {
		t.Fatal("editing desynchronised mask")
	}
}
