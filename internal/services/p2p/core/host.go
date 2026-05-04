// Package core предоставляет функции для управления P2P хостом
package core

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
	} else {
		// Ключ не зашифрован
		privKey, err = crypto.UnmarshalPrivateKey(profile.PrivateKey)
		if err != nil {
			return fmt.Errorf("ошибка десериализации приватного ключа: %w", err)
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

	// Формируем адреса для прослушивания с портом из настроек
	listenAddrs := n.config.ListenAddrs
	if n.config.ListenPort > 0 {
		// Если порт задан, используем его
		listenAddrs = []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", n.config.ListenPort),
			fmt.Sprintf("/ip6/::/tcp/%d", n.config.ListenPort),
		}
	}

	// Опции хоста - зависят от настроек!
	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddrs...),
	}

	// NAT Port Mapping - только если включено в настройках
	if n.config.EnableNATPortMap {
		opts = append(opts, libp2p.NATPortMap())
	}

	// Relay - только если включено в настройках
	if n.config.EnableRelay {
		opts = append(opts, libp2p.EnableRelay())
	}

	// AutoRelay - только если включены Relay и AutoRelay
	if n.config.EnableRelay && n.config.EnableAutoRelay {
		opts = append(opts, libp2p.EnableAutoRelayWithStaticRelays(staticRelays))
	}

	// Hole Punching - всегда включён для прямых соединений
	opts = append(opts, libp2p.EnableHolePunching())

	// AutoRelay с сервисом релея - критично для соединения через NAT
	// RelayService позволяет этому узлу выступать как релеи для других
	opts = append(opts, libp2p.EnableRelayService())

	// Дополнительные опции
	opts = append(opts, libp2p.UserAgent("ProjectT/1.0"))

	// Prometheus метрики libp2p (если включено)
	if n.config.EnablePrometheusMetrics && n.prometheusRegistry != nil {
		opts = append(opts, libp2p.PrometheusRegisterer(n.prometheusRegistry))
	}

	// Создаём хост
	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("ошибка создания хоста: %w", err)
	}

	n.host = h
	n.localPrivKey = privKey
	n.localPubKey = pubKey

	// === ЛОГИРОВАНИЕ: Информация о хосте после создания ===
	log.Printf("[P2P/Host] ========================================")
	log.Printf("[P2P/Host] Хост создан, PeerID: %s", h.ID().String())
	log.Printf("[P2P/Host] Слушаемые адреса (ListenAddrs):")
	for _, addr := range h.Addrs() {
		log.Printf("[P2P/Host]   - %s/p2p/%s", addr.String(), h.ID().String())
	}

	// Проверяем наличие публичных адресов
	hasPublicAddr := false
	for _, addr := range h.Addrs() {
		if isPublicMultiaddr(addr) {
			hasPublicAddr = true
			log.Printf("[P2P/Host]   ✅ Обнаружен публичный адрес: %s", addr.String())
		}
	}
	if !hasPublicAddr {
		log.Printf("[P2P/Host]   ⚠️ Публичные адреса не обнаружены (возможно, за NAT)")
	}
	log.Printf("[P2P/Host] NAT Port Map: %v", n.config.EnableNATPortMap)
	log.Printf("[P2P/Host] Relay: %v, AutoRelay: %v", n.config.EnableRelay, n.config.EnableAutoRelay)
	log.Printf("[P2P/Host] Hole Punching: enabled (always)")
	log.Printf("[P2P/Host] RelayService: enabled (always)")
	log.Printf("[P2P/Host] Bootstrap релеи: %d загружено", len(staticRelays))
	for _, r := range staticRelays {
		log.Printf("[P2P/Host]   - Релей: %s", r.ID.String())
	}
	log.Printf("[P2P/Host] ========================================")

	// Инициализируем DHT
	if n.config.EnableDHT {
		_ = n.initDHT()
	}

	// Инициализируем PubSub
	_ = n.initPubSub()

	// Инициализируем ChatService
	_ = n.initChat()

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
	return nil
}

// isPublicMultiaddr проверяет, является ли multiaddr публичным
func isPublicMultiaddr(addr multiaddr.Multiaddr) bool {
	ipStr, err := addr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		ipStr, err = addr.ValueForProtocol(multiaddr.P_IP6)
		if err != nil {
			return false
		}
	}
	return !isPrivateIP(ipStr)
}

// isPrivateIP проверяет, является ли IP частным
func isPrivateIP(ip string) bool {
	if len(ip) >= 8 && ip[:8] == "192.168." {
		return true
	}
	if len(ip) >= 3 && ip[:3] == "10." {
		return true
	}
	if len(ip) >= 7 && (ip[:7] == "172.16." || ip[:7] == "172.17." || ip[:7] == "172.18." ||
		ip[:7] == "172.19." || ip[:7] == "172.20." || ip[:7] == "172.21." ||
		ip[:7] == "172.22." || ip[:7] == "172.23." || ip[:7] == "172.24." ||
		ip[:7] == "172.25." || ip[:7] == "172.26." || ip[:7] == "172.27." ||
		ip[:7] == "172.28." || ip[:7] == "172.29." || ip[:7] == "172.30." ||
		ip[:7] == "172.31.") {
		return true
	}
	if ip == "127.0.0.1" || (len(ip) >= 4 && ip[:4] == "127.") {
		return true
	}
	if ip == "0.0.0.0" {
		return true
	}
	return false
}
