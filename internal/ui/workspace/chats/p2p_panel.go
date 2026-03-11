package chats

import (
	"fmt"
	"image/color"
	"time"

	"projectT/internal/services/p2p/network"
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createControlPanel создает панель управления P2P (тип 1)
func (ui *UI) createControlPanel() *fyne.Container {
	// Заголовок
	title := widget.NewLabel("Настройки P2P и контакты")
	title.TextStyle = fyne.TextStyle{Bold: true}

	// === Ваш адрес ===
	addressSection := ui.createAddressSection()

	// === Добавить контакт ===
	addContactSection := ui.createAddContactSection()

	// === Состояние подключения ===
	connectionSection := ui.createConnectionSection()

	// === Подключённые пиры ===
	connectedPeersSection := ui.createConnectedPeersSection()

	// === Настройки P2P ===
	p2pSection := ui.createP2PSettingsSection()

	// === Bootstrap пиры ===
	bootstrapSection := ui.createBootstrapSection()

	// === Обнаруженные пиры ===
	discoveredSection := ui.createDiscoveredPeersSection()

	// Собираем все секции
	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		addressSection,
		widget.NewSeparator(),
		addContactSection,
		widget.NewSeparator(),
		connectionSection,
		widget.NewSeparator(),
		connectedPeersSection,
		widget.NewSeparator(),
		p2pSection,
		widget.NewSeparator(),
		bootstrapSection,
		widget.NewSeparator(),
		discoveredSection,
	)

	scroll := container.NewScroll(content)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 45, G: 45, B: 45, A: 255})

	return container.NewStack(bg, scroll)
}

// createAddressSection создает секцию управления адресом
func (ui *UI) createAddressSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Ваш адрес")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	ui.myAddressLabel = widget.NewLabel("Адрес: P2P не запущен")

	copyButton := widget.NewButtonWithIcon("Копировать", theme.ContentCopyIcon(), func() {
		ui.copyMyAddress()
	})

	checkPortButton := widget.NewButton("Проверить порт", func() {
		ui.checkPortAccessibility()
	})

	addressRow := container.NewHBox(ui.myAddressLabel, copyButton)
	buttonsRow := container.NewHBox(checkPortButton)

	return container.NewVBox(sectionTitle, addressRow, buttonsRow)
}

// createAddContactSection создает секцию добавления контакта
func (ui *UI) createAddContactSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Добавить контакт")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Поле ввода адреса
	ui.addressEntry = widget.NewEntry()
	ui.addressEntry.SetPlaceHolder("projectt:peerid@/ip4/.../tcp/.../p2p/...")

	// Поле ввода имени (опционально)
	ui.usernameEntry = widget.NewEntry()
	ui.usernameEntry.SetPlaceHolder("Имя контакта (необязательно)")

	// Кнопки
	addButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		ui.addContactByAddress()
	})
	addButton.Importance = widget.HighImportance

	connectButton := widget.NewButtonWithIcon("Подключиться", theme.FolderIcon(), func() {
		ui.connectToContact()
	})

	buttonsRow := container.NewHBox(addButton, connectButton)

	return container.NewVBox(
		sectionTitle,
		ui.addressEntry,
		ui.usernameEntry,
		buttonsRow,
	)
}

// createConnectionSection создает секцию информации о подключении
func (ui *UI) createConnectionSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Состояние подключения")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	ui.connectionStatusLabel = widget.NewLabel("Статус: P2P не запущен")
	ui.peersCountLabel = widget.NewLabel("Подключённые пиры: 0")

	// NAT статус
	ui.natStatusLabel = widget.NewLabel("NAT: неизвестно")
	ui.natStatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка обновления статуса
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		ui.refreshConnectionStatus()
	})

	statusRows := container.NewVBox(
		ui.connectionStatusLabel,
		ui.peersCountLabel,
		ui.natStatusLabel,
	)

	return container.NewBorder(nil, refreshBtn, nil, nil,
		container.NewVBox(sectionTitle, statusRows),
	)
}

// createConnectedPeersSection создает секцию подключённых пиров
func (ui *UI) createConnectedPeersSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Подключённые пиры")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		// Запускаем обнаружение пиров
		ui.startPeerDiscovery()
		// Загружаем список подключённых пиров
		ui.loadConnectedPeers()
	})

	headerRow := container.NewBorder(nil, refreshBtn, nil, nil,
		widget.NewLabel(sectionTitle.Text),
	)

	// Список подключённых пиров
	ui.connectedPeersList = container.NewVBox()

	// Список контактов в панели управления
	ui.contactsListInPanel = container.NewVBox()

	return container.NewVBox(headerRow, ui.connectedPeersList, ui.contactsListInPanel)
}

// createP2PSettingsSection создает секцию настроек P2P
func (ui *UI) createP2PSettingsSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Настройки P2P")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Порт прослушивания с фоном
	portLabel := widget.NewLabel("Порт прослушивания:")
	portBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	portBg.SetMinSize(fyne.NewSize(100, 30))
	ui.portEntry = widget.NewEntry()
	ui.portEntry.SetText("8080")
	portWrapper := container.NewStack(portBg, ui.portEntry)
	portRow := container.NewHBox(portLabel, portWrapper)

	// Чекбоксы настроек
	ui.natPortMapCheck = widget.NewCheck("NAT Port Mapping (UPnP/NAT-PMP)", nil)
	ui.relayCheck = widget.NewCheck("Relay (обход NAT)", nil)
	ui.autoRelayCheck = widget.NewCheck("Автообнаружение Relay", nil)
	ui.dhtCheck = widget.NewCheck("DHT (глобальное обнаружение)", nil)
	ui.mdnsCheck = widget.NewCheck("mDNS (локальная сеть)", nil)
	ui.stunCheck = widget.NewCheck("STUN клиент", nil)
	ui.helperModeCheck = widget.NewCheck("Режим помощника", nil)

	// STUN сервер с фоном
	stunLabel := widget.NewLabel("STUN сервер:")
	stunBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	stunBg.SetMinSize(fyne.NewSize(200, 30))
	ui.stunServerEntry = widget.NewEntry()
	ui.stunServerEntry.SetText("stun.l.google.com:19302")
	stunWrapper := container.NewStack(stunBg, ui.stunServerEntry)
	stunRow := container.NewHBox(stunLabel, stunWrapper)

	// Кнопки
	loadSettingsBtn := widget.NewButton("Загрузить настройки", func() {
		ui.loadP2PSettings()
	})

	saveSettingsBtn := widget.NewButtonWithIcon("Сохранить настройки", theme.DocumentSaveIcon(), func() {
		ui.saveP2PSettings()
	})

	buttonsRow := container.NewHBox(loadSettingsBtn, saveSettingsBtn)

	return container.NewVBox(
		sectionTitle,
		portRow,
		ui.natPortMapCheck,
		ui.relayCheck,
		ui.autoRelayCheck,
		ui.dhtCheck,
		ui.mdnsCheck,
		ui.stunCheck,
		stunRow,
		ui.helperModeCheck,
		widget.NewSeparator(),
		buttonsRow,
	)
}

// createBootstrapSection создает секцию bootstrap пиров
func (ui *UI) createBootstrapSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Bootstrap пиры")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Поле ввода bootstrap адреса
	ui.bootstrapEntry = widget.NewEntry()
	ui.bootstrapEntry.SetPlaceHolder("/ip4/1.2.3.4/tcp/5678/p2p/QmPeerID...")
	ui.bootstrapEntry.MultiLine = true
	ui.bootstrapEntry.Wrapping = fyne.TextWrapBreak

	// Кнопки
	addBootstrapBtn := widget.NewButtonWithIcon("Добавить", theme.ContentAddIcon(), func() {
		ui.addBootstrapPeer()
	})

	refreshBootstrapBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		ui.loadBootstrapPeers()
	})

	buttonsRow := container.NewHBox(addBootstrapBtn, refreshBootstrapBtn)

	// Список bootstrap пиров
	ui.bootstrapList = container.NewVBox()

	return container.NewVBox(
		sectionTitle,
		ui.bootstrapEntry,
		buttonsRow,
		ui.bootstrapList,
	)
}

// createDiscoveredPeersSection создает секцию обнаруженных пиров
func (ui *UI) createDiscoveredPeersSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Обнаруженные пиры (mDNS/DHT)")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		// Запускаем обнаружение пиров
		ui.startPeerDiscovery()
		// Загружаем список обнаруженных пиров
		ui.loadDiscoveredPeers()
	})

	headerRow := container.NewBorder(nil, refreshBtn, nil, nil,
		widget.NewLabel(sectionTitle.Text),
	)

	// Список обнаруженных пиров
	ui.discoveredPeersList = container.NewVBox()

	return container.NewVBox(headerRow, ui.discoveredPeersList)
}

// refreshConnectionStatus обновляет статус подключения
func (ui *UI) refreshConnectionStatus() {
	if ui.connectionStatusLabel == nil {
		return
	}

	if ui.p2pUI == nil {
		ui.connectionStatusLabel.SetText("Статус: P2P не запущен")
		ui.peersCountLabel.SetText("Подключённые пиры: 0")
		if ui.natStatusLabel != nil {
			ui.natStatusLabel.SetText("NAT: неизвестно")
		}
		return
	}

	status := ui.p2pUI.GetStatus()

	if status.IsRunning {
		ui.connectionStatusLabel.SetText("Статус: подключено")
		ui.peersCountLabel.SetText(fmt.Sprintf("Подключённые пиры: %d", status.ConnectedPeers))

		natInfo := ui.p2pUI.GetNATStatus()
		if ui.natStatusLabel != nil {
			ui.natStatusLabel.SetText(fmt.Sprintf("NAT: %s", natInfo.Message))
		}
	} else {
		ui.connectionStatusLabel.SetText("Статус: отключено")
		ui.peersCountLabel.SetText("Подключённые пиры: 0")
		if ui.natStatusLabel != nil {
			ui.natStatusLabel.SetText("NAT: неизвестно")
		}
	}
}

// copyMyAddress копирует свой адрес в буфер обмена
func (ui *UI) copyMyAddress() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	addr, err := ui.p2pUI.CopyPeerAddress()
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось получить адрес: %v", err))
		return
	}

	// Копируем в буфер обмена
	ui.window.Clipboard().SetContent(addr)
	ui.showInfoDialog("Адрес скопирован", "Ваш адрес скопирован в буфер обмена")
}

// checkPortAccessibility проверяет доступность порта
func (ui *UI) checkPortAccessibility() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	if ui.window == nil {
		ui.showErrorDialog("Ошибка", "Окно не инициализировано")
		return
	}

	// Получаем порт из настроек
	port := 8080
	if ui.portEntry != nil && ui.portEntry.Text != "" {
		_, err := fmt.Sscanf(ui.portEntry.Text, "%d", &port)
		if err != nil {
			ui.showErrorDialog("Ошибка", "Неверный формат порта")
			return
		}
	}

	// Показываем информацию о брандмауэре
	firewallInfo := ui.p2pUI.CheckFirewall(port)

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
	d := dialog.NewInformation("Брандмауэр", message, ui.window)
	d.Show()
}

// addContactByAddress добавляет контакт по адресу
func (ui *UI) addContactByAddress() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	addrStr := ui.addressEntry.Text
	if addrStr == "" {
		ui.showErrorDialog("Ошибка", "Введите адрес контакта")
		return
	}

	username := ui.usernameEntry.Text

	err := ui.p2pUI.AddContactByAddress(addrStr, username)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить контакт: %v", err))
		return
	}

	ui.showInfoDialog("Успешно", "Контакт добавлен")
	ui.addressEntry.SetText("")
	ui.usernameEntry.SetText("")
}

// connectToContact подключается к контакту
func (ui *UI) connectToContact() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	addrStr := ui.addressEntry.Text
	if addrStr == "" {
		ui.showErrorDialog("Ошибка", "Введите адрес контакта")
		return
	}

	err := ui.p2pUI.ConnectToContact(addrStr)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось подключиться: %v", err))
		return
	}

	ui.showInfoDialog("Подключение", "Попытка подключения к пиру...")
}

// startPeerDiscovery запускает обнаружение пиров
func (ui *UI) startPeerDiscovery() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	err := ui.p2pUI.StartPeerDiscovery()
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось запустить обнаружение пиров: %v", err))
		return
	}

	ui.showInfoDialog("Обнаружение пиров", "Запущено обнаружение пиров...\nПроверьте секцию 'Подключённые пиры' через несколько секунд")
}

// loadConnectedPeers загружает список подключённых пиров
func (ui *UI) loadConnectedPeers() {
	if ui.connectedPeersList == nil {
		return
	}

	ui.connectedPeersList.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.connectedPeersList.Add(emptyLabel)
		ui.connectedPeersList.Refresh()
		return
	}

	peers := ui.p2pUI.GetConnectedPeers()

	if len(peers) == 0 {
		emptyLabel := widget.NewLabel("Нет подключённых пиров")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.connectedPeersList.Add(emptyLabel)
	} else {
		for _, peer := range peers {
			peerItem := ui.createConnectedPeerItem(peer)
			ui.connectedPeersList.Add(peerItem)
		}
	}

	ui.connectedPeersList.Refresh()
}

// createConnectedPeerItem создает элемент подключённого пира
func (ui *UI) createConnectedPeerItem(peer *network.PeerInfo) *fyne.Container {
	// Индикатор статуса (зелёный для подключённых)
	statusInd := canvas.NewCircle(color.RGBA{R: 76, G: 175, B: 80, A: 255})

	// Имя
	nameLabel := widget.NewLabel(peer.Username)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// PeerID (сокращённый)
	peerIDShort := peer.PeerID
	if len(peerIDShort) > 16 {
		peerIDShort = peerIDShort[:8] + "..." + peerIDShort[len(peerIDShort)-8:]
	}
	peerIDLabel := widget.NewLabel(peerIDShort)
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Latency
	latencyLabel := widget.NewLabel(fmt.Sprintf("%d мс", peer.LatencyMs))

	// Кнопка отключения
	disconnectBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		// TODO: реализовать отключение от пира
	})

	content := container.NewBorder(
		nil, nil,
		container.NewHBox(statusInd, container.NewVBox(nameLabel, peerIDLabel)),
		container.NewHBox(latencyLabel, disconnectBtn),
		widget.NewSeparator(),
	)

	return content
}

// loadP2PSettings загружает настройки P2P
func (ui *UI) loadP2PSettings() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	settings := ui.p2pUI.GetSettings()

	ui.portEntry.SetText(fmt.Sprintf("%d", settings.ListenPort))
	ui.natPortMapCheck.SetChecked(settings.EnableNATPortMap)
	ui.relayCheck.SetChecked(settings.EnableRelay)
	ui.autoRelayCheck.SetChecked(settings.EnableAutoRelay)
	ui.dhtCheck.SetChecked(settings.EnableDHT)
	ui.mdnsCheck.SetChecked(settings.EnableMDNS)
	ui.stunCheck.SetChecked(settings.EnableSTUN)
	ui.stunServerEntry.SetText(settings.STUNServer)
	ui.helperModeCheck.SetChecked(settings.EnableHelperMode)
}

// saveP2PSettings сохраняет настройки P2P
func (ui *UI) saveP2PSettings() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	var port int
	if _, err := fmt.Sscanf(ui.portEntry.Text, "%d", &port); err != nil {
		ui.showErrorDialog("Ошибка", "Неверный формат порта")
		return
	}

	settings := &network.P2PSettings{
		ListenPort:       port,
		EnableNATPortMap: ui.natPortMapCheck.Checked,
		EnableRelay:      ui.relayCheck.Checked,
		EnableAutoRelay:  ui.autoRelayCheck.Checked,
		EnableDHT:        ui.dhtCheck.Checked,
		EnableMDNS:       ui.mdnsCheck.Checked,
		EnableSTUN:       ui.stunCheck.Checked,
		STUNServer:       ui.stunServerEntry.Text,
		EnableHelperMode: ui.helperModeCheck.Checked,
	}

	err := ui.p2pUI.UpdateSettings(settings)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось сохранить настройки: %v", err))
		return
	}

	ui.showInfoDialog("Успешно", "Настройки P2P сохранены")
}

// loadBootstrapPeers загружает список bootstrap пиров
func (ui *UI) loadBootstrapPeers() {
	if ui.bootstrapList == nil {
		return
	}

	ui.bootstrapList.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.bootstrapList.Add(emptyLabel)
		ui.bootstrapList.Refresh()
		return
	}

	peers := ui.p2pUI.GetBootstrapPeers()

	if len(peers) == 0 {
		emptyLabel := widget.NewLabel("Нет bootstrap пиров")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.bootstrapList.Add(emptyLabel)
	} else {
		for _, peer := range peers {
			peerItem := ui.createBootstrapPeerItem(peer)
			ui.bootstrapList.Add(peerItem)
		}
	}

	ui.bootstrapList.Refresh()
}

// createBootstrapPeerItem создает элемент bootstrap пира
func (ui *UI) createBootstrapPeerItem(peer *models.BootstrapPeer) *fyne.Container {
	// Multiaddr (сокращённый)
	addrShort := peer.Multiaddr
	if len(addrShort) > 50 {
		addrShort = addrShort[:30] + "..." + addrShort[len(addrShort)-20:]
	}
	addrLabel := widget.NewLabel(addrShort)
	addrLabel.Wrapping = fyne.TextWrapBreak

	// Статус
	statusText := "неактивен"
	if peer.IsActive {
		statusText = "активен"
	}
	statusLabel := widget.NewLabel(statusText)
	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка удаления
	removeBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		ui.removeBootstrapPeer(peer.Multiaddr)
	})

	content := container.NewBorder(
		nil, removeBtn,
		nil, nil,
		container.NewVBox(addrLabel, statusLabel),
	)

	return content
}

// addBootstrapPeer добавляет bootstrap пир
func (ui *UI) addBootstrapPeer() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	addrStr := ui.bootstrapEntry.Text
	if addrStr == "" {
		ui.showErrorDialog("Ошибка", "Введите адрес bootstrap пира")
		return
	}

	err := ui.p2pUI.AddBootstrapPeer(addrStr)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить bootstrap пир: %v", err))
		return
	}

	ui.showInfoDialog("Успешно", "Bootstrap пир добавлен")
	ui.bootstrapEntry.SetText("")
	ui.loadBootstrapPeers()
}

// removeBootstrapPeer удаляет bootstrap пир
func (ui *UI) removeBootstrapPeer(multiaddr string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	err := ui.p2pUI.RemoveBootstrapPeer(multiaddr)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось удалить bootstrap пир: %v", err))
		return
	}

	ui.loadBootstrapPeers()
}

// loadDiscoveredPeers загружает список обнаруженных пиров
func (ui *UI) loadDiscoveredPeers() {
	if ui.discoveredPeersList == nil {
		return
	}

	ui.discoveredPeersList.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.discoveredPeersList.Add(emptyLabel)
		ui.discoveredPeersList.Refresh()
		return
	}

	peers := ui.p2pUI.GetDiscoveredPeers()

	if len(peers) == 0 {
		emptyLabel := widget.NewLabel("Нет обнаруженных пиров")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.discoveredPeersList.Add(emptyLabel)
	} else {
		for peerID, lastSeen := range peers {
			peerItem := ui.createDiscoveredPeerItem(peerID, lastSeen)
			ui.discoveredPeersList.Add(peerItem)
		}
	}

	ui.discoveredPeersList.Refresh()
}

// createDiscoveredPeerItem создает элемент обнаруженного пира
func (ui *UI) createDiscoveredPeerItem(peerID string, lastSeen time.Time) *fyne.Container {
	// PeerID (сокращённый)
	peerIDShort := peerID
	if len(peerIDShort) > 16 {
		peerIDShort = peerIDShort[:8] + "..." + peerIDShort[len(peerIDShort)-8:]
	}
	peerIDLabel := widget.NewLabel(peerIDShort)

	// Время последнего обнаружения
	lastSeenLabel := widget.NewLabel(fmt.Sprintf("Обнаружен: %s", lastSeen.Format("15:04:05")))
	lastSeenLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка подключения
	connectBtn := widget.NewButton("Подключиться", func() {
		ui.addressEntry.SetText(fmt.Sprintf("projectt:%s@...", peerID))
	})

	content := container.NewBorder(
		nil, connectBtn,
		nil, nil,
		container.NewVBox(peerIDLabel, lastSeenLabel),
	)

	return content
}

// showInfoDialog показывает информационный диалог
func (ui *UI) showInfoDialog(title, message string) {
	if ui.window == nil {
		fmt.Printf("[%s] %s\n", title, message)
		return
	}
	dialog.ShowInformation(title, message, ui.window)
}
