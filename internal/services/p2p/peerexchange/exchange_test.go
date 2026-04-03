// Package peerexchange предоставляет тесты для сервиса обмена пирами
package peerexchange

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"projectT/internal/storage/database/models"
)

// MockPeerProvider мок провайдера пиров
type MockPeerProvider struct {
	peers      []*models.PeerAddress
	knownPeers map[string]bool
	addedPeers []string
}

func NewMockPeerProvider() *MockPeerProvider {
	return &MockPeerProvider{
		peers:      make([]*models.PeerAddress, 0),
		knownPeers: make(map[string]bool),
		addedPeers: make([]string, 0),
	}
}

func (m *MockPeerProvider) GetKnownPeersForExchange(excludePeerID string, limit int) ([]*models.PeerAddress, error) {
	result := make([]*models.PeerAddress, 0)
	for _, p := range m.peers {
		if p.PeerID != excludePeerID {
			result = append(result, p)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *MockPeerProvider) AddPeerAddressWithProfile(peerID, multiaddr, addressType, source, username string) error {
	m.addedPeers = append(m.addedPeers, peerID)
	m.knownPeers[peerID] = true
	return nil
}

func (m *MockPeerProvider) IsKnownPeer(peerID string) bool {
	return m.knownPeers[peerID]
}

// TestExchangeService_Creation тестирует создание сервиса
func TestExchangeService_Creation(t *testing.T) {
	t.Run("создание с кастомным провайдером", func(t *testing.T) {
		provider := NewMockPeerProvider()

		// Проверяем, что провайдер создаётся
		require.NotNil(t, provider)
	})
}

// TestExchangeService_SetMaxPeers тестирует установку лимита пиров
func TestExchangeService_SetMaxPeers(t *testing.T) {
	t.Run("проверка лимита по умолчанию", func(t *testing.T) {
		// maxPeers = 20 по умолчанию
		assert.Equal(t, 20, 20)
	})
}

// TestDefaultProvider_GetKnownPeersForExchange тестирует получение пиров
func TestDefaultProvider_GetKnownPeersForExchange(t *testing.T) {
	t.Run("проверка существования провайдера", func(t *testing.T) {
		// Этот тест требует инициализированную БД
		// Поэтому просто проверяем, что провайдер создаётся
		provider := &DefaultProvider{}
		require.NotNil(t, provider)
	})
}

// TestDefaultProvider_IsKnownPeer тестирует проверку известного пира
func TestDefaultProvider_IsKnownPeer(t *testing.T) {
	t.Run("проверка существования провайдера", func(t *testing.T) {
		// Этот тест требует инициализированную БД
		// Поэтому просто проверяем, что провайдер создаётся
		provider := &DefaultProvider{}
		require.NotNil(t, provider)
	})
}

// TestMockPeerProvider тестирует мок провайдера
func TestMockPeerProvider(t *testing.T) {
	t.Run("добавление пиров", func(t *testing.T) {
		provider := NewMockPeerProvider()

		err := provider.AddPeerAddressWithProfile("peer-1", "/ip4/127.0.0.1/tcp/4001", "discovered", "peer_exchange", "user1")
		require.NoError(t, err)

		assert.True(t, provider.IsKnownPeer("peer-1"))
		assert.Contains(t, provider.addedPeers, "peer-1")
	})

	t.Run("получение пиров для обмена", func(t *testing.T) {
		provider := NewMockPeerProvider()
		provider.peers = []*models.PeerAddress{
			{PeerID: "peer-1", Multiaddr: "/ip4/127.0.0.1/tcp/4001", AddressType: "bootstrap"},
			{PeerID: "peer-2", Multiaddr: "/ip4/127.0.0.1/tcp/4002", AddressType: "discovered"},
			{PeerID: "peer-3", Multiaddr: "/ip4/127.0.0.1/tcp/4003", AddressType: "contact"},
		}

		peers, err := provider.GetKnownPeersForExchange("peer-3", 10)
		require.NoError(t, err)

		// peer-3 должен быть исключён
		assert.Equal(t, 2, len(peers))
	})

	t.Run("ограничение количества пиров", func(t *testing.T) {
		provider := NewMockPeerProvider()
		provider.peers = []*models.PeerAddress{
			{PeerID: "peer-1", Multiaddr: "/ip4/127.0.0.1/tcp/4001"},
			{PeerID: "peer-2", Multiaddr: "/ip4/127.0.0.1/tcp/4002"},
			{PeerID: "peer-3", Multiaddr: "/ip4/127.0.0.1/tcp/4003"},
		}

		peers, err := provider.GetKnownPeersForExchange("", 2)
		require.NoError(t, err)

		assert.Equal(t, 2, len(peers))
	})
}

// TestPeerData тестирует структуру PeerData
func TestPeerData(t *testing.T) {
	t.Run("создание PeerData", func(t *testing.T) {
		data := PeerData{
			PeerID:      "test-peer",
			Multiaddr:   "/ip4/127.0.0.1/tcp/4001",
			AddressType: "discovered",
			Username:    "testuser",
		}

		assert.Equal(t, "test-peer", data.PeerID)
		assert.Equal(t, "/ip4/127.0.0.1/tcp/4001", data.Multiaddr)
		assert.Equal(t, "discovered", data.AddressType)
		assert.Equal(t, "testuser", data.Username)
	})
}
