// Package address предоставляет утилиты для работы с адресами P2P
package address

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"

	p2p "projectT/internal/services/p2p"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ProtocolPrefix префикс для всех идентификаторов проекта
const ProtocolPrefix = p2p.ProtocolPrefix

// PublicAddressInfo информация о публичном адресе
type PublicAddressInfo struct {
	PublicIP    string   `json:"public_ip"`
	PeerID      string   `json:"peer_id"`
	FullAddress string   `json:"full_address"`
	LocalAddrs  []string `json:"local_addresses"`
	Protocol    string   `json:"protocol"`
	Port        int      `json:"port"`
}

// GetPublicIP получает внешний IP адрес через сторонний сервис
func GetPublicIP() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org?format=json", nil)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("ошибка декодирования: %w", err)
	}

	ip, ok := result["ip"]
	if !ok {
		return "", fmt.Errorf("не удалось получить IP из ответа")
	}

	return ip, nil
}

// GeneratePublicAddress генерирует полный адрес для подключения из другой сети
func GeneratePublicAddress(h host.Host, port int) (*PublicAddressInfo, error) {
	if h == nil {
		return nil, fmt.Errorf("хост не инициализирован")
	}

	// Получаем внешний IP
	publicIP, err := GetPublicIP()
	if err != nil {
		// Если не удалось получить, используем заглушку
		publicIP = "<не удалось определить>"
	}

	// Получаем PeerID
	peerID := h.ID().String()

	// Формируем полный адрес
	fullAddr := fmt.Sprintf("%s://%s:%d/p2p/%s", ProtocolPrefix, publicIP, port, peerID)

	// Получаем локальные адреса
	var localAddrs []string
	for _, addr := range h.Addrs() {
		addrStr := addr.String()
		// Пропускаем localhost
		if strings.Contains(addrStr, "127.0.0.1") {
			continue
		}
		localAddrs = append(localAddrs, fmt.Sprintf("%s://%s/p2p/%s", ProtocolPrefix, addrStr, peerID))
	}

	return &PublicAddressInfo{
		PublicIP:    publicIP,
		PeerID:      peerID,
		FullAddress: fullAddr,
		LocalAddrs:  localAddrs,
		Protocol:    ProtocolPrefix,
		Port:        port,
	}, nil
}

// CheckPortAccessibility проверяет доступность порта через подключение к самому себе
func CheckPortAccessibility(h host.Host, port int) (*PortCheckResult, error) {
	if h == nil {
		return nil, fmt.Errorf("хост не инициализирован")
	}

	result := &PortCheckResult{
		Port:       port,
		Accessible: false,
	}

	// Получаем внешний IP
	publicIP, err := GetPublicIP()
	if err != nil {
		result.Error = fmt.Sprintf("не удалось определить внешний IP: %v", err)
		return result, nil
	}
	result.PublicIP = publicIP

	// Формируем адрес для проверки
	addrStr := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", publicIP, port, h.ID().String())
	ma, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		result.Error = fmt.Sprintf("ошибка парсинга адреса: %v", err)
		return result, nil
	}

	info, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		result.Error = fmt.Sprintf("ошибка извлечения PeerID: %v", err)
		return result, nil
	}

	// Пытаемся подключиться к самим себе через публичный IP
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startTime := time.Now()
	err = h.Connect(ctx, *info)
	elapsed := time.Since(startTime)

	if err != nil {
		result.Accessible = false
		result.Error = fmt.Sprintf("порт недоступен: %v", err)
		result.ResponseTime = elapsed
		return result, nil
	}

	result.Accessible = true
	result.ResponseTime = elapsed
	result.Message = "Порт доступен для внешних подключений"

	return result, nil
}

// PortCheckResult результат проверки порта
type PortCheckResult struct {
	Port         int           `json:"port"`
	PublicIP     string        `json:"public_ip"`
	Accessible   bool          `json:"accessible"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
	Message      string        `json:"message,omitempty"`
}

// FirewallRuleInfo информация о правиле брандмауэра
type FirewallRuleInfo struct {
	Port       int    `json:"port"`
	Protocol   string `json:"protocol"`
	RuleName   string `json:"rule_name"`
	Platform   string `json:"platform"`
	PowerShell string `json:"powershell_command"`
	CMD        string `json:"cmd_command"`
}

// GenerateFirewallRule генерирует команду для открытия порта в брандмауэре
func GenerateFirewallRule(port int, ruleName string) *FirewallRuleInfo {
	if ruleName == "" {
		ruleName = "ProjectT P2P"
	}

	return &FirewallRuleInfo{
		Port:       port,
		Protocol:   "TCP",
		RuleName:   ruleName,
		Platform:   runtime.GOOS,
		PowerShell: fmt.Sprintf(`New-NetFirewallRule -DisplayName "%s" -Direction Inbound -LocalPort %d -Protocol TCP -Action Allow`, ruleName, port),
		CMD:        fmt.Sprintf(`netsh advfirewall firewall add rule name="%s" dir=in action=allow protocol=TCP localport=%d`, ruleName, port),
	}
}

// OpenFirewallRule пытается открыть порт в брандмауэре автоматически
func OpenFirewallRule(port int, ruleName string) (*FirewallResult, error) {
	if runtime.GOOS != "windows" {
		return &FirewallResult{
			Success: false,
			Message: "Автоматическое открытие поддерживается только в Windows",
		}, nil
	}

	// Проверяем, запущены ли мы от имени администратора
	isAdmin, err := checkAdminRights()
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки прав: %w", err)
	}

	if !isAdmin {
		return &FirewallResult{
			Success: false,
			Message: "Требуются права администратора. Запустите от имени администратора или выполните команду вручную.",
			Command: GenerateFirewallRule(port, ruleName),
		}, nil
	}

	// Открываем порт через netsh
	cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		fmt.Sprintf(`name=%s`, ruleName),
		"dir=in",
		"action=allow",
		"protocol=TCP",
		fmt.Sprintf("localport=%d", port))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Если правило уже существует - это нормально
		if strings.Contains(string(output), "The object already exists") ||
			strings.Contains(string(output), "Объект уже существует") {
			return &FirewallResult{
				Success: true,
				Message: "Правило уже существует",
			}, nil
		}

		return &FirewallResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка: %v", err),
			Output:  string(output),
		}, nil
	}

	return &FirewallResult{
		Success: true,
		Message: fmt.Sprintf("Порт %d успешно открыт в брандмауэре", port),
	}, nil
}

// FirewallResult результат открытия брандмауэра
type FirewallResult struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Output  string            `json:"output,omitempty"`
	Command *FirewallRuleInfo `json:"command,omitempty"`
}

// checkAdminRights проверяет, запущены ли мы от имени администратора (Windows)
func checkAdminRights() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, nil
	}

	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil, nil
}

// GetNATStatus возвращает информацию о статусе NAT
func GetNATStatus(h host.Host) *NATStatusInfo {
	if h == nil {
		return &NATStatusInfo{
			UPnPEnabled:    false,
			NATPMPEndabled: false,
			Message:        "Хост не инициализирован",
		}
	}

	// Получаем наблюдаемые адреса (если UPnP работает, они будут содержать публичный IP)
	// В текущей версии libp2p это делается через AutoNAT

	// Проверяем, есть ли адреса с публичным IP
	hasPublicAddr := false
	for _, addr := range h.Addrs() {
		if isPublicAddress(addr) {
			hasPublicAddr = true
			break
		}
	}

	return &NATStatusInfo{
		UPnPEnabled:    hasPublicAddr,
		NATPMPEndabled: hasPublicAddr,
		HasPublicAddr:  hasPublicAddr,
		Message:        getNATMessage(hasPublicAddr),
	}
}

// NATStatusInfo информация о статусе NAT
type NATStatusInfo struct {
	UPnPEnabled    bool   `json:"upnp_enabled"`
	NATPMPEndabled bool   `json:"natpmp_enabled"`
	HasPublicAddr  bool   `json:"has_public_addr"`
	Message        string `json:"message"`
}

// isPublicAddress проверяет, является ли адрес публичным
func isPublicAddress(addr multiaddr.Multiaddr) bool {
	// Извлекаем IP из адреса
	ipStr, err := addr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		ipStr, err = addr.ValueForProtocol(multiaddr.P_IP6)
		if err != nil {
			return false
		}
	}

	// Простые проверки на частные диапазоны
	return !isPrivateIP(ipStr)
}

// isPrivateIP проверяет, является ли IP частным
func isPrivateIP(ip string) bool {
	// IPv4 private ranges
	if strings.HasPrefix(ip, "192.168.") {
		return true
	}
	if strings.HasPrefix(ip, "10.") {
		return true
	}
	if strings.HasPrefix(ip, "172.16.") || strings.HasPrefix(ip, "172.17.") ||
		strings.HasPrefix(ip, "172.18.") || strings.HasPrefix(ip, "172.19.") ||
		strings.HasPrefix(ip, "172.20.") || strings.HasPrefix(ip, "172.21.") ||
		strings.HasPrefix(ip, "172.22.") || strings.HasPrefix(ip, "172.23.") ||
		strings.HasPrefix(ip, "172.24.") || strings.HasPrefix(ip, "172.25.") ||
		strings.HasPrefix(ip, "172.26.") || strings.HasPrefix(ip, "172.27.") ||
		strings.HasPrefix(ip, "172.28.") || strings.HasPrefix(ip, "172.29.") ||
		strings.HasPrefix(ip, "172.30.") || strings.HasPrefix(ip, "172.31.") {
		return true
	}
	if ip == "127.0.0.1" || strings.HasPrefix(ip, "127.") {
		return true
	}
	return false
}

// getNATMessage возвращает сообщение о статусе NAT
func getNATMessage(hasPublicAddr bool) string {
	if hasPublicAddr {
		return "UPnP/NAT-PMP работает. Обнаружен публичный адрес."
	}
	return "UPnP/NAT-PMP может не работать. Публичный адрес не обнаружен. Используйте relay или STUN."
}

// PeerAddress структура для экспорта адреса пира
type PeerAddress struct {
	PeerID      string   `json:"peer_id"`
	Multiaddrs  []string `json:"multiaddrs"` // Все адреса пира
	PublicKey   string   `json:"public_key"`
	AddressType string   `json:"address_type"` // Тип лучшего адреса: "public", "lan", "localhost"
}

// GetPeerAddress возвращает ВСЕ адреса текущего пира для экспорта
func GetPeerAddress(h host.Host) (*PeerAddress, error) {
	if h == nil {
		return nil, errors.New("хост не инициализирован")
	}

	// Получаем приватный ключ для извлечения публичного
	privKey := h.Peerstore().PrivKey(h.ID())
	if privKey == nil {
		return nil, errors.New("не удалось получить приватный ключ")
	}

	pubKeyBytes, err := privKey.GetPublic().Raw()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения публичного ключа: %w", err)
	}

	// Добавляем префикс к публичному ключу
	prefixedPubKey := addPrefixToData(pubKeyBytes)

	peerID := h.ID().String()
	addrs := h.Addrs()
	log.Printf("[GetPeerAddress] Всего адресов: %d", len(addrs))

	var multiaddrs []string
	var bestAddrType string
	bestPriority := 0

	for i, addr := range addrs {
		addrStr := addr.String()
		addrType := "unknown"
		priority := 0

		ipStr, ipErr := addr.ValueForProtocol(multiaddr.P_IP4)
		if ipErr != nil {
			ipStr, ipErr = addr.ValueForProtocol(multiaddr.P_IP6)
		}

		if ipErr == nil {
			if isPrivateIP(ipStr) {
				if ipStr == "127.0.0.1" || ipStr == "::1" {
					addrType = "localhost"
					priority = 1
				} else {
					addrType = "LAN"
					priority = 2
				}
			} else {
				addrType = "public"
				priority = 3
			}
		}

		// Формируем полный multiaddr с PeerID
		fullAddr := fmt.Sprintf("%s/p2p/%s", addrStr, peerID)
		multiaddrs = append(multiaddrs, fullAddr)
		bestAddrType = addrType

		log.Printf("[GetPeerAddress]   [%d] %s (type=%s, priority=%d)", i, fullAddr, addrType, priority)

		if priority > bestPriority {
			bestPriority = priority
			bestAddrType = addrType
		}
	}

	if len(multiaddrs) == 0 {
		return nil, errors.New("нет доступных адресов")
	}

	log.Printf("[GetPeerAddress] ✅ Адресов для экспорта: %d, лучший тип: %s", len(multiaddrs), bestAddrType)

	return &PeerAddress{
		PeerID:      peerID,
		Multiaddrs:  multiaddrs,
		PublicKey:   base64.StdEncoding.EncodeToString(prefixedPubKey),
		AddressType: bestAddrType,
	}, nil
}

// ImportPeerAddress импортирует адрес пира и добавляет в контакты
// Поддерживает: новый компактный формат, старый полный формат, legacy формат
func ImportPeerAddress(h host.Host, addrStr string) (*PeerAddress, error) {
	if h == nil {
		return nil, errors.New("хост не инициализирован")
	}

	fmt.Println("=== ImportPeerAddress ===")
	fmt.Printf("[DEBUG] Исходная строка адреса: %q\n", addrStr)

	originalAddrStr := addrStr

	// === Удаляем префиксы ===
	for strings.HasPrefix(addrStr, ProtocolPrefix+"://") || strings.HasPrefix(addrStr, ProtocolPrefix+":") {
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+":")
	}

	fmt.Printf("[DEBUG] После удаления префиксов: %q\n", addrStr)

	// === Пробуем новый компактный формат: PeerID@ip1:port1;ip2:port2 ===
	targetPeerID, allAddrs, err := parsePeerAddress(addrStr)
	if err != nil {
		// === Пробуем legacy формат: peerid@/ip4/.../tcp/.../p2p/peerid ===
		if strings.Contains(originalAddrStr, "@") {
			fmt.Println("[DEBUG] Пробуем legacy формат...")
			parts := strings.SplitN(originalAddrStr, "@", 2)
			if len(parts) == 2 {
				maStr := parts[1]
				for strings.HasPrefix(maStr, ProtocolPrefix+"://") || strings.HasPrefix(maStr, ProtocolPrefix+":") {
					maStr = strings.TrimPrefix(maStr, ProtocolPrefix+"://")
					maStr = strings.TrimPrefix(maStr, ProtocolPrefix+":")
				}

				maParts := strings.Split(maStr, ";")
				for _, maPart := range maParts {
					maPart = strings.TrimSpace(maPart)
					if maPart == "" {
						continue
					}
					addr, maErr := multiaddr.NewMultiaddr(maPart)
					if maErr != nil {
						continue
					}
					info, infoErr := peer.AddrInfoFromP2pAddr(addr)
					if infoErr != nil {
						continue
					}
					if targetPeerID == "" {
						targetPeerID = info.ID
					}
					allAddrs = append(allAddrs, addr)
				}
			}
		}

		// === Пробуем чистый multiaddr ===
		if targetPeerID == "" {
			addr, maErr := multiaddr.NewMultiaddr(addrStr)
			if maErr == nil {
				info, infoErr := peer.AddrInfoFromP2pAddr(addr)
				if infoErr == nil {
					targetPeerID = info.ID
					allAddrs = append(allAddrs, addr)
				}
			}
		}
	}

	if targetPeerID == "" {
		return nil, fmt.Errorf("не удалось распознать формат адреса")
	}

	if len(allAddrs) == 0 {
		return nil, fmt.Errorf("нет валидных адресов")
	}

	// Добавляем все адреса в peerstore
	for _, addr := range allAddrs {
		h.Peerstore().AddAddr(targetPeerID, addr, peerstore.PermanentAddrTTL)
	}

	pubKey := h.Peerstore().PubKey(targetPeerID)
	if pubKey == nil {
		return nil, errors.New("публичный ключ не найден")
	}

	pubKeyBytes, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения публичного ключа: %w", err)
	}

	username := targetPeerID.String()[:8]
	if err := queries.EnsureProfileForContact(targetPeerID.String(), username, ""); err != nil {
		log.Printf("Предупреждение: не удалось создать профиль: %v", err)
	}

	// Сохраняем оригинальную строку адреса в БД
	contact := &models.Contact{
		PeerID:    targetPeerID.String(),
		Multiaddr: originalAddrStr,
		Notes:     "",
		IsBlocked: false,
	}

	if err := queries.CreateContact(contact); err != nil {
		if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, fmt.Errorf("ошибка создания контакта: %w", err)
		}
	}

	bestType := "unknown"
	for _, addr := range allAddrs {
		ipStr, ipErr := addr.ValueForProtocol(multiaddr.P_IP4)
		if ipErr != nil {
			ipStr, ipErr = addr.ValueForProtocol(multiaddr.P_IP6)
		}
		if ipErr == nil {
			if !isPrivateIP(ipStr) {
				bestType = "public"
				break
			} else if ipStr != "127.0.0.1" && ipStr != "::1" {
				bestType = "LAN"
			}
		}
	}

	var multiaddrStrings []string
	for _, addr := range allAddrs {
		multiaddrStrings = append(multiaddrStrings, addr.String())
	}

	return &PeerAddress{
		PeerID:      targetPeerID.String(),
		Multiaddrs:  multiaddrStrings,
		PublicKey:   base64.StdEncoding.EncodeToString(pubKeyBytes),
		AddressType: bestType,
	}, nil
}

// ExtractPeerIDFromAddress извлекает PeerID из адреса
// Поддерживает: новый компактный формат, legacy формат, чистый multiaddr
func ExtractPeerIDFromAddress(addrStr string) (string, error) {
	originalAddrStr := addrStr

	// === Удаляем префиксы ===
	for strings.HasPrefix(addrStr, ProtocolPrefix+"://") || strings.HasPrefix(addrStr, ProtocolPrefix+":") {
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+":")
	}

	// === Пробуем новый компактный формат: PeerID@ip1:port1;ip2:port2 ===
	targetPeerID, _, err := parsePeerAddress(addrStr)
	if err == nil {
		return targetPeerID.String(), nil
	}

	// === Пробуем legacy формат: peerid@/ip4/.../tcp/.../p2p/peerid ===
	if strings.Contains(originalAddrStr, "@") {
		parts := strings.SplitN(originalAddrStr, "@", 2)
		if len(parts) == 2 {
			// Если левая часть — валидный PeerID, возвращаем её
			pid, pidErr := peer.Decode(parts[0])
			if pidErr == nil {
				return pid.String(), nil
			}

			// Иначе пробуем распарсить правую часть как multiaddr
			maStr := parts[1]
			for strings.HasPrefix(maStr, ProtocolPrefix+"://") || strings.HasPrefix(maStr, ProtocolPrefix+":") {
				maStr = strings.TrimPrefix(maStr, ProtocolPrefix+"://")
				maStr = strings.TrimPrefix(maStr, ProtocolPrefix+":")
			}

			maParts := strings.Split(maStr, ";")
			for _, maPart := range maParts {
				maPart = strings.TrimSpace(maPart)
				if maPart == "" {
					continue
				}
				addr, maErr := multiaddr.NewMultiaddr(maPart)
				if maErr != nil {
					continue
				}
				info, infoErr := peer.AddrInfoFromP2pAddr(addr)
				if infoErr == nil {
					return info.ID.String(), nil
				}
			}
		}
	}

	// === Пробуем чистый multiaddr ===
	addr, maErr := multiaddr.NewMultiaddr(addrStr)
	if maErr == nil {
		info, infoErr := peer.AddrInfoFromP2pAddr(addr)
		if infoErr == nil {
			return info.ID.String(), nil
		}
	}

	return "", fmt.Errorf("не удалось извлечь PeerID из адреса")
}

// ConnectToPeer подключается к пиру по адресу
// Поддерживаемые форматы:
//   - Новый компактный: projectt:PeerID@ip1:port1;ip2:port2
//   - Старый полный: projectt:/ip4/.../tcp/.../p2p/PeerID;/ip4/.../tcp/.../p2p/PeerID
//   - Legacy: projectt:peerid@/ip4/.../tcp/.../p2p/peerid
func ConnectToPeer(ctx context.Context, h host.Host, addrStr string) error {
	if h == nil {
		return errors.New("хост не инициализирован")
	}

	log.Printf("[ConnectToPeer] ========================================")
	log.Printf("[ConnectToPeer] Попытка подключения к: %s", addrStr)
	log.Printf("[ConnectToPeer] Наш PeerID: %s", h.ID().String())
	log.Printf("[ConnectToPeer] Наши адреса:")
	for _, addr := range h.Addrs() {
		log.Printf("[ConnectToPeer]   - %s/p2p/%s", addr.String(), h.ID().String())
	}

	fmt.Println("=== ConnectToPeer ===")
	fmt.Printf("[DEBUG] Исходная строка адреса: %q\n", addrStr)

	// Сохраняем оригинальную строку
	originalAddrStr := addrStr

	// === Удаляем префиксы ===
	for strings.HasPrefix(addrStr, ProtocolPrefix+"://") || strings.HasPrefix(addrStr, ProtocolPrefix+":") {
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+":")
	}

	fmt.Printf("[DEBUG] После удаления префиксов: %q\n", addrStr)

	// === Пробуем распарсить новый компактный формат: PeerID@ip1:port1;ip2:port2 ===
	targetPeerID, allAddrs, err := parsePeerAddress(addrStr)
	if err != nil {
		// === Пробуем старый формат: peerid@/ip4/.../tcp/.../p2p/peerid ===
		if strings.Contains(originalAddrStr, "@") {
			fmt.Println("[DEBUG] Пробуем legacy формат peerid@multiaddr...")
			parts := strings.SplitN(originalAddrStr, "@", 2)
			if len(parts) == 2 {
				maStr := parts[1]
				for strings.HasPrefix(maStr, ProtocolPrefix+"://") || strings.HasPrefix(maStr, ProtocolPrefix+":") {
					maStr = strings.TrimPrefix(maStr, ProtocolPrefix+"://")
					maStr = strings.TrimPrefix(maStr, ProtocolPrefix+":")
				}

				// Поддержка нескольких multiaddr через ;
				maParts := strings.Split(maStr, ";")
				for _, maPart := range maParts {
					maPart = strings.TrimSpace(maPart)
					if maPart == "" {
						continue
					}
					addr, maErr := multiaddr.NewMultiaddr(maPart)
					if maErr != nil {
						continue
					}
					info, infoErr := peer.AddrInfoFromP2pAddr(addr)
					if infoErr != nil {
						continue
					}
					if targetPeerID == "" {
						targetPeerID = info.ID
					}
					allAddrs = append(allAddrs, addr)
				}
			}
		}

		// === Пробуем как чистый multiaddr ===
		if targetPeerID == "" {
			addr, maErr := multiaddr.NewMultiaddr(addrStr)
			if maErr == nil {
				info, infoErr := peer.AddrInfoFromP2pAddr(addr)
				if infoErr == nil {
					targetPeerID = info.ID
					allAddrs = append(allAddrs, addr)
				}
			}
		}
	}

	if targetPeerID == "" {
		return fmt.Errorf("не удалось распознать формат адреса")
	}

	if len(allAddrs) == 0 {
		return fmt.Errorf("нет валидных адресов для подключения")
	}

	info := peer.AddrInfo{
		ID:    targetPeerID,
		Addrs: allAddrs,
	}

	log.Printf("[ConnectToPeer] Целевой пир: %s", info.ID.String())
	log.Printf("[ConnectToPeer] Целевые адреса (%d шт.):", len(info.Addrs))
	for _, a := range info.Addrs {
		addrType := "unknown"
		ipStr, ipErr := a.ValueForProtocol(multiaddr.P_IP4)
		if ipErr != nil {
			ipStr, ipErr = a.ValueForProtocol(multiaddr.P_IP6)
		}
		if ipErr == nil {
			if isPrivateIP(ipStr) {
				if ipStr == "127.0.0.1" || ipStr == "::1" {
					addrType = "localhost"
				} else {
					addrType = "LAN"
				}
			} else {
				addrType = "public"
			}
		}
		log.Printf("[ConnectToPeer]   - %s [%s]", a.String(), addrType)
	}

	// Проверяем localhost
	for _, a := range info.Addrs {
		ipStr, ipErr := a.ValueForProtocol(multiaddr.P_IP4)
		if ipErr != nil {
			ipStr, ipErr = a.ValueForProtocol(multiaddr.P_IP6)
		}
		if ipErr == nil && (ipStr == "127.0.0.1" || ipStr == "::1") {
			log.Printf("[ConnectToPeer] ⚠️ Целевой пир использует localhost (%s)!", ipStr)
			log.Printf("[ConnectToPeer] ⚠️ Работает ТОЛЬКО если оба пира на одной машине")
			break
		}
	}

	// Проверяем существующие соединения
	existingConns := h.Network().ConnsToPeer(info.ID)
	if len(existingConns) > 0 {
		log.Printf("[ConnectToPeer] ⚠️ Уже есть %d соединений с %s", len(existingConns), info.ID.String()[:8])
	}

	log.Printf("[ConnectToPeer] Начинаем подключение...")
	startTime := time.Now()

	h.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.TempAddrTTL)
	log.Printf("[ConnectToPeer] Адреса добавлены в peerstore: %d шт.", len(info.Addrs))
	log.Printf("[ConnectToPeer]   - Подключённые пиры: %d", len(h.Network().Peers()))
	log.Printf("[ConnectToPeer]   - Слушаем на: %d адресах", len(h.Addrs()))

	if err := h.Connect(ctx, info); err != nil {
		elapsed := time.Since(startTime)
		log.Printf("[ConnectToPeer] ❌ ОШИБКА за %v: %v", elapsed, err)

		deadline, hasDeadline := ctx.Deadline()
		if hasDeadline {
			log.Printf("[ConnectToPeer]   - deadline=%v", deadline)
		}
		log.Printf("[ConnectToPeer]   - PeerID: %s", info.ID.String())
		log.Printf("[ConnectToPeer]   - Адресов: %d", len(info.Addrs))

		errStr := err.Error()
		if strings.Contains(errStr, "actively refused") || strings.Contains(errStr, "connection refused") {
			log.Printf("[ConnectToPeer] 💡 Соединение отклонено — порт закрыт или пир не запущен")
		} else if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
			log.Printf("[ConnectToPeer] 💡 Таймаут — пир за NAT, пробросьте порт или включите Relay")
			log.Printf("[ConnectToPeer] 💡 РЕШЕНИЕ: Включите Relay в настройках P2P")
		} else if strings.Contains(errStr, "no route to host") {
			log.Printf("[ConnectToPeer] 💡 Нет маршрута — неверный IP или сетевая проблема")
		}
		log.Printf("[ConnectToPeer] ========================================")
		return fmt.Errorf("ошибка подключения к пиру %s: %w", info.ID, err)
	}

	elapsed := time.Since(startTime)
	log.Printf("[ConnectToPeer] ✅ Подключено к %s за %v", info.ID.String(), elapsed)

	conns := h.Network().ConnsToPeer(info.ID)
	log.Printf("[ConnectToPeer] Соединений: %d", len(conns))
	for i, conn := range conns {
		remoteAddr := conn.RemoteMultiaddr()
		localAddr := conn.LocalMultiaddr()
		connType := "DIRECT"
		if strings.Contains(remoteAddr.String(), "/p2p-circuit") {
			connType = "RELAYED"
		}
		log.Printf("[ConnectToPeer]   #%d: %s, local=%s, remote=%s", i+1, connType, localAddr.String(), remoteAddr.String())
	}
	log.Printf("[ConnectToPeer] ========================================")
	return nil
}

// parsePeerAddress парсит компактный формат: PeerID@ip1:port1;ip2:port2
func parsePeerAddress(addrStr string) (peer.ID, []multiaddr.Multiaddr, error) {
	if !strings.Contains(addrStr, "@") {
		return "", nil, fmt.Errorf("нет разделителя @")
	}

	parts := strings.SplitN(addrStr, "@", 2)
	peerIDStr := parts[0]
	endpointsStr := parts[1]

	// Валидируем PeerID
	targetPeerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return "", nil, fmt.Errorf("неверный PeerID: %w", err)
	}

	// Разделяем endpoint'ы
	endpoints := strings.Split(endpointsStr, ";")
	var addrs []multiaddr.Multiaddr

	for _, ep := range endpoints {
		ep = strings.TrimSpace(ep)
		if ep == "" {
			continue
		}

		ma, err := endpointToMultiaddr(ep, peerIDStr)
		if err != nil {
			fmt.Printf("[DEBUG] Ошибка конвертации endpoint %q: %v\n", ep, err)
			continue
		}
		addrs = append(addrs, ma)
	}

	if len(addrs) == 0 {
		return "", nil, fmt.Errorf("нет валидных endpoint'ов")
	}

	return targetPeerID, addrs, nil
}

// endpointToMultiaddr конвертирует "ip:port" или "[ipv6]:port" в multiaddr с PeerID
func endpointToMultiaddr(endpoint, peerIDStr string) (multiaddr.Multiaddr, error) {
	var ip, port string

	// IPv6: [::1]:8079
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
		// IPv4: 192.168.0.1:8079
		lastColon := strings.LastIndex(endpoint, ":")
		if lastColon == -1 {
			return nil, fmt.Errorf("нет порта: %s", endpoint)
		}
		ip = endpoint[:lastColon]
		port = endpoint[lastColon+1:]
	}

	// Определяем протокол
	protocol := "ip4"
	if strings.Contains(ip, ":") {
		protocol = "ip6"
	}

	maStr := fmt.Sprintf("/%s/%s/tcp/%s/p2p/%s", protocol, ip, port, peerIDStr)
	return multiaddr.NewMultiaddr(maStr)
}

// ParsePeerAddressString парсит строку адреса в формате peerid@multiaddr
func ParsePeerAddressString(addrStr string) (*PeerAddress, error) {
	// Сохраняем оригинальную строку для отладки
	originalAddrStr := addrStr

	// Удаляем префикс проекта если есть (поддерживаем оба формата)
	// Формат 1: projectt://peerid@multiaddr
	// Формат 2: projectt:peerid@multiaddr
	addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")
	addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+":")

	parts := strings.SplitN(addrStr, "@", 2)
	if len(parts) != 2 {
		// Пробуем распарсить как полный multiaddr из оригинальной строки
		addr, err := multiaddr.NewMultiaddr(originalAddrStr)
		if err != nil {
			return nil, errors.New("неверный формат адреса")
		}

		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return nil, errors.New("не удалось извлечь PeerID")
		}

		return &PeerAddress{
			PeerID:     info.ID.String(),
			Multiaddrs: []string{originalAddrStr},
		}, nil
	}

	peerID := parts[0]
	// Удаляем префикс из PeerID если есть
	peerID = strings.TrimPrefix(peerID, ProtocolPrefix+":")

	ma := parts[1]

	// Валидируем PeerID
	pid, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("неверный PeerID: %w", err)
	}

	// Валидируем multiaddr
	_, err = multiaddr.NewMultiaddr(ma)
	if err != nil {
		return nil, fmt.Errorf("неверный multiaddr: %w", err)
	}

	return &PeerAddress{
		PeerID:     pid.String(),
		Multiaddrs: []string{ma},
	}, nil
}

// FormatPeerAddress форматирует все адреса пира в одну компактную строку
// Формат: projectt:PeerID@ip1:port1;ip2:port2;ip3:port3
// PeerID указывается один раз, IP:port разделяются точкой с запятой
func FormatPeerAddress(peerID string, multiaddrs []string) string {
	var endpoints []string
	for _, ma := range multiaddrs {
		addr, err := multiaddr.NewMultiaddr(ma)
		if err != nil {
			continue
		}

		ip, ipErr := addr.ValueForProtocol(multiaddr.P_IP4)
		protocol := "ip4"
		if ipErr != nil {
			ip, ipErr = addr.ValueForProtocol(multiaddr.P_IP6)
			protocol = "ip6"
			if ipErr != nil {
				continue
			}
		}

		portStr, portErr := addr.ValueForProtocol(multiaddr.P_TCP)
		if portErr != nil {
			continue
		}

		var endpoint string
		if protocol == "ip6" {
			endpoint = fmt.Sprintf("[%s]:%s", ip, portStr)
		} else {
			endpoint = fmt.Sprintf("%s:%s", ip, portStr)
		}
		endpoints = append(endpoints, endpoint)
	}

	combined := strings.Join(endpoints, ";")
	return fmt.Sprintf("%s:%s@%s", ProtocolPrefix, peerID, combined)
}

// FormatSinglePeerAddress форматирует один адрес (для обратной совместимости)
func FormatSinglePeerAddress(peerID, multiaddr string) string {
	return fmt.Sprintf("%s:%s", ProtocolPrefix, multiaddr)
}

// addPrefixToData добавляет префикс проекта к данным
func addPrefixToData(data []byte) []byte {
	prefix := []byte(ProtocolPrefix + ":")
	result := make([]byte, len(prefix)+len(data))
	copy(result, prefix)
	copy(result[len(prefix):], data)
	return result
}
