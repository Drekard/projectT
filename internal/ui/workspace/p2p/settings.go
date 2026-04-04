// Package p2p содержит компонент вкладки "Подключение"
package p2p

import (
	"fmt"
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createSettingsTab создает вкладку с настройками подключения
func (ui *UI) createSettingsTab() fyne.CanvasObject {
	// === Ваш адрес ===
	addressSection := ui.createAddressSection()

	// === Настройки подключения ===
	settingsSection := ui.createP2PSettingsSection()

	content := container.NewVBox(
		widget.NewSeparator(),
		addressSection,
		widget.NewSeparator(),
		settingsSection,
	)

	return container.NewScroll(content)
}

// createAddressSection создает секцию управления адресом
func (ui *UI) createAddressSection() *fyne.Container {
	label := widget.NewLabel("Твой адрес: ")
	copyButton := widget.NewButtonWithIcon("Копировать адрес", theme.ContentCopyIcon(), func() {
		ui.copyMyAddress()
	})

	checkPortButton := widget.NewButton("Проверить порт", func() {
		ui.checkPortAccessibility()
	})

	// Кнопка показа локальных адресов
	showLocalButton := widget.NewButton("Локальные адреса", func() {
		ui.showLocalAddresses()
	})

	addressRow := container.NewHBox(label, copyButton)
	buttonsRow := container.NewHBox(checkPortButton, showLocalButton)

	return container.NewVBox(addressRow, buttonsRow)
}

// createP2PSettingsSection создает секцию настроек P2P
func (ui *UI) createP2PSettingsSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Настройки подключения")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Порт прослушивания с фоном
	portLabel := widget.NewLabel("Порт прослушивания:")
	portBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	portBg.SetMinSize(fyne.NewSize(100, 30))
	ui.portEntry = widget.NewEntry()
	portWrapper := container.NewStack(portBg, ui.portEntry)
	portRow := container.NewHBox(portLabel, portWrapper)

	// Чекбоксы настроек
	ui.natPortMapCheck = widget.NewCheck("NAT Port Mapping (UPnP/NAT-PMP)", nil)
	ui.relayCheck = widget.NewCheck("Relay (обход NAT)", nil)
	ui.autoRelayCheck = widget.NewCheck("Автообнаружение Relay", nil)
	ui.dhtCheck = widget.NewCheck("DHT (глобальное обнаружение)", nil)
	ui.mdnsCheck = widget.NewCheck("mDNS (локальная сеть)", nil)
	ui.helperModeCheck = widget.NewCheck("Режим помощника", nil)

	// STUN сервер с фоном
	stunLabel := widget.NewLabel("STUN сервер:")
	stunBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	stunBg.SetMinSize(fyne.NewSize(200, 30))
	ui.stunServerEntry = widget.NewEntry()
	stunWrapper := container.NewStack(stunBg, ui.stunServerEntry)
	stunRow := container.NewHBox(stunLabel, stunWrapper)

	// Кнопки сохранения и сброса
	saveBtn := widget.NewButtonWithIcon("Сохранить", theme.ConfirmIcon(), func() {
		ui.saveP2PSettings()
	})
	saveBtn.Importance = widget.HighImportance

	resetBtn := widget.NewButtonWithIcon("Сбросить", theme.CancelIcon(), func() {
		ui.resetP2PSettings()
	})

	buttonsRow := container.NewHBox(saveBtn, resetBtn)

	return container.NewVBox(
		sectionTitle,
		portRow,
		ui.natPortMapCheck,
		ui.relayCheck,
		ui.autoRelayCheck,
		ui.dhtCheck,
		ui.mdnsCheck,
		stunRow,
		ui.helperModeCheck,
		widget.NewSeparator(),
		buttonsRow,
	)
}

// saveP2PSettings сохраняет настройки P2P
func (ui *UI) saveP2PSettings() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	// Получаем текущие настройки
	settings := ui.p2pUI.GetSettings()

	// Обновляем из UI
	if ui.portEntry.Text != "" {
		if _, err := fmt.Sscanf(ui.portEntry.Text, "%d", &settings.ListenPort); err != nil {
			log.Printf("Ошибка парсинга порта: %v", err)
		}
	}
	settings.EnableNATPortMap = ui.natPortMapCheck.Checked
	settings.EnableRelay = ui.relayCheck.Checked
	settings.EnableAutoRelay = ui.autoRelayCheck.Checked
	settings.EnableDHT = ui.dhtCheck.Checked
	settings.EnableMDNS = ui.mdnsCheck.Checked
	settings.STUNServer = ui.stunServerEntry.Text
	settings.EnableHelperMode = ui.helperModeCheck.Checked

	// Сохраняем настройки через UpdateSettings
	if err := ui.p2pUI.UpdateSettings(settings); err != nil {
		ui.showErrorDialog("Ошибка", "Не удалось сохранить настройки: "+err.Error())
		return
	}

	ui.showInfoDialog("Успешно", "Настройки сохранены. Перезапустите P2P для применения.")
}

// resetP2PSettings сбрасывает настройки к значениям по умолчанию
func (ui *UI) resetP2PSettings() {
	if ui.p2pUI == nil {
		return
	}

	settings := ui.p2pUI.GetSettings()
	ui.portEntry.SetText(fmt.Sprintf("%d", settings.ListenPort))
	ui.natPortMapCheck.SetChecked(settings.EnableNATPortMap)
	ui.relayCheck.SetChecked(settings.EnableRelay)
	ui.autoRelayCheck.SetChecked(settings.EnableAutoRelay)
	ui.dhtCheck.SetChecked(settings.EnableDHT)
	ui.mdnsCheck.SetChecked(settings.EnableMDNS)
	ui.stunServerEntry.SetText(settings.STUNServer)
	ui.helperModeCheck.SetChecked(settings.EnableHelperMode)
}

// loadP2PSettings загружает настройки P2P
func (ui *UI) loadP2PSettings() {
	if ui.p2pUI == nil {
		log.Printf("[loadP2PSettings] p2pUI == nil")
		return
	}

	log.Printf("[loadP2PSettings] Загрузка настроек...")
	settings := ui.p2pUI.GetSettings()
	log.Printf("[loadP2PSettings] Получены настройки: Port=%d, NAT=%v, Relay=%v",
		settings.ListenPort, settings.EnableNATPortMap, settings.EnableRelay)

	// Устанавливаем порт (если 0 или не задан - используем 8080)
	port := settings.ListenPort
	if port <= 0 {
		port = 8080
	}
	ui.portEntry.SetText(fmt.Sprintf("%d", port))

	// Устанавливаем чекбоксы
	ui.natPortMapCheck.SetChecked(settings.EnableNATPortMap)
	ui.relayCheck.SetChecked(settings.EnableRelay)
	ui.autoRelayCheck.SetChecked(settings.EnableAutoRelay)
	ui.dhtCheck.SetChecked(settings.EnableDHT)
	ui.mdnsCheck.SetChecked(settings.EnableMDNS)
	ui.stunServerEntry.SetText(settings.STUNServer)
	ui.helperModeCheck.SetChecked(settings.EnableHelperMode)

	log.Printf("[loadP2PSettings] Настройки загружены")
}

// copyMyAddress копирует мой адрес в буфер обмена
func (ui *UI) copyMyAddress() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	addr, err := ui.p2pUI.GetPeerAddress()
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось получить адрес: %v", err))
		return
	}

	// Копируем в буфер обмена
	clipboard := ui.window.Clipboard()
	clipboard.SetContent(addr)

	ui.showInfoDialog("Успешно", "Адрес скопирован в буфер обмена")
}

// checkPortAccessibility проверяет доступность порта
func (ui *UI) checkPortAccessibility() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	ui.showInfoDialog("Проверка", "Проверка порта... (требуется внешний сервис)")
	// TODO: Реализовать проверку порта через STUN или внешний сервис
}

// showLocalAddresses показывает локальные адреса
func (ui *UI) showLocalAddresses() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	addrs := ui.p2pUI.GetLocalAddresses()
	if len(addrs) == 0 {
		ui.showInfoDialog("Локальные адреса", "Локальные адреса не найдены")
		return
	}

	// Формируем текст адресов для отображения
	addrsText := ""
	for i, addr := range addrs {
		addrsText += fmt.Sprintf("%d. %s\n", i+1, addr)
	}

	// Формируем чистый текст адресов для копирования (без нумерации)
	addrsTextClean := ""
	for _, addr := range addrs {
		addrsTextClean += addr + "\n"
	}

	// Создаём метку с адресами
	addrLabel := widget.NewLabel(addrsText)
	addrLabel.Wrapping = fyne.TextWrapBreak

	// Кнопка копирования
	copyButton := widget.NewButtonWithIcon("Копировать", theme.ContentCopyIcon(), func() {
		// Копируем все адреса в буфер обмена (без нумерации)
		clipboard := ui.window.Clipboard()
		clipboard.SetContent(addrsTextClean)
		ui.showInfoDialog("Успешно", "Локальные адреса скопированы в буфер обмена")
	})

	// Создаём диалог с кнопками
	content := container.NewVBox(addrLabel, copyButton)

	customDialog := dialog.NewCustom("Локальные адреса", "Закрыть", content, ui.window)
	customDialog.Show()
}

// showErrorDialog показывает диалог ошибки
func (ui *UI) showErrorDialog(title, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), ui.window)
}

// showInfoDialog показывает информационный диалог
func (ui *UI) showInfoDialog(title, message string) {
	dialog.ShowInformation(title, message, ui.window)
}
