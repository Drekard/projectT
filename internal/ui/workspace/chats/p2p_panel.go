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

// createControlPanel создает панель управления P2P с тремя вкладками
func (ui *UI) createControlPanel() *fyne.Container {
	// Создаем вкладки
	tabs := container.NewAppTabs(
		container.NewTabItem("Контакты", ui.createContactsTab()),
		container.NewTabItem("Настройки P2P", ui.createSettingsTab()),
		container.NewTabItem("Сеть", ui.createNetworkTab()),
	)

	tabs.SetTabLocation(container.TabLocationTop)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})

	return container.NewStack(bg, tabs)
}

// ============================================================================
// Вкладка "Контакты"
// ============================================================================

// createContactsTab создает вкладку с контактами
func (ui *UI) createContactsTab() fyne.CanvasObject {
	// Заголовок
	title := widget.NewLabel("Контакты")
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка добавления контакта
	addButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		ui.showAddContactDialog()
	})
	addButton.Importance = widget.HighImportance

	// Список контактов
	ui.contactsListInPanel = container.NewVBox()
	ui.loadContactsList()

	// Разделитель
	sep := widget.NewSeparator()

	// Разделитель для добавления контакта вручную
	manualLabel := widget.NewLabel("Добавить контакт по адресу")
	manualLabel.TextStyle = fyne.TextStyle{Italic: true}

	ui.addressEntry = widget.NewEntry()
	ui.addressEntry.SetPlaceHolder("projectt:peerid@/ip4/.../tcp/.../p2p/...")

	ui.usernameEntry = widget.NewEntry()
	ui.usernameEntry.SetPlaceHolder("Имя контакта (необязательно)")

	addManualButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		ui.addContactByAddress()
	})
	addManualButton.Importance = widget.HighImportance

	manualSection := container.NewVBox(
		manualLabel,
		ui.addressEntry,
		ui.usernameEntry,
		addManualButton,
	)

	content := container.NewVBox(
		title,
		addButton,
		sep,
		ui.contactsListInPanel,
		widget.NewSeparator(),
		manualSection,
	)

	return container.NewScroll(content)
}

// loadContactsList загружает список контактов из базы данных
func (ui *UI) loadContactsList() {
	if ui.contactsListInPanel == nil {
		return
	}

	ui.contactsListInPanel.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.contactsListInPanel.Add(emptyLabel)
		ui.contactsListInPanel.Refresh()
		return
	}

	// Получаем контакты из базы данных
	contacts, err := ui.p2pUI.GetContacts()
	if err != nil {
		emptyLabel := widget.NewLabel(fmt.Sprintf("Ошибка загрузки контактов: %v", err))
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.contactsListInPanel.Add(emptyLabel)
		ui.contactsListInPanel.Refresh()
		return
	}

	if len(contacts) == 0 {
		emptyLabel := widget.NewLabel("Список контактов пуст")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.contactsListInPanel.Add(emptyLabel)
	} else {
		for _, contact := range contacts {
			contactItem := ui.createContactItem(contact)
			ui.contactsListInPanel.Add(contactItem)
		}
	}

	ui.contactsListInPanel.Refresh()
}

// createContactItem создает элемент контакта
func (ui *UI) createContactItem(contact *models.Contact) *fyne.Container {
	// Индикатор статуса (зеленый если онлайн)
	statusInd := canvas.NewCircle(color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Проверяем, подключен ли контакт
	if ui.p2pUI != nil && contact.PeerID != "" {
		connectedPeers := ui.p2pUI.GetConnectedPeers()
		for _, peer := range connectedPeers {
			if peer.PeerID == contact.PeerID {
				statusInd.FillColor = color.RGBA{R: 76, G: 175, B: 80, A: 255}
				break
			}
		}
	}

	// Имя контакта
	nameLabel := widget.NewLabel(contact.Username)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// PeerID (сокращенный)
	peerIDText := "нет ID"
	if contact.PeerID != "" {
		peerIDText = contact.PeerID
		if len(peerIDText) > 16 {
			peerIDText = peerIDText[:8] + "..." + peerIDText[len(peerIDText)-8:]
		}
	}
	peerIDLabel := widget.NewLabel(peerIDText)
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка начала чата
	chatBtn := widget.NewButtonWithIcon("", theme.MailComposeIcon(), func() {
		ui.openChatWithContact(contact)
	})

	// Кнопка подключения
	connectBtn := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		ui.connectToContactByContact(contact)
	})

	// Кнопка удаления
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		ui.deleteContact(contact)
	})

	content := container.NewBorder(
		nil,
		container.NewHBox(chatBtn, connectBtn, deleteBtn),
		container.NewHBox(statusInd, container.NewVBox(nameLabel, peerIDLabel)),
		nil,
		widget.NewSeparator(),
	)

	return content
}

// showAddContactDialog показывает диалог добавления контакта
func (ui *UI) showAddContactDialog() {
	if ui.window == nil {
		return
	}

	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("projectt:peerid@/ip4/.../tcp/.../p2p/...")
	addressEntry.MultiLine = true
	addressEntry.Wrapping = fyne.TextWrapBreak

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Имя контакта (необязательно)")

	content := container.NewVBox(
		widget.NewLabel("Введите адрес контакта:"),
		addressEntry,
		widget.NewLabel("Имя (необязательно):"),
		usernameEntry,
	)

	d := dialog.NewCustomConfirm(
		"Добавить контакт",
		"Добавить",
		"Отмена",
		content,
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if addressEntry.Text == "" {
				ui.showErrorDialog("Ошибка", "Введите адрес контакта")
				return
			}

			if ui.p2pUI == nil {
				ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
				return
			}

			err := ui.p2pUI.AddContactByAddress(addressEntry.Text, usernameEntry.Text)
			if err != nil {
				ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить контакт: %v", err))
				return
			}

			ui.showInfoDialog("Успешно", "Контакт добавлен")
			ui.loadContactsList()

			// Обновляем список контактов в левой панели
			ui.RefreshContactsList()
		},
		ui.window,
	)
	d.Show()
}

// openChatWithContact открывает чат с контактом
func (ui *UI) openChatWithContact(contact *models.Contact) {
	// Открываем чат через публичный метод
	if contact.PeerID != "" {
		ui.OpenPeerChat(contact.PeerID, contact.Username)
	} else {
		ui.showErrorDialog("Ошибка", "У контакта нет PeerID")
	}
}

// connectToContactByContact подключается к контакту
func (ui *UI) connectToContactByContact(contact *models.Contact) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	if contact.PeerID == "" {
		ui.showErrorDialog("Ошибка", "У контакта нет PeerID")
		return
	}

	// Получаем multiaddr контакта
	multiaddr := contact.Multiaddr
	if multiaddr == "" {
		ui.showErrorDialog("Ошибка", "У контакта нет адреса")
		return
	}

	addrStr := fmt.Sprintf("%s@%s", contact.PeerID, multiaddr)

	err := ui.p2pUI.ConnectToContact(addrStr)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось подключиться: %v", err))
		return
	}

	ui.showInfoDialog("Подключение", "Попытка подключения к контакту...")
}

// deleteContact удаляет контакт
func (ui *UI) deleteContact(contact *models.Contact) {
	if ui.window == nil {
		return
	}

	dialog.ShowConfirm(
		"Удаление контакта",
		fmt.Sprintf("Вы действительно хотите удалить контакт \"%s\"?", contact.Username),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if ui.p2pUI == nil {
				ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
				return
			}

			err := ui.p2pUI.DeleteContact(contact.ID)
			if err != nil {
				ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось удалить контакт: %v", err))
				return
			}

			ui.loadContactsList()

			// Обновляем список контактов в левой панели
			ui.RefreshContactsList()
		},
		ui.window,
	)
}

// ============================================================================
// Вкладка "Настройки P2P"
// ============================================================================

// createSettingsTab создает вкладку с настройками P2P
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
	sectionTitle := widget.NewLabel("Ваш адрес")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	ui.myAddressLabel = widget.NewLabel("Адрес: P2P не запущен")

	copyButton := widget.NewButtonWithIcon("Копировать", theme.ContentCopyIcon(), func() {
		ui.copyMyAddress()
	})

	checkPortButton := widget.NewButton("Проверить порт", func() {
		ui.checkPortAccessibility()
	})

	// Кнопка показа локальных адресов
	showLocalButton := widget.NewButton("Локальные адреса", func() {
		ui.showLocalAddresses()
	})

	addressRow := container.NewHBox(ui.myAddressLabel, copyButton)
	buttonsRow := container.NewHBox(checkPortButton, showLocalButton)

	return container.NewVBox(sectionTitle, addressRow, buttonsRow)
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
	saveSettingsBtn := widget.NewButtonWithIcon("Сохранить", theme.DocumentSaveIcon(), func() {
		ui.saveP2PSettings()
	})

	restartBtn := widget.NewButtonWithIcon("Применить и перезапустить", theme.ViewRefreshIcon(), func() {
		ui.restartP2PWithNewSettings()
	})
	restartBtn.Importance = widget.HighImportance

	buttonsRow := container.NewHBox(saveSettingsBtn, restartBtn)

	// Загружаем настройки P2P при создании панели
	ui.loadP2PSettings()

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

// ============================================================================
// Вкладка "Сеть"
// ============================================================================

// createNetworkTab создает вкладку с сетью
func (ui *UI) createNetworkTab() fyne.CanvasObject {
	// === Подключение по адресу ===
	connectSection := ui.createConnectByAddressSection()

	// === Состояние подключения ===
	connectionSection := ui.createConnectionSection()

	// === Подключённые пиры ===
	connectedSection := ui.createConnectedPeersSection()

	// === Bootstrap пиры ===
	bootstrapSection := ui.createBootstrapSection()

	// === Обнаруженные пиры ===
	discoveredSection := ui.createDiscoveredPeersSection()

	// === Профили (из таблицы profiles) ===
	profilesSection := ui.createProfilesSection()

	content := container.NewVBox(
		widget.NewSeparator(),
		connectSection,
		widget.NewSeparator(),
		connectionSection,
		widget.NewSeparator(),
		connectedSection,
		widget.NewSeparator(),
		bootstrapSection,
		widget.NewSeparator(),
		discoveredSection,
		widget.NewSeparator(),
		profilesSection,
	)

	return container.NewScroll(content)
}

// createConnectByAddressSection создает секцию подключения по адресу
func (ui *UI) createConnectByAddressSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Подключение по адресу")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	instruction := widget.NewLabel("Введите адрес пира для подключения (без добавления в контакты)")
	instruction.TextStyle = fyne.TextStyle{Italic: true}
	instruction.Wrapping = fyne.TextWrapWord

	// Поле ввода адреса
	connectAddressEntry := widget.NewEntry()
	connectAddressEntry.SetPlaceHolder("projectt:peerid@/ip4/.../tcp/.../p2p/...")
	connectAddressEntry.MultiLine = false

	// Кнопка подключения
	connectButton := widget.NewButtonWithIcon("Подключиться", theme.FolderIcon(), func() {
		addrStr := connectAddressEntry.Text
		if addrStr == "" {
			ui.showErrorDialog("Ошибка", "Введите адрес пира")
			return
		}

		if ui.p2pUI == nil {
			ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
			return
		}

		err := ui.p2pUI.ConnectToContact(addrStr)
		if err != nil {
			ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось подключиться: %v", err))
			return
		}

		ui.showInfoDialog("Подключение", "Попытка подключения к пиру...")
		connectAddressEntry.SetText("")

		// Обновляем список подключённых пиров через пару секунд
		time.AfterFunc(3*time.Second, func() {
			ui.loadConnectedPeers()
		})
	})
	connectButton.Importance = widget.HighImportance

	return container.NewVBox(
		sectionTitle,
		instruction,
		connectAddressEntry,
		connectButton,
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

	return container.NewVBox(headerRow, ui.connectedPeersList)
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

// createProfilesSection создает секцию профилей из таблицы profiles
func (ui *UI) createProfilesSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Профили")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		ui.loadProfiles()
	})

	headerRow := container.NewBorder(nil, refreshBtn, nil, nil,
		widget.NewLabel(sectionTitle.Text),
	)

	// Список профилей
	profilesList := container.NewVBox()

	// Сохраняем ссылку для последующего обновления
	ui.profilesListInPanel = profilesList

	return container.NewVBox(headerRow, profilesList)
}

// ============================================================================
// Методы загрузки и отображения данных
// ============================================================================

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

// showLocalAddresses показывает локальные адреса для подключения в одной сети
func (ui *UI) showLocalAddresses() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	if ui.window == nil {
		ui.showErrorDialog("Ошибка", "Окно не инициализировано")
		return
	}

	status := ui.p2pUI.GetStatus()

	if !status.IsRunning {
		ui.showErrorDialog("Ошибка", "P2P не запущен")
		return
	}

	// Получаем локальные адреса через публичный API
	localIPs := ui.p2pUI.GetLocalAddresses()

	var localAddresses string

	if len(localIPs) == 0 {
		localAddresses = "Не удалось определить локальные IP адреса\n\n"
	} else {
		localAddresses = "=== Локальные адреса для подключения ===\n\n"
		for i, addr := range localIPs {
			localAddresses += fmt.Sprintf("%d. %s\n", i+1, addr)
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
				ui.window.Clipboard().SetContent(addr)
				ui.showInfoDialog("Скопировано", fmt.Sprintf("Адрес %d скопирован в буфер обмена", i+1))
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

	d := dialog.NewCustom("Локальные адреса", "Закрыть", scroll, ui.window)
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

	// Обновляем список контактов в левой панели
	ui.RefreshContactsList()

	// Обновляем список контактов во вкладке "Контакты"
	ui.loadContactsList()
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

	// Кнопка начала чата
	chatBtn := widget.NewButtonWithIcon("", theme.MailComposeIcon(), func() {
		ui.openChatWithConnectedPeer(peer.PeerID, peer.Username)
	})

	// Кнопка добавления в контакты
	addContactBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		ui.addConnectedPeerToContacts(peer.PeerID)
	})

	// Кнопка отключения
	disconnectBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		ui.disconnectFromPeer(peer.PeerID)
	})

	content := container.NewBorder(
		nil, nil,
		container.NewHBox(statusInd, container.NewVBox(nameLabel, peerIDLabel)),
		container.NewHBox(latencyLabel, chatBtn, addContactBtn, disconnectBtn),
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

	ui.showInfoDialog("Успешно", "Настройки P2P сохранены\n\nДля применения настроек нажмите 'Применить и перезапустить P2P'")
}

// restartP2PWithNewSettings перезапускает P2P с новыми настройками
func (ui *UI) restartP2PWithNewSettings() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
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
			ui.saveP2PSettingsSilent()

			// Останавливаем P2P
			if err := ui.p2pUI.Stop(); err != nil {
				ui.showErrorDialog("Ошибка", fmt.Sprintf("Ошибка остановки P2P: %v", err))
				return
			}

			// Запускаем P2P заново
			if err := ui.p2pUI.Start(); err != nil {
				ui.showErrorDialog("Ошибка", fmt.Sprintf("Ошибка запуска P2P: %v", err))
				return
			}

			ui.showInfoDialog("P2P перезапущен", "Настройки применены успешно")

			// Обновляем отображение
			ui.refreshConnectionStatus()
			ui.loadConnectedPeers()
		},
		ui.window,
	)
}

// saveP2PSettingsSilent сохраняет настройки без показа диалога
func (ui *UI) saveP2PSettingsSilent() {
	if ui.p2pUI == nil {
		return
	}

	var port int
	if _, err := fmt.Sscanf(ui.portEntry.Text, "%d", &port); err != nil {
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

	_ = ui.p2pUI.UpdateSettings(settings)
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
		ui.connectToDiscoveredPeer(peerID)
	})

	content := container.NewBorder(
		nil, connectBtn,
		nil, nil,
		container.NewVBox(peerIDLabel, lastSeenLabel),
	)

	return content
}

// loadProfiles загружает профили из таблицы profiles
func (ui *UI) loadProfiles() {
	if ui.profilesListInPanel == nil {
		return
	}

	ui.profilesListInPanel.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.profilesListInPanel.Add(emptyLabel)
		ui.profilesListInPanel.Refresh()
		return
	}

	// Получаем профили из базы данных
	profiles, err := ui.p2pUI.GetProfiles()
	if err != nil {
		emptyLabel := widget.NewLabel(fmt.Sprintf("Ошибка загрузки профилей: %v", err))
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.profilesListInPanel.Add(emptyLabel)
		ui.profilesListInPanel.Refresh()
		return
	}

	if len(profiles) == 0 {
		emptyLabel := widget.NewLabel("Нет профилей")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.profilesListInPanel.Add(emptyLabel)
	} else {
		for _, profile := range profiles {
			profileItem := ui.createProfileItem(profile)
			ui.profilesListInPanel.Add(profileItem)
		}
	}

	ui.profilesListInPanel.Refresh()
}

// createProfileItem создает элемент профиля
func (ui *UI) createProfileItem(profile *models.Profile) *fyne.Container {
	// Имя профиля (используем Title)
	nameLabel := widget.NewLabel(profile.Title)
	if profile.Title == "" {
		nameLabel = widget.NewLabel(profile.Username)
	}
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// PeerID (сокращённый)
	peerIDShort := profile.PeerID
	if len(peerIDShort) > 16 {
		peerIDShort = peerIDShort[:8] + "..." + peerIDShort[len(peerIDShort)-8:]
	}
	peerIDLabel := widget.NewLabel(peerIDShort)
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка подключения
	connectBtn := widget.NewButtonWithIcon("Подключиться", theme.FolderIcon(), func() {
		ui.connectToProfile(profile)
	})

	content := container.NewBorder(
		nil, connectBtn,
		nil, nil,
		container.NewVBox(nameLabel, peerIDLabel),
		widget.NewSeparator(),
	)

	return content
}

// connectToProfile подключается к профилю
func (ui *UI) connectToProfile(profile *models.Profile) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	if profile.PeerID == "" {
		ui.showErrorDialog("Ошибка", "У профиля нет PeerID")
		return
	}

	// Для профилей нужен multiaddr, но в модели Profile его нет
	// Поэтому показываем сообщение что подключение невозможно без адреса
	ui.showErrorDialog("Ошибка", "Невозможно подключиться к профилю без адреса (multiaddr)")
}

// ============================================================================
// Вспомогательные методы
// ============================================================================

// showInfoDialog показывает информационный диалог
func (ui *UI) showInfoDialog(title, message string) {
	if ui.window == nil {
		fmt.Printf("[%s] %s\n", title, message)
		return
	}
	dialog.ShowInformation(title, message, ui.window)
}

// disconnectFromPeer отключается от пира
func (ui *UI) disconnectFromPeer(peerID string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	// Показываем диалог подтверждения
	dialog.ShowConfirm(
		"Отключение от пира",
		fmt.Sprintf("Вы действительно хотите отключиться от пира %s?", peerID[:8]+"..."),
		func(ok bool) {
			if !ok {
				return
			}

			err := ui.p2pUI.DisconnectPeer(peerID)
			if err != nil {
				ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось отключиться от пира: %v", err))
				return
			}

			ui.showInfoDialog("Успешно", "Отключено от пира")
			// Обновляем список подключённых пиров
			ui.loadConnectedPeers()
		},
		ui.window,
	)
}

// addConnectedPeerToContacts добавляет подключённого пира в контакты
func (ui *UI) addConnectedPeerToContacts(peerID string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	if ui.window == nil {
		ui.showErrorDialog("Ошибка", "Окно не инициализировано")
		return
	}

	// Получаем информацию о пире для получения multiaddr
	peerInfo := ui.p2pUI.GetPeerInfo(peerID)
	if peerInfo == nil {
		ui.showErrorDialog("Ошибка", "Не удалось получить информацию о пире")
		return
	}

	// Получаем multiaddr пира
	multiaddr := peerInfo.Address
	if multiaddr == "" {
		// Пробуем получить из peerstore
		addrs := ui.p2pUI.GetPeerAddresses(peerID)
		if len(addrs) > 0 {
			multiaddr = addrs[0]
		}
	}

	if multiaddr == "" {
		ui.showErrorDialog("Ошибка", "Не удалось получить адрес пира")
		return
	}

	// Формируем строку адреса в формате peerid@multiaddr
	addrStr := fmt.Sprintf("%s@%s", peerID, multiaddr)

	// Показываем диалог подтверждения с полем для имени
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Имя контакта (необязательно)")
	usernameEntry.SetText(peerInfo.Username)

	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Добавить пир %s в контакты?", peerID[:8]+"...")),
		widget.NewLabel(fmt.Sprintf("Адрес: %s", multiaddr)),
		usernameEntry,
	)

	d := dialog.NewCustomConfirm("Добавление контакта", "Добавить", "Отмена", content, func(confirmed bool) {
		if !confirmed {
			return
		}

		// Добавляем контакт через P2P сервис
		err := ui.p2pUI.AddContactByAddress(addrStr, usernameEntry.Text)
		if err != nil {
			ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить контакт: %v", err))
			return
		}

		ui.showInfoDialog("Успешно", "Контакт добавлен")

		// Обновляем список контактов в левой панели
		ui.RefreshContactsList()

		// Обновляем список контактов во вкладке "Контакты"
		ui.loadContactsList()
	}, ui.window)

	d.Show()
}

// openChatWithConnectedPeer открывает чат с подключённым пиром (без добавления в контакты)
func (ui *UI) openChatWithConnectedPeer(peerID, username string) {
	// Открываем чат через публичный метод
	ui.OpenPeerChat(peerID, username)
}

// connectToDiscoveredPeer подключается к обнаруженному пиру
func (ui *UI) connectToDiscoveredPeer(peerID string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	err := ui.p2pUI.ConnectToDiscoveredPeer(peerID)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось подключиться к пиру: %v", err))
		return
	}

	ui.showInfoDialog("Подключение", fmt.Sprintf("Попытка подключения к пиру %s...", peerID[:8]+"..."))

	// Обновляем список подключённых пиров через пару секунд
	time.AfterFunc(3*time.Second, func() {
		ui.loadConnectedPeers()
	})
}
