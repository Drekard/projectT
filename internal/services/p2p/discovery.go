package p2p

// Package p2p содержит сервисы для P2P связи на базе libp2p.

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

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
	config          *P2PConfig
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	discoveredPeers map[string]time.Time // map[peerID]lastSeen
	bootstrapPeers  []peer.AddrInfo
}

// NewDiscoveryService создаёт сервис обнаружения пиров
func NewDiscoveryService(host host.Host, dht *routing.RoutingDiscovery, config *P2PConfig) *DiscoveryService {
	ctx, cancel := context.WithCancel(context.Background())
	return &DiscoveryService{
		host:            host,
		dht:             dht,
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		discoveredPeers: make(map[string]time.Time),
		bootstrapPeers:  []peer.AddrInfo{},
	}
}

// Start запускает все сервисы обнаружения
func (ds *DiscoveryService) Start() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Загружаем bootstrap-пиры из БД
	if err := ds.loadBootstrapPeers(); err != nil {
		log.Printf("Предупреждение: не удалось загрузить bootstrap-пиры: %v", err)
	}

	// НЕ подключаемся к bootstrap-узлам автоматически - только по запросу пользователя
	// if err := ds.connectToBootstrapPeers(); err != nil {
	// 	log.Printf("Предупреждение: не удалось подключиться к bootstrap-узлам: %v", err)
	// }

	// Запускаем mDNS обнаружение если включено (только локальная сеть)
	if ds.config.EnableMDNS {
		if err := ds.startMDNSDiscovery(); err != nil {
			log.Printf("Предупреждение: mDNS не инициализирован: %v", err)
		} else {
			log.Println("mDNS обнаружение запущено")
		}
	}

	// НЕ запускаем DHT обнаружение автоматически - только по запросу пользователя
	// if ds.config.EnableDHT && ds.dht != nil {
	// 	ds.startDHTDiscovery()
	// }

	log.Println("Сервис обнаружения запущен (ожидание ручного подключения)")
	return nil
}

// StartDiscovery запускает обнаружение пиров (DHT + bootstrap) по запросу пользователя
func (ds *DiscoveryService) StartDiscovery() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Подключаемся к bootstrap-узлам
	if err := ds.connectToBootstrapPeers(); err != nil {
		log.Printf("Предупреждение: не удалось подключиться к bootstrap-узлам: %v", err)
	}

	// Запускаем DHT обнаружение
	if ds.config.EnableDHT && ds.dht != nil {
		ds.startDHTDiscovery()
		log.Println("DHT обнаружение запущено")
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

	log.Println("Сервис обнаружения остановлен")
	return nil
}

// loadBootstrapPeers загружает bootstrap-пиры из базы данных
func (ds *DiscoveryService) loadBootstrapPeers() error {
	// Оборачиваем весь вызов в recover для обработки паники при доступе к nil БД
	var peers []*models.BootstrapPeer
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				// БД не инициализирована - пропускаем загрузку bootstrap пиров
				peers = nil
				err = nil
			}
		}()

		peers, err = queries.GetActiveBootstrapPeers()
	}()

	// Если паника произошла, peers будет nil
	if peers == nil && err == nil {
		ds.bootstrapPeers = []peer.AddrInfo{}
		return nil
	}

	if err != nil {
		return fmt.Errorf("ошибка получения bootstrap-пиров: %w", err)
	}

	ds.bootstrapPeers = make([]peer.AddrInfo, 0, len(peers))
	for _, p := range peers {
		addr, err := multiaddr.NewMultiaddr(p.Multiaddr)
		if err != nil {
			log.Printf("Предупреждение: неверный адрес bootstrap-пира %s: %v", p.Multiaddr, err)
			continue
		}

		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Printf("Предупреждение: неверная информация о bootstrap-пире %s: %v", p.Multiaddr, err)
			continue
		}

		ds.bootstrapPeers = append(ds.bootstrapPeers, *info)
	}

	log.Printf("Загружено %d bootstrap-пиров", len(ds.bootstrapPeers))
	return nil
}

// connectToBootstrapPeers подключается к bootstrap-узлам
func (ds *DiscoveryService) connectToBootstrapPeers() error {
	if len(ds.bootstrapPeers) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ds.ctx, 30*time.Second)
	defer cancel()

	var connected int
	for _, peerInfo := range ds.bootstrapPeers {
		if err := ds.host.Connect(ctx, peerInfo); err != nil {
			log.Printf("Предупреждение: не удалось подключиться к bootstrap-пиру %s: %v", peerInfo.ID, err)
			continue
		}

		connected++
		log.Printf("Подключено к bootstrap-пиру: %s", peerInfo.ID)

		// Обновляем время подключения в БД
		for _, addr := range peerInfo.Addrs {
			_ = queries.UpdateBootstrapPeerLastConnected(addr.String())
		}
	}

	log.Printf("Подключено %d из %d bootstrap-пиров", connected, len(ds.bootstrapPeers))
	return nil
}

// startMDNSDiscovery запускает mDNS обнаружение для локальной сети
// Примечание: go-libp2p-mdns был заархивирован.
// В будущих версиях будет использован новый подход через zeroconf или кастомную реализацию.
// Пока mDNS не доступен, используем только DHT discovery.
func (ds *DiscoveryService) startMDNSDiscovery() error {
	// TODO: Реализовать mDNS через zeroconf/v2 или другой доступный механизм
	// Для локального тестирования используйте DHT discovery или прямое подключение

	log.Println("mDNS временно недоступен - используется только DHT discovery")
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
			// Уже видели этого пира недавно
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
		// Пир в контактах - пробуем подключиться
		go ds.connectToDiscoveredPeer(peerInfo, contact.ID)
	} else {
		log.Printf("Обнаружен новый пир: %s", peerID)
	}
}

// startDHTDiscovery запускает DHT обнаружение для глобальной сети
func (ds *DiscoveryService) startDHTDiscovery() {
	go ds.runDHTDiscovery()
	log.Println("DHT обнаружение запущено")
}

// runDHTDiscovery выполняет периодическое DHT обнаружение
func (ds *DiscoveryService) runDHTDiscovery() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			// Ищем пиров через DHT
			peers, err := ds.discoverDHTPeers()
			if err != nil {
				log.Printf("Ошибка DHT обнаружения: %v", err)
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

	// Рекламируем наш сервис в DHT и ищем других пиров
	_, err := ds.dht.Advertise(ctx, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка рекламы в DHT: %w", err)
	}

	// Ищем другие пиры с нашим protocol ID
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

	// Обновляем статус контакта
	now := time.Now()
	_ = queries.UpdateContactStatus(contactID, "online", &now)
}

// AddBootstrapPeer добавляет bootstrap-узел в БД
func (ds *DiscoveryService) AddBootstrapPeer(addrStr string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Проверяем, существует ли уже
	exists, err := queries.BootstrapPeerExists(addrStr)
	if err != nil {
		return fmt.Errorf("ошибка проверки bootstrap-пира: %w", err)
	}
	if exists {
		return fmt.Errorf("bootstrap-пир уже существует")
	}

	// Парсим адрес для извлечения PeerID
	var peerID sql.NullString
	addr, err := multiaddr.NewMultiaddr(addrStr)
	if err == nil {
		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err == nil {
			peerID = sql.NullString{String: info.ID.String(), Valid: true}
		}
	}

	// Создаём запись в БД
	bootstrapPeer := &models.BootstrapPeer{
		Multiaddr: addrStr,
		PeerID:    peerID,
		IsActive:  true,
	}

	if err := queries.CreateBootstrapPeer(bootstrapPeer); err != nil {
		return fmt.Errorf("ошибка создания bootstrap-пира: %w", err)
	}

	// Перезагружаем bootstrap-пиры
	_ = ds.loadBootstrapPeers()

	log.Printf("Добавлен bootstrap-пир: %s", addrStr)
	return nil
}

// RemoveBootstrapPeer удаляет bootstrap-узел из БД
func (ds *DiscoveryService) RemoveBootstrapPeer(addrStr string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if err := queries.DeleteBootstrapPeerByMultiaddr(addrStr); err != nil {
		return fmt.Errorf("ошибка удаления bootstrap-пира: %w", err)
	}

	// Перезагружаем bootstrap-пиры
	_ = ds.loadBootstrapPeers()

	log.Printf("Удалён bootstrap-пир: %s", addrStr)
	return nil
}

// GetBootstrapPeers возвращает список bootstrap-узлов
func (ds *DiscoveryService) GetBootstrapPeers() ([]*models.BootstrapPeer, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return queries.GetAllBootstrapPeers()
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
