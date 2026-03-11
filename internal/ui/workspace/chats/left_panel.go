package chats

import (
	"image/color"

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
	//ui.chatsList = ui.createChatsList()

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
	//selfChatIcon := ui.createSelfChatIcon()

	// Вертикальная компоновка иконок
	icons := container.NewVBox(
		contactsIcon,
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

// showContactsPanel показывает панель управления P2P
func (ui *UI) showContactsPanel() {
	controlPanel := ui.createControlPanel()
	ui.chatArea.Objects = []fyne.CanvasObject{controlPanel}
	ui.chatArea.Refresh()
}
