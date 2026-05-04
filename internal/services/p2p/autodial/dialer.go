// Package autodial предоставляет сервисы для автоматического подключения к пирам
package autodial

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

// Config конфигурация автоподключения
type Config struct {
	MaxConcurrentConnections int           // Максимум одновременных подключений
	ConnectionTimeout        time.Duration // Таймаут подключения
	ReconnectInterval        time.Duration // Интервал переподключения
	MaxReconnectAttempts     int           // Максимум попыток переподключения
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrentConnections: 50,
		ConnectionTimeout:        15 * time.Second,
		ReconnectInterval:        30 * time.Second,
		MaxReconnectAttempts:     5,
	}
}

// PeerAddress представляет адрес пира для подключения
type PeerAddress struct {
	PeerID      string
	Multiaddr   string
	AddressType string // bootstrap, contact, discovered
	Priority    int
}

// Dialer сервис для автоматического подключения к пирам
type Dialer struct {
	mu                sync.RWMutex
	host              host.Host
	config            *Config
	ctx               context.Context
	cancel            context.CancelFunc
	reconnectQueue    []peer.ID
	reconnectAttempts map[peer.ID]int
	connectedCount    int
}

// NewDialer создаёт новый автодиалер
func NewDialer(h host.Host, config *Config) *Dialer {
	if config == nil {
		config = DefaultConfig()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Dialer{
		host:              h,
		config:            config,
		ctx:               ctx,
		cancel:            cancel,
		reconnectQueue:    make([]peer.ID, 0),
		reconnectAttempts: make(map[peer.ID]int),
	}
}

// Start запускает автоподключение
func (d *Dialer) Start() error {
	return nil
}

// Stop останавливает автоподключение
func (d *Dialer) Stop() error {
	d.cancel()
	return nil
}

// DialResult результат подключения
type DialResult struct {
	PeerID  peer.ID
	Success bool
	Error   error
}

// DialMany подключается к множеству пиров параллельно
func (d *Dialer) DialMany(addresses []*PeerAddress) <-chan DialResult {
	results := make(chan DialResult, len(addresses))

	d.mu.RLock()
	currentConnected := d.connectedCount
	maxToConnect := d.config.MaxConcurrentConnections - currentConnected
	d.mu.RUnlock()

	if maxToConnect <= 0 {
		log.Printf("[Autodial] ⚠️ Достигнут лимит подключений: %d/%d", currentConnected, d.config.MaxConcurrentConnections)
		close(results)
		return results
	}

	// Ограничиваем количество подключений
	if len(addresses) > maxToConnect {
		log.Printf("[Autodial] Ограничиваем подключение: %d → %d (лимит)", len(addresses), maxToConnect)
		addresses = addresses[:maxToConnect]
	}

	log.Printf("[Autodial] Подключение к %d пирам (лимит: %d)", len(addresses), maxToConnect)

	// Подключаемся параллельно
	for _, addr := range addresses {
		go func(addr *PeerAddress) {
			result := d.dialOne(addr)
			results <- result
		}(addr)
	}

	return results
}

// dialOne подключается к одному пиру
func (d *Dialer) dialOne(addr *PeerAddress) DialResult {
	peerID, err := peer.Decode(addr.PeerID)
	if err != nil {
		log.Printf("[Autodial] ❌ Ошибка декодирования PeerID %s: %v", addr.PeerID, err)
		return DialResult{Success: false, Error: err}
	}

	ma, err := multiaddr.NewMultiaddr(addr.Multiaddr)
	if err != nil {
		log.Printf("[Autodial] ❌ Ошибка парсинга адреса %s: %v", addr.Multiaddr, err)
		return DialResult{Success: false, Error: err}
	}

	// Добавляем адрес в peerstore
	d.host.Peerstore().AddAddr(peerID, ma, peerstore.PermanentAddrTTL)

	log.Printf("[Autodial] Подключение к %s (%s) через %s...", addr.PeerID[:8], addr.AddressType, ma.String())

	// Подключаемся
	ctx, cancel := context.WithTimeout(d.ctx, d.config.ConnectionTimeout)
	defer cancel()

	startTime := time.Now()
	err = d.host.Connect(ctx, peer.AddrInfo{
		ID:    peerID,
		Addrs: []multiaddr.Multiaddr{ma},
	})
	elapsed := time.Since(startTime)

	if err != nil {
		log.Printf("[Autodial] ❌ Подключение к %s (%s) за %v: %v", addr.PeerID[:8], addr.AddressType, elapsed, err)

		// Анализируем ошибку
		errStr := err.Error()
		if strings.Contains(errStr, "timeout") {
			log.Printf("[Autodial]   💡 Timeout: пир может быть за NAT или офлайн")
		} else if strings.Contains(errStr, "refused") {
			log.Printf("[Autodial]   💡 Connection refused: порт закрыт или брандмауэр блокирует")
		}

		// Добавляем в очередь переподключения если это bootstrap/contact
		if addr.AddressType == "bootstrap" || addr.AddressType == "contact" {
			d.addToReconnectQueue(peerID)
		}

		return DialResult{PeerID: peerID, Success: false, Error: err}
	}

	log.Printf("[Autodial] ✅ Подключение к %s (%s) успешно за %v", addr.PeerID[:8], addr.AddressType, elapsed)

	// Определяем тип соединения
	conns := d.host.Network().ConnsToPeer(peerID)
	for i, conn := range conns {
		remoteAddr := conn.RemoteMultiaddr()
		connType := "DIRECT"
		if strings.Contains(remoteAddr.String(), "/p2p-circuit") {
			connType = "RELAYED"
		}
		log.Printf("[Autodial]   Соединение #%d: тип=%s, адрес=%s", i+1, connType, remoteAddr.String())
	}

	d.mu.Lock()
	d.connectedCount++
	d.mu.Unlock()

	return DialResult{PeerID: peerID, Success: true}
}

// AddToReconnectQueue добавляет пира в очередь переподключения
func (d *Dialer) AddToReconnectQueue(peerID peer.ID) {
	d.addToReconnectQueue(peerID)
}

// addToReconnectQueue внутренняя функция добавления в очередь
func (d *Dialer) addToReconnectQueue(peerID peer.ID) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Проверяем, есть ли уже в очереди
	for _, p := range d.reconnectQueue {
		if p == peerID {
			return
		}
	}

	d.reconnectQueue = append(d.reconnectQueue, peerID)
	log.Printf("[Autodial] Добавлен в очередь на переподключение: %s", peerID.String()[:8])
}

// ProcessReconnectQueue обрабатывает очередь переподключения
func (d *Dialer) ProcessReconnectQueue() {
	d.mu.Lock()
	if len(d.reconnectQueue) == 0 {
		d.mu.Unlock()
		return
	}

	peerID := d.reconnectQueue[0]
	d.reconnectQueue = d.reconnectQueue[1:]
	d.mu.Unlock()

	d.attemptReconnect(peerID)
}

// attemptReconnect пытается переподключиться к пиру
func (d *Dialer) attemptReconnect(peerID peer.ID) {
	d.mu.RLock()
	attempts := d.reconnectAttempts[peerID]
	d.mu.RUnlock()

	if attempts >= d.config.MaxReconnectAttempts {
		log.Printf("[Autodial] Превышено количество попыток переподключения к %s (%d)", peerID.String()[:8], attempts)
		return
	}

	d.mu.Lock()
	d.reconnectAttempts[peerID] = attempts + 1
	d.mu.Unlock()

	log.Printf("[Autodial] Переподключение к %s (попытка %d/%d)...", peerID.String()[:8], attempts+1, d.config.MaxReconnectAttempts)

	ctx, cancel := context.WithTimeout(d.ctx, d.config.ConnectionTimeout)
	defer cancel()

	addrs := d.host.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		log.Printf("[Autodial] Нет адресов для переподключения к %s", peerID.String()[:8])
		return
	}

	log.Printf("[Autodial] Доступные адреса для %s:", peerID.String()[:8])
	for _, a := range addrs {
		log.Printf("[Autodial]   - %s", a.String())
	}

	startTime := time.Now()
	err := d.host.Connect(ctx, peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	})
	elapsed := time.Since(startTime)

	d.mu.Lock()
	defer d.mu.Unlock()

	if err != nil {
		log.Printf("[Autodial] Переподключение к %s не удалось за %v: %v", peerID.String()[:8], elapsed, err)
		d.reconnectQueue = append(d.reconnectQueue, peerID)
	} else {
		log.Printf("[Autodial] Переподключение к %s успешно за %v", peerID.String()[:8], elapsed)

		// Определяем тип соединения
		conns := d.host.Network().ConnsToPeer(peerID)
		for i, conn := range conns {
			remoteAddr := conn.RemoteMultiaddr()
			connType := "DIRECT"
			if strings.Contains(remoteAddr.String(), "/p2p-circuit") {
				connType = "RELAYED"
			}
			log.Printf("[Autodial]   Соединение #%d: тип=%s, адрес=%s", i+1, connType, remoteAddr.String())
		}

		d.reconnectAttempts[peerID] = 0
		d.connectedCount++
	}
}

// GetConnectedCount возвращает количество подключений
func (d *Dialer) GetConnectedCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connectedCount
}

// CanConnect больше ли можно подключаться
func (d *Dialer) CanConnect() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connectedCount < d.config.MaxConcurrentConnections
}

// GetReconnectQueueLength возвращает длину очереди переподключения
func (d *Dialer) GetReconnectQueueLength() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.reconnectQueue)
}
