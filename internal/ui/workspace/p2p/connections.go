// Package p2p содержит компонент вкладки "Подключение"
package p2p

import (
	"fmt"
	"image/color"
	"time"

	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createProfilesTab создает вкладку с подключениями и профилями
func (ui *UI) createProfilesTab() fyne.CanvasObject {
	createConnectSection := ui.createConnectByAddressSection()

	// === Подключённые пиры ===
	connectedSection := ui.createConnectedPeersSection()

	// === Обнаруженные пиры ===
	discoveredSection := ui.createDiscoveredPeersSection()

	// === Профили (из таблицы profiles) ===
	profilesSection := ui.createProfilesSection()

	content := container.NewVBox(
		widget.NewSeparator(),
		createConnectSection,
		widget.NewSeparator(),
		connectedSection,
		widget.NewSeparator(),
		discoveredSection,
		widget.NewSeparator(),
		profilesSection,
	)

	return container.NewScroll(content)
}

// createConnectedPeersSection создает секцию подключённых пиров
func (ui *UI) createConnectedPeersSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Подключённые пиры")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		ui.loadConnectedPeers()
	})

	headerRow := container.NewHBox(sectionTitle, refreshBtn)

	ui.connectedPeersList = container.NewVBox()

	return container.NewVBox(headerRow, ui.connectedPeersList)
}

// createDiscoveredPeersSection создает секцию обнаруженных пиров
func (ui *UI) createDiscoveredPeersSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Обнаруженные пиры (DHT/mDNS)")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		ui.loadDiscoveredPeers()
	})

	headerRow := container.NewHBox(sectionTitle, refreshBtn)

	ui.discoveredPeersList = container.NewVBox()

	return container.NewVBox(headerRow, ui.discoveredPeersList)
}

// createProfilesSection создает секцию профилей
func (ui *UI) createProfilesSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Профили пиров")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		ui.loadProfiles()
	})

	headerRow := container.NewHBox(sectionTitle, refreshBtn)

	ui.profilesList = container.NewVBox()

	return container.NewVBox(headerRow, ui.profilesList)
}

// createConnectByAddressSection создает секцию подключения по адресу
func (ui *UI) createConnectByAddressSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Подключение по адресу")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

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
	connectAddressWrapper := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	connectAddressWrapper.SetMinSize(fyne.NewSize(450, 30))
	connect := container.NewStack(connectAddressWrapper, connectAddressEntry)

	return container.NewVBox(
		sectionTitle,
		container.NewHBox(connect, connectButton),
	)
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
	// Имя пира
	nameLabel := widget.NewLabel(peer.Username)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// PeerID (сокращённый)
	peerIDShort := peer.PeerID
	if len(peerIDShort) > 8 {
		peerIDShort = peerIDShort[:8]
	}
	peerIDLabel := widget.NewLabel(fmt.Sprintf("ID: %s", peerIDShort))
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Задержка (latency)
	latencyLabel := widget.NewLabel(fmt.Sprintf("Ping: %d ms", peer.LatencyMs))
	latencyLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Индикатор статуса
	statusInd := canvas.NewCircle(color.RGBA{R: 0, G: 255, B: 0, A: 255})

	// Кнопка чата
	chatBtn := widget.NewButtonWithIcon("Чат", theme.MailComposeIcon(), func() {
		ui.openPeerChat(peer.PeerID, peer.Username)
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

	discovered := ui.p2pUI.GetDiscoveredPeers()

	if len(discovered) == 0 {
		emptyLabel := widget.NewLabel("Нет обнаруженных пиров")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.discoveredPeersList.Add(emptyLabel)
	} else {
		for peerID, lastSeen := range discovered {
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
	if len(peerIDShort) > 8 {
		peerIDShort = peerIDShort[:8]
	}
	peerIDLabel := widget.NewLabel(fmt.Sprintf("ID: %s", peerIDShort))
	peerIDLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Время последнего обнаружения
	lastSeenLabel := widget.NewLabel(fmt.Sprintf("Обнаружен: %s", lastSeen.Format("15:04:05")))
	lastSeenLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка подключения
	connectBtn := widget.NewButtonWithIcon("Подключиться", theme.FolderIcon(), func() {
		ui.connectToDiscoveredPeer(peerID)
	})

	content := container.NewBorder(
		nil, connectBtn,
		nil, nil,
		container.NewVBox(peerIDLabel, lastSeenLabel),
		widget.NewSeparator(),
	)

	return content
}

// loadProfiles загружает список профилей
func (ui *UI) loadProfiles() {
	if ui.profilesList == nil {
		return
	}

	ui.profilesList.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.profilesList.Add(emptyLabel)
		ui.profilesList.Refresh()
		return
	}

	profiles, _ := ui.p2pUI.GetProfiles()

	if len(profiles) == 0 {
		emptyLabel := widget.NewLabel("Нет профилей")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.profilesList.Add(emptyLabel)
	} else {
		for _, profile := range profiles {
			profileItem := ui.createProfileItem(profile)
			ui.profilesList.Add(profileItem)
		}
	}

	ui.profilesList.Refresh()
}

// createProfileItem создает элемент профиля
func (ui *UI) createProfileItem(profile *models.Profile) *fyne.Container {
	// Имя пользователя
	usernameLabel := widget.NewLabel(profile.Username)
	usernameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// PeerID (сокращённый)
	peerIDShort := profile.PeerID
	if len(peerIDShort) > 8 {
		peerIDShort = peerIDShort[:8]
	}
	peerIDLabel := widget.NewLabel(fmt.Sprintf("ID: %s", peerIDShort))
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Время кэширования
	cachedAtLabel := widget.NewLabel("")
	if profile.CachedAt != nil {
		cachedAtLabel.SetText(fmt.Sprintf("Обновлён: %s", profile.CachedAt.Format("15:04:05")))
		cachedAtLabel.TextStyle = fyne.TextStyle{Italic: true}
	}

	// Кнопка чата
	chatBtn := widget.NewButtonWithIcon("Чат", theme.MailComposeIcon(), func() {
		ui.openPeerChat(profile.PeerID, profile.Username)
	})

	content := container.NewBorder(
		nil, chatBtn,
		nil, nil,
		container.NewVBox(usernameLabel, peerIDLabel, cachedAtLabel),
		widget.NewSeparator(),
	)

	return content
}

// openPeerChat открывает чат с пиром
func (ui *UI) openPeerChat(peerID, username string) {
	if ui.p2pUIProvider != nil {
		ui.p2pUIProvider.OpenPeerChat(peerID, username)
	}
}

// addConnectedPeerToContacts добавляет подключённого пира в контакты
func (ui *UI) addConnectedPeerToContacts(peerID string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	// Получаем адрес пира
	addrs := ui.p2pUI.GetPeerAddresses(peerID)
	if len(addrs) == 0 {
		ui.showErrorDialog("Ошибка", "Адрес пира не найден")
		return
	}

	// Добавляем в контакты
	err := ui.p2pUI.AddContactByAddress(addrs[0], "")
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить в контакты: %v", err))
		return
	}

	ui.showInfoDialog("Успешно", "Пир добавлен в контакты")
}

// disconnectFromPeer отключается от пира
func (ui *UI) disconnectFromPeer(peerID string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	err := ui.p2pUI.DisconnectPeer(peerID)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось отключиться: %v", err))
		return
	}

	// Обновляем список через пару секунд
	time.AfterFunc(2*time.Second, func() {
		ui.loadConnectedPeers()
	})
}

// connectToDiscoveredPeer подключается к обнаруженному пиру
func (ui *UI) connectToDiscoveredPeer(peerID string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	// Получаем адрес пира из discovered
	// TODO: Получить адрес из peerstore
	ui.showInfoDialog("Информация", "Подключение к обнаруженному пиру требует реализации")
}
