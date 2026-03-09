package chats

import (
	"image/color"

	"projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/chats/widgets"

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
	ui.chatsList = ui.createChatsList()

	// Вертикальная компоновка
	content := container.NewVBox(header, ui.chatsList)

	// Оборачиваем в скролл
	scroll := container.NewVScroll(content)

	return container.NewStack(scroll)
}

// createLeftPanelHeader создает заголовок левой панели с иконками
func (ui *UI) createLeftPanelHeader() *fyne.Container {
	// Иконка контактов
	contactsIcon := ui.createContactsIcon()

	// Иконка чата с собой
	selfChatIcon := ui.createSelfChatIcon()

	// Вертикальная компоновка иконок
	icons := container.NewVBox(
		contactsIcon,
		selfChatIcon,
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

	// Создаем индикатор статуса
	statusInd := widgets.NewStatusIndicator(widgets.StatusOffline)

	// Собираем все элементы в стек
	stack := container.NewStack(avatar, statusInd)

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
	return container.NewStack(stack, btnContainer)
}

// createSelfChatIcon создает иконку для чата с самим собой
func (ui *UI) createSelfChatIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	avatar := canvas.NewRectangle(color.RGBA{R: 100, G: 100, B: 100, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем кнопку с иконкой поверх графики
	btn := widget.NewButtonWithIcon("", theme.MailAttachmentIcon(), func() {
		ui.showSelfChat()
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	// Оборачиваем в контейнер
	return container.NewStack(avatar, btnContainer)
}

// createChatsList создает список чатов с пирами
func (ui *UI) createChatsList() *fyne.Container {
	// Создаем контейнер для чатов
	chats := container.NewVBox()

	// Разделитель после заголовка
	separator := canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255})
	separator.SetMinSize(fyne.NewSize(0, 1))
	chats.Add(separator)

	// Добавляем тестовые чаты для демонстрации
	testContacts := []*models.Contact{
		{Username: "Алексей", PeerID: "alex_peer"},
		{Username: "Мария", PeerID: "maria_peer"},
		{Username: "Дмитрий", PeerID: "dmitry_peer"},
		{Username: "Елена", PeerID: "elena_peer"},
	}

	// Цвета для аватаров
	colors := []color.RGBA{
		{R: 144, G: 238, B: 144, A: 255},
		{R: 173, G: 216, B: 230, A: 255},
		{R: 255, G: 182, B: 193, A: 255},
		{R: 255, G: 218, B: 185, A: 255},
	}

	for i, contact := range testContacts {
		unreadCount := 0
		if i%2 == 0 {
			unreadCount = i + 1
		}

		// Тип 3: PeerChatIcon - аватарка для чата с пиром
		chatIcon := widgets.NewChatIcon(widgets.PeerChatIcon, colors[i%len(colors)], unreadCount, widgets.StatusOnline, func() {
			ui.selectChat(contact)
		})
		ui.chatIcons = append(ui.chatIcons, chatIcon)
		chats.Add(chatIcon)
	}

	return chats
}

// showContactsPanel показывает панель контактов вместо чата
func (ui *UI) showContactsPanel() {
	ui.contactsPanel = ui.createContactsPanel()
	ui.chatArea.Objects = []fyne.CanvasObject{ui.contactsPanel}
	ui.chatArea.Refresh()
}

// showSelfChat показывает пустой чат с самим собой
func (ui *UI) showSelfChat() {
	selfChat := ui.createSelfChat()
	ui.chatArea.Objects = []fyne.CanvasObject{selfChat}
	ui.chatArea.Refresh()
}
