package chats

import (
	"image/color"
	"log"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createLeftPanel создает левую панель со списком чатов
func (ui *UI) createLeftPanel() *fyne.Container {
	// Заголовок с иконками
	header := ui.createLeftPanelHeader()

	// Список чатов
	ui.chatsList = container.NewVBox()

	// Загружаем контакты из БД
	ui.loadContactsToChatsList()

	// Вертикальная компоновка
	content := container.NewVBox(header, ui.chatsList)

	// Оборачиваем в скролл
	scroll := container.NewVScroll(content)

	return container.NewStack(scroll)
}

// loadContactsToChatsList загружает контакты из БД в список чатов
func (ui *UI) loadContactsToChatsList() {
	if ui.chatsList == nil {
		return
	}

	ui.chatsList.Objects = nil

	contacts, err := queries.GetAllContacts()
	if err != nil {
		log.Printf("Ошибка загрузки контактов: %v", err)
		// Показываем сообщение об ошибке
		emptyLabel := widget.NewLabel("Ошибка загрузки контактов")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.chatsList.Add(emptyLabel)
		return
	}

	if len(contacts) == 0 {
		emptyLabel := widget.NewLabel("Нет контактов")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.chatsList.Add(emptyLabel)
	} else {
		for _, contact := range contacts {
			peerItem := ui.createPeerItem(contact)
			ui.chatsList.Add(peerItem)
		}
	}

	ui.chatsList.Refresh()
}

// createPeerItem создает элемент пира в списке
func (ui *UI) createPeerItem(contact *models.Contact) *fyne.Container {
	// Индикатор статуса (зелёный для онлайн, серый для оффлайн)
	statusColor := color.RGBA{R: 158, G: 158, B: 158, A: 255}
	statusText := "оффлайн"
	if contact.Status == "online" || contact.Status == "connected" {
		statusColor = color.RGBA{R: 76, G: 175, B: 80, A: 255}
		statusText = "онлайн"
	}

	statusInd := canvas.NewCircle(statusColor)

	// Имя контакта
	nameLabel := widget.NewLabel(contact.Username)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Статус
	statusLabel := widget.NewLabel(statusText)
	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Иконка пира (кнопка)
	peerBtn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		ui.openPeerChat(contact)
	})
	peerBtn.Importance = widget.LowImportance

	// Компонуем имя и статус
	infoContainer := container.NewVBox(nameLabel, statusLabel)

	// Основная компоновка: статус + информация + кнопка
	content := container.NewBorder(
		nil, nil,
		container.NewHBox(statusInd, infoContainer),
		peerBtn,
		widget.NewSeparator(),
	)

	return content
}

// openPeerChat открывает чат с пиром
func (ui *UI) openPeerChat(contact *models.Contact) {
	if ui.window == nil {
		log.Printf("Окно не инициализировано")
		return
	}

	// Выбираем чат
	ui.selectChat(contact)

	// Загружаем сообщения для контакта
	ui.loadMessagesForContact(contact.ID)

	// Обновляем UI
	if ui.chatArea != nil {
		ui.chatArea.Refresh()
	}
}

// createLeftPanelHeader создает заголовок левой панели с иконками
func (ui *UI) createLeftPanelHeader() *fyne.Container {
	// Иконка контактов
	contactsIcon := ui.createContactsIcon()

	// Иконка чата с собой
	faworiteIcon := ui.createFaworiteIcon()

	// Вертикальная компоновка иконок
	icons := container.NewVBox(
		contactsIcon,
		faworiteIcon,
	)

	return container.NewPadded(icons)
}

// createContactsIcon создает иконку для панели контактов
func (ui *UI) createContactsIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	avatar := canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем кнопку с иконкой поверх графики
	btn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		ui.showContactsPanel()
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	// Оборачиваем в контейнер
	return container.NewStack(avatar, btnContainer)
}

// createContactsIcon создает иконку для панели контактов
func (ui *UI) createFaworiteIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	avatar := canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем кнопку с иконкой поверх графики
	btn := widget.NewButtonWithIcon("", theme.MailAttachmentIcon(), func() {
		ui.showContactsPanel()
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	// Оборачиваем в контейнер
	return container.NewStack(avatar, btnContainer)
}

// showContactsPanel показывает панель управления P2P
func (ui *UI) showContactsPanel() {
	controlPanel := ui.createControlPanel()
	ui.chatArea.Objects = []fyne.CanvasObject{controlPanel}
	ui.chatArea.Refresh()
}
