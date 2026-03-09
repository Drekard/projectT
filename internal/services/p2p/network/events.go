// Package network предоставляет функции для обработки событий P2P сети
package network

import (
	"context"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/queries"
)

// onPeerConnected вызывается при подключении пира
func (n *P2PNetwork) onPeerConnected(peerID peer.ID) {
	log.Printf("Пир подключён: %s", peerID.String())

	// Обновляем статус контакта в БД
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil {
		now := time.Now()
		_ = queries.UpdateContactStatus(contact.ID, "online", &now)
	}

	// Запрашиваем профиль у пира
	if n.profileExchange != nil {
		go func() {
			ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
			defer cancel()

			if _, err := n.profileExchange.RequestPeerProfile(ctx, peerID); err != nil {
				log.Printf("Не удалось получить профиль от %s: %v", peerID, err)
			}
		}()
	}
}

// onPeerDisconnected вызывается при отключении пира
func (n *P2PNetwork) onPeerDisconnected(peerID peer.ID) {
	log.Printf("Пир отключён: %s", peerID.String())

	// Обновляем статус контакта в БД
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil {
		now := time.Now()
		_ = queries.UpdateContactStatus(contact.ID, "offline", &now)
	}
}

// handleChatStream обрабатывает входящий поток чата
func (n *P2PNetwork) handleChatStream(stream network.Stream) {
	defer stream.Close()
	// Делегируем обработку в ChatService
	if n.chat != nil {
		n.chat.HandleChatStream(stream)
	} else {
		log.Printf("Получен поток чата от: %s (ChatService не инициализирован)", stream.Conn().RemotePeer().String())
	}
}
