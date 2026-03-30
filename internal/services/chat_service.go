// Package services предоставляет сервисы для бизнес-логики приложения
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// globalP2PNetwork - глобальный доступ к P2P сети (устанавливается при инициализации)
var globalP2PNetwork interface {
	SendMessage(ctx context.Context, peerID peer.ID, content, contentType, metadata string) error
}
var p2pNetworkMu sync.RWMutex

// SetGlobalP2PNetwork устанавливает глобальный P2P сервис для использования в ChatService
func SetGlobalP2PNetwork(p2pNet interface {
	SendMessage(ctx context.Context, peerID peer.ID, content, contentType, metadata string) error
}) {
	p2pNetworkMu.Lock()
	defer p2pNetworkMu.Unlock()
	globalP2PNetwork = p2pNet
	log.Println("[Chat] ✅ Глобальный P2P сервис установлен")
}

// getGlobalP2PNetwork возвращает глобальный P2P сервис
func getGlobalP2PNetwork() interface {
	SendMessage(ctx context.Context, peerID peer.ID, content, contentType, metadata string) error
} {
	p2pNetworkMu.RLock()
	defer p2pNetworkMu.RUnlock()
	return globalP2PNetwork
}

// SetTransferService устанавливает Transfer Service для отправки прогресса
func (cs *ChatService) SetTransferService(transferSvc interface {
	SendElementMetadata(ctx context.Context, peerID peer.ID, elementUUID, title, description, contentMeta string) (string, error)
}) {
	cs.transferSvc = transferSvc
	log.Println("[Chat] ✅ Transfer Service установлен")
}

// SetItemSyncService устанавливает ItemSync Service для загрузки элементов
func (cs *ChatService) SetItemSyncService(itemSyncSvc interface {
	RequestItemByElementUUID(ctx context.Context, peerID peer.ID, elementUUID string) (*models.Item, error)
}) {
	cs.itemSync = itemSyncSvc
	log.Println("[Chat] ✅ ItemSync Service установлен")
}

// ItemSync возвращает ItemSync сервис
func (cs *ChatService) ItemSync() interface {
	RequestItemByElementUUID(ctx context.Context, peerID peer.ID, elementUUID string) (*models.Item, error)
} {
	return cs.itemSync
}

// ChatMessageEvent представляет событие нового сообщения
type ChatMessageEvent struct {
	ContactID   int
	ContactName string
	Message     *models.ChatMessage
	IsOutgoing  bool
}

// ChatService предоставляет сервис для работы с чатами
type ChatService struct {
	// Канал для уведомлений о новых сообщениях
	messageChannel chan *ChatMessageEvent
	// Подписчики на события сообщений
	subscribers []chan *ChatMessageEvent
	transferSvc interface {
		SendElementMetadata(ctx context.Context, peerID peer.ID, elementUUID, title, description, contentMeta string) (string, error)
	} // Transfer Service для отправки прогресса
	itemSync interface {
		RequestItemByElementUUID(ctx context.Context, peerID peer.ID, elementUUID string) (*models.Item, error)
	} // ItemSync Service для загрузки элементов
}

// globalChatService - глобальный экземпляр сервиса чатов
var globalChatService *ChatService

// init инициализирует глобальный сервис чатов
func init() {
	globalChatService = NewChatService()
}

// GetChatService возвращает глобальный экземпляр сервиса чатов
func GetChatService() *ChatService {
	return globalChatService
}

// NewChatService создает новый экземпляр сервиса чатов
func NewChatService() *ChatService {
	cs := &ChatService{
		messageChannel: make(chan *ChatMessageEvent, 100),
		subscribers:    make([]chan *ChatMessageEvent, 0),
	}
	// Запускаем обработчик событий
	go cs.processEvents()
	return cs
}

// processEvents обрабатывает события сообщений
func (cs *ChatService) processEvents() {
	for event := range cs.messageChannel {
		// Рассылаем событие всем подписчикам
		for _, sub := range cs.subscribers {
			select {
			case sub <- event:
				// Успешно отправлено
			default:
				// Канал переполнен, пропускаем
			}
		}
	}
}

// Subscribe подписывается на события сообщений
func (cs *ChatService) Subscribe() <-chan *ChatMessageEvent {
	ch := make(chan *ChatMessageEvent, 10)
	cs.subscribers = append(cs.subscribers, ch)
	return ch
}

// SendTextMessage отправляет текстовое сообщение
func (cs *ChatService) SendTextMessage(contactID int, recipientPeerID, fromPeerID string, content string) (*models.ChatMessage, error) {
	log.Printf("[Chat] 📤 SendTextMessage: contactID=%d, recipientPeerID=%s, fromPeerID=%s, len=%d",
		contactID, recipientPeerID[:min(10, len(recipientPeerID))], fromPeerID[:min(10, len(fromPeerID))], len(content))

	// Получаем peer_id из контакта
	var peerID string
	var contactIDPtr *int
	if contactID == 0 {
		// Для contactID=0 проверяем, локальный ли это чат
		if recipientPeerID == fromPeerID {
			// Локальный чат
			peerID = fromPeerID
			contactIDPtr = nil
			log.Printf("[Chat] 📝 Локальный чат: peerID=%s", peerID[:8])
		} else {
			// Это чат с пиром, но без контакта в БД (временный контакт)
			// Используем recipientPeerID как peer_id получателя
			peerID = recipientPeerID
			contactIDPtr = nil
			log.Printf("[Chat] 👤 Чат с пиром (временный контакт): peerID=%s", peerID[:8])
		}
	} else {
		contact, err := queries.GetContact(contactID)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения контакта: %w", err)
		}
		peerID = contact.PeerID
		contactIDPtr = &contactID
		log.Printf("[Chat] 👤 Чат с пиром: contactID=%d, peerID=%s, username=%q", contactID, peerID[:8], contact.Username)
	}

	// Получаем или создаём чат
	chat, err := queries.GetOrCreateChat(peerID, contactIDPtr)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения чата: %w", err)
	}
	log.Printf("[Chat] 📋 Чат получен/создан: chat_id=%d, peer_id=%s", chat.ID, peerID[:8])

	message := &models.ChatMessage{
		ChatID:      chat.ID,
		FromPeerID:  fromPeerID,
		Content:     content,
		ContentType: "text",
		SentAt:      time.Now(),
		IsRead:      true,
	}

	if err := queries.CreateChatMessage(message); err != nil {
		return nil, fmt.Errorf("ошибка сохранения сообщения: %w", err)
	}

	// Обновляем время последнего сообщения в чате
	if err := queries.UpdateChatLastMessage(chat.ID, message.SentAt); err != nil {
		log.Printf("Предупреждение: не удалось обновить время чата: %v", err)
	}

	// Получаем имя контакта для события
	contactName := "Локальный чат"
	if contactID != 0 {
		contact, err := queries.GetContact(contactID)
		if err == nil && contact != nil {
			contactName = contact.Username
		}
	}

	// Отправляем событие
	cs.messageChannel <- &ChatMessageEvent{
		ContactID:   contactID,
		ContactName: contactName,
		Message:     message,
		IsOutgoing:  true,
	}

	return message, nil
}

// SendElementMessage отправляет элемент в чат
// Если chat с peer_id != local peer и P2P активен - отправляет элемент через P2P
func (cs *ChatService) SendElementMessage(contactID int, recipientPeerID, fromPeerID string, item *models.Item) (*models.ChatMessage, error) {
	log.Printf("[Chat] 📤 SendElementMessage: contactID=%d, recipientPeerID=%s, fromPeerID=%s, item_id=%d, element_uuid=%s, title=%q",
		contactID, recipientPeerID[:min(10, len(recipientPeerID))], fromPeerID[:min(10, len(fromPeerID))], item.ID, item.ElementUUID, item.Title)

	// Получаем peer_id из контакта
	var peerID string
	var isLocalChat bool
	if contactID == 0 {
		// Для contactID=0 проверяем, локальный ли это чат
		// Если recipientPeerID совпадает с fromPeerID - это локальный чат
		if recipientPeerID == fromPeerID {
			peerID = fromPeerID
			isLocalChat = true
			log.Printf("[Chat] 📝 Локальный чат: peerID=%s", peerID[:8])
		} else {
			// Это чат с пиром, но без контакта в БД (временный контакт)
			// Используем recipientPeerID как peer_id получателя
			peerID = recipientPeerID
			isLocalChat = false
			log.Printf("[Chat] 👤 Чат с пиром (временный контакт): peerID=%s", peerID[:8])
		}
	} else {
		contact, err := queries.GetContact(contactID)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения контакта: %w", err)
		}
		peerID = contact.PeerID
		isLocalChat = false
		log.Printf("[Chat] 👤 Чат с пиром: contactID=%d, peerID=%s, username=%q", contactID, peerID[:8], contact.Username)
	}

	// Получаем или создаём чат
	chat, err := queries.GetOrCreateChat(peerID, &contactID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения чата: %w", err)
	}
	log.Printf("[Chat] 📋 Чат получен/создан: chat_id=%d, peer_id=%s", chat.ID, peerID[:8])

	// Создаём метаданные элемента
	// Хеш используется для проверки дубликатов, передача идёт по element_uuid
	metadata := map[string]interface{}{
		"item_id":      item.ID,
		"item_type":    string(item.Type),
		"item_title":   item.Title,
		"item_desc":    item.Description,
		"content_meta": item.ContentMeta,
		"item_hash":    item.Hash,
		"sent_at":      item.CreatedAt.Format(time.RFC3339),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	// Создаём сообщение
	message := &models.ChatMessage{
		ChatID:      chat.ID,
		FromPeerID:  fromPeerID,
		Content:     item.ElementUUID,
		ContentType: "element",
		Metadata:    string(metadataJSON),
		SentAt:      time.Now(),
		IsRead:      true,
	}

	if err := queries.CreateChatMessage(message); err != nil {
		return nil, fmt.Errorf("ошибка сохранения сообщения: %w", err)
	}
	log.Printf("[Chat] 💾 Сообщение сохранено в БД: message_id=%d, chat_id=%d", message.ID, chat.ID)

	// Обновляем время последнего сообщения в чате
	if err := queries.UpdateChatLastMessage(chat.ID, message.SentAt); err != nil {
		log.Printf("Предупреждение: не удалось обновить время чата: %v", err)
	}

	// Получаем имя контакта для события
	contactName := "Локальный чат"
	if contactID != 0 {
		contact, err := queries.GetContact(contactID)
		if err == nil && contact != nil {
			contactName = contact.Username
		}
	}

	// Отправляем событие UI
	cs.messageChannel <- &ChatMessageEvent{
		ContactID:   contactID,
		ContactName: contactName,
		Message:     message,
		IsOutgoing:  true,
	}
	log.Printf("[Chat] 📢 Уведомление UI отправлено: contactID=%d, contactName=%q", contactID, contactName)

	// Если это не локальный чат - отправляем элемент через P2P
	if !isLocalChat {
		log.Printf("[Chat] 📤 Отправка элемента через P2P пиру %s: element_uuid=%s", peerID[:8], item.ElementUUID)
		go func() {
			// Получаем глобальный P2P сервис
			p2pNet := getGlobalP2PNetwork()
			if p2pNet == nil {
				log.Printf("[Chat] ❌ P2P сеть не инициализирована")
				return
			}

			// Декодируем PeerID
			targetPeerID, err := peer.Decode(peerID)
			if err != nil {
				log.Printf("[Chat] ❌ Ошибка декодирования PeerID: %v", err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Отправляем метаданные через Transfer Service (для отображения прогресса)
			if cs.transferSvc != nil {
				log.Printf("[Chat] 📊 Отправка метаданных через Transfer Service...")
				transferID, err := cs.transferSvc.SendElementMetadata(
					ctx, targetPeerID,
					item.ElementUUID,
					item.Title,
					item.Description,
					item.ContentMeta,
				)
				if err != nil {
					log.Printf("[Chat] ❌ Ошибка Transfer Service: %v", err)
				} else {
					log.Printf("[Chat] ✅ Transfer ID=%s, метаданные отправляются...", transferID)
				}
			}

			// Отправляем сообщение через P2P
			if err := p2pNet.SendMessage(ctx, targetPeerID, item.ElementUUID, "element", string(metadataJSON)); err != nil {
				log.Printf("[Chat] ❌ Ошибка P2P отправки элемента: %v", err)
			} else {
				log.Printf("[Chat] ✅ Элемент отправлен через P2P пиру %s", peerID[:8])
			}
		}()
	}

	return message, nil
}

// NotifyNewMessage отправляет уведомление о новом сообщении всем подписчикам
// Используется P2P сервисом при получении входящих сообщений
func (cs *ChatService) NotifyNewMessage(contactID int, contactName string, message *models.ChatMessage, isIncoming bool) {
	select {
	case cs.messageChannel <- &ChatMessageEvent{
		ContactID:   contactID,
		ContactName: contactName,
		Message:     message,
		IsOutgoing:  !isIncoming, // isIncoming = true → IsOutgoing = false
	}:
		// Успешно отправлено
	default:
		// Канал переполнен, пропускаем
	}
}
