package chats

import (
	"image/color"
	"time"

	"projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/chats/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createChatArea создает центральную область чата
func (ui *UI) createChatArea() *fyne.Container {
	// По умолчанию показываем панель управления
	controlPanel := ui.createControlPanel()
	ui.chatArea = container.NewStack(controlPanel)
	return ui.chatArea
}

// createControlPanel создает панель управления (тип 1)
func (ui *UI) createControlPanel() *fyne.Container {
	// Заголовок
	title := widget.NewLabel("Панель управления")
	title.TextStyle = fyne.TextStyle{Bold: true}

	// === Управление адресом ===
	addressSection := ui.createAddressSection()

	// === Информация о подключении ===
	connectionSection := ui.createConnectionSection()

	// === Настройки P2P ===
	p2pSection := ui.createP2PSettingsSection()

	// === Список контактов ===
	contactsSection := ui.createContactsListSection()

	// Собираем все секции
	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		addressSection,
		widget.NewSeparator(),
		connectionSection,
		widget.NewSeparator(),
		p2pSection,
		widget.NewSeparator(),
		contactsSection,
	)

	scroll := container.NewScroll(content)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 45, G: 45, B: 45, A: 255})

	return container.NewStack(bg, scroll)
}

// createAddressSection создает секцию управления адресом
func (ui *UI) createAddressSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Управление адресом")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	ui.myAddressLabel = widget.NewLabel("Адрес: загрузка...")

	copyButton := widget.NewButtonWithIcon("Копировать", theme.ContentCopyIcon(), func() {
		// TODO: копировать адрес в буфер обмена
	})

	addressRow := container.NewHBox(ui.myAddressLabel, copyButton)

	return container.NewVBox(sectionTitle, addressRow)
}

// createConnectionSection создает секцию информации о подключении
func (ui *UI) createConnectionSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Состояние подключения")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	ui.connectionStatusLabel = widget.NewLabel("Статус: отключено")
	ui.peersCountLabel = widget.NewLabel("Подключенные пиры: 0")

	return container.NewVBox(sectionTitle, ui.connectionStatusLabel, ui.peersCountLabel)
}

// createP2PSettingsSection создает секцию настроек P2P
func (ui *UI) createP2PSettingsSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Настройки P2P")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Порт прослушивания
	portLabel := widget.NewLabel("Порт прослушивания:")
	ui.portEntry = widget.NewEntry()
	ui.portEntry.SetText("8080")

	portRow := container.NewHBox(portLabel, ui.portEntry)

	// Чекбокс автозапуска
	ui.autoStartCheck = widget.NewCheck("Автозапуск при старте", nil)

	// Кнопка сохранения настроек
	saveButton := widget.NewButton("Сохранить настройки", func() {
		// TODO: сохранить настройки P2P
	})

	return container.NewVBox(sectionTitle, portRow, ui.autoStartCheck, saveButton)
}

// createContactsListSection создает секцию списка контактов
func (ui *UI) createContactsListSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Контакты")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка добавления контакта
	addContactButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		ui.showAddContactDialog()
	})

	headerRow := container.NewHBox(sectionTitle, addContactButton)

	// Список контактов
	ui.contactsListInPanel = container.NewVBox()

	// Загружаем тестовые контакты
	ui.loadContactsList()

	return container.NewVBox(headerRow, ui.contactsListInPanel)
}

// loadContactsList загружает список контактов в панель управления
func (ui *UI) loadContactsList() {
	if ui.contactsListInPanel == nil {
		return
	}

	ui.contactsListInPanel.Objects = nil

	// Тестовые контакты
	testContacts := []*models.Contact{
		{Username: "Алексей", PeerID: "alex_peer"},
		{Username: "Мария", PeerID: "maria_peer"},
		{Username: "Дмитрий", PeerID: "dmitry_peer"},
		{Username: "Елена", PeerID: "elena_peer"},
	}

	for _, contact := range testContacts {
		contactItem := widgets.NewContactItem(contact, 0, widgets.StatusOnline, func() {
			ui.selectChat(contact)
		})
		ui.contactsListInPanel.Add(contactItem)
	}

	ui.contactsListInPanel.Refresh()
}

// createSelfChat создает пустой чат с самим собой (тип 2)
func (ui *UI) createSelfChat() *fyne.Container {
	// Заголовок
	header := widget.NewLabel("Чат с самим собой")
	header.TextStyle = fyne.TextStyle{Bold: true}

	// Подзаголовок
	subtitle := widget.NewLabel("Здесь вы можете хранить заметки и важные сообщения")
	subtitle.TextStyle = fyne.TextStyle{Italic: true}

	headerContainer := container.NewVBox(header, subtitle)

	// Область сообщений (пустая)
	ui.messagesList = container.NewVBox()
	ui.messageScroll = container.NewScroll(ui.messagesList)

	// Пустое сообщение по умолчанию
	emptyLabel := widget.NewLabel("Нет сообщений. Отправьте первое сообщение себе!")
	emptyLabel.Alignment = fyne.TextAlignCenter
	emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
	ui.messagesList.Add(emptyLabel)

	// Поле ввода сообщения
	ui.messageEntry = widget.NewEntry()
	ui.messageEntry.SetPlaceHolder("Введите сообщение...")
	ui.messageEntry.MultiLine = false
	ui.messageEntry.OnSubmitted = func(text string) {
		ui.sendMessageToSelf()
	}

	// Кнопка отправки
	ui.sendButton = widget.NewButtonWithIcon("➤", theme.MailComposeIcon(), func() {
		ui.sendMessageToSelf()
	})
	ui.sendButton.Importance = widget.HighImportance

	// Панель ввода
	inputPanel := container.NewHBox(ui.messageEntry, ui.sendButton)

	// Вертикальная компоновка
	content := container.NewBorder(headerContainer, inputPanel, nil, nil, ui.messageScroll)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 40, G: 40, B: 40, A: 255})

	return container.NewStack(bg, content)
}

// selectChat выбирает чат с пиром
func (ui *UI) selectChat(contact *models.Contact) {
	ui.currentChatID = 0 // TODO: использовать реальный ID
	ui.updateChatHeader(contact.Username, contact.PeerID)
	ui.loadMessages(contact)
	ui.updateProfile(contact)
}

// updateChatHeader обновляет заголовок чата
func (ui *UI) updateChatHeader(name, address string) {
	if ui.chatTitle != nil {
		ui.chatTitle.SetText(name)
	}
	if ui.chatStatus != nil {
		ui.chatStatus.SetText("онлайн • " + address)
	}
	if ui.chatArea != nil {
		ui.chatArea.Refresh()
	}
}

// loadMessages загружает сообщения чата
func (ui *UI) loadMessages(contact *models.Contact) {
	ui.messagesList.Objects = nil

	// Тестовые сообщения
	messages := []struct {
		text     string
		time     time.Time
		position widgets.MessagePosition
	}{
		{"Привет! Как дела?", time.Now().Add(-2 * time.Hour), widgets.MessageLeft},
		{"Привет! Все хорошо, спасибо!", time.Now().Add(-2*time.Hour + 5*time.Minute), widgets.MessageRight},
		{"Работаешь над новым проектом?", time.Now().Add(-1 * time.Hour), widgets.MessageLeft},
		{"Да, делаю мессенджер на Go", time.Now().Add(-55 * time.Minute), widgets.MessageRight},
		{"Звучит круто! Покажешь потом?", time.Now().Add(-30 * time.Minute), widgets.MessageLeft},
		{"Конечно! Скоро будет готово", time.Now().Add(-25 * time.Minute), widgets.MessageRight},
	}

	for _, msg := range messages {
		bubble := widgets.NewMessageBubble(msg.text, msg.time, msg.position)
		ui.messagesList.Add(bubble)
	}

	ui.messagesList.Refresh()

	// Прокрутка вниз
	ui.messageScroll.ScrollToBottom()
}

// sendMessageToSelf отправляет сообщение себе
func (ui *UI) sendMessageToSelf() {
	text := ui.messageEntry.Text
	if text == "" {
		return
	}

	// Добавляем сообщение в чат
	bubble := widgets.NewMessageBubble(text, time.Now(), widgets.MessageRight)
	ui.messagesList.Add(bubble)
	ui.messagesList.Refresh()

	// Прокрутка вниз
	ui.messageScroll.ScrollToBottom()

	// Очищаем поле ввода
	ui.messageEntry.SetText("")
}

// showAddContactDialog показывает диалог добавления контакта
func (ui *UI) showAddContactDialog() {
	// TODO: реализовать диалог добавления контакта
}
