// Package connection предоставляет сервисы для управления подключениями P2P
package connection

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	p2p "projectT/internal/services/p2p"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ConnectionStatus статус подключения к пиру
type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusReconnecting ConnectionStatus = "reconnecting"
	StatusUnknown      ConnectionStatus = "unknown"
)

// PeerConnectionInfo информация о подключении к пиру
type PeerConnectionInfo struct {
	Status          ConnectionStatus
	LastSeen        time.Time
	LastPing        time.Time
	LastPingLatency time.Duration
	ReconnectCount  int
	AddedAt         time.Time
	LastProfileExch time.Time // Время последнего обмена профиля
}

// Service сервис мониторинга соединений
type Service struct {
	host           host.Host
	config         *p2p.P2PConfig
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	peerStatus     map[peer.ID]*PeerConnectionInfo
	keepAliveFail  map[peer.ID]int
	pendingProfile map[peer.ID]time.Time
}

// NewService создаёт сервис мониторинга соединений
func NewService(host host.Host, config *p2p.P2PConfig) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		host:           host,
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
		peerStatus:     make(map[peer.ID]*PeerConnectionInfo),
		keepAliveFail:  make(map[peer.ID]int),
		pendingProfile: make(map[peer.ID]time.Time),
	}
}

// PingProtocolID идентификатор протокола ping
const PingProtocolID = "/projectt/ping/1.0.0"

// Start запускает мониторинг соединений
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Устанавливаем обработчик stream для ping
	s.host.SetStreamHandler(PingProtocolID, s.handlePing)

	// Инициализируем подключения
	s.initializeConnections()

	// Запускаем мониторинг
	go s.monitorConnections()

	// Запускаем KeepAlive
	go s.startKeepAlive()

	// Переподключение НЕ выполняется — только одна попытка при запуске

	return nil
}

// Stop останавливает сервис
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancel()
	return nil
}

// startKeepAlive запускает периодическую отправку ping
func (s *Service) startKeepAlive() {
	ticker := time.NewTicker(s.config.KeepAlive)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.sendKeepAlive()
		}
	}
}

// sendKeepAlive отправляет ping всем подключённым пирам
func (s *Service) sendKeepAlive() {
	s.mu.RLock()
	peers := s.host.Network().Peers()
	pending := s.pendingProfile
	s.mu.RUnlock()

	for _, peerID := range peers {
		// Пропускаем пиров, у которых профиль ещё не получен
		if _, ok := pending[peerID]; ok {
			// Проверяем, не истёк ли таймаут (60 секунд)
			if time.Since(pending[peerID]) > 60*time.Second {
				// Таймаут истёк, убираем из pending
				s.mu.Lock()
				delete(s.pendingProfile, peerID)
				s.mu.Unlock()
				log.Printf("[connection/ping.go] Таймаут ожидания профиля для %s, начинаем ping", peerID.String()[:8])
			} else {
				continue
			}
		}
		go s.pingPeer(peerID)
	}
}

// pingPeer отправляет ping конкретному пиру
func (s *Service) pingPeer(peerID peer.ID) {
	ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
	defer cancel()

	startTime := time.Now()
	stream, err := s.host.NewStream(ctx, peerID, PingProtocolID)
	if err != nil {
		log.Printf("[connection/ping.go] Ошибка создания стрима для %s за %v: %v", peerID.String()[:8], time.Since(startTime), err)
		s.handlePingFailure(peerID, fmt.Errorf("не удалось создать стрим: %w", err))
		return
	}
	defer func() { _ = stream.Close() }()

	// Отправляем "ping"
	_, err = stream.Write([]byte("ping"))
	if err != nil {
		log.Printf("[connection/ping.go] Ошибка записи ping для %s: %v", peerID.String()[:8], err)
		s.handlePingFailure(peerID, fmt.Errorf("ошибка записи: %w", err))
		return
	}

	// Читаем "pong"
	if err := stream.SetReadDeadline(time.Now().Add(40 * time.Second)); err != nil {
		log.Printf("[connection/ping.go] Ошибка установки таймаута для %s: %v", peerID.String()[:8], err)
		s.handlePingFailure(peerID, fmt.Errorf("ошибка установки таймаута: %w", err))
		return
	}
	response := make([]byte, 4)
	n, err := stream.Read(response)
	latency := time.Since(startTime)

	if err != nil {
		log.Printf("[connection/ping.go] Ошибка чтения pong от %s: %v", peerID.String()[:8], err)
		s.handlePingFailure(peerID, fmt.Errorf("ошибка чтения или неверный ответ: %w", err))
		return
	}

	if n != 4 || string(response) != "pong" {
		log.Printf("[connection/ping.go] Неверный ответ от %s: n=%d, response=%q", peerID.String()[:8], n, string(response))
		s.handlePingFailure(peerID, fmt.Errorf("ошибка чтения или неверный ответ"))
		return
	}

	log.Printf("[connection/ping.go] Pong от %s, latency: %v", peerID.String()[:8], latency)
	s.handlePingSuccess(peerID, latency)
}

// handlePingSuccess обрабатывает успешный ping
func (s *Service) handlePingSuccess(peerID peer.ID, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.keepAliveFail, peerID)

	if info, exists := s.peerStatus[peerID]; exists {
		info.LastPing = time.Now()
		info.LastPingLatency = latency
	}
}

// handlePingFailure обрабатывает неудачный ping
func (s *Service) handlePingFailure(peerID peer.ID, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keepAliveFail[peerID]++
	failCount := s.keepAliveFail[peerID]

	log.Printf("[connection/ping.go] KeepAlive failed для %s (попытка %d/3): %v", peerID, failCount, err)

	if failCount >= 3 {
		if info, exists := s.peerStatus[peerID]; exists {
			info.Status = StatusDisconnected
			info.LastSeen = time.Now()
			log.Printf("[connection/ping.go] Пир %s помечен как offline (3 неудачных ping)", peerID)
			go s.updateContactLastSeen(peerID)
		}

		delete(s.keepAliveFail, peerID)
		// Переподключение НЕ выполняется — только одна попытка при запуске
	}
}

// handlePing обрабатывает входящий ping
func (s *Service) handlePing(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	buffer := make([]byte, 4)
	n, err := stream.Read(buffer)
	if err != nil {
		return
	}

	if n == 4 && string(buffer) == "ping" {
		if _, err := stream.Write([]byte("pong")); err != nil {
			log.Printf("[connection/ping.go] Ошибка записи pong: %v", err)
		}
	}
}
