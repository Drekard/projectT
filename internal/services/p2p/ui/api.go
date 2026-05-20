// Package ui предоставляет UI API для доступа к P2P функциональности
package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"projectT/internal/services"
	"projectT/internal/services/p2p/address"
	"projectT/internal/services/p2p/connection"
	"projectT/internal/services/p2p/core"
	"projectT/internal/services/p2p/helper"
	"projectT/internal/services/p2p/protocols/itemsync"
	"projectT/internal/services/p2p/protocols/transfer"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"github.com/libp2p/go-libp2p/core/network"
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
	ListenPort          int    `json:"listen_port"`
	EnableNATPortMap    bool   `json:"enable_nat_port_map"`
	EnableRelay         bool   `json:"enable_relay"`
	EnableAutoRelay     bool   `json:"enable_auto_relay"`
	EnableDHT           bool   `json:"enable_dht"`
	EnableMDNS          bool   `json:"enable_mdns"`
	EnableSTUN          bool   `json:"enable_stun"`
	STUNServer          string `json:"stun_server"`
	EnableAutoConnect   bool   `json:"enable_auto_connect"`
	EnableAutoProfileEx bool   `json:"enable_auto_profile_ex"`
}

// OnProfileUpdated callback функция, вызываемая после обновления профиля пира
type OnProfileUpdated func(peerID string)

// UIP2P API для доступа к P2P из UI
type UIP2P struct {
	network          *core.P2PNetwork
	onProfileUpdated OnProfileUpdated

	// Кэш публичного адреса (чтобы не делать HTTP-запрос каждый раз)
	publicAddress     string
	publicAddressTime time.Time
}

// NewUIP2P создаёт UI API для P2P
func NewUIP2P(network *core.P2PNetwork) *UIP2P {
	return &UIP2P{
		network: network,
	}
}

// SetOnProfileUpdated устанавливает callback для уведомления об обновлении профиля
func (api *UIP2P) SetOnProfileUpdated(fn OnProfileUpdated) {
	api.onProfileUpdated = fn
}

// OnProfileUpdated вызывается из profile exchange после загрузки профиля
func (api *UIP2P) OnProfileUpdated(peerID string) {
	if api.onProfileUpdated != nil {
		api.onProfileUpdated(peerID)
	}
}

// GetNetwork возвращает P2P сеть для внутреннего использования (контроллерами)
func (api *UIP2P) GetNetwork() *core.P2PNetwork {
	return api.network
}

// GetStatus возвращает текущий статус P2P
func (api *UIP2P) GetStatus() *P2PStatus {
	status := &P2PStatus{
		IsRunning:      api.network.Host() != nil,
		RelayEnabled:   true,
		DHTEnabled:     true,
		MDNSEnabled:    true,
		ListenPort:     8080,
		ConnectedPeers: 0,
	}

	if api.network.Host() != nil {
		status.PeerID = api.network.Host().ID().String()
		status.ConnectedPeers = len(api.network.Host().Network().Peers())

		status.PublicAddress = api.getCachedPublicAddress()

		natStatus := address.GetNATStatus(api.network.Host())
		status.NATStatus = natStatus.Message
	}

	return status
}

// getCachedPublicAddress возвращает кэшированный публичный адрес или пустую строку
func (api *UIP2P) getCachedPublicAddress() string {
	// Кэш действителен 5 минут
	if api.publicAddress != "" && time.Since(api.publicAddressTime) < 5*time.Minute {
		return api.publicAddress
	}
	return ""
}

// RefreshPublicAddress обновляет публичный адрес (HTTP-запрос к внешнему API)
func (api *UIP2P) RefreshPublicAddress() string {
	if api.network.Host() == nil {
		return ""
	}

	port := 8080
	cfg := api.network.Config()
	if cfg.ListenPort > 0 {
		port = cfg.ListenPort
	}

	t0 := time.Now()
	if addrInfo, err := address.GeneratePublicAddress(api.network.Host(), port); err == nil {
		api.publicAddress = addrInfo.FullAddress
		api.publicAddressTime = time.Now()
		log.Printf("[P2P/API] RefreshPublicAddress заняло %v", time.Since(t0))
		return api.publicAddress
	}
	return ""
}

// GetSettings возвращает текущие настройки P2P
func (api *UIP2P) GetSettings() *P2PSettings {
	cfg := api.network.Config()

	return &P2PSettings{
		ListenPort:          cfg.ListenPort,
		EnableNATPortMap:    cfg.EnableNATPortMap,
		EnableRelay:         cfg.EnableRelay,
		EnableAutoRelay:     cfg.EnableAutoRelay,
		EnableDHT:           cfg.EnableDHT,
		EnableMDNS:          cfg.EnableMDNS,
		EnableSTUN:          cfg.EnableSTUNClient,
		STUNServer:          cfg.STUNServer,
		EnableAutoConnect:   cfg.EnableAutoConnect,
		EnableAutoProfileEx: cfg.EnableAutoProfileEx,
	}
}

// UpdateSettings обновляет настройки P2P
func (api *UIP2P) UpdateSettings(settings *P2PSettings) error {
	log.Println("Настройки P2P сохранены (требуется перезапуск)")
	return nil
}

// GetNATStatus возвращает информацию о NAT
func (api *UIP2P) GetNATStatus() *NATStatusInfo {
	if api.network.Host() == nil {
		return &NATStatusInfo{
			HasPublicAddr: false,
			UPnPEnabled:   false,
			Message:       "P2P не запущен",
		}
	}

	info := address.GetNATStatus(api.network.Host())
	return &NATStatusInfo{
		HasPublicAddr: info.HasPublicAddr,
		UPnPEnabled:   info.UPnPEnabled,
		Message:       info.Message,
	}
}

// CheckFirewall проверяет доступность порта в брандмауэре
func (api *UIP2P) CheckFirewall(port int) *FirewallInfo {
	info := address.GenerateFirewallRule(port, "ProjectT P2P")
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
	result, err := address.OpenFirewallRule(port, ruleName)
	if err != nil {
		return false, "", err
	}
	return result.Success, result.Message, nil
}

// GetPeerAddress возвращает адрес текущего пира для экспорта (все адреса в одной строке)
func (api *UIP2P) GetPeerAddress() (string, error) {

	if api.network.Host() == nil {
		return "", fmt.Errorf("P2P не запущен")
	}

	addr, err := address.GetPeerAddress(api.network.Host())
	if err != nil {
		return "", err
	}
	return address.FormatPeerAddress(addr.PeerID, addr.Multiaddrs), nil
}

// CopyPeerAddress копирует адрес пира в буфер обмена
func (api *UIP2P) CopyPeerAddress() (string, error) {
	return api.GetPeerAddress()
}

// AddContactByAddress добавляет контакт по адресу
func (api *UIP2P) AddContactByAddress(addrStr, username string) error {

	if api.network.Host() == nil {
		return fmt.Errorf("P2P не запущен")
	}

	// Импортируем адрес пира и добавляем в peerstore
	peerAddr, err := address.ImportPeerAddress(api.network.Host(), addrStr)
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

	// Сохраняем адрес пира в БД для автоподключения при следующем запуске
	if err := queries.AddPeerAddressWithProfile(peerID.String(), addrStr, "contact", "add_contact", username); err != nil {
		log.Printf("⚠️ Не удалось сохранить адрес пира в БД: %v", err)
	} else {
		log.Printf("✅ Адрес пира сохранён в peer_addresses: %s", peerID.String()[:8])
	}

	// Пробуем подключиться к пиру для получения профиля
	// Увеличиваем таймаут до 60 секунд для подключения через NAT
	ctx, cancel := context.WithTimeout(api.network.Ctx(), 60*time.Second)
	defer cancel()

	// Подключаемся к пиру
	log.Printf("Попытка подключения к пиру...")
	if err := address.ConnectToPeer(ctx, api.network.Host(), addrStr); err != nil {
		// Подключение не удалось, но контакт всё равно создан
		// Профиль будет запрошен позже при следующей попытке
		log.Printf("❌ Не удалось подключиться к пиру %s: %v", peerID.String(), err)
	} else {
		// Подключение успешно — запрашиваем профиль
		log.Printf("✅ Подключение успешно, запрашиваем профиль...")
		log.Printf("profileExchange = %v", api.network.ProfileExchange() != nil)

		if api.network.ProfileExchange() == nil {
			log.Printf("❌ profileExchange не инициализирован!")
		} else {
			go func() {
				log.Printf("[ГОРУТИНКА] Запуск запроса профиля...")
				// Увеличиваем таймаут для запроса профиля до 30 секунд
				profileCtx, profileCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer profileCancel()

				// Запрашиваем профиль у пира
				log.Printf("Запрос профиля у пира %s...", peerID.String())
				profileWithSig, err := api.network.ProfileExchange().RequestPeerProfile(profileCtx, peerID)
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

// ConnectToContact подключается к контакту по адресу (НЕ создаёт контакт в БД)
// Для добавления в контакты используйте AddContactByAddress
func (api *UIP2P) ConnectToContact(addrStr string) error {

	if api.network.Host() == nil {
		return fmt.Errorf("P2P не запущен")
	}

	ctx, cancel := context.WithTimeout(api.network.Ctx(), 30*time.Second)
	defer cancel()

	// Подключаемся к пиру
	if err := address.ConnectToPeer(ctx, api.network.Host(), addrStr); err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}

	// Получаем PeerID из адреса (без добавления в peerstore и создания контакта)
	peerID, err := address.ExtractPeerIDFromAddress(addrStr)
	if err != nil {
		return fmt.Errorf("ошибка извлечения PeerID: %w", err)
	}

	// Сохраняем адрес пира в БД для автоподключения при следующем запуске
	if err := queries.AddPeerAddressWithProfile(peerID, addrStr, "contact", "manual_connect", ""); err != nil {
		log.Printf("[ConnectToContact] ⚠️ Не удалось сохранить адрес пира в БД: %v", err)
	} else {
		log.Printf("[ConnectToContact] ✅ Адрес пира сохранён в peer_addresses: %s", peerID[:8])
	}

	// Профиль будет запрошен автоматически через onPeerConnected (events.go)
	// чтобы избежать race condition с дублирующими стримами

	return nil
}

// GetConnectedPeers возвращает список подключённых пиров
func (api *UIP2P) GetConnectedPeers() []*PeerInfo {

	if api.network.Host() == nil || api.network.Connections() == nil {
		return []*PeerInfo{}
	}

	var peers []*PeerInfo
	for _, peerID := range api.network.Host().Network().Peers() {
		info := api.network.Connections().GetPeerInfo(peerID)
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

	contacts, err := queries.GetAllContacts()
	if err != nil {
		return []*PeerInfo{}
	}

	var peers []*PeerInfo
	for _, contact := range contacts {
		var peerInfo *connection.PeerConnectionInfo
		if api.network.Connections() != nil && contact.PeerID != "" {
			// Пробуем распарсить PeerID
			pid, err := peer.Decode(contact.PeerID)
			if err == nil {
				peerInfo = api.network.Connections().GetPeerInfo(pid)
			}
		}

		latencyMs := int64(0)
		isConnected := false
		lastSeen := time.Time{}

		if peerInfo != nil {
			latencyMs = peerInfo.LastPingLatency.Milliseconds()
			isConnected = peerInfo.Status == connection.StatusConnected
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
func (api *UIP2P) GetBootstrapPeers() []*models.PeerAddress {

	if api.network.DiscoveryService() == nil {
		return []*models.PeerAddress{}
	}

	peers, _ := api.network.DiscoveryService().GetBootstrapPeers()
	return peers
}

// AddBootstrapPeer добавляет bootstrap пир
func (api *UIP2P) AddBootstrapPeer(multiaddr string) error {

	if api.network.DiscoveryService() == nil {
		return fmt.Errorf("сервис обнаружения не инициализирован")
	}
	return api.network.DiscoveryService().AddBootstrapPeer(multiaddr)
}

// RemoveBootstrapPeer удаляет bootstrap пир
func (api *UIP2P) RemoveBootstrapPeer(multiaddr string) error {

	if api.network.DiscoveryService() == nil {
		return fmt.Errorf("сервис обнаружения не инициализирован")
	}
	return api.network.DiscoveryService().RemoveBootstrapPeer(multiaddr)
}

// GetDiscoveredPeers возвращает список обнаруженных пиров
func (api *UIP2P) GetDiscoveredPeers() map[string]time.Time {

	if api.network.DiscoveryService() == nil {
		return make(map[string]time.Time)
	}
	return api.network.DiscoveryService().GetDiscoveredPeers()
}

// StartPeerDiscovery запускает обнаружение пиров (DHT + bootstrap)
func (api *UIP2P) StartPeerDiscovery() error {

	if api.network.DiscoveryService() == nil {
		return fmt.Errorf("сервис обнаружения не инициализирован")
	}
	return api.network.DiscoveryService().StartDiscovery()
}

// GetHelperPeers возвращает список пиров из helper режима
func (api *UIP2P) GetHelperPeers() []helper.PeerEntry {

	if api.network.Helper() == nil || api.network.HelperService() == nil {
		return []helper.PeerEntry{}
	}
	return api.network.HelperService().List()
}

// RequestProfile запрашивает профиль у пира
func (api *UIP2P) RequestProfile(peerIDStr string) error {

	if api.network.ProfileExchange() == nil {
		return fmt.Errorf("сервис обмена профилями не инициализирован")
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return err
	}

	// Проверяем, есть ли уже профиль в БД
	existingProfile, err := queries.GetRemoteProfile(peerIDStr)
	if err == nil && existingProfile != nil && existingProfile.Username != "" {
		log.Printf("[Profile] ✅ Профиль уже есть в БД для %s (username: %s), повторный запрос не требуется", peerIDStr[:8], existingProfile.Username)
		return nil
	}

	ctx, cancel := context.WithTimeout(api.network.Ctx(), 10*time.Second)
	defer cancel()

	_, err = api.network.ProfileExchange().RequestPeerProfile(ctx, peerID)
	return err
}

// RequestAllProfiles запрашивает профили у всех контактов
func (api *UIP2P) RequestAllProfiles() {

	if api.network.ProfileExchange() == nil {
		return
	}

	ctx, cancel := context.WithTimeout(api.network.Ctx(), 30*time.Second)
	defer cancel()

	api.network.ProfileExchange().RequestProfilesForAllContacts(ctx)
}

// GetConnectedPeersCount возвращает количество подключённых пиров
func (api *UIP2P) GetConnectedPeersCount() int {
	return api.network.GetConnectedPeersCount()
}

// ConnectToAll подключается ко всем известным пирам
func (api *UIP2P) ConnectToAll() {
	if api.network.Autodial() == nil {
		return
	}

	go func() {
		results := api.network.Autodial().ConnectToAll()
		for result := range results {
			peerShort := "unknown"
			if result.PeerID.String() != "" {
				peerShort = result.PeerID.String()[:min(8, len(result.PeerID.String()))]
			}
			if result.Error != nil {
				log.Printf("[P2P] ❌ Подключение к пиру %s не удалось: %v", peerShort, result.Error)
			}
		}
	}()
}

// ExchangeProfileLists запускает синхронизацию профилей со всеми подключёнными пирами
func (api *UIP2P) ExchangeProfileLists() {
	if api.network.ProfileSync() == nil {
		return
	}

	peers := api.network.Host().Network().Peers()
	for _, peerID := range peers {
		go func(pid peer.ID) {
			ctx, cancel := context.WithTimeout(api.network.Ctx(), 30*time.Second)
			defer cancel()
			_ = api.network.ProfileSync().SyncWithPeer(ctx, pid)
		}(peerID)
	}
}

// GetPeerID декодирует PeerID из строки
func (api *UIP2P) GetPeerID(peerIDStr string) (peer.ID, error) {
	return peer.Decode(peerIDStr)
}

// GetPeerInfo возвращает информацию о пире
func (api *UIP2P) GetPeerInfo(peerIDStr string) *PeerInfo {

	if api.network.Host() == nil || api.network.Connections() == nil {
		return nil
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return nil
	}

	info := api.network.Connections().GetPeerInfo(peerID)
	contact, _ := queries.GetContactByPeerID(peerIDStr)

	username := peerIDStr[:8]
	if contact != nil {
		username = contact.Username
	}

	latencyMs := int64(0)
	status := ""
	lastSeen := time.Time{}
	address := ""

	if info != nil {
		latencyMs = info.LastPingLatency.Milliseconds()
		status = string(info.Status)
		lastSeen = info.LastSeen
	}

	// Получаем адрес пира
	addrs := api.network.Host().Peerstore().Addrs(peerID)
	if len(addrs) > 0 {
		address = addrs[0].String()
	}

	return &PeerInfo{
		PeerID:      peerIDStr,
		Username:    username,
		Status:      status,
		IsConnected: api.network.Host().Network().Connectedness(peerID) == network.Connected,
		LastSeen:    lastSeen,
		LatencyMs:   latencyMs,
		Address:     address,
	}
}

// GetPeerAddresses возвращает адреса пира
func (api *UIP2P) GetPeerAddresses(peerIDStr string) []string {

	if api.network.Host() == nil {
		return []string{}
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return []string{}
	}

	addrs := api.network.Host().Peerstore().Addrs(peerID)
	result := make([]string, len(addrs))
	for i, addr := range addrs {
		result[i] = addr.String()
	}

	return result
}

// SendMessage отправляет текстовое сообщение пиру
func (api *UIP2P) SendMessage(peerID peer.ID, content string) error {
	log.Printf("[Chat] 📤 UIP2P.SendMessage: пиру %s, len=%d", peerID[:8], len(content))

	// Сначала сохраняем сообщение в БД через ChatService
	chatSvc := services.GetChatService()
	if chatSvc != nil {
		localPeerID := ""
		if api.network != nil && api.network.Host() != nil {
			localPeerID = api.network.Host().ID().String()
		}
		log.Printf("[Chat] 📝 Сохранение сообщения в БД через ChatService (fromPeerID=%s)", localPeerID[:min(10, len(localPeerID))])
		_, err := chatSvc.SendTextMessage(0, peerID.String(), localPeerID, content)
		if err != nil {
			log.Printf("[Chat] ❌ Ошибка сохранения сообщения в БД: %v", err)
			// Не прерываем отправку, если сохранение не удалось
		} else {
			log.Printf("[Chat] ✅ Сообщение сохранено в БД")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := api.network.SendTextMessage(ctx, peerID, content)
	if err != nil {
		log.Printf("[Chat] ❌ UIP2P.SendMessage ошибка: %v", err)
	} else {
		log.Printf("[Chat] ✅ UIP2P.SendMessage успешно")
	}
	return err
}

// SendElementMessage отправляет элемент пиру через P2P
func (api *UIP2P) SendElementMessage(peerID peer.ID, item *models.Item) error {
	log.Printf("[Chat] 📤 UIP2P.SendElementMessage: пиру %s, element_uuid=%s", peerID[:8], item.ElementUUID)

	// Создаём метаданные элемента
	metadata := map[string]interface{}{
		"item_id":      item.ID,
		"item_type":    string(item.Type),
		"item_title":   item.Title,
		"item_desc":    item.Description,
		"content_meta": item.ContentMeta,
		"item_hash":    item.Hash,
		"sent_at":      item.CreatedAt.Format(time.RFC3339),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("[Chat] ❌ Ошибка сериализации метаданных элемента: %v", err)
		return fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Отправляем элемент через P2P Chat Service
	err = api.network.SendMessage(ctx, peerID, item.ElementUUID, "element", string(metadataJSON))
	if err != nil {
		log.Printf("[Chat] ❌ UIP2P.SendElementMessage ошибка: %v", err)
	} else {
		log.Printf("[Chat] ✅ UIP2P.SendElementMessage успешно")
	}
	return err
}

// GetMessagesForContact получает сообщения для контакта
func (api *UIP2P) GetMessagesForContact(contactID, limit, offset int) ([]*models.ChatMessage, error) {
	return api.network.GetMessagesForContact(contactID, limit, offset)
}

// DisconnectPeer отключается от пира
func (api *UIP2P) DisconnectPeer(peerIDStr string) error {

	if api.network.Host() == nil {
		return fmt.Errorf("P2P не запущен")
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	// Закрываем все соединения с пиром
	if err := api.network.Host().Network().ClosePeer(peerID); err != nil {
		log.Printf("Предупреждение: ошибка при отключении от пира %s: %v", peerID, err)
	}
	log.Printf("Отключено от пира: %s", peerID)

	return nil
}

// ConnectToDiscoveredPeer подключается к обнаруженному пиру по PeerID
func (api *UIP2P) ConnectToDiscoveredPeer(peerIDStr string) error {

	if api.network.Host() == nil {
		return fmt.Errorf("P2P не запущен")
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	// Получаем адреса из peerstore
	addrs := api.network.Host().Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		return fmt.Errorf("адреса пира не найдены в peerstore")
	}

	// Создаём peer info
	peerInfo := peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}

	// Подключаемся
	ctx, cancel := context.WithTimeout(api.network.Ctx(), 30*time.Second)
	defer cancel()

	if err := api.network.Host().Connect(ctx, peerInfo); err != nil {
		return fmt.Errorf("ошибка подключения к пиру: %w", err)
	}

	log.Printf("Подключено к пиру: %s", peerID)
	return nil
}

// GetLocalAddresses возвращает список всех адресов для подключения
func (api *UIP2P) GetLocalAddresses() []string {

	var addresses []string

	if api.network.Host() == nil {
		log.Printf("[GetLocalAddresses] Host == nil")
		return addresses
	}

	peerID := api.network.Host().ID().String()
	configPort := api.network.Config().ListenPort

	log.Printf("[GetLocalAddresses] PeerID: %s, Конфигурированный порт: %d", peerID, configPort)
	log.Printf("[GetLocalAddresses] Всего адресов: %d", len(api.network.Host().Addrs()))

	for _, addr := range api.network.Host().Addrs() {
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

			// Пропускаем порт 0 и порты не из конфигурации
			if port <= 0 {
				continue
			}

			// Если порт не совпадает с конфигурированным - пропускаем
			if configPort > 0 && port != configPort {
				log.Printf("[GetLocalAddresses] Пропущен адрес с портом %d (ожидался %d)", port, configPort)
				continue
			}

			// Компактный формат: projectt:PeerID@ip:port
			addrFormatted := fmt.Sprintf("%s:%s@%s:%d", address.ProtocolPrefix, peerID, ip, port)
			addresses = append(addresses, addrFormatted)
			log.Printf("[GetLocalAddresses] ✅ Адрес: %s", addrFormatted)
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

// GetContacts возвращает все контакты из базы данных
func (api *UIP2P) GetContacts() ([]*models.Contact, error) {
	return queries.GetAllContacts()
}

// GetProfiles возвращает все remote профили из базы данных
func (api *UIP2P) GetProfiles() ([]*models.Profile, error) {
	return queries.GetAllRemoteProfiles()
}

// DeleteContact удаляет контакт по ID
func (api *UIP2P) DeleteContact(id int) error {
	return queries.DeleteContact(id)
}

// DeleteProfile удаляет профиль пира по PeerID
func (api *UIP2P) DeleteProfile(peerID string) error {
	return queries.DeleteRemoteProfile(peerID)
}

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SendBatch отправляет пакет элементов пиру
func (api *UIP2P) SendBatch(peerID peer.ID, elementUUIDs []string, batchType transfer.TransferType) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return api.network.SendBatch(ctx, peerID, elementUUIDs, batchType)
}

// SendFolder отправляет папку пиру
func (api *UIP2P) SendFolder(peerID peer.ID, parentUUID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return api.network.SendFolder(ctx, peerID, parentUUID)
}

// SendPinnedItems отправляет закреплённые элементы пиру
func (api *UIP2P) SendPinnedItems(peerID peer.ID) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return api.network.SendPinnedItems(ctx, peerID)
}

// SendSelection отправляет выбранные элементы пиру
func (api *UIP2P) SendSelection(peerID peer.ID, elementUUIDs []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return api.network.SendSelection(ctx, peerID, elementUUIDs)
}

// GetBatchProgress возвращает прогресс батча
func (api *UIP2P) GetBatchProgress(batchID string) *transfer.BatchProgress {
	return api.network.GetBatchProgress(batchID)
}

// RequestBatchByUUIDs запрашивает батч элементов у пира
func (api *UIP2P) RequestBatchByUUIDs(peerID peer.ID, elementUUIDs []string) ([]*models.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return api.network.RequestBatchByUUIDs(ctx, peerID, elementUUIDs)
}

// RequestBatchByUUIDsAsync запрашивает батч элементов асинхронно с коллбэками
func (api *UIP2P) RequestBatchByUUIDsAsync(peerID peer.ID, elementUUIDs []string, callbacks itemsync.BatchRequestCallbacks) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	api.network.RequestBatchByUUIDsAsync(ctx, peerID, elementUUIDs, callbacks)
}

// RequestFolder запрашивает папку у пира
func (api *UIP2P) RequestFolder(peerID peer.ID, parentUUID string) ([]*models.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return api.network.RequestFolder(ctx, peerID, parentUUID)
}

// RequestRandomItemsAsync запрашивает случайные элементы у пира асинхронно
func (api *UIP2P) RequestRandomItemsAsync(peerID peer.ID, count int, callbacks itemsync.BatchRequestCallbacks) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	api.network.RequestRandomItemsAsync(ctx, peerID, count, callbacks)
}

// GetPinnedItemUUIDs возвращает список UUID закреплённых элементов
func (api *UIP2P) GetPinnedItemUUIDs() ([]string, error) {
	return queries.GetPinnedItemUUIDs()
}

// GetItemsByParentUUID возвращает элементы папки по parent_uuid
func (api *UIP2P) GetItemsByParentUUID(parentUUID string) ([]*models.Item, error) {
	return queries.GetItemsByParentUUID(parentUUID)
}
