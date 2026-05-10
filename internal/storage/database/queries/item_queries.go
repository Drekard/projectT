package queries

import (
	"database/sql"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetItemsByParent возвращает элементы по родительскому ID
func GetItemsByParent(parentID int) ([]*models.Item, error) {
	var query string
	var rows *sql.Rows
	var err error

	if parentID == 0 {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, cached_at, created_at, updated_at
			FROM items
			WHERE parent_id = 0 OR parent_id IS NULL
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query)
	} else {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
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

// GetItemsByParentUUID возвращает дочерние элементы по parent_uuid
func GetItemsByParentUUID(parentUUID string) ([]*models.Item, error) {
	var query string
	var rows *sql.Rows
	var err error

	if parentUUID == "" {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE parent_uuid IS NULL OR parent_uuid = ''
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query)
	} else {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE parent_uuid = ?
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query, parentUUID)
	}

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

// GetAllItems возвращает все элементы из базы данных
func GetAllItems() ([]*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_id, parent_uuid,
		       signature, version, status, cached_at, created_at, updated_at
		FROM items
		ORDER BY created_at DESC
	`
	rows, err := database.DB.Query(query)
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

// GetSavedItems возвращает только сохранённые элементы (status = 'saved')
func GetSavedItems() ([]*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_id, parent_uuid,
		       signature, version, status, cached_at, created_at, updated_at
		FROM items
		WHERE status = 'saved'
		ORDER BY created_at DESC
	`
	rows, err := database.DB.Query(query)
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

// GetPreviewItems возвращает элементы со статусом 'preview'
func GetPreviewItems() ([]*models.Item, error) {
	query := `
		SELECT id, element_uuid, hash,
		       owner_type, source_peer_id,
		       type, title, description, content_meta, parent_id, parent_uuid,
		       signature, version, status, cached_at, created_at, updated_at
		FROM items
		WHERE status = 'preview'
		ORDER BY created_at DESC
	`
	rows, err := database.DB.Query(query)
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

// GetSavedItemsByParent возвращает элементы со статусом 'saved' по родительскому ID
func GetSavedItemsByParent(parentID int) ([]*models.Item, error) {
	var query string
	var rows *sql.Rows
	var err error

	if parentID == 0 {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE (parent_id = 0 OR parent_id IS NULL) AND status = 'saved'
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query)
	} else {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE parent_id = ? AND status = 'saved'
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query, parentID)
	}

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

// GetSavedItemsByParentUUID возвращает элементы со статусом 'saved' по parent_uuid
func GetSavedItemsByParentUUID(parentUUID string) ([]*models.Item, error) {
	var query string
	var rows *sql.Rows
	var err error

	if parentUUID == "" {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE (parent_uuid IS NULL OR parent_uuid = '') AND status = 'saved'
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query)
	} else {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE parent_uuid = ? AND status = 'saved'
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query, parentUUID)
	}

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

// GetPreviewItemsByParent возвращает элементы со статусом 'preview' по родительскому ID
func GetPreviewItemsByParent(parentID int) ([]*models.Item, error) {
	var query string
	var rows *sql.Rows
	var err error

	if parentID == 0 {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE (parent_id = 0 OR parent_id IS NULL) AND status = 'preview'
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query)
	} else {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE parent_id = ? AND status = 'preview'
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query, parentID)
	}

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

// GetPreviewItemsByParentUUID возвращает элементы со статусом 'preview' по parent_uuid
func GetPreviewItemsByParentUUID(parentUUID string) ([]*models.Item, error) {
	var query string
	var rows *sql.Rows
	var err error

	if parentUUID == "" {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE (parent_uuid IS NULL OR parent_uuid = '') AND status = 'preview'
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query)
	} else {
		query = `
			SELECT id, element_uuid, hash,
			       owner_type, source_peer_id,
			       type, title, description, content_meta, parent_id, parent_uuid,
			       signature, version, status, cached_at, created_at, updated_at
			FROM items
			WHERE parent_uuid = ? AND status = 'preview'
			ORDER BY updated_at DESC
		`
		rows, err = database.DB.Query(query, parentUUID)
	}

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
