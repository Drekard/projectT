package models

// Tag представляет тег
type Tag struct {
	ID          int    `json:"-"`             // Локальный ID для FK (скрыт из JSON)
	TagUUID     string `json:"tag_uuid"`      // Глобальный UUID тега для P2P
	OwnerPeerID string `json:"owner_peer_id"` // Владелец тега ('local' или PeerID)
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	ItemCount   int    `json:"item_count"`
}

// IsLocal возвращает true, если тег локальный
func (t *Tag) IsLocal() bool {
	return t.OwnerPeerID == "" || t.OwnerPeerID == "local"
}

// IsRemote возвращает true, если тег получен от другого пира
func (t *Tag) IsRemote() bool {
	return !t.IsLocal()
}

// ItemTag связывает элемент и тег
type ItemTag struct {
	ItemID          int    `json:"-"`                 // Локальный ID элемента (для совместимости)
	ItemElementUUID string `json:"item_element_uuid"` // UUID элемента для P2P
	TagID           int    `json:"-"`                 // Локальный ID тега (для совместимости)
	TagUUID         string `json:"tag_uuid"`          // UUID тега для P2P
}
