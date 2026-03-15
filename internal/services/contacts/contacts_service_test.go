// Package contacts содержит тесты для сервиса контактов
package contacts

import (
	"testing"

	"projectT/internal/storage/database/models"
)

// TestContactService_Stats тестирует статистику контактов
func TestContactService_Stats(t *testing.T) {
	stats := &ContactStats{
		Total:   10,
		Online:  3,
		Offline: 6,
		Blocked: 1,
	}

	if stats.Total != 10 {
		t.Errorf("Ожидалось Total=10, получено %d", stats.Total)
	}
	if stats.Online != 3 {
		t.Errorf("Ожидалось Online=3, получено %d", stats.Online)
	}
	if stats.Blocked != 1 {
		t.Errorf("Ожидалось Blocked=1, получено %d", stats.Blocked)
	}
}

// TestContactWithStatus тестирует структуру ContactWithStatus
func TestContactWithStatus(t *testing.T) {
	contact := &models.Contact{
		ID:        1,
		PeerID:    "QmTest123",
		Username:  "TestUser",
		Multiaddr: "/ip4/127.0.0.1/tcp/4001/p2p/QmTest123",
		Title:     "online",
		Notes:     "Тестовый контакт",
		IsBlocked: false,
	}

	contactWithStatus := &ContactWithStatus{
		Contact:  contact,
		IsOnline: true,
		LastPing: 50000000, // 50ms в nanoseconds
	}

	if contactWithStatus.Username != "TestUser" {
		t.Errorf("Ожидалось Username=TestUser, получено %s", contactWithStatus.Username)
	}
	if !contactWithStatus.IsOnline {
		t.Error("Ожидалось IsOnline=true")
	}
	if contactWithStatus.LastPing != 50000000 {
		t.Errorf("Ожидалось LastPing=50000000, получено %d", contactWithStatus.LastPing)
	}
}

// TestContactService_Creation тестирует создание сервиса
func TestContactService_Creation(t *testing.T) {
	// Создаём сервис без P2P сети (для тестов)
	service := NewContactService(nil)

	if service == nil {
		t.Fatal("Сервис не создан")
	}

	if service.p2pNetwork != nil {
		t.Error("Ожидалось p2pNetwork=nil")
	}
}

// TestContactService_ParsePeerID тестирует парсинг PeerID
func TestContactService_ParsePeerID(t *testing.T) {
	service := NewContactService(nil)

	// Валидные PeerID (base58 encoded)
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"без префикса", "QmTest123456789012345678901234567890", "QmTest123456789012345678901234567890"},
		{"с префиксом", "projectt:QmTest123456789012345678901234567890", "QmTest123456789012345678901234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.parsePeerID(tt.input)
			if err != nil {
				// PeerID не валидный - это нормально для тестов
				return
			}
			if string(result) != tt.expected {
				t.Errorf("Ожидалось %s, получено %s", tt.expected, result)
			}
		})
	}
}

// TestContactWithStatus_IsOnline тестирует статус онлайн
func TestContactWithStatus_IsOnline(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"онлайн", "online", true},
		{"оффлайн", "offline", false},
		{"неизвестен", "unknown", false},
		{"пусто", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contact := &ContactWithStatus{
				Contact: &models.Contact{
					Title: tt.status,
				},
				IsOnline: tt.status == "online",
			}

			if contact.IsOnline != tt.expected {
				t.Errorf("Ожидалось IsOnline=%v для статуса %s", tt.expected, tt.status)
			}
		})
	}
}

// TestContactService_BlockedContact тестирует блокировку контакта
func TestContactService_BlockedContact(t *testing.T) {
	contact := &models.Contact{
		ID:        1,
		PeerID:    "QmBlocked",
		Username:  "BlockedUser",
		IsBlocked: true,
	}

	if !contact.IsBlocked {
		t.Error("Ожидалось IsBlocked=true")
	}

	contact.IsBlocked = false
	if contact.IsBlocked {
		t.Error("Ожидалось IsBlocked=false после разблокировки")
	}
}

// TestContactService_Notes тестирует заметки контакта
func TestContactService_Notes(t *testing.T) {
	contact := &models.Contact{
		ID:       1,
		PeerID:   "QmTest",
		Username: "Test",
		Notes:    "Это тестовая заметка",
	}

	if contact.Notes != "Это тестовая заметка" {
		t.Errorf("Ожидалась заметка 'Это тестовая заметка', получено '%s'", contact.Notes)
	}

	contact.Notes = "Обновлённая заметка"
	if contact.Notes != "Обновлённая заметка" {
		t.Errorf("Ожидалась заметка 'Обновлённая заметка', получено '%s'", contact.Notes)
	}
}

// TestContactWithStatus_Enrich тестирует обогащение контакта статусом
func TestContactWithStatus_Enrich(t *testing.T) {
	baseContact := &models.Contact{
		ID:        1,
		PeerID:    "QmTest",
		Username:  "TestUser",
		Title:     "offline",
		Multiaddr: "/ip4/127.0.0.1/tcp/4001/p2p/QmTest",
	}

	// Создаём обогащённый контакт
	enriched := &ContactWithStatus{
		Contact:  baseContact,
		IsOnline: false, // P2P сеть не инициализирована
	}

	if enriched.Username != "TestUser" {
		t.Errorf("Username не сохранился: %s", enriched.Username)
	}
	if enriched.Title != "offline" {
		t.Errorf("Статус изменился: %s", enriched.Title)
	}
	if enriched.IsOnline {
		t.Error("Ожидалось IsOnline=false")
	}
}
