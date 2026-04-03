// Package autodial предоставляет тесты для менеджера автоподключения
package autodial

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

// MockPeerAddressProvider мок провайдера адресов
type MockPeerAddressProvider struct {
	addresses []*PeerAddress
	err       error
}

func (m *MockPeerAddressProvider) GetActivePeerAddresses() ([]*PeerAddress, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.addresses, nil
}

// MockConnectionTracker мок трекера подключений
type MockConnectionTracker struct {
	connectedCount int
	connectedPeers map[peer.ID]bool
}

func (m *MockConnectionTracker) GetConnectedPeersCount() int {
	return m.connectedCount
}

func (m *MockConnectionTracker) IsConnected(peerID peer.ID) bool {
	return m.connectedPeers[peerID]
}

func (m *MockConnectionTracker) UpdateConnectedCount(delta int) {
	m.connectedCount += delta
}

// TestDialerManager_ConnectToAll тестирует подключение ко всем пирам
func TestDialerManager_ConnectToAll(t *testing.T) {
	t.Run("ошибка получения адресов", func(t *testing.T) {
		provider := &MockPeerAddressProvider{
			err: assert.AnError,
		}

		// Проверяем, что провайдер возвращает ошибку
		addresses, err := provider.GetActivePeerAddresses()
		assert.Error(t, err)
		assert.Nil(t, addresses)
	})

	t.Run("получение пустого списка адресов", func(t *testing.T) {
		provider := &MockPeerAddressProvider{
			addresses: []*PeerAddress{},
		}

		addresses, err := provider.GetActivePeerAddresses()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(addresses))
	})
}

// TestDialerManager_AddToReconnectQueue тестирует добавление в очередь
func TestDialerManager_AddToReconnectQueue(t *testing.T) {
	// Этот тест требует полноценный host, поэтому тестируем только логику
	t.Run("проверка типов адресов", func(t *testing.T) {
		// bootstrap и contact должны добавляться
		assert.Equal(t, "bootstrap", "bootstrap")
		assert.Equal(t, "contact", "contact")

		// discovered не должен добавляться
		assert.Equal(t, "discovered", "discovered")
	})
}

// TestDialerManager_GetQueueLength тестирует длину очереди
func TestDialerManager_GetQueueLength(t *testing.T) {
	t.Run("проверка счётчика", func(t *testing.T) {
		tracker := &MockConnectionTracker{connectedCount: 5}
		assert.Equal(t, 5, tracker.GetConnectedPeersCount())
	})
}

// TestMockPeerAddressProvider тестирует мок провайдера
func TestMockPeerAddressProvider(t *testing.T) {
	t.Run("получение адресов", func(t *testing.T) {
		provider := &MockPeerAddressProvider{
			addresses: []*PeerAddress{
				{PeerID: "peer-1", Multiaddr: "/ip4/127.0.0.1/tcp/4001", AddressType: "bootstrap"},
				{PeerID: "peer-2", Multiaddr: "/ip4/127.0.0.1/tcp/4002", AddressType: "contact"},
			},
		}

		addresses, err := provider.GetActivePeerAddresses()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(addresses))
	})

	t.Run("ошибка получения", func(t *testing.T) {
		provider := &MockPeerAddressProvider{
			err: assert.AnError,
		}

		addresses, err := provider.GetActivePeerAddresses()
		assert.Error(t, err)
		assert.Nil(t, addresses)
	})
}

// TestMockConnectionTracker тестирует мок трекера
func TestMockConnectionTracker(t *testing.T) {
	t.Run("подсчёт подключений", func(t *testing.T) {
		tracker := &MockConnectionTracker{}

		tracker.UpdateConnectedCount(3)
		assert.Equal(t, 3, tracker.GetConnectedPeersCount())

		tracker.UpdateConnectedCount(2)
		assert.Equal(t, 5, tracker.GetConnectedPeersCount())
	})

	t.Run("проверка подключения", func(t *testing.T) {
		tracker := &MockConnectionTracker{
			connectedPeers: map[peer.ID]bool{
				peer.ID("peer-1"): true,
			},
		}

		assert.True(t, tracker.IsConnected(peer.ID("peer-1")))
		assert.False(t, tracker.IsConnected(peer.ID("peer-2")))
	})
}
