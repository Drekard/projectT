package models

import "time"

// ItemType определяет тип элемента
type ItemType string

const (
	ItemTypeFolder  ItemType = "folder"
	ItemTypeElement ItemType = "element"
)

// Item представляет элемент в системе
type Item struct {
	ID          int       `json:"id"`
	Type        ItemType  `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	ContentMeta string    `json:"content_meta,omitempty"` // JSON для составных элементов
	ParentID    *int      `json:"parent_id,omitempty"`    // ID родительского элемента (если есть)
	IsPinned    *bool     `json:"is_pinned,omitempty"`    // Состояние закрепления элемента
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
