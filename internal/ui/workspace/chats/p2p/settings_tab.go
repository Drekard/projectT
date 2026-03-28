// Package p2p содержит компоненты панели управления P2P
package p2p

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"strings"

	"projectT/internal/config"
	network "projectT/internal/services/p2p/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createSettingsTab создает вкладку с настройками P2P
func (p *Panel) createSettingsTab() fyne.CanvasObject {
	// === Ваш адрес ===
	addressSection := p.createAddressSection()

	// === Настройки подключения ===
	settingsSection := p.createP2PSettingsSection()

	content := container.NewVBox(
		widget.NewSeparator(),
		addressSection,
		widget.NewSeparator(),
		settingsSection,
	)

	return container.NewScroll(content)
}

// createAddressSection создает секцию управления адресом
func (p *Panel) createAddressSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Ваш адрес")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	p.myAddressLabel = widget.NewLabel("Адрес: P2P не запущен")

	copyButton := widget.NewButtonWithIcon("Копировать", theme.ContentCopyIcon(), func() {
		p.copyMyAddress()
	})

	checkPortButton := widget.NewButton("Проверить порт", func() {
		p.checkPortAccessibility()
	})

	// Кнопка показа локальных адресов
	showLocalButton := widget.NewButton("Локальные адреса", func() {
		p.showLocalAddresses()
	})

	addressRow := container.NewHBox(p.myAddressLabel, copyButton)
	buttonsRow := container.NewHBox(checkPortButton, showLocalButton)

	return container.NewVBox(sectionTitle, addressRow, buttonsRow)
}

// createP2PSettingsSection создает секцию настроек P2P
func (p *Panel) createP2PSettingsSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Настройки подключения")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Порт прослушивания с фоном
	portLabel := widget.NewLabel("Порт прослушивания:")
	portBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	portBg.SetMinSize(fyne.NewSize(100, 30))
	p.portEntry = widget.NewEntry()
	portWrapper := container.NewStack(portBg, p.portEntry)
	portRow := container.NewHBox(portLabel, portWrapper)

	// Чекбоксы настроек
	p.natPortMapCheck = widget.NewCheck("NAT Port Mapping (UPnP/NAT-PMP)", nil)
	p.relayCheck = widget.NewCheck("Relay (обход NAT)", nil)
	p.autoRelayCheck = widget.NewCheck("Автообнаружение Relay", nil)
	p.dhtCheck = widget.NewCheck("DHT (глобальное обнаружение)", nil)
	p.mdnsCheck = widget.NewCheck("mDNS (локальная сеть)", nil)
	p.stunCheck = widget.NewCheck("STUN клиент", nil)
	p.helperModeCheck = widget.NewCheck("Режим помощника", nil)

	// STUN сервер с фоном
	stunLabel := widget.NewLabel("STUN сервер:")
	stunBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	stunBg.SetMinSize(fyne.NewSize(200, 30))
	p.stunServerEntry = widget.NewEntry()
	p.stunServerEntry.SetText("stun.l.google.com:19302")
	stunWrapper := container.NewStack(stunBg, p.stunServerEntry)
	stunRow := container.NewHBox(stunLabel, stunWrapper)

	// Кнопки
	saveSettingsBtn := widget.NewButtonWithIcon("Сохранить (в памяти)", theme.DocumentSaveIcon(), func() {
		p.saveP2PSettings()
	})

	saveConfigBtn := widget.NewButtonWithIcon("💾 Сохранить в config.yaml", theme.FolderOpenIcon(), func() {
		p.saveP2PSettingsToConfig()
	})

	restartBtn := widget.NewButtonWithIcon("Применить и перезапустить", theme.ViewRefreshIcon(), func() {
		p.restartP2PWithNewSettings()
	})
	restartBtn.Importance = widget.HighImportance

	buttonsRow := container.NewHBox(saveSettingsBtn, saveConfigBtn, restartBtn)

	return container.NewVBox(
		sectionTitle,
		portRow,
		p.natPortMapCheck,
		p.relayCheck,
		p.autoRelayCheck,
		p.dhtCheck,
		p.mdnsCheck,
		p.stunCheck,
		stunRow,
		p.helperModeCheck,
		widget.NewSeparator(),
		buttonsRow,
	)
}

// loadP2PSettings загружает настройки P2P
func (p *Panel) loadP2PSettings() {
	if p.p2pUI == nil {
		log.Printf("[loadP2PSettings] p2pUI == nil")
		return
	}

	log.Printf("[loadP2PSettings] Загрузка настроек...")
	settings := p.p2pUI.GetSettings()
	log.Printf("[loadP2PSettings] Получены настройки: Port=%d, NAT=%v, Relay=%v",
		settings.ListenPort, settings.EnableNATPortMap, settings.EnableRelay)

	// Устанавливаем порт (если 0 или не задан - используем 8080)
	port := settings.ListenPort
	if port <= 0 {
		port = 8080
	}
	p.portEntry.SetText(fmt.Sprintf("%d", port))

	p.natPortMapCheck.SetChecked(settings.EnableNATPortMap)
	p.relayCheck.SetChecked(settings.EnableRelay)
	p.autoRelayCheck.SetChecked(settings.EnableAutoRelay)
	p.dhtCheck.SetChecked(settings.EnableDHT)
	p.mdnsCheck.SetChecked(settings.EnableMDNS)
	p.stunCheck.SetChecked(settings.EnableSTUN)
	p.stunServerEntry.SetText(settings.STUNServer)
	p.helperModeCheck.SetChecked(settings.EnableHelperMode)
	log.Printf("[loadP2PSettings] Настройки загружены, порт=%d", port)
}

// saveP2PSettings сохраняет настройки P2P
func (p *Panel) saveP2PSettings() {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	var port int
	if _, err := fmt.Sscanf(p.portEntry.Text, "%d", &port); err != nil {
		p.showErrorDialog("Ошибка", "Неверный формат порта")
		return
	}

	settings := &network.P2PSettings{
		ListenPort:       port,
		EnableNATPortMap: p.natPortMapCheck.Checked,
		EnableRelay:      p.relayCheck.Checked,
		EnableAutoRelay:  p.autoRelayCheck.Checked,
		EnableDHT:        p.dhtCheck.Checked,
		EnableMDNS:       p.mdnsCheck.Checked,
		EnableSTUN:       p.stunCheck.Checked,
		STUNServer:       p.stunServerEntry.Text,
		EnableHelperMode: p.helperModeCheck.Checked,
	}

	err := p.p2pUI.UpdateSettings(settings)
	if err != nil {
		p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось сохранить настройки: %v", err))
		return
	}

	p.showInfoDialog("Успешно", "Настройки P2P сохранены\n\nДля применения настроек нажмите 'Применить и перезапустить P2P'")
}

// saveP2PSettingsToConfig сохраняет настройки в config.yaml
func (p *Panel) saveP2PSettingsToConfig() {
	window := p.chatsUI.GetWindow()
	if window == nil {
		return
	}

	var port int
	if _, err := fmt.Sscanf(p.portEntry.Text, "%d", &port); err != nil {
		p.showErrorDialog("Ошибка", "Неверный формат порта")
		return
	}

	// Показываем диалог подтверждения
	dialog.ShowConfirm(
		"Сохранение в config.yaml",
		fmt.Sprintf("Сохранить порт %d в config.yaml?\n\nЭто изменит файл конфигурации.\nНовый порт будет использоваться при следующем запуске.", port),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Получаем путь к config.yaml
			configPath := "config.yaml"

			// Проверяем существование файла
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				p.showErrorDialog("Ошибка", "Файл config.yaml не найден\n\nЗапустите приложение из директории с config.yaml")
				return
			}

			// Загружаем текущую конфигурацию (только YAML, без ENV!)
			loader := config.NewLoader()
			cfg, err := loader.LoadFromYAMLOnly(configPath)
			if err != nil {
				p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось загрузить config.yaml: %v", err))
				return
			}

			// Обновляем порт
			oldPort := cfg.P2P.Port
			cfg.P2P.Port = port

			log.Printf("[SaveConfig] Старый порт: %d, Новый порт: %d", oldPort, port)

			// Сохраняем конфигурацию
			err = config.Save(cfg, configPath)
			if err != nil {
				p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось сохранить config.yaml: %v", err))
				return
			}

			log.Printf("[SaveConfig] Конфигурация сохранена в %s", configPath)

			// Проверяем что записалось
			checkLoader := config.NewLoader()
			checkCfg, err := checkLoader.Load()
			if err == nil {
				log.Printf("[SaveConfig] Проверка: порт в config.yaml = %d", checkCfg.P2P.Port)
			}

			p.showInfoDialog("Успешно",
				fmt.Sprintf("Порт %d сохранён в config.yaml\n\n⚠️ Для применения настроек перезапустите приложение\n\n❌ НЕ используйте PROJECTT_P2P_PORT - это перезаписывает config!", port))
		},
		window,
	)
}

// restartP2PWithNewSettings перезапускает P2P с новыми настройками
func (p *Panel) restartP2PWithNewSettings() {
	window := p.chatsUI.GetWindow()
	if window == nil {
		return
	}

	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	// Показываем диалог подтверждения
	dialog.ShowConfirm(
		"Перезапуск P2P",
		"Для применения настроек требуется перезапуск P2P.\n\nТекущие подключения будут разорваны.\n\nПродолжить?",
		func(ok bool) {
			if !ok {
				return
			}

			// Сохраняем настройки
			p.saveP2PSettingsSilent()

			// Останавливаем P2P
			if err := p.p2pUI.Stop(); err != nil {
				p.showErrorDialog("Ошибка", fmt.Sprintf("Ошибка остановки P2P: %v", err))
				return
			}

			// Запускаем P2P заново
			if err := p.p2pUI.Start(); err != nil {
				p.showErrorDialog("Ошибка", fmt.Sprintf("Ошибка запуска P2P: %v", err))
				return
			}

			p.showInfoDialog("P2P перезапущен", "Настройки применены успешно")

			// Обновляем отображение
			p.refreshConnectionStatus()
			p.loadConnectedPeers()
		},
		window,
	)
}

// saveP2PSettingsSilent сохраняет настройки без показа диалога
func (p *Panel) saveP2PSettingsSilent() {
	if p.p2pUI == nil {
		return
	}

	var port int
	if _, err := fmt.Sscanf(p.portEntry.Text, "%d", &port); err != nil {
		return
	}

	settings := &network.P2PSettings{
		ListenPort:       port,
		EnableNATPortMap: p.natPortMapCheck.Checked,
		EnableRelay:      p.relayCheck.Checked,
		EnableAutoRelay:  p.autoRelayCheck.Checked,
		EnableDHT:        p.dhtCheck.Checked,
		EnableMDNS:       p.mdnsCheck.Checked,
		EnableSTUN:       p.stunCheck.Checked,
		STUNServer:       p.stunServerEntry.Text,
		EnableHelperMode: p.helperModeCheck.Checked,
	}

	_ = p.p2pUI.UpdateSettings(settings)
}

// copyMyAddress копирует свой адрес в буфер обмена
func (p *Panel) copyMyAddress() {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	window := p.chatsUI.GetWindow()
	if window == nil {
		return
	}

	// Получаем ВСЕ локальные адреса
	addresses := p.p2pUI.GetLocalAddresses()
	if len(addresses) == 0 {
		p.showErrorDialog("Ошибка", "Не удалось получить адрес")
		return
	}

	// Выбираем ПЕРВЫЙ адрес с 192.168.x.x (для локальной сети)
	selectedAddr := ""
	for _, addr := range addresses {
		if strings.Contains(addr, "/ip4/192.168.") {
			selectedAddr = addr
			break
		}
	}

	// Если нет 192.168, берём первый не-localhost
	if selectedAddr == "" {
		for _, addr := range addresses {
			if !strings.Contains(addr, "127.0.0.1") && !strings.Contains(addr, "::1") {
				selectedAddr = addr
				break
			}
		}
	}

	// Если всё ещё пусто, берём первый
	if selectedAddr == "" && len(addresses) > 0 {
		selectedAddr = addresses[0]
	}

	// Копируем в буфер обмена
	window.Clipboard().SetContent(selectedAddr)
	p.showInfoDialog("Адрес скопирован", fmt.Sprintf("Адрес скопирован в буфер обмена:\n%s", selectedAddr))
}

// checkPortAccessibility проверяет доступность порта
func (p *Panel) checkPortAccessibility() {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	window := p.chatsUI.GetWindow()
	if window == nil {
		p.showErrorDialog("Ошибка", "Окно не инициализировано")
		return
	}

	// Получаем порт из настроек
	port := 8080
	if p.portEntry != nil && p.portEntry.Text != "" {
		_, err := fmt.Sscanf(p.portEntry.Text, "%d", &port)
		if err != nil {
			p.showErrorDialog("Ошибка", "Неверный формат порта")
			return
		}
	}

	// Показываем информацию о брандмауэре
	firewallInfo := p.p2pUI.CheckFirewall(port)

	message := fmt.Sprintf(
		"Порт: %d\n"+
			"Правило: %s\n\n"+
			"PowerShell:\n%s\n\n"+
			"CMD:\n%s",
		firewallInfo.Port,
		firewallInfo.RuleName,
		firewallInfo.PowerShellCmd,
		firewallInfo.CMDCmd,
	)

	// Создаём и показываем диалог явно
	d := dialog.NewInformation("Брандмауэр", message, window)
	d.Show()
}

// showLocalAddresses показывает локальные адреса для подключения в одной сети
func (p *Panel) showLocalAddresses() {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	window := p.chatsUI.GetWindow()
	if window == nil {
		p.showErrorDialog("Ошибка", "Окно не инициализировано")
		return
	}

	status := p.p2pUI.GetStatus()

	if !status.IsRunning {
		p.showErrorDialog("Ошибка", "P2P не запущен")
		return
	}

	// Получаем локальные адреса через публичный API
	localIPs := p.p2pUI.GetLocalAddresses()

	var localAddresses string

	if len(localIPs) == 0 {
		localAddresses = "Не удалось определить локальные IP адреса\n\n"
	} else {
		localAddresses = "=== Локальные адреса для подключения ===\n\n"
		for i, addr := range localIPs {
			localAddresses += fmt.Sprintf("%d. %s", i+1, addr)
		}
		localAddresses += "\n"
	}

	localAddresses += "=== Как использовать ===\n"
	localAddresses += "1. Нажмите 'Копировать' у нужного адреса\n"
	localAddresses += "2. На другом ПК вставьте в поле 'Добавить контакт'\n"
	localAddresses += "3. Нажмите 'Добавить контакт' или 'Подключиться'\n\n"
	localAddresses += "Примечание: Оба ПК должны быть в одной сети (Wi-Fi/кабель)"

	// Создаём кастомный диалог с кнопками копирования
	content := container.NewVBox()

	if len(localIPs) > 0 {
		content.Add(widget.NewLabel("=== Локальные адреса ==="))
		content.Add(widget.NewSeparator())

		for i, addr := range localIPs {
			addrLabel := widget.NewLabel(fmt.Sprintf("%d. %s", i+1, addr))
			addrLabel.Wrapping = fyne.TextWrapBreak

			copyBtn := widget.NewButtonWithIcon("Копировать", theme.ContentCopyIcon(), func() {
				window.Clipboard().SetContent(addr)
				p.showInfoDialog("Скопировано", fmt.Sprintf("Адрес %d скопирован в буфер обмена", i+1))
			})

			row := container.NewBorder(nil, copyBtn, nil, nil, addrLabel)
			content.Add(row)
		}

		content.Add(widget.NewSeparator())
	}

	instructions := widget.NewLabel(localAddresses)
	instructions.Wrapping = fyne.TextWrapWord
	content.Add(instructions)

	scroll := container.NewScroll(content)
	scroll.SetMinSize(fyne.NewSize(500, 400))

	d := dialog.NewCustom("Локальные адреса", "Закрыть", scroll, window)
	d.Show()
}

// refreshConnectionStatus обновляет статус подключения
func (p *Panel) refreshConnectionStatus() {
	if p.connectionStatusLabel == nil {
		return
	}

	if p.p2pUI == nil {
		p.connectionStatusLabel.SetText("Статус: P2P не запущен")
		p.peersCountLabel.SetText("Подключённые пиры: 0")
		if p.natStatusLabel != nil {
			p.natStatusLabel.SetText("NAT: неизвестно")
		}
		return
	}

	status := p.p2pUI.GetStatus()

	if status.IsRunning {
		p.connectionStatusLabel.SetText("Статус: подключено")
		p.peersCountLabel.SetText(fmt.Sprintf("Подключённые пиры: %d", status.ConnectedPeers))

		natInfo := p.p2pUI.GetNATStatus()
		if p.natStatusLabel != nil {
			p.natStatusLabel.SetText(fmt.Sprintf("NAT: %s", natInfo.Message))
		}
	} else {
		p.connectionStatusLabel.SetText("Статус: отключено")
		p.peersCountLabel.SetText("Подключённые пиры: 0")
		if p.natStatusLabel != nil {
			p.natStatusLabel.SetText("NAT: неизвестно")
		}
	}
}

// showErrorDialog показывает диалог ошибки
func (p *Panel) showErrorDialog(title, message string) {
	window := p.chatsUI.GetWindow()
	if window == nil {
		fmt.Printf("[%s] %s\n", title, message)
		return
	}
	dialog.ShowError(fmt.Errorf("%s", message), window)
}

// showInfoDialog показывает информационный диалог
func (p *Panel) showInfoDialog(title, message string) {
	window := p.chatsUI.GetWindow()
	if window == nil {
		fmt.Printf("[%s] %s\n", title, message)
		return
	}
	dialog.ShowInformation(title, message, window)
}
