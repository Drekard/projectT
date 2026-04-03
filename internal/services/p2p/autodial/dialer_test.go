// Package autodial предоставляет тесты для сервиса автоподключения
package autodial

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDialer_Creation тестирует создание диалера
func TestDialer_Creation(t *testing.T) {
	t.Run("создание с конфигурацией по умолчанию", func(t *testing.T) {
		config := DefaultConfig()

		require.NotNil(t, config)
		assert.Equal(t, 50, config.MaxConcurrentConnections)
		assert.Equal(t, 15*time.Second, config.ConnectionTimeout)
		assert.Equal(t, 30*time.Second, config.ReconnectInterval)
		assert.Equal(t, 5, config.MaxReconnectAttempts)
	})

	t.Run("создание с кастомной конфигурацией", func(t *testing.T) {
		config := &Config{
			MaxConcurrentConnections: 10,
			ConnectionTimeout:        5 * time.Second,
		}

		require.NotNil(t, config)
		assert.Equal(t, 10, config.MaxConcurrentConnections)
		assert.Equal(t, 5*time.Second, config.ConnectionTimeout)
	})
}

// TestDialer_CanConnect тестирует проверку возможности подключения
func TestDialer_CanConnect(t *testing.T) {
	t.Run("можно подключаться когда счётчик < лимита", func(t *testing.T) {
		dialer := &Dialer{
			config:         &Config{MaxConcurrentConnections: 5},
			connectedCount: 3,
		}

		assert.True(t, dialer.CanConnect())
	})

	t.Run("нельзя подключаться когда счётчик >= лимита", func(t *testing.T) {
		dialer := &Dialer{
			config:         &Config{MaxConcurrentConnections: 5},
			connectedCount: 5,
		}

		assert.False(t, dialer.CanConnect())
	})
}

// TestDialer_AddToReconnectQueue тестирует добавление в очередь переподключения
func TestDialer_AddToReconnectQueue(t *testing.T) {
	t.Run("добавление пира в очередь", func(t *testing.T) {
		dialer := &Dialer{
			reconnectQueue:    make([]peer.ID, 0),
			reconnectAttempts: make(map[peer.ID]int),
		}

		peerID := peer.ID("test-peer-1")
		dialer.AddToReconnectQueue(peerID)

		assert.Equal(t, 1, dialer.GetReconnectQueueLength())
	})

	t.Run("дубликат не добавляется в очередь", func(t *testing.T) {
		dialer := &Dialer{
			reconnectQueue:    make([]peer.ID, 0),
			reconnectAttempts: make(map[peer.ID]int),
		}

		peerID := peer.ID("test-peer-1")
		dialer.AddToReconnectQueue(peerID)
		dialer.AddToReconnectQueue(peerID)

		assert.Equal(t, 1, dialer.GetReconnectQueueLength())
	})

	t.Run("добавление нескольких пиров", func(t *testing.T) {
		dialer := &Dialer{
			reconnectQueue:    make([]peer.ID, 0),
			reconnectAttempts: make(map[peer.ID]int),
		}

		dialer.AddToReconnectQueue(peer.ID("peer-1"))
		dialer.AddToReconnectQueue(peer.ID("peer-2"))
		dialer.AddToReconnectQueue(peer.ID("peer-3"))

		assert.Equal(t, 3, dialer.GetReconnectQueueLength())
	})
}

// TestDialer_GetConnectedCount тестирует счётчик подключений
func TestDialer_GetConnectedCount(t *testing.T) {
	t.Run("получение количества подключений", func(t *testing.T) {
		dialer := &Dialer{
			connectedCount: 5,
		}

		assert.Equal(t, 5, dialer.GetConnectedCount())
	})
}

// TestDefaultConfig тестирует конфигурацию по умолчанию
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 50, config.MaxConcurrentConnections)
	assert.Equal(t, 15*time.Second, config.ConnectionTimeout)
	assert.Equal(t, 30*time.Second, config.ReconnectInterval)
	assert.Equal(t, 5, config.MaxReconnectAttempts)
}

// TestPeerAddress тестирует структуру PeerAddress
func TestPeerAddress(t *testing.T) {
	t.Run("создание PeerAddress", func(t *testing.T) {
		addr := &PeerAddress{
			PeerID:      "test-peer",
			Multiaddr:   "/ip4/127.0.0.1/tcp/4001",
			AddressType: "bootstrap",
			Priority:    10,
		}

		assert.Equal(t, "test-peer", addr.PeerID)
		assert.Equal(t, "/ip4/127.0.0.1/tcp/4001", addr.Multiaddr)
		assert.Equal(t, "bootstrap", addr.AddressType)
		assert.Equal(t, 10, addr.Priority)
	})
}

// TestDialResult тестирует структуру DialResult
func TestDialResult(t *testing.T) {
	t.Run("успешный результат", func(t *testing.T) {
		result := DialResult{
			PeerID:  peer.ID("test-peer"),
			Success: true,
		}

		assert.True(t, result.Success)
		assert.Equal(t, peer.ID("test-peer"), result.PeerID)
		assert.Nil(t, result.Error)
	})

	t.Run("неуспешный результат", func(t *testing.T) {
		result := DialResult{
			PeerID:  peer.ID("test-peer"),
			Success: false,
			Error:   assert.AnError,
		}

		assert.False(t, result.Success)
		assert.NotNil(t, result.Error)
	})
}
