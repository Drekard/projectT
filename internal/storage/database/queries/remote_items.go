package queries

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// CreateRemoteItem создаёт кэшированный элемент от другого пира в таблице items
// По умолчанию устанавливается status = 'preview' (элемент загружен для просмотра)
func CreateRemoteItem(item *models.Item) error {
	// Устанавливаем status = 'preview' по умолчанию для remote элементов
	if item.Status == "" {
		item.Status = models.ItemStatusPreview
	}

	query := `
		INSERT INTO items (
			element_uuid, hash,
			owner_type, source_peer_id,
			type, title, description, content_meta, parent_uuid,
			signature, version, status,
			created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(source_peer_id, element_uuid) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			content_meta = excluded.content_meta,
			parent_uuid = excluded.parent_uuid,
			signature = excluded.signature,
			version = version + 1,
			cached_at = CURRENT_TIMESTAMP
	`
	result, err := database.DB.Exec(query,
		item.ElementUUID, item.Hash,
		models.OwnerTypeRemote, item.SourcePeerID,
		item.Type, item.Title, item.Description, item.ContentMeta, item.ParentUUID,
		item.Signature, item.Version, item.Status,
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

// GetRemoteItemByElementUUID возвращает элемент по element_uuid и PeerID владельца
func GetRemoteItemByElementUUID(sourcePeerID, elementUUID string) (*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_uuid,
		       signature, version, status, cached_at,
		       created_at, updated_at
		FROM items
		WHERE source_peer_id = ? AND element_uuid = ? AND owner_type = 'remote'
		LIMIT 1
	`

	rows, err := database.DB.Query(query, sourcePeerID, elementUUID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if rows.Next() {
		item := scanRemoteItemRow(rows)
		if item != nil {
			return item, nil
		}
	}

	return nil, errors.New("элемент не найден")
}

// GetRemoteItemByHash возвращает элемент по hash и PeerID владельца (устаревшее, используйте GetRemoteItemByElementUUID)
func GetRemoteItemByHash(sourcePeerID, hash string) (*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_uuid,
		       signature, version, status, cached_at,
		       created_at, updated_at
		FROM items
		WHERE source_peer_id = ? AND hash = ? AND owner_type = 'remote'
		LIMIT 1
	`

	rows, err := database.DB.Query(query, sourcePeerID, hash)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if rows.Next() {
		item := scanRemoteItemRow(rows)
		if item != nil {
			return item, nil
		}
	}

	return nil, errors.New("элемент не найден")
}

// GetRemoteItemsByPeer возвращает все кэшированные элементы от пира
func GetRemoteItemsByPeer(sourcePeerID string) ([]*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_uuid,
		       signature, version, status, cached_at,
		       created_at, updated_at
		FROM items
		WHERE source_peer_id = ? AND owner_type = 'remote'
		ORDER BY title
	`
	rows, err := database.DB.Query(query, sourcePeerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []*models.Item
	for rows.Next() {
		item := scanRemoteItemRow(rows)
		if item == nil {
			continue
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// GetRemoteItemsByParentUUID возвращает дочерние элементы remote папки по parent_uuid
func GetRemoteItemsByParentUUID(parentUUID string) ([]*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_uuid,
		       signature, version, status, cached_at,
		       created_at, updated_at
		FROM items
		WHERE owner_type = 'remote' AND parent_uuid = ?
		ORDER BY title
	`
	rows, err := database.DB.Query(query, parentUUID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []*models.Item
	for rows.Next() {
		item := scanRemoteItemRow(rows)
		if item == nil {
			continue
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// GetRemoteItemsByElementUUIDs возвращает кэшированные элементы по списку element_uuid
func GetRemoteItemsByElementUUIDs(sourcePeerID string, elementUUIDs []string) ([]*models.Item, error) {
	if len(elementUUIDs) == 0 {
		return []*models.Item{}, nil
	}

	placeholders := make([]string, len(elementUUIDs))
	args := make([]interface{}, 1+len(elementUUIDs))
	args[0] = sourcePeerID
	for i, uuid := range elementUUIDs {
		placeholders[i] = "?"
		args[i+1] = uuid
	}

	query := fmt.Sprintf(`
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_uuid,
		       signature, version, status, cached_at,
		       created_at, updated_at
		FROM items
		WHERE source_peer_id = ? AND owner_type = 'remote'
		  AND element_uuid IN (%s)
		ORDER BY title
	`, strings.Join(placeholders, ","))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []*models.Item
	for rows.Next() {
		item := scanRemoteItemRow(rows)
		if item == nil {
			continue
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// scanRemoteItemRow сканирует строку remote элемента
func scanRemoteItemRow(rows *sql.Rows) *models.Item {
	var item models.Item
	var parentUUID sql.NullString
	var sourcePeerIDNull sql.NullString
	var cachedAt, createdAt, updatedAt sql.NullTime
	var status string

	err := rows.Scan(
		&item.ID, &item.ElementUUID, &item.Hash,
		&item.OwnerType, &sourcePeerIDNull,
		&item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentUUID,
		&item.Signature, &item.Version, &status, &cachedAt,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil
	}

	if cachedAt.Valid {
		item.CachedAt = &cachedAt.Time
	}
	item.CreatedAt = createdAt.Time
	item.UpdatedAt = updatedAt.Time
	item.Status = models.ItemStatus(status)

	if parentUUID.Valid {
		item.ParentUUID = &parentUUID.String
	}

	if sourcePeerIDNull.Valid {
		item.SourcePeerID = &sourcePeerIDNull.String
	}

	return &item
}

// GetRemoteItemByID возвращает элемент по локальному ID
func GetRemoteItemByID(id int) (*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_uuid,
		       signature, version, status, cached_at,
		       created_at, updated_at
		FROM items
		WHERE id = ?
	`

	rows, err := database.DB.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if rows.Next() {
		item := scanRemoteItemRow(rows)
		if item != nil {
			return item, nil
		}
	}

	return nil, errors.New("элемент не найден")
}

// UpdateRemoteItem обновляет кэшированный элемент
func UpdateRemoteItem(item *models.Item) error {
	query := `
		UPDATE items
		SET element_uuid = ?, hash = ?,
		    title = ?, description = ?, content_meta = ?, signature = ?,
		    version = ?, status = ?, cached_at = CURRENT_TIMESTAMP
		WHERE id = ? AND owner_type = 'remote'
	`
	_, err := database.DB.Exec(query,
		item.ElementUUID, item.Hash,
		item.Title, item.Description, item.ContentMeta,
		item.Signature, item.Version, item.Status, item.ID,
	)
	return err
}

// DeleteRemoteItem удаляет кэшированный элемент
func DeleteRemoteItem(id int) error {
	_, err := database.DB.Exec(`DELETE FROM items WHERE id = ? AND owner_type = 'remote'`, id)
	return err
}

// DeleteRemoteItemsByPeer удаляет все кэшированные элементы от пира
func DeleteRemoteItemsByPeer(sourcePeerID string) error {
	_, err := database.DB.Exec(`DELETE FROM items WHERE source_peer_id = ? AND owner_type = 'remote'`, sourcePeerID)
	return err
}

// HasRemoteItem проверяет, существует ли элемент от пира с таким element_uuid
func HasRemoteItem(sourcePeerID, elementUUID string) (bool, error) {
	var exists bool
	err := database.DB.QueryRow(
		`SELECT COUNT(*) > 0 FROM items WHERE source_peer_id = ? AND element_uuid = ? AND owner_type = 'remote'`,
		sourcePeerID, elementUUID,
	).Scan(&exists)
	return exists, err
}

// HasRemoteItemByHash проверяет, существует ли элемент от пира с таким hash (устаревшее, используйте HasRemoteItem)
func HasRemoteItemByHash(sourcePeerID, hash string) (bool, error) {
	return HasRemoteItem(sourcePeerID, hash)
}

// GetAllRemoteItemsCount возвращает общее количество кэшированных элементов
func GetAllRemoteItemsCount() (int, error) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM items WHERE owner_type = 'remote'`).Scan(&count)
	return count, err
}

// GetLocalItemsCount возвращает количество локальных элементов
func GetLocalItemsCount() (int, error) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM items WHERE owner_type = 'local'`).Scan(&count)
	return count, err
}

// GetAllItemsCount возвращает общее количество всех элементов
func GetAllItemsCount() (int, error) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&count)
	return count, err
}
