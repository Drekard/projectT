// Package models содержит модели данных для работы с базой данных.
package models

import "time"

// ChatMessage представляет сообщение чата
type ChatMessage struct {
	ID          int       `json:"id"`
	ContactID   int       `json:"contact_id"`
	FromPeerID  string    `json:"from_peer_id"`
	Content     string    `json:"content"`
	ContentType string    `json:"content_type"`
	Metadata    string    `json:"metadata"`
	IsRead      bool      `json:"is_read"`
	SentAt      time.Time `json:"sent_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
