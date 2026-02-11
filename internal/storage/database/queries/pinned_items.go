package queries

import (
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetPinnedItems возвращает все закрепленные элементы
func GetPinnedItems() ([]*models.Item, error) {
	query := `
		SELECT i.id, i.type, i.title, i.description, i.content_meta, i.parent_id, i.created_at, i.updated_at
		FROM items i
		INNER JOIN pinned_items pi ON i.id = pi.item_id
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Description, &item.ContentMeta, &item.ParentID, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	return items, nil
}