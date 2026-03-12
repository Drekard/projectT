// Package models содержит модели данных для работы с базой данных.
package models

import "time"

// RemoteItem представляет кэшированный элемент от другого пира
type RemoteItem struct {
	ID           int       `json:"id"`
	SourcePeerID string    `json:"source_peer_id"` // PeerID владельца
	OriginalID   int       `json:"original_id"`    // ID элемента у владельца
	OriginalHash string    `json:"original_hash"`  // Content hash у владельца
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	ContentMeta  string    `json:"content_meta,omitempty"`
	Signature    []byte    `json:"signature,omitempty"` // Подпись владельца
	Version      int       `json:"version"`
	CachedAt     time.Time `json:"cached_at"`
}
