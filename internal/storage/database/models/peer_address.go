// Package models содержит модели данных для работы с базой данных.
package models

import (
	"time"
)

// PeerAddress представляет адрес пира для подключения
type PeerAddress struct {
	ID            int        `json:"id"`
	ProfileID     int        `json:"profile_id"`
	Multiaddr     string     `json:"multiaddr"`
	AddressType   string     `json:"address_type"` // bootstrap, contact, discovered
	IsActive      bool       `json:"is_active"`
	LastConnected *time.Time `json:"last_connected"`
	LastSeen      *time.Time `json:"last_seen"`
	Priority      int        `json:"priority"`
	Source        string     `json:"source"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Данные из профиля (для удобства)
	PeerID   string `json:"peer_id"`
	Username string `json:"username"`
}

// AddrInfo возвращает информацию для подключения
func (pa *PeerAddress) AddrInfo() (string, string) {
	return pa.PeerID, pa.Multiaddr
}

// IsBootstrap проверяет, является ли адрес bootstrap-узлом
func (pa *PeerAddress) IsBootstrap() bool {
	return pa.AddressType == "bootstrap"
}

// IsContact проверяет, является ли адрес контактом
func (pa *PeerAddress) IsContact() bool {
	return pa.AddressType == "contact"
}

// IsDiscovered проверяет, является ли адрес обнаруженным
func (pa *PeerAddress) IsDiscovered() bool {
	return pa.AddressType == "discovered"
}
