package chats

import (
	"log"

	"projectT/internal/controllers"
	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/chats/center"
	"projectT/internal/ui/workspace/chats/dialogs"
	"projectT/internal/ui/workspace/chats/left"
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
	chatController *controllers.ChatController
	currentChatID  int
	currentContact *models.Contact
	chatPanel      *center.ChatPanel

	// Компоненты панелей
	leftPanel   *left.Panel
	rightPanel  *right.Panel
	chatAreaObj *fyne.Container

	// Менеджеры
	chatMenuManager *dialogs.ChatMenuManager
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

// SetP2PService устанавливает P2P сервис и создаёт контроллер
func (ui *UI) SetP2PService(p2pUI *network.UIP2P) {
	ui.p2pUI = p2pUI

	// Создаём контроллер чатов
	ui.chatController = controllers.NewChatController()
	ui.chatController.SetP2PService(p2pUI)

	// Настраиваем callback-функции контроллера
	ui.setupControllerCallbacks()
}

// setupControllerCallbacks настраивает callback-функции контроллера для связи с UI
func (ui *UI) setupControllerCallbacks() {
	if ui.chatController == nil {
		return
	}

	// Callback при открытии чата
	ui.chatController.SetOnChatOpened(func(contact *models.Contact) {
		log.Printf("[Chat] 🗨️ Чат открыт: %s", contact.Username)

		// Создаём панель чата
		chatPanel := ui.createChatPanel(contact)
		ui.chatAreaObj.Objects = []fyne.CanvasObject{chatPanel}
		ui.chatAreaObj.Refresh()

		// Обновляем профиль
		if ui.rightPanel != nil {
			ui.rightPanel.UpdateProfile(contact)
		}
	})

	// Callback при закрытии чата
	ui.chatController.SetOnChatClosed(func() {
		log.Printf("[Chat] 🚪 Чат закрыт")

		// Показываем пустую панель
		emptyPanel := ui.createEmptyPanel()
		ui.chatAreaObj.Objects = []fyne.CanvasObject{emptyPanel}
		ui.chatAreaObj.Refresh()
	})

	// Callback при обновлении контактов
	ui.chatController.SetOnContactsRefreshed(func() {
		log.Printf("[Chat] 🔄 Контакты обновлены")
		if ui.leftPanel != nil {
			ui.leftPanel.Refresh()
		}
	})

	// Callback при получении сообщения
	ui.chatController.SetOnMessageReceived(func(event *controllers.ChatMessageEvent) {
		log.Printf("[Chat] 📬 Получено сообщение: contactID=%d", event.ContactID)

		// Проверяем, открыт ли чат с этим контактом
		if ui.currentContact != nil && ui.currentContact.ID == event.ContactID {
			log.Printf("[Chat] 📭 Чат открыт, добавляем сообщение в UI")
			if ui.chatPanel != nil {
				ui.chatPanel.AddMessage(event.Message, event.IsOutgoing)
			}
		}

		// Обновляем левую панель
		if ui.leftPanel != nil {
			ui.leftPanel.Refresh()
		}
	})
}

// selectChat выбирает чат с пиром (устарел, использовать через контроллер)
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

	// Используем контроллер для открытия чата
	if ui.chatController != nil {
		if err := ui.chatController.OpenPeerChat(peerID, username); err != nil {
			log.Printf("[Chat] ❌ Ошибка открытия чата: %v", err)
		}
	} else {
		// Fallback для обратной совместимости
		ui.openPeerChatLegacy(peerID, username)
	}
}

// openPeerChatLegacy открывает чат по-старому (для обратной совместимости)
func (ui *UI) openPeerChatLegacy(peerID, username string) {
	log.Printf("[Chat] ⚠️ Использование legacy метода openPeerChatLegacy")

	// Получаем профиль пира из БД для корректного отображения
	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		log.Printf("[Chat] 📋 Профиль пира найден: username=%q", profile.Username)
	}

	contactUsername := username
	contactAvatarPath := ""
	if profile != nil {
		contactUsername = profile.Username
		contactAvatarPath = profile.AvatarPath
	}

	tempContact := &models.Contact{
		PeerID:     peerID,
		Username:   contactUsername,
		AvatarPath: contactAvatarPath,
		ID:         0,
	}

	ui.selectChat(tempContact)
	ui.loadMessagesForPeer(peerID)

	if ui.p2pUI != nil {
		go func() {
			_ = ui.p2pUI.RequestProfile(peerID)
		}()
	}

	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(tempContact)
	}

	if ui.chatAreaObj != nil {
		ui.chatAreaObj.Refresh()
	}
}

// SubscribeToMessages подписывается на события сообщений (устарел, контроллер сам подписывается)
func (ui *UI) SubscribeToMessages() {
	log.Printf("[Chat] ℹ️ SubscribeToMessages вызван - контроллер сам подписывается на события")
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

// sendMessage отправляет сообщение (вызывается из chatPanel)
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

	// Отправляем сообщение через контроллер
	if ui.chatController != nil {
		if err := ui.chatController.SendMessage(text); err != nil {
			log.Printf("[Chat] ❌ Ошибка отправки сообщения: %v", err)
		}
	}
}

// loadMessagesForPeer загружает сообщения для пира по peer_id
func (ui *UI) loadMessagesForPeer(peerID string) {
	if ui.chatPanel == nil {
		return
	}

	// Очищаем текущие сообщения
	ui.chatPanel.Clear()

	// Получаем сообщения через контроллер
	messages, err := ui.chatController.LoadMessages()
	if err != nil {
		log.Printf("[Chat] ❌ Ошибка загрузки сообщений: %v", err)
		return
	}

	log.Printf("[Chat] 📚 Получено %d сообщений из БД", len(messages))

	// Загружаем сообщения в панель
	ui.chatPanel.LoadMessages(messages, ui.chatController.GetLocalPeerID())
}

// closeChat закрывает текущий чат (вызывается из chatPanel)
func (ui *UI) closeChat() {
	if ui.chatController != nil {
		_ = ui.chatController.CloseChat()
	}
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
	// Удаляем контакт через контроллер
	if ui.chatController != nil {
		if err := ui.chatController.DeleteContact(chatID, peerID); err != nil {
			log.Printf("[Chat] ❌ Ошибка удаления чата: %v", err)
		}
	}

	log.Printf("Чат %s (%d) удалён", peerID, chatID)
}

// OpenLocalChat открывает локальный чат с самим собой (реализация для left.UIProvider)
func (ui *UI) OpenLocalChat() {
	log.Printf("[Chat] 🗨️ Открытие локального чата (Избранное)")

	// Используем контроллер для открытия локального чата
	if ui.chatController != nil {
		if err := ui.chatController.OpenLocalChat(); err != nil {
			log.Printf("[Chat] ❌ Ошибка открытия локального чата: %v", err)
		}
	} else {
		// Fallback для обратной совместимости
		ui.openLocalChatLegacy()
	}
}

// openLocalChatLegacy открывает локальный чат по-старому (для обратной совместимости)
func (ui *UI) openLocalChatLegacy() {
	log.Printf("[Chat] ⚠️ Использование legacy метода openLocalChatLegacy")

	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		log.Printf("[Chat] ❌ Ошибка загрузки локального профиля: %v", err)
		return
	}

	localChat, err := queries.GetOrCreateLocalChat(localProfile.PeerID)
	if err != nil {
		log.Printf("[Chat] ❌ Ошибка получения локального чата: %v", err)
		return
	}

	localContact := models.NewLocalContact(
		localProfile.Username,
		localProfile.Title,
		localProfile.AvatarPath,
	)

	ui.selectChat(localContact)
	ui.loadMessagesForChat(localChat.ID)

	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(localContact)
	}

	if ui.chatAreaObj != nil {
		ui.chatAreaObj.Refresh()
	}
}

// loadMessagesForChat загружает сообщения для чата по ID
func (ui *UI) loadMessagesForChat(chatID int) {
	if ui.chatPanel == nil {
		return
	}

	// Очищаем текущие сообщения
	ui.chatPanel.Clear()

	// Загружаем сообщения через контроллер
	messages, err := ui.chatController.LoadMessagesForChat(chatID)
	if err != nil {
		log.Printf("[Chat] ❌ Ошибка загрузки сообщений: %v", err)
		return
	}

	// Загружаем сообщения в панель
	ui.chatPanel.LoadMessages(messages, ui.chatController.GetLocalPeerID())
}

// RefreshContactsList обновляет список чатов (публичный метод для вызова извне)
func (ui *UI) RefreshContactsList() {
	if ui.leftPanel != nil {
		ui.leftPanel.Refresh()
	}
}

// RefreshRightPanel обновляет правую панель с профилем пира (вызывается из callback после загрузки профиля)
func (ui *UI) RefreshRightPanel(peerID string) {
	// Используем контроллер для получения информации о пире
	if ui.chatController != nil {
		contact, err := ui.chatController.GetContactByPeerID(peerID)
		if err == nil && contact != nil {
			if ui.rightPanel != nil {
				ui.rightPanel.UpdateProfile(contact)
				log.Printf("[Profile] 🔄 Правая панель обновлена для %s", peerID[:8])
			}
			if ui.leftPanel != nil {
				ui.leftPanel.Refresh()
				log.Printf("[Profile] 🔄 Левая панель обновлена")
			}
			return
		}
	}

	// Fallback: получаем профиль напрямую из БД
	profile, err := queries.GetProfileByPeerID(peerID)
	if err != nil || profile == nil {
		log.Printf("[Profile] ⚠️ Не удалось получить профиль для обновления UI: %v", err)
		return
	}

	contact := &models.Contact{
		PeerID:     peerID,
		Username:   profile.Username,
		Title:      profile.Title,
		AvatarPath: profile.AvatarPath,
	}

	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(contact)
	}
	if ui.leftPanel != nil {
		ui.leftPanel.Refresh()
	}
}

// RefreshDemoElementsAfterSync обновляет витрину элементов правой панели после синхронизации элементов
func (ui *UI) RefreshDemoElementsAfterSync(peerID string) {
	if ui.rightPanel != nil {
		ui.rightPanel.RefreshDemoElementsAfterSync(peerID)
		log.Printf("[Profile] 📢 Витрина элементов обновлена через callback для %s", peerID[:8])
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
