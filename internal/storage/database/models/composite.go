package models

// CompositeContentItem представляет элемент составного контента
type CompositeContentItem struct {
	ItemID int    `json:"item_id"` // ID связанного элемента
	Title  string `json:"title"`   // Название элемента
	Type   string `json:"type"`    // Тип элемента
	Order  int    `json:"order"`   // Порядок в составном элементе
}

// CompositeContent представляет структуру составного элемента
type CompositeContent struct {
	Items []CompositeContentItem `json:"items"` // Список элементов в составном элементе
	Meta  map[string]interface{} `json:"meta"`  // Дополнительные метаданные
}
