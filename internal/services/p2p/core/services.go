// Package core предоставляет функции для инициализации сервисов P2P
package core

import (
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"

	"projectT/internal/services/p2p/connection"
	"projectT/internal/services/p2p/discovery"
	"projectT/internal/services/p2p/helper"
	"projectT/internal/services/p2p/protocols/chat"
	"projectT/internal/services/p2p/protocols/itemsync"
	"projectT/internal/services/p2p/protocols/profile"
	"projectT/internal/services/p2p/protocols/transfer"
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
