// Package models содержит модели данных для работы с базой данных.
package models

import "time"

// OwnerType определяет тип владельца профиля
type OwnerType string

const (
	// OwnerTypeLocal локальный профиль (текущий пользователь)
	OwnerTypeLocal OwnerType = "local"
	// OwnerTypeRemote чужой профиль (кэшированный от другого пира)
	OwnerTypeRemote OwnerType = "remote"
)

// Profile представляет профиль пользователя (локальный или чужой)
type Profile struct {
	ID             int        `json:"id"`
	OwnerType      OwnerType  `json:"owner_type"` // "local" или "remote"
	PeerID         string     `json:"peer_id"`
	Username       string     `json:"username"`
	Status         string     `json:"status"`
	AvatarPath     string     `json:"avatar_path"`
	BackgroundPath string     `json:"background_path"`
	ContentChar    string     `json:"content_characteristic"`
	DemoElements   string     `json:"demo_elements"`
	CachedAt       *time.Time `json:"cached_at,omitempty"` // Только для remote
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
