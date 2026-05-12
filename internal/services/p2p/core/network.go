// Package core предоставляет функции для управления P2P сетью
package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"

	"projectT/internal/services"
	p2p "projectT/internal/services/p2p"
	"projectT/internal/services/p2p/address"
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
	"projectT/internal/storage/database/queries"
)

// HelperService сервис режима помощника
type HelperService struct {
	helper *helper.Helper
}

// P2PNetwork представляет P2P сеть проекта
type P2PNetwork struct {
	host               host.Host
	dht                *dht.IpfsDHT
	dhtDiscovery       *routing.RoutingDiscovery
	pubsub             *pubsub.PubSub
	discovery          *discovery.DiscoveryService
	connections        *connection.Service
	chat               *chat.Service
	profileExchange    *profile.ExchangeService
	itemSync           *itemsync.Service
	transfer           *transfer.Service
	avatar             *avatar.Service               // ✅ Сервис загрузки аватарок
	autodial           *autodial.DialerManager       // ✅ Менеджер автоподключения
	peerExchange       *peerexchange.ExchangeService // ✅ Сервис обмена пирами
	profileSync        *profilesync.SyncService      // ✅ Сервис синхронизации профилей
	helper             *HelperService
	config             *p2p.P2PConfig
	prometheusRegistry prometheus.Registerer // Prometheus registry для libp2p метрик
	ctx                context.Context
	cancel             context.CancelFunc
	mu                 sync.RWMutex
	peerAddrs          map[peer.ID]multiaddr.Multiaddr
	localPrivKey       crypto.PrivKey
	localPubKey        crypto.PubKey
	profileMgr         *ProfileManager
	keyMgr             *KeyManager
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

// SetPrometheusRegistry устанавливает Prometheus registry для libp2p метрик
func (n *P2PNetwork) SetPrometheusRegistry(registry prometheus.Registerer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.prometheusRegistry = registry
	n.config.EnablePrometheusMetrics = true
}

// SetPort устанавливает порт для P2P соединений
func (n *P2PNetwork) SetPort(port int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.config.ListenPort = port
}

// SetAutoConnect устанавливает флаг автоподключения
func (n *P2PNetwork) SetAutoConnect(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.config.EnableAutoConnect = enabled
}

// SetAutoProfileEx устанавливает флаг автоматического обмена профилями
func (n *P2PNetwork) SetAutoProfileEx(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.config.EnableAutoProfileEx = enabled
}

// Config возвращает текущую конфигурацию
func (n *P2PNetwork) Config() *p2p.P2PConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.config
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

	// Автоматически открываем порт в брандмауэре Windows
	n.openFirewallPort()

	// Настраиваем обработчики соединений
	n.host.SetStreamHandler(chat.ProtocolID, n.handleChatStream)
	n.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			n.onPeerConnected(conn.RemotePeer())
		},
		DisconnectedF: func(net network.Network, conn network.Conn) {
			n.onPeerDisconnected(conn.RemotePeer())
		},
	})

	// Обновляем профиль в БД
	if err := UpdateProfileAddrs(n.host.ID(), n.host.Addrs()); err != nil {
		// Игнорируем ошибку, если поле listen_addrs временно недоступно
		if !strings.Contains(err.Error(), "listen_addrs") {
			_ = err // Ignore error
		}
	}

	// Инициализируем сервис соединений (должен быть до profileExchange)
	if err := n.initConnections(); err != nil {
		_ = err // Ignore error
	}

	// ✅ Инициализируем сервис загрузки аватарок (ДО profileExchange, т.к. он использует avatar)
	if err := n.initAvatar(); err != nil {
		_ = err // Ignore error
	}

	// Инициализируем сервис обмена профилями
	if err := n.initProfileExchange(); err != nil {
		_ = err // Ignore error
	}

	// Инициализируем сервис передачи файлов
	if err := n.initTransfer(); err != nil {
		_ = err // Ignore error
	}

	// Инициализируем сервис синхронизации элементов
	if err := n.initItemSync(); err != nil {
		_ = err // Ignore error
	}

	// Инициализируем сервис чата (после transfer для интеграции)
	if err := n.initChat(); err != nil {
		_ = err // Ignore error
	}

	// Связываем сервис чата с сервисом передачи файлов
	if n.chat != nil && n.transfer != nil {
		n.chat.SetTransferService(n.transfer)
		n.chat.SetItemSyncService(n.itemSync)
	}

	// Устанавливаем ItemSync сервис в глобальный ChatService для использования из profile_exchange
	globalChatSvc := services.GetChatService()
	if globalChatSvc != nil && n.itemSync != nil {
		globalChatSvc.SetItemSyncService(n.itemSync)
	}

	// Инициализируем и запускаем сервис обнаружения
	if err := n.initDiscovery(); err != nil {
		_ = err // Ignore error
	}

	// ✅ Инициализируем сервис автоподключения
	if err := n.initAutodial(); err != nil {
		_ = err // Ignore error
	}

	// ✅ Запускаем автоподключение к известным пирам (если включено в настройках)
	if n.config.EnableAutoConnect {
		go func() {
			log.Printf("[P2P] 🚀 Запуск автоподключения к известным пирам...")
			results := n.autodial.ConnectToAll()
			for result := range results {
				peerShort := "unknown"
				if result.PeerID.String() != "" {
					peerShort = result.PeerID.String()[:8]
				}
				if result.Success {
					log.Printf("[P2P] ✅ Подключение к пиру %s успешно", peerShort)
				} else {
					log.Printf("[P2P] ❌ Подключение к пиру %s не удалось: %v", peerShort, result.Error)
				}
			}
		}()
	}

	// ✅ Инициализируем сервис обмена пирами
	if err := n.initPeerExchange(); err != nil {
		_ = err // Ignore error
	}

	// ✅ Инициализируем сервис синхронизации профилей
	if err := n.initProfileSync(); err != nil {
		_ = err // Ignore error
	}

	// ✅ Автоматический обмен профилями с подключёнными пирами (если включено в настройках)
	if n.config.EnableAutoProfileEx {
		go func() {
			// Ждём немного, чтобы пиры успели подключиться
			time.Sleep(5 * time.Second)
			peers := n.host.Network().Peers()
			if len(peers) > 0 && n.profileSync != nil {
				log.Printf("[P2P] 🔄 Запуск автоматического обмена профилями с %d пирами...", len(peers))
				for _, peerID := range peers {
					go func(pid peer.ID) {
						ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
						defer cancel()
						_ = n.profileSync.SyncWithPeer(ctx, pid)
					}(peerID)
				}
			}
		}()
	}

	return nil
}

// Stop останавливает P2P сеть
func (n *P2PNetwork) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.cancel()

	var errs []string

	// Останавливаем сервис передачи файлов
	if n.transfer != nil {
		if err := n.transfer.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("Transfer: %v", err))
		}
	}

	// Останавливаем сервис синхронизации элементов
	if n.itemSync != nil {
		if err := n.itemSync.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("ItemSync: %v", err))
		}
	}

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

	// Останавливаем сервис синхронизации профилей
	if n.profileSync != nil {
		if err := n.profileSync.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("ProfileSync: %v", err))
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

// Ctx возвращает контекст P2P сети
func (n *P2PNetwork) Ctx() context.Context {
	return n.ctx
}

// Avatar возвращает сервис загрузки аватарок
func (n *P2PNetwork) Avatar() *avatar.Service {
	return n.avatar
}

// openFirewallPort автоматически открывает порт P2P в брандмауэре Windows
func (n *P2PNetwork) openFirewallPort() {
	port := n.config.ListenPort
	if port <= 0 {
		port = 8080
	}

	ruleName := "ProjectT P2P"

	log.Printf("[Firewall] Проверка порта %d в брандмауэре...", port)

	result, err := address.OpenFirewallRule(port, ruleName)
	if err != nil {
		log.Printf("[Firewall] ⚠️ Ошибка проверки брандмауэра: %v", err)
		log.Printf("[Firewall] 💡 Если подключение не работает, откройте порт вручную:")
		log.Printf("[Firewall] 💡   netsh advfirewall firewall add rule name=\"%s\" dir=in action=allow protocol=TCP localport=%d", ruleName, port)
		return
	}

	if result.Success {
		log.Printf("[Firewall] ✅ Порт %d открыт в брандмауэре: %s", port, result.Message)
	} else {
		log.Printf("[Firewall] ⚠️ Не удалось открыть порт: %s", result.Message)
		if result.Command != nil {
			log.Printf("[Firewall] 💡 Выполните от имени администратора:")
			log.Printf("[Firewall] 💡   %s", result.Command.PowerShell)
		}
	}
}
