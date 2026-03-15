// Package p2p содержит сервисы для P2P связи на базе libp2p.
package p2p

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

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

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
	defer resp.Body.Close()

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
	PeerID    string `json:"peer_id"`
	Multiaddr string `json:"multiaddr"`
	PublicKey string `json:"public_key"`
}

// GetPeerAddress возвращает адрес текущего пира для экспорта
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

	// Формируем полный адрес с префиксом
	addr := h.Addrs()[0].String()
	fullAddr := fmt.Sprintf("%s/p2p/%s", addr, h.ID().String())
	// Добавляем префикс к адресу
	prefixedAddr := ProtocolPrefix + "://" + fullAddr

	return &PeerAddress{
		PeerID:    ProtocolPrefix + ":" + h.ID().String(),
		Multiaddr: prefixedAddr,
		PublicKey: base64.StdEncoding.EncodeToString(prefixedPubKey),
	}, nil
}

// ImportPeerAddress импортирует адрес пира и добавляет в контакты
func ImportPeerAddress(h host.Host, addrStr string) (*PeerAddress, error) {
	if h == nil {
		return nil, errors.New("хост не инициализирован")
	}

	// Удаляем префикс если есть
	addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")

	// Парсим адрес
	addr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга адреса: %w", err)
	}

	// Извлекаем PeerID
	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return nil, fmt.Errorf("ошибка извлечения PeerID: %w", err)
	}

	// Добавляем в peerstore
	h.Peerstore().AddAddr(info.ID, addr, peerstore.PermanentAddrTTL)

	// Получаем публичный ключ
	pubKey := h.Peerstore().PubKey(info.ID)
	if pubKey == nil {
		return nil, errors.New("публичный ключ не найден")
	}

	pubKeyBytes, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения публичного ключа: %w", err)
	}

	// Создаём профиль для контакта если он ещё не существует
	username := info.ID.String()[:8] // Первые 8 символов как временное имя
	if err := queries.EnsureProfileForContact(info.ID.String(), username, ""); err != nil {
		log.Printf("Предупреждение: не удалось создать профиль: %v", err)
	}

	// Создаём контакт в БД
	contact := &models.Contact{
		PeerID:    info.ID.String(),
		Multiaddr: addrStr,
		Notes:     "",
		IsBlocked: false,
	}

	if err := queries.CreateContact(contact); err != nil {
		if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, fmt.Errorf("ошибка создания контакта: %w", err)
		}
		// Контакт уже существует - игнорируем ошибку
	}

	return &PeerAddress{
		PeerID:    info.ID.String(),
		Multiaddr: addrStr,
		PublicKey: base64.StdEncoding.EncodeToString(pubKeyBytes),
	}, nil
}

// ConnectToPeer подключается к пиру по адресу
func ConnectToPeer(ctx context.Context, h host.Host, addrStr string) error {
	if h == nil {
		return errors.New("хост не инициализирован")
	}

	// Удаляем префикс если есть
	addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")

	addr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("ошибка парсинга адреса: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("ошибка извлечения информации о пире: %w", err)
	}

	if err := h.Connect(ctx, *info); err != nil {
		return fmt.Errorf("ошибка подключения к пиру %s: %w", info.ID, err)
	}

	log.Printf("Подключено к пиру: %s", info.ID.String())
	return nil
}

// ParsePeerAddressString парсит строку адреса в формате peerid@multiaddr
func ParsePeerAddressString(addrStr string) (*PeerAddress, error) {
	// Удаляем префикс если есть
	addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")

	parts := strings.SplitN(addrStr, "@", 2)
	if len(parts) != 2 {
		// Пробуем распарсить как полный multiaddr
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			return nil, errors.New("неверный формат адреса")
		}

		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return nil, errors.New("не удалось извлечь PeerID")
		}

		return &PeerAddress{
			PeerID:    info.ID.String(),
			Multiaddr: addrStr,
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
		PeerID:    pid.String(),
		Multiaddr: ma,
	}, nil
}

// FormatPeerAddress форматирует адрес для шаринга с префиксом проекта
func FormatPeerAddress(peerID, multiaddr string) string {
	return fmt.Sprintf("%s:%s@%s", ProtocolPrefix, peerID, multiaddr)
}

// addPrefixToData добавляет префикс проекта к данным
func addPrefixToData(data []byte) []byte {
	prefix := []byte(ProtocolPrefix + ":")
	result := make([]byte, len(prefix)+len(data))
	copy(result, prefix)
	copy(result[len(prefix):], data)
	return result
}
