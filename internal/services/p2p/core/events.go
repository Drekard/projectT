// Package core предоставляет функции для обработки событий P2P сети
package core

import (
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/queries"
)

// onPeerConnected вызывается при подключении пира
func (n *P2PNetwork) onPeerConnected(peerID peer.ID) {
	log.Printf("Пир подключён: %s", peerID.String())

	// Обновляем время последней активности контакта
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil {
		now := time.Now()
		_ = queries.UpdateContactLastSeen(contact.ID, &now)
	}

	// Не запрашиваем профиль автоматически — это делается через UI при подключении
	// Профиль запрашивается только инициатором подключения, чтобы избежать гонки
}

// onPeerDisconnected вызывается при отключении пира
func (n *P2PNetwork) onPeerDisconnected(peerID peer.ID) {
	log.Printf("Пир отключён: %s", peerID.String())

	// Обновляем время последней активности контакта
	contact, err := queries.GetContactByPeerID(peerID.String())
	if err == nil && contact != nil {
		now := time.Now()
		_ = queries.UpdateContactLastSeen(contact.ID, &now)
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
