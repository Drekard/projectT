package chats

import (
	"image/color"

	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// createProfileArea создает правую панель с профилем
func (ui *UI) createProfileArea() *fyne.Container {
	// Заголовок
	header := widget.NewLabel("Профиль")
	header.TextStyle = fyne.TextStyle{Bold: true}

	// Аватар
	ui.profileAvatar = canvas.NewCircle(color.RGBA{R: 100, G: 100, B: 100, A: 255})

	// Имя
	ui.profileName = widget.NewLabel("Имя собеседника")
	ui.profileName.Alignment = fyne.TextAlignCenter

	// Статус
	ui.profileStatus = widget.NewLabel("онлайн")
	ui.profileStatus.Alignment = fyne.TextAlignCenter

	// Адрес
	ui.profileAddress = widget.NewLabel("address@example.com")
	ui.profileAddress.Alignment = fyne.TextAlignCenter

	// Информация
	info := container.NewVBox(
		ui.profileAvatar,
		layout.NewSpacer(),
		ui.profileName,
		ui.profileStatus,
		ui.profileAddress,
		layout.NewSpacer(),
	)

	// Разделитель
	separator := canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255})

	// Настройки
	settingsLabel := widget.NewLabel("Настройки чата")

	muteButton := widget.NewButton("🔕 Уведомления", func() {
		// TODO: реализовать отключение уведомлений
	})
	muteButton.Importance = widget.LowImportance

	clearButton := widget.NewButton("🗑 Очистить историю", func() {
		// TODO: реализовать очистку истории
	})
	clearButton.Importance = widget.LowImportance

	settings := container.NewVBox(
		separator,
		settingsLabel,
		muteButton,
		clearButton,
	)

	content := container.NewVBox(
		container.NewPadded(info),
		container.NewPadded(settings),
	)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 35, G: 35, B: 35, A: 255})

	return container.NewStack(bg, content)
}

// updateProfile обновляет профиль собеседника
func (ui *UI) updateProfile(contact *models.Contact) {
	if ui.profileAvatar != nil {
		ui.profileAvatar.FillColor = ui.getAvatarColorForContact(contact)
		ui.profileAvatar.Refresh()
	}
	if ui.profileName != nil {
		ui.profileName.SetText(contact.Username)
		ui.profileName.Refresh()
	}
	if ui.profileStatus != nil {
		ui.profileStatus.SetText("онлайн")
		ui.profileStatus.Refresh()
	}
	if ui.profileAddress != nil {
		ui.profileAddress.SetText(contact.PeerID)
		ui.profileAddress.Refresh()
	}
}

func (ui *UI) getAvatarColorForContact(contact *models.Contact) color.Color {
	if contact == nil || contact.Username == "" {
		return color.RGBA{R: 100, G: 100, B: 100, A: 255}
	}
	hash := 0
	for _, c := range contact.Username {
		hash = int(c) + (hash * 31)
	}
	colors := []color.RGBA{
		{R: 144, G: 238, B: 144, A: 255},
		{R: 173, G: 216, B: 230, A: 255},
		{R: 255, G: 182, B: 193, A: 255},
		{R: 255, G: 218, B: 185, A: 255},
		{R: 221, G: 160, B: 221, A: 255},
		{R: 175, G: 238, B: 238, A: 255},
		{R: 255, G: 255, B: 153, A: 255},
		{R: 255, G: 224, B: 189, A: 255},
	}
	return colors[hash%len(colors)]
}

// createContactsPanel создает панель контактов
func (ui *UI) createContactsPanel() *fyne.Container {
	// Заголовок
	header := widget.NewLabel("Контакты")
	header.TextStyle = fyne.TextStyle{Bold: true}

	// Информация о подключении
	connectionInfo := ui.createConnectionInfo()

	// Управление адресом
	addressControl := ui.createAddressControl()

	// Кнопка добавления контактов
	addContactBtn := widget.NewButton("+ Добавить контакт", func() {
		ui.addContact()
	})
	addContactBtn.Importance = widget.HighImportance

	// Настройки P2P
	p2pSettings := ui.createP2PSettings()

	// Список контактов
	contactsList := ui.createContactsList()

	// Вертикальная компоновка
	content := container.NewVBox(
		header,
		connectionInfo,
		addressControl,
		addContactBtn,
		p2pSettings,
		contactsList,
	)

	// Скролл
	scroll := container.NewScroll(content)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 35, G: 35, B: 35, A: 255})

	return container.NewStack(bg, scroll)
}

// createConnectionInfo создает информацию о подключении
func (ui *UI) createConnectionInfo() *fyne.Container {
	title := widget.NewLabel("Состояние подключения")

	status := widget.NewLabel("● Подключено")

	peers := widget.NewLabel("Пиров: 5")

	return container.NewVBox(title, status, peers)
}

// createAddressControl создает управление адресом
func (ui *UI) createAddressControl() *fyne.Container {
	title := widget.NewLabel("Ваш адрес")

	addressEntry := widget.NewEntry()
	addressEntry.SetText("user@projectT.local")
	addressEntry.Disable()

	copyButton := widget.NewButton("Копировать", func() {
		// TODO: копировать адрес в буфер обмена
	})

	return container.NewVBox(title, container.NewHBox(addressEntry, copyButton))
}

// createP2PSettings создает настройки P2P
func (ui *UI) createP2PSettings() *fyne.Container {
	title := widget.NewLabel("Настройки P2P")

	// Чекбокс включения P2P
	p2pEnabled := widget.NewCheck("Включить P2P", func(checked bool) {
		// TODO: включить/выключить P2P
	})
	p2pEnabled.Checked = true

	// Чекбокс автоподключения
	autoConnect := widget.NewCheck("Автоподключение", func(checked bool) {
		// TODO: включить/выключить автоподключение
	})
	autoConnect.Checked = true

	return container.NewVBox(title, p2pEnabled, autoConnect)
}

// createContactsList создает список контактов
func (ui *UI) createContactsList() *fyne.Container {
	title := widget.NewLabel("Все контакты")

	contacts := container.NewVBox()

	// Тестовые контакты
	testContacts := []*models.Contact{
		{Username: "Алексей", PeerID: "alex_peer"},
		{Username: "Мария", PeerID: "maria_peer"},
		{Username: "Дмитрий", PeerID: "dmitry_peer"},
		{Username: "Елена", PeerID: "elena_peer"},
		{Username: "Иван", PeerID: "ivan_peer"},
	}

	for _, contact := range testContacts {
		contactItem := ui.createContactListItem(contact)
		contacts.Add(contactItem)
	}

	return container.NewVBox(title, contacts)
}

// createContactListItem создает элемент списка контактов
func (ui *UI) createContactListItem(contact *models.Contact) *fyne.Container {
	name := widget.NewLabel(contact.Username)

	address := widget.NewLabel(contact.PeerID)

	// Индикатор статуса
	status := canvas.NewCircle(color.RGBA{R: 76, G: 175, B: 80, A: 255})

	content := container.NewHBox(status, container.NewVBox(name, address))

	// Фон при наведении
	bg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 0})

	stack := container.NewStack(bg, container.NewPadded(content))

	return stack
}

// addContact добавляет новый контакт
func (ui *UI) addContact() {
	// TODO: реализовать добавление контакта
}
