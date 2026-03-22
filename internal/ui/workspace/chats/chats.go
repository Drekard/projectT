package chats

import (
	"log"
	"projectT/internal/services"
	"projectT/internal/services/p2p/network"
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/chats/center"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// UI представляет интерфейс чатов
type UI struct {
	content                  fyne.CanvasObject
	window                   fyne.Window
	p2pUI                    *network.UIP2P
	currentChatID            int
	currentContact           *models.Contact
	contacts                 []*models.Contact
	chatsList                *fyne.Container
	chatArea                 *fyne.Container
	chatPanel                *center.ChatPanel
	profileArea              *fyne.Container
	profileAvatar            *canvas.Image
	profileName              *widget.Label
	profileStatus            *widget.Label
	characteristicsContainer *fyne.Container
	demoElementsContainer    *fyne.Container
	myAddressLabel           *widget.Label
	connectionStatusLabel    *widget.Label
	peersCountLabel          *widget.Label
	natStatusLabel           *widget.Label
	portEntry                *widget.Entry
	contactsListInPanel      *fyne.Container
	connectedPeersList       *fyne.Container
	bootstrapList            *fyne.Container
	discoveredPeersList      *fyne.Container
	profilesListInPanel      *fyne.Container
	addressEntry             *widget.Entry
	usernameEntry            *widget.Entry
	bootstrapEntry           *widget.Entry
	stunServerEntry          *widget.Entry
	natPortMapCheck          *widget.Check
	relayCheck               *widget.Check
	autoRelayCheck           *widget.Check
	dhtCheck                 *widget.Check
	mdnsCheck                *widget.Check
	stunCheck                *widget.Check
	helperModeCheck          *widget.Check
	onContactClick           func(contactID int)
	onSendMessage            func(text string)
	messageChannel           <-chan *services.ChatMessageEvent
	ChatsUI                  *UI
	chatMenuManager          *ChatMenuManager // Менеджер меню для чатов
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
	ui.currentContact = contact
	ui.currentChatID = contact.ID

	// Создаём панель чата
	chatPanel := ui.createChatPanel(contact)
	ui.chatArea.Objects = []fyne.CanvasObject{chatPanel}
	ui.chatArea.Refresh()

	// Обновляем профиль (для локального чата показываем свой профиль)
	ui.updateProfile(contact)
}

// OpenPeerChat открывает чат с пиром (публичный метод для вызова из p2p_panel)
func (ui *UI) OpenPeerChat(peerID, username string) {
	// Создаём временный контакт для чата
	tempContact := &models.Contact{
		PeerID:   peerID,
		Username: username,
		ID:       0, // ID = 0 означает, что контакт не в БД
	}

	// Выбираем чат
	ui.selectChat(tempContact)

	// Загружаем сообщения (будет пусто для нового пира)
	// Для пира с ID = 0 используем заглушку, сообщения загрузятся после получения chat_id
	ui.loadMessagesForContact(0)

	// Запрашиваем профиль у пира если P2P инициализирован
	if ui.p2pUI != nil {
		go func() {
			err := ui.p2pUI.RequestProfile(peerID)
			if err != nil {
				log.Printf("Не удалось запросить профиль у пира %s: %v", peerID, err)
			}
		}()
	}

	// Обновляем правую панель с профилем
	ui.updateProfile(tempContact)

	// Обновляем UI
	if ui.chatArea != nil {
		ui.chatArea.Refresh()
	}
}

// SubscribeToMessages подписывается на события сообщений и запускает обработчик
func (ui *UI) SubscribeToMessages() {
	// Получаем глобальный экземпляр сервиса чатов
	chatSvc := services.GetChatService()
	if chatSvc != nil {
		ui.messageChannel = chatSvc.Subscribe()
		go ui.handleMessageEvents()
	}
}

// handleMessageEvents обрабатывает события новых сообщений
func (ui *UI) handleMessageEvents() {
	for event := range ui.messageChannel {
		// Проверяем, открыт ли сейчас чат с этим контактом
		if ui.currentContact != nil && ui.currentContact.ID == event.ContactID {
			// Добавляем сообщение в UI
			if ui.chatPanel != nil {
				ui.chatPanel.AddMessage(event.Message, event.IsOutgoing)
			}
		}
		// Обновляем список чатов (можно добавить подсветку нового сообщения)
	}
}
