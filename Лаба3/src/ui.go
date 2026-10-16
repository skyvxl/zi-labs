package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	AuthorName  = "Нилов Дмитрий Александрович"
	AuthorGroup = "ИДБ-23-02"
	AppTitle    = "Разграничение доступа"
)

type UI struct {
	app                                        fyne.App
	service                                    *Service
	window                                     fyne.Window
	dataPath                                   string
	quit                                       func()
	message                                    *widget.Label
	changeItem, usersItem, newItem, logoutItem *fyne.MenuItem
	passphrase, passphraseConfirm              *MaskedEntry
	unlockSubmit                               *widget.Button
	loginName                                  *widget.Entry
	loginPassword                              *MaskedEntry
	loginSubmit                                *widget.Button
	setupPassword, setupConfirm                *MaskedEntry
	setupSubmit                                *widget.Button
	newName                                    *widget.Entry
	newSubmit                                  *widget.Button
	oldPassword, newPassword, passwordConfirm  *MaskedEntry
	passwordSubmit                             *widget.Button
	activeDialog                               dialog.Dialog
	visibleUsers                               []User
	stopAcknowledge                            *widget.Button
}

func NewUI(app fyne.App, service *Service, dataPath string) *UI {
	ui := &UI{app: app, service: service, dataPath: dataPath}
	ui.window = app.NewWindow(AppTitle)
	ui.window.Resize(fyne.NewSize(860, 600))
	ui.quit = app.Quit
	ui.window.SetCloseIntercept(ui.finish)
	ui.installMenu()
	return ui
}

func (ui *UI) finish() {
	if ui.unlockSubmit != nil && ui.loginName == nil && ui.service.State() == StateGuest {
		ui.cancelUnlock()
		return
	}
	if ui.passphrase != nil {
		ui.passphrase.Clear()
	}
	if ui.passphraseConfirm != nil {
		ui.passphraseConfirm.Clear()
	}
	if err := ui.service.store.Close(); err != nil {
		dialog.ShowError(err, ui.window)
		return
	}
	ui.service.Stop()
	ui.quit()
}

func (ui *UI) installMenu() {
	ui.changeItem = fyne.NewMenuItem("Сменить пароль", ui.showChangePassword)
	ui.logoutItem = fyne.NewMenuItem("Выйти из аккаунта", func() { ui.service.Logout(); ui.ShowLogin() })
	ui.usersItem = fyne.NewMenuItem("Все пользователи", ui.showUsers)
	ui.newItem = fyne.NewMenuItem("Новый пользователь", ui.showNewUser)
	quitItem := fyne.NewMenuItem("Завершить работу", ui.finish)
	quitItem.IsQuit = true
	ui.window.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("Учётная запись", ui.changeItem, ui.logoutItem, fyne.NewMenuItemSeparator(), quitItem),
		fyne.NewMenu("Администрирование", ui.usersItem, ui.newItem),
		fyne.NewMenu("Справка", fyne.NewMenuItem("О программе", ui.showAbout)),
	))
	ui.updateMenu()
}

func (ui *UI) updateMenu() {
	user, err := ui.service.CurrentUser()
	authenticated := err == nil && ui.service.State() == StateAuthenticated
	ui.changeItem.Disabled = !authenticated
	ui.logoutItem.Disabled = !authenticated
	ui.usersItem.Disabled = !authenticated || user.Username != AdminUsername
	ui.newItem.Disabled = ui.usersItem.Disabled
	ui.window.MainMenu().Refresh()
}

func wrapLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}

func titleLabel(text string) *widget.Label {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func messageLabel() *widget.Label {
	label := wrapLabel("")
	label.Importance = widget.DangerImportance
	return label
}

func (ui *UI) frame(title, subtitle string, body fyne.CanvasObject) {
	header := container.NewVBox(titleLabel(title))
	if subtitle != "" {
		header.Add(wrapLabel(subtitle))
	}
	header.Add(widget.NewSeparator())
	ui.window.SetContent(container.NewPadded(container.NewBorder(header, nil, nil, nil, body)))
	ui.updateMenu()
}

func (ui *UI) ShowLogin() {
	ui.loginName = widget.NewEntry()
	ui.loginName.SetPlaceHolder("Имя учётной записи")
	ui.loginPassword = NewMaskedEntry()
	ui.loginPassword.SetPlaceHolder("Пароль")
	ui.message = messageLabel()
	ui.loginSubmit = widget.NewButtonWithIcon("Войти", theme.LoginIcon(), func() {
		err := ui.service.Login(ui.loginName.Text, ui.loginPassword.Value())
		ui.loginPassword.Clear()
		if err != nil {
			ui.message.SetText(err.Error())
			ui.message.Importance = widget.DangerImportance
			ui.message.Refresh()
			if ui.service.State() == StateStopped {
				ui.showStopped(err.Error())
			}
			return
		}
		if ui.service.State() == StateSetup || ui.service.State() == StatePasswordChange {
			ui.showSetup()
		} else {
			ui.showHome()
		}
	})
	ui.loginSubmit.Importance = widget.HighImportance
	ui.loginPassword.OnSubmitted = func(string) { ui.loginSubmit.OnTapped() }
	form := widget.NewForm(widget.NewFormItem("Имя", ui.loginName), widget.NewFormItem("Пароль", ui.loginPassword))
	actions := container.NewHBox(layout.NewSpacer(), widget.NewButton("Завершить работу", ui.finish), ui.loginSubmit)
	body := container.NewVBox(form, ui.message, actions)
	ui.frame("Вход в программу", "Введите имя и пароль зарегистрированного пользователя.", container.NewBorder(body, nil, nil, nil, layout.NewSpacer()))
	ui.window.Canvas().Focus(ui.loginName)
}

func (ui *UI) showSetup() {
	user, err := ui.service.CurrentUser()
	if err != nil {
		ui.ShowLogin()
		return
	}
	requiredChange := ui.service.State() == StatePasswordChange
	ui.oldPassword = nil
	if requiredChange {
		ui.oldPassword = NewMaskedEntry()
	}
	ui.setupPassword = NewMaskedEntry()
	ui.setupConfirm = NewMaskedEntry()
	ui.message = messageLabel()
	ui.setupSubmit = widget.NewButton("Установить пароль", func() {
		var err error
		if requiredChange {
			err = ui.service.ChangePassword(ui.oldPassword.Value(), ui.setupPassword.Value(), ui.setupConfirm.Value())
			ui.oldPassword.Clear()
		} else {
			err = ui.service.SetInitialPassword(ui.setupPassword.Value(), ui.setupConfirm.Value())
		}
		ui.setupPassword.Clear()
		ui.setupConfirm.Clear()
		if err != nil {
			ui.message.SetText(err.Error())
			return
		}
		ui.ShowLogin()
		ui.message.SetText("Пароль сохранён. Введите имя и новый пароль для входа.")
		ui.message.Importance = widget.SuccessImportance
		ui.message.Refresh()
	})
	ui.setupSubmit.Importance = widget.HighImportance
	ui.setupConfirm.OnSubmitted = func(string) { ui.setupSubmit.OnTapped() }
	hint := "Ограничения отключены. Новый пароль должен быть непустым."
	if user.RestrictionsEnabled {
		hint = PolicyHint
	}
	form := widget.NewForm()
	if requiredChange {
		form.Append("Старый пароль", ui.oldPassword)
	}
	form.Append("Новый пароль", ui.setupPassword)
	form.Append("Подтверждение", ui.setupConfirm)
	body := container.NewVBox(wrapLabel(hint), form, ui.message, container.NewHBox(layout.NewSpacer(), widget.NewButton("Завершить работу", ui.finish), ui.setupSubmit))
	title := "Первоначальная установка пароля"
	subtitle := "Учётная запись: " + user.Username + ". Установите пароль и повторите его."
	if requiredChange {
		title = "Обязательная смена пароля"
		subtitle = "Текущий пароль не соответствует включённым ограничениям. Для входа установите новый пароль."
		ui.setupSubmit.SetText("Сохранить пароль")
	}
	ui.frame(title, subtitle, container.NewBorder(body, nil, nil, nil, layout.NewSpacer()))
	if requiredChange {
		ui.window.Canvas().Focus(ui.oldPassword)
	} else {
		ui.window.Canvas().Focus(ui.setupPassword)
	}
}

func (ui *UI) showHome() {
	user, err := ui.service.CurrentUser()
	if err != nil {
		ui.ShowLogin()
		return
	}
	role := "Обычный пользователь"
	description := "Вы можете изменить свой пароль или завершить работу программы."
	buttons := []fyne.CanvasObject{widget.NewButtonWithIcon("Сменить пароль", theme.AccountIcon(), ui.showChangePassword)}
	if user.Username == AdminUsername {
		role = "Администратор"
		description = "Управляйте учётными записями, блокировкой и проверкой новых паролей."
		buttons = append(buttons, widget.NewButtonWithIcon("Все пользователи", theme.ListIcon(), ui.showUsers), widget.NewButtonWithIcon("Новый пользователь", theme.ContentAddIcon(), ui.showNewUser))
	}
	body := container.NewVBox(widget.NewCard("Режим работы", role, wrapLabel(description)),
		container.NewGridWithColumns(2, buttons...),
		container.NewHBox(layout.NewSpacer(), widget.NewButton("Выйти из аккаунта", func() { ui.service.Logout(); ui.ShowLogin() }), widget.NewButton("Завершить работу", ui.finish)))
	ui.frame("Учётная запись "+user.Username, "", container.NewBorder(body, nil, nil, nil, layout.NewSpacer()))
}

func (ui *UI) showChangePassword() {
	if _, err := ui.service.authorizedUser(); err != nil {
		dialog.ShowError(err, ui.window)
		return
	}
	user, _ := ui.service.CurrentUser()
	ui.oldPassword = NewMaskedEntry()
	ui.newPassword = NewMaskedEntry()
	ui.passwordConfirm = NewMaskedEntry()
	message := messageLabel()
	form := widget.NewForm(widget.NewFormItem("Старый пароль", ui.oldPassword), widget.NewFormItem("Новый пароль", ui.newPassword), widget.NewFormItem("Подтверждение", ui.passwordConfirm))
	hint := "Ограничения отключены. Новый пароль должен быть непустым."
	if user.RestrictionsEnabled {
		hint = PolicyHint
	}
	var d *dialog.CustomDialog
	ui.passwordSubmit = widget.NewButton("Сохранить пароль", func() {
		err := ui.service.ChangePassword(ui.oldPassword.Value(), ui.newPassword.Value(), ui.passwordConfirm.Value())
		ui.oldPassword.Clear()
		ui.newPassword.Clear()
		ui.passwordConfirm.Clear()
		if err != nil {
			message.SetText(err.Error())
			return
		}
		d.Hide()
		ui.ShowLogin()
		ui.message.SetText("Пароль изменён. Введите имя и новый пароль для входа.")
		ui.message.Importance = widget.SuccessImportance
		ui.message.Refresh()
	})
	ui.passwordSubmit.Importance = widget.HighImportance
	ui.passwordConfirm.OnSubmitted = func(string) { ui.passwordSubmit.OnTapped() }
	d = dialog.NewCustomWithoutButtons("Смена пароля", container.NewVBox(wrapLabel(hint), form, message,
		container.NewHBox(layout.NewSpacer(), widget.NewButton("Отмена", func() { d.Hide() }), ui.passwordSubmit)), ui.window)
	d.SetOnClosed(func() { ui.oldPassword.Clear(); ui.newPassword.Clear(); ui.passwordConfirm.Clear() })
	d.Resize(fyne.NewSize(650, 360))
	ui.activeDialog = d
	d.Show()
	ui.window.Canvas().Focus(ui.oldPassword)
}

func (ui *UI) showNewUser() {
	if _, err := ui.service.Users(); err != nil {
		dialog.ShowError(err, ui.window)
		return
	}
	ui.newName = widget.NewEntry()
	ui.newName.SetPlaceHolder("Уникальное имя")
	message := messageLabel()
	var d *dialog.CustomDialog
	ui.newSubmit = widget.NewButton("Добавить", func() {
		if err := ui.service.AddUser(ui.newName.Text); err != nil {
			message.SetText(err.Error())
			return
		}
		d.Hide()
		ui.showHome()
		dialog.ShowInformation("Пользователь добавлен", "При первом входе с пустым паролем пользователь установит свой пароль. Ограничения включены.", ui.window)
	})
	ui.newSubmit.Importance = widget.HighImportance
	d = dialog.NewCustomWithoutButtons("Новый пользователь", container.NewVBox(
		wrapLabel("Администратор задаёт только имя. Пароль пользователь выбирает самостоятельно при первом входе."),
		widget.NewForm(widget.NewFormItem("Имя", ui.newName)), message,
		container.NewHBox(layout.NewSpacer(), widget.NewButton("Отмена", func() { d.Hide() }), ui.newSubmit)), ui.window)
	d.Resize(fyne.NewSize(610, 270))
	ui.activeDialog = d
	d.Show()
	ui.window.Canvas().Focus(ui.newName)
}

func yesNo(value bool) string {
	if value {
		return "Да"
	}
	return "Нет"
}

func (ui *UI) showUsers() {
	users, err := ui.service.Users()
	if err != nil {
		dialog.ShowError(err, ui.window)
		return
	}
	ui.visibleUsers = users
	rows := container.NewVBox(container.NewGridWithColumns(4, titleLabel("Имя"), titleLabel("Блокировка"), titleLabel("Ограничения"), titleLabel("Параметры")), widget.NewSeparator())
	for _, user := range users {
		rows.Add(container.NewGridWithColumns(4, widget.NewLabel(user.Username), widget.NewLabel(yesNo(user.Blocked)), widget.NewLabel(yesNo(user.RestrictionsEnabled)),
			widget.NewButton("Изменить", func() { ui.activeDialog.Hide(); ui.showUserSettings(user.Username) })))
	}
	scroll := container.NewVScroll(rows)
	scroll.SetMinSize(fyne.NewSize(720, 300))
	d := dialog.NewCustom("Все пользователи", "Закрыть", container.NewBorder(wrapLabel("Блокировка запрещает вход. Ограничение проверяет только вновь выбираемые пароли."), nil, nil, nil, scroll), ui.window)
	d.Resize(fyne.NewSize(770, 440))
	ui.activeDialog = d
	d.Show()
}

func (ui *UI) showUserSettings(username string) {
	users, err := ui.service.Users()
	if err != nil {
		dialog.ShowError(err, ui.window)
		return
	}
	var selected User
	for _, user := range users {
		if user.Username == username {
			selected = user
		}
	}
	blocked := widget.NewCheck("Заблокировать учётную запись", nil)
	blocked.SetChecked(selected.Blocked)
	restrictions := widget.NewCheck("Ограничения пароля", nil)
	restrictions.SetChecked(selected.RestrictionsEnabled)
	if username == AdminUsername {
		blocked.Disable()
	}
	message := messageLabel()
	var d *dialog.CustomDialog
	save := widget.NewButton("Сохранить", func() {
		// One transaction for both flags avoids a partially saved settings form.
		if err := ui.service.SetParameters(username, blocked.Checked, restrictions.Checked); err != nil {
			message.SetText(err.Error())
			return
		}
		d.Hide()
		ui.showHome()
		ui.showUsers()
	})
	save.Importance = widget.HighImportance
	body := container.NewVBox(titleLabel("Учётная запись: "+username), blocked, restrictions, wrapLabel(PolicyHint),
		wrapLabel("Включение проверки не меняет уже установленный пароль. Блокировка ADMIN запрещена."), message,
		container.NewHBox(layout.NewSpacer(), widget.NewButton("Отмена", func() { d.Hide(); ui.showUsers() }), save))
	d = dialog.NewCustomWithoutButtons("Параметры пользователя", body, ui.window)
	d.Resize(fyne.NewSize(650, 330))
	ui.activeDialog = d
	d.Show()
}

func (ui *UI) showAbout() {
	text := fmt.Sprintf("Защита файла учётных записей\n\nАвтор: %s\nГруппа: %s\n\nВариант %d\n%s\n\nШифрование файла: AES-128, CFB\nКлюч: MD5 парольной фразы с солью\nКонтроль целостности: HMAC-SHA256\n\nАрифметические знаки: + - * / %%", AuthorName, AuthorGroup, VariantNumber, VariantDescription)
	d := dialog.NewCustom("О программе", "Закрыть", wrapLabel(text), ui.window)
	d.Resize(fyne.NewSize(650, 400))
	ui.activeDialog = d
	d.Show()
}

func (ui *UI) showStopped(message string) {
	ui.loginSubmit.Disable()
	ui.updateMenu()
	ui.stopAcknowledge = widget.NewButton("Завершить работу", ui.finish)
	ui.stopAcknowledge.Importance = widget.HighImportance
	ui.frame("Работа программы завершается", message, container.NewVBox(wrapLabel("Трижды введён неверный пароль. После закрытия этого сообщения программа завершится. Учётные записи сохранены."), ui.stopAcknowledge))
}
