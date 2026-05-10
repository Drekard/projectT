package address

import (
	"strings"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
)

// NATStatusInfo информация о статусе NAT
type NATStatusInfo struct {
	UPnPEnabled    bool   `json:"upnp_enabled"`
	NATPMPEndabled bool   `json:"natpmp_enabled"`
	HasPublicAddr  bool   `json:"has_public_addr"`
	Message        string `json:"message"`
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

func isPublicAddress(addr multiaddr.Multiaddr) bool {
	ipStr, err := addr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		ipStr, err = addr.ValueForProtocol(multiaddr.P_IP6)
		if err != nil {
			return false
		}
	}

	return !isPrivateIP(ipStr)
}

func isPrivateIP(ip string) bool {
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

func getNATMessage(hasPublicAddr bool) string {
	if hasPublicAddr {
		return "UPnP/NAT-PMP работает. Обнаружен публичный адрес."
	}
	return "UPnP/NAT-PMP может не работать. Публичный адрес не обнаружен. Используйте relay или STUN."
}
