// Package core предоставляет функции для обработки событий P2P сети
package core

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"

	"projectT/internal/storage/database/queries"
)

// onPeerConnected вызывается при подключении пира
func (n *P2PNetwork) onPeerConnected(peerID peer.ID, remoteAddr multiaddr.Multiaddr) {
	log.Printf("[P2P/Event] ========================================")
	log.Printf("[P2P/Event] Пир подключён: %s", peerID.String())

	// Сохраняем адрес пира в Peerstore, чтобы Connectedness() работал корректно
	if n.host != nil && remoteAddr != nil {
		n.host.Peerstore().AddAddr(peerID, remoteAddr, peerstore.PermanentAddrTTL)
		log.Printf("[P2P/Event] 📌 Адрес пира сохранён в Peerstore: %s", remoteAddr.String())

		// ✅ Сохраняем адрес пира в БД для восстановления после перезапуска
		// Формируем полный multiaddr с PeerID
		fullAddr := remoteAddr.String() + "/p2p/" + peerID.String()
		if err := queries.AddPeerAddressWithProfile(peerID.String(), fullAddr, "discovered", "auto_connect", ""); err != nil {
			log.Printf("[P2P/Event] ⚠️ Не удалось сохранить адрес в БД: %v", err)
		} else {
			log.Printf("[P2P/Event] 💾 Адрес пира сохранён в БД: %s", fullAddr)
		}
	}

	// Определяем тип соединения и логируем детали
	n.logConnectionDetails(peerID)

	// Обновляем время последней активности контакта
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil {
		now := time.Now()
		_ = queries.UpdateContactLastSeen(contact.ID, &now)
	}

	// Запускаем синхронизацию профилей асинхронно
	// Чтобы избежать race condition (оба пира инициируют одновременно),
	// только пир с БОЛЬШИМ PeerID инициирует синхронизацию
	if n.profileSync != nil && n.host != nil {
		ourID := n.host.ID()
		if ourID > peerID {
			go func() {
				ctx, cancel := context.WithTimeout(n.ctx, 60*time.Second)
				defer cancel()

				if err := n.profileSync.SyncWithPeer(ctx, peerID); err != nil {
					log.Printf("[ProfileSync] ⚠️ Ошибка синхронизации с %s: %v", peerID.String()[:8], err)
				}
			}()
		} else {
			log.Printf("[ProfileSync] ⏳ Ждём входящую синхронизацию от %s (наш PeerID меньше)", peerID.String()[:8])
		}
	}

	// Всегда обмениваемся ключами шифрования через ProfileExchange
	// Чтобы избежать race condition, только пир с БОЛЬШИМ PeerID инициирует обмен
	if n.profileExchange != nil && n.host != nil {
		ourID := n.host.ID()
		if ourID > peerID {
			log.Printf("[ProfileExchange] 📤 Инициируем обмен профилями с %s (наш PeerID больше)", peerID.String()[:8])
			go func() {
				ctx, cancel := context.WithTimeout(n.ctx, 60*time.Second)
				defer cancel()

				if _, err := n.profileExchange.RequestPeerProfile(ctx, peerID); err != nil {
					log.Printf("[ProfileExchange] ⚠️ Ошибка обмена профилями с %s: %v", peerID.String()[:8], err)
				} else {
					log.Printf("[ProfileExchange] ✅ Обмен профилями с %s завершён", peerID.String()[:8])
				}
			}()
		} else {
			log.Printf("[ProfileExchange] ⏳ Ждём обмен профилями от %s (наш PeerID меньше)", peerID.String()[:8])
		}
	} else {
		log.Printf("[ProfileExchange] ⚠️ profileExchange не инициализирован, обмен невозможен")
	}

	// Обмениваемся адресами пиров для расширения сети
	if n.peerExchange != nil && n.host != nil {
		go func() {
			ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
			defer cancel()

			if err := n.peerExchange.ExchangeWithPeer(ctx, peerID); err != nil {
				log.Printf("[PeerExchange] ⚠️ Ошибка обмена с %s: %v", peerID.String()[:8], err)
			}
		}()
	}
	log.Printf("[P2P/Event] ========================================")
}

// logConnectionDetails логирует детали подключения (тип, транспорт, адрес)
func (n *P2PNetwork) logConnectionDetails(peerID peer.ID) {
	if n.host == nil {
		return
	}

	// Получаем все соединения с этим пиром
	conns := n.host.Network().ConnsToPeer(peerID)
	if len(conns) == 0 {
		log.Printf("[P2P/Event] ⚠️ Соединений с %s не найдено в сети", peerID.String()[:8])
		return
	}

	for i, conn := range conns {
		remoteAddr := conn.RemoteMultiaddr()
		localAddr := conn.LocalMultiaddr()

		// Определяем тип соединения: direct или relay
		connType := "DIRECT"
		isCircuit := false

		addrStr := remoteAddr.String()
		if strings.Contains(addrStr, "/p2p-circuit") {
			connType = "RELAYED (circuit)"
			isCircuit = true
		} else if strings.Contains(addrStr, "/relay") {
			connType = "RELAYED"
		}

		// Определяем транспорт
		transport := "unknown"
		if strings.Contains(addrStr, "/tcp/") {
			transport = "TCP"
		} else if strings.Contains(addrStr, "/udp/") {
			if strings.Contains(addrStr, "/quic/") {
				transport = "QUIC"
			} else if strings.Contains(addrStr, "/webtransport/") {
				transport = "WebTransport"
			} else {
				transport = "UDP"
			}
		} else if strings.Contains(addrStr, "/ws/") || strings.Contains(addrStr, "/wss/") {
			transport = "WebSocket"
		}

		// Определяем, является ли удалённый адрес публичным
		remoteIsPublic := false
		if ip, err := remoteAddr.ValueForProtocol(4); err == nil {
			remoteIsPublic = !isPrivateIP(ip)
		} else if ip, err := remoteAddr.ValueForProtocol(41); err == nil {
			remoteIsPublic = !isPrivateIP(ip)
		}

		log.Printf("[P2P/Event] Соединение #%d с %s:", i+1, peerID.String()[:8])
		log.Printf("[P2P/Event]   Тип: %s", connType)
		log.Printf("[P2P/Event]   Транспорт: %s", transport)
		log.Printf("[P2P/Event]   Удалённый адрес: %s", remoteAddr.String())
		log.Printf("[P2P/Event]   Локальный адрес: %s", localAddr.String())
		log.Printf("[P2P/Event]   Удалённый публичный: %v", remoteIsPublic)

		if isCircuit {
			// Извлекаем адрес релея
			log.Printf("[P2P/Event]   ⚠️ Соединение через релеи — может быть высокая задержка")
		} else if !remoteIsPublic {
			log.Printf("[P2P/Event]   ℹ️ Удалённый пир находится в частной сети (NAT)")
		} else {
			log.Printf("[P2P/Event]   ✅ Прямое соединение с публичным адресом")
		}
	}
}

// onPeerDisconnected вызывается при отключении пира
func (n *P2PNetwork) onPeerDisconnected(peerID peer.ID) {
	log.Printf("[P2P/Event] Пир отключён: %s", peerID.String())

	// Уменьшаем счётчик подключений в autodial
	if n.autodial != nil {
		n.autodial.DecrementConnectedCount()
	}

	// Обновляем время последней активности контакта
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil {
		now := time.Now()
		_ = queries.UpdateContactLastSeen(contact.ID, &now)
	}
}

// handleChatStream обрабатывает входящий поток чата
func (n *P2PNetwork) handleChatStream(stream network.Stream) {
	defer func() { _ = stream.Close() }()
	// Делегируем обработку в ChatService
	if n.chat != nil {
		n.chat.HandleChatStream(stream)
	} else {
		log.Printf("[P2P/Event] Получен поток чата от: %s (ChatService не инициализирован)", stream.Conn().RemotePeer().String())
	}
}
