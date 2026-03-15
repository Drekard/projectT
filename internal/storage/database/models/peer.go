// Package models содержит модели данных для работы с базой данных.
package models

import "time"

// P2PProfile представляет профиль P2P узла
type P2PProfile struct {
	ID             int       `json:"id"`
	PeerID         string    `json:"peer_id"`
	PrivateKey     []byte    `json:"-"`                // Не экспортируем в JSON
	PublicKey      []byte    `json:"-"`                // Не экспортируем в JSON
	IsKeyEncrypted bool      `json:"is_key_encrypted"` // Маркер: зашифрован ли приватный ключ
	Username       string    `json:"username"`
	Title          string    `json:"title"`
	ListenAddrs    string    `json:"listen_addrs"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
