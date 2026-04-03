// Package autodial предоставляет сервисы для автоматического подключения к пирам
package autodial

import (
	"context"
	"log"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// PeerAddressProvider интерфейс для получения адресов пиров
type PeerAddressProvider interface {
	GetActivePeerAddresses() ([]*PeerAddress, error)
}

// ConnectionTracker интерфейс для отслеживания подключений
type ConnectionTracker interface {
	GetConnectedPeersCount() int
	IsConnected(peer.ID) bool
	UpdateConnectedCount(delta int)
}

// DialerManager управляет автоподключением и переподключением
type DialerManager struct {
	dialer   *Dialer
	queue    *ReconnectQueue
	provider PeerAddressProvider
	tracker  ConnectionTracker
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewDialerManager создаёт менеджер автоподключения
func NewDialerManager(
	h host.Host,
	config *Config,
	provider PeerAddressProvider,
	tracker ConnectionTracker,
) *DialerManager {
	ctx, cancel := context.WithCancel(context.Background())

	dialer := NewDialer(h, config)
	queue := NewReconnectQueue(config.ReconnectInterval, config.MaxReconnectAttempts)

	manager := &DialerManager{
		dialer:   dialer,
		queue:    queue,
		provider: provider,
		tracker:  tracker,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Запускаем обработку очереди переподключения
	queue.Start(manager.handleReconnect)

	return manager
}

// Start запускает менеджер
func (m *DialerManager) Start() error {
	return m.dialer.Start()
}

// Stop останавливает менеджер
func (m *DialerManager) Stop() error {
	m.cancel()
	m.queue.Stop()
	return m.dialer.Stop()
}

// ConnectToAll подключается ко всем известным пирам
func (m *DialerManager) ConnectToAll() <-chan DialResult {
	addresses, err := m.provider.GetActivePeerAddresses()
	if err != nil {
		log.Printf("[Autodial] Ошибка получения адресов: %v", err)
		results := make(chan DialResult)
		close(results)
		return results
	}

	return m.dialer.DialMany(addresses)
}

// handleReconnect обрабатывает переподключение
func (m *DialerManager) handleReconnect(peerID peer.ID) {
	// Проверяем, можно ли ещё подключаться
	if !m.dialer.CanConnect() {
		m.queue.Add(peerID) // Возвращаем в очередь
		return
	}

	m.dialer.attemptReconnect(peerID)
}

// AddToReconnectQueue добавляет пира в очередь переподключения
func (m *DialerManager) AddToReconnectQueue(peerID peer.ID, addressType string) {
	// Только bootstrap и contact добавляем в очередь
	if addressType == "bootstrap" || addressType == "contact" {
		if added := m.queue.Add(peerID); added {
			log.Printf("[Autodial] Пир %s добавлен в очередь переподключения", peerID.String()[:8])
		}
	}
}

// GetQueueLength возвращает длину очереди переподключения
func (m *DialerManager) GetQueueLength() int {
	return m.queue.Length()
}

// GetConnectedCount возвращает количество подключений
func (m *DialerManager) GetConnectedCount() int {
	return m.dialer.GetConnectedCount()
}
