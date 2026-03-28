// Package p2p содержит компоненты панели управления P2P
package p2p

import (
	"fmt"
	"image/color"
	"log"
	"time"

	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createNetworkTab создает вкладку с сетью
func (p *Panel) createNetworkTab() fyne.CanvasObject {
	// === Подключение по адресу ===
	connectSection := p.createConnectByAddressSection()

	// === Состояние подключения ===
	connectionSection := p.createConnectionSection()

	// === Подключённые пиры ===
	connectedSection := p.createConnectedPeersSection()

	// === Bootstrap пиры ===
	bootstrapSection := p.createBootstrapSection()

	// === Обнаруженные пиры ===
	discoveredSection := p.createDiscoveredPeersSection()

	// === Профили (из таблицы profiles) ===
	profilesSection := p.createProfilesSection()

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
func (p *Panel) createConnectByAddressSection() *fyne.Container {
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
			p.showErrorDialog("Ошибка", "Введите адрес пира")
			return
		}

		if p.p2pUI == nil {
			p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
			return
		}

		err := p.p2pUI.ConnectToContact(addrStr)
		if err != nil {
			p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось подключиться: %v", err))
			return
		}

		p.showInfoDialog("Подключение", "Попытка подключения к пиру...")
		connectAddressEntry.SetText("")

		// Обновляем список подключённых пиров через пару секунд
		time.AfterFunc(3*time.Second, func() {
			p.loadConnectedPeers()
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
func (p *Panel) createConnectionSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Состояние подключения")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	p.connectionStatusLabel = widget.NewLabel("Статус: P2P не запущен")
	p.peersCountLabel = widget.NewLabel("Подключённые пиры: 0")

	// NAT статус
	p.natStatusLabel = widget.NewLabel("NAT: неизвестно")
	p.natStatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка обновления статуса
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		p.refreshConnectionStatus()
	})

	statusRows := container.NewVBox(
		p.connectionStatusLabel,
		p.peersCountLabel,
		p.natStatusLabel,
	)

	return container.NewBorder(nil, refreshBtn, nil, nil,
		container.NewVBox(sectionTitle, statusRows),
	)
}

// createConnectedPeersSection создает секцию подключённых пиров
func (p *Panel) createConnectedPeersSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Подключённые пиры")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		// Запускаем обнаружение пиров
		p.startPeerDiscovery()
		// Загружаем список подключённых пиров
		p.loadConnectedPeers()
	})

	headerRow := container.NewBorder(nil, refreshBtn, nil, nil,
		widget.NewLabel(sectionTitle.Text),
	)

	// Список подключённых пиров
	p.connectedPeersList = container.NewVBox()

	return container.NewVBox(headerRow, p.connectedPeersList)
}

// createBootstrapSection создает секцию bootstrap пиров
func (p *Panel) createBootstrapSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Bootstrap пиры")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Поле ввода bootstrap адреса
	p.bootstrapEntry = widget.NewEntry()
	p.bootstrapEntry.SetPlaceHolder("/ip4/1.2.3.4/tcp/5678/p2p/QmPeerID...")
	p.bootstrapEntry.MultiLine = true
	p.bootstrapEntry.Wrapping = fyne.TextWrapBreak

	// Кнопки
	addBootstrapBtn := widget.NewButtonWithIcon("Добавить", theme.ContentAddIcon(), func() {
		p.addBootstrapPeer()
	})

	refreshBootstrapBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		p.loadBootstrapPeers()
	})

	buttonsRow := container.NewHBox(addBootstrapBtn, refreshBootstrapBtn)

	// Список bootstrap пиров
	p.bootstrapList = container.NewVBox()

	return container.NewVBox(
		sectionTitle,
		p.bootstrapEntry,
		buttonsRow,
		p.bootstrapList,
	)
}

// createDiscoveredPeersSection создает секцию обнаруженных пиров
func (p *Panel) createDiscoveredPeersSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Обнаруженные пиры (mDNS/DHT)")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		// Запускаем обнаружение пиров
		p.startPeerDiscovery()
		// Загружаем список обнаруженных пиров
		p.loadDiscoveredPeers()
	})

	headerRow := container.NewBorder(nil, refreshBtn, nil, nil,
		widget.NewLabel(sectionTitle.Text),
	)

	// Список обнаруженных пиров
	p.discoveredPeersList = container.NewVBox()

	return container.NewVBox(headerRow, p.discoveredPeersList)
}

// createProfilesSection создает секцию профилей из таблицы profiles
func (p *Panel) createProfilesSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Профили")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		p.loadProfiles()
	})

	headerRow := container.NewBorder(nil, refreshBtn, nil, nil,
		widget.NewLabel(sectionTitle.Text),
	)

	// Список профилей
	profilesList := container.NewVBox()

	// Сохраняем ссылку для последующего обновления
	p.profilesListInPanel = profilesList

	return container.NewVBox(headerRow, profilesList)
}

// loadConnectedPeers загружает список подключённых пиров
func (p *Panel) loadConnectedPeers() {
	if p.connectedPeersList == nil {
		return
	}

	p.connectedPeersList.Objects = nil

	if p.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.connectedPeersList.Add(emptyLabel)
		p.connectedPeersList.Refresh()
		return
	}

	peers := p.p2pUI.GetConnectedPeers()

	if len(peers) == 0 {
		emptyLabel := widget.NewLabel("Нет подключённых пиров")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.connectedPeersList.Add(emptyLabel)
	} else {
		for _, peer := range peers {
			peerItem := p.createConnectedPeerItem(peer)
			p.connectedPeersList.Add(peerItem)
		}
	}

	p.connectedPeersList.Refresh()
}

// createConnectedPeerItem создает элемент подключённого пира
func (p *Panel) createConnectedPeerItem(peer *network.PeerInfo) *fyne.Container {
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
		log.Printf("[Chat] 📝 Нажата кнопка создания чата с пиром: %s (%s)", peer.Username, peer.PeerID[:8])
		log.Printf("[Chat] 🔍 ОТЛАДКА: peer.PeerID полный = %s", peer.PeerID)
		log.Printf("[Chat] 🔍 ОТЛАДКА: peer.Username = %s", peer.Username)
		p.openChatWithConnectedPeer(peer.PeerID, peer.Username)
	})

	// Кнопка добавления в контакты
	addContactBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		log.Printf("[Contact] ➕ Нажата кнопка добавления пира в контакты: %s (%s)", peer.Username, peer.PeerID[:8])
		p.addConnectedPeerToContacts(peer.PeerID)
	})

	// Кнопка отключения
	disconnectBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		log.Printf("[Connection] ❌ Нажата кнопка отключения от пира: %s (%s)", peer.Username, peer.PeerID[:8])
		p.disconnectFromPeer(peer.PeerID)
	})

	content := container.NewBorder(
		nil, nil,
		container.NewHBox(statusInd, container.NewVBox(nameLabel, peerIDLabel)),
		container.NewHBox(latencyLabel, chatBtn, addContactBtn, disconnectBtn),
		widget.NewSeparator(),
	)

	return content
}

// loadBootstrapPeers загружает список bootstrap пиров
func (p *Panel) loadBootstrapPeers() {
	if p.bootstrapList == nil {
		return
	}

	p.bootstrapList.Objects = nil

	if p.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.bootstrapList.Add(emptyLabel)
		p.bootstrapList.Refresh()
		return
	}

	peers := p.p2pUI.GetBootstrapPeers()

	if len(peers) == 0 {
		emptyLabel := widget.NewLabel("Нет bootstrap пиров")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.bootstrapList.Add(emptyLabel)
	} else {
		for _, peer := range peers {
			peerItem := p.createBootstrapPeerItem(peer)
			p.bootstrapList.Add(peerItem)
		}
	}

	p.bootstrapList.Refresh()
}

// createBootstrapPeerItem создает элемент bootstrap пира
func (p *Panel) createBootstrapPeerItem(peer *models.BootstrapPeer) *fyne.Container {
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
		p.removeBootstrapPeer(peer.Multiaddr)
	})

	content := container.NewBorder(
		nil, removeBtn,
		nil, nil,
		container.NewVBox(addrLabel, statusLabel),
	)

	return content
}

// addBootstrapPeer добавляет bootstrap пир
func (p *Panel) addBootstrapPeer() {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	addrStr := p.bootstrapEntry.Text
	if addrStr == "" {
		p.showErrorDialog("Ошибка", "Введите адрес bootstrap пира")
		return
	}

	err := p.p2pUI.AddBootstrapPeer(addrStr)
	if err != nil {
		p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить bootstrap пир: %v", err))
		return
	}

	p.showInfoDialog("Успешно", "Bootstrap пир добавлен")
	p.bootstrapEntry.SetText("")
	p.loadBootstrapPeers()
}

// removeBootstrapPeer удаляет bootstrap пир
func (p *Panel) removeBootstrapPeer(multiaddr string) {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	err := p.p2pUI.RemoveBootstrapPeer(multiaddr)
	if err != nil {
		p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось удалить bootstrap пир: %v", err))
		return
	}

	p.loadBootstrapPeers()
}

// loadDiscoveredPeers загружает список обнаруженных пиров
func (p *Panel) loadDiscoveredPeers() {
	if p.discoveredPeersList == nil {
		return
	}

	p.discoveredPeersList.Objects = nil

	if p.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.discoveredPeersList.Add(emptyLabel)
		p.discoveredPeersList.Refresh()
		return
	}

	peers := p.p2pUI.GetDiscoveredPeers()

	if len(peers) == 0 {
		emptyLabel := widget.NewLabel("Нет обнаруженных пиров")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.discoveredPeersList.Add(emptyLabel)
	} else {
		for peerID, lastSeen := range peers {
			peerItem := p.createDiscoveredPeerItem(peerID, lastSeen)
			p.discoveredPeersList.Add(peerItem)
		}
	}

	p.discoveredPeersList.Refresh()
}

// createDiscoveredPeerItem создает элемент обнаруженного пира
func (p *Panel) createDiscoveredPeerItem(peerID string, lastSeen time.Time) *fyne.Container {
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
		p.connectToDiscoveredPeer(peerID)
	})

	content := container.NewBorder(
		nil, connectBtn,
		nil, nil,
		container.NewVBox(peerIDLabel, lastSeenLabel),
	)

	return content
}

// loadProfiles загружает профили из таблицы profiles
func (p *Panel) loadProfiles() {
	if p.profilesListInPanel == nil {
		return
	}

	p.profilesListInPanel.Objects = nil

	if p.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.profilesListInPanel.Add(emptyLabel)
		p.profilesListInPanel.Refresh()
		return
	}

	// Получаем профили из базы данных
	profiles, err := p.p2pUI.GetProfiles()
	if err != nil {
		emptyLabel := widget.NewLabel(fmt.Sprintf("Ошибка загрузки профилей: %v", err))
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.profilesListInPanel.Add(emptyLabel)
		p.profilesListInPanel.Refresh()
		return
	}

	if len(profiles) == 0 {
		emptyLabel := widget.NewLabel("Нет профилей")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.profilesListInPanel.Add(emptyLabel)
	} else {
		for _, profile := range profiles {
			profileItem := p.createProfileItem(profile)
			p.profilesListInPanel.Add(profileItem)
		}
	}

	p.profilesListInPanel.Refresh()
}

// createProfileItem создает элемент профиля
func (p *Panel) createProfileItem(profile *models.Profile) *fyne.Container {
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
		p.connectToProfile(profile)
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
func (p *Panel) connectToProfile(profile *models.Profile) {
	p.showErrorDialog("Ошибка", "Невозможно подключиться к профилю без адреса (multiaddr)")
}

// startPeerDiscovery запускает обнаружение пиров
func (p *Panel) startPeerDiscovery() {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	err := p.p2pUI.StartPeerDiscovery()
	if err != nil {
		p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось запустить обнаружение пиров: %v", err))
		return
	}

	p.showInfoDialog("Обнаружение пиров", "Запущено обнаружение пиров...\nПроверьте секцию 'Подключённые пиры' через несколько секунд")
}

// disconnectFromPeer отключается от пира
func (p *Panel) disconnectFromPeer(peerID string) {
	window := p.chatsUI.GetWindow()
	if window == nil {
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

			err := p.p2pUI.DisconnectPeer(peerID)
			if err != nil {
				p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось отключиться от пира: %v", err))
				return
			}

			p.showInfoDialog("Успешно", "Отключено от пира")
			// Обновляем список подключённых пиров
			p.loadConnectedPeers()
		},
		window,
	)
}

// addConnectedPeerToContacts добавляет подключённого пира в контакты
func (p *Panel) addConnectedPeerToContacts(peerID string) {
	log.Printf("[Contact] 📇 Начало добавления пира в контакты: %s", peerID[:8])

	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		log.Printf("[Contact] ❌ Ошибка: P2P сервис не инициализирован")
		return
	}

	window := p.chatsUI.GetWindow()
	if window == nil {
		p.showErrorDialog("Ошибка", "Окно не инициализировано")
		log.Printf("[Contact] ❌ Ошибка: окно не инициализировано")
		return
	}

	// Получаем информацию о пире для получения multiaddr
	peerInfo := p.p2pUI.GetPeerInfo(peerID)
	if peerInfo == nil {
		p.showErrorDialog("Ошибка", "Не удалось получить информацию о пире")
		log.Printf("[Contact] ❌ Ошибка: не удалось получить информацию о пире %s", peerID[:8])
		return
	}
	log.Printf("[Contact] ℹ️ Получена информация о пире: %s (%s)", peerInfo.Username, peerID[:8])

	// Получаем multiaddr пира
	multiaddr := peerInfo.Address
	if multiaddr == "" {
		// Пробуем получить из peerstore
		addrs := p.p2pUI.GetPeerAddresses(peerID)
		if len(addrs) > 0 {
			multiaddr = addrs[0]
			log.Printf("[Contact] ℹ️ Multiaddr получен из Peerstore: %s", multiaddr[:50]+"...")
		}
	}

	if multiaddr == "" {
		p.showErrorDialog("Ошибка", "Не удалось получить адрес пира")
		log.Printf("[Contact] ❌ Ошибка: не удалось получить адрес пира %s", peerID[:8])
		return
	}

	// Формируем строку адреса в формате peerid@multiaddr
	addrStr := fmt.Sprintf("%s@%s", peerID, multiaddr)
	log.Printf("[Contact] 📍 Сформирован адрес: %s@...", peerID[:8])

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
			log.Printf("[Contact] ⛔ Отменено пользователем добавление контакта: %s", peerID[:8])
			return
		}
		log.Printf("[Contact] ✅ Подтверждено добавление контакта: %s (имя: %s)", peerID[:8], usernameEntry.Text)

		// Добавляем контакт через P2P сервис
		err := p.p2pUI.AddContactByAddress(addrStr, usernameEntry.Text)
		if err != nil {
			log.Printf("[Contact] ❌ Ошибка добавления контакта: %v", err)
			p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить контакт: %v", err))
			return
		}
		log.Printf("[Contact] ✅ Контакт добавлен в БД: %s (%s)", usernameEntry.Text, peerID[:8])

		p.showInfoDialog("Успешно", "Контакт добавлен")
		log.Printf("[Contact] 🔄 Обновление списка контактов в левой панели...")

		// Обновляем список контактов в левой панели
		p.chatsUI.RefreshContactsList()

		// Обновляем список контактов во вкладке "Контакты"
		p.loadContactsList()
		log.Printf("[Contact] ✅ Список контактов обновлён")
	}, window)

	d.Show()
}

// openChatWithConnectedPeer открывает чат с подключённым пиром (без добавления в контакты)
func (p *Panel) openChatWithConnectedPeer(peerID, username string) {
	log.Printf("[Chat] 🗨️ Открытие чата с пиром: %s (%s)", username, peerID[:8])
	// Открываем чат через публичный метод
	p.chatsUI.OpenPeerChat(peerID, username)
	log.Printf("[Chat] ✅ Чат открыт в UI для пира: %s (%s)", username, peerID[:8])
}

// connectToDiscoveredPeer подключается к обнаруженному пиру
func (p *Panel) connectToDiscoveredPeer(peerID string) {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	err := p.p2pUI.ConnectToDiscoveredPeer(peerID)
	if err != nil {
		p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось подключиться к пиру: %v", err))
		return
	}

	p.showInfoDialog("Подключение", fmt.Sprintf("Попытка подключения к пиру %s...", peerID[:8]+"..."))

	// Обновляем список подключённых пиров через пару секунд
	time.AfterFunc(3*time.Second, func() {
		p.loadConnectedPeers()
	})
}
