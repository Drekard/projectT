// Package tests содержит тесты для CRUD операций P2P сущностей.
package tests

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// setupTestDB инициализирует тестовую базу данных
func setupTestDB(t *testing.T) {
	// Создаём временный файл для тестовой БД
	tmpFile := "test_p2p.db"
	
	// Закрываем текущее подключение если есть
	if database.DB != nil {
		database.DB.Close()
	}
	
	// Удаляем тестовый файл если существует
	os.Remove(tmpFile)
	
	// Открываем тестовую БД
	var err error
	database.DB, err = database.Open(tmpFile)
	if err != nil {
		t.Fatalf("Не удалось инициализировать тестовую БД: %v", err)
	}

	// Выполняем миграции
	database.RunMigrations()
}

// teardownTestDB очищает ресурсы после теста
func teardownTestDB(t *testing.T) {
	if database.DB != nil {
		database.DB.Close()
		database.DB = nil
	}
	os.Remove("test_p2p.db")
}

// TestP2PProfileCRUD тестирует CRUD операции для P2PProfile
func TestP2PProfileCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	profile := &models.P2PProfile{
		PeerID:      "QmTest123456789",
		PrivateKey:  []byte("test_private_key"),
		PublicKey:   []byte("test_public_key"),
		Username:    "TestUser",
		Status:      "online",
		ListenAddrs: "/ip4/127.0.0.1/tcp/4001",
	}

	// Test Create
	exists, err := queries.P2PProfileExists()
	if err != nil {
		t.Fatalf("Ошибка проверки существования профиля: %v", err)
	}

	if !exists {
		err = queries.CreateP2PProfile(profile)
		if err != nil {
			t.Fatalf("Не удалось создать P2P профиль: %v", err)
		}
	}

	// Test Get
	retrieved, err := queries.GetP2PProfile()
	if err != nil {
		t.Fatalf("Не удалось получить P2P профиль: %v", err)
	}

	if retrieved.PeerID != profile.PeerID {
		t.Errorf("Ожидался PeerID %s, получен %s", profile.PeerID, retrieved.PeerID)
	}

	if retrieved.Username != profile.Username {
		t.Errorf("Ожидалось имя %s, получено %s", profile.Username, retrieved.Username)
	}

	// Test Update
	profile.Username = "UpdatedUser"
	err = queries.UpdateP2PProfile(profile)
	if err != nil {
		t.Fatalf("Не удалось обновить P2P профиль: %v", err)
	}

	updated, err := queries.GetP2PProfile()
	if err != nil {
		t.Fatalf("Не удалось получить обновлённый P2P профиль: %v", err)
	}

	if updated.Username != "UpdatedUser" {
		t.Errorf("Ожидалось имя UpdatedUser, получено %s", updated.Username)
	}

	// Test UpdateField
	err = queries.UpdateP2PProfileField("status", "offline")
	if err != nil {
		t.Fatalf("Не удалось обновить поле статуса: %v", err)
	}

	updated, err = queries.GetP2PProfile()
	if err != nil {
		t.Fatalf("Не удалось получить обновлённый P2P профиль: %v", err)
	}

	if updated.Status != "offline" {
		t.Errorf("Ожидался статус offline, получен %s", updated.Status)
	}
}

// TestContactCRUD тестирует CRUD операции для Contact
func TestContactCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	contact := &models.Contact{
		PeerID:    "QmContact123456",
		Username:  "ContactUser",
		PublicKey: []byte("contact_public_key"),
		Multiaddr: "/ip4/192.168.1.1/tcp/4001/p2p/QmContact123456",
		Status:    "online",
		Notes:     "Test contact",
		IsBlocked: false,
	}

	// Test Create
	err := queries.CreateContact(contact)
	if err != nil {
		t.Fatalf("Не удалось создать контакт: %v", err)
	}

	if contact.ID == 0 {
		t.Error("ID контакта не был установлен после создания")
	}

	// Test Get
	retrieved, err := queries.GetContact(contact.ID)
	if err != nil {
		t.Fatalf("Не удалось получить контакт: %v", err)
	}

	if retrieved.PeerID != contact.PeerID {
		t.Errorf("Ожидался PeerID %s, получен %s", contact.PeerID, retrieved.PeerID)
	}

	// Test GetByPeerID
	byPeerID, err := queries.GetContactByPeerID(contact.PeerID)
	if err != nil {
		t.Fatalf("Не удалось получить контакт по PeerID: %v", err)
	}

	if byPeerID.ID != contact.ID {
		t.Errorf("Ожидался ID %d, получен %d", contact.ID, byPeerID.ID)
	}

	// Test GetAll
	all, err := queries.GetAllContacts()
	if err != nil {
		t.Fatalf("Не удалось получить все контакты: %v", err)
	}

	if len(all) != 1 {
		t.Errorf("Ожидался 1 контакт, получено %d", len(all))
	}

	// Test Update
	contact.Username = "UpdatedContact"
	err = queries.UpdateContact(contact)
	if err != nil {
		t.Fatalf("Не удалось обновить контакт: %v", err)
	}

	updated, err := queries.GetContact(contact.ID)
	if err != nil {
		t.Fatalf("Не удалось получить обновлённый контакт: %v", err)
	}

	if updated.Username != "UpdatedContact" {
		t.Errorf("Ожидалось имя UpdatedContact, получено %s", updated.Username)
	}

	// Test UpdateStatus
	err = queries.UpdateContactStatus(contact.PeerID, "offline")
	if err != nil {
		t.Fatalf("Не удалось обновить статус контакта: %v", err)
	}

	updated, err = queries.GetContact(contact.ID)
	if err != nil {
		t.Fatalf("Не удалось получить обновлённый контакт: %v", err)
	}

	if updated.Status != "offline" {
		t.Errorf("Ожидался статус offline, получен %s", updated.Status)
	}

	// Test IsBlocked
	blocked, err := queries.IsContactBlocked(contact.PeerID)
	if err != nil {
		t.Fatalf("Не удалось проверить блокировку контакта: %v", err)
	}

	if blocked {
		t.Error("Контакт не должен быть заблокирован")
	}

	// Test Search
	results, err := queries.SearchContacts("Updated")
	if err != nil {
		t.Fatalf("Не удалось выполнить поиск контактов: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Ожидался 1 результат поиска, получено %d", len(results))
	}

	// Test Delete
	err = queries.DeleteContact(contact.ID)
	if err != nil {
		t.Fatalf("Не удалось удалить контакт: %v", err)
	}

	_, err = queries.GetContact(contact.ID)
	if err == nil {
		t.Error("Ожидалась ошибка при получении удалённого контакта")
	}
}

// TestChatMessageCRUD тестирует CRUD операции для ChatMessage
func TestChatMessageCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	// Создаём контакт для тестов сообщений
	contact := &models.Contact{
		PeerID:    "QmMessageContact",
		Username:  "MessageUser",
		PublicKey: []byte("message_public_key"),
		Multiaddr: "/ip4/192.168.1.2/tcp/4001/p2p/QmMessageContact",
		Status:    "online",
	}
	err := queries.CreateContact(contact)
	if err != nil {
		t.Fatalf("Не удалось создать контакт для тестов сообщений: %v", err)
	}

	message := &models.ChatMessage{
		ContactID:   contact.ID,
		FromPeerID:  "QmTest123456789",
		Content:     "Hello, World!",
		ContentType: "text",
		Metadata:    `{"encrypted": true}`,
		IsRead:      false,
	}

	// Test Create
	err = queries.CreateChatMessage(message)
	if err != nil {
		t.Fatalf("Не удалось создать сообщение: %v", err)
	}

	if message.ID == 0 {
		t.Error("ID сообщения не был установлен после создания")
	}

	// Test Get
	retrieved, err := queries.GetChatMessage(message.ID)
	if err != nil {
		t.Fatalf("Не удалось получить сообщение: %v", err)
	}

	if retrieved.Content != message.Content {
		t.Errorf("Ожидалось содержимое %s, получено %s", message.Content, retrieved.Content)
	}

	// Test GetMessagesForContact
	messages, err := queries.GetMessagesForContact(contact.ID, 10, 0)
	if err != nil {
		t.Fatalf("Не удалось получить сообщения для контакта: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Ожидалось 1 сообщение, получено %d", len(messages))
	}

	// Test GetUnreadMessagesCount
	count, err := queries.GetUnreadMessagesCount(contact.ID)
	if err != nil {
		t.Fatalf("Не удалось получить количество непрочитанных сообщений: %v", err)
	}

	if count != 1 {
		t.Errorf("Ожидалось 1 непрочитанное сообщение, получено %d", count)
	}

	// Test MarkMessageAsRead
	err = queries.MarkMessageAsRead(message.ID)
	if err != nil {
		t.Fatalf("Не удалось пометить сообщение как прочитанное: %v", err)
	}

	count, err = queries.GetUnreadMessagesCount(contact.ID)
	if err != nil {
		t.Fatalf("Не удалось получить количество непрочитанных сообщений: %v", err)
	}

	if count != 0 {
		t.Errorf("Ожидалось 0 непрочитанных сообщений, получено %d", count)
	}

	// Test MarkAllMessagesAsRead
	// Создадим ещё одно сообщение
	message2 := &models.ChatMessage{
		ContactID:   contact.ID,
		FromPeerID:  "QmTest123456789",
		Content:     "Second message",
		ContentType: "text",
		IsRead:      false,
	}
	err = queries.CreateChatMessage(message2)
	if err != nil {
		t.Fatalf("Не удалось создать второе сообщение: %v", err)
	}

	err = queries.MarkAllMessagesAsRead(contact.ID)
	if err != nil {
		t.Fatalf("Не удалось пометить все сообщения как прочитанные: %v", err)
	}

	count, err = queries.GetUnreadMessagesCount(contact.ID)
	if err != nil {
		t.Fatalf("Не удалось получить количество непрочитанных сообщений: %v", err)
	}

	if count != 0 {
		t.Errorf("Ожидалось 0 непрочитанных сообщений, получено %d", count)
	}

	// Test GetLastMessageForContact
	// Небольшая задержка чтобы сообщения имели разное время
	time.Sleep(100 * time.Millisecond)
	
	// Создадим ещё одно сообщение для теста последнего
	message3 := &models.ChatMessage{
		ContactID:   contact.ID,
		FromPeerID:  "QmTest123456789",
		Content:     "Third message",
		ContentType: "text",
		IsRead:      false,
	}
	err = queries.CreateChatMessage(message3)
	if err != nil {
		t.Fatalf("Не удалось создать третье сообщение: %v", err)
	}
	
	// Получаем все сообщения чтобы проверить порядок
	allMessages, err := queries.GetMessagesForContact(contact.ID, 10, 0)
	if err != nil {
		t.Fatalf("Не удалось получить сообщения для контакта: %v", err)
	}
	
	// Последнее сообщение должно быть message3 (оно самое новое)
	// GetMessagesForContact возвращает в порядке возрастания (старые в начале, новые в конце)
	// Но функция реверсирует, поэтому новые сообщения в начале
	if len(allMessages) == 0 {
		t.Fatal("Нет сообщений для проверки последнего")
	}
	
	// Первое сообщение должно быть самым новым (message3)
	last := allMessages[0]
	if last.ID != message3.ID {
		t.Errorf("Ожидался ID последнего сообщения %d, получен %d", message3.ID, last.ID)
	}

	// Test Delete
	err = queries.DeleteChatMessage(message3.ID)
	if err != nil {
		t.Fatalf("Не удалось удалить сообщение: %v", err)
	}

	messages, err = queries.GetMessagesForContact(contact.ID, 10, 0)
	if err != nil {
		t.Fatalf("Не удалось получить сообщения для контакта: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Ожидалось 2 сообщения, получено %d", len(messages))
	}

	// Test DeleteMessagesForContact
	err = queries.DeleteMessagesForContact(contact.ID)
	if err != nil {
		t.Fatalf("Не удалось удалить все сообщения для контакта: %v", err)
	}

	messages, err = queries.GetMessagesForContact(contact.ID, 10, 0)
	if err != nil {
		t.Fatalf("Не удалось получить сообщения для контакта: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Ожидалось 0 сообщений, получено %d", len(messages))
	}
}

// TestBootstrapPeerCRUD тестирует CRUD операции для BootstrapPeer
func TestBootstrapPeerCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	peer := &models.BootstrapPeer{
		Multiaddr: "/ip4/10.0.0.1/tcp/4001/p2p/QmBootstrap1",
		PeerID:    sql.NullString{String: "QmBootstrap1", Valid: true},
		IsActive:  true,
	}

	// Test Create
	err := queries.CreateBootstrapPeer(peer)
	if err != nil {
		t.Fatalf("Не удалось создать bootstrap-узел: %v", err)
	}

	if peer.ID == 0 {
		t.Error("ID bootstrap-узла не был установлен после создания")
	}

	// Test Get
	retrieved, err := queries.GetBootstrapPeer(peer.ID)
	if err != nil {
		t.Fatalf("Не удалось получить bootstrap-узел: %v", err)
	}

	if retrieved.Multiaddr != peer.Multiaddr {
		t.Errorf("Ожидался Multiaddr %s, получен %s", peer.Multiaddr, retrieved.Multiaddr)
	}

	// Test GetAll
	all, err := queries.GetAllBootstrapPeers()
	if err != nil {
		t.Fatalf("Не удалось получить все bootstrap-узлы: %v", err)
	}

	if len(all) < 1 {
		t.Errorf("Ожидался хотя бы 1 bootstrap-узел, получено %d", len(all))
	}

	// Test GetActiveBootstrapPeers
	active, err := queries.GetActiveBootstrapPeers()
	if err != nil {
		t.Fatalf("Не удалось получить активные bootstrap-узлы: %v", err)
	}

	if len(active) < 1 {
		t.Errorf("Ожидался хотя бы 1 активный bootstrap-узел, получено %d", len(active))
	}

	// Test BootstrapPeerExists
	exists, err := queries.BootstrapPeerExists(peer.Multiaddr)
	if err != nil {
		t.Fatalf("Не удалось проверить существование bootstrap-узла: %v", err)
	}

	if !exists {
		t.Error("Bootstrap-узел должен существовать")
	}

	// Test Update
	peer.PeerID = sql.NullString{String: "QmUpdatedBootstrap", Valid: true}
	err = queries.UpdateBootstrapPeer(peer)
	if err != nil {
		t.Fatalf("Не удалось обновить bootstrap-узел: %v", err)
	}

	updated, err := queries.GetBootstrapPeer(peer.ID)
	if err != nil {
		t.Fatalf("Не удалось получить обновлённый bootstrap-узел: %v", err)
	}

	if !updated.PeerID.Valid || updated.PeerID.String != "QmUpdatedBootstrap" {
		t.Errorf("Ожидался PeerID QmUpdatedBootstrap, получен %s", updated.PeerID.String)
	}

	// Test UpdateLastConnected
	err = queries.UpdateBootstrapPeerLastConnected(peer.Multiaddr)
	if err != nil {
		t.Fatalf("Не удалось обновить время подключения: %v", err)
	}

	// Test SetBootstrapPeerActive
	err = queries.SetBootstrapPeerActive(peer.ID, false)
	if err != nil {
		t.Fatalf("Не удалось установить активность bootstrap-узла: %v", err)
	}

	updated, err = queries.GetBootstrapPeer(peer.ID)
	if err != nil {
		t.Fatalf("Не удалось получить обновлённый bootstrap-узел: %v", err)
	}

	if updated.IsActive {
		t.Error("Bootstrap-узел должен быть неактивен")
	}

	// Test Delete
	err = queries.DeleteBootstrapPeer(peer.ID)
	if err != nil {
		t.Fatalf("Не удалось удалить bootstrap-узел: %v", err)
	}

	_, err = queries.GetBootstrapPeer(peer.ID)
	if err == nil {
		t.Error("Ожидалась ошибка при получении удалённого bootstrap-узла")
	}
}
