package models

// Favorite представляет избранный элемент (может быть тегом или папкой)
type Favorite struct {
	ID         int    `json:"id"`
	EntityType string `json:"entity_type"`
	EntityID   int    `json:"entity_id"`
}
