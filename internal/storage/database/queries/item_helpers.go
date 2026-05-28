package queries

import (
	"database/sql"
	"fmt"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
	"strings"
)

// GetItemsByParentUUIDs возвращает элементы по нескольким parent_uuid (batch запрос)
func GetItemsByParentUUIDs(parentUUIDs []string) ([]*models.Item, error) {
	if len(parentUUIDs) == 0 {
		return []*models.Item{}, nil
	}

	placeholders := make([]string, len(parentUUIDs))
	args := make([]interface{}, len(parentUUIDs))
	for i, uuid := range parentUUIDs {
		placeholders[i] = "?"
		args[i] = uuid
	}

	query := fmt.Sprintf(`
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, show_description, parent_id, parent_uuid,
		       signature, version, status, visibility, cached_at, created_at, updated_at
		FROM items
		WHERE parent_uuid IN (%s)
		ORDER BY parent_uuid, updated_at DESC
	`, strings.Join(placeholders, ","))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []*models.Item
	for rows.Next() {
		item := scanItemRowFull(rows)
		if item == nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// GetElementUUIDByID возвращает element_uuid по внутреннему ID
func GetElementUUIDByID(id int) (string, error) {
	var elementUUID string
	err := database.DB.QueryRow(`SELECT element_uuid FROM items WHERE id = ?`, id).Scan(&elementUUID)
	if err != nil {
		return "", err
	}
	return elementUUID, nil
}

// GetElementUUIDsByIDs возвращает список element_uuid по списку ID
func GetElementUUIDsByIDs(ids []int) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`SELECT element_uuid FROM items WHERE id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := database.DB.Query(query, args...)
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

// scanItemRowFull сканирует полную строку элемента (со status и visibility)
func scanItemRowFull(rows *sql.Rows) *models.Item {
	var item models.Item
	var parentID sql.NullInt64
	var parentUUID sql.NullString
	var sourcePeerID sql.NullString
	var cachedAt, createdAt, updatedAt sql.NullTime
	var status, visibility string

	err := rows.Scan(
		&item.ID, &item.ElementUUID, &item.Hash,
		&item.OwnerType, &sourcePeerID,
		&item.Type, &item.Title, &item.Description, &item.ContentMeta, &item.ShowDescription, &parentID, &parentUUID,
		&item.Signature, &item.Version, &status, &visibility, &cachedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil
	}

	if parentID.Valid {
		parentIDValue := int(parentID.Int64)
		item.ParentID = &parentIDValue
	}

	if parentUUID.Valid {
		item.ParentUUID = &parentUUID.String
	}

	if sourcePeerID.Valid {
		item.SourcePeerID = &sourcePeerID.String
	}

	if cachedAt.Valid {
		item.CachedAt = &cachedAt.Time
	}

	item.CreatedAt = createdAt.Time
	item.UpdatedAt = updatedAt.Time
	item.Status = models.ItemStatus(status)
	item.Visibility = models.ItemVisibility(visibility)

	return &item
}

// scanItemRow сканирует строку элемента (без status, для обратной совместимости)
func scanItemRow(rows *sql.Rows) *models.Item {
	var item models.Item
	var parentID sql.NullInt64
	var parentUUID sql.NullString
	var sourcePeerID sql.NullString
	var cachedAt, createdAt, updatedAt sql.NullTime

	err := rows.Scan(
		&item.ID, &item.ElementUUID, &item.Hash,
		&item.OwnerType, &sourcePeerID,
		&item.Type, &item.Title, &item.Description, &item.ContentMeta, &item.ShowDescription, &parentID, &parentUUID,
		&item.Signature, &item.Version, &cachedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil
	}

	if parentID.Valid {
		parentIDValue := int(parentID.Int64)
		item.ParentID = &parentIDValue
	}

	if parentUUID.Valid {
		item.ParentUUID = &parentUUID.String
	}

	if sourcePeerID.Valid {
		item.SourcePeerID = &sourcePeerID.String
	}

	if cachedAt.Valid {
		item.CachedAt = &cachedAt.Time
	}

	item.CreatedAt = createdAt.Time
	item.UpdatedAt = updatedAt.Time

	return &item
}
