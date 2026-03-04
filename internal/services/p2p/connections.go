// Package p2p содержит сервисы для P2P связи на базе libp2p.
package p2p

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ConnectionStatus статус подключения к пиру
type ConnectionStatus string

const (
	// StatusConnected пир подключён
	StatusConnected ConnectionStatus = "connected"
	// StatusDisconnected пир отключён
	StatusDisconnected ConnectionStatus = "disconnected"
	// StatusReconnecting идёт переподключение
	StatusReconnecting ConnectionStatus = "reconnecting"
	// StatusUnknown статус неизвестен
	StatusUnknown ConnectionStatus = "unknown"
)

// PingProtocolID идентификатор протокола ping
const PingProtocolID = "/" + ProtocolPrefix + "/ping/1.0.0"

// ConnectionService сервис мониторинга соединений
type ConnectionService struct {
	host           host.Host
	config         *P2PConfig
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	peerStatus     map[peer.ID]*PeerConnectionInfo // статус пиров
	reconnectQueue []peer.ID                       // очередь на переподключение
	keepAliveFail  map[peer.ID]int                 // счётчик неудачных ping
}

// PeerConnectionInfo информация о подключении к пиру
type PeerConnectionInfo struct {
	Status          ConnectionStatus
	LastSeen        time.Time
	LastPing        time.Time
	LastPingLatency time.Duration
	ReconnectCount  int
	AddedAt         time.Time
}

// NewConnectionService создаёт сервис мониторинга соединений
func NewConnectionService(host host.Host, config *P2PConfig) *ConnectionService {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConnectionService{
		host:           host,
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
		peerStatus:     make(map[peer.ID]*PeerConnectionInfo),
		reconnectQueue: make([]peer.ID, 0),
		keepAliveFail:  make(map[peer.ID]int),
	}
}

// Start запускает мониторинг соединений
func (cs *ConnectionService) Start() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Устанавливаем обработчик stream для ping
	cs.host.SetStreamHandler(PingProtocolID, cs.handlePing)

	// Подключаемся к существующим пирам
	cs.initializeConnections()

	// Запускаем мониторинг
	go cs.monitorConnections()

	// Запускаем KeepAlive
	go cs.startKeepAlive()

	// Запускаем обработчик очереди переподключения
	go cs.processReconnectQueue()

	log.Println("ConnectionService запущен")
	return nil
}

// Stop останавливает сервис
func (cs *ConnectionService) Stop() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.cancel()
	log.Println("ConnectionService остановлен")
	return nil
}

// initializeConnections инициализирует подключения к известным пирам
func (cs *ConnectionService) initializeConnections() {
	// Получаем все контакты из БД
	// Оборачиваем вызов в recover для обработки паники при доступе к nil БД
	var contacts []*models.Contact
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				// БД не инициализирована - пропускаем загрузку контактов
				contacts = nil
				err = nil
			}
		}()

		contacts, err = queries.GetAllContacts()
	}()

	// Если паника произошла, contacts будет nil
	if contacts == nil && err == nil {
		return
	}

	if err != nil {
		log.Printf("Предупреждение: не удалось загрузить контакты: %v", err)
		return
	}

	for _, contact := range contacts {
		if contact.Multiaddr == "" {
			continue
		}

		peerID, err := peer.Decode(contact.PeerID)
		if err != nil {
			log.Printf("Предупреждение: неверный PeerID контакта %s: %v", contact.PeerID, err)
			continue
		}

		// Инициализируем статус
		cs.peerStatus[peerID] = &PeerConnectionInfo{
			Status:  StatusDisconnected,
			AddedAt: time.Now(),
		}

		// Добавляем адрес в peerstore
		addr, err := parseMultiaddr(contact.Multiaddr)
		if err != nil {
			continue
		}
		cs.host.Peerstore().AddAddr(peerID, addr, peerstore.PermanentAddrTTL)
	}

	log.Printf("Инициализировано %d контактов", len(cs.peerStatus))
}

// monitorConnections отслеживает активные соединения
func (cs *ConnectionService) monitorConnections() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			cs.checkConnections()
		}
	}
}

// checkConnections проверяет статус всех подключений
func (cs *ConnectionService) checkConnections() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	connectedPeers := cs.host.Network().Peers()
	connectedSet := make(map[peer.ID]bool)

	// Отмечаем подключённых пиров
	for _, p := range connectedPeers {
		connectedSet[p] = true

		if info, exists := cs.peerStatus[p]; exists {
			if info.Status != StatusConnected {
				info.Status = StatusConnected
				info.LastSeen = time.Now()
				log.Printf("Пир подключён: %s", p)

				// Обновляем статус в БД
				go cs.updateContactStatus(p, "online")
			}
		} else {
			// Новый пир
			cs.peerStatus[p] = &PeerConnectionInfo{
				Status:   StatusConnected,
				LastSeen: time.Now(),
				AddedAt:  time.Now(),
			}
		}
	}

	// Проверяем отключённых пиров
	for peerID, info := range cs.peerStatus {
		if !connectedSet[peerID] && info.Status == StatusConnected {
			info.Status = StatusDisconnected
			info.LastSeen = time.Now()
			log.Printf("Пир отключён: %s", peerID)

			// Обновляем статус в БД
			go cs.updateContactStatus(peerID, "offline")

			// Добавляем в очередь на переподключение (если это контакт)
			if cs.isContact(peerID) {
				cs.addToReconnectQueue(peerID)
			}
		}
	}
}

// startKeepAlive запускает периодическую отправку ping
func (cs *ConnectionService) startKeepAlive() {
	ticker := time.NewTicker(cs.config.KeepAlive)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			cs.sendKeepAlive()
		}
	}
}

// sendKeepAlive отправляет ping всем подключённым пирам
func (cs *ConnectionService) sendKeepAlive() {
	cs.mu.RLock()
	peers := cs.host.Network().Peers()
	cs.mu.RUnlock()

	for _, peerID := range peers {
		go cs.pingPeer(peerID)
	}
}

// pingPeer отправляет ping конкретному пиру
func (cs *ConnectionService) pingPeer(peerID peer.ID) {
	ctx, cancel := context.WithTimeout(cs.ctx, 10*time.Second)
	defer cancel()

	// Создаём стрим для ping
	stream, err := cs.host.NewStream(ctx, peerID, PingProtocolID)
	if err != nil {
		cs.handlePingFailure(peerID, fmt.Errorf("не удалось создать стрим: %w", err))
		return
	}
	defer stream.Close()

	// Отправляем "ping"
	startTime := time.Now()
	_, err = stream.Write([]byte("ping"))
	if err != nil {
		cs.handlePingFailure(peerID, fmt.Errorf("ошибка записи: %w", err))
		return
	}

	// Читаем "pong"
	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		cs.handlePingFailure(peerID, fmt.Errorf("ошибка установки таймаута: %w", err))
		return
	}
	response := make([]byte, 4)
	n, err := stream.Read(response)
	latency := time.Since(startTime)

	if err != nil || n != 4 || string(response) != "pong" {
		cs.handlePingFailure(peerID, fmt.Errorf("ошибка чтения или неверный ответ: %w", err))
		return
	}

	// Успешный ping
	cs.handlePingSuccess(peerID, latency)
}

// handlePingSuccess обрабатывает успешный ping
func (cs *ConnectionService) handlePingSuccess(peerID peer.ID, latency time.Duration) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Сбрасываем счётчик неудач
	delete(cs.keepAliveFail, peerID)

	// Обновляем информацию
	if info, exists := cs.peerStatus[peerID]; exists {
		info.LastPing = time.Now()
		info.LastPingLatency = latency
		log.Printf("KeepAlive: %s - latency: %v", peerID, latency)
	}
}

// handlePingFailure обрабатывает неудачный ping
func (cs *ConnectionService) handlePingFailure(peerID peer.ID, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Увеличиваем счётчик неудач
	cs.keepAliveFail[peerID]++
	failCount := cs.keepAliveFail[peerID]

	log.Printf("KeepAlive failed для %s (попытка %d/3): %v", peerID, failCount, err)

	// Если 3 неудачи подряд - помечаем как offline
	if failCount >= 3 {
		if info, exists := cs.peerStatus[peerID]; exists {
			info.Status = StatusDisconnected
			info.LastSeen = time.Now()
			log.Printf("Пир %s помечен как offline (3 неудачных ping)", peerID)

			// Обновляем статус в БД
			go cs.updateContactStatus(peerID, "offline")
		}

		// Сбрасываем счётчик
		delete(cs.keepAliveFail, peerID)

		// Добавляем в очередь на переподключение
		if cs.isContact(peerID) {
			cs.addToReconnectQueue(peerID)
		}
	}
}

// handlePing обрабатывает входящий ping
func (cs *ConnectionService) handlePing(stream network.Stream) {
	defer stream.Close()

	// Читаем "ping"
	reader := bufio.NewReader(stream)
	request, err := reader.ReadString('\n')
	if err != nil {
		// Пробуем прочитать без \n
		request = "ping"
	}

	// Если получили "ping" - отвечаем "pong"
	if len(request) >= 4 && request[:4] == "ping" {
		if _, err := stream.Write([]byte("pong")); err != nil {
			log.Printf("Ошибка записи pong: %v", err)
		}
	}
}

// processReconnectQueue обрабатывает очередь переподключения
func (cs *ConnectionService) processReconnectQueue() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			cs.processNextReconnect()
		}
	}
}

// processNextReconnect обрабатывает следующую попытку переподключения
func (cs *ConnectionService) processNextReconnect() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if len(cs.reconnectQueue) == 0 {
		return
	}

	// Берём первый элемент из очереди
	peerID := cs.reconnectQueue[0]
	cs.reconnectQueue = cs.reconnectQueue[1:]

	info, exists := cs.peerStatus[peerID]
	if !exists {
		return
	}

	// Проверяем, не подключён ли уже
	if cs.host.Network().Connectedness(peerID) == network.Connected {
		info.Status = StatusConnected
		info.LastSeen = time.Now()
		return
	}

	// Проверяем лимит попыток
	if info.ReconnectCount >= 5 {
		log.Printf("Превышено количество попыток переподключения к %s (5)", peerID)
		info.Status = StatusDisconnected
		return
	}

	// Пытаемся подключиться
	info.Status = StatusReconnecting
	info.ReconnectCount++

	go cs.attemptReconnect(peerID)
}

// attemptReconnect пытается переподключиться к пиру
func (cs *ConnectionService) attemptReconnect(peerID peer.ID) {
	ctx, cancel := context.WithTimeout(cs.ctx, 30*time.Second)
	defer cancel()

	// Получаем адреса из peerstore
	addrs := cs.host.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		log.Printf("Нет адресов для переподключения к %s", peerID)
		return
	}

	// Создаём PeerInfo
	peerInfo := peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}

	// Подключаемся
	err := cs.host.Connect(ctx, peerInfo)

	cs.mu.Lock()
	defer cs.mu.Unlock()

	if err != nil {
		log.Printf("Переподключение к %s не удалось: %v", peerID, err)
		// Возвращаем в очередь
		cs.reconnectQueue = append(cs.reconnectQueue, peerID)
	} else {
		log.Printf("Переподключение к %s успешно", peerID)
		if info, exists := cs.peerStatus[peerID]; exists {
			info.Status = StatusConnected
			info.LastSeen = time.Now()
			info.ReconnectCount = 0
		}
	}
}

// addToReconnectQueue добавляет пира в очередь на переподключение
func (cs *ConnectionService) addToReconnectQueue(peerID peer.ID) {
	// Проверяем, нет ли уже в очереди
	for _, p := range cs.reconnectQueue {
		if p == peerID {
			return
		}
	}

	cs.reconnectQueue = append(cs.reconnectQueue, peerID)
	log.Printf("Добавлен в очередь на переподключение: %s", peerID)
}

// isContact проверяет, является ли пир контактом
func (cs *ConnectionService) isContact(peerID peer.ID) bool {
	contact, err := queries.GetContactByPeerID(peerID.String())
	return err == nil && contact != nil
}

// updateContactStatus обновляет статус контакта в БД
func (cs *ConnectionService) updateContactStatus(peerID peer.ID, status string) {
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err != nil || contact == nil {
		return
	}

	var lastSeen *time.Time
	if status == "offline" {
		now := time.Now()
		lastSeen = &now
	}

	_ = queries.UpdateContactStatus(contact.ID, status, lastSeen)
}

// GetConnectionStatus возвращает статус подключения к пиру
func (cs *ConnectionService) GetConnectionStatus(peerID peer.ID) ConnectionStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if info, exists := cs.peerStatus[peerID]; exists {
		return info.Status
	}
	return StatusUnknown
}

// GetAllConnectionStatuses возвращает статусы всех пиров
func (cs *ConnectionService) GetAllConnectionStatuses() map[peer.ID]ConnectionStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make(map[peer.ID]ConnectionStatus)
	for pid, info := range cs.peerStatus {
		result[pid] = info.Status
	}
	return result
}

// GetPeerInfo возвращает информацию о подключении к пиру
func (cs *ConnectionService) GetPeerInfo(peerID peer.ID) *PeerConnectionInfo {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if info, exists := cs.peerStatus[peerID]; exists {
		// Возвращаем копию
		infoCopy := *info
		return &infoCopy
	}
	return nil
}

// GetConnectedPeersCount возвращает количество подключённых пиров
func (cs *ConnectionService) GetConnectedPeersCount() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	count := 0
	for _, info := range cs.peerStatus {
		if info.Status == StatusConnected {
			count++
		}
	}
	return count
}

// parseMultiaddr парсит multiaddr из строки
func parseMultiaddr(addrStr string) (multiaddr.Multiaddr, error) {
	return multiaddr.NewMultiaddr(addrStr)
}
