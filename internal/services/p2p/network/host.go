// Package network предоставляет функции для управления P2P хостом
package network

import (
	"errors"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"

	p2pcrypto "projectT/internal/services/crypto"
	p2p "projectT/internal/services/p2p"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

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
