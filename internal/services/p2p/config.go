// Package p2p содержит сервисы для P2P связи на базе libp2p.
package p2p

import (
	"time"
)

// ProtocolID идентификатор протокола ProjectT
const ProtocolID = "/projectt/1.0.0"

// ChatProtocolID идентификатор протокола чата
const ChatProtocolID = "/projectt/chat/1.0.0"

// P2PConfig конфигурация P2P сети
type P2PConfig struct {
	// ListenPort порт для прослушивания входящих соединений
	ListenPort int

	// ListenAddrs адреса для прослушивания
	ListenAddrs []string

	// ExternalAddrs внешние адреса для объявления
	ExternalAddrs []string

	// EnableNATPortMap включить проброс портов через NAT (UPnP/NAT-PMP)
	EnableNATPortMap bool

	// EnableRelay включить ретрансляцию через relay-узлы
	EnableRelay bool

	// EnableAutoRelay включить автоматический выбор relay
	EnableAutoRelay bool

	// EnableDHT включить распределённую хеш-таблицу
	EnableDHT bool

	// EnableMDNS включить mDNS для локальной сети
	EnableMDNS bool

	// ConnectionTimeout таймаут подключения
	ConnectionTimeout time.Duration

	// DialTimeout таймаут установки соединения
	DialTimeout time.Duration

	// KeepAlive интервал keep-alive сообщений
	KeepAlive time.Duration

	// MaxConnections максимальное количество соединений
	MaxConnections int

	// MinConnections минимальное количество соединений
	MinConnections int

	// BootstrapPeers список bootstrap-узлов
	BootstrapPeers []string
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *P2PConfig {
	return &P2PConfig{
		ListenPort:       0, // 0 = случайный порт
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip6/::/tcp/0",
		},
		ExternalAddrs:      []string{},
		EnableNATPortMap:   true,
		EnableRelay:        true,
		EnableAutoRelay:    true,
		EnableDHT:          true,
		EnableMDNS:         true,
		ConnectionTimeout:  30 * time.Second,
		DialTimeout:        10 * time.Second,
		KeepAlive:          30 * time.Second,
		MaxConnections:     100,
		MinConnections:     5,
		BootstrapPeers:     []string{},
	}
}
