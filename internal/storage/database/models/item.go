package models

import "time"

// ItemType определяет тип элемента
type ItemType string

const (
	ItemTypeFolder  ItemType = "folder"
	ItemTypeElement ItemType = "element"
)

// ItemStatus определяет статус просмотра элемента
type ItemStatus string

const (
	ItemStatusSaved    ItemStatus = "saved"    // Сохранён в коллекции
	ItemStatusPreview  ItemStatus = "preview"  // Загружен для просмотра из чата
	ItemStatusArchived ItemStatus = "archived" // Архивированный элемент
)

// Item представляет элемент в системе (локальный или кэшированный от другого пира)
type Item struct {
	ID           int        `json:"-"`                        // Внутренний ID для FK в SQLite (скрыт из JSON)
	ElementUUID  string     `json:"element_uuid"`             // Основной уникальный ID для P2P (UUID v4)
	Hash         string     `json:"hash"`                     // Хеш содержимого (title|description|content_meta) для дедупликации
	OwnerType    OwnerType  `json:"owner_type"`               // 'local' или 'remote'
	SourcePeerID *string    `json:"source_peer_id,omitempty"` // PeerID владельца (для remote)
	Type         ItemType   `json:"type"`
	Title        string     `json:"title"`
	Description  string     `json:"description,omitempty"`
	ContentMeta  string     `json:"content_meta,omitempty"` // JSON для составных элементов
	ParentID     *int       `json:"parent_id,omitempty"`    // ID родительского элемента (если есть)
	Signature    []byte     `json:"signature,omitempty"`    // Подпись владельца (для remote)
	Version      int        `json:"version"`                // Версия элемента
	Status       ItemStatus `json:"status"`                 // Статус просмотра: 'saved', 'preview', 'archived'
	CachedAt     *time.Time `json:"cached_at,omitempty"`    // Время кэширования (для remote)
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// IsLocal возвращает true, если элемент локальный
func (i *Item) IsLocal() bool {
	return i.OwnerType == OwnerTypeLocal || i.OwnerType == ""
}

// IsRemote возвращает true, если элемент кэширован от другого пира
func (i *Item) IsRemote() bool {
	return i.OwnerType == OwnerTypeRemote
}

// GetIdentifier возвращает основной идентификатор элемента для P2P
// Для локальных элементов возвращает ElementUUID
// Для remote элементов возвращает ElementUUID (оригинальный UUID владельца)
func (i *Item) GetIdentifier() string {
	return i.ElementUUID
}

// IsDuplicateOf проверяет, является ли этот элемент дубликатом другого по содержимому
func (i *Item) IsDuplicateOf(other *Item) bool {
	if i.Hash == "" || other.Hash == "" {
		return false
	}
	return i.Hash == other.Hash
}

// IsSaved возвращает true, если элемент сохранён в коллекции
func (i *Item) IsSaved() bool {
	return i.Status == ItemStatusSaved
}

// IsPreview возвращает true, если элемент загружен для просмотра
func (i *Item) IsPreview() bool {
	return i.Status == ItemStatusPreview
}

// IsArchived возвращает true, если элемент архивирован
func (i *Item) IsArchived() bool {
	return i.Status == ItemStatusArchived
}
