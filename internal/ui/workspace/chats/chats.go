package chats

import (
	"log"

	"projectT/internal/services"
	"projectT/internal/services/p2p/network"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/chats/center"
	"projectT/internal/ui/workspace/chats/dialogs"
	"projectT/internal/ui/workspace/chats/left"
	"projectT/internal/ui/workspace/chats/p2p"
	"projectT/internal/ui/workspace/chats/right"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// UI представляет интерфейс чатов
type UI struct {
	content        fyne.CanvasObject
	window         fyne.Window
	p2pUI          *network.UIP2P
	currentChatID  int
	currentContact *models.Contact
	chatPanel      *center.ChatPanel

	// Компоненты панелей
	leftPanel   *left.Panel
	rightPanel  *right.Panel
	p2pPanel    *p2p.Panel
	chatAreaObj *fyne.Container

	// Менеджеры
	chatMenuManager *dialogs.ChatMenuManager

	// Каналы сообщений
	messageChannel <-chan *services.ChatMessageEvent
	onSendMessage  func(text string)
}

// New создает и возвращает новый UI чатов
func New() *UI {
	ui := &UI{}
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
	ui.leftPanel = left.New(ui)
	leftPanelContainer := ui.leftPanel.Container()

	// Центральная область с чатом (пустая по умолчанию)
	ui.chatAreaObj = ui.createChatArea()

	// Правая панель с профилем
	ui.rightPanel = right.New(ui)
	rightPanelContainer := ui.rightPanel.Container()

	// Основная компоновка: левая панель | чат | профиль
	mainContent := container.NewBorder(
		nil, nil,
		leftPanelContainer,
		rightPanelContainer,
		ui.chatAreaObj,
	)

	return mainContent
}

// Refresh обновляет UI
func (ui *UI) Refresh() {
	if ui.content != nil {
		ui.content.Refresh()
	}

	// Обновляем левую панель
	if ui.leftPanel != nil {
		ui.leftPanel.Refresh()
	}

	// Обновляем правую панель
	if ui.rightPanel != nil {
		ui.rightPanel.Refresh()
	}
}

// CreateView возвращает canvas object для UI чатов
func (ui *UI) CreateView() fyne.CanvasObject {
	return ui.content
}

// SetOnSendMessage устанавливает обработчик отправки сообщения
func (ui *UI) SetOnSendMessage(handler func(text string)) {
	ui.onSendMessage = handler
}

// SetP2PService устанавливает P2P сервис
func (ui *UI) SetP2PService(p2pUI *network.UIP2P) {
	ui.p2pUI = p2pUI

	// Передаем P2P сервис в p2p панель
	if ui.p2pPanel != nil {
		ui.p2pPanel.SetP2PService(p2pUI)
	}
}

// selectChat выбирает чат с пиром
func (ui *UI) selectChat(contact *models.Contact) {
	ui.currentContact = contact
	ui.currentChatID = contact.ID

	// Создаём панель чата
	chatPanel := ui.createChatPanel(contact)
	ui.chatAreaObj.Objects = []fyne.CanvasObject{chatPanel}
	ui.chatAreaObj.Refresh()

	// Обновляем профиль (для локального чата показываем свой профиль)
	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(contact)
	}
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
	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(tempContact)
	}

	// Обновляем UI
	if ui.chatAreaObj != nil {
		ui.chatAreaObj.Refresh()
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
	}
}

// createChatArea создает центральную область чата
func (ui *UI) createChatArea() *fyne.Container {
	// По умолчанию показываем пустую панель
	emptyPanel := ui.createEmptyPanel()
	ui.chatAreaObj = container.NewStack(emptyPanel)
	return ui.chatAreaObj
}

// createEmptyPanel создает пустую панель с подсказкой
func (ui *UI) createEmptyPanel() *fyne.Container {
	// Иконка
	icon := fyne.CurrentApp().Icon()

	// Заголовок
	title := widget.NewLabel("Чаты")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Подзаголовок
	subtitle := widget.NewLabel("Выберите чат в левой панели")
	subtitle.Alignment = fyne.TextAlignCenter
	subtitle.TextStyle = fyne.TextStyle{Italic: true}

	content := container.NewVBox(
		widget.NewIcon(icon),
		title,
		subtitle,
	)

	centered := container.NewCenter(content)

	return centered
}

// createChatPanel создает панель чата с сообщениями и полем ввода
func (ui *UI) createChatPanel(contact *models.Contact) fyne.CanvasObject {
	// Получаем локальный PeerID
	localPeerID := ""
	if ui.p2pUI != nil {
		status := ui.p2pUI.GetStatus()
		if status != nil {
			localPeerID = status.PeerID
		}
	} else {
		// Если P2P не инициализирован, используем PeerID из локального профиля
		localProfile, err := queries.GetLocalProfile()
		if err == nil {
			localPeerID = localProfile.PeerID
		}
	}

	// Создаём панель чата с использованием нового компонента
	ui.chatPanel = center.NewChatPanel(
		contact,
		ui.sendMessage,
		ui.closeChat,
		localPeerID,
	)

	return ui.chatPanel.Container()
}

// sendMessage отправляет сообщение
func (ui *UI) sendMessage() {
	if ui.chatPanel == nil {
		return
	}

	text := ui.chatPanel.MessageInput().Text()
	if text == "" {
		return
	}

	// Очищаем поле ввода
	ui.chatPanel.MessageInput().Clear()

	// Вызываем внешний обработчик если установлен
	if ui.onSendMessage != nil {
		ui.onSendMessage(text)
	}
}

// loadMessagesForContact загружает сообщения для контакта
func (ui *UI) loadMessagesForContact(contactID int) {
	if ui.chatPanel == nil {
		return
	}

	// Очищаем текущие сообщения
	ui.chatPanel.Clear()

	// Обычный чат с контактом
	messages, err := queries.GetMessagesForContact(contactID, 100, 0)
	if err != nil {
		log.Printf("Ошибка загрузки сообщений: %v", err)
		return
	}

	// Получаем наш локальный PeerID для определения направления
	localPeerID := ""
	if ui.p2pUI != nil {
		status := ui.p2pUI.GetStatus()
		if status != nil {
			localPeerID = status.PeerID
		}
	}

	// Загружаем сообщения
	ui.chatPanel.LoadMessages(messages, localPeerID)
}

// closeChat закрывает текущий чат
func (ui *UI) closeChat() {
	ui.currentContact = nil
	ui.currentChatID = 0
	ui.chatPanel = nil

	// Показываем пустую панель
	emptyPanel := ui.createEmptyPanel()
	ui.chatAreaObj.Objects = []fyne.CanvasObject{emptyPanel}
	ui.chatAreaObj.Refresh()
}

// ShowChatMenu показывает меню действий для чата (реализация для left.UIProvider)
func (ui *UI) ShowChatMenu(chatID int, peerID string, username string, cont fyne.CanvasObject) {
	// Создаём менеджер меню если не существует
	if ui.chatMenuManager == nil {
		ui.chatMenuManager = dialogs.NewChatMenuManager(ui, ui.OnChatDeleted)
	}
	ui.chatMenuManager.ShowChatMenu(chatID, peerID, username, cont)
}

// OnChatDeleted обработчик удаления чата (реализация для left.UIProvider)
func (ui *UI) OnChatDeleted(chatID int, peerID string) {
	// Обновляем левую панель
	if ui.leftPanel != nil {
		ui.leftPanel.Refresh()
	}

	// Если текущий чат был удалён, закрываем его
	if ui.currentChatID == chatID {
		ui.currentChatID = 0
		ui.currentContact = nil
		ui.chatAreaObj.Objects = []fyne.CanvasObject{}
		ui.chatAreaObj.Refresh()
	}

	log.Printf("Чат %s (%d) удалён", peerID, chatID)
}

// OpenLocalChat открывает локальный чат с самим собой (реализация для left.UIProvider)
func (ui *UI) OpenLocalChat() {
	// Загружаем локальный профиль
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		log.Printf("Ошибка загрузки локального профиля: %v", err)
		return
	}

	// Получаем или создаём локальный чат (с contact_id = NULL)
	localChat, err := queries.GetOrCreateLocalChat(localProfile.PeerID)
	if err != nil {
		log.Printf("Ошибка получения локального чата: %v", err)
		return
	}

	// Создаём специальный контакт для локального чата (виртуальный, не из БД)
	localContact := models.NewLocalContact(
		localProfile.Username,
		localProfile.Title,
		localProfile.AvatarPath,
	)

	// Выбираем чат
	ui.selectChat(localContact)

	// Загружаем сообщения для локального чата через chat_id
	ui.loadMessagesForChat(localChat.ID)

	// Обновляем правую панель с профилем
	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(localContact)
	}

	// Обновляем UI
	if ui.chatAreaObj != nil {
		ui.chatAreaObj.Refresh()
	}
}

// ShowContactsPanel показывает панель управления P2P (реализация для left.UIProvider)
func (ui *UI) ShowContactsPanel() {
	// Создаём P2P панель если не существует
	if ui.p2pPanel == nil {
		ui.p2pPanel = p2p.New(ui)
		ui.p2pPanel.SetP2PService(ui.p2pUI)
	}

	controlPanel := ui.p2pPanel.Container()
	ui.chatAreaObj.Objects = []fyne.CanvasObject{controlPanel}
	ui.chatAreaObj.Refresh()
}

// loadMessagesForChat загружает сообщения для чата по ID
func (ui *UI) loadMessagesForChat(chatID int) {
	if ui.chatPanel == nil {
		return
	}

	// Очищаем текущие сообщения
	ui.chatPanel.Clear()

	// Загружаем сообщения чата
	messages, err := queries.GetMessagesForChat(chatID, 100, 0)
	if err != nil {
		log.Printf("Ошибка загрузки сообщений чата: %v", err)
		return
	}

	// Получаем наш локальный PeerID для определения направления
	localPeerID := ""
	if ui.p2pUI != nil {
		status := ui.p2pUI.GetStatus()
		if status != nil {
			localPeerID = status.PeerID
		}
	} else {
		// Если P2P не инициализирован, используем PeerID из локального профиля
		localProfile, err := queries.GetLocalProfile()
		if err == nil {
			localPeerID = localProfile.PeerID
		}
	}

	// Загружаем сообщения (все сообщения в локальном чате - исходящие)
	ui.chatPanel.LoadMessages(messages, localPeerID)
}

// RefreshContactsList обновляет список чатов (публичный метод для вызова извне)
func (ui *UI) RefreshContactsList() {
	if ui.leftPanel != nil {
		ui.leftPanel.Refresh()
	}
}

// GetP2PService возвращает P2P сервис
func (ui *UI) GetP2PService() *network.UIP2P {
	return ui.p2pUI
}

// GetWindow возвращает окно
func (ui *UI) GetWindow() fyne.Window {
	return ui.window
}
