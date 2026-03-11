package chats

import (
	"time"

	"projectT/internal/services/p2p/network"
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/chats/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// UI представляет интерфейс чатов
type UI struct {
	content               fyne.CanvasObject
	window                fyne.Window
	p2pUI                 *network.UIP2P
	currentChatID         int
	contacts              []*models.Contact
	chatsList             *fyne.Container
	chatArea              *fyne.Container
	profileArea           *fyne.Container
	messageScroll         *container.Scroll
	messagesList          *fyne.Container
	chatTitle             *widget.Label
	chatStatus            *widget.Label
	profileAvatar         *canvas.Circle
	profileName           *widget.Label
	profileStatus         *widget.Label
	profileAddress        *widget.Label
	myAddressLabel        *widget.Label
	connectionStatusLabel *widget.Label
	peersCountLabel       *widget.Label
	natStatusLabel        *widget.Label
	portEntry             *widget.Entry
	contactsListInPanel   *fyne.Container
	connectedPeersList    *fyne.Container
	bootstrapList         *fyne.Container
	discoveredPeersList   *fyne.Container
	addressEntry          *widget.Entry
	usernameEntry         *widget.Entry
	bootstrapEntry        *widget.Entry
	stunServerEntry       *widget.Entry
	natPortMapCheck       *widget.Check
	relayCheck            *widget.Check
	autoRelayCheck        *widget.Check
	dhtCheck              *widget.Check
	mdnsCheck             *widget.Check
	stunCheck             *widget.Check
	helperModeCheck       *widget.Check
	onContactClick        func(contactID int)
	onSendMessage         func(text string)
}

// New создает и возвращает новый UI чатов
func New() *UI {
	ui := &UI{
		contacts: make([]*models.Contact, 0),
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

	// Обновляем статус подключения если P2P инициализирован и UI создан
	// НЕ запускаем обнаружение пиров автоматически - только по кнопке
	if ui.p2pUI != nil && ui.connectionStatusLabel != nil {
		ui.refreshConnectionStatus()
		// ui.loadConnectedPeers() // Не загружаем автоматически
		// ui.loadBootstrapPeers() // Не загружаем автоматически
		// ui.loadDiscoveredPeers() // Не загружаем автоматически
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

// SetP2PService устанавливает P2P сервис
func (ui *UI) SetP2PService(p2pUI *network.UIP2P) {
	ui.p2pUI = p2pUI
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
	// Инициализируем при необходимости
	if ui.messagesList == nil {
		ui.messagesList = container.NewVBox()
	}
	if ui.messageScroll == nil {
		ui.messageScroll = container.NewScroll(ui.messagesList)
	}

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
