// Package network предоставляет функции для инициализации сервисов P2P
package network

import (
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"

	p2p "projectT/internal/services/p2p"
	"projectT/internal/storage/database/queries"
)

// initDiscovery инициализирует и запускает сервис обнаружения
func (n *P2PNetwork) initDiscovery() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.discovery = p2p.NewDiscoveryService(n.host, n.dhtDiscovery, n.config)
	return n.discovery.Start()
}

// initConnections инициализирует и запускает сервис соединений
func (n *P2PNetwork) initConnections() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.connections = p2p.NewConnectionService(n.host, n.config)
	return n.connections.Start()
}

// initHelper инициализирует режим помощника
func (n *P2PNetwork) initHelper() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.helper = &HelperService{
		helper: p2p.NewHelper(n.host, nil),
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

	n.profileExchange = p2p.NewProfileExchangeService(n.host, privKey, pubKey)

	return n.profileExchange.Start()
}

// initChat инициализирует сервис чата
func (n *P2PNetwork) initChat() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.chat = p2p.NewChatService(n.host, n.config, n.localPrivKey, n.localPubKey)
	return n.chat.Start()
}
