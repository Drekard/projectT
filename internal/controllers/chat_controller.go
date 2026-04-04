// Package controllers предоставляет контроллеры для управления бизнес-логикой приложения
package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"projectT/internal/services"
	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ChatMessageEvent представляет событие нового сообщения (алиас для удобства)
type ChatMessageEvent = services.ChatMessageEvent

// ChatController контролирует бизнес-логику чатов
// UI взаимодействует с этим контроллером через callback-функции
type ChatController struct {
	p2pUI          *network.UIP2P
	messageChannel <-chan *ChatMessageEvent

	// Callback-функции, устанавливаемые UI
	onMessageSent          func(message *models.ChatMessage)
	onMessageReceived      func(event *services.ChatMessageEvent)
	onChatOpened           func(contact *models.Contact)
	onChatClosed           func()
	onContactsRefreshed    func()
	onPinnedElementsLoaded func(peerID string)

	// Состояние
	currentContact *models.Contact
	currentChatID  int
	localPeerID    string
}

// NewChatController создаёт новый контроллер чатов
func NewChatController() *ChatController {
	cc := &ChatController{}

	// Подписываемся на события сообщений
	chatSvc := services.GetChatService()
	if chatSvc != nil {
		cc.messageChannel = chatSvc.Subscribe()
		go cc.handleMessageEvents()
	}

	return cc
}

// SetP2PService устанавливает P2P сервис
func (cc *ChatController) SetP2PService(p2pUI *network.UIP2P) {
	cc.p2pUI = p2pUI

	// Получаем локальный PeerID
	if p2pUI != nil {
		status := p2pUI.GetStatus()
		if status != nil {
			cc.localPeerID = status.PeerID
		}
	}
}

// SetOnMessageSent устанавливает callback при отправке сообщения
func (cc *ChatController) SetOnMessageSent(handler func(message *models.ChatMessage)) {
	cc.onMessageSent = handler
}

// SetOnMessageReceived устанавливает callback при получении сообщения
func (cc *ChatController) SetOnMessageReceived(handler func(event *services.ChatMessageEvent)) {
	cc.onMessageReceived = handler
}

// SetOnChatOpened устанавливает callback при открытии чата
func (cc *ChatController) SetOnChatOpened(handler func(contact *models.Contact)) {
	cc.onChatOpened = handler
}

// SetOnChatClosed устанавливает callback при закрытии чата
func (cc *ChatController) SetOnChatClosed(handler func()) {
	cc.onChatClosed = handler
}

// SetOnContactsRefreshed устанавливает callback при обновлении списка контактов
func (cc *ChatController) SetOnContactsRefreshed(handler func()) {
	cc.onContactsRefreshed = handler
}

// SetOnPinnedElementsLoaded устанавливает callback при загрузке закреплённых элементов
func (cc *ChatController) SetOnPinnedElementsLoaded(handler func(peerID string)) {
	cc.onPinnedElementsLoaded = handler
}

// handleMessageEvents обрабатывает события новых сообщений
func (cc *ChatController) handleMessageEvents() {
	for event := range cc.messageChannel {
		log.Printf("[ChatController] 📬 Получено событие сообщения: contactID=%d", event.ContactID)

		// Вызываем callback UI
		if cc.onMessageReceived != nil {
			cc.onMessageReceived(event)
		}

		// Обновляем список контактов
		if cc.onContactsRefreshed != nil {
			cc.onContactsRefreshed()
		}
	}
}

// SendMessage отправляет текстовое сообщение
func (cc *ChatController) SendMessage(text string) error {
	log.Printf("[ChatController] 📤 Отправка сообщения: len=%d", len(text))

	if cc.p2pUI == nil {
		return fmt.Errorf("P2P сервис не инициализирован")
	}

	if cc.currentContact == nil || cc.currentContact.PeerID == "" {
		return fmt.Errorf("текущий контакт не выбран или не имеет PeerID")
	}

	// Для локального чата не отправляем через P2P
	if cc.currentContact.IsLocalChat() {
		log.Printf("[ChatController] ℹ️ Локальный чат, отправка только в БД")
		// Отправляем только в ChatService для сохранения в БД
		chatSvc := services.GetChatService()
		if chatSvc != nil {
			_, err := chatSvc.SendTextMessage(0, cc.currentContact.PeerID, cc.localPeerID, text)
			return err
		}
		return nil
	}

	peerID, err := peer.Decode(cc.currentContact.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	log.Printf("[ChatController] 📤 Отправка сообщения пиру %s", peerID[:8])

	// Отправляем через P2P сервис (который сохранит в БД и отправит через сеть)
	if err := cc.p2pUI.SendMessage(peerID, text); err != nil {
		return fmt.Errorf("ошибка отправки сообщения: %w", err)
	}

	log.Printf("[ChatController] ✅ Сообщение отправлено")
	return nil
}

// SendElementMessage отправляет элемент в чат
func (cc *ChatController) SendElementMessage(item *models.Item) error {
	log.Printf("[ChatController] 📤 Отправка элемента: element_uuid=%s", item.ElementUUID)

	if cc.p2pUI == nil {
		return fmt.Errorf("P2P сервис не инициализирован")
	}

	if cc.currentContact == nil || cc.currentContact.PeerID == "" {
		return fmt.Errorf("текущий контакт не выбран или не имеет PeerID")
	}

	// Для локального чата не отправляем через P2P
	if cc.currentContact.IsLocalChat() {
		log.Printf("[ChatController] ℹ️ Локальный чат, элемент не отправляется через P2P")
		return nil
	}

	peerID, err := peer.Decode(cc.currentContact.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	log.Printf("[ChatController] 📤 Отправка элемента пиру %s", peerID[:8])

	// Отправляем через P2P сервис
	if err := cc.p2pUI.SendElementMessage(peerID, item); err != nil {
		return fmt.Errorf("ошибка отправки элемента: %w", err)
	}

	log.Printf("[ChatController] ✅ Элемент отправлен")
	return nil
}

// OpenChat открывает чат с контактом
func (cc *ChatController) OpenChat(contact *models.Contact) error {
	log.Printf("[ChatController] 🗨️ Открытие чата: peerID=%s, username=%s", contact.PeerID[:8], contact.Username)

	cc.currentContact = contact
	cc.currentChatID = contact.ID

	// Вызываем callback UI
	if cc.onChatOpened != nil {
		cc.onChatOpened(contact)
	}

	// Загружаем сообщения
	if _, err := cc.LoadMessages(); err != nil {
		log.Printf("[ChatController] ⚠️ Ошибка загрузки сообщений: %v", err)
	}

	// ✅ Загружаем pinned элементы ТОЛЬКО при открытии чата
	// Это происходит когда пользователь явно начинает общение
	if cc.p2pUI != nil && !contact.IsLocalChat() {
		go func() {
			// Сначала запрашиваем профиль
			err := cc.p2pUI.RequestProfile(contact.PeerID)
			if err != nil {
				log.Printf("[ChatController] ❌ Не удалось запросить профиль у пира: %v", err)
				return
			}

			// Затем загружаем pinned элементы из профиля
			cc.downloadPinnedElements(contact.PeerID)
		}()
	}

	return nil
}

// OpenPeerChat открывает чат с пиром (создаёт временный контакт)
func (cc *ChatController) OpenPeerChat(peerID, username string) error {
	log.Printf("[ChatController] 🆕 Открытие чата с пиром: %s (%s)", username, peerID[:8])

	// Получаем профиль пира из БД для корректного отображения
	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		log.Printf("[ChatController] 📋 Профиль пира найден: username=%q", profile.Username)
	}

	// Создаём временный контакт для чата
	contactUsername := username
	contactTitle := ""
	contactAvatarPath := ""
	if profile != nil {
		contactUsername = profile.Username
		contactTitle = profile.Title
		contactAvatarPath = profile.AvatarPath
	}

	tempContact := &models.Contact{
		PeerID:     peerID,
		Username:   contactUsername,
		Title:      contactTitle,
		AvatarPath: contactAvatarPath,
		ID:         0, // ID = 0 означает, что контакт не в БД
	}

	return cc.OpenChat(tempContact)
}

// OpenLocalChat открывает локальный чат с самим собой
func (cc *ChatController) OpenLocalChat() error {
	log.Printf("[ChatController] 🗨️ Открытие локального чата (Избранное)")

	// Загружаем локальный профиль
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		return fmt.Errorf("ошибка загрузки локального профиля: %w", err)
	}

	// Получаем или создаём локальный чат (с contact_id = NULL)
	_, err = queries.GetOrCreateLocalChat(localProfile.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка получения локального чата: %w", err)
	}

	// Создаём специальный контакт для локального чата (виртуальный, не из БД)
	localContact := models.NewLocalContact(
		localProfile.Username,
		localProfile.Title,
		localProfile.AvatarPath,
	)

	return cc.OpenChat(localContact)
}

// CloseChat закрывает текущий чат
func (cc *ChatController) CloseChat() error {
	log.Printf("[ChatController] 🚪 Закрытие чата")

	cc.currentContact = nil
	cc.currentChatID = 0

	// Вызываем callback UI
	if cc.onChatClosed != nil {
		cc.onChatClosed()
	}

	return nil
}

// LoadMessages загружает сообщения для текущего чата
func (cc *ChatController) LoadMessages() ([]*models.ChatMessage, error) {
	if cc.currentContact == nil {
		return nil, fmt.Errorf("текущий контакт не выбран")
	}

	log.Printf("[ChatController] 📥 Загрузка сообщений для контакта: peerID=%s", cc.currentContact.PeerID[:8])

	// Получаем чат по peer_id
	chat, err := queries.GetChatByPeerID(cc.currentContact.PeerID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения чата: %w", err)
	}
	if chat == nil {
		log.Printf("[ChatController] 📚 Чат не найден, сообщений нет")
		return []*models.ChatMessage{}, nil
	}

	// Загружаем сообщения по chat_id
	messages, err := queries.GetMessagesForChat(chat.ID, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки сообщений: %w", err)
	}

	log.Printf("[ChatController] 📚 Получено %d сообщений из БД", len(messages))
	return messages, nil
}

// LoadMessagesForChat загружает сообщения для чата по ID
func (cc *ChatController) LoadMessagesForChat(chatID int) ([]*models.ChatMessage, error) {
	log.Printf("[ChatController] 📥 Загрузка сообщений для чата ID=%d", chatID)

	// Загружаем сообщения по chat_id
	messages, err := queries.GetMessagesForChat(chatID, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки сообщений: %w", err)
	}

	log.Printf("[ChatController] 📚 Получено %d сообщений из БД", len(messages))
	return messages, nil
}

// GetCurrentContact возвращает текущий контакт
func (cc *ChatController) GetCurrentContact() *models.Contact {
	return cc.currentContact
}

// GetCurrentChatID возвращает текущий ID чата
func (cc *ChatController) GetCurrentChatID() int {
	return cc.currentChatID
}

// GetLocalPeerID возвращает локальный PeerID
func (cc *ChatController) GetLocalPeerID() string {
	return cc.localPeerID
}

// GetContactByPeerID получает контакт по PeerID
func (cc *ChatController) GetContactByPeerID(peerID string) (*models.Contact, error) {
	return queries.GetContactByPeerID(peerID)
}

// RefreshContacts обновляет список контактов
func (cc *ChatController) RefreshContacts() {
	log.Printf("[ChatController] 🔄 Обновление списка контактов")
	if cc.onContactsRefreshed != nil {
		cc.onContactsRefreshed()
	}
}

// DeleteContact удаляет контакт
func (cc *ChatController) DeleteContact(chatID int, peerID string) error {
	log.Printf("[ChatController] 🗑️ Удаление контакта: chatID=%d, peerID=%s", chatID, peerID[:8])

	if err := queries.DeleteContact(chatID); err != nil {
		return fmt.Errorf("ошибка удаления контакта: %w", err)
	}

	// Если текущий чат был удалён, закрываем его
	if cc.currentChatID == chatID {
		if err := cc.CloseChat(); err != nil {
			log.Printf("[ChatController] ⚠️ Ошибка закрытия чата: %v", err)
		}
	}

	// Обновляем список контактов
	cc.RefreshContacts()

	return nil
}

// SendElementMetadata отправляет метаданные элемента через P2P
func (cc *ChatController) SendElementMetadata(ctx context.Context, item *models.Item) (string, error) {
	if cc.p2pUI == nil || cc.currentContact == nil {
		return "", fmt.Errorf("P2P сервис или контакт не инициализированы")
	}

	peerID, err := peer.Decode(cc.currentContact.PeerID)
	if err != nil {
		return "", fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	// Получаем P2P Network для доступа к Transfer Service
	p2pNet := cc.p2pUI.GetNetwork()
	if p2pNet == nil {
		return "", fmt.Errorf("P2P сеть не инициализирована")
	}

	// Получаем Transfer Service
	transferSvc := p2pNet.Transfer()
	if transferSvc == nil {
		return "", fmt.Errorf("TransferService не инициализирован")
	}

	// Отправляем метаданные через Transfer Service
	transferID, err := transferSvc.SendElementMetadata(ctx, peerID, item.ElementUUID, item.Title, item.Description, item.ContentMeta)
	if err != nil {
		return "", fmt.Errorf("ошибка отправки метаданных: %w", err)
	}

	return transferID, nil
}

// RequestItem запрашивает элемент у пира
func (cc *ChatController) RequestItem(ctx context.Context, peerIDStr, elementUUID string) (*models.Item, error) {
	if cc.p2pUI == nil {
		return nil, fmt.Errorf("P2P сервис не инициализирован")
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	// Получаем P2P Network для доступа к ItemSync Service
	p2pNet := cc.p2pUI.GetNetwork()
	if p2pNet == nil {
		return nil, fmt.Errorf("P2P сеть не инициализирована")
	}

	// Получаем ItemSync Service
	itemSyncSvc := p2pNet.ItemSync()
	if itemSyncSvc == nil {
		return nil, fmt.Errorf("ItemSyncService не инициализирован")
	}

	// Запрашиваем элемент через ItemSync Service
	item, err := itemSyncSvc.RequestItemByElementUUID(ctx, peerID, elementUUID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса элемента: %w", err)
	}

	return item, nil
}

// downloadPinnedElements загружает pinned элементы из профиля пира
// Вызывается ТОЛЬКО при открытии чата (не при добавлении контакта!)
func (cc *ChatController) downloadPinnedElements(peerIDStr string) {
	if cc.p2pUI == nil {
		log.Printf("[ChatController] ❌ P2P сервис не инициализирован")
		return
	}

	// Получаем профиль пира из БД
	profile, err := queries.GetProfileByPeerID(peerIDStr)
	if err != nil || profile == nil {
		log.Printf("[ChatController] ⚠️ Профиль пира %s не найден в БД", peerIDStr[:8])
		return
	}

	// Парсим JSON с pinned UUIDs
	var pinnedUUIDs []string
	if err := json.Unmarshal([]byte(profile.PinnedUUIDs), &pinnedUUIDs); err != nil {
		log.Printf("[ChatController] ⚠️ Ошибка парсинга PinnedUUIDs: %v", err)
		return
	}

	if len(pinnedUUIDs) == 0 {
		log.Printf("[ChatController] ℹ️ У пира %s нет закреплённых элементов", peerIDStr[:8])
		return
	}

	log.Printf("[ChatController] 📌 Загрузка %d pinned элементов у пира %s", len(pinnedUUIDs), peerIDStr[:8])

	// Декодируем PeerID
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		log.Printf("[ChatController] ❌ Ошибка декодирования PeerID: %v", err)
		return
	}

	// Получаем ItemSync сервис
	p2pNet := cc.p2pUI.GetNetwork()
	if p2pNet == nil {
		log.Printf("[ChatController] ❌ P2P сеть не инициализирована")
		return
	}

	itemSyncSvc := p2pNet.ItemSync()
	if itemSyncSvc == nil {
		log.Printf("[ChatController] ❌ ItemSyncService не инициализирован")
		return
	}

	// Загружаем каждый элемент
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	loadedCount := 0
	for _, uuid := range pinnedUUIDs {
		// Проверяем, есть ли уже элемент в БД
		existing, err := queries.GetItemByElementUUID(uuid)
		if err == nil && existing != nil {
			log.Printf("[ChatController] ✅ Элемент уже существует: UUID=%s, ID=%d", uuid[:8], existing.ID)
			loadedCount++
			continue
		}

		// Запрашиваем элемент у пира
		log.Printf("[ChatController] 📥 Запрос элемента: UUID=%s", uuid[:8])
		item, err := itemSyncSvc.RequestItemByElementUUID(ctx, peerID, uuid)
		if err != nil {
			log.Printf("[ChatController] ❌ Ошибка загрузки элемента %s: %v", uuid[:8], err)
			continue
		}

		if item != nil {
			loadedCount++
			log.Printf("[ChatController] ✅ Элемент загружён: UUID=%s, ID=%d, Title=%q", uuid[:8], item.ID, item.Title)
		}
	}

	log.Printf("[ChatController] 📊 Загружено %d из %d pinned элементов", loadedCount, len(pinnedUUIDs))

	// Уведомляем UI о завершении загрузки закреплённых элементов
	if cc.onPinnedElementsLoaded != nil && loadedCount > 0 {
		cc.onPinnedElementsLoaded(peerIDStr)
	}
}
