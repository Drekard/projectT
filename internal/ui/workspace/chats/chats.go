package chats

import (
	"log"

	"projectT/internal/services"
	network "projectT/internal/services/p2p/ui"
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
	"github.com/libp2p/go-libp2p/core/peer"
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
	messageChannel       <-chan *services.ChatMessageEvent
	onSendMessage        func(text string)
	onSendElementMessage func(item *models.Item)
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

// SetOnSendElementMessage устанавливает обработчик отправки элемента
func (ui *UI) SetOnSendElementMessage(handler func(item *models.Item)) {
	ui.onSendElementMessage = handler
}

// SetP2PService устанавливает P2P сервис
func (ui *UI) SetP2PService(p2pUI *network.UIP2P) {
	ui.p2pUI = p2pUI

	// Передаем P2P сервис в p2p панель
	if ui.p2pPanel != nil {
		ui.p2pPanel.SetP2PService(p2pUI)
	}

	// Устанавливаем обработчик отправки сообщений
	log.Printf("[Chat] 🔧 Установка обработчика отправки сообщений (SetOnSendMessage)")
	ui.SetOnSendMessage(func(text string) {
		log.Printf("[Chat] 📨 Обработчик отправки сообщения вызван: len=%d", len(text))
		log.Printf("[Chat] 🔍 ОТЛАДКА: ui.currentContact = %+v", ui.currentContact)
		if ui.currentContact != nil {
			log.Printf("[Chat] 🔍 ОТЛАДКА: currentContact.PeerID полный = %s", ui.currentContact.PeerID)
			log.Printf("[Chat] 🔍 ОТЛАДКА: currentContact.PeerID len = %d", len(ui.currentContact.PeerID))
		}

		if ui.p2pUI == nil {
			log.Printf("[Chat] ❌ Ошибка: P2P сервис не инициализирован")
			return
		}

		// Получаем PeerID пира
		if ui.currentContact == nil || ui.currentContact.PeerID == "" {
			log.Printf("[Chat] ❌ Ошибка: текущий контакт не выбран или не имеет PeerID")
			return
		}

		peerID, err := peer.Decode(ui.currentContact.PeerID)
		if err != nil {
			log.Printf("[Chat] ❌ Ошибка декодирования PeerID: %v", err)
			log.Printf("[Chat] 🔍 ОТЛАДКА: ui.currentContact.PeerID = %q", ui.currentContact.PeerID)
			return
		}

		log.Printf("[Chat] 📤 Отправка сообщения пиру %s: %q", peerID[:8], text)

		// Отправляем сообщение через P2P сервис
		if err := ui.p2pUI.SendMessage(peerID, text); err != nil {
			log.Printf("[Chat] ❌ Ошибка отправки сообщения: %v", err)
			return
		}

		log.Printf("[Chat] ✅ Сообщение отправлено пиру %s", peerID[:8])
	})

	// Устанавливаем обработчик отправки элементов
	log.Printf("[Chat] 🔧 Установка обработчика отправки элементов (SetOnSendElementMessage)")
	ui.SetOnSendElementMessage(func(item *models.Item) {
		log.Printf("[Chat] 📨 Обработчик отправки элемента вызван: element_uuid=%s", item.ElementUUID)

		if ui.p2pUI == nil {
			log.Printf("[Chat] ❌ Ошибка: P2P сервис не инициализирован")
			return
		}

		// Получаем PeerID пира
		if ui.currentContact == nil || ui.currentContact.PeerID == "" {
			log.Printf("[Chat] ❌ Ошибка: текущий контакт не выбран или не имеет PeerID")
			return
		}

		// Для локального чата не отправляем через P2P
		if ui.currentContact.IsLocalChat() {
			log.Printf("[Chat] ℹ️ Локальный чат, элемент не отправляется через P2P")
			return
		}

		peerID, err := peer.Decode(ui.currentContact.PeerID)
		if err != nil {
			log.Printf("[Chat] ❌ Ошибка декодирования PeerID: %v", err)
			return
		}

		log.Printf("[Chat] 📤 Отправка элемента пиру %s: element_uuid=%s", peerID[:8], item.ElementUUID)

		// Отправляем элемент через P2P сервис
		if err := ui.p2pUI.SendElementMessage(peerID, item); err != nil {
			log.Printf("[Chat] ❌ Ошибка отправки элемента: %v", err)
			return
		}

		log.Printf("[Chat] ✅ Элемент отправлен пиру %s", peerID[:8])
	})
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
	log.Printf("[Chat] 🆕 OpenPeerChat: создание чата с пиром %s (%s)", username, peerID[:8])
	log.Printf("[Chat] 🔍 ОТЛАДКА: OpenPeerChat peerID полный = %s", peerID)
	log.Printf("[Chat] 🔍 ОТЛАДКА: OpenPeerChat username = %s", username)

	// Получаем профиль пира из БД для корректного отображения
	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		log.Printf("[Chat] 📋 Профиль пира найден: username=%q, avatar_path=%q, content_char=%q, pinned_uuids=%s",
			profile.Username, profile.AvatarPath, profile.ContentChar, profile.PinnedUUIDs)
	} else {
		log.Printf("[Chat] ⚠️ Профиль пира не найден или ошибка: %v", err)
	}

	// Создаём временный контакт для чата
	// Используем данные из профиля если он найден, иначе используем параметры функции
	contactUsername := username
	contactAvatarPath := ""
	if profile != nil {
		contactUsername = profile.Username
		contactAvatarPath = profile.AvatarPath
		log.Printf("[Chat] 📋 Используем данные из профиля: username=%q, avatar_path=%q", contactUsername, contactAvatarPath)
	}

	tempContact := &models.Contact{
		PeerID:     peerID,
		Username:   contactUsername,
		AvatarPath: contactAvatarPath,
		ID:         0, // ID = 0 означает, что контакт не в БД
	}
	log.Printf("[Chat] 📋 Временный контакт создан: ID=0, PeerID=%s (полный=%s), Username=%q, AvatarPath=%q",
		peerID[:8], tempContact.PeerID, tempContact.Username, tempContact.AvatarPath)

	// Выбираем чат
	ui.selectChat(tempContact)
	log.Printf("[Chat] ✅ Чат выбран в UI")

	// Загружаем сообщения по chat_id (получаем чат по peer_id)
	ui.loadMessagesForPeer(peerID)
	log.Printf("[Chat] 📥 Загружены сообщения для контакта (ожидается 0 сообщений)")

	// Запрашиваем профиль у пира если P2P инициализирован
	if ui.p2pUI != nil {
		log.Printf("[Profile] 🔍 Запрос профиля у пира %s (асинхронно)...", peerID[:8])
		go func() {
			err := ui.p2pUI.RequestProfile(peerID)
			if err != nil {
				log.Printf("[Profile] ❌ Не удалось запросить профиль у пира %s: %v", peerID, err)
			} else {
				log.Printf("[Profile] ✅ Профиль запрошен у пира %s", peerID[:8])
			}
		}()
	}

	// Обновляем правую панель с профилем
	if ui.rightPanel != nil {
		log.Printf("[Profile] 🔄 Обновление правой панели с профилем пира %s", peerID[:8])
		ui.rightPanel.UpdateProfile(tempContact)
	}

	// Обновляем UI
	if ui.chatAreaObj != nil {
		ui.chatAreaObj.Refresh()
		log.Printf("[Chat] ✅ UI чата обновлён")
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
		// Безопасное получение PeerID для логирования
		fromPeer := ""
		if len(event.Message.FromPeerID) >= 8 {
			fromPeer = event.Message.FromPeerID[:8]
		} else if event.Message.FromPeerID != "" {
			fromPeer = event.Message.FromPeerID
		} else {
			fromPeer = "unknown"
		}

		log.Printf("[Chat] 📬 Получено событие сообщения: contactID=%d, from=%s, len=%d",
			event.ContactID, fromPeer, len(event.Message.Content))

		// Проверяем, открыт ли сейчас чат с этим контактом
		if ui.currentContact != nil && ui.currentContact.ID == event.ContactID {
			log.Printf("[Chat] 📭 Чат открыт, добавляем сообщение в UI")
			// Добавляем сообщение в UI
			if ui.chatPanel != nil {
				ui.chatPanel.AddMessage(event.Message, event.IsOutgoing)
				log.Printf("[Chat] ✅ Сообщение добавлено в UI чата")
			}
		} else {
			log.Printf("[Chat] ℹ️ Чат не открыт (currentContactID=%v, event.ContactID=%d), сообщение сохранено в БД",
				ui.currentContact != nil && ui.currentContact.ID != 0, event.ContactID)
		}

		// ✅ Обновляем левую панель со списком чатов
		// Это нужно для отображения последнего сообщения и времени
		if ui.leftPanel != nil {
			log.Printf("[Chat] 🔄 Обновление левой панели со списком чатов")
			ui.leftPanel.Refresh()
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

	log.Printf("[Chat] ✏️ Пользователь ввёл сообщение: %q (len=%d)", text, len(text))

	// Очищаем поле ввода
	ui.chatPanel.MessageInput().Clear()
	log.Printf("[Chat] 🧹 Поле ввода очищено")

	// Вызываем внешний обработчик если установлен
	if ui.onSendMessage != nil {
		log.Printf("[Chat] 🚀 Вызов обработчика отправки сообщения (onSendMessage)")
		ui.onSendMessage(text)
	}
}

// loadMessagesForPeer загружает сообщения для пира по peer_id
func (ui *UI) loadMessagesForPeer(peerID string) {
	if ui.chatPanel == nil {
		return
	}

	// Очищаем текущие сообщения
	ui.chatPanel.Clear()

	// Получаем чат по peer_id
	chat, err := queries.GetChatByPeerID(peerID)
	if err != nil {
		log.Printf("Ошибка получения чата: %v", err)
		return
	}
	if chat == nil {
		log.Printf("[Chat] 📚 Чат не найден, сообщений нет")
		return
	}

	// Загружаем сообщения по chat_id
	messages, err := queries.GetMessagesForChat(chat.ID, 100, 0)
	if err != nil {
		log.Printf("Ошибка загрузки сообщений: %v", err)
		return
	}

	log.Printf("[Chat] 📚 Получено %d сообщений из БД", len(messages))

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
	log.Printf("[Chat] 🗨️ Открытие локального чата (Избранное)")

	// Загружаем локальный профиль
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		log.Printf("[Chat] ❌ Ошибка загрузки локального профиля: %v", err)
		return
	}
	log.Printf("[Chat] 📋 Локальный профиль загружён: %s", localProfile.Username)

	// Получаем или создаём локальный чат (с contact_id = NULL)
	localChat, err := queries.GetOrCreateLocalChat(localProfile.PeerID)
	if err != nil {
		log.Printf("[Chat] ❌ Ошибка получения локального чата: %v", err)
		return
	}
	log.Printf("[Chat] 💬 Локальный чат: ID=%d, peer_id=%s", localChat.ID, localChat.PeerID[:8])

	// Создаём специальный контакт для локального чата (виртуальный, не из БД)
	localContact := models.NewLocalContact(
		localProfile.Username,
		localProfile.Title,
		localProfile.AvatarPath,
	)
	log.Printf("[Chat] 📝 Виртуальный контакт создан: ID=0, PeerID=__local__")

	// Выбираем чат
	ui.selectChat(localContact)
	log.Printf("[Chat] ✅ Чат выбран в UI")

	// Загружаем сообщения для локального чата через chat_id
	ui.loadMessagesForChat(localChat.ID)
	log.Printf("[Chat] 📥 Загружены сообщения для локального чата ID=%d", localChat.ID)

	// Обновляем правую панель с профилем
	if ui.rightPanel != nil {
		log.Printf("[Profile] 🔄 Обновление правой панели с локальным профилем")
		ui.rightPanel.UpdateProfile(localContact)
	}

	// Обновляем UI
	if ui.chatAreaObj != nil {
		ui.chatAreaObj.Refresh()
		log.Printf("[Chat] ✅ UI локального чата обновлён")
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

// RefreshRightPanel обновляет правую панель с профилем пира (вызывается из callback после загрузки профиля)
func (ui *UI) RefreshRightPanel(peerID string) {
	// Получаем профиль из БД
	profile, err := queries.GetProfileByPeerID(peerID)
	if err != nil || profile == nil {
		log.Printf("[Profile] ⚠️ Не удалось получить профиль для обновления UI: %v", err)
		return
	}

	// Создаём контакт с обновлёнными данными
	contact := &models.Contact{
		PeerID:     peerID,
		Username:   profile.Username,
		Title:      profile.Title,
		AvatarPath: profile.AvatarPath,
	}

	// Обновляем правую панель
	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(contact)
		log.Printf("[Profile] 🔄 Правая панель обновлена для %s", peerID[:8])
	}

	// Также обновляем левую панель (список чатов)
	if ui.leftPanel != nil {
		ui.leftPanel.Refresh()
		log.Printf("[Profile] 🔄 Левая панель обновлена")
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
