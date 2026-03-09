// Package network предоставляет функции для управления P2P сетью
package network

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"

	p2p "projectT/internal/services/p2p"
	"projectT/internal/storage/database/queries"
)

// HelperService сервис режима помощника
type HelperService struct {
	helper *p2p.Helper
}

// P2PNetwork представляет P2P сеть проекта
type P2PNetwork struct {
	host            host.Host
	dht             *dht.IpfsDHT
	dhtDiscovery    *routing.RoutingDiscovery
	pubsub          *pubsub.PubSub
	discovery       *p2p.DiscoveryService
	connections     *p2p.ConnectionService
	chat            *p2p.ChatService
	profileExchange *p2p.ProfileExchangeService
	helper          *HelperService
	config          *p2p.P2PConfig
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	peerAddrs       map[peer.ID]multiaddr.Multiaddr
	localPrivKey    crypto.PrivKey
	localPubKey     crypto.PubKey
	profileMgr      *ProfileManager
	keyMgr          *KeyManager
}

// NewP2PNetwork создаёт новый экземпляр P2P сети
func NewP2PNetwork() *P2PNetwork {
	ctx, cancel := context.WithCancel(context.Background())
	return &P2PNetwork{
		config:     p2p.DefaultConfig(),
		ctx:        ctx,
		cancel:     cancel,
		peerAddrs:  make(map[peer.ID]multiaddr.Multiaddr),
		profileMgr: NewProfileManager(),
		keyMgr:     NewKeyManager(),
	}
}

// SetMasterPassword устанавливает мастер-пароль для шифрования приватного ключа
// Должен вызываться перед Start()
func (n *P2PNetwork) SetMasterPassword(password string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.config.MasterPassword = password
	n.profileMgr.SetMasterPassword(password)
}

// VerifyPassword проверяет правильность пароля для расшифровки приватного ключа
func (n *P2PNetwork) VerifyPassword(password string) (bool, error) {
	return VerifyPassword(password)
}

// ChangePassword меняет пароль шифрования приватного ключа
func (n *P2PNetwork) ChangePassword(oldPassword, newPassword string) error {
	n.mu.RLock()
	privKey := n.localPrivKey
	n.mu.RUnlock()
	return ChangePassword(oldPassword, newPassword, privKey)
}

// IsKeyEncrypted возвращает true, если приватный ключ зашифрован
func (n *P2PNetwork) IsKeyEncrypted() (bool, error) {
	return IsKeyEncrypted()
}

// EnableEncryption включает шифрование приватного ключа с заданным паролем
func (n *P2PNetwork) EnableEncryption(password string) error {
	profile, err := queries.GetP2PProfile()
	if err != nil {
		return fmt.Errorf("ошибка загрузки профиля: %w", err)
	}
	return EnableEncryption(profile, password)
}

// Start запускает P2P сеть
func (n *P2PNetwork) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Загружаем или создаём профиль
	profile, err := n.profileMgr.LoadOrCreateProfile()
	if err != nil {
		return fmt.Errorf("ошибка загрузки профиля: %w", err)
	}

	// Создаём хост
	if err := n.createHost(profile); err != nil {
		return fmt.Errorf("ошибка создания хоста: %w", err)
	}

	// Настраиваем обработчики соединений
	n.host.SetStreamHandler(p2p.ChatProtocolID, n.handleChatStream)
	n.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			n.onPeerConnected(conn.RemotePeer())
		},
		DisconnectedF: func(net network.Network, conn network.Conn) {
			n.onPeerDisconnected(conn.RemotePeer())
		},
	})

	log.Printf("P2P хост запущен: %s", n.host.ID().String())
	log.Printf("Адреса для подключения: %v", n.host.Addrs())

	// Обновляем профиль в БД
	if err := UpdateProfileAddrs(n.host.ID(), n.host.Addrs()); err != nil {
		log.Printf("Предупреждение: не удалось обновить адреса в профиле: %v", err)
	}

	// Инициализируем сервис обмена профилями
	if err := n.initProfileExchange(); err != nil {
		log.Printf("Предупреждение: сервис обмена профилями не инициализирован: %v", err)
	}

	// Инициализируем и запускаем сервис обнаружения
	if err := n.initDiscovery(); err != nil {
		log.Printf("Предупреждение: сервис обнаружения не инициализирован: %v", err)
	}

	// Инициализируем и запускаем сервис соединений
	if err := n.initConnections(); err != nil {
		log.Printf("Предупреждение: сервис соединений не инициализирован: %v", err)
	}

	// Инициализируем режим помощника если включён
	if n.config.EnableHelperMode {
		if err := n.initHelper(); err != nil {
			log.Printf("Предупреждение: режим помощника не инициализирован: %v", err)
		} else {
			log.Println("Режим ПОМОЩНИКА инициализирован")
		}
	}

	return nil
}

// Stop останавливает P2P сеть
func (n *P2PNetwork) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.cancel()

	var errs []string

	// Останавливаем сервис чата
	if n.chat != nil {
		if err := n.chat.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("Chat: %v", err))
		}
	}

	// Останавливаем сервис обнаружения
	if n.discovery != nil {
		if err := n.discovery.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("Discovery: %v", err))
		}
	}

	// Останавливаем сервис соединений
	if n.connections != nil {
		if err := n.connections.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("Connections: %v", err))
		}
	}

	// Останавливаем режим помощника
	if n.helper != nil && n.helper.helper != nil {
		if err := n.helper.helper.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("Helper: %v", err))
		}
	}

	// Останавливаем сервис обмена профилями
	if n.profileExchange != nil {
		if err := n.profileExchange.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("ProfileExchange: %v", err))
		}
	}

	if n.dht != nil {
		if err := n.dht.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("DHT: %v", err))
		}
	}
	if n.host != nil {
		if err := n.host.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("Host: %v", err))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
