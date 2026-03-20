// Package network предоставляет UI API для доступа к P2P функциональности
package network

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"projectT/internal/services/p2p"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// P2PStatus статус P2P подключения
type P2PStatus struct {
	IsRunning      bool   `json:"is_running"`
	PeerID         string `json:"peer_id"`
	ListenPort     int    `json:"listen_port"`
	ConnectedPeers int    `json:"connected_peers"`
	PublicAddress  string `json:"public_address"`
	NATStatus      string `json:"nat_status"`
	RelayEnabled   bool   `json:"relay_enabled"`
	DHTEnabled     bool   `json:"dht_enabled"`
	MDNSEnabled    bool   `json:"mdns_enabled"`
	HelperMode     bool   `json:"helper_mode"`
}

// PeerInfo информация о пире для UI
type PeerInfo struct {
	PeerID      string    `json:"peer_id"`
	Username    string    `json:"username"`
	Status      string    `json:"status"`
	IsConnected bool      `json:"is_connected"`
	LastSeen    time.Time `json:"last_seen"`
	LatencyMs   int64     `json:"latency_ms"`
	Address     string    `json:"address"`
}

// NATStatusInfo информация о NAT
type NATStatusInfo struct {
	HasPublicAddr bool   `json:"has_public_addr"`
	UPnPEnabled   bool   `json:"upnp_enabled"`
	Message       string `json:"message"`
}

// FirewallInfo информация о брандмауэре
type FirewallInfo struct {
	Port          int    `json:"port"`
	IsOpen        bool   `json:"is_open"`
	RuleName      string `json:"rule_name"`
	PowerShellCmd string `json:"powershell_cmd"`
	CMDCmd        string `json:"cmd_cmd"`
}

// P2PSettings настройки P2P
type P2PSettings struct {
	ListenPort       int    `json:"listen_port"`
	EnableNATPortMap bool   `json:"enable_nat_port_map"`
	EnableRelay      bool   `json:"enable_relay"`
	EnableAutoRelay  bool   `json:"enable_auto_relay"`
	EnableDHT        bool   `json:"enable_dht"`
	EnableMDNS       bool   `json:"enable_mdns"`
	EnableSTUN       bool   `json:"enable_stun"`
	STUNServer       string `json:"stun_server"`
	EnableHelperMode bool   `json:"enable_helper_mode"`
}

// UIP2P API для доступа к P2P из UI
type UIP2P struct {
	network *P2PNetwork
}

// NewUIP2P создаёт UI API для P2P
func NewUIP2P(network *P2PNetwork) *UIP2P {
	return &UIP2P{
		network: network,
	}
}

// GetStatus возвращает текущий статус P2P
func (api *UIP2P) GetStatus() *P2PStatus {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	status := &P2PStatus{
		IsRunning:      api.network.host != nil,
		RelayEnabled:   api.network.config.EnableRelay,
		DHTEnabled:     api.network.config.EnableDHT,
		MDNSEnabled:    api.network.config.EnableMDNS,
		HelperMode:     api.network.config.EnableHelperMode,
		ListenPort:     api.network.config.ListenPort,
		ConnectedPeers: 0,
	}

	if api.network.host != nil {
		status.PeerID = api.network.host.ID().String()
		status.ConnectedPeers = len(api.network.host.Network().Peers())

		// Получаем публичный адрес
		if addrInfo, err := p2p.GeneratePublicAddress(api.network.host, api.network.config.ListenPort); err == nil {
			status.PublicAddress = addrInfo.FullAddress
		}

		// Получаем NAT статус
		natStatus := p2p.GetNATStatus(api.network.host)
		status.NATStatus = natStatus.Message
	}

	return status
}

// GetSettings возвращает текущие настройки P2P
func (api *UIP2P) GetSettings() *P2PSettings {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	return &P2PSettings{
		ListenPort:       api.network.config.ListenPort,
		EnableNATPortMap: api.network.config.EnableNATPortMap,
		EnableRelay:      api.network.config.EnableRelay,
		EnableAutoRelay:  api.network.config.EnableAutoRelay,
		EnableDHT:        api.network.config.EnableDHT,
		EnableMDNS:       api.network.config.EnableMDNS,
		EnableSTUN:       api.network.config.EnableSTUNClient,
		STUNServer:       api.network.config.STUNServer,
		EnableHelperMode: api.network.config.EnableHelperMode,
	}
}

// UpdateSettings обновляет настройки P2P
func (api *UIP2P) UpdateSettings(settings *P2PSettings) error {
	api.network.mu.Lock()
	defer api.network.mu.Unlock()

	// Проверяем, изменились ли настройки
	settingsChanged := api.network.config.ListenPort != settings.ListenPort ||
		api.network.config.EnableNATPortMap != settings.EnableNATPortMap ||
		api.network.config.EnableRelay != settings.EnableRelay ||
		api.network.config.EnableAutoRelay != settings.EnableAutoRelay ||
		api.network.config.EnableDHT != settings.EnableDHT ||
		api.network.config.EnableMDNS != settings.EnableMDNS ||
		api.network.config.EnableSTUNClient != settings.EnableSTUN ||
		api.network.config.STUNServer != settings.STUNServer ||
		api.network.config.EnableHelperMode != settings.EnableHelperMode

	// Сохраняем настройки
	api.network.config.ListenPort = settings.ListenPort
	api.network.config.EnableNATPortMap = settings.EnableNATPortMap
	api.network.config.EnableRelay = settings.EnableRelay
	api.network.config.EnableAutoRelay = settings.EnableAutoRelay
	api.network.config.EnableDHT = settings.EnableDHT
	api.network.config.EnableMDNS = settings.EnableMDNS
	api.network.config.EnableSTUNClient = settings.EnableSTUN
	api.network.config.STUNServer = settings.STUNServer
	api.network.config.EnableHelperMode = settings.EnableHelperMode

	log.Println("Настройки P2P сохранены")

	// Если настройки изменились и хост запущен - требуется перезапуск
	if settingsChanged && api.network.host != nil {
		log.Println("Настройки изменились - требуется перезапуск P2P для применения")
		// TODO: реализовать автоматический рестарт или показать уведомление пользователю
	}

	// TODO: сохранить настройки в БД когда будет реализовано
	return nil
}

// GetNATStatus возвращает информацию о NAT
func (api *UIP2P) GetNATStatus() *NATStatusInfo {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.host == nil {
		return &NATStatusInfo{
			HasPublicAddr: false,
			UPnPEnabled:   false,
			Message:       "P2P не запущен",
		}
	}

	info := p2p.GetNATStatus(api.network.host)
	return &NATStatusInfo{
		HasPublicAddr: info.HasPublicAddr,
		UPnPEnabled:   info.UPnPEnabled,
		Message:       info.Message,
	}
}

// CheckFirewall проверяет доступность порта в брандмауэре
func (api *UIP2P) CheckFirewall(port int) *FirewallInfo {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	info := p2p.GenerateFirewallRule(port, "ProjectT P2P")
	return &FirewallInfo{
		Port:          port,
		IsOpen:        false, // Требуется ручная проверка
		RuleName:      info.RuleName,
		PowerShellCmd: info.PowerShell,
		CMDCmd:        info.CMD,
	}
}

// OpenFirewall пытается открыть порт в брандмауэре
func (api *UIP2P) OpenFirewall(port int, ruleName string) (bool, string, error) {
	result, err := p2p.OpenFirewallRule(port, ruleName)
	if err != nil {
		return false, "", err
	}
	return result.Success, result.Message, nil
}

// GetPeerAddress возвращает адрес текущего пира для экспорта
func (api *UIP2P) GetPeerAddress() (string, error) {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.host == nil {
		return "", fmt.Errorf("P2P не запущен")
	}

	addr, err := p2p.GetPeerAddress(api.network.host)
	if err != nil {
		return "", err
	}
	return p2p.FormatPeerAddress(addr.PeerID, addr.Multiaddr), nil
}

// CopyPeerAddress копирует адрес пира в буфер обмена
func (api *UIP2P) CopyPeerAddress() (string, error) {
	return api.GetPeerAddress()
}

// AddContactByAddress добавляет контакт по адресу
func (api *UIP2P) AddContactByAddress(addrStr, username string) error {
	api.network.mu.Lock()
	defer api.network.mu.Unlock()

	if api.network.host == nil {
		return fmt.Errorf("P2P не запущен")
	}

	// Импортируем адрес пира и добавляем в peerstore
	peerAddr, err := p2p.ImportPeerAddress(api.network.host, addrStr)
	if err != nil {
		return fmt.Errorf("ошибка импорта адреса: %w", err)
	}

	// Получаем PeerID пира
	peerID, err := peer.Decode(peerAddr.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	log.Printf("=== AddContactByAddress ===")
	log.Printf("PeerID: %s", peerID.String())
	log.Printf("Адрес: %s", addrStr)

	// Пробуем подключиться к пиру для получения профиля
	// Увеличиваем таймаут до 60 секунд для подключения через NAT
	ctx, cancel := context.WithTimeout(api.network.ctx, 60*time.Second)
	defer cancel()

	// Подключаемся к пиру
	log.Printf("Попытка подключения к пиру...")
	if err := p2p.ConnectToPeer(ctx, api.network.host, addrStr); err != nil {
		// Подключение не удалось, но контакт всё равно создан
		// Профиль будет запрошен позже при следующей попытке
		log.Printf("❌ Не удалось подключиться к пиру %s: %v", peerID.String(), err)
	} else {
		// Подключение успешно — запрашиваем профиль
		log.Printf("✅ Подключение успешно, запрашиваем профиль...")
		log.Printf("profileExchange = %v", api.network.profileExchange != nil)

		if api.network.profileExchange == nil {
			log.Printf("❌ profileExchange не инициализирован!")
		} else {
			go func() {
				log.Printf("[ГОРУТИНКА] Запуск запроса профиля...")
				// Увеличиваем таймаут для запроса профиля до 30 секунд
				profileCtx, profileCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer profileCancel()

				// Запрашиваем профиль у пира
				log.Printf("Запрос профиля у пира %s...", peerID.String())
				profileWithSig, err := api.network.profileExchange.RequestPeerProfile(profileCtx, peerID)
				if err != nil {
					log.Printf("❌ Не удалось получить профиль у пира %s: %v", peerID.String(), err)
					return
				}

				// Обновляем remote профиль в БД
				if profileWithSig != nil && profileWithSig.Profile != nil {
					if err := queries.UpdateRemoteProfile(profileWithSig.Profile); err != nil {
						log.Printf("Не удалось обновить профиль пира %s: %v", peerID.String(), err)
					} else {
						log.Printf("✅ Профиль пира %s получен и сохранён: %s", peerID.String(), profileWithSig.Profile.Username)
					}
				}
			}()
		}
	}

	// Обновляем multiaddr контакта
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil && contact.Multiaddr != "" {
		if err := queries.UpdateContactByPeerID(peerID.String(), contact.Multiaddr); err != nil {
			log.Printf("Не удалось обновить multiaddr контакта: %v", err)
		}
	}

	return nil
}

// ConnectToContact подключается к контакту по адресу
func (api *UIP2P) ConnectToContact(addrStr string) error {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.host == nil {
		return fmt.Errorf("P2P не запущен")
	}

	ctx, cancel := context.WithTimeout(api.network.ctx, 30*time.Second)
	defer cancel()

	// Подключаемся к пиру
	if err := p2p.ConnectToPeer(ctx, api.network.host, addrStr); err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}

	// Получаем PeerID из адреса
	peerAddr, err := p2p.ImportPeerAddress(api.network.host, addrStr)
	if err != nil {
		return fmt.Errorf("ошибка импорта адреса: %w", err)
	}
	peerID, err := peer.Decode(peerAddr.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	// Запрашиваем профиль после подключения
	go func() {
		log.Printf("[ConnectToContact] Подключение к %s успешно, запрашиваем профиль...", peerID.String()[:8])
		profileCtx, profileCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer profileCancel()

		if api.network.profileExchange == nil {
			log.Printf("[ConnectToContact] ❌ profileExchange не инициализирован!")
			return
		}

		profileWithSig, err := api.network.profileExchange.RequestPeerProfile(profileCtx, peerID)
		if err != nil {
			log.Printf("[ConnectToContact] ❌ Не удалось получить профиль у пира %s: %v", peerID.String(), err)
			return
		}

		// Обновляем remote профиль в БД
		if profileWithSig != nil && profileWithSig.Profile != nil {
			if err := queries.UpdateRemoteProfile(profileWithSig.Profile); err != nil {
				log.Printf("[ConnectToContact] Не удалось обновить профиль пира %s: %v", peerID.String(), err)
			} else {
				log.Printf("[ConnectToContact] ✅ Профиль пира %s получен: %s", peerID.String(), profileWithSig.Profile.Username)
			}
		}
	}()

	return nil
}

// GetConnectedPeers возвращает список подключённых пиров
func (api *UIP2P) GetConnectedPeers() []*PeerInfo {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.host == nil || api.network.connections == nil {
		return []*PeerInfo{}
	}

	var peers []*PeerInfo
	for _, peerID := range api.network.host.Network().Peers() {
		info := api.network.connections.GetPeerInfo(peerID)
		contact, _ := queries.GetContactByPeerID(peerID.String())

		username := peerID.String()[:8]
		if contact != nil {
			username = contact.Username
		}

		latencyMs := int64(0)
		status := ""
		lastSeen := time.Time{}

		if info != nil {
			latencyMs = info.LastPingLatency.Milliseconds()
			status = string(info.Status)
			lastSeen = info.LastSeen
		}

		peers = append(peers, &PeerInfo{
			PeerID:      peerID.String(),
			Username:    username,
			Status:      status,
			IsConnected: true,
			LastSeen:    lastSeen,
			LatencyMs:   latencyMs,
		})
	}

	return peers
}

// GetAllContacts возвращает все контакты с их статусами
func (api *UIP2P) GetAllContacts() []*PeerInfo {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	contacts, err := queries.GetAllContacts()
	if err != nil {
		return []*PeerInfo{}
	}

	var peers []*PeerInfo
	for _, contact := range contacts {
		var peerInfo *p2p.PeerConnectionInfo
		if api.network.connections != nil && contact.PeerID != "" {
			// Пробуем распарсить PeerID
			pid, err := peer.Decode(contact.PeerID)
			if err == nil {
				peerInfo = api.network.connections.GetPeerInfo(pid)
			}
		}

		latencyMs := int64(0)
		isConnected := false
		lastSeen := time.Time{}

		if peerInfo != nil {
			latencyMs = peerInfo.LastPingLatency.Milliseconds()
			isConnected = peerInfo.Status == p2p.StatusConnected
			lastSeen = peerInfo.LastSeen
		}

		peers = append(peers, &PeerInfo{
			PeerID:      contact.PeerID,
			Username:    contact.Username,
			Status:      contact.Title, // Используем Title из profiles
			IsConnected: isConnected,
			LastSeen:    lastSeen,
			LatencyMs:   latencyMs,
			Address:     contact.Multiaddr,
		})
	}

	return peers
}

// GetBootstrapPeers возвращает список bootstrap пиров
func (api *UIP2P) GetBootstrapPeers() []*models.BootstrapPeer {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.discovery == nil {
		return []*models.BootstrapPeer{}
	}

	peers, _ := api.network.discovery.GetBootstrapPeers()
	return peers
}

// AddBootstrapPeer добавляет bootstrap пир
func (api *UIP2P) AddBootstrapPeer(multiaddr string) error {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.discovery == nil {
		return fmt.Errorf("сервис обнаружения не инициализирован")
	}
	return api.network.discovery.AddBootstrapPeer(multiaddr)
}

// RemoveBootstrapPeer удаляет bootstrap пир
func (api *UIP2P) RemoveBootstrapPeer(multiaddr string) error {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.discovery == nil {
		return fmt.Errorf("сервис обнаружения не инициализирован")
	}
	return api.network.discovery.RemoveBootstrapPeer(multiaddr)
}

// GetDiscoveredPeers возвращает список обнаруженных пиров
func (api *UIP2P) GetDiscoveredPeers() map[string]time.Time {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.discovery == nil {
		return make(map[string]time.Time)
	}
	return api.network.discovery.GetDiscoveredPeers()
}

// StartPeerDiscovery запускает обнаружение пиров (DHT + bootstrap)
func (api *UIP2P) StartPeerDiscovery() error {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.discovery == nil {
		return fmt.Errorf("сервис обнаружения не инициализирован")
	}
	return api.network.discovery.StartDiscovery()
}

// GetHelperPeers возвращает список пиров из helper режима
func (api *UIP2P) GetHelperPeers() []p2p.PeerEntry {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.helper == nil || api.network.helper.helper == nil {
		return []p2p.PeerEntry{}
	}
	return api.network.helper.helper.List()
}

// RequestProfile запрашивает профиль у пира
func (api *UIP2P) RequestProfile(peerIDStr string) error {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.profileExchange == nil {
		return fmt.Errorf("сервис обмена профилями не инициализирован")
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(api.network.ctx, 10*time.Second)
	defer cancel()

	_, err = api.network.profileExchange.RequestPeerProfile(ctx, peerID)
	return err
}

// RequestAllProfiles запрашивает профили у всех контактов
func (api *UIP2P) RequestAllProfiles() {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.profileExchange == nil {
		return
	}

	ctx, cancel := context.WithTimeout(api.network.ctx, 30*time.Second)
	defer cancel()

	api.network.profileExchange.RequestProfilesForAllContacts(ctx)
}

// GetPeerID декодирует PeerID из строки
func (api *UIP2P) GetPeerID(peerIDStr string) (peer.ID, error) {
	return peer.Decode(peerIDStr)
}

// SendMessage отправляет текстовое сообщение пиру
func (api *UIP2P) SendMessage(peerID peer.ID, content string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return api.network.SendTextMessage(ctx, peerID, content)
}

// GetMessagesForContact получает сообщения для контакта
func (api *UIP2P) GetMessagesForContact(contactID, limit, offset int) ([]*models.ChatMessage, error) {
	return api.network.GetMessagesForContact(contactID, limit, offset)
}

// DisconnectPeer отключается от пира
func (api *UIP2P) DisconnectPeer(peerIDStr string) error {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.host == nil {
		return fmt.Errorf("P2P не запущен")
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	// Закрываем все соединения с пиром
	if err := api.network.host.Network().ClosePeer(peerID); err != nil {
		log.Printf("Предупреждение: ошибка при отключении от пира %s: %v", peerID, err)
	}
	log.Printf("Отключено от пира: %s", peerID)

	return nil
}

// ConnectToDiscoveredPeer подключается к обнаруженному пиру по PeerID
func (api *UIP2P) ConnectToDiscoveredPeer(peerIDStr string) error {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	if api.network.host == nil {
		return fmt.Errorf("P2P не запущен")
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	// Получаем адреса из peerstore
	addrs := api.network.host.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		return fmt.Errorf("адреса пира не найдены в peerstore")
	}

	// Создаём peer info
	peerInfo := peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}

	// Подключаемся
	ctx, cancel := context.WithTimeout(api.network.ctx, 30*time.Second)
	defer cancel()

	if err := api.network.host.Connect(ctx, peerInfo); err != nil {
		return fmt.Errorf("ошибка подключения к пиру: %w", err)
	}

	log.Printf("Подключено к пиру: %s", peerID)
	return nil
}

// GetLocalAddresses возвращает список локальных адресов для подключения в одной сети
func (api *UIP2P) GetLocalAddresses() []string {
	api.network.mu.RLock()
	defer api.network.mu.RUnlock()

	var addresses []string

	if api.network.host == nil {
		return addresses
	}

	peerID := api.network.host.ID().String()

	log.Printf("[GetLocalAddresses] PeerID: %s", peerID)

	for _, addr := range api.network.host.Addrs() {
		addrStr := addr.String()
		// Пропускаем localhost и IPv6
		if strings.Contains(addrStr, "127.0.0.1") || strings.Contains(addrStr, "::1") {
			continue
		}
		// Извлекаем IP и порт из multiaddr
		ip, err := addr.ValueForProtocol(multiaddr.P_IP4)
		if err == nil {
			portStr, err := addr.ValueForProtocol(multiaddr.P_TCP)
			if err != nil {
				continue
			}

			var port int
			if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
				log.Printf("[GetLocalAddresses] Ошибка парсинга порта %s: %v", portStr, err)
				continue
			}

			// Пропускаем порт 0
			if port <= 0 {
				continue
			}

			address := fmt.Sprintf("%s:%s@/ip4/%s/tcp/%d/p2p/%s", p2p.ProtocolPrefix, peerID, ip, port, peerID)
			addresses = append(addresses, address)
			log.Printf("[GetLocalAddresses] Адрес: %s", address)
		}
	}

	return addresses
}

// Stop останавливает P2P сеть
func (api *UIP2P) Stop() error {
	return api.network.Stop()
}

// Start запускает P2P сеть
func (api *UIP2P) Start() error {
	return api.network.Start()
}
