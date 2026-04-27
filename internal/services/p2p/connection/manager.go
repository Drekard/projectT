// Package connection предоставляет сервисы для управления подключениями P2P
package connection

import (
	"context"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"

	"projectT/internal/metrics"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// MaxConcurrentConnections максимальное количество одновременных подключений
// Ограничивает потребление ресурсов (память, сеть, CPU)
const MaxConcurrentConnections = 50

// MarkProfilePending отмечает пира как ожидающего обмена профиля
func (s *Service) MarkProfilePending(peerID peer.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingProfile[peerID] = time.Now()
	log.Printf("[Profile] Пир %s отмечен как ожидающий обмена профиля", peerID.String()[:8])
}

// MarkProfileComplete отмечает завершение обмена профиля
func (s *Service) MarkProfileComplete(peerID peer.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pendingProfile, peerID)
	// Обновляем время последнего обмена профиля
	if info, exists := s.peerStatus[peerID]; exists {
		info.LastProfileExch = time.Now()
	}
	log.Printf("[Profile] Обмен профиля с %s завершён", peerID.String()[:8])
}

// CanRequestProfile проверяет, можно ли запросить профиль у пира
// Возвращает false, если профиль уже был получен недавно (менее 5 минут назад)
func (s *Service) CanRequestProfile(peerID peer.ID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if info, exists := s.peerStatus[peerID]; exists {
		// Если профиль ещё не получен - можно запрашивать
		if info.LastProfileExch.IsZero() {
			return true
		}
		// Если прошло меньше 5 минут с последнего обмена - не запрашиваем
		if time.Since(info.LastProfileExch) < 5*time.Minute {
			return false
		}
	}
	return true
}

// initializeConnections инициализирует подключения к известным пирам
func (s *Service) initializeConnections() {
	// Загружаем ВСЕ активные адреса для подключения (bootstrap + contact + discovered)
	var addresses []*models.PeerAddress
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				addresses = nil
				err = nil
			}
		}()
		addresses, err = queries.GetActivePeerAddresses()
	}()

	if addresses == nil && err == nil {
		return
	}

	if err != nil {
		log.Printf("Предупреждение: не удалось загрузить адреса пиров: %v", err)
		return
	}

	log.Printf("[Connection] Загружено %d адресов для подключения", len(addresses))

	// ✅ Проверяем лимит подключений
	connectedCount := s.GetConnectedPeersCount()
	if connectedCount >= MaxConcurrentConnections {
		log.Printf("[Connection] ⚠️ Достигнут лимит подключений: %d/%d", connectedCount, MaxConcurrentConnections)
		return
	}

	// Ограничиваем количество попыток подключения с учётом уже подключённых
	maxToConnect := MaxConcurrentConnections - connectedCount
	log.Printf("[Connection] Максимум подключений: %d (уже подключено: %d)", maxToConnect, connectedCount)

	// Счётчик попыток подключения
	var connectAttempt int

	for _, addr := range addresses {
		// ✅ Ограничиваем количество подключений
		if connectAttempt >= maxToConnect {
			log.Printf("[Connection] ⚠️ Достигнут лимит подключений (%d), остальные адреса в очереди", connectAttempt)
			break
		}

		peerID, err := peer.Decode(addr.PeerID)
		if err != nil {
			log.Printf("Предупреждение: неверный PeerID %s: %v", addr.PeerID, err)
			continue
		}

		// Инициализируем статус пира
		s.peerStatus[peerID] = &PeerConnectionInfo{
			Status:  StatusDisconnected,
			AddedAt: time.Now(),
		}

		multiaddrStr := addr.Multiaddr
		ma, err := multiaddr.NewMultiaddr(multiaddrStr)
		if err != nil {
			log.Printf("Предупреждение: неверный адрес %s: %v", addr.Multiaddr, err)
			continue
		}

		// Добавляем адрес в peerstore
		s.host.Peerstore().AddAddr(peerID, ma, peerstore.PermanentAddrTTL)

		// Увеличиваем счётчик
		connectAttempt++

		// ✅ Автоподключение ко ВСЕМ известным пирам (не только контакты!)
		go func(addr *models.PeerAddress) {
			ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
			defer cancel()

			peerID, err := peer.Decode(addr.PeerID)
			if err != nil {
				return
			}

			if err := s.host.Connect(ctx, peer.AddrInfo{
				ID:    peerID,
				Addrs: []multiaddr.Multiaddr{ma},
			}); err != nil {
				log.Printf("❌ Автоподключение к %s (%s): %v", addr.PeerID[:8], addr.AddressType, err)

				// Если это bootstrap или contact — добавляем в очередь переподключения
				if addr.AddressType == "bootstrap" || addr.AddressType == "contact" {
					s.addToReconnectQueue(peerID)
				}
			} else {
				log.Printf("✅ Автоподключение к %s (%s) успешно", addr.PeerID[:8], addr.AddressType)

				// Обновляем время подключения в БД
				_ = queries.UpdatePeerAddressLastConnected(addr.Multiaddr)
				_ = queries.UpdateProfileLastConnected(addr.PeerID)
			}
		}(addr)
	}

	log.Printf("[Connection] Инициализировано %d известных пиров", len(addresses))
}

// monitorConnections отслеживает активные соединения
func (s *Service) monitorConnections() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkConnections()
		}
	}
}

// checkConnections проверяет статус всех подключений
func (s *Service) checkConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	connectedPeers := s.host.Network().Peers()
	connectedSet := make(map[peer.ID]bool)

	for _, p := range connectedPeers {
		connectedSet[p] = true

		if info, exists := s.peerStatus[p]; exists {
			if info.Status != StatusConnected {
				info.Status = StatusConnected
				info.LastSeen = time.Now()
				go s.updateContactLastSeen(p)
			}
		} else {
			s.peerStatus[p] = &PeerConnectionInfo{
				Status:   StatusConnected,
				LastSeen: time.Now(),
				AddedAt:  time.Now(),
			}
		}
	}

	for peerID, info := range s.peerStatus {
		if !connectedSet[peerID] && info.Status == StatusConnected {
			info.Status = StatusDisconnected
			info.LastSeen = time.Now()
			go s.updateContactLastSeen(peerID)

			if s.isContact(peerID) {
				s.addToReconnectQueue(peerID)
			}
		}
	}

	// Обновляем метрики количества пиров
	if metrics.IsInitialized() {
		m := metrics.Get().Metrics
		m.P2PPeersTotal.Set(float64(len(connectedPeers)))
	}
}

// processReconnectQueue обрабатывает очередь переподключения
func (s *Service) processReconnectQueue() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processNextReconnect()
		}
	}
}

// processNextReconnect обрабатывает следующую попытку переподключения
func (s *Service) processNextReconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.reconnectQueue) == 0 {
		return
	}

	peerID := s.reconnectQueue[0]
	s.reconnectQueue = s.reconnectQueue[1:]

	info, exists := s.peerStatus[peerID]
	if !exists {
		return
	}

	if s.host.Network().Connectedness(peerID) == network.Connected {
		info.Status = StatusConnected
		info.LastSeen = time.Now()
		return
	}

	if info.ReconnectCount >= 5 {
		log.Printf("Превышено количество попыток переподключения к %s (5)", peerID)
		info.Status = StatusDisconnected
		return
	}

	info.Status = StatusReconnecting
	info.ReconnectCount++

	go s.attemptReconnect(peerID)
}

// attemptReconnect пытается переподключиться к пиру
func (s *Service) attemptReconnect(peerID peer.ID) {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	addrs := s.host.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		log.Printf("Нет адресов для переподключения к %s", peerID)
		return
	}

	peerInfo := peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}

	err := s.host.Connect(ctx, peerInfo)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		log.Printf("Переподключение к %s не удалось: %v", peerID, err)
		s.reconnectQueue = append(s.reconnectQueue, peerID)
	} else {
		log.Printf("Переподключение к %s успешно", peerID)
		if info, exists := s.peerStatus[peerID]; exists {
			info.Status = StatusConnected
			info.LastSeen = time.Now()
			info.ReconnectCount = 0

			// Обновляем метрики
			if metrics.IsInitialized() {
				metrics.Get().Metrics.P2PConnectionsTotal.Inc()
			}
		}
	}
}

// addToReconnectQueue добавляет пира в очередь на переподключение
func (s *Service) addToReconnectQueue(peerID peer.ID) {
	for _, p := range s.reconnectQueue {
		if p == peerID {
			return
		}
	}

	s.reconnectQueue = append(s.reconnectQueue, peerID)
	log.Printf("Добавлен в очередь на переподключение: %s", peerID)
}

// isContact проверяет, является ли пир контактом
func (s *Service) isContact(peerID peer.ID) bool {
	contact, err := queries.GetContactByPeerID(peerID.String())
	return err == nil && contact != nil
}

// updateContactLastSeen обновляет время последней активности контакта в БД
func (s *Service) updateContactLastSeen(peerID peer.ID) {
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err != nil || contact == nil {
		return
	}

	now := time.Now()
	_ = queries.UpdateContactLastSeen(contact.ID, &now)
}

// GetConnectionStatus возвращает статус подключения к пиру
func (s *Service) GetConnectionStatus(peerID peer.ID) ConnectionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if info, exists := s.peerStatus[peerID]; exists {
		return info.Status
	}
	return StatusUnknown
}

// GetAllConnectionStatuses возвращает статусы всех пиров
func (s *Service) GetAllConnectionStatuses() map[peer.ID]ConnectionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[peer.ID]ConnectionStatus)
	for pid, info := range s.peerStatus {
		result[pid] = info.Status
	}
	return result
}

// GetPeerInfo возвращает информацию о подключении к пиру
func (s *Service) GetPeerInfo(peerID peer.ID) *PeerConnectionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if info, exists := s.peerStatus[peerID]; exists {
		infoCopy := *info
		return &infoCopy
	}
	return nil
}

// GetConnectedPeersCount возвращает количество подключённых пиров
func (s *Service) GetConnectedPeersCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, info := range s.peerStatus {
		if info.Status == StatusConnected {
			count++
		}
	}
	return count
}
