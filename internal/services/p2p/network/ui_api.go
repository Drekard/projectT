// Package network предоставляет UI API для доступа к P2P функциональности
package network

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"projectT/internal/services/p2p"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
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

	api.network.config.ListenPort = settings.ListenPort
	api.network.config.EnableNATPortMap = settings.EnableNATPortMap
	api.network.config.EnableRelay = settings.EnableRelay
	api.network.config.EnableAutoRelay = settings.EnableAutoRelay
	api.network.config.EnableDHT = settings.EnableDHT
	api.network.config.EnableMDNS = settings.EnableMDNS
	api.network.config.EnableSTUNClient = settings.EnableSTUN
	api.network.config.STUNServer = settings.STUNServer
	api.network.config.EnableHelperMode = settings.EnableHelperMode

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

	// Пробуем подключиться к пиру для получения профиля
	ctx, cancel := context.WithTimeout(api.network.ctx, 10*time.Second)
	defer cancel()

	// Подключаемся к пиру
	if err := p2p.ConnectToPeer(ctx, api.network.host, addrStr); err != nil {
		// Подключение не удалось, но контакт всё равно создан
		// Профиль будет запрошен позже при следующей попытке
		log.Printf("Не удалось подключиться к пиру %s: %v", peerID.String(), err)
	} else {
		// Подключение успешно — запрашиваем профиль
		if api.network.profileExchange != nil {
			go func() {
				profileCtx, profileCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer profileCancel()

				// Запрашиваем профиль у пира
				profileWithSig, err := api.network.profileExchange.RequestPeerProfile(profileCtx, peerID)
				if err != nil {
					log.Printf("Не удалось получить профиль у пира %s: %v", peerID.String(), err)
					return
				}

				// Обновляем контакт с данными профиля
				if profileWithSig != nil && profileWithSig.Profile != nil {
					if err := queries.UpdateContactProfile(peerID.String(), profileWithSig.Profile.Username, profileWithSig.Profile.AvatarPath); err != nil {
						log.Printf("Не удалось обновить контакт: %v", err)
					} else {
						log.Printf("Профиль пира %s получен и сохранён: %s", peerID.String(), profileWithSig.Profile.Username)
					}
				}
			}()
		}
	}

	// Обновляем имя контакта если указано пользователем
	if username != "" {
		if err := queries.UpdateContactByPeerID(peerID.String(), username, ""); err != nil {
			log.Printf("Не удалось обновить имя контакта: %v", err)
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

	return p2p.ConnectToPeer(ctx, api.network.host, addrStr)
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
			Status:      contact.Status,
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
