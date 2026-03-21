// Package services предоставляет сервисы для бизнес-логики приложения
package services

import (
	"encoding/json"
	"fmt"
	"log"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"time"
)

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
func (cs *ChatService) SendTextMessage(contactID int, fromPeerID string, content string) (*models.ChatMessage, error) {
	// Получаем peer_id из контакта
	var peerID string
	if contactID == 0 {
		// Локальный чат
		profile, err := queries.GetLocalProfile()
		if err != nil {
			return nil, fmt.Errorf("ошибка получения локального профиля: %w", err)
		}
		peerID = profile.PeerID
	} else {
		contact, err := queries.GetContact(contactID)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения контакта: %w", err)
		}
		peerID = contact.PeerID
	}

	// Получаем или создаём чат
	chat, err := queries.GetOrCreateChat(peerID, &contactID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения чата: %w", err)
	}

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
func (cs *ChatService) SendElementMessage(contactID int, fromPeerID string, item *models.Item) (*models.ChatMessage, error) {
	// Получаем peer_id из контакта
	var peerID string
	if contactID == 0 {
		// Локальный чат
		profile, err := queries.GetLocalProfile()
		if err != nil {
			return nil, fmt.Errorf("ошибка получения локального профиля: %w", err)
		}
		peerID = profile.PeerID
	} else {
		contact, err := queries.GetContact(contactID)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения контакта: %w", err)
		}
		peerID = contact.PeerID
	}

	// Получаем или создаём чат
	chat, err := queries.GetOrCreateChat(peerID, &contactID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения чата: %w", err)
	}

	// Создаём метаданные элемента
	metadata := map[string]interface{}{
		"item_id":      item.ID,
		"item_type":    string(item.Type),
		"item_title":   item.Title,
		"item_desc":    item.Description,
		"content_meta": item.ContentMeta,
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
