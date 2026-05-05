// Package controllers предоставляет контроллеры для управления бизнес-логикой приложения
package controllers

import (
	"context"
	"fmt"
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

	// Сохраняем адрес пира в БД для автоподключения при следующем запуске
	if err := queries.AddPeerAddressWithProfile(peerID.String(), addrStr, "contact", "add_contact", ""); err != nil {
		// Не критичная ошибка, продолжаем
		_ = err
	}

	// Пробуем подключиться к пиру для получения профиля
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Подключаемся к пиру
	if err := address.ConnectToPeer(ctx, cc.p2pUI.GetNetwork().Host(), addrStr); err != nil {
		// Ignore error
	} else {
		go cc.requestProfileAfterConnect(peerID.String())
	}

	// Обновляем multiaddr контакта
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil && contact.Multiaddr != "" {
		_ = queries.UpdateContactByPeerID(peerID.String(), contact.Multiaddr)
	}

	return nil
}

// ConnectToContact подключается к контакту по адресу (НЕ создаёт контакт в БД)
func (cc *ContactController) ConnectToContact(addrStr string) error {
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

	// Сохраняем адрес пира в БД для автоподключения при следующем запуске
	if err := queries.AddPeerAddressWithProfile(peerID, addrStr, "contact", "manual_connect", ""); err != nil {
		// Не критичная ошибка, продолжаем
		_ = err
	}

	// Запрашиваем профиль после подключения
	go cc.requestProfileAfterConnect(decodedPeerID.String())

	return nil
}

// requestProfileAfterConnect запрашивает профиль у пира после подключения
func (cc *ContactController) requestProfileAfterConnect(peerIDStr string) {
	if cc.p2pUI == nil {
		return
	}

	// Декодируем PeerID
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return
	}

	profileCtx, profileCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer profileCancel()

	profileWithSig, err := cc.p2pUI.GetNetwork().ProfileExchange().RequestPeerProfile(profileCtx, peerID)
	if err != nil {
		return
	}

	// Обновляем remote профиль в БД
	if profileWithSig != nil && profileWithSig.Profile != nil {
		_ = queries.UpdateRemoteProfile(profileWithSig.Profile)
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
