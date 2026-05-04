// Package core предоставляет функции для инициализации сервисов P2P
package core

import (
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/services/p2p/autodial"
	"projectT/internal/services/p2p/connection"
	"projectT/internal/services/p2p/discovery"
	"projectT/internal/services/p2p/helper"
	"projectT/internal/services/p2p/peerexchange"
	"projectT/internal/services/p2p/protocols/avatar"
	"projectT/internal/services/p2p/protocols/chat"
	"projectT/internal/services/p2p/protocols/itemsync"
	"projectT/internal/services/p2p/protocols/profile"
	"projectT/internal/services/p2p/protocols/profilesync"
	"projectT/internal/services/p2p/protocols/transfer"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// initDiscovery инициализирует и запускает сервис обнаружения
func (n *P2PNetwork) initDiscovery() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.discovery = discovery.NewDiscoveryService(n.host, n.dhtDiscovery, n.config)
	return n.discovery.Start()
}

// initConnections инициализирует и запускает сервис соединений
func (n *P2PNetwork) initConnections() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.connections = connection.NewService(n.host, n.config)
	return n.connections.Start()
}

// initHelper инициализирует режим помощника
func (n *P2PNetwork) initHelper() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.helper = &HelperService{
		helper: helper.NewHelper(n.host, nil),
	}
	return n.helper.helper.Start()
}

// initProfileExchange инициализирует сервис обмена профилями
func (n *P2PNetwork) initProfileExchange() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	// Получаем локальный профиль для получения ID
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		return fmt.Errorf("ошибка получения профиля: %w", err)
	}

	// Получаем ключи из profile_keys
	keys, err := queries.GetProfileKeys(localProfile.ID)
	if err != nil {
		return fmt.Errorf("ошибка получения ключей: %w", err)
	}

	// Восстанавливаем приватный ключ для подписи
	privKey, err := crypto.UnmarshalPrivateKey(keys.PrivateKey)
	if err != nil {
		return fmt.Errorf("ошибка восстановления приватного ключа: %w", err)
	}

	pubKey, err := crypto.UnmarshalPublicKey(keys.PublicKey)
	if err != nil {
		return fmt.Errorf("ошибка восстановления публичного ключа: %w", err)
	}

	n.profileExchange = profile.NewExchangeService(n.host, privKey, pubKey)
	// Передаём connectionService для отслеживания статуса обмена профиля
	n.profileExchange.SetConnectionService(n.connections)
	// ✅ Передаём avatarService для загрузки аватарок
	n.profileExchange.SetAvatarService(n.avatar)

	return n.profileExchange.Start()
}

// initChat инициализирует сервис чата
func (n *P2PNetwork) initChat() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.chat = chat.NewService(n.host, n.localPrivKey, n.localPubKey, n.profileExchange)
	return n.chat.Start()
}

// initItemSync инициализирует сервис синхронизации элементов
func (n *P2PNetwork) initItemSync() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.itemSync = itemsync.NewService(n.host, n.localPrivKey, n.localPubKey)
	return n.itemSync.Start()
}

// initTransfer инициализирует сервис передачи файлов
func (n *P2PNetwork) initTransfer() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.transfer = transfer.NewService(n.host)
	return n.transfer.Start()
}

// initAvatar инициализирует сервис загрузки аватарок
func (n *P2PNetwork) initAvatar() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.avatar = avatar.NewService(n.host)
	return n.avatar.Start()
}

// initAutodial инициализирует менеджер автоподключения
func (n *P2PNetwork) initAutodial() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	config := autodial.DefaultConfig()
	provider := &AutodialProvider{network: n}
	tracker := &ConnectionTracker{network: n}

	n.autodial = autodial.NewDialerManager(n.host, config, provider, tracker)
	return n.autodial.Start()
}

// initPeerExchange инициализирует сервис обмена пирами
func (n *P2PNetwork) initPeerExchange() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	provider := &PeerExchangeProvider{network: n}
	n.peerExchange = peerexchange.NewExchangeService(n.host, provider)
	return n.peerExchange.Start()
}

// initProfileSync инициализирует сервис синхронизации профилей
func (n *P2PNetwork) initProfileSync() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.profileSync = profilesync.NewSyncService(n.host, nil)
	return n.profileSync.Start()
}

// AutodialProvider адаптер для получения адресов пиров
type AutodialProvider struct {
	network *P2PNetwork
}

// GetActivePeerAddresses возвращает все активные адреса
func (p *AutodialProvider) GetActivePeerAddresses() ([]*autodial.PeerAddress, error) {
	addresses, err := queries.GetActivePeerAddresses()
	if err != nil {
		return nil, err
	}

	result := make([]*autodial.PeerAddress, 0, len(addresses))
	for _, addr := range addresses {
		result = append(result, &autodial.PeerAddress{
			PeerID:      addr.PeerID,
			Multiaddr:   addr.Multiaddr,
			AddressType: addr.AddressType,
			Priority:    addr.Priority,
		})
	}

	return result, nil
}

// ConnectionTracker адаптер для отслеживания подключений
type ConnectionTracker struct {
	network *P2PNetwork
}

// GetConnectedPeersCount возвращает количество подключений
func (t *ConnectionTracker) GetConnectedPeersCount() int {
	return t.network.GetConnectedPeersCount()
}

// IsConnected проверяет, подключён ли пир
func (t *ConnectionTracker) IsConnected(peerID peer.ID) bool {
	return t.network.GetConnectionStatus(peerID) == connection.StatusConnected
}

// UpdateConnectedCount обновляет счётчик (не используется, реализовано в dialer)
func (t *ConnectionTracker) UpdateConnectedCount(delta int) {
	// Счётчик обновляется автоматически в dialer
}

// PeerExchangeProvider адаптер для обмена пирами
type PeerExchangeProvider struct {
	network *P2PNetwork
}

// GetKnownPeersForExchange возвращает пиров для обмена
func (p *PeerExchangeProvider) GetKnownPeersForExchange(excludePeerID string, limit int) ([]*models.PeerAddress, error) {
	return queries.GetKnownPeersForExchange(excludePeerID, limit)
}

// AddPeerAddressWithProfile добавляет адрес пира
func (p *PeerExchangeProvider) AddPeerAddressWithProfile(peerID, multiaddr, addressType, source, username string) error {
	return queries.AddPeerAddressWithProfile(peerID, multiaddr, addressType, source, username)
}

// IsKnownPeer проверяет, известен ли пир
func (p *PeerExchangeProvider) IsKnownPeer(peerID string) bool {
	addresses, err := queries.GetActivePeerAddresses()
	if err != nil {
		return false
	}

	for _, addr := range addresses {
		if addr.PeerID == peerID {
			return true
		}
	}
	return false
}
