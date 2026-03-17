package models

// Favorite представляет избранный элемент (может быть тегом или папкой)
type Favorite struct {
	ID         int    `json:"-"`           // Локальный ID (скрыт из JSON)
	EntityType string `json:"entity_type"` // 'tag' или 'folder'
	EntityUUID string `json:"entity_uuid"` // Глобальный UUID тега или папки
}
