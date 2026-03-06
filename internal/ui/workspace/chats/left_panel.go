package chats

import (
	"image/color"

	"projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/chats/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// createLeftPanel создает левую панель со списком чатов
func (ui *UI) createLeftPanel() *fyne.Container {
	// Заголовок с кнопкой контактов
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
	// Иконка контактов (серая)
	contactsColor := color.RGBA{R: 158, G: 158, B: 158, A: 255}
	ui.contactsIcon = widgets.NewChatIcon(contactsColor, 0, widgets.StatusOffline, func() {
		ui.showContactsPanel()
	})

	// Иконка избранного (зеленая)
	favoritesColor := color.RGBA{R: 76, G: 175, B: 80, A: 255}
	ui.favoritesIcon = widgets.NewChatIcon(favoritesColor, 0, widgets.StatusOnline, func() {
		ui.showFavorites()
	})

	// Вертикальная компоновка иконок
	icons := container.NewVBox(
		ui.contactsIcon,
		ui.favoritesIcon,
	)

	return container.NewPadded(icons)
}

// createChatsList создает список чатов
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

		chatIcon := widgets.NewChatIcon(colors[i%len(colors)], unreadCount, widgets.StatusOnline, func() {
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

// showFavorites показывает избранное
func (ui *UI) showFavorites() {
	// TODO: реализовать отображение избранного
	ui.showMessage("Избранное", "Здесь будут сохраненные сообщения")
}
