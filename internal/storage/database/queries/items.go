package queries

import (
	"database/sql"
	"encoding/json"
	"errors"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
	"time"
)

// CreateItem создает новый элемент
func CreateItem(item *models.Item) error {
	// Устанавливаем значения по умолчанию для новых полей
	if item.OwnerType == "" {
		item.OwnerType = models.OwnerTypeLocal
	}
	if item.Version == 0 {
		item.Version = 1
	}
	if item.Status == "" {
		item.Status = models.ItemStatusSaved
	}

	query := `
		INSERT INTO items (
			element_uuid, hash,
			owner_type, source_peer_id,
			type, title, description, content_meta, parent_id,
			signature, version, status, cached_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := database.DB.Exec(query,
		item.ElementUUID, item.Hash,
		item.OwnerType, item.SourcePeerID,
		item.Type, item.Title, item.Description, item.ContentMeta, item.ParentID,
		item.Signature, item.Version, item.Status, item.CachedAt, time.Now(), time.Now(),
	)
	if err != nil {
		return err
	}

	// Получаем ID вставленной записи
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Обновляем ID элемента
	item.ID = int(id)
	return nil
}

// GetItemByID возвращает элемент по ID
func GetItemByID(id int) (*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_id,
		       signature, version, status, cached_at, created_at, updated_at
		FROM items
		WHERE id = ?
	`
	var item models.Item
	var parentID sql.NullInt64
	var sourcePeerID sql.NullString
	var cachedAt, createdAt, updatedAt sql.NullTime
	var status string

	err := database.DB.QueryRow(query, id).Scan(
		&item.ID, &item.ElementUUID, &item.Hash,
		&item.OwnerType, &sourcePeerID,
		&item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID,
		&item.Signature, &item.Version, &status, &cachedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		parentIDValue := int(parentID.Int64)
		item.ParentID = &parentIDValue
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

	return &item, nil
}

// GetItemByHash возвращает элемент по хешу содержимого
func GetItemByHash(hash string) (*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_id,
		       signature, version, status, cached_at, created_at, updated_at
		FROM items
		WHERE hash = ?
	`
	var item models.Item
	var parentID sql.NullInt64
	var sourcePeerID sql.NullString
	var cachedAt, createdAt, updatedAt sql.NullTime
	var status string

	err := database.DB.QueryRow(query, hash).Scan(
		&item.ID, &item.ElementUUID, &item.Hash,
		&item.OwnerType, &sourcePeerID,
		&item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID,
		&item.Signature, &item.Version, &status, &cachedAt, &createdAt, &updatedAt,
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

	if sourcePeerID.Valid {
		item.SourcePeerID = &sourcePeerID.String
	}

	if cachedAt.Valid {
		item.CachedAt = &cachedAt.Time
	}

	item.CreatedAt = createdAt.Time
	item.UpdatedAt = updatedAt.Time
	item.Status = models.ItemStatus(status)

	return &item, nil
}

// GetItemByElementUUID возвращает элемент по element_uuid
func GetItemByElementUUID(elementUUID string) (*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_id,
		       signature, version, status, cached_at, created_at, updated_at
		FROM items
		WHERE element_uuid = ?
	`
	var item models.Item
	var parentID sql.NullInt64
	var sourcePeerID sql.NullString
	var cachedAt, createdAt, updatedAt sql.NullTime
	var status string

	err := database.DB.QueryRow(query, elementUUID).Scan(
		&item.ID, &item.ElementUUID, &item.Hash,
		&item.OwnerType, &sourcePeerID,
		&item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID,
		&item.Signature, &item.Version, &status, &cachedAt, &createdAt, &updatedAt,
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

	if sourcePeerID.Valid {
		item.SourcePeerID = &sourcePeerID.String
	}

	if cachedAt.Valid {
		item.CachedAt = &cachedAt.Time
	}

	item.CreatedAt = createdAt.Time
	item.UpdatedAt = updatedAt.Time
	item.Status = models.ItemStatus(status)

	return &item, nil
}

// GetItemByHash возвращает элемент по original_hash (устаревшее имя, используйте GetItemByHash)
// Оставлено для обратной совместимости
func GetItemByOriginalHash(hash string) (*models.Item, error) {
	return GetItemByHash(hash)
}

// GetItemsByParent возвращает элементы по родительскому ID
func GetItemsByParent(parentID int) ([]*models.Item, error) {
	var query string
	var rows *sql.Rows
	var err error

	if parentID == 0 {
		// Для корневого уровня (parent_id = 0 или parent_id IS NULL)
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id,
			       signature, version, cached_at, created_at, updated_at
			FROM items
			WHERE parent_id = 0 OR parent_id IS NULL
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query)
	} else {
		// Для конкретной папки
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id,
			       signature, version, cached_at, created_at, updated_at
			FROM items
			WHERE parent_id = ?
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query, parentID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID sql.NullInt64
		var sourcePeerID sql.NullString
		var cachedAt, createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&item.ID, &item.ElementUUID, &item.Hash,
			&item.OwnerType, &sourcePeerID,
			&item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID,
			&item.Signature, &item.Version, &cachedAt, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			parentIDValue := int(parentID.Int64)
			item.ParentID = &parentIDValue
		}

		if sourcePeerID.Valid {
			item.SourcePeerID = &sourcePeerID.String
		}

		if cachedAt.Valid {
			item.CachedAt = &cachedAt.Time
		}

		item.CreatedAt = createdAt.Time
		item.UpdatedAt = updatedAt.Time

		items = append(items, &item)
	}

	return items, nil
}

// GetAllItems возвращает все элементы из базы данных
func GetAllItems() ([]*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_id,
		       signature, version, status, cached_at, created_at, updated_at
		FROM items
		ORDER BY created_at DESC
	`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID sql.NullInt64
		var sourcePeerID sql.NullString
		var cachedAt, createdAt, updatedAt sql.NullTime
		var status string

		err := rows.Scan(
			&item.ID, &item.ElementUUID, &item.Hash,
			&item.OwnerType, &sourcePeerID,
			&item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID,
			&item.Signature, &item.Version, &status, &cachedAt, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			parentIDValue := int(parentID.Int64)
			item.ParentID = &parentIDValue
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

		items = append(items, &item)
	}

	return items, nil
}

// GetSavedItems возвращает только сохранённые элементы (status = 'saved')
func GetSavedItems() ([]*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_id,
		       signature, version, status, cached_at, created_at, updated_at
		FROM items
		WHERE status = 'saved'
		ORDER BY created_at DESC
	`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID sql.NullInt64
		var sourcePeerID sql.NullString
		var cachedAt, createdAt, updatedAt sql.NullTime
		var status string

		err := rows.Scan(
			&item.ID, &item.ElementUUID, &item.Hash,
			&item.OwnerType, &sourcePeerID,
			&item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID,
			&item.Signature, &item.Version, &status, &cachedAt, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			parentIDValue := int(parentID.Int64)
			item.ParentID = &parentIDValue
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

		items = append(items, &item)
	}

	return items, nil
}

// GetPreviewItems возвращает элементы со статусом 'preview' (загружены для просмотра)
func GetPreviewItems() ([]*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_id,
		       signature, version, status, cached_at, created_at, updated_at
		FROM items
		WHERE status = 'preview'
		ORDER BY created_at DESC
	`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID sql.NullInt64
		var sourcePeerID sql.NullString
		var cachedAt, createdAt, updatedAt sql.NullTime
		var status string

		err := rows.Scan(
			&item.ID, &item.ElementUUID, &item.Hash,
			&item.OwnerType, &sourcePeerID,
			&item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID,
			&item.Signature, &item.Version, &status, &cachedAt, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			parentIDValue := int(parentID.Int64)
			item.ParentID = &parentIDValue
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

		items = append(items, &item)
	}

	return items, nil
}

// PinItem закрепляет элемент
func PinItem(itemID int) error {
	// Добавляем в pinned_items
	query := `INSERT INTO pinned_items (item_id) VALUES (?)`
	_, err := database.DB.Exec(query, itemID)
	if err != nil {
		return err
	}

	// Синхронизируем pinned_uuids в profiles
	return updatePinnedUUIDs()
}

// UnpinItem открепляет элемент
func UnpinItem(itemID int) error {
	// Удаляем из pinned_items
	query := `DELETE FROM pinned_items WHERE item_id = ?`
	_, err := database.DB.Exec(query, itemID)
	if err != nil {
		return err
	}

	// Синхронизируем pinned_uuids в profiles
	return updatePinnedUUIDs()
}

// updatePinnedUUIDs обновляет pinned_uuids в profiles на основе pinned_items
func updatePinnedUUIDs() error {
	// Получаем все pinned items для локального профиля
	rows, err := database.DB.Query(`
		SELECT i.element_uuid 
		FROM pinned_items pi
		JOIN items i ON pi.item_id = i.id
		WHERE i.owner_type = 'local'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var uuids []string
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return err
		}
		uuids = append(uuids, uuid)
	}

	// Сериализуем UUID в JSON
	uuidsJSON, err := json.Marshal(uuids)
	if err != nil {
		return err
	}

	// Обновляем локальный профиль
	_, err = database.DB.Exec(`UPDATE profiles SET pinned_uuids = ? WHERE owner_type = 'local'`, string(uuidsJSON))
	return err
}

// IsItemPinned проверяет, закреплен ли элемент
func IsItemPinned(itemID int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM pinned_items WHERE item_id = ?`
	err := database.DB.QueryRow(query, itemID).Scan(&count)
	return count > 0, err
}

// UpdateItem обновляет элемент
func UpdateItem(item *models.Item) error {
	query := `
	UPDATE items
	SET element_uuid = ?, hash = ?,
	    owner_type = ?, source_peer_id = ?,
	    type = ?, title = ?, description = ?, content_meta = ?, parent_id = ?,
	    signature = ?, version = ?, status = ?, cached_at = ?, updated_at = ?
	WHERE id = ?
	`
	_, err := database.DB.Exec(query,
		item.ElementUUID, item.Hash,
		item.OwnerType, item.SourcePeerID,
		item.Type, item.Title, item.Description, item.ContentMeta, item.ParentID,
		item.Signature, item.Version, item.Status, item.CachedAt, time.Now(), item.ID,
	)
	return err
}

// SetItemStatus обновляет статус элемента
func SetItemStatus(itemID int, status models.ItemStatus) error {
	query := `UPDATE items SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := database.DB.Exec(query, status, itemID)
	return err
}

// DeleteItem удаляет элемент по ID
func DeleteItem(id int) error {
	query := `DELETE FROM items WHERE id = ?`
	_, err := database.DB.Exec(query, id)
	return err
}

// SearchItems выполняет поиск элементов по названию или тегам
func SearchItems(query string) ([]*models.Item, error) {
	// Подготавливаем параметры для поиска
	searchPattern := "%" + query + "%"

	// SQL-запрос для поиска по названию и через связь с тегами
	sqlQuery := `
	SELECT DISTINCT i.id, i.element_uuid, i.hash,
	       i.owner_type, i.source_peer_id,
	       i.type, i.title, i.description, i.content_meta, i.parent_id,
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
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID sql.NullInt64
		var sourcePeerID sql.NullString
		var cachedAt, createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&item.ID, &item.ElementUUID, &item.Hash,
			&item.OwnerType, &sourcePeerID,
			&item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID,
			&item.Signature, &item.Version, &cachedAt, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			parentIDValue := int(parentID.Int64)
			item.ParentID = &parentIDValue
		}

		if sourcePeerID.Valid {
			item.SourcePeerID = &sourcePeerID.String
		}

		if cachedAt.Valid {
			item.CachedAt = &cachedAt.Time
		}

		item.CreatedAt = createdAt.Time
		item.UpdatedAt = updatedAt.Time

		items = append(items, &item)
	}

	return items, nil
}
