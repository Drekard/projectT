// Package autodial предоставляет сервисы для автоматического подключения к пирам
package autodial

import (
	"context"
	"fmt"
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
	// Пробуем распарсить как multiaddr
	ma, maErr := multiaddr.NewMultiaddr(addr.Multiaddr)
	if maErr == nil {
		return d.dialWithMultiaddr(addr, ma)
	}

	// Если не multiaddr — пробуем компактный формат: projectt:PeerID@ip:port;ip2:port2
	if strings.Contains(addr.Multiaddr, "@") {
		return d.dialWithCompactAddress(addr)
	}

	log.Printf("[Autodial] ❌ Ошибка парсинга адреса %s: %v", addr.Multiaddr, maErr)
	return DialResult{Success: false, Error: maErr}
}

// dialWithMultiaddr подключается используя готовый multiaddr
func (d *Dialer) dialWithMultiaddr(addr *PeerAddress, ma multiaddr.Multiaddr) DialResult {
	peerID, err := peer.Decode(addr.PeerID)
	if err != nil {
		log.Printf("[Autodial] ❌ Ошибка декодирования PeerID %s: %v", addr.PeerID, err)
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

// dialWithCompactAddress подключается используя компактный формат адреса
func (d *Dialer) dialWithCompactAddress(addr *PeerAddress) DialResult {
	addrStr := addr.Multiaddr

	// Удаляем префикс projectt:
	for strings.HasPrefix(addrStr, "projectt://") || strings.HasPrefix(addrStr, "projectt:") {
		addrStr = strings.TrimPrefix(addrStr, "projectt://")
		addrStr = strings.TrimPrefix(addrStr, "projectt:")
	}

	// Парсим PeerID@endpoints
	parts := strings.SplitN(addrStr, "@", 2)
	if len(parts) != 2 {
		return DialResult{Success: false, Error: fmt.Errorf("неверный формат адреса")}
	}

	targetPeerID, err := peer.Decode(parts[0])
	if err != nil {
		log.Printf("[Autodial] ❌ Ошибка декодирования PeerID %s: %v", parts[0], err)
		return DialResult{Success: false, Error: err}
	}

	// Парсим endpoints: ip1:port1;ip2:port2;[ipv6]:port
	endpoints := strings.Split(parts[1], ";")
	var publicAddrs, lanAddrs, loopbackAddrs []multiaddr.Multiaddr
	for _, ep := range endpoints {
		ep = strings.TrimSpace(ep)
		if ep == "" {
			continue
		}
		ma, err := endpointToMultiaddr(ep, parts[0])
		if err != nil {
			continue
		}
		// Сортируем адреса: публичные > LAN > loopback
		if isLoopbackEndpoint(ep) {
			loopbackAddrs = append(loopbackAddrs, ma)
		} else if isLANEndpoint(ep) {
			lanAddrs = append(lanAddrs, ma)
		} else {
			publicAddrs = append(publicAddrs, ma)
		}
	}

	// Объединяем в порядке приоритета: публичные, затем LAN, затем loopback
	var addrs []multiaddr.Multiaddr
	addrs = append(addrs, publicAddrs...)
	addrs = append(addrs, lanAddrs...)
	addrs = append(addrs, loopbackAddrs...)

	if len(addrs) == 0 {
		return DialResult{PeerID: targetPeerID, Success: false, Error: fmt.Errorf("нет валидных адресов")}
	}

	// Добавляем все адреса в peerstore
	for _, a := range addrs {
		d.host.Peerstore().AddAddr(targetPeerID, a, peerstore.PermanentAddrTTL)
	}

	log.Printf("[Autodial] Подключение к %s (%s), адресов: %d (public: %d, lan: %d, loopback: %d)...", targetPeerID.String()[:8], addr.AddressType, len(addrs), len(publicAddrs), len(lanAddrs), len(loopbackAddrs))

	ctx, cancel := context.WithTimeout(d.ctx, d.config.ConnectionTimeout)
	defer cancel()

	startTime := time.Now()
	err = d.host.Connect(ctx, peer.AddrInfo{
		ID:    targetPeerID,
		Addrs: addrs,
	})
	elapsed := time.Since(startTime)

	if err != nil {
		log.Printf("[Autodial] ❌ Подключение к %s (%s) за %v: %v", targetPeerID.String()[:8], addr.AddressType, elapsed, err)

		errStr := err.Error()
		if strings.Contains(errStr, "timeout") {
			log.Printf("[Autodial]   💡 Timeout: пир может быть за NAT или офлайн")
		} else if strings.Contains(errStr, "refused") {
			log.Printf("[Autodial]   💡 Connection refused: порт закрыт или брандмауэр блокирует")
		}

		if addr.AddressType == "bootstrap" || addr.AddressType == "contact" {
			d.addToReconnectQueue(targetPeerID)
		}

		return DialResult{PeerID: targetPeerID, Success: false, Error: err}
	}

	log.Printf("[Autodial] ✅ Подключение к %s (%s) успешно за %v", targetPeerID.String()[:8], addr.AddressType, elapsed)

	conns := d.host.Network().ConnsToPeer(targetPeerID)
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

	return DialResult{PeerID: targetPeerID, Success: true}
}

// isLoopbackEndpoint проверяет, является ли endpoint loopback адресом
func isLoopbackEndpoint(ep string) bool {
	return strings.HasPrefix(ep, "127.0.0.1") || strings.HasPrefix(ep, "[::1]") || strings.HasPrefix(ep, "localhost")
}

// isLANEndpoint проверяет, является ли endpoint LAN адресом
func isLANEndpoint(ep string) bool {
	return strings.HasPrefix(ep, "192.168.") ||
		strings.HasPrefix(ep, "10.") ||
		(strings.HasPrefix(ep, "172.") && is172Private(ep))
}

// is172Private проверяет, является ли 172.x.x.x частным адресом (172.16.0.0 - 172.31.255.255)
func is172Private(ep string) bool {
	parts := strings.SplitN(ep, ":", 2)
	if len(parts) < 1 {
		return false
	}
	ipParts := strings.Split(parts[0], ".")
	if len(ipParts) < 2 {
		return false
	}
	var secondOctet int
	_, _ = fmt.Sscanf(ipParts[1], "%d", &secondOctet)
	return secondOctet >= 16 && secondOctet <= 31
}

// endpointToMultiaddr конвертирует "ip:port" или "[ipv6]:port" в multiaddr
func endpointToMultiaddr(endpoint, peerIDStr string) (multiaddr.Multiaddr, error) {
	var ip, port string

	if strings.HasPrefix(endpoint, "[") {
		closeIdx := strings.Index(endpoint, "]")
		if closeIdx == -1 {
			return nil, fmt.Errorf("неверный IPv6 формат: %s", endpoint)
		}
		ip = endpoint[1:closeIdx]
		if closeIdx+2 >= len(endpoint) || endpoint[closeIdx+1] != ':' {
			return nil, fmt.Errorf("нет порта после IPv6: %s", endpoint)
		}
		port = endpoint[closeIdx+2:]
	} else {
		lastColon := strings.LastIndex(endpoint, ":")
		if lastColon == -1 {
			return nil, fmt.Errorf("нет порта: %s", endpoint)
		}
		ip = endpoint[:lastColon]
		port = endpoint[lastColon+1:]
	}

	protocol := "ip4"
	if strings.Contains(ip, ":") {
		protocol = "ip6"
	}

	maStr := fmt.Sprintf("/%s/%s/tcp/%s/p2p/%s", protocol, ip, port, peerIDStr)
	return multiaddr.NewMultiaddr(maStr)
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

// DecrementConnectedCount уменьшает счётчик подключений (вызывается при отключении пира)
func (d *Dialer) DecrementConnectedCount() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.connectedCount > 0 {
		d.connectedCount--
	}
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
