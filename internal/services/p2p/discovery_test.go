package p2p

// Package p2p содержит сервисы для P2P связи на базе libp2p.

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiscoveryService_Init тестирует инициализацию сервиса обнаружения
func TestDiscoveryService_Init(t *testing.T) {
	// Создаём тестовый хост
	h, err := makeTestHost()
	require.NoError(t, err)
	defer h.Close()

	// Создаём сервис обнаружения
	config := DefaultConfig()
	config.EnableDHT = false // Отключаем DHT для простоты теста
	config.EnableMDNS = false

	ds := NewDiscoveryService(h, nil, config)
	require.NotNil(t, ds)
}

// TestDiscoveryService_BootstrapPeers тестирует загрузку bootstrap-пиров
// Примечание: этот тест требует инициализированной БД, поэтому пропускается
func TestDiscoveryService_BootstrapPeers(t *testing.T) {
	t.Skip("Требует инициализированную БД")

	h, err := makeTestHost()
	require.NoError(t, err)
	defer h.Close()

	config := DefaultConfig()
	config.EnableDHT = false
	config.EnableMDNS = false

	ds := NewDiscoveryService(h, nil, config)

	// Проверяем, что загрузка работает
	err = ds.loadBootstrapPeers()
	assert.NoError(t, err)
}

// TestDiscoveryService_AddRemoveBootstrapPeer тестирует добавление и удаление bootstrap-пира
// Примечание: этот тест требует инициализированной БД, поэтому пропускается в unit-тестах
func TestDiscoveryService_AddRemoveBootstrapPeer(t *testing.T) {
	t.Skip("Требует инициализированную БД")

	h, err := makeTestHost()
	require.NoError(t, err)
	defer h.Close()

	config := DefaultConfig()
	config.EnableDHT = false
	config.EnableMDNS = false

	ds := NewDiscoveryService(h, nil, config)

	// Тестовый multiaddr
	testAddr := "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN"

	// Добавляем bootstrap-пир
	err = ds.AddBootstrapPeer(testAddr)
	assert.NoError(t, err)

	// Проверяем, что пир добавлен
	peers, err := ds.GetBootstrapPeers()
	assert.NoError(t, err)
	assert.NotEmpty(t, peers)

	// Удаляем bootstrap-пир
	err = ds.RemoveBootstrapPeer(testAddr)
	assert.NoError(t, err)

	// Проверяем, что пир удалён
	peers, err = ds.GetBootstrapPeers()
	assert.NoError(t, err)
	// Пир должен быть удалён (или не найден в активных)
	found := false
	for _, p := range peers {
		if p.Multiaddr == testAddr {
			found = true
			break
		}
	}
	assert.False(t, found, "Bootstrap-пир должен быть удалён")
}

// TestDiscoveryService_GetDiscoveredPeers тестирует получение обнаруженных пиров
func TestDiscoveryService_GetDiscoveredPeers(t *testing.T) {
	h, err := makeTestHost()
	require.NoError(t, err)
	defer h.Close()

	config := DefaultConfig()
	config.EnableDHT = false
	config.EnableMDNS = false

	ds := NewDiscoveryService(h, nil, config)

	// Проверяем, что изначально пусто
	peers := ds.GetDiscoveredPeers()
	assert.Empty(t, peers)

	// Добавляем тестового пира в кэш
	ds.mu.Lock()
	testPeerID := peer.ID("test-peer-id")
	ds.discoveredPeers[testPeerID.String()] = time.Now()
	ds.mu.Unlock()

	// Проверяем, что пир появился
	peers = ds.GetDiscoveredPeers()
	assert.NotEmpty(t, peers)
	assert.Contains(t, peers, testPeerID.String())

	// Очищаем кэш
	ds.ClearDiscoveredPeers()
	peers = ds.GetDiscoveredPeers()
	assert.Empty(t, peers)
}

// TestIsLocalAddress тестирует функцию проверки локального адреса
func TestIsLocalAddress(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected bool
	}{
		{"localhost IPv4", "/ip4/127.0.0.1/tcp/4001", true},
		{"private IPv4", "/ip4/192.168.1.1/tcp/4001", true},
		{"private IPv4 class B", "/ip4/172.16.0.1/tcp/4001", true},
		{"private IPv4 class A", "/ip4/10.0.0.1/tcp/4001", true},
		{"link-local IPv4", "/ip4/169.254.1.1/tcp/4001", true},
		{"localhost IPv6", "/ip6/::1/tcp/4001", true},
		{"public IPv4", "/ip4/8.8.8.8/tcp/4001", false},
		{"public IPv6", "/ip6/2001:4860:4860::8888/tcp/4001", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := multiaddr.NewMultiaddr(tt.addr)
			require.NoError(t, err)

			result := isLocalAddress(addr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDiscoveryService_StartStop тестирует запуск и остановку сервиса
// Примечание: этот тест требует инициализированной БД, поэтому пропускается
func TestDiscoveryService_StartStop(t *testing.T) {
	t.Skip("Требует инициализированную БД")

	h, err := makeTestHost()
	require.NoError(t, err)
	defer h.Close()

	config := DefaultConfig()
	config.EnableDHT = false
	config.EnableMDNS = false

	ds := NewDiscoveryService(h, nil, config)

	// Запускаем сервис
	err = ds.Start()
	assert.NoError(t, err)

	// Останавливаем сервис
	err = ds.Stop()
	assert.NoError(t, err)
}

// makeTestHost создаёт тестовый хост
func makeTestHost() (host.Host, error) {
	// Генерируем ключи для тестового хоста
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	// Создаём хост
	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		return nil, err
	}

	return h, nil
}

// TestDiscoveryService_HandleDiscoveredPeer тестирует обработку обнаруженного пира
// Примечание: этот тест требует инициализированной БД, поэтому пропускается
func TestDiscoveryService_HandleDiscoveredPeer(t *testing.T) {
	t.Skip("Требует инициализированную БД")

	h, err := makeTestHost()
	require.NoError(t, err)
	defer h.Close()

	config := DefaultConfig()
	config.EnableDHT = false
	config.EnableMDNS = false

	ds := NewDiscoveryService(h, nil, config)

	// Создаём тестового пира
	testPeer := peer.AddrInfo{
		ID:    peer.ID("test-peer"),
		Addrs: []multiaddr.Multiaddr{},
	}

	// Обрабатываем пира
	ds.handleDiscoveredPeer(testPeer)

	// Проверяем, что пир добавлен в кэш
	peers := ds.GetDiscoveredPeers()
	assert.Contains(t, peers, "test-peer")
}

// TestDiscoveryService_DuplicatePeerHandling тестирует обработку дубликатов пиров
// Примечание: этот тест требует инициализированной БД, поэтому пропускается
func TestDiscoveryService_DuplicatePeerHandling(t *testing.T) {
	t.Skip("Требует инициализированную БД")

	h, err := makeTestHost()
	require.NoError(t, err)
	defer h.Close()

	config := DefaultConfig()
	config.EnableDHT = false
	config.EnableMDNS = false

	ds := NewDiscoveryService(h, nil, config)

	testPeer := peer.AddrInfo{
		ID:    peer.ID("test-peer-dup"),
		Addrs: []multiaddr.Multiaddr{},
	}

	// Обрабатываем пира первый раз
	ds.handleDiscoveredPeer(testPeer)
	firstCheck := ds.GetDiscoveredPeers()
	assert.Contains(t, firstCheck, "test-peer-dup")

	// Ждём немного
	time.Sleep(100 * time.Millisecond)

	// Обрабатываем того же пира второй раз (должен быть пропущен)
	ds.handleDiscoveredPeer(testPeer)
	secondCheck := ds.GetDiscoveredPeers()

	// Количество пиров не должно измениться
	assert.Equal(t, len(firstCheck), len(secondCheck))
}

// TestDiscoveryService_ConnectToDiscoveredPeer тестирует подключение к обнаруженному пиру
func TestDiscoveryService_ConnectToDiscoveredPeer(t *testing.T) {
	h, err := makeTestHost()
	require.NoError(t, err)
	defer h.Close()

	testPeer := peer.AddrInfo{
		ID:    peer.ID("test-peer-connect"),
		Addrs: []multiaddr.Multiaddr{},
	}

	// Пытаемся подключиться (должно завершиться ошибкой, так как адрес пустой)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = h.Connect(ctx, testPeer)
	assert.Error(t, err)
}
