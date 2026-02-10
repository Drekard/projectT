package models

// Tag представляет тег
type Tag struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	ItemCount   int    `json:"item_count"`
}

// ItemTag связывает элемент и тег
type ItemTag struct {
	ItemID int `json:"item_id"`
	TagID  int `json:"tag_id"`
}
