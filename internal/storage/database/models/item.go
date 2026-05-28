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

// ItemVisibility определяет видимость элемента для P2P
type ItemVisibility string

const (
	ItemVisibilityPublic  ItemVisibility = "public"  // Элемент доступен другим пирам
	ItemVisibilityPrivate ItemVisibility = "private" // Элемент не передаётся другим пирам
)

// Item представляет элемент в системе (локальный или кэшированный от другого пира)
type Item struct {
	ID              int            `json:"-"`                        // Внутренний ID для FK в SQLite (скрыт из JSON)
	ElementUUID     string         `json:"element_uuid"`             // Основной уникальный ID для P2P (UUID v4)
	Hash            string         `json:"hash"`                     // Хеш содержимого (title|description|content_meta) для дедупликации
	OwnerType       OwnerType      `json:"owner_type"`               // 'local' или 'remote'
	SourcePeerID    *string        `json:"source_peer_id,omitempty"` // PeerID владельца (для remote)
	Type            ItemType       `json:"type"`
	Title           string         `json:"title"`
	Description     string         `json:"description,omitempty"`
	ContentMeta     string         `json:"content_meta,omitempty"` // JSON для составных элементов
	ShowDescription bool           `json:"show_description"`       // Показывать описание на карточке
	ParentID        *int           `json:"parent_id,omitempty"`    // ID родительского элемента (legacy, используйте ParentUUID)
	ParentUUID      *string        `json:"parent_uuid,omitempty"`  // UUID родительского элемента для P2P
	Signature       []byte         `json:"signature,omitempty"`    // Подпись владельца (для remote)
	Version         int            `json:"version"`                // Версия элемента
	Status          ItemStatus     `json:"status"`                 // Статус просмотра: 'saved', 'preview', 'archived'
	Visibility      ItemVisibility `json:"visibility"`             // Видимость: 'public' или 'private'
	CachedAt        *time.Time     `json:"cached_at,omitempty"`    // Время кэширования (для remote)
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
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

// IsPublic возвращает true, если элемент публичный (доступен другим пирам)
func (i *Item) IsPublic() bool {
	return i.Visibility == "" || i.Visibility == ItemVisibilityPublic
}

// IsPrivate возвращает true, если элемент приватный (не передаётся другим пирам)
func (i *Item) IsPrivate() bool {
	return i.Visibility == ItemVisibilityPrivate
}
