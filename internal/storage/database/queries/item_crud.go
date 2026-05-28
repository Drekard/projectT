package queries

import (
	"database/sql"
	"errors"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
	"time"
)

// CreateItem создает новый элемент
func CreateItem(item *models.Item) error {
	if item.OwnerType == "" {
		item.OwnerType = models.OwnerTypeLocal
	}
	if item.Version == 0 {
		item.Version = 1
	}
	if item.Status == "" {
		item.Status = models.ItemStatusSaved
	}
	if item.Visibility == "" {
		item.Visibility = models.ItemVisibilityPublic
	}

	query := `
		INSERT INTO items (
			element_uuid, hash,
			owner_type, source_peer_id,
			type, title, description, content_meta, show_description, parent_id, parent_uuid,
			signature, version, status, visibility, cached_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := database.DB.Exec(query,
		item.ElementUUID, item.Hash,
		item.OwnerType, item.SourcePeerID,
		item.Type, item.Title, item.Description, item.ContentMeta, item.ShowDescription, item.ParentID, item.ParentUUID,
		item.Signature, item.Version, item.Status, item.Visibility, item.CachedAt, time.Now(), time.Now(),
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	item.ID = int(id)
	return nil
}

// GetItemByID возвращает элемент по ID
func GetItemByID(id int) (*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, show_description, parent_id, parent_uuid,
		       signature, version, status, visibility, cached_at, created_at, updated_at
		FROM items
		WHERE id = ?
	`
	var item models.Item
	var parentID sql.NullInt64
	var parentUUID sql.NullString
	var sourcePeerID sql.NullString
	var cachedAt, createdAt, updatedAt sql.NullTime
	var status, visibility string

	err := database.DB.QueryRow(query, id).Scan(
		&item.ID, &item.ElementUUID, &item.Hash,
		&item.OwnerType, &sourcePeerID,
		&item.Type, &item.Title, &item.Description, &item.ContentMeta, &item.ShowDescription, &parentID, &parentUUID,
		&item.Signature, &item.Version, &status, &visibility, &cachedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
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

	return &item, nil
}

// GetItemByHash возвращает элемент по хешу содержимого
func GetItemByHash(hash string) (*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, show_description, parent_id, parent_uuid,
		       signature, version, status, visibility, cached_at, created_at, updated_at
		FROM items
		WHERE hash = ?
	`
	var item models.Item
	var parentID sql.NullInt64
	var parentUUID sql.NullString
	var sourcePeerID sql.NullString
	var cachedAt, createdAt, updatedAt sql.NullTime
	var status, visibility string

	err := database.DB.QueryRow(query, hash).Scan(
		&item.ID, &item.ElementUUID, &item.Hash,
		&item.OwnerType, &sourcePeerID,
		&item.Type, &item.Title, &item.Description, &item.ContentMeta, &item.ShowDescription, &parentID, &parentUUID,
		&item.Signature, &item.Version, &status, &visibility, &cachedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("элемент не найден")
		}
		return nil, err
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

	return &item, nil
}

// GetItemByElementUUID возвращает элемент по element_uuid
func GetItemByElementUUID(elementUUID string) (*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, show_description, parent_id, parent_uuid,
		       signature, version, status, visibility, cached_at, created_at, updated_at
		FROM items
		WHERE element_uuid = ?
	`
	var item models.Item
	var parentID sql.NullInt64
	var parentUUID sql.NullString
	var sourcePeerID sql.NullString
	var cachedAt, createdAt, updatedAt sql.NullTime
	var status, visibility string

	err := database.DB.QueryRow(query, elementUUID).Scan(
		&item.ID, &item.ElementUUID, &item.Hash,
		&item.OwnerType, &sourcePeerID,
		&item.Type, &item.Title, &item.Description, &item.ContentMeta, &item.ShowDescription, &parentID, &parentUUID,
		&item.Signature, &item.Version, &status, &visibility, &cachedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("элемент не найден")
		}
		return nil, err
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

	return &item, nil
}

// GetItemByOriginalHash возвращает элемент по original_hash (устаревшее имя, используйте GetItemByHash)
func GetItemByOriginalHash(hash string) (*models.Item, error) {
	return GetItemByHash(hash)
}

// UpdateItem обновляет элемент
func UpdateItem(item *models.Item) error {
	query := `
	UPDATE items
	SET element_uuid = ?, hash = ?,
	    owner_type = ?, source_peer_id = ?,
	    type = ?, title = ?, description = ?, content_meta = ?, show_description = ?, parent_id = ?, parent_uuid = ?,
	    signature = ?, version = ?, status = ?, visibility = ?, cached_at = ?, updated_at = ?
	WHERE id = ?
	`
	_, err := database.DB.Exec(query,
		item.ElementUUID, item.Hash,
		item.OwnerType, item.SourcePeerID,
		item.Type, item.Title, item.Description, item.ContentMeta, item.ShowDescription, item.ParentID, item.ParentUUID,
		item.Signature, item.Version, item.Status, item.Visibility, item.CachedAt, time.Now(), item.ID,
	)
	return err
}

// DeleteItem удаляет элемент по ID
func DeleteItem(id int) error {
	query := `DELETE FROM items WHERE id = ?`
	_, err := database.DB.Exec(query, id)
	return err
}
