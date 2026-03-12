// Package models содержит модели данных для работы с базой данных.
package models

// ItemFile представляет файл элемента (свой или чужой)
type ItemFile struct {
	ItemID       int    `json:"item_id"`
	Hash         string `json:"hash"`
	FilePath     string `json:"file_path"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mime_type,omitempty"`
	IsRemote     bool   `json:"is_remote"`
	SourcePeerID string `json:"source_peer_id,omitempty"` // Если remote
}
