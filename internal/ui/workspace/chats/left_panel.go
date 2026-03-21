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

	// Загружаем чаты из БД
	ui.loadChatsList()

	// Вертикальная компоновка
	content := container.NewVBox(header, ui.chatsList)

	// Оборачиваем в скролл
	scroll := container.NewVScroll(content)

	return container.NewStack(scroll)
}

// loadChatsList загружает чаты с последними сообщениями в список чатов
func (ui *UI) loadChatsList() {
	if ui.chatsList == nil {
		return
	}

	ui.chatsList.Objects = nil

	chats, err := queries.GetChatsWithLastMessages()
	if err != nil {
		log.Printf("Ошибка загрузки чатов: %v", err)
		// Показываем сообщение об ошибке
		emptyLabel := widget.NewLabel("Ошибка загрузки чатов")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.chatsList.Add(emptyLabel)
		ui.chatsList.Refresh()
		return
	}

	if len(chats) == 0 {
		emptyLabel := widget.NewLabel("Нет чатов")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.chatsList.Add(emptyLabel)
	} else {
		for _, chat := range chats {
			chatItem := ui.createChatItem(chat)
			ui.chatsList.Add(chatItem)
		}
	}

	ui.chatsList.Refresh()
}

// RefreshContactsList обновляет список чатов (публичный метод для вызова извне)
func (ui *UI) RefreshContactsList() {
	ui.loadChatsList()
}

// createChatItem создает элемент чата с аватаром 50x50
func (ui *UI) createChatItem(chat *models.ChatWithLastMessage) *fyne.Container {
	// Аватар 50x50
	avatarContainer := ui.createChatAvatarIcon(chat)

	// Основная компоновка: только аватар
	content := container.NewHBox(
		avatarContainer,
		widget.NewSeparator(),
	)

	return content
}

// createChatAvatarIcon создает иконку чата с аватаром 50x50
func (ui *UI) createChatAvatarIcon(chat *models.ChatWithLastMessage) *fyne.Container {
	if chat.AvatarPath != "" {
		// Пробуем загрузить аватар из файла
		avatarRes, err := fyne.LoadResourceFromPath(chat.AvatarPath)
		if err == nil && avatarRes != nil {
			// Создаём кнопку с аватаром
			btn := widget.NewButtonWithIcon("", avatarRes, func() {
				ui.openChatByID(chat.ID)
			})
			btn.Importance = widget.LowImportance
			return container.NewStack(btn)
		}
	}

	// Аватара нет - используем иконку по умолчанию
	btn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		ui.openChatByID(chat.ID)
	})
	btn.Importance = widget.LowImportance
	return container.NewStack(btn)
}

// openChatByID открывает чат по ID контакта
func (ui *UI) openChatByID(contactID int) {
	// Получаем контакт по ID
	contact, err := queries.GetContact(contactID)
	if err != nil {
		log.Printf("Ошибка получения контакта %d: %v", contactID, err)
		return
	}

	// Открываем чат
	ui.openPeerChat(contact)
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

	// Запрашиваем профиль у пира если P2P инициализирован
	if ui.p2pUI != nil {
		go func() {
			err := ui.p2pUI.RequestProfile(contact.PeerID)
			if err != nil {
				log.Printf("Не удалось запросить профиль у пира %s: %v", contact.PeerID, err)
			}
		}()
	}

	// Обновляем правую панель с профилем
	ui.updateProfile(contact)

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

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		ui.RefreshContactsList()
	})

	// Вертикальная компоновка иконок
	icons := container.NewVBox(
		contactsIcon,
		faworiteIcon,
	)

	return container.NewBorder(nil, refreshBtn, nil, nil,
		container.NewPadded(icons),
	)
}

// createContactsIcon создает иконку для панели контактов
func (ui *UI) createContactsIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	avatar := canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(100, 100))

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

// createFaworiteIcon создает иконку для чата Избранного (локальный чат с самим собой)
func (ui *UI) createFaworiteIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	avatar := canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем кнопку с иконкой поверх графики
	btn := widget.NewButtonWithIcon("", theme.ContentRedoIcon(), func() {
		ui.openLocalChat()
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	// Оборачиваем в контейнер
	return container.NewStack(avatar, btnContainer)
}

// openLocalChat открывает локальный чат с самим собой
func (ui *UI) openLocalChat() {
	if ui.window == nil {
		log.Printf("Окно не инициализировано")
		return
	}

	// Загружаем локальный профиль для создания контакта
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		log.Printf("Ошибка загрузки локального профиля: %v", err)
		return
	}

	// Создаём специальный контакт для локального чата
	localContact := models.NewLocalContact(
		localProfile.Username,
		localProfile.Title,
		localProfile.AvatarPath,
	)

	// Выбираем чат
	ui.selectChat(localContact)

	// Загружаем сообщения для локального чата
	ui.loadMessagesForContact(0) // ID = 0 для локального чата

	// Обновляем правую панель с профилем
	ui.updateProfile(localContact)

	// Обновляем UI
	if ui.chatArea != nil {
		ui.chatArea.Refresh()
	}
}

// showContactsPanel показывает панель управления P2P
func (ui *UI) showContactsPanel() {
	controlPanel := ui.createControlPanel()
	ui.chatArea.Objects = []fyne.CanvasObject{controlPanel}
	ui.chatArea.Refresh()
}
