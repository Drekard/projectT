// Package discovery предоставляет сервисы для обнаружения пиров
package discovery

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"

	"projectT/internal/services/p2p"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ProtocolID идентификатор протокола
const ProtocolID = "/projectt/1.0.0"

// MDNSService интерфейс для mDNS сервиса (для возможности мокирования в тестах)
type MDNSService interface {
	Close() error
}

// mdnsNotifee интерфейс для обработчика обнаружения пиров
type MDNSNotifee interface {
	HandlePeerFound(peer.AddrInfo)
}

// DiscoveryService сервис для обнаружения пиров
type DiscoveryService struct {
	host            host.Host
	dht             *routing.RoutingDiscovery
	mdnsService     MDNSService
	config          *p2p.P2PConfig
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	discoveredPeers map[string]time.Time // map[peerID]lastSeen
	peerAddresses   []peer.AddrInfo      // Адреса всех известных пиров для подключения
}

// NewDiscoveryService создаёт сервис обнаружения пиров
func NewDiscoveryService(host host.Host, dht *routing.RoutingDiscovery, config *p2p.P2PConfig) *DiscoveryService {
	ctx, cancel := context.WithCancel(context.Background())
	return &DiscoveryService{
		host:            host,
		dht:             dht,
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		discoveredPeers: make(map[string]time.Time),
		peerAddresses:   []peer.AddrInfo{},
	}
}

// Start запускает все сервисы обнаружения
func (ds *DiscoveryService) Start() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	_ = ds.loadPeerAddresses()

	// Подключение к известным пирам происходит через connection.Service и autodial
	// (избегаем дублирования и блокировки старта)

	// Запускаем mDNS обнаружение если включено (только локальная сеть)
	if ds.config.EnableMDNS {
		_ = ds.startMDNSDiscovery()
	}

	// Запускаем DHT обнаружение для глобальной сети
	if ds.config.EnableDHT && ds.dht != nil {
		ds.startDHTDiscovery()
	}

	return nil
}

// StartDiscovery запускает обнаружение пиров (DHT + bootstrap) по запросу пользователя
func (ds *DiscoveryService) StartDiscovery() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Подключаемся к bootstrap-узлам
	if err := ds.connectToKnownPeers(); err != nil {
		log.Printf("Предупреждение: не удалось подключиться к bootstrap-узлам: %v", err)
	}

	// Запускаем DHT обнаружение
	if ds.config.EnableDHT && ds.dht != nil {
		ds.startDHTDiscovery()
	}

	return nil
}

// Stop останавливает сервис обнаружения
func (ds *DiscoveryService) Stop() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.cancel()

	// Останавливаем mDNS сервис
	if ds.mdnsService != nil {
		if err := ds.mdnsService.Close(); err != nil {
			log.Printf("Предупреждение: ошибка остановки mDNS: %v", err)
		}
	}

	return nil
}

// loadPeerAddresses загружает все адреса пиров из базы данных
func (ds *DiscoveryService) loadPeerAddresses() error {
	// Оборачиваем весь вызов в recover для обработки паники при доступе к nil БД
	var addresses []*models.PeerAddress
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				// БД не инициализирована - пропускаем загрузку
				addresses = nil
				err = nil
			}
		}()

		addresses, err = queries.GetActivePeerAddresses()
	}()

	// Если паника произошла, addresses будет nil
	if addresses == nil && err == nil {
		ds.peerAddresses = []peer.AddrInfo{}
		return nil
	}

	if err != nil {
		return fmt.Errorf("[discovery/service.go] ошибка получения адресов пиров: %w", err)
	}

	ds.peerAddresses = make([]peer.AddrInfo, 0, len(addresses))
	for _, addr := range addresses {
		ma, err := multiaddr.NewMultiaddr(addr.Multiaddr)
		if err != nil {
			continue
		}

		info, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			continue
		}

		ds.peerAddresses = append(ds.peerAddresses, *info)
	}
	return nil
}

// connectToKnownPeers подключается ко всем известным пирам (ОДНОКРАТНО при запуске)
func (ds *DiscoveryService) connectToKnownPeers() error {
	if len(ds.peerAddresses) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ds.ctx, 30*time.Second)
	defer cancel()

	var connected int
	for _, peerInfo := range ds.peerAddresses {
		startTime := time.Now()
		if err := ds.host.Connect(ctx, peerInfo); err != nil {
			elapsed := time.Since(startTime)
			_ = elapsed
			continue
		}

		elapsed := time.Since(startTime)
		connected++
		_ = elapsed

		conns := ds.host.Network().ConnsToPeer(peerInfo.ID)
		for j, conn := range conns {
			remoteAddr := conn.RemoteMultiaddr()
			_ = j
			_ = remoteAddr
		}

		for _, addr := range peerInfo.Addrs {
			_ = queries.UpdatePeerAddressLastConnected(addr.String())
			_ = queries.UpdateProfileLastConnected(peerInfo.ID.String())
		}
	}
	_ = connected
	return nil
}

// startMDNSDiscovery запускает mDNS обнаружение для локальной сети
func (ds *DiscoveryService) startMDNSDiscovery() error {
	return nil
}

// handleDiscoveredPeer обрабатывает обнаруженного пира
func (ds *DiscoveryService) handleDiscoveredPeer(peerInfo peer.AddrInfo) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	peerID := peerInfo.ID.String()

	// Проверяем, не обрабатывали ли уже этого пира недавно
	if lastSeen, exists := ds.discoveredPeers[peerID]; exists {
		if time.Since(lastSeen) < 5*time.Minute {
			return
		}
	}

	// Добавляем в peerstore
	ds.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, 10*time.Minute)

	// Обновляем время последнего обнаружения
	ds.discoveredPeers[peerID] = time.Now()

	// Проверяем, есть ли пир в контактах
	contact, err := queries.GetContactByPeerID(peerID)
	if err == nil && contact != nil {
		go ds.connectToDiscoveredPeer(peerInfo, contact.ID)
	} else {
		log.Printf("Обнаружен новый пир: %s", peerID)
	}
}

// startDHTDiscovery запускает DHT обнаружение для глобальной сети
func (ds *DiscoveryService) startDHTDiscovery() {
	go ds.runDHTDiscovery()
}

// runDHTDiscovery выполняет периодическое DHT обнаружение
func (ds *DiscoveryService) runDHTDiscovery() {
	// ✅ Интервал увеличен до 5 минут для снижения сетевого трафика
	// Было: 30 секунд (избыточно для стабильной сети)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			peers, err := ds.discoverDHTPeers()
			if err != nil {
				continue
			}

			for _, p := range peers {
				ds.handleDiscoveredPeer(p)
			}
		}
	}
}

// discoverDHTPeers ищет пиров через DHT
func (ds *DiscoveryService) discoverDHTPeers() ([]peer.AddrInfo, error) {
	ctx, cancel := context.WithTimeout(ds.ctx, 10*time.Second)
	defer cancel()

	_, err := ds.dht.Advertise(ctx, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка рекламы в DHT: %w", err)
	}

	peersChan, err := ds.dht.FindPeers(ctx, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска пиров в DHT: %w", err)
	}

	var discovered []peer.AddrInfo
	for p := range peersChan {
		if p.ID != ds.host.ID() && len(p.Addrs) > 0 {
			discovered = append(discovered, p)
		}
	}

	return discovered, nil
}

// connectToDiscoveredPeer пытается подключиться к обнаруженному пиру
func (ds *DiscoveryService) connectToDiscoveredPeer(peerInfo peer.AddrInfo, contactID int) {
	ctx, cancel := context.WithTimeout(ds.ctx, 10*time.Second)
	defer cancel()

	if err := ds.host.Connect(ctx, peerInfo); err != nil {
		log.Printf("Не удалось подключиться к пиру %s: %v", peerInfo.ID, err)
		return
	}

	log.Printf("Подключено к пиру из контактов: %s", peerInfo.ID)

	now := time.Now()
	_ = queries.UpdateContactLastSeen(contactID, &now)
}

// AddBootstrapPeer добавляет bootstrap-узел в БД
func (ds *DiscoveryService) AddBootstrapPeer(addrStr string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	exists, err := queries.PeerAddressExists(addrStr)
	if err != nil {
		return fmt.Errorf("ошибка проверки адреса: %w", err)
	}
	if exists {
		return fmt.Errorf("адрес уже существует")
	}

	addr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("ошибка парсинга адреса: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("ошибка извлечения PeerID: %w", err)
	}

	peerIDStr := info.ID.String()
	username := peerIDStr[:8]

	if err := queries.AddPeerAddressWithProfile(peerIDStr, addrStr, "bootstrap", "user_added", username); err != nil {
		return fmt.Errorf("ошибка добавления адреса: %w", err)
	}

	_ = ds.loadPeerAddresses()

	log.Printf("Добавлен bootstrap-пир: %s", addrStr)
	return nil
}

// RemoveBootstrapPeer удаляет bootstrap-узел из БД
func (ds *DiscoveryService) RemoveBootstrapPeer(addrStr string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if err := queries.DeletePeerAddress(addrStr); err != nil {
		return fmt.Errorf("ошибка удаления адреса: %w", err)
	}

	_ = ds.loadPeerAddresses()

	log.Printf("Удалён bootstrap-пир: %s", addrStr)
	return nil
}

// GetBootstrapPeers возвращает список bootstrap-узлов
func (ds *DiscoveryService) GetBootstrapPeers() ([]*models.PeerAddress, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return queries.GetBootstrapAddresses()
}

// GetDiscoveredPeers возвращает список обнаруженных пиров
func (ds *DiscoveryService) GetDiscoveredPeers() map[string]time.Time {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	result := make(map[string]time.Time)
	for k, v := range ds.discoveredPeers {
		result[k] = v
	}
	return result
}

// ClearDiscoveredPeers очищает кэш обнаруженных пиров
func (ds *DiscoveryService) ClearDiscoveredPeers() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.discoveredPeers = make(map[string]time.Time)
}
