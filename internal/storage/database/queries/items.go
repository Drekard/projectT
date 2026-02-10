package queries

import (
	"database/sql"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
	"time"
)

// CreateItem создает новый элемент
func CreateItem(item *models.Item) error {
	query := `
		INSERT INTO items (type, title, description, content_meta, parent_id, is_pinned, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	isPinned := false
	if item.IsPinned != nil {
		isPinned = *item.IsPinned
	}
	result, err := database.DB.Exec(query, item.Type, item.Title, item.Description, item.ContentMeta, item.ParentID, isPinned, time.Now(), time.Now())
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
		SELECT id, type, title, description, content_meta, parent_id, is_pinned, created_at, updated_at
		FROM items
	WHERE id = ?
	`
	var item models.Item
	var parentID sql.NullInt64
	var isPinned bool
	err := database.DB.QueryRow(query, id).Scan(
		&item.ID, &item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID, &isPinned, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		parentIDValue := int(parentID.Int64)
		item.ParentID = &parentIDValue
	}

	item.IsPinned = &isPinned

	return &item, nil
}

// GetItemsByParent возвращает элементы по родительскому ID
func GetItemsByParent(parentID int) ([]*models.Item, error) {
	query := `
		SELECT id, type, title, description, content_meta, parent_id, is_pinned, created_at, updated_at
		FROM items
		WHERE (parent_id = ? OR (parent_id IS NULL AND ? = 0))
		ORDER BY updated_at DESC
	`
	rows, err := database.DB.Query(query, parentID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID sql.NullInt64
		var isPinned bool
		err := rows.Scan(
			&item.ID, &item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID, &isPinned, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			parentIDValue := int(parentID.Int64)
			item.ParentID = &parentIDValue
		}

		item.IsPinned = &isPinned

		items = append(items, &item)
	}

	return items, nil
}

// GetAllItems возвращает все элементы из базы данных
func GetAllItems() ([]*models.Item, error) {
	db := database.DB
	query := `SELECT id, type, title, description, content_meta, parent_id, is_pinned, created_at, updated_at FROM items ORDER BY created_at DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID *int
		var isPinned *bool

		err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.Title,
			&item.Description,
			&item.ContentMeta,
			&parentID,
			&isPinned,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		item.ParentID = parentID
		item.IsPinned = isPinned

		items = append(items, &item)
	}

	return items, nil
}

// PinItem устанавливает значение is_pinned в true для указанного элемента
func PinItem(itemID int) error {
	query := `UPDATE items SET is_pinned = 1 WHERE id = ?`
	_, err := database.DB.Exec(query, itemID)
	return err
}

// UnpinItem устанавливает значение is_pinned в false для указанного элемента
func UnpinItem(itemID int) error {
	query := `UPDATE items SET is_pinned = 0 WHERE id = ?`
	_, err := database.DB.Exec(query, itemID)
	return err
}

// IsItemPinned возвращает значение is_pinned для указанного элемента
func IsItemPinned(itemID int) (bool, error) {
	var isPinned bool
	query := `SELECT is_pinned FROM items WHERE id = ?`
	err := database.DB.QueryRow(query, itemID).Scan(&isPinned)
	if err != nil {
		return false, err
	}
	return isPinned, nil
}

// UpdateItem обновляет элемент
func UpdateItem(item *models.Item) error {
	query := `
	UPDATE items
	SET type = ?, title = ?, description = ?, content_meta = ?, parent_id = ?, is_pinned = ?, updated_at = ?
	WHERE id = ?
	`
	isPinned := false
	if item.IsPinned != nil {
		isPinned = *item.IsPinned
	}
	_, err := database.DB.Exec(query, item.Type, item.Title, item.Description, item.ContentMeta, item.ParentID, isPinned, time.Now(), item.ID)
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
	SELECT DISTINCT i.id, i.type, i.title, i.description, i.content_meta, i.parent_id, i.is_pinned, i.created_at, i.updated_at
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
		var isPinned bool

		err := rows.Scan(
			&item.ID, &item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID, &isPinned, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			parentIDValue := int(parentID.Int64)
			item.ParentID = &parentIDValue
		}

		item.IsPinned = &isPinned

		items = append(items, &item)
	}

	return items, nil
}
