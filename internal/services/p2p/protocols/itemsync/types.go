package itemsync

import (
	"projectT/internal/storage/database/models"
)

// ProtocolID идентификатор протокола синхронизации элементов
const ProtocolID = "/projectt/itemsync/1.0.0"

// ItemRequest запрос элементов
type ItemRequest struct {
	ItemIDs []int  `json:"item_ids,omitempty"`
	All     bool   `json:"all,omitempty"`
	Hash    string `json:"hash,omitempty"`
}

// ItemResponse ответ с элементом
type ItemResponse struct {
	ElementUUID string          `json:"element_uuid"`
	ItemID      int             `json:"item_id"`
	Hash        string          `json:"hash"`
	Type        models.ItemType `json:"type"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	ContentMeta string          `json:"content_meta,omitempty"`
	ParentUUID  *string         `json:"parent_uuid,omitempty"`
	Signature   []byte          `json:"signature,omitempty"`
	Timestamp   int64           `json:"timestamp"`
	FileData    *ItemFileData   `json:"file_data,omitempty"`
}

// ItemFileData данные о файле элемента
type ItemFileData struct {
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Content  []byte `json:"content,omitempty"`
}
