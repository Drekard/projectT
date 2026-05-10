package address

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// PortCheckResult результат проверки порта
type PortCheckResult struct {
	Port         int           `json:"port"`
	PublicIP     string        `json:"public_ip"`
	Accessible   bool          `json:"accessible"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
	Message      string        `json:"message,omitempty"`
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

	publicIP, err := GetPublicIP()
	if err != nil {
		result.Error = fmt.Sprintf("не удалось определить внешний IP: %v", err)
		return result, nil
	}
	result.PublicIP = publicIP

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

// FirewallResult результат открытия брандмауэра
type FirewallResult struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Output  string            `json:"output,omitempty"`
	Command *FirewallRuleInfo `json:"command,omitempty"`
}

// OpenFirewallRule пытается открыть порт в брандмауэре автоматически
func OpenFirewallRule(port int, ruleName string) (*FirewallResult, error) {
	if runtime.GOOS != "windows" {
		return &FirewallResult{
			Success: false,
			Message: "Автоматическое открытие поддерживается только в Windows",
		}, nil
	}

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

	cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		fmt.Sprintf(`name=%s`, ruleName),
		"dir=in",
		"action=allow",
		"protocol=TCP",
		fmt.Sprintf("localport=%d", port))

	output, err := cmd.CombinedOutput()
	if err != nil {
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

func checkAdminRights() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, nil
	}

	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil, nil
}
