// Package p2p содержит сервисы для P2P связи на базе libp2p.
package p2p

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

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

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

// ChatMessage protobuf сообщение для передачи
type ChatMessage struct {
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

// ChatService сервис для управления чатом
type ChatService struct {
	host          host.Host
	config        *P2PConfig
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	messageQueue  map[peer.ID][]*QueuedMessage // очередь сообщений для оффлайн-пиров
	localPrivKey  crypto.PrivKey               // локальный приватный ключ для подписи
	localPubKey   crypto.PubKey                // локальный публичный ключ
	encryptionKey []byte                       // ключ для симметричного шифрования
}

// NewChatService создаёт сервис чата
func NewChatService(host host.Host, config *P2PConfig, privKey crypto.PrivKey, pubKey crypto.PubKey) *ChatService {
	ctx, cancel := context.WithCancel(context.Background())

	// Генерируем ключ для симметричного шифрования
	encryptionKey := generateEncryptionKey()

	return &ChatService{
		host:          host,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		messageQueue:  make(map[peer.ID][]*QueuedMessage),
		localPrivKey:  privKey,
		localPubKey:   pubKey,
		encryptionKey: encryptionKey,
	}
}

// Start запускает сервис чата
func (cs *ChatService) Start() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Устанавливаем обработчик входящих сообщений
	cs.host.SetStreamHandler(ChatProtocolID, cs.handleChatStream)

	// Запускаем обработчик очереди сообщений
	go cs.processMessageQueue()

	log.Println("ChatService запущен")
	return nil
}

// Stop останавливает сервис чата
func (cs *ChatService) Stop() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.cancel()
	log.Println("ChatService остановлен")
	return nil
}

// SendMessage отправляет сообщение пиру
func (cs *ChatService) SendMessage(ctx context.Context, peerID peer.ID, content, contentType, metadata string) error {
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
	msg := &ChatMessage{
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
	stream, err := host.NewStream(ctx, peerID, ChatProtocolID)
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
func (cs *ChatService) handleChatStream(stream network.Stream) {
	cs.HandleChatStream(stream)
}

// HandleChatStream обрабатывает входящий поток чата (публичный метод)
func (cs *ChatService) HandleChatStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("Получен поток чата от: %s", remotePeer.String())

	// Проверяем, не заблокирован ли пир
	contact, err := queries.GetContactByPeerID(remotePeer.String())
	if err != nil {
		log.Printf("Ошибка получения контакта: %v", err)
		return
	}

	if contact != nil && contact.IsBlocked {
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
	msg := &ChatMessage{}
	if err := json.Unmarshal(data, msg); err != nil {
		log.Printf("Ошибка десериализации сообщения: %v", err)
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

	// Проверяем подпись
	if !cs.verifyMessageSignature(msg) {
		log.Printf("Неверная подпись сообщения от %s", remotePeer)
		return
	}

	// Сохраняем сообщение в БД
	if err := cs.saveMessage(remotePeer.String(), msg.Content, msg.ContentType, msg.Metadata, true); err != nil {
		log.Printf("Ошибка сохранения сообщения: %v", err)
		return
	}

	// Отправляем подтверждение
	if _, err := stream.Write([]byte{0x01}); err != nil {
		log.Printf("Ошибка отправки подтверждения: %v", err)
	}

	log.Printf("Получено сообщение от %s: %s", remotePeer, msg.Content)
}

// saveMessage сохраняет сообщение в базу данных
func (cs *ChatService) saveMessage(fromPeerID, content, contentType, metadata string, isIncoming bool) error {
	// Получаем контакт по PeerID
	contact, err := queries.GetContactByPeerID(fromPeerID)
	if err != nil {
		// Контакт не найден - создаём временный
		contact = &models.Contact{
			PeerID:   fromPeerID,
			Username: fromPeerID[:8],
		}
		if err := queries.CreateContact(contact); err != nil && !contains(err.Error(), "UNIQUE constraint") {
			return fmt.Errorf("ошибка создания контакта: %w", err)
		}
		// Перечитываем контакт
		contact, err = queries.GetContactByPeerID(fromPeerID)
		if err != nil {
			return fmt.Errorf("ошибка получения контакта: %w", err)
		}
	}

	// Создаём сообщение
	message := &models.ChatMessage{
		ContactID:   contact.ID,
		FromPeerID:  fromPeerID,
		Content:     content,
		ContentType: contentType,
		Metadata:    metadata,
		IsRead:      isIncoming, // Входящие считаем прочитанными
	}

	if err := queries.CreateChatMessage(message); err != nil {
		return fmt.Errorf("ошибка сохранения сообщения: %w", err)
	}

	log.Printf("Сообщение сохранено в БД (ID: %d)", message.ID)
	return nil
}

// queueMessage добавляет сообщение в очередь для оффлайн-пира
func (cs *ChatService) queueMessage(peerID peer.ID, content, contentType, metadata string) {
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
func (cs *ChatService) processMessageQueue() {
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
func (cs *ChatService) retryQueuedMessages() {
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
func (cs *ChatService) sendQueuedMessage(ctx context.Context, peerID peer.ID, msg *QueuedMessage) error {
	// Создаём сообщение
	chatMsg := &ChatMessage{
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
	stream, err := cs.host.NewStream(ctx, peerID, ChatProtocolID)
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
func (cs *ChatService) signMessage(msg *ChatMessage) ([]byte, error) {
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
func (cs *ChatService) verifyMessageSignature(msg *ChatMessage) bool {
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
func (cs *ChatService) encryptMessage(msg *ChatMessage) (*ChatMessage, []byte, error) {
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
	encryptedMsg := &ChatMessage{
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
func (cs *ChatService) decryptMessage(msg *ChatMessage) (*ChatMessage, error) {
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
	originalMsg := &ChatMessage{}
	if err := json.Unmarshal(decrypted, originalMsg); err != nil {
		return nil, fmt.Errorf("ошибка десериализации: %w", err)
	}

	return originalMsg, nil
}

// parseMessageType определяет тип сообщения по content type
func (cs *ChatService) parseMessageType(contentType string) MessageType {
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
func (cs *ChatService) GetQueuedMessagesCount(peerID peer.ID) int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	queue := cs.messageQueue[peerID]
	return len(queue)
}

// ClearQueuedMessages очищает очередь сообщений для пира
func (cs *ChatService) ClearQueuedMessages(peerID peer.ID) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.messageQueue, peerID)
	log.Printf("Очередь сообщений очищена для %s", peerID)
}

// GetMessagesForContact получает сообщения для контакта
func (cs *ChatService) GetMessagesForContact(contactID int, limit, offset int) ([]*models.ChatMessage, error) {
	return queries.GetMessagesForContact(contactID, limit, offset)
}

// GetUnreadMessagesCount получает количество непрочитанных сообщений
func (cs *ChatService) GetUnreadMessagesCount(contactID int) (int, error) {
	return queries.GetUnreadMessagesCount(contactID)
}

// MarkMessageAsRead помечает сообщение как прочитанное
func (cs *ChatService) MarkMessageAsRead(id int) error {
	return queries.MarkMessageAsRead(id)
}

// MarkAllMessagesAsRead помечает все сообщения для контакта как прочитанные
func (cs *ChatService) MarkAllMessagesAsRead(contactID int) error {
	return queries.MarkAllMessagesAsRead(contactID)
}

// DeleteMessage удаляет сообщение
func (cs *ChatService) DeleteMessage(id int) error {
	return queries.DeleteChatMessage(id)
}

// DeleteMessagesForContact удаляет все сообщения для контакта
func (cs *ChatService) DeleteMessagesForContact(contactID int) error {
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
func (cs *ChatService) SendFileMessage(ctx context.Context, peerID peer.ID, filePath, fileName, mimeType string) error {
	// В реальной реализации здесь будет отправка файла через отдельный стрим
	// Для пока просто создаём метаданные файла
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
func (cs *ChatService) SendImageMessage(ctx context.Context, peerID peer.ID, imagePath, imageName string) error {
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
