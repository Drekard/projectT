package queries

import (
	"database/sql"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetPinnedItems возвращает все закрепленные элементы
func GetPinnedItems() ([]*models.Item, error) {
	query := `
		SELECT i.id, i.element_uuid, i.hash, i.type, i.title, i.description, i.content_meta, i.parent_id, i.parent_uuid, i.created_at, i.updated_at
		FROM items i
		INNER JOIN pinned_items pi ON i.id = pi.item_id
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentUUID sql.NullString
		err := rows.Scan(&item.ID, &item.ElementUUID, &item.Hash, &item.Type, &item.Title, &item.Description, &item.ContentMeta, &item.ParentID, &parentUUID, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if parentUUID.Valid {
			item.ParentUUID = &parentUUID.String
		}
		items = append(items, &item)
	}

	return items, nil
}

// GetPinnedItemUUIDs возвращает список element_uuid закреплённых элементов
func GetPinnedItemUUIDs() ([]string, error) {
	query := `
		SELECT i.element_uuid
		FROM items i
		INNER JOIN pinned_items pi ON i.id = pi.item_id
		ORDER BY pi.order_num
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var uuids []string
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			continue
		}
		uuids = append(uuids, uuid)
	}

	return uuids, nil
}
