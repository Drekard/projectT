package queries

import (
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// SearchItems выполняет поиск элементов по названию или тегам
func SearchItems(query string) ([]*models.Item, error) {
	searchPattern := "%" + query + "%"

	sqlQuery := `
	SELECT DISTINCT i.id, i.element_uuid, i.hash,
	       i.owner_type, i.source_peer_id,
	       i.type, i.title, i.description, i.content_meta, i.parent_id, i.parent_uuid,
	       i.signature, i.version, i.cached_at, i.created_at, i.updated_at
	FROM items i
	LEFT JOIN item_tags it ON i.id = it.item_id
	LEFT JOIN tags t ON it.tag_id = t.id
	WHERE i.title LIKE ? OR i.description LIKE ? OR t.name LIKE ?
	ORDER BY i.updated_at DESC
	`

	rows, err := database.DB.Query(sqlQuery, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []*models.Item
	for rows.Next() {
		item := scanItemRow(rows)
		if item == nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}
