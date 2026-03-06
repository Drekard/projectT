package chats

import (
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/chats/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// UI представляет интерфейс чатов
type UI struct {
	content        fyne.CanvasObject
	window         fyne.Window
	currentChatID  int
	contacts       []*models.Contact
	chatsList      *fyne.Container
	chatArea       *fyne.Container
	profileArea    *fyne.Container
	contactsPanel  *fyne.Container
	messageScroll  *container.Scroll
	messagesList   *fyne.Container
	messageEntry   *widget.Entry
	sendButton     *widget.Button
	contactsIcon   *widgets.ChatIcon
	favoritesIcon  *widgets.ChatIcon
	chatIcons      []*widgets.ChatIcon
	chatTitle      *widget.Label
	chatStatus     *widget.Label
	profileAvatar  *canvas.Circle
	profileName    *widget.Label
	profileStatus  *widget.Label
	profileAddress *widget.Label
	onContactClick func(contactID int)
	onSendMessage  func(text string)
}

// New создает и возвращает новый UI чатов
func New() *UI {
	ui := &UI{
		chatIcons: make([]*widgets.ChatIcon, 0),
		contacts:  make([]*models.Contact, 0),
	}
	ui.content = ui.createViewContent()
	return ui
}

// SetWindow устанавливает окно
func (ui *UI) SetWindow(window fyne.Window) {
	ui.window = window
}

// createViewContent создает основное представление UI чатов
func (ui *UI) createViewContent() fyne.CanvasObject {
	// Левая панель со списком чатов
	leftPanel := ui.createLeftPanel()

	// Центральная область с чатом
	ui.chatArea = ui.createChatArea()

	// Правая панель с профилем
	ui.profileArea = ui.createProfileArea()

	// Основная компоновка: левая панель | чат | профиль
	mainContent := container.NewBorder(
		nil, nil,
		leftPanel,
		ui.profileArea,
		ui.chatArea,
	)

	return mainContent
}

// Refresh обновляет UI
func (ui *UI) Refresh() {
	if ui.content != nil {
		ui.content.Refresh()
	}
}

// CreateView возвращает canvas object для UI чатов
func (ui *UI) CreateView() fyne.CanvasObject {
	return ui.content
}

// SetOnContactClick устанавливает обработчик клика по контакту
func (ui *UI) SetOnContactClick(handler func(contactID int)) {
	ui.onContactClick = handler
}

// SetOnSendMessage устанавливает обработчик отправки сообщения
func (ui *UI) SetOnSendMessage(handler func(text string)) {
	ui.onSendMessage = handler
}
