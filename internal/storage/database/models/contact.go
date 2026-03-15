// Package models содержит модели данных для работы с базой данных.
package models

import "time"

// LocalChatPeerID специальный PeerID для локального чата (с самим собой)
const LocalChatPeerID = "__local__"

// Contact представляет контакт в адресной книге (избранные пользователи)
// Хранит только уникальные данные: адрес для подключения, заметки, настройки
// Профиль пользователя (username, avatar, title) хранится в таблице profiles
type Contact struct {
	ID        int       `json:"id"`
	PeerID    string    `json:"peer_id"` // FK → profiles.peer_id
	Multiaddr string    `json:"multiaddr"`
	Notes     string    `json:"notes"`
	IsBlocked bool      `json:"is_blocked"`
	AddedAt   time.Time `json:"added_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Поля для расширения (не хранятся в БД, заполняются через JOIN с profiles)
	Username   string     `json:"username,omitempty"`    // из profiles.username
	Title      string     `json:"title,omitempty"`       // из profiles.title (статус)
	AvatarPath string     `json:"avatar_path,omitempty"` // из profiles.avatar_path
	LastSeen   *time.Time `json:"last_seen,omitempty"`   // локальная история активности
	IsOnline   bool       `json:"is_online,omitempty"`   // динамический статус (не хранится в БД)
}

// IsLocalChat возвращает true, если это локальный чат (с самим собой)
func (c *Contact) IsLocalChat() bool {
	return c != nil && c.PeerID == LocalChatPeerID
}

// NewLocalContact создаёт контакт для локального чата
func NewLocalContact(username, title, avatarPath string) *Contact {
	return &Contact{
		ID:         0, // Специальный ID для локального чата
		PeerID:     LocalChatPeerID,
		Username:   username,
		Title:      title, // Используем title как статус в локальном чате
		AvatarPath: avatarPath,
	}
}
