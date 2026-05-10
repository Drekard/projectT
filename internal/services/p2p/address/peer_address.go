package address

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// PeerAddress структура для экспорта адреса пира
type PeerAddress struct {
	PeerID      string   `json:"peer_id"`
	Multiaddrs  []string `json:"multiaddrs"`
	PublicKey   string   `json:"public_key"`
	AddressType string   `json:"address_type"`
}

// GetPeerAddress возвращает ВСЕ адреса текущего пира для экспорта
func GetPeerAddress(h host.Host) (*PeerAddress, error) {
	if h == nil {
		return nil, errors.New("хост не инициализирован")
	}

	privKey := h.Peerstore().PrivKey(h.ID())
	if privKey == nil {
		return nil, errors.New("не удалось получить приватный ключ")
	}

	pubKeyBytes, err := privKey.GetPublic().Raw()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения публичного ключа: %w", err)
	}

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
func ImportPeerAddress(h host.Host, addrStr string) (*PeerAddress, error) {
	if h == nil {
		return nil, errors.New("хост не инициализирован")
	}

	fmt.Println("=== ImportPeerAddress ===")
	fmt.Printf("[DEBUG] Исходная строка адреса: %q\n", addrStr)

	originalAddrStr := addrStr

	for strings.HasPrefix(addrStr, ProtocolPrefix+"://") || strings.HasPrefix(addrStr, ProtocolPrefix+":") {
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+":")
	}

	fmt.Printf("[DEBUG] После удаления префиксов: %q\n", addrStr)

	targetPeerID, allAddrs, err := parsePeerAddress(addrStr)
	if err != nil {
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
func ExtractPeerIDFromAddress(addrStr string) (string, error) {
	originalAddrStr := addrStr

	for strings.HasPrefix(addrStr, ProtocolPrefix+"://") || strings.HasPrefix(addrStr, ProtocolPrefix+":") {
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+":")
	}

	targetPeerID, _, err := parsePeerAddress(addrStr)
	if err == nil {
		return targetPeerID.String(), nil
	}

	if strings.Contains(originalAddrStr, "@") {
		parts := strings.SplitN(originalAddrStr, "@", 2)
		if len(parts) == 2 {
			pid, pidErr := peer.Decode(parts[0])
			if pidErr == nil {
				return pid.String(), nil
			}

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

	originalAddrStr := addrStr

	for strings.HasPrefix(addrStr, ProtocolPrefix+"://") || strings.HasPrefix(addrStr, ProtocolPrefix+":") {
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")
		addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+":")
	}

	fmt.Printf("[DEBUG] После удаления префиксов: %q\n", addrStr)

	targetPeerID, allAddrs, err := parsePeerAddress(addrStr)
	if err != nil {
		if strings.Contains(originalAddrStr, "@") {
			fmt.Println("[DEBUG] Пробуем legacy формат peerid@multiaddr...")
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

	targetPeerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return "", nil, fmt.Errorf("неверный PeerID: %w", err)
	}

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

// ParsePeerAddressString парсит строку адреса в формате peerid@multiaddr
func ParsePeerAddressString(addrStr string) (*PeerAddress, error) {
	originalAddrStr := addrStr

	addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+"://")
	addrStr = strings.TrimPrefix(addrStr, ProtocolPrefix+":")

	parts := strings.SplitN(addrStr, "@", 2)
	if len(parts) != 2 {
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
	peerID = strings.TrimPrefix(peerID, ProtocolPrefix+":")

	ma := parts[1]

	pid, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("неверный PeerID: %w", err)
	}

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

func addPrefixToData(data []byte) []byte {
	prefix := []byte(ProtocolPrefix + ":")
	result := make([]byte, len(prefix)+len(data))
	copy(result, prefix)
	copy(result[len(prefix):], data)
	return result
}
