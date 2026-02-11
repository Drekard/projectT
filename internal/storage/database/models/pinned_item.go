package models

import "time"

// PinnedItem представляет закрепленный элемент
type PinnedItem struct {
	ID       int       `json:"id"`
	ItemID   int       `json:"item_id"`
	OrderNum int       `json:"order_num"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}