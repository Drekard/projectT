// Package p2p содержит сервисы для P2P связи на базе libp2p.
package p2p

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"

	p2pcrypto "projectT/internal/services/crypto"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// PeerAddress структура для экспорта адреса пира
type PeerAddress struct {
	PeerID    string `json:"peer_id"`
	Multiaddr string `json:"multiaddr"`
	PublicKey string `json:"public_key"`
}

// P2PNetwork представляет P2P сеть проекта
type P2PNetwork struct {
	host      host.Host
	dht       *dht.IpfsDHT
	pubsub    *pubsub.PubSub
	config    *P2PConfig
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	peerAddrs map[peer.ID]multiaddr.Multiaddr
}

// NewP2PNetwork создаёт новый экземпляр P2P сети
func NewP2PNetwork() *P2PNetwork {
	ctx, cancel := context.WithCancel(context.Background())
	return &P2PNetwork{
		config:    DefaultConfig(),
		ctx:       ctx,
		cancel:    cancel,
		peerAddrs: make(map[peer.ID]multiaddr.Multiaddr),
	}
}

// SetMasterPassword устанавливает мастер-пароль для шифрования приватного ключа
// Должен вызываться перед Start()
func (n *P2PNetwork) SetMasterPassword(password string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.config.MasterPassword = password
}

// VerifyPassword проверяет правильность пароля для расшифровки приватного ключа
// Можно использовать для проверки пароля перед запуском P2P сети
func (n *P2PNetwork) VerifyPassword(password string) (bool, error) {
	profile, err := queries.GetP2PProfile()
	if err != nil {
		return false, fmt.Errorf("ошибка загрузки профиля: %w", err)
	}

	if !profile.IsKeyEncrypted {
		// Ключ не зашифрован — пароль не требуется
		return true, nil
	}

	// Проверяем пароль
	if !p2pcrypto.VerifyPassword(profile.PrivateKey, password) {
		return false, errors.New("неверный пароль")
	}

	return true, nil
}

// ChangePassword меняет пароль шифрования приватного ключа
// Требует ввода старого и нового пароля
func (n *P2PNetwork) ChangePassword(oldPassword, newPassword string) error {
	profile, err := queries.GetP2PProfile()
	if err != nil {
		return fmt.Errorf("ошибка загрузки профиля: %w", err)
	}

	if !profile.IsKeyEncrypted {
		// Если ключ не зашифрован, просто шифруем новым паролем
		privKeyRaw, err := crypto.MarshalPrivateKey(n.host.Peerstore().PrivKey(n.host.ID()))
		if err != nil {
			return fmt.Errorf("ошибка сериализации ключа: %w", err)
		}
		encryptedKey, err := p2pcrypto.EncryptPrivateKey(privKeyRaw, newPassword)
		if err != nil {
			return fmt.Errorf("ошибка шифрования ключа: %w", err)
		}
		return queries.ChangeP2PKeyPassword(encryptedKey)
	}

	// Расшифровываем старым паролем и шифруем новым
	newEncryptedKey, err := p2pcrypto.ChangePassword(profile.PrivateKey, oldPassword, newPassword)
	if err != nil {
		return fmt.Errorf("ошибка смены пароля: %w", err)
	}

	return queries.ChangeP2PKeyPassword(newEncryptedKey)
}

// IsKeyEncrypted возвращает true, если приватный ключ зашифрован
func (n *P2PNetwork) IsKeyEncrypted() (bool, error) {
	return queries.IsP2PKeyEncrypted()
}

// EnableEncryption включает шифрование приватного ключа с заданным паролем
// Используется, если профиль был создан без шифрования
func (n *P2PNetwork) EnableEncryption(password string) error {
	profile, err := queries.GetP2PProfile()
	if err != nil {
		return fmt.Errorf("ошибка загрузки профиля: %w", err)
	}

	if profile.IsKeyEncrypted {
		return errors.New("ключ уже зашифрован")
	}

	// Шифруем приватный ключ
	encryptedKey, err := p2pcrypto.EncryptPrivateKey(profile.PrivateKey, password)
	if err != nil {
		return fmt.Errorf("ошибка шифрования ключа: %w", err)
	}

	return queries.ChangeP2PKeyPassword(encryptedKey)
}

// Start запускает P2P сеть
func (n *P2PNetwork) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Загружаем или создаём профиль
	profile, err := n.loadOrCreateProfile()
	if err != nil {
		return fmt.Errorf("ошибка загрузки профиля: %w", err)
	}

	// Создаём хост
	if err := n.createHost(profile); err != nil {
		return fmt.Errorf("ошибка создания хоста: %w", err)
	}

	// Настраиваем обработчики соединений
	n.host.SetStreamHandler(ChatProtocolID, n.handleChatStream)
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
	if err := n.updateProfileAddrs(); err != nil {
		log.Printf("Предупреждение: не удалось обновить адреса в профиле: %v", err)
	}

	return nil
}

// Stop останавливает P2P сеть
func (n *P2PNetwork) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.cancel()

	var errs []string
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
func (n *P2PNetwork) GetPeerAddress() (*PeerAddress, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.host == nil {
		return nil, errors.New("хост не инициализирован")
	}

	// Получаем приватный ключ для извлечения публичного
	privKey := n.host.Peerstore().PrivKey(n.host.ID())
	if privKey == nil {
		return nil, errors.New("не удалось получить приватный ключ")
	}

	pubKeyBytes, err := privKey.GetPublic().Raw()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения публичного ключа: %w", err)
	}

	// Формируем полный адрес
	addr := n.host.Addrs()[0].String()
	fullAddr := fmt.Sprintf("%s/p2p/%s", addr, n.host.ID().String())

	return &PeerAddress{
		PeerID:    n.host.ID().String(),
		Multiaddr: fullAddr,
		PublicKey: base64.StdEncoding.EncodeToString(pubKeyBytes),
	}, nil
}

// ImportPeerAddress импортирует адрес пира и добавляет в контакты
func (n *P2PNetwork) ImportPeerAddress(addrStr string) (*PeerAddress, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.host == nil {
		return nil, errors.New("хост не инициализирован")
	}

	// Парсим адрес
	addr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга адреса: %w", err)
	}

	// Извлекаем PeerID
	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return nil, fmt.Errorf("ошибка извлечения PeerID: %w", err)
	}

	// Добавляем в peerstore
	n.host.Peerstore().AddAddr(info.ID, addr, peerstore.PermanentAddrTTL)

	// Получаем публичный ключ
	pubKey := n.host.Peerstore().PubKey(info.ID)
	if pubKey == nil {
		return nil, errors.New("публичный ключ не найден")
	}

	pubKeyBytes, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения публичного ключа: %w", err)
	}

	// Создаём контакт в БД
	contact := &models.Contact{
		PeerID:    info.ID.String(),
		Username:  info.ID.String()[:8], // Первые 8 символов как временное имя
		Multiaddr: addrStr,
		PublicKey: pubKeyBytes,
		Status:    "offline",
		Notes:     "",
	}

	if err := queries.CreateContact(contact); err != nil {
		if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, fmt.Errorf("ошибка создания контакта: %w", err)
		}
		// Контакт уже существует
		existingContact, err := queries.GetContactByPeerID(info.ID.String())
		if err != nil {
			return nil, fmt.Errorf("контакт уже существует, но не удалось получить: %w", err)
		}
		contact = existingContact
	}

	return &PeerAddress{
		PeerID:    info.ID.String(),
		Multiaddr: addrStr,
		PublicKey: base64.StdEncoding.EncodeToString(pubKeyBytes),
	}, nil
}

// ConnectToPeer подключается к пиру по адресу
func (n *P2PNetwork) ConnectToPeer(ctx context.Context, addrStr string) error {
	n.mu.RLock()
	host := n.host
	n.mu.RUnlock()

	if host == nil {
		return errors.New("хост не инициализирован")
	}

	addr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("ошибка парсинга адреса: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("ошибка извлечения информации о пире: %w", err)
	}

	if err := host.Connect(ctx, *info); err != nil {
		return fmt.Errorf("ошибка подключения к пиру %s: %w", info.ID, err)
	}

	log.Printf("Подключено к пиру: %s", info.ID.String())
	return nil
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

// loadOrCreateProfile загружает существующий профиль или создаёт новый
func (n *P2PNetwork) loadOrCreateProfile() (*models.P2PProfile, error) {
	// Проверяем существование профиля
	exists, err := queries.P2PProfileExists()
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки профиля: %w", err)
	}

	if exists {
		// Загружаем существующий
		profile, err := queries.GetP2PProfile()
		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки профиля: %w", err)
		}
		log.Printf("Загружен существующий P2P профиль: %s", profile.PeerID)
		return profile, nil
	}

	// Создаём новый профиль
	log.Println("Создание нового P2P профиля...")

	// Генерируем ключи
	privKey, pubKey, err := n.generateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации ключей: %w", err)
	}

	// Получаем PeerID
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения PeerID: %w", err)
	}

	// Сериализуем публичный ключ
	pubKeyBytes, err := crypto.MarshalPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации публичного ключа: %w", err)
	}

	// Шифруем приватный ключ с паролем
	var privKeyBytes []byte
	var isEncrypted bool
	if n.config.MasterPassword != "" {
		privKeyRaw, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("ошибка сериализации приватного ключа: %w", err)
		}
		privKeyBytes, err = p2pcrypto.EncryptPrivateKey(privKeyRaw, n.config.MasterPassword)
		if err != nil {
			return nil, fmt.Errorf("ошибка шифрования приватного ключа: %w", err)
		}
		isEncrypted = true
		log.Println("Приватный ключ зашифрован")
	} else {
		// Без шифрования (не рекомендуется)
		privKeyBytes, err = crypto.MarshalPrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("ошибка сериализации приватного ключа: %w", err)
		}
		isEncrypted = false
		log.Println("Предупреждение: приватный ключ сохранён без шифрования")
	}

	// Получаем имя пользователя из профиля
	username, err := queries.GetProfileUsername()
	if err != nil {
		username = fmt.Sprintf("User_%s", peerID.String()[:8])
	}

	// Создаём профиль
	profile := &models.P2PProfile{
		ID:             1,
		PeerID:         peerID.String(),
		PrivateKey:     privKeyBytes,
		PublicKey:      pubKeyBytes,
		IsKeyEncrypted: isEncrypted,
		Username:       username,
		Status:         "online",
	}

	if err := queries.CreateP2PProfile(profile); err != nil {
		return nil, fmt.Errorf("ошибка сохранения профиля: %w", err)
	}

	log.Printf("Создан новый P2P профиль: %s", profile.PeerID)
	return profile, nil
}

// generateKeyPair генерирует пару ключей Ed25519
func (n *P2PNetwork) generateKeyPair() (crypto.PrivKey, crypto.PubKey, error) {
	privKey, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privKey, pubKey, nil
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

	// Опции хоста
	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(n.config.ListenAddrs...),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithStaticRelays(staticRelays),
		libp2p.UserAgent("ProjectT/1.0"),
	}

	// Создаём хост
	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("ошибка создания хоста: %w", err)
	}

	n.host = h

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
		dht.ProtocolPrefix("/projectt"),
	}

	kdht, err := dht.New(ctx, n.host, dhtOpts...)
	if err != nil {
		return fmt.Errorf("ошибка создания DHT: %w", err)
	}

	n.dht = kdht

	// Подключаемся к bootstrap-узлам
	if err := n.connectToBootstrapPeers(); err != nil {
		log.Printf("Предупреждение: не удалось подключиться к bootstrap-узлам: %v", err)
	}

	// Запускаем обнаружение через DHT
	discovery := routing.NewRoutingDiscovery(kdht)
	_ = discovery // Можно использовать для обнаружения

	log.Println("DHT инициализирована")
	return nil
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

// connectToBootstrapPeers подключается к bootstrap-узлам из БД
func (n *P2PNetwork) connectToBootstrapPeers() error {
	peers, err := queries.GetAllBootstrapPeers()
	if err != nil {
		return fmt.Errorf("ошибка получения bootstrap-узлов: %w", err)
	}

	ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer cancel()

	var connected int
	for _, p := range peers {
		addr, err := multiaddr.NewMultiaddr(p.Multiaddr)
		if err != nil {
			log.Printf("Предупреждение: неверный адрес bootstrap-узла %s: %v", p.Multiaddr, err)
			continue
		}

		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Printf("Предупреждение: неверная информация о пире %s: %v", p.Multiaddr, err)
			continue
		}

		if err := n.host.Connect(ctx, *info); err != nil {
			log.Printf("Предупреждение: не удалось подключиться к bootstrap-узлу %s: %v", p.Multiaddr, err)
			continue
		}

		connected++
		log.Printf("Подключено к bootstrap-узлу: %s", p.Multiaddr)

		// Обновляем время подключения в БД
		_ = queries.UpdateBootstrapPeerLastConnected(p.Multiaddr)
	}

	log.Printf("Подключено %d из %d bootstrap-узлов", connected, len(peers))
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
	// Обработка будет реализована в chat.go
	log.Printf("Получен поток чата от: %s", stream.Conn().RemotePeer().String())
}

// updateProfileAddrs обновляет адреса прослушивания в профиле
func (n *P2PNetwork) updateProfileAddrs() error {
	if n.host == nil {
		return errors.New("хост не инициализирован")
	}

	var addrs []string
	for _, addr := range n.host.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), n.host.ID().String())
		addrs = append(addrs, fullAddr)
	}

	addrsStr := strings.Join(addrs, "|")
	return queries.UpdateP2PProfileField("listen_addrs", addrsStr)
}

// ParsePeerAddressString парсит строку адреса в формате peerid@multiaddr
func ParsePeerAddressString(addrStr string) (*PeerAddress, error) {
	parts := strings.SplitN(addrStr, "@", 2)
	if len(parts) != 2 {
		// Пробуем распарсить как полный multiaddr
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			return nil, errors.New("неверный формат адреса")
		}

		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return nil, errors.New("не удалось извлечь PeerID")
		}

		return &PeerAddress{
			PeerID:    info.ID.String(),
			Multiaddr: addrStr,
		}, nil
	}

	peerID := parts[0]
	ma := parts[1]

	// Валидируем PeerID
	pid, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("неверный PeerID: %w", err)
	}

	// Валидируем multiaddr
	_, err = multiaddr.NewMultiaddr(ma)
	if err != nil {
		return nil, fmt.Errorf("неверный multiaddr: %w", err)
	}

	return &PeerAddress{
		PeerID:    pid.String(),
		Multiaddr: ma,
	}, nil
}

// FormatPeerAddress форматирует адрес для шаринга
func FormatPeerAddress(peerID, multiaddr string) string {
	return fmt.Sprintf("%s@%s", peerID, multiaddr)
}
