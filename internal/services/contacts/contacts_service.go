// Package contacts предоставляет сервис для управления контактами
package contacts

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	p2p "projectT/internal/services/p2p"
	p2pnet "projectT/internal/services/p2p/network"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ContactService сервис для управления контактами
type ContactService struct {
	p2pNetwork *p2pnet.P2PNetwork
}

// ContactWithStatus контакт с расширенной информацией о статусе
type ContactWithStatus struct {
	*models.Contact
	IsOnline      bool          `json:"is_online"`
	ConnectionErr error         `json:"connection_err,omitempty"`
	LastPing      time.Duration `json:"last_ping,omitempty"`
}

// NewContactService создаёт сервис управления контактами
func NewContactService(p2pNetwork *p2pnet.P2PNetwork) *ContactService {
	return &ContactService{
		p2pNetwork: p2pNetwork,
	}
}

// AddContactByAddress добавляет контакт по P2P адресу
func (s *ContactService) AddContactByAddress(addrStr string, notes string) (*models.Contact, error) {
	// Парсим адрес
	addr, err := p2p.ParsePeerAddressString(addrStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга адреса: %w", err)
	}

	// Проверяем, не существует ли уже контакт с таким PeerID
	existing, err := queries.GetContactByPeerID(addr.PeerID)
	if err == nil && existing != nil {
		return nil, errors.New("контакт с таким PeerID уже существует")
	}

	// Создаём контакт
	contact := &models.Contact{
		PeerID:    addr.PeerID,
		Username:  addr.PeerID[:8], // Первые 8 символов как временное имя
		Multiaddr: addr.Multiaddr,
		Status:    "offline",
		Notes:     notes,
		IsBlocked: false,
	}

	if err := queries.CreateContact(contact); err != nil {
		return nil, fmt.Errorf("ошибка создания контакта: %w", err)
	}

	log.Printf("Добавлен контакт: %s (%s)", contact.Username, contact.PeerID)
	return contact, nil
}

// AddContactByPeerID добавляет контакт по PeerID (если уже есть в peerstore)
func (s *ContactService) AddContactByPeerID(peerID, username, multiaddr string, notes string) (*models.Contact, error) {
	// Проверяем, не существует ли уже
	existing, err := queries.GetContactByPeerID(peerID)
	if err == nil && existing != nil {
		return nil, errors.New("контакт с таким PeerID уже существует")
	}

	contact := &models.Contact{
		PeerID:    peerID,
		Username:  username,
		Multiaddr: multiaddr,
		Status:    "offline",
		Notes:     notes,
		IsBlocked: false,
	}

	if err := queries.CreateContact(contact); err != nil {
		return nil, fmt.Errorf("ошибка создания контакта: %w", err)
	}

	log.Printf("Добавлен контакт: %s (%s)", contact.Username, contact.PeerID)
	return contact, nil
}

// GetContact получает контакт по ID
func (s *ContactService) GetContact(id int) (*ContactWithStatus, error) {
	contact, err := queries.GetContact(id)
	if err != nil {
		return nil, err
	}

	return s.enrichContactWithStatus(contact), nil
}

// GetContactByPeerID получает контакт по PeerID
func (s *ContactService) GetContactByPeerID(peerID string) (*ContactWithStatus, error) {
	contact, err := queries.GetContactByPeerID(peerID)
	if err != nil {
		return nil, err
	}

	return s.enrichContactWithStatus(contact), nil
}

// GetAllContacts получает все контакты с информацией о статусе
func (s *ContactService) GetAllContacts() ([]*ContactWithStatus, error) {
	contacts, err := queries.GetAllContacts()
	if err != nil {
		return nil, err
	}

	result := make([]*ContactWithStatus, 0, len(contacts))
	for _, c := range contacts {
		result = append(result, s.enrichContactWithStatus(c))
	}

	return result, nil
}

// enrichContactWithStatus добавляет информацию о статусе подключения
func (s *ContactService) enrichContactWithStatus(contact *models.Contact) *ContactWithStatus {
	if contact == nil {
		return nil
	}

	result := &ContactWithStatus{
		Contact:  contact,
		IsOnline: contact.Status == "online",
	}

	// Проверяем реальное состояние подключения через P2P сеть
	if s.p2pNetwork != nil {
		peerID, err := s.parsePeerID(contact.PeerID)
		if err == nil {
			status := s.p2pNetwork.GetConnectionStatus(peerID)
			result.IsOnline = status == p2p.StatusConnected

			// Получаем дополнительную информацию
			info := s.p2pNetwork.GetPeerInfo(peerID)
			if info != nil {
				result.LastPing = info.LastPingLatency
			}
		}
	}

	return result
}

// parsePeerID парсит PeerID из строки
func (s *ContactService) parsePeerID(peerIDStr string) (peer.ID, error) {
	// Удаляем префикс если есть
	peerIDStr = strings.TrimPrefix(peerIDStr, "projectt:")
	return peer.Decode(peerIDStr)
}

// UpdateContact обновляет контакт
func (s *ContactService) UpdateContact(id int, username, multiaddr, notes string) error {
	contact, err := queries.GetContact(id)
	if err != nil {
		return err
	}

	contact.Username = username
	contact.Multiaddr = multiaddr
	contact.Notes = notes

	return queries.UpdateContact(contact)
}

// DeleteContact удаляет контакт по ID
func (s *ContactService) DeleteContact(id int) error {
	contact, err := queries.GetContact(id)
	if err != nil {
		return err
	}

	log.Printf("Удаление контакта: %s (%s)", contact.Username, contact.PeerID)
	return queries.DeleteContact(id)
}

// BlockContact блокирует контакт
func (s *ContactService) BlockContact(id int) error {
	contact, err := queries.GetContact(id)
	if err != nil {
		return err
	}

	contact.IsBlocked = true
	if err := queries.UpdateContact(contact); err != nil {
		return err
	}

	log.Printf("Заблокирован контакт: %s", contact.Username)
	return nil
}

// UnblockContact разблокирует контакт
func (s *ContactService) UnblockContact(id int) error {
	contact, err := queries.GetContact(id)
	if err != nil {
		return err
	}

	contact.IsBlocked = false
	if err := queries.UpdateContact(contact); err != nil {
		return err
	}

	log.Printf("Разблокирован контакт: %s", contact.Username)
	return nil
}

// IsContactBlocked проверяет, заблокирован ли контакт
func (s *ContactService) IsContactBlocked(peerID string) (bool, error) {
	return queries.IsContactBlocked(peerID)
}

// SearchContacts ищет контакты по имени
func (s *ContactService) SearchContacts(query string) ([]*ContactWithStatus, error) {
	contacts, err := queries.SearchContacts(query)
	if err != nil {
		return nil, err
	}

	result := make([]*ContactWithStatus, 0, len(contacts))
	for _, c := range contacts {
		result = append(result, s.enrichContactWithStatus(c))
	}

	return result, nil
}

// RefreshContactStatus обновляет статус контакта
func (s *ContactService) RefreshContactStatus(id int) error {
	contact, err := queries.GetContact(id)
	if err != nil {
		return err
	}

	if s.p2pNetwork == nil {
		return errors.New("P2P сеть не инициализирована")
	}

	peerID, err := s.parsePeerID(contact.PeerID)
	if err != nil {
		return err
	}

	// Проверяем подключение
	status := s.p2pNetwork.GetConnectionStatus(peerID)
	if status == p2p.StatusConnected {
		return queries.UpdateContactStatus(id, "online", nil)
	}

	// Пытаемся подключиться
	if contact.Multiaddr != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := s.p2pNetwork.ConnectToPeer(ctx, contact.Multiaddr)
		if err == nil {
			return queries.UpdateContactStatus(id, "online", nil)
		}
	}

	// Не удалось подключиться
	now := time.Now()
	return queries.UpdateContactStatus(id, "offline", &now)
}

// RefreshAllContactsStatuses обновляет статусы всех контактов
func (s *ContactService) RefreshAllContactsStatuses() {
	contacts, err := s.GetAllContacts()
	if err != nil {
		log.Printf("Ошибка получения контактов: %v", err)
		return
	}

	for _, c := range contacts {
		if err := s.RefreshContactStatus(c.ID); err != nil {
			log.Printf("Ошибка обновления статуса контакта %s: %v", c.Username, err)
		}
		time.Sleep(100 * time.Millisecond) // Небольшая задержка между запросами
	}
}

// SendMessage отправляет сообщение контакту
func (s *ContactService) SendMessage(contactID int, content string) error {
	contact, err := queries.GetContact(contactID)
	if err != nil {
		return err
	}

	if contact.IsBlocked {
		return errors.New("контакт заблокирован")
	}

	if s.p2pNetwork == nil {
		return errors.New("P2P сеть не инициализирована")
	}

	peerID, err := s.parsePeerID(contact.PeerID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.p2pNetwork.SendTextMessage(ctx, peerID, content)
}

// SendFileMessage отправляет файл контакту
func (s *ContactService) SendFileMessage(contactID int, filePath, fileName, mimeType string) error {
	contact, err := queries.GetContact(contactID)
	if err != nil {
		return err
	}

	if contact.IsBlocked {
		return errors.New("контакт заблокирован")
	}

	if s.p2pNetwork == nil {
		return errors.New("P2P сеть не инициализирована")
	}

	peerID, err := s.parsePeerID(contact.PeerID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return s.p2pNetwork.SendFileMessage(ctx, peerID, filePath, fileName, mimeType)
}

// GetContactStats возвращает статистику по контактам
func (s *ContactService) GetContactStats() (*ContactStats, error) {
	contacts, err := s.GetAllContacts()
	if err != nil {
		return nil, err
	}

	stats := &ContactStats{}
	for _, c := range contacts {
		stats.Total++
		if c.IsOnline {
			stats.Online++
		}
		if c.IsBlocked {
			stats.Blocked++
		}
	}

	return stats, nil
}

// ContactStats статистика по контактам
type ContactStats struct {
	Total   int `json:"total"`
	Online  int `json:"online"`
	Offline int `json:"offline"`
	Blocked int `json:"blocked"`
}

// ImportContactFromP2PAddress импортирует контакт из P2P адреса
func (s *ContactService) ImportContactFromP2PAddress(addrStr string, notes string) (*ContactWithStatus, error) {
	contact, err := s.AddContactByAddress(addrStr, notes)
	if err != nil {
		return nil, err
	}

	// Пробуем подключиться
	if s.p2pNetwork != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.p2pNetwork.ConnectToPeer(ctx, addrStr); err == nil {
			_ = queries.UpdateContactStatus(contact.ID, "online", nil)
		}
	}

	return s.enrichContactWithStatus(contact), nil
}

// ExportContactAddress экспортирует адрес контакта
func (s *ContactService) ExportContactAddress(id int) (string, error) {
	contact, err := queries.GetContact(id)
	if err != nil {
		return "", err
	}

	if contact.Multiaddr == "" {
		return "", errors.New("у контакта нет адреса")
	}

	return contact.Multiaddr, nil
}
