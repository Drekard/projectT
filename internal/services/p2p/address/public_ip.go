package address

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
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

	publicIP, err := GetPublicIP()
	if err != nil {
		publicIP = "<не удалось определить>"
	}

	peerID := h.ID().String()

	fullAddr := fmt.Sprintf("%s://%s:%d/p2p/%s", ProtocolPrefix, publicIP, port, peerID)

	var localAddrs []string
	for _, addr := range h.Addrs() {
		addrStr := addr.String()
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
