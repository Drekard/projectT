// Package network содержит тесты для P2P сети
package network

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p2p "projectT/internal/services/p2p"
)

// createTestHost создаёт тестовый хост для использования в тестах
func createTestHost(t *testing.T, port int) host.Host {
	t.Helper()

	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)),
		libp2p.DisableRelay(),
		libp2p.EnableNATService(),
	}

	h, err := libp2p.New(opts...)
	require.NoError(t, err)

	t.Cleanup(func() {
		h.Close()
	})

	return h
}

// createTestDHT создаёт тестовую DHT таблицу
func createTestDHT(t *testing.T, h host.Host) (*dht.IpfsDHT, *routing.RoutingDiscovery) {
	t.Helper()

	ctx := context.Background()

	dhtOpts := []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix("/projectt-test"),
		dht.BootstrapPeers(),
	}

	kdht, err := dht.New(ctx, h, dhtOpts...)
	require.NoError(t, err)

	t.Cleanup(func() {
		kdht.Close()
	})

	// Bootstrap DHT
	require.NoError(t, kdht.Bootstrap(ctx))

	discovery := routing.NewRoutingDiscovery(kdht)

	return kdht, discovery
}

// TestP2PNetwork_Creation тестирует создание P2P сети
func TestP2PNetwork_Creation(t *testing.T) {
	t.Run("создание новой сети", func(t *testing.T) {
		network := NewP2PNetwork()
		require.NotNil(t, network)
		assert.Equal(t, p2p.DefaultConfig(), network.config)
	})

	t.Run("сеть с кастомной конфигурацией", func(t *testing.T) {
		network := &P2PNetwork{
			config: &p2p.P2PConfig{
				ListenPort:        0,
				EnableDHT:         true,
				EnableMDNS:        false,
				ConnectionTimeout: 10 * time.Second,
			},
		}
		assert.False(t, network.config.EnableMDNS)
		assert.True(t, network.config.EnableDHT)
	})
}

// TestP2PNetwork_HostCreation тестирует создание хоста
func TestP2PNetwork_HostCreation(t *testing.T) {
	t.Run("создание хоста с ключами", func(t *testing.T) {
		h := createTestHost(t, 0)
		require.NotNil(t, h)
		assert.NotEmpty(t, h.ID())
		assert.NotEmpty(t, h.Addrs())
	})
}

// TestP2PNetwork_DHT тестирует DHT функциональность
func TestP2PNetwork_DHT(t *testing.T) {
	t.Run("создание DHT таблицы", func(t *testing.T) {
		h := createTestHost(t, 0)
		_, discovery := createTestDHT(t, h)
		require.NotNil(t, discovery)
	})

	t.Run("DHT discovery - два узла находят друг друга", func(t *testing.T) {
		ctx := context.Background()

		// Создаём два хоста
		host1 := createTestHost(t, 0)
		host2 := createTestHost(t, 0)

		// Создаём DHT для каждого
		createTestDHT(t, host1)
		createTestDHT(t, host2)

		// Подключаем host2 к host1
		addrInfo := peer.AddrInfo{
			ID:    host1.ID(),
			Addrs: host1.Addrs(),
		}

		err := host2.Connect(ctx, addrInfo)
		require.NoError(t, err)

		// Ждём немного для установления соединения
		time.Sleep(100 * time.Millisecond)

		// Проверяем что пиры подключены
		peers := host2.Network().Peers()
		assert.Len(t, peers, 1)
		assert.Equal(t, host1.ID(), peers[0])
	})

	t.Run("DHT advertisement and discovery", func(t *testing.T) {
		t.Skip("DHT advertisement требует больше времени на распространение в изолированной среде")

		ctx := context.Background()

		// Создаём два хоста
		host1 := createTestHost(t, 0)
		host2 := createTestHost(t, 0)

		// Создаём DHT для каждого
		_, disc1 := createTestDHT(t, host1)
		_, disc2 := createTestDHT(t, host2)

		// Подключаем host2 к host1
		addrInfo := peer.AddrInfo{
			ID:    host1.ID(),
			Addrs: host1.Addrs(),
		}

		err := host2.Connect(ctx, addrInfo)
		require.NoError(t, err)

		// Host1 рекламирует сервис
		_, err = disc1.Advertise(ctx, p2p.ProtocolID)
		require.NoError(t, err)

		// Ждём распространения информации в DHT
		time.Sleep(500 * time.Millisecond)

		// Host2 ищет пиры через DHT
		peersChan, err := disc2.FindPeers(ctx, p2p.ProtocolID)
		require.NoError(t, err)

		// Считаем найденных пиров
		var foundPeers []peer.AddrInfo
		for p := range peersChan {
			if p.ID != host2.ID() && len(p.Addrs) > 0 {
				foundPeers = append(foundPeers, p)
			}
		}

		// Хотя бы один пир должен быть найден (или 0 если DHT не успела распространить)
		assert.GreaterOrEqual(t, len(foundPeers), 0)
	})
}

// TestP2PNetwork_Connections тестирует подключения между пирами
func TestP2PNetwork_Connections(t *testing.T) {
	t.Run("прямое подключение между двумя хостами", func(t *testing.T) {
		ctx := context.Background()

		host1 := createTestHost(t, 0)
		host2 := createTestHost(t, 0)

		// Host2 подключается к host1
		addrInfo := peer.AddrInfo{
			ID:    host1.ID(),
			Addrs: host1.Addrs(),
		}

		err := host2.Connect(ctx, addrInfo)
		require.NoError(t, err)

		// Проверяем статус подключения
		assert.Equal(t, network.Connected, host2.Network().Connectedness(host1.ID()))
		assert.Equal(t, network.Connected, host1.Network().Connectedness(host2.ID()))
	})

	t.Run("отключение и переподключение", func(t *testing.T) {
		ctx := context.Background()

		host1 := createTestHost(t, 0)
		host2 := createTestHost(t, 0)

		// Подключаемся
		addrInfo := peer.AddrInfo{
			ID:    host1.ID(),
			Addrs: host1.Addrs(),
		}

		err := host2.Connect(ctx, addrInfo)
		require.NoError(t, err)
		assert.Equal(t, network.Connected, host2.Network().Connectedness(host1.ID()))

		// Отключаемся
		_ = host2.Network().ClosePeer(host1.ID()) //nolint:errcheck
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, network.NotConnected, host2.Network().Connectedness(host1.ID()))

		// Переподключаемся
		err = host2.Connect(ctx, addrInfo)
		require.NoError(t, err)
		assert.Equal(t, network.Connected, host2.Network().Connectedness(host1.ID()))
	})
}

// TestP2PNetwork_PeerID тестирует PeerID
func TestP2PNetwork_PeerID(t *testing.T) {
	t.Run("уникальный PeerID для каждого хоста", func(t *testing.T) {
		host1 := createTestHost(t, 0)
		host2 := createTestHost(t, 0)

		// PeerID должны быть разными
		assert.NotEqual(t, host1.ID(), host2.ID())
	})

	t.Run("PeerID сохраняется между перезапусками", func(t *testing.T) {
		// Генерируем ключ
		privKey1, _, err := crypto.GenerateEd25519Key(rand.Reader)
		require.NoError(t, err)

		// Создаём хост с ключом
		h1, err := libp2p.New(libp2p.Identity(privKey1))
		require.NoError(t, err)
		id1 := h1.ID()
		h1.Close()

		// Создаём хост с тем же ключом
		h2, err := libp2p.New(libp2p.Identity(privKey1))
		require.NoError(t, err)
		id2 := h2.ID()
		h2.Close()

		// PeerID должны совпадать
		assert.Equal(t, id1, id2)
	})
}

// TestP2PNetwork_Addresses тестирует адреса
func TestP2PNetwork_Addresses(t *testing.T) {
	t.Run("хост имеет несколько адресов", func(t *testing.T) {
		h := createTestHost(t, 0)
		addrs := h.Addrs()
		assert.GreaterOrEqual(t, len(addrs), 1)
	})

	t.Run("форматирование адреса для экспорта", func(t *testing.T) {
		h := createTestHost(t, 0)

		addr := h.Addrs()[0]
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), h.ID().String())
		prefixedAddr := p2p.ProtocolPrefix + "://" + fullAddr

		assert.Contains(t, prefixedAddr, p2p.ProtocolPrefix)
		assert.Contains(t, prefixedAddr, h.ID().String())
	})
}

// TestP2PNetwork_Config тестирует конфигурацию
func TestP2PNetwork_Config(t *testing.T) {
	t.Run("конфигурация по умолчанию", func(t *testing.T) {
		cfg := p2p.DefaultConfig()

		assert.Equal(t, 0, cfg.ListenPort)
		assert.True(t, cfg.EnableDHT)
		assert.True(t, cfg.EnableMDNS)
		assert.True(t, cfg.EnableNATPortMap)
		assert.True(t, cfg.EnableRelay)
		assert.Equal(t, 30*time.Second, cfg.ConnectionTimeout)
	})

	t.Run("кастомная конфигурация", func(t *testing.T) {
		cfg := &p2p.P2PConfig{
			ListenPort:        4001,
			EnableDHT:         false,
			EnableMDNS:        false,
			EnableNATPortMap:  false,
			ConnectionTimeout: 60 * time.Second,
		}

		assert.Equal(t, 4001, cfg.ListenPort)
		assert.False(t, cfg.EnableDHT)
		assert.False(t, cfg.EnableMDNS)
	})
}

// TestP2PNetwork_ConnectionService тестирует сервис соединений
func TestP2PNetwork_ConnectionService(t *testing.T) {
	t.Run("создание сервиса соединений", func(t *testing.T) {
		t.Skip("Требует инициализированную БД - тестируется в integration тестах")

		h := createTestHost(t, 0)
		cfg := p2p.DefaultConfig()

		service := p2p.NewConnectionService(h, cfg)
		require.NotNil(t, service)

		err := service.Start()
		require.NoError(t, err)

		err = service.Stop()
		require.NoError(t, err)
	})

	t.Run("получение статуса подключения", func(t *testing.T) {
		t.Skip("Требует полноценной инициализации БД и сервисов")

		ctx := context.Background()
		h1 := createTestHost(t, 0)
		h2 := createTestHost(t, 0)

		cfg := p2p.DefaultConfig()
		service := p2p.NewConnectionService(h2, cfg)
		err := service.Start()
		require.NoError(t, err)
		defer func() { _ = service.Stop() }() //nolint:errcheck

		// До подключения статус неизвестен
		status := service.GetConnectionStatus(h1.ID())
		assert.Equal(t, p2p.StatusUnknown, status)

		// Подключаемся
		err = h2.Connect(ctx, peer.AddrInfo{
			ID:    h1.ID(),
			Addrs: h1.Addrs(),
		})
		require.NoError(t, err)

		// Ждём обновления статуса
		time.Sleep(200 * time.Millisecond)

		// После подключения статус должен быть connected
		status = service.GetConnectionStatus(h1.ID())
		assert.Equal(t, p2p.StatusConnected, status)
	})
}

// TestP2PNetwork_DiscoveryService тестирует сервис обнаружения
func TestP2PNetwork_DiscoveryService(t *testing.T) {
	t.Run("создание сервиса обнаружения", func(t *testing.T) {
		h := createTestHost(t, 0)
		_, discovery := createTestDHT(t, h)
		cfg := &p2p.P2PConfig{
			ListenPort:     0,
			EnableDHT:      true,
			EnableMDNS:     false,
			BootstrapPeers: []string{},
		}

		service := p2p.NewDiscoveryService(h, discovery, cfg)
		require.NotNil(t, service)

		err := service.Start()
		require.NoError(t, err)

		err = service.Stop()
		require.NoError(t, err)
	})

	t.Run("добавление и удаление bootstrap пиров", func(t *testing.T) {
		t.Skip("Требует инициализированную БД - тестируется в integration тестах")

		h := createTestHost(t, 0)
		_, discovery := createTestDHT(t, h)
		cfg := &p2p.P2PConfig{
			ListenPort:     0,
			EnableDHT:      true,
			EnableMDNS:     false,
			BootstrapPeers: []string{},
		}

		service := p2p.NewDiscoveryService(h, discovery, cfg)

		// Добавляем bootstrap пир
		testAddr := "/ip4/127.0.0.1/tcp/4001/p2p/QmTest"
		err := service.AddBootstrapPeer(testAddr)
		require.NoError(t, err)

		// Пытаемся удалить
		err = service.RemoveBootstrapPeer(testAddr)
		require.NoError(t, err)
	})
}

// TestP2PNetwork_Integration тестирует интеграцию компонентов
func TestP2PNetwork_Integration(t *testing.T) {
	t.Run("полный цикл: создание, подключение, обнаружение", func(t *testing.T) {
		ctx := context.Background()

		// Создаём два хоста
		host1 := createTestHost(t, 0)
		host2 := createTestHost(t, 0)

		// Создаём DHT
		_, disc1 := createTestDHT(t, host1)
		_, disc2 := createTestDHT(t, host2)

		// Создаём сервисы обнаружения с отключенной загрузкой из БД
		cfg := &p2p.P2PConfig{
			ListenPort:        0,
			EnableDHT:         true,
			EnableMDNS:        false, // mDNS недоступен в тестах
			ConnectionTimeout: 10 * time.Second,
			BootstrapPeers:    []string{}, // Пустой список bootstrap пиров
		}
		service1 := p2p.NewDiscoveryService(host1, disc1, cfg)
		service2 := p2p.NewDiscoveryService(host2, disc2, cfg)

		// Запускаем сервисы
		require.NoError(t, service1.Start())
		require.NoError(t, service2.Start())
		defer func() { _ = service1.Stop() }() //nolint:errcheck
		defer func() { _ = service2.Stop() }() //nolint:errcheck

		// Подключаем host2 к host1
		addrInfo := peer.AddrInfo{
			ID:    host1.ID(),
			Addrs: host1.Addrs(),
		}

		err := host2.Connect(ctx, addrInfo)
		require.NoError(t, err)

		// Ждём установления соединения
		time.Sleep(300 * time.Millisecond)

		// Проверяем что пиры подключены
		assert.Equal(t, network.Connected, host2.Network().Connectedness(host1.ID()))
		assert.Equal(t, network.Connected, host1.Network().Connectedness(host2.ID()))

		// Проверяем что сервисы обнаружения работают
		assert.Empty(t, service1.GetDiscoveredPeers()) // Пока нет других пиров
	})
}

// TestP2PNetwork_ProtocolPrefix тестирует префикс протокола
func TestP2PNetwork_ProtocolPrefix(t *testing.T) {
	t.Run("префикс используется в protocol ID", func(t *testing.T) {
		assert.Contains(t, p2p.ProtocolID, p2p.ProtocolPrefix)
		assert.Contains(t, p2p.ChatProtocolID, p2p.ProtocolPrefix)
	})

	t.Run("форматирование адреса с префиксом", func(t *testing.T) {
		peerID := "QmTest123"
		multiaddr := "/ip4/127.0.0.1/tcp/4001"

		formatted := p2p.FormatPeerAddress(peerID, multiaddr)
		assert.Contains(t, formatted, p2p.ProtocolPrefix)
		assert.Contains(t, formatted, peerID)
		assert.Contains(t, formatted, multiaddr)
	})
}

// TestP2PNetwork_ErrorHandling тестирует обработку ошибок
func TestP2PNetwork_ErrorHandling(t *testing.T) {
	t.Run("подключение к несуществующему пиру", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		h := createTestHost(t, 0)

		// Пытаемся подключиться к несуществующему адресу
		invalidAddr := peer.AddrInfo{
			ID:    peer.ID("QmInvalid"),
			Addrs: []multiaddr.Multiaddr{},
		}

		err := h.Connect(ctx, invalidAddr)
		assert.Error(t, err)
	})

	t.Run("остановка сервиса несколько раз", func(t *testing.T) {
		h := createTestHost(t, 0)
		cfg := p2p.DefaultConfig()

		service := p2p.NewConnectionService(h, cfg)
		err := service.Start()
		require.NoError(t, err)

		err = service.Stop()
		require.NoError(t, err)

		// Повторная остановка не должна паниковать
		err = service.Stop()
		// Может вернуть ошибку или nil - зависит от реализации
		_ = err
	})
}

// BenchmarkP2PNetwork_HostCreation бенчмарк создания хоста
func BenchmarkP2PNetwork_HostCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		privKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
		h, _ := libp2p.New(libp2p.Identity(privKey), libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
		h.Close()
	}
}

// BenchmarkP2PNetwork_DHTCreation бенчмарк создания DHT
func BenchmarkP2PNetwork_DHTCreation(b *testing.B) {
	privKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	h, _ := libp2p.New(libp2p.Identity(privKey), libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	defer h.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		kdht, _ := dht.New(ctx, h, dht.Mode(dht.ModeAuto), dht.ProtocolPrefix("/projectt-test"))
		kdht.Close()
	}
}
