// Package network предоставляет функции для управления P2P сетью
package network

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"

	p2pcrypto "projectT/internal/services/crypto"
	p2p "projectT/internal/services/p2p"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// HelperService сервис режима помощника
type HelperService struct {
	helper *p2p.Helper
}

// P2PNetwork представляет P2P сеть проекта
type P2PNetwork struct {
	host         host.Host
	dht          *dht.IpfsDHT
	dhtDiscovery *routing.RoutingDiscovery
	pubsub       *pubsub.PubSub
	discovery    *p2p.DiscoveryService
	connections  *p2p.ConnectionService
	chat         *p2p.ChatService
	helper       *HelperService
	config       *p2p.P2PConfig
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	peerAddrs    map[peer.ID]multiaddr.Multiaddr
	localPrivKey crypto.PrivKey
	localPubKey  crypto.PubKey
	profileMgr   *ProfileManager
	keyMgr       *KeyManager
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

// Host возвращает libp2p хост
func (n *P2PNetwork) Host() host.Host {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.host
}

// DHT возвращает DHT таблицу
func (n *P2PNetwork) DHT() *dht.IpfsDHT {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.dht
}

// PubSub возвращает PubSub систему
func (n *P2PNetwork) PubSub() *pubsub.PubSub {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.pubsub
}

// PeerID возвращает идентификатор текущего пира
func (n *P2PNetwork) PeerID() peer.ID {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.host == nil {
		return ""
	}
	return n.host.ID()
}

// GetPeerAddress возвращает адрес текущего пира для экспорта
func (n *P2PNetwork) GetPeerAddress() (*p2p.PeerAddress, error) {
	n.mu.RLock()
	host := n.host
	n.mu.RUnlock()
	return p2p.GetPeerAddress(host)
}

// ImportPeerAddress импортирует адрес пира и добавляет в контакты
func (n *P2PNetwork) ImportPeerAddress(addrStr string) (*p2p.PeerAddress, error) {
	n.mu.Lock()
	host := n.host
	n.mu.Unlock()
	return p2p.ImportPeerAddress(host, addrStr)
}

// ConnectToPeer подключается к пиру по адресу
func (n *P2PNetwork) ConnectToPeer(ctx context.Context, addrStr string) error {
	n.mu.RLock()
	host := n.host
	n.mu.RUnlock()
	return p2p.ConnectToPeer(ctx, host, addrStr)
}

// GetConnectedPeers возвращает список подключённых пиров
func (n *P2PNetwork) GetConnectedPeers() []peer.ID {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.host == nil {
		return []peer.ID{}
	}
	return n.host.Network().Peers()
}

// createHost создаёт libp2p хост
func (n *P2PNetwork) createHost(profile *models.P2PProfile) error {
	// Десериализуем приватный ключ (с расшифровкой если нужно)
	var privKey crypto.PrivKey
	var err error

	// Проверяем, зашифрован ли ключ по маркеру в данных
	if p2pcrypto.IsEncryptedKey(profile.PrivateKey) {
		// Ключ зашифрован — требуем пароль
		if n.config.MasterPassword == "" {
			return errors.New("приватный ключ зашифрован, но мастер-пароль не установлен")
		}

		// Пробуем расшифровать
		privKeyRaw, err := p2pcrypto.DecryptPrivateKey(profile.PrivateKey, n.config.MasterPassword)
		if err != nil {
			return fmt.Errorf("ошибка расшифровки приватного ключа (неверный пароль?): %w", err)
		}
		privKey, err = crypto.UnmarshalPrivateKey(privKeyRaw)
		if err != nil {
			return fmt.Errorf("ошибка десериализации приватного ключа: %w", err)
		}
		log.Println("Приватный ключ расшифрован")
	} else {
		// Ключ не зашифрован
		privKey, err = crypto.UnmarshalPrivateKey(profile.PrivateKey)
		if err != nil {
			return fmt.Errorf("ошибка десериализации приватного ключа: %w", err)
		}
		if n.config.MasterPassword != "" {
			log.Println("Предупреждение: приватный ключ не зашифрован, хотя пароль установлен")
		}
	}

	// Загружаем bootstrap-пиры для использования как статические релеи
	bootstrapPeers, _ := queries.GetAllBootstrapPeers()
	var staticRelays []peer.AddrInfo
	for _, p := range bootstrapPeers {
		addr, err := multiaddr.NewMultiaddr(p.Multiaddr)
		if err != nil {
			continue
		}
		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			continue
		}
		staticRelays = append(staticRelays, *info)
	}

	// Получаем публичный ключ из приватного
	pubKey := privKey.GetPublic()

	// Опции хоста
	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(n.config.ListenAddrs...),
		libp2p.NATPortMap(),                                  // Проброс портов через NAT (UPnP/NAT-PMP)
		libp2p.EnableRelay(),                                 // Включает relay для обхода NAT
		libp2p.EnableAutoRelayWithStaticRelays(staticRelays), // Автовыбор relay
		libp2p.EnableHolePunching(),                          // 🔥 NAT Hole Punching для прямых соединений
		libp2p.UserAgent("ProjectT/1.0"),
	}

	// Создаём хост
	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("ошибка создания хоста: %w", err)
	}

	n.host = h
	n.localPrivKey = privKey
	n.localPubKey = pubKey

	// Инициализируем DHT
	if n.config.EnableDHT {
		if err := n.initDHT(); err != nil {
			log.Printf("Предупреждение: DHT не инициализирована: %v", err)
		}
	}

	// Инициализируем PubSub
	if err := n.initPubSub(); err != nil {
		log.Printf("Предупреждение: PubSub не инициализирована: %v", err)
	}

	// Инициализируем ChatService
	if err := n.initChat(); err != nil {
		log.Printf("Предупреждение: ChatService не инициализирован: %v", err)
	}

	return nil
}

// initDHT инициализирует DHT таблицу
func (n *P2PNetwork) initDHT() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	ctx := n.ctx
	dhtOpts := []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix("/" + p2p.ProtocolPrefix),
	}

	kdht, err := dht.New(ctx, n.host, dhtOpts...)
	if err != nil {
		return fmt.Errorf("ошибка создания DHT: %w", err)
	}

	n.dht = kdht

	// Создаём RoutingDiscovery для использования в сервисе обнаружения
	n.dhtDiscovery = routing.NewRoutingDiscovery(kdht)

	log.Println("DHT инициализирована")
	return nil
}

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

// initPubSub инициализирует PubSub систему
func (n *P2PNetwork) initPubSub() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	ps, err := pubsub.NewGossipSub(n.ctx, n.host)
	if err != nil {
		return fmt.Errorf("ошибка создания PubSub: %w", err)
	}

	n.pubsub = ps
	log.Println("PubSub инициализирована")
	return nil
}

// onPeerConnected вызывается при подключении пира
func (n *P2PNetwork) onPeerConnected(peerID peer.ID) {
	log.Printf("Пир подключён: %s", peerID.String())

	// Обновляем статус контакта в БД
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil {
		now := time.Now()
		_ = queries.UpdateContactStatus(contact.ID, "online", &now)
	}
}

// onPeerDisconnected вызывается при отключении пира
func (n *P2PNetwork) onPeerDisconnected(peerID peer.ID) {
	log.Printf("Пир отключён: %s", peerID.String())

	// Обновляем статус контакта в БД
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil {
		now := time.Now()
		_ = queries.UpdateContactStatus(contact.ID, "offline", &now)
	}
}

// handleChatStream обрабатывает входящий поток чата
func (n *P2PNetwork) handleChatStream(stream network.Stream) {
	defer stream.Close()
	// Делегируем обработку в ChatService
	if n.chat != nil {
		n.chat.HandleChatStream(stream)
	} else {
		log.Printf("Получен поток чата от: %s (ChatService не инициализирован)", stream.Conn().RemotePeer().String())
	}
}

// initChat инициализирует сервис чата
func (n *P2PNetwork) initChat() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	n.chat = p2p.NewChatService(n.host, n.config, n.localPrivKey, n.localPubKey)
	return n.chat.Start()
}

// Discovery возвращает сервис обнаружения
func (n *P2PNetwork) Discovery() *p2p.DiscoveryService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.discovery
}

// AddBootstrapPeer добавляет bootstrap-узел
func (n *P2PNetwork) AddBootstrapPeer(multiaddr string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.discovery == nil {
		return errors.New("сервис обнаружения не инициализирован")
	}
	return n.discovery.AddBootstrapPeer(multiaddr)
}

// RemoveBootstrapPeer удаляет bootstrap-узел
func (n *P2PNetwork) RemoveBootstrapPeer(multiaddr string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.discovery == nil {
		return errors.New("сервис обнаружения не инициализирован")
	}
	return n.discovery.RemoveBootstrapPeer(multiaddr)
}

// GetBootstrapPeers возвращает список bootstrap-узлов
func (n *P2PNetwork) GetBootstrapPeers() ([]*models.BootstrapPeer, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.discovery == nil {
		return nil, errors.New("сервис обнаружения не инициализирован")
	}
	return n.discovery.GetBootstrapPeers()
}

// GetDiscoveredPeers возвращает список обнаруженных пиров
func (n *P2PNetwork) GetDiscoveredPeers() map[string]time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.discovery == nil {
		return make(map[string]time.Time)
	}
	return n.discovery.GetDiscoveredPeers()
}

// Connections возвращает сервис соединений
func (n *P2PNetwork) Connections() *p2p.ConnectionService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connections
}

// GetConnectionStatus возвращает статус подключения к пиру
func (n *P2PNetwork) GetConnectionStatus(peerID peer.ID) p2p.ConnectionStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.connections == nil {
		return p2p.StatusUnknown
	}
	return n.connections.GetConnectionStatus(peerID)
}

// GetConnectedPeersCount возвращает количество подключённых пиров
func (n *P2PNetwork) GetConnectedPeersCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.connections == nil {
		return 0
	}
	return n.connections.GetConnectedPeersCount()
}

// GetPeerInfo возвращает информацию о подключении к пиру
func (n *P2PNetwork) GetPeerInfo(peerID peer.ID) *p2p.PeerConnectionInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.connections == nil {
		return nil
	}
	return n.connections.GetPeerInfo(peerID)
}

// Chat возвращает сервис чата
func (n *P2PNetwork) Chat() *p2p.ChatService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.chat
}

// SendMessage отправляет сообщение пиру
func (n *P2PNetwork) SendMessage(ctx context.Context, peerID peer.ID, content, contentType, metadata string) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.SendMessage(ctx, peerID, content, contentType, metadata)
}

// SendTextMessage отправляет текстовое сообщение
func (n *P2PNetwork) SendTextMessage(ctx context.Context, peerID peer.ID, content string) error {
	return n.SendMessage(ctx, peerID, content, "text", "")
}

// SendFileMessage отправляет сообщение с файлом
func (n *P2PNetwork) SendFileMessage(ctx context.Context, peerID peer.ID, filePath, fileName, mimeType string) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.SendFileMessage(ctx, peerID, filePath, fileName, mimeType)
}

// SendImageMessage отправляет сообщение с изображением
func (n *P2PNetwork) SendImageMessage(ctx context.Context, peerID peer.ID, imagePath, imageName string) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.SendImageMessage(ctx, peerID, imagePath, imageName)
}

// GetMessagesForContact получает сообщения для контакта
func (n *P2PNetwork) GetMessagesForContact(contactID int, limit, offset int) ([]*models.ChatMessage, error) {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return nil, errors.New("ChatService не инициализирован")
	}
	return chat.GetMessagesForContact(contactID, limit, offset)
}

// GetUnreadMessagesCount получает количество непрочитанных сообщений
func (n *P2PNetwork) GetUnreadMessagesCount(contactID int) (int, error) {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return 0, errors.New("ChatService не инициализирован")
	}
	return chat.GetUnreadMessagesCount(contactID)
}

// MarkMessageAsRead помечает сообщение как прочитанное
func (n *P2PNetwork) MarkMessageAsRead(id int) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.MarkMessageAsRead(id)
}

// MarkAllMessagesAsRead помечает все сообщения для контакта как прочитанные
func (n *P2PNetwork) MarkAllMessagesAsRead(contactID int) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.MarkAllMessagesAsRead(contactID)
}

// GetQueuedMessagesCount возвращает количество сообщений в очереди для пира
func (n *P2PNetwork) GetQueuedMessagesCount(peerID peer.ID) int {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return 0
	}
	return chat.GetQueuedMessagesCount(peerID)
}

// Helper возвращает сервис режима помощника
func (n *P2PNetwork) Helper() *HelperService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.helper
}

// HelperRegister регистрирует адрес пира в хранилище помощника
func (n *P2PNetwork) HelperRegister(peerID, address string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.helper == nil || n.helper.helper == nil {
		return errors.New("режим помощника не инициализирован")
	}
	return n.helper.helper.Register(peerID, address)
}

// HelperAsk запрашивает адрес пира из хранилища помощника
func (n *P2PNetwork) HelperAsk(peerID string) (*p2p.PeerAddressData, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.helper == nil || n.helper.helper == nil {
		return nil, false
	}
	return n.helper.helper.Ask(peerID)
}

// HelperList возвращает список всех зарегистрированных пиров
func (n *P2PNetwork) HelperList() []p2p.PeerEntry {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.helper == nil || n.helper.helper == nil {
		return nil
	}
	return n.helper.helper.List()
}

// HelperGetPeerCount возвращает количество зарегистрированных пиров
func (n *P2PNetwork) HelperGetPeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.helper == nil || n.helper.helper == nil {
		return 0
	}
	return n.helper.helper.GetPeerCount()
}
