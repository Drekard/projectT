// Package models содержит модели данных для работы с базой данных.
package models

import (
	"database/sql"
	"time"
)

// BootstrapPeer представляет узел для подключения к сети
type BootstrapPeer struct {
	ID            int            `json:"id"`
	Multiaddr     string         `json:"multiaddr"`
	PeerID        sql.NullString `json:"peer_id"`
	IsActive      bool           `json:"is_active"`
	LastConnected *time.Time     `json:"last_connected"`
	AddedAt       time.Time      `json:"added_at"`
}
