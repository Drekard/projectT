// Package network предоставляет функции для инициализации сервисов P2P
package network

import (
	"errors"
	"fmt"

	p2p "projectT/internal/services/p2p"
	"projectT/internal/storage/database/models"
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

	// Загружаем локальный профиль
	profile, err := queries.GetP2PProfile()
	if err != nil {
		return fmt.Errorf("ошибка загрузки профиля: %w", err)
	}

	// Получаем информацию о профиле из БД
	userProfile, err := queries.GetProfile()
	if err != nil {
		// Если профиль не найден, используем значения по умолчанию
		userProfile = &models.Profile{
			Username: "Аноним",
			Status:   "Доступен",
		}
	}

	// Создаём сервис обмена профилями
	n.profileExchange = p2p.NewProfileExchangeService(
		n.host,
		userProfile.Username,
		userProfile.AvatarPath,
		userProfile.Status,
		profile.PublicKey,
	)

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
