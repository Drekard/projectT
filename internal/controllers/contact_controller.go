// Package controllers предоставляет контроллеры для управления бизнес-логикой приложения
package controllers

import (
	"context"
	"fmt"
	"log"
	"projectT/internal/services/p2p/address"
	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ContactController контролирует бизнес-логику управления контактами
type ContactController struct {
	p2pUI *network.UIP2P
}

// NewContactController создаёт новый контроллер контактов
func NewContactController() *ContactController {
	return &ContactController{}
}

// SetP2PService устанавливает P2P сервис
func (cc *ContactController) SetP2PService(p2pUI *network.UIP2P) {
	cc.p2pUI = p2pUI
}

// AddContactByAddress добавляет контакт по адресу
func (cc *ContactController) AddContactByAddress(addrStr, username string) error {
	log.Printf("[ContactController] 📝 Добавление контакта: addr=%s, username=%s", addrStr[:min(20, len(addrStr))], username)

	if cc.p2pUI == nil {
		return fmt.Errorf("P2P сервис не инициализирован")
	}

	// Импортируем адрес пира и добавляем в peerstore
	peerAddr, err := address.ImportPeerAddress(cc.p2pUI.GetNetwork().Host(), addrStr)
	if err != nil {
		return fmt.Errorf("ошибка импорта адреса: %w", err)
	}

	// Получаем PeerID пира
	peerID, err := peer.Decode(peerAddr.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	log.Printf("[ContactController] === AddContactByAddress ===")
	log.Printf("[ContactController] PeerID: %s", peerID.String())
	log.Printf("[ContactController] Адрес: %s", addrStr)

	// Пробуем подключиться к пиру для получения профиля
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Подключаемся к пиру
	log.Printf("[ContactController] Попытка подключения к пиру...")
	if err := address.ConnectToPeer(ctx, cc.p2pUI.GetNetwork().Host(), addrStr); err != nil {
		log.Printf("[ContactController] ❌ Не удалось подключиться к пиру %s: %v", peerID.String(), err)
	} else {
		log.Printf("[ContactController] ✅ Подключение успешно, запрашиваем профиль...")
		go cc.requestProfileAfterConnect(peerID.String())
	}

	// Обновляем multiaddr контакта
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil && contact.Multiaddr != "" {
		if err := queries.UpdateContactByPeerID(peerID.String(), contact.Multiaddr); err != nil {
			log.Printf("[ContactController] Не удалось обновить multiaddr контакта: %v", err)
		}
	}

	return nil
}

// ConnectToContact подключается к контакту по адресу (НЕ создаёт контакт в БД)
func (cc *ContactController) ConnectToContact(addrStr string) error {
	log.Printf("[ContactController] 🔌 Подключение к контакту: addr=%s", addrStr[:min(20, len(addrStr))])

	if cc.p2pUI == nil {
		return fmt.Errorf("P2P сервис не инициализирован")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Подключаемся к пиру
	if err := address.ConnectToPeer(ctx, cc.p2pUI.GetNetwork().Host(), addrStr); err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}

	// Получаем PeerID из адреса
	peerID, err := address.ExtractPeerIDFromAddress(addrStr)
	if err != nil {
		return fmt.Errorf("ошибка извлечения PeerID: %w", err)
	}

	decodedPeerID, err := peer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	// Запрашиваем профиль после подключения
	go cc.requestProfileAfterConnect(decodedPeerID.String())

	return nil
}

// requestProfileAfterConnect запрашивает профиль у пира после подключения
func (cc *ContactController) requestProfileAfterConnect(peerIDStr string) {
	if cc.p2pUI == nil {
		log.Printf("[ContactController] ❌ P2P сервис не инициализирован")
		return
	}

	// Декодируем PeerID
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		log.Printf("[ContactController] ❌ Ошибка декодирования PeerID: %v", err)
		return
	}

	profileCtx, profileCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer profileCancel()

	profileWithSig, err := cc.p2pUI.GetNetwork().ProfileExchange().RequestPeerProfile(profileCtx, peerID)
	if err != nil {
		log.Printf("[ContactController] ❌ Не удалось получить профиль у пира %s: %v", peerIDStr, err)
		return
	}

	// Обновляем remote профиль в БД
	if profileWithSig != nil && profileWithSig.Profile != nil {
		if err := queries.UpdateRemoteProfile(profileWithSig.Profile); err != nil {
			log.Printf("[ContactController] Не удалось обновить профиль пира %s: %v", peerIDStr, err)
		} else {
			log.Printf("[ContactController] ✅ Профиль пира %s получен и сохранён: %s", peerIDStr, profileWithSig.Profile.Username)
		}
	}
}

// GetAllContacts возвращает все контакты
func (cc *ContactController) GetAllContacts() ([]*models.Contact, error) {
	return queries.GetAllContacts()
}

// GetContactByPeerID получает контакт по PeerID
func (cc *ContactController) GetContactByPeerID(peerID string) (*models.Contact, error) {
	return queries.GetContactByPeerID(peerID)
}

// DeleteContact удаляет контакт по ID
func (cc *ContactController) DeleteContact(id int) error {
	return queries.DeleteContact(id)
}

// GetConnectedPeers возвращает список подключённых пиров
func (cc *ContactController) GetConnectedPeers() []*network.PeerInfo {
	if cc.p2pUI == nil {
		return []*network.PeerInfo{}
	}
	return cc.p2pUI.GetConnectedPeers()
}

// GetAllContactsInfo возвращает все контакты с их статусами
func (cc *ContactController) GetAllContactsInfo() []*network.PeerInfo {
	if cc.p2pUI == nil {
		return []*network.PeerInfo{}
	}
	return cc.p2pUI.GetAllContacts()
}

// GetPeerInfo возвращает информацию о пире
func (cc *ContactController) GetPeerInfo(peerIDStr string) *network.PeerInfo {
	if cc.p2pUI == nil {
		return nil
	}
	return cc.p2pUI.GetPeerInfo(peerIDStr)
}

// DisconnectPeer отключается от пира
func (cc *ContactController) DisconnectPeer(peerIDStr string) error {
	if cc.p2pUI == nil {
		return fmt.Errorf("P2P сервис не инициализирован")
	}
	return cc.p2pUI.DisconnectPeer(peerIDStr)
}

// GetPeerAddresses возвращает адреса пира
func (cc *ContactController) GetPeerAddresses(peerIDStr string) []string {
	if cc.p2pUI == nil {
		return []string{}
	}
	return cc.p2pUI.GetPeerAddresses(peerIDStr)
}

// GetLocalAddresses возвращает список локальных адресов
func (cc *ContactController) GetLocalAddresses() []string {
	if cc.p2pUI == nil {
		return []string{}
	}
	return cc.p2pUI.GetLocalAddresses()
}

// GetPeerAddress возвращает адрес текущего пира
func (cc *ContactController) GetPeerAddress() (string, error) {
	if cc.p2pUI == nil {
		return "", fmt.Errorf("P2P сервис не инициализирован")
	}
	return cc.p2pUI.GetPeerAddress()
}

// RequestProfile запрашивает профиль у пира
func (cc *ContactController) RequestProfile(peerIDStr string) error {
	if cc.p2pUI == nil {
		return fmt.Errorf("P2P сервис не инициализирован")
	}
	return cc.p2pUI.RequestProfile(peerIDStr)
}

// RequestAllProfiles запрашивает профили у всех контактов
func (cc *ContactController) RequestAllProfiles() {
	if cc.p2pUI == nil {
		return
	}
	cc.p2pUI.RequestAllProfiles()
}

// GetProfiles возвращает все remote профили
func (cc *ContactController) GetProfiles() ([]*models.Profile, error) {
	return queries.GetAllRemoteProfiles()
}

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
