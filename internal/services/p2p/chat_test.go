// Package p2p содержит тесты для P2P сервисов.
package p2p

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/services/p2p/protocols/chat"
)

// createTestHost создаёт тестовый хост для использования в тестах
func createTestHost(t *testing.T, port int) host.Host {
	t.Helper()

	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)),
		libp2p.DisableRelay(),
		libp2p.EnableNATService(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		t.Fatalf("Ошибка создания хоста: %v", err)
	}

	t.Cleanup(func() {
		h.Close()
	})

	return h
}

// TestChatServiceCreation тестирует создание ChatService
func TestChatServiceCreation(t *testing.T) {
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	host := createTestHost(t, 0)

	pubKey := privKey.GetPublic()
	chatService := chat.NewService(host, privKey, pubKey, nil)

	if chatService == nil {
		t.Fatal("ChatService не создан")
	}
}

// TestChatServiceStartStop тестирует запуск и остановку ChatService
func TestChatServiceStartStop(t *testing.T) {
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	host := createTestHost(t, 0)

	pubKey := privKey.GetPublic()
	chatService := chat.NewService(host, privKey, pubKey, nil)

	if err := chatService.Start(); err != nil {
		t.Fatalf("Ошибка запуска ChatService: %v", err)
	}

	// Проверяем, что обработчик установлен (проверка через stream handler)
	// В libp2p нет публичного API для проверки обработчиков, поэтому просто проверяем запуск

	if err := chatService.Stop(); err != nil {
		t.Fatalf("Ошибка остановки ChatService: %v", err)
	}
}

// TestQueueMessage тестирует добавление сообщений в очередь
func TestQueueMessage(t *testing.T) {
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	host := createTestHost(t, 0)

	pubKey := privKey.GetPublic()
	chatService := chat.NewService(host, privKey, pubKey, nil)

	peerID := host.ID()

	// Добавляем сообщение в очередь
	chatService.QueueMessage(peerID, "Test message", "text", "")

	// Проверяем количество сообщений в очереди
	count := chatService.GetQueuedMessagesCount(peerID)
	if count != 1 {
		t.Errorf("Ожидается 1 сообщение в очереди, получено: %d", count)
	}

	// Добавляем ещё одно сообщение
	chatService.QueueMessage(peerID, "Another message", "text", "")

	count = chatService.GetQueuedMessagesCount(peerID)
	if count != 2 {
		t.Errorf("Ожидается 2 сообщения в очереди, получено: %d", count)
	}
}

// TestClearQueuedMessages тестирует очистку очереди сообщений
func TestClearQueuedMessages(t *testing.T) {
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	host := createTestHost(t, 0)

	pubKey := privKey.GetPublic()
	chatService := chat.NewService(host, privKey, pubKey, nil)

	peerID := host.ID()

	// Добавляем сообщения в очередь
	chatService.QueueMessage(peerID, "Message 1", "text", "")
	chatService.QueueMessage(peerID, "Message 2", "text", "")

	// Очищаем очередь
	chatService.ClearQueuedMessages(peerID)

	// Проверяем, что очередь пуста
	count := chatService.GetQueuedMessagesCount(peerID)
	if count != 0 {
		t.Errorf("Ожидается 0 сообщений в очереди, получено: %d", count)
	}
}

// TestParseMessageType тестирует определение типа сообщения
func TestParseMessageType(t *testing.T) {
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	host := createTestHost(t, 0)

	pubKey := privKey.GetPublic()
	chatService := chat.NewService(host, privKey, pubKey, nil)

	tests := []struct {
		contentType string
		expected    chat.MessageType
	}{
		{"", chat.MessageTypeText},
		{"text", chat.MessageTypeText},
		{"file", chat.MessageTypeFile},
		{"application/octet-stream", chat.MessageTypeFile},
		{"image", chat.MessageTypeImage},
		{"image/png", chat.MessageTypeImage},
		{"image/jpeg", chat.MessageTypeImage},
		{"unknown", chat.MessageTypeText},
	}

	for _, tt := range tests {
		result := chatService.ParseMessageType(tt.contentType)
		if result != tt.expected {
			t.Errorf("parseMessageType(%q) = %v, ожидается %v", tt.contentType, result, tt.expected)
		}
	}
}

// TestSignAndVerifyMessage тестирует подпись и проверку сообщений
func TestSignAndVerifyMessage(t *testing.T) {
	privKey, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	// Создаём хост с тем же приватным ключом
	host, err := libp2p.New(libp2p.Identity(privKey))
	if err != nil {
		t.Fatalf("Ошибка создания хоста: %v", err)
	}
	defer host.Close()

	chatService := chat.NewService(host, privKey, pubKey, nil)

	msg := &chat.Message{
		FromPeerID:  host.ID().String(),
		Content:     "Test message",
		Timestamp:   time.Now().UnixNano(),
		MessageType: chat.MessageTypeText,
	}

	// Создаём данные для подписи
	data := fmt.Sprintf("%s:%s:%d", msg.FromPeerID, msg.Content, msg.Timestamp)

	// Подписываем данные напрямую
	signature, err := privKey.Sign([]byte(data))
	if err != nil {
		t.Fatalf("Ошибка подписи: %v", err)
	}

	msg.Signature = signature

	// Проверяем подпись напрямую
	valid, err := pubKey.Verify([]byte(data), signature)
	if err != nil {
		t.Fatalf("Ошибка проверки подписи: %v", err)
	}
	if !valid {
		t.Fatal("Подпись не прошла проверку (прямая проверка)")
	}

	// Теперь проверяем через сервис
	if !chatService.VerifyMessageSignature(msg) {
		t.Error("Подпись не прошла проверку через сервис")
	}

	// Создаём новое сообщение с повреждённым контентом и старой подписью
	tamperedMsg := &chat.Message{
		FromPeerID:  msg.FromPeerID,
		Content:     "Tampered message", // Изменённый контент
		Timestamp:   msg.Timestamp,      // То же время
		MessageType: msg.MessageType,
		Signature:   msg.Signature, // Старая подпись
	}

	// Проверяем, что подпись не пройдёт для повреждённого сообщения
	if chatService.VerifyMessageSignature(tamperedMsg) {
		t.Error("Подпись прошла проверку для повреждённого сообщения")
	}
}

// TestSendMessageToOfflinePeer тестирует отправку сообщения оффлайн-пиру
func TestSendMessageToOfflinePeer(t *testing.T) {
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("Ошибка генерации ключей: %v", err)
	}

	host := createTestHost(t, 0)

	pubKey := privKey.GetPublic()
	chatService := chat.NewService(host, privKey, pubKey, nil)

	if err := chatService.Start(); err != nil {
		t.Fatalf("Ошибка запуска ChatService: %v", err)
	}
	defer func() { _ = chatService.Stop() }()

	// Создаём случайный PeerID (оффлайн пир)
	randomPrivKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	randomPeerID, _ := peer.IDFromPrivateKey(randomPrivKey)

	ctx := context.Background()
	err = chatService.SendMessage(ctx, randomPeerID, "Test message", "text", "")

	// Ожидаем ошибку, так как пир оффлайн
	if err == nil {
		t.Error("Ожидается ошибка при отправке оффлайн-пиру")
	}

	// Проверяем, что сообщение добавлено в очередь
	count := chatService.GetQueuedMessagesCount(randomPeerID)
	if count != 1 {
		t.Errorf("Ожидается 1 сообщение в очереди, получено: %d", count)
	}
}

// TestChatMessageSerialization тестирует сериализацию сообщений
func TestChatMessageSerialization(t *testing.T) {
	msg := &chat.Message{
		FromPeerID:  "test-peer-id",
		Content:     "Test content",
		ContentType: "text",
		Timestamp:   time.Now().UnixNano(),
		MessageType: chat.MessageTypeText,
		Metadata:    `{"key": "value"}`,
		Signature:   []byte("test-signature"),
		Encrypted:   false,
		Nonce:       []byte("test-nonce"),
	}

	// Сериализуем в JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Ошибка сериализации: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Сериализованные данные пустые")
	}

	// Десериализуем
	deserialized := &chat.Message{}
	if err := json.Unmarshal(data, deserialized); err != nil {
		t.Fatalf("Ошибка десериализации: %v", err)
	}

	// Проверяем поля
	if deserialized.FromPeerID != msg.FromPeerID {
		t.Errorf("FromPeerID: ожидается %s, получено %s", msg.FromPeerID, deserialized.FromPeerID)
	}
	if deserialized.Content != msg.Content {
		t.Errorf("Content: ожидается %s, получено %s", msg.Content, deserialized.Content)
	}
	if deserialized.ContentType != msg.ContentType {
		t.Errorf("ContentType: ожидается %s, получено %s", msg.ContentType, deserialized.ContentType)
	}
	if deserialized.MessageType != msg.MessageType {
		t.Errorf("MessageType: ожидается %v, получено %v", msg.MessageType, deserialized.MessageType)
	}
}
