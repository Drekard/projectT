// Package connection предоставляет сервисы для управления подключениями P2P
package connection

import (
	"context"
	"log"
	"strings"
	"time"

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

// initializeConnections инициализирует подключения к известным пирам (ОДНОКРАТНО при запуске)
func (s *Service) initializeConnections() {
	log.Printf("[connection/manager.go] ========================================")
	log.Printf("[connection/manager.go] ЗАПУСК: Однократное подключение к известным пирам")
	log.Printf("[connection/manager.go] ========================================")

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
		log.Printf("[connection/manager.go] Нет известных пиров в БД, подключение не выполняется")
		return
	}

	if err != nil {
		log.Printf("[connection/manager.go] Предупреждение: не удалось загрузить адреса пиров: %v", err)
		return
	}

	log.Printf("[connection/manager.go] Загружено %d адресов из БД для подключения", len(addresses))

	// Проверяем лимит подключений
	connectedCount := s.GetConnectedPeersCount()
	if connectedCount >= MaxConcurrentConnections {
		log.Printf("[connection/manager.go] Достигнут лимит подключений: %d/%d", connectedCount, MaxConcurrentConnections)
		return
	}

	// Ограничиваем количество попыток подключения с учётом уже подключённых
	maxToConnect := MaxConcurrentConnections - connectedCount
	log.Printf("[connection/manager.go] Лимит: %d одновременных подключений, уже подключено: %d, можно подключить: %d",
		MaxConcurrentConnections, connectedCount, maxToConnect)

	// Счётчик попыток подключения
	var connectAttempt int

	for i, addr := range addresses {
		// Ограничиваем количество подключений
		if connectAttempt >= maxToConnect {
			log.Printf("[connection/manager.go] Достигнут лимит подключений (%d/%d), остальные %d адресов не обрабатываются",
				connectAttempt, maxToConnect, len(addresses)-i)
			break
		}

		peerID, err := peer.Decode(addr.PeerID)
		if err != nil {
			log.Printf("[connection/manager.go] [%d/%d] Неверный PeerID %s: %v", i+1, len(addresses), addr.PeerID, err)
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
			log.Printf("[connection/manager.go] [%d/%d] Неверный адрес для %s (%s): %v", i+1, len(addresses), addr.PeerID[:8], addr.AddressType, err)
			continue
		}

		// Добавляем адрес в peerstore
		s.host.Peerstore().AddAddr(peerID, ma, peerstore.PermanentAddrTTL)

		// Увеличиваем счётчик
		connectAttempt++

		// Подключение в горутине
		go func(idx int, addr *models.PeerAddress, pid peer.ID, ma multiaddr.Multiaddr) {
			log.Printf("[connection/manager] [%d/%d] Попытка подключения к %s (тип: %s, адрес: %s)...",
				idx, len(addresses), pid.String()[:8], addr.AddressType, ma.String())

			ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
			defer cancel()

			startTime := time.Now()
			if err := s.host.Connect(ctx, peer.AddrInfo{
				ID:    pid,
				Addrs: []multiaddr.Multiaddr{ma},
			}); err != nil {
				elapsed := time.Since(startTime)
				log.Printf("[connection/manager] [%d/%d] FAILED подключение к %s (%s) за %v: %v",
					idx, len(addresses), pid.String()[:8], addr.AddressType, elapsed, err)

				// Анализируем ошибку для подсказок
				errStr := err.Error()
				if strings.Contains(errStr, "timeout") {
					log.Printf("[connection/manager]   💡 Timeout: пир может быть за NAT или офлайн")
				} else if strings.Contains(errStr, "refused") || strings.Contains(errStr, "connect") {
					log.Printf("[connection/manager]   💡 Connection refused: порт может быть закрыт брандмауэром")
				} else if strings.Contains(errStr, "no good addresses") {
					log.Printf("[connection/manager]   💡 No good addresses: проверьте формат multiaddr")
				}
				// Переподключение НЕ выполняется — только одна попытка при запуске
			} else {
				elapsed := time.Since(startTime)
				log.Printf("[connection/manager] [%d/%d] SUCCESS подключение к %s (%s) за %v",
					idx, len(addresses), pid.String()[:8], addr.AddressType, elapsed)

				// Определяем тип установленного соединения
				conns := s.host.Network().ConnsToPeer(pid)
				for i, conn := range conns {
					remoteAddr := conn.RemoteMultiaddr()
					connType := "DIRECT"
					if strings.Contains(remoteAddr.String(), "/p2p-circuit") {
						connType = "RELAYED"
					}
					log.Printf("[connection/manager]   Соединение #%d: тип=%s, адрес=%s", i+1, connType, remoteAddr.String())
				}

				// Обновляем время подключения в БД
				_ = queries.UpdatePeerAddressLastConnected(addr.Multiaddr)
				_ = queries.UpdateProfileLastConnected(addr.PeerID)
			}
		}(i+1, addr, peerID, ma)
	}

	log.Printf("[connection/manager.go] Инициировано %d подключений из %d известных пиров (лимит: %d)",
		connectAttempt, len(addresses), maxToConnect)
	log.Printf("[connection/manager.go] ========================================")
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
			// Переподключение НЕ выполняется — только одна попытка при запуске
		}
	}

	// Обновляем метрики количества пиров
	if metrics.IsInitialized() {
		m := metrics.Get().Metrics
		m.P2PPeersTotal.Set(float64(len(connectedPeers)))
	}
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
