// Package models содержит модели данных для работы с базой данных.
package models

import "time"

// Contact представляет контакт в адресной книге
type Contact struct {
	ID          int        `json:"id"`
	PeerID      string     `json:"peer_id"`
	Username    string     `json:"username"`
	PublicKey   []byte     `json:"-"` // Не экспортируем в JSON
	Multiaddr   string     `json:"multiaddr"`
	Status      string     `json:"status"`
	LastSeen    *time.Time `json:"last_seen"`
	Notes       string     `json:"notes"`
	IsBlocked   bool       `json:"is_blocked"`
	AddedAt     time.Time  `json:"added_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
