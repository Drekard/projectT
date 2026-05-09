package models

import "time"

// PinnedItem представляет закрепленный элемент
type PinnedItem struct {
	ID              int       `json:"id"`
	ItemID          int       `json:"item_id"`
	ItemElementUUID string    `json:"item_element_uuid"`
	OrderNum        int       `json:"order_num"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
