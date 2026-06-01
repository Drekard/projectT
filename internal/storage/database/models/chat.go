package models

import "time"

// Chat представляет чат (диалог с контактом или временный)
type Chat struct {
	ID            int        `json:"id"`
	ContactID     *int       `json:"contact_id"`      // NULL для временных чатов
	PeerID        string     `json:"peer_id"`         // PeerID собеседника
	IsTemporary   bool       `json:"is_temporary"`    // true для временных чатов
	LastMessageAt *time.Time `json:"last_message_at"` // Время последнего сообщения
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Данные из связанного контакта (если есть)
	Username   string `json:"username"`
	AvatarPath string `json:"avatar_path"`
}

// ChatWithLastMessage представляет чат с последним сообщением для отображения в UI
type ChatWithLastMessage struct {
	ID              int        `json:"id"`
	ContactID       *int       `json:"contact_id"`
	PeerID          string     `json:"peer_id"`
	Username        string     `json:"username"`
	AvatarPath      string     `json:"avatar_path"`
	IsTemporary     bool       `json:"is_temporary"`
	LastMessageID   int        `json:"last_message_id"`
	LastMessage     string     `json:"last_message"`
	LastMessageType string     `json:"last_message_type"`
	LastMessageAt   *time.Time `json:"last_message_at"`
	IsOutgoing      bool       `json:"is_outgoing"` // true если последнее сообщение от нас
	UnreadCount     int        `json:"unread_count"`
}

// UnifiedChatItem представляет объединённый элемент списка чатов (диалог, группа или канал)
type UnifiedChatItem struct {
	ID              int        `json:"id"`
	ChatType        string     `json:"chat_type"` // "direct", "group", "channel", "local"
	PeerID          string     `json:"peer_id"`   // для direct; для group/channel — GroupUUID
	GroupUUID       string     `json:"group_uuid"`
	Username        string     `json:"username"`
	AvatarPath      string     `json:"avatar_path"`
	LastMessageID   int        `json:"last_message_id"`
	LastMessage     string     `json:"last_message"`
	LastMessageType string     `json:"last_message_type"`
	LastMessageAt   *time.Time `json:"last_message_at"`
	IsOutgoing      bool       `json:"is_outgoing"`
	UnreadCount     int        `json:"unread_count"`
}
