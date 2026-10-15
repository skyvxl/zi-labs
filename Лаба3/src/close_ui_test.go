package main

import "testing"

func TestUICloseFailureAllowsRetry(t *testing.T) {
	ui := newTestUI(t)
	loginAdmin(t, ui.service)
	path := ui.service.store.storePath
	ui.service.store.storePath = t.TempDir()
	calls := 0
	ui.quit = func() { calls++ }
	ui.finish()
	if calls != 0 || ui.service.State() != StateAuthenticated {
		t.Fatal("close failure discarded active session")
	}
	ui.service.store.storePath = path
	ui.finish()
	if calls != 1 || ui.service.State() != StateStopped {
		t.Fatal("close retry failed")
	}
}

func TestUIMenuUsesCustomQuit(t *testing.T) {
	ui := newTestUI(t)
	items := ui.window.MainMenu().Items[0].Items
	if len(items) != 4 {
		t.Fatalf("unexpected account menu items: %d", len(items))
	}
	quitItem := items[len(items)-1]
	if quitItem.Label != "Завершить работу" || !quitItem.IsQuit {
		t.Fatal("Fyne will append its own quit item")
	}
	calls := 0
	ui.quit = func() { calls++ }
	quitItem.Action()
	if calls != 1 || ui.service.State() != StateStopped || ui.service.store.cipher != nil {
		t.Fatal("menu quit bypassed store close")
	}
}
