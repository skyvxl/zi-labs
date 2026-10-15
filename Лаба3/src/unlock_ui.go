package main

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (ui *UI) ShowUnlock(firstRun bool) {
	ui.passphrase = NewMaskedEntry()
	ui.passphrase.SetPlaceHolder("Парольная фраза файла")
	ui.passphraseConfirm = nil
	ui.message = messageLabel()
	title := "Открытие файла пользователей"
	hint := "Введите парольную фразу, заданную при создании файла."
	action := "Открыть"
	form := widget.NewForm(widget.NewFormItem("Парольная фраза", ui.passphrase))
	if firstRun {
		title = "Создание файла пользователей"
		hint = "Задайте парольную фразу для шифрования файла. Она вводится при каждом запуске и отличается от пароля учётной записи."
		action = "Создать"
		ui.passphraseConfirm = NewMaskedEntry()
		form.Append("Подтверждение", ui.passphraseConfirm)
	}
	ui.unlockSubmit = widget.NewButton(action, func() {
		phrase := ui.passphrase.Value()
		matches := !firstRun || phrase == ui.passphraseConfirm.Value()
		ui.passphrase.Clear()
		if ui.passphraseConfirm != nil {
			ui.passphraseConfirm.Clear()
		}
		if firstRun && (phrase == "" || !matches) {
			ui.message.SetText("Введите непустую парольную фразу и совпадающее подтверждение.")
			return
		}
		if err := ui.service.store.OpenStore(ui.dataPath, phrase); err != nil {
			ui.showUnlockFailure(err.Error())
			return
		}
		ui.ShowLogin()
	})
	ui.unlockSubmit.Importance = widget.HighImportance
	submit := func(string) { ui.unlockSubmit.OnTapped() }
	ui.passphrase.OnSubmitted = submit
	if ui.passphraseConfirm != nil {
		ui.passphraseConfirm.OnSubmitted = submit
	}
	actions := container.NewHBox(layout.NewSpacer(), widget.NewButton("Отмена", ui.cancelUnlock), ui.unlockSubmit)
	body := container.NewVBox(wrapLabel(hint), form, ui.message, actions)
	ui.frame(title, "", container.NewBorder(body, nil, nil, nil, layout.NewSpacer()))
	ui.window.Canvas().Focus(ui.passphrase)
}

func (ui *UI) showUnlockFailure(message string) {
	ui.service.Stop()
	ui.unlockSubmit.Disable()
	ui.stopAcknowledge = widget.NewButton("Завершить работу", ui.finish)
	ui.stopAcknowledge.Importance = widget.HighImportance
	ui.frame("Файл пользователей не открыт", "", container.NewVBox(
		wrapLabel(message), ui.stopAcknowledge,
	))
}

func (ui *UI) cancelUnlock() {
	ui.passphrase.Clear()
	if ui.passphraseConfirm != nil {
		ui.passphraseConfirm.Clear()
	}
	ui.showUnlockFailure("Ввод парольной фразы отменён. Работа программы завершается.")
}
