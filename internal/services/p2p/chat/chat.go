// Package chat содержит сервисы для обмена сообщениями между пирами.
package chat

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/services"
	"projectT/internal/services/p2p/transfer"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ProtocolID идентификатор протокола чата
const ProtocolID = "/projectt/chat/1.0.0"

// MessageType тип сообщения
type MessageType string

const (
	// MessageTypeText текстовое сообщение
	MessageTypeText MessageType = "text"
	// MessageTypeFile сообщение с файлом
	MessageTypeFile MessageType = "file"
	// MessageTypeImage сообщение с изображением
	MessageTypeImage MessageType = "image"
	// MessageTypeAck подтверждение получения
	MessageTypeAck MessageType = "ack"
)

// Message protobuf сообщение для передачи
type Message struct {
	ID          int64       `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	FromPeerID  string      `protobuf:"bytes,2,opt,name=from_peer_id,json=fromPeerId,proto3" json:"from_peer_id,omitempty"`
	Content     string      `protobuf:"bytes,3,opt,name=content,proto3" json:"content,omitempty"`
	ContentType string      `protobuf:"bytes,4,opt,name=content_type,json=contentType,proto3" json:"content_type,omitempty"`
	Metadata    string      `protobuf:"bytes,5,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Timestamp   int64       `protobuf:"varint,6,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	MessageType MessageType `protobuf:"bytes,7,opt,name=message_type,json=messageType,proto3" json:"message_type,omitempty"`
	Signature   []byte      `protobuf:"bytes,8,opt,name=signature,proto3" json:"signature,omitempty"`
	Encrypted   bool        `protobuf:"varint,9,opt,name=encrypted,proto3" json:"encrypted,omitempty"`
	Nonce       []byte      `protobuf:"bytes,10,opt,name=nonce,proto3" json:"nonce,omitempty"`
}

// QueuedMessage сообщение в очереди для оффлайн-режима
type QueuedMessage struct {
	ContactID   int
	Content     string
	ContentType string
	Metadata    string
	CreatedAt   time.Time
	RetryCount  int
}

// Service сервис для управления чатом
type Service struct {
	host          host.Host
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	messageQueue  map[peer.ID][]*QueuedMessage // очередь сообщений для оффлайн-пиров
	localPrivKey  crypto.PrivKey               // локальный приватный ключ для подписи
	localPubKey   crypto.PubKey                // локальный публичный ключ
	encryptionKey []byte                       // ключ для симметричного шифрования
	transferSvc   *transfer.Service            // сервис передачи файлов
}

// NewService создаёт сервис чата
func NewService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	// Генерируем ключ для симметричного шифрования
	encryptionKey := generateEncryptionKey()

	return &Service{
		host:          host,
		ctx:           ctx,
		cancel:        cancel,
		messageQueue:  make(map[peer.ID][]*QueuedMessage),
		localPrivKey:  privKey,
		localPubKey:   pubKey,
		encryptionKey: encryptionKey,
	}
}

// SetTransferService устанавливает сервис передачи файлов
func (cs *Service) SetTransferService(transferSvc *transfer.Service) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.transferSvc = transferSvc
}

// getTransferService возвращает сервис передачи файлов
func (cs *Service) getTransferService() *transfer.Service {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.transferSvc
}

// Start запускает сервис чата
func (cs *Service) Start() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Устанавливаем обработчик входящих сообщений
	cs.host.SetStreamHandler(ProtocolID, cs.handleChatStream)

	// Запускаем обработчик очереди сообщений
	go cs.processMessageQueue()

	log.Println("ChatService запущен")
	return nil
}

// Stop останавливает сервис чата
func (cs *Service) Stop() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.cancel()
	log.Println("ChatService остановлен")
	return nil
}

// SendMessage отправляет сообщение пиру
func (cs *Service) SendMessage(ctx context.Context, peerID peer.ID, content, contentType, metadata string) error {
	cs.mu.RLock()
	host := cs.host
	cs.mu.RUnlock()

	if host == nil {
		return errors.New("хост не инициализирован")
	}

	// Проверяем подключение
	if host.Network().Connectedness(peerID) != network.Connected {
		// Пир оффлайн - добавляем в очередь
		cs.queueMessage(peerID, content, contentType, metadata)
		return errors.New("пир оффлайн, сообщение добавлено в очередь")
	}

	// Создаём сообщение
	msg := &Message{
		FromPeerID:  host.ID().String(),
		Content:     content,
		ContentType: contentType,
		Timestamp:   time.Now().UnixNano(),
		MessageType: cs.parseMessageType(contentType),
	}

	// Добавляем метаданные если есть
	if metadata != "" {
		msg.Metadata = metadata
	}

	// Подписываем сообщение
	signature, err := cs.signMessage(msg)
	if err != nil {
		return fmt.Errorf("ошибка подписи сообщения: %w", err)
	}
	msg.Signature = signature

	// Шифруем сообщение если включено шифрование
	if cs.encryptionKey != nil {
		encrypted, nonce, err := cs.encryptMessage(msg)
		if err != nil {
			log.Printf("Предупреждение: не удалось зашифровать сообщение: %v", err)
		} else {
			msg = encrypted
			msg.Nonce = nonce
			msg.Encrypted = true
		}
	}

	// Сериализуем сообщение в JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("ошибка сериализации сообщения: %w", err)
	}

	// Создаём стрим
	stream, err := host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		// Не удалось создать стрим - добавляем в очередь
		cs.queueMessage(peerID, content, contentType, metadata)
		return fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

	// Отправляем сообщение
	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(data); err != nil {
		// Ошибка отправки - добавляем в очередь
		cs.queueMessage(peerID, content, contentType, metadata)
		return fmt.Errorf("ошибка отправки сообщения: %w", err)
	}

	if err := writer.Flush(); err != nil {
		cs.queueMessage(peerID, content, contentType, metadata)
		return fmt.Errorf("ошибка flush: %w", err)
	}

	// Читаем подтверждение
	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	ackBuf := make([]byte, 1)
	n, err := stream.Read(ackBuf)
	if err == nil && n == 1 && ackBuf[0] == 0x01 {
		// Получили подтверждение - сохраняем в БД
		return cs.saveMessage(peerID.String(), content, contentType, metadata, false)
	}

	// Подтверждение не получено - добавляем в очередь
	cs.queueMessage(peerID, content, contentType, metadata)
	return errors.New("подтверждение не получено, сообщение в очереди")
}

// handleChatStream обрабатывает входящий поток чата
func (cs *Service) handleChatStream(stream network.Stream) {
	cs.HandleChatStream(stream)
}

// HandleChatStream обрабатывает входящий поток чата (публичный метод)
func (cs *Service) HandleChatStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("Получен поток чата от: %s", remotePeer.String())

	// Проверяем, не заблокирован ли пир (если контакт есть)
	contact, err := queries.GetContactByPeerID(remotePeer.String())
	if err == nil && contact != nil && contact.IsBlocked {
		log.Printf("Пир %s заблокирован, сообщение отклонено", remotePeer)
		return
	}

	// Читаем сообщение
	reader := bufio.NewReader(stream)
	data, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("Ошибка чтения сообщения: %v", err)
		return
	}

	// Десериализуем сообщение из JSON
	msg := &Message{}
	if err := json.Unmarshal(data, msg); err != nil {
		log.Printf("Ошибка десериализации сообщения: %v", err)
		// Отправляем подтверждение чтобы не было повторной отправки
		if _, writeErr := stream.Write([]byte{0x01}); writeErr != nil {
			log.Printf("Ошибка отправки подтверждения: %v", writeErr)
		}
		return
	}

	// Расшифровываем сообщение если зашифровано
	if msg.Encrypted && cs.encryptionKey != nil && len(msg.Nonce) > 0 {
		decrypted, err := cs.decryptMessage(msg)
		if err != nil {
			log.Printf("Ошибка расшифровки сообщения: %v", err)
			// Не расшифровываем - пробуем обработать как есть
		} else {
			msg = decrypted
		}
	}

	// Сохраняем сообщение в БД
	if err := cs.saveMessage(remotePeer.String(), msg.Content, msg.ContentType, msg.Metadata, true); err != nil {
		log.Printf("Ошибка сохранения сообщения: %v", err)
		// Отправляем подтверждение чтобы не было повторной отправки
		if _, writeErr := stream.Write([]byte{0x01}); writeErr != nil {
			log.Printf("Ошибка отправки подтверждения: %v", writeErr)
		}
		return
	}

	// Отправляем подтверждение СРАЗУ после сохранения
	if _, err := stream.Write([]byte{0x01}); err != nil {
		log.Printf("Ошибка отправки подтверждения: %v", err)
	}

	// Проверяем подпись ПОСЛЕ отправки подтверждения
	if !cs.verifyMessageSignature(msg) {
		log.Printf("⚠️ Неверная подпись сообщения от %s (сообщение сохранено)", remotePeer)
	} else {
		log.Printf("Получено сообщение от %s: %s", remotePeer, msg.Content)
	}
}

// saveMessage сохраняет сообщение в базу данных
func (cs *Service) saveMessage(fromPeerID, content, contentType, metadata string, isIncoming bool) error {
	// Проверяем, есть ли профиль пира. Если нет - создаём
	profile, err := queries.GetProfileByPeerID(fromPeerID)
	if err != nil || profile == nil {
		// Профиль не найден - создаём remote профиль
		profile = &models.Profile{
			OwnerType: models.OwnerTypeRemote,
			PeerID:    fromPeerID,
			Username:  fromPeerID[:8], // Временное имя, обновится при обмене профилями
			Title:     "",
		}
		if err := queries.CreateRemoteProfile(profile); err != nil {
			if !contains(err.Error(), "UNIQUE constraint") {
				log.Printf("Предупреждение: не удалось создать профиль: %v", err)
			}
			// Профиль уже есть - получаем его
			if _, getErr := queries.GetProfileByPeerID(fromPeerID); getErr != nil {
				log.Printf("Ошибка получения профиля: %v", getErr)
			}
		} else {
			log.Printf("[Profile] ✅ Remote профиль создан для %s (username: %s)", fromPeerID[:8], profile.Username)
		}
	}

	// Получаем или создаём чат по PeerID (контакт не требуется!)
	// contact_id будет nil если контакт не добавлен
	chat, err := queries.GetOrCreateChat(fromPeerID, nil)
	if err != nil {
		return fmt.Errorf("ошибка получения чата: %w", err)
	}

	// Создаём сообщение
	message := &models.ChatMessage{
		ChatID:      chat.ID,
		FromPeerID:  fromPeerID,
		Content:     content,
		ContentType: contentType,
		Metadata:    metadata,
		IsRead:      isIncoming, // Входящие считаем прочитанными
	}

	if err := queries.CreateChatMessage(message); err != nil {
		return fmt.Errorf("ошибка сохранения сообщения: %w", err)
	}

	// Обновляем время последнего сообщения в чате
	if err := queries.UpdateChatLastMessage(chat.ID, message.SentAt); err != nil {
		log.Printf("Предупреждение: не удалось обновить время чата: %v", err)
	}

	// ОТПРАВЛЯЕМ УВЕДОМЛЕНИЕ UI для обновления в реальном времени
	chatSvc := services.GetChatService()
	if chatSvc != nil {
		// Получаем контакт если есть (для отображения имени)
		contact, _ := queries.GetContactByPeerID(fromPeerID)
		contactName := ""
		contactID := 0
		if contact != nil {
			contactName = contact.Username
			contactID = contact.ID
		}
		chatSvc.NotifyNewMessage(contactID, contactName, message, isIncoming)
	}

	log.Printf("Сообщение сохранено в БД (ID: %d)", message.ID)
	return nil
}

// queueMessage добавляет сообщение в очередь для оффлайн-пира
func (cs *Service) queueMessage(peerID peer.ID, content, contentType, metadata string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	queue := cs.messageQueue[peerID]
	queue = append(queue, &QueuedMessage{
		Content:     content,
		ContentType: contentType,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
		RetryCount:  0,
	})

	cs.messageQueue[peerID] = queue
	log.Printf("Сообщение добавлено в очередь для %s (размер очереди: %d)", peerID, len(queue))
}

// processMessageQueue обрабатывает очередь сообщений
func (cs *Service) processMessageQueue() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			cs.retryQueuedMessages()
		}
	}
}

// retryQueuedMessages пытается повторно отправить сообщения из очереди
func (cs *Service) retryQueuedMessages() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for peerID, queue := range cs.messageQueue {
		if len(queue) == 0 {
			continue
		}

		// Проверяем подключение
		if cs.host.Network().Connectedness(peerID) != network.Connected {
			continue
		}

		// Пир подключён - пытаемся отправить сообщения
		var failed []*QueuedMessage
		for _, msg := range queue {
			if msg.RetryCount >= 5 {
				log.Printf("Превышено количество попыток отправки сообщения для %s", peerID)
				continue // Пропускаем сообщение после 5 попыток
			}

			if err := cs.sendQueuedMessage(cs.ctx, peerID, msg); err != nil {
				log.Printf("Не удалось отправить сообщение из очереди: %v", err)
				msg.RetryCount++
				failed = append(failed, msg)
			}
		}

		cs.messageQueue[peerID] = failed
	}
}

// sendQueuedMessage отправляет сообщение из очереди
func (cs *Service) sendQueuedMessage(ctx context.Context, peerID peer.ID, msg *QueuedMessage) error {
	// Создаём сообщение
	chatMsg := &Message{
		FromPeerID:  cs.host.ID().String(),
		Content:     msg.Content,
		ContentType: msg.ContentType,
		Timestamp:   time.Now().UnixNano(),
		MessageType: cs.parseMessageType(msg.ContentType),
	}

	if msg.Metadata != "" {
		chatMsg.Metadata = msg.Metadata
	}

	// Подписываем сообщение
	signature, err := cs.signMessage(chatMsg)
	if err != nil {
		return fmt.Errorf("ошибка подписи: %w", err)
	}
	chatMsg.Signature = signature

	// Сериализуем в JSON
	data, err := json.Marshal(chatMsg)
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	// Создаём стрим
	stream, err := cs.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return err
	}
	defer stream.Close()

	// Отправляем
	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(data); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	// Читаем подтверждение
	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	ackBuf := make([]byte, 1)
	n, err := stream.Read(ackBuf)
	if err == nil && n == 1 && ackBuf[0] == 0x01 {
		// Сохраняем в БД
		return cs.saveMessage(peerID.String(), msg.Content, msg.ContentType, msg.Metadata, false)
	}

	return errors.New("подтверждение не получено")
}

// signMessage подписывает сообщение приватным ключом
func (cs *Service) signMessage(msg *Message) ([]byte, error) {
	if cs.localPrivKey == nil {
		return nil, errors.New("приватный ключ не установлен")
	}

	// Создаём данные для подписи
	data := fmt.Sprintf("%s:%s:%d", msg.FromPeerID, msg.Content, msg.Timestamp)

	// Подписываем
	signature, err := cs.localPrivKey.Sign([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("ошибка подписи: %w", err)
	}

	return signature, nil
}

// verifyMessageSignature проверяет подпись сообщения
func (cs *Service) verifyMessageSignature(msg *Message) bool {
	if len(msg.Signature) == 0 {
		return false
	}

	// Получаем публичный ключ отправителя
	peerID, err := peer.Decode(msg.FromPeerID)
	if err != nil {
		log.Printf("Ошибка декодирования PeerID: %v", err)
		return false
	}

	pubKey := cs.host.Peerstore().PubKey(peerID)
	if pubKey == nil {
		log.Printf("Публичный ключ не найден для %s", peerID)
		return false
	}

	// Создаём данные для проверки
	data := fmt.Sprintf("%s:%s:%d", msg.FromPeerID, msg.Content, msg.Timestamp)

	// Проверяем подпись
	valid, err := pubKey.Verify([]byte(data), msg.Signature)
	if err != nil {
		log.Printf("Ошибка проверки подписи: %v", err)
		return false
	}

	return valid
}

// encryptMessage шифрует сообщение с использованием симметричного ключа
func (cs *Service) encryptMessage(msg *Message) (*Message, []byte, error) {
	if cs.encryptionKey == nil {
		return msg, nil, errors.New("ключ шифрования не инициализирован")
	}

	// Сериализуем сообщение в JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, nil, err
	}

	// Генерируем nonce
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("ошибка генерации nonce: %w", err)
	}

	// Шифруем XOR с ключом и nonce (упрощённое шифрование)
	encrypted := make([]byte, len(data))
	for i, b := range data {
		encrypted[i] = b ^ cs.encryptionKey[i%len(cs.encryptionKey)] ^ nonce[i%len(nonce)]
	}

	// Создаём новое сообщение с зашифрованным контентом
	encryptedMsg := &Message{
		FromPeerID:  msg.FromPeerID,
		ContentType: msg.ContentType,
		Timestamp:   msg.Timestamp,
		MessageType: msg.MessageType,
		Content:     base64.StdEncoding.EncodeToString(encrypted),
		Encrypted:   true,
	}

	return encryptedMsg, nonce, nil
}

// decryptMessage расшифровывает сообщение
func (cs *Service) decryptMessage(msg *Message) (*Message, error) {
	if cs.encryptionKey == nil || len(msg.Nonce) == 0 {
		return nil, errors.New("ключ шифрования не инициализирован или nonce отсутствует")
	}

	// Декодируем зашифрованные данные
	encrypted, err := base64.StdEncoding.DecodeString(msg.Content)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования: %w", err)
	}

	// Расшифровываем
	decrypted := make([]byte, len(encrypted))
	for i, b := range encrypted {
		decrypted[i] = b ^ cs.encryptionKey[i%len(cs.encryptionKey)] ^ msg.Nonce[i%len(msg.Nonce)]
	}

	// Десериализуем сообщение из JSON
	originalMsg := &Message{}
	if err := json.Unmarshal(decrypted, originalMsg); err != nil {
		return nil, fmt.Errorf("ошибка десериализации: %w", err)
	}

	return originalMsg, nil
}

// parseMessageType определяет тип сообщения по content type
func (cs *Service) parseMessageType(contentType string) MessageType {
	switch contentType {
	case "text", "":
		return MessageTypeText
	case "file", "application/octet-stream":
		return MessageTypeFile
	case "image", "image/png", "image/jpeg", "image/gif":
		return MessageTypeImage
	default:
		return MessageTypeText
	}
}

// GetQueuedMessagesCount возвращает количество сообщений в очереди для пира
func (cs *Service) GetQueuedMessagesCount(peerID peer.ID) int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	queue := cs.messageQueue[peerID]
	return len(queue)
}

// ClearQueuedMessages очищает очередь сообщений для пира
func (cs *Service) ClearQueuedMessages(peerID peer.ID) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.messageQueue, peerID)
	log.Printf("Очередь сообщений очищена для %s", peerID)
}

// GetMessagesForContact получает сообщения для контакта
func (cs *Service) GetMessagesForContact(contactID int, limit, offset int) ([]*models.ChatMessage, error) {
	return queries.GetMessagesForContact(contactID, limit, offset)
}

// GetUnreadMessagesCount получает количество непрочитанных сообщений
func (cs *Service) GetUnreadMessagesCount(contactID int) (int, error) {
	return queries.GetUnreadMessagesCount(contactID)
}

// MarkMessageAsRead помечает сообщение как прочитанное
func (cs *Service) MarkMessageAsRead(id int) error {
	return queries.MarkMessageAsRead(id)
}

// MarkAllMessagesAsRead помечает все сообщения для контакта как прочитанные
func (cs *Service) MarkAllMessagesAsRead(contactID int) error {
	return queries.MarkAllMessagesAsRead(contactID)
}

// DeleteMessage удаляет сообщение
func (cs *Service) DeleteMessage(id int) error {
	return queries.DeleteChatMessage(id)
}

// DeleteMessagesForContact удаляет все сообщения для контакта
func (cs *Service) DeleteMessagesForContact(contactID int) error {
	return queries.DeleteMessagesForContact(contactID)
}

// generateEncryptionKey генерирует ключ для симметричного шифрования
func generateEncryptionKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		// Fallback к детерминированному ключу в случае ошибки
		return []byte("projectt-chat-encryption-key-32")
	}
	return key
}

// contains проверяет, содержит ли строка подстроку
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

// findSubstring ищет подстроку в строке
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SendFileMessage отправляет сообщение с файлом
func (cs *Service) SendFileMessage(ctx context.Context, peerID peer.ID, filePath, fileName, mimeType string) error {
	transferSvc := cs.getTransferService()

	// Если transfer service доступен - используем реальную передачу файла
	if transferSvc != nil {
		// Отправляем файл через transfer service
		transferID, err := transferSvc.SendFile(ctx, peerID, filePath, fileName, mimeType, transfer.TransferTypeFile)
		if err != nil {
			return fmt.Errorf("ошибка передачи файла: %w", err)
		}

		// Создаём метаданные с transfer_id
		metadata := map[string]string{
			"file_name":   fileName,
			"file_path":   filePath,
			"mime_type":   mimeType,
			"transfer_id": transferID,
		}

		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("ошибка сериализации метаданных: %w", err)
		}

		content := fmt.Sprintf("Файл: %s", fileName)
		return cs.SendMessage(ctx, peerID, content, "file", string(metadataJSON))
	}

	// Fallback: отправляем только метаданные (без реальной передачи)
	metadata := map[string]string{
		"file_name": fileName,
		"file_path": filePath,
		"mime_type": mimeType,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	content := fmt.Sprintf("Файл: %s", fileName)
	return cs.SendMessage(ctx, peerID, content, "file", string(metadataJSON))
}

// SendImageMessage отправляет сообщение с изображением
func (cs *Service) SendImageMessage(ctx context.Context, peerID peer.ID, imagePath, imageName string) error {
	transferSvc := cs.getTransferService()

	// Если transfer service доступен - используем реальную передачу изображения
	if transferSvc != nil {
		// Отправляем изображение через transfer service
		transferID, err := transferSvc.SendImage(ctx, peerID, imagePath, imageName)
		if err != nil {
			return fmt.Errorf("ошибка передачи изображения: %w", err)
		}

		// Создаём метаданные с transfer_id
		metadata := map[string]string{
			"image_name":  imageName,
			"image_path":  imagePath,
			"transfer_id": transferID,
		}

		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("ошибка сериализации метаданных: %w", err)
		}

		content := fmt.Sprintf("Изображение: %s", imageName)
		return cs.SendMessage(ctx, peerID, content, "image", string(metadataJSON))
	}

	// Fallback: отправляем только метаданные (без реальной передачи)
	metadata := map[string]string{
		"image_name": imageName,
		"image_path": imagePath,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	content := fmt.Sprintf("Изображение: %s", imageName)
	return cs.SendMessage(ctx, peerID, content, "image", string(metadataJSON))
}

// QueueMessage добавляет сообщение в очередь для оффлайн-пира (публичный метод для тестов)
func (cs *Service) QueueMessage(peerID peer.ID, content, contentType, metadata string) {
	cs.queueMessage(peerID, content, contentType, metadata)
}

// ParseMessageType определяет тип сообщения по content type (публичный метод для тестов)
func (cs *Service) ParseMessageType(contentType string) MessageType {
	return cs.parseMessageType(contentType)
}

// VerifyMessageSignature проверяет подпись сообщения (публичный метод для тестов)
func (cs *Service) VerifyMessageSignature(msg *Message) bool {
	return cs.verifyMessageSignature(msg)
}
