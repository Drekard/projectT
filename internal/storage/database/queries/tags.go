package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
	"strings"
	"sync"
	"time"
)

// Кэш структуры таблицы тегов
var (
	tagTableChecked   bool
	tagTableHasDesc   bool
	tagTableMutex     sync.RWMutex
	tagTableCheckTime time.Time
)

// checkTagTableStructure проверяет структуру таблицы тегов с кэшированием
// Если tx передана, использует её, иначе использует database.DB
func checkTagTableStructure(ctx context.Context, tx *sql.Tx) (bool, error) {
	tagTableMutex.RLock()
	// Кэшируем на 5 минут
	if tagTableChecked && time.Since(tagTableCheckTime) < 5*time.Minute {
		hasDesc := tagTableHasDesc
		tagTableMutex.RUnlock()
		return hasDesc, nil
	}
	tagTableMutex.RUnlock()

	tagTableMutex.Lock()
	defer tagTableMutex.Unlock()

	// Двойная проверка
	if tagTableChecked && time.Since(tagTableCheckTime) < 5*time.Minute {
		return tagTableHasDesc, nil
	}

	query := `PRAGMA table_info(tags)`
	var rows *sql.Rows
	var err error

	if tx != nil {
		rows, err = tx.QueryContext(ctx, query)
	} else {
		rows, err = database.DB.QueryContext(ctx, query)
	}

	if err != nil {
		return false, fmt.Errorf("ошибка проверки структуры таблицы тегов: %w", err)
	}
	defer func() { _ = rows.Close() }()

	hasDescription := false
	for rows.Next() {
		var cid int
		var name string
		var notnull, dfltValue, pk interface{}
		// Используем sql.RawBytes для пропуска ненужных полей
		var typeRaw sql.RawBytes
		err := rows.Scan(&cid, &name, &typeRaw, &notnull, &dfltValue, &pk)
		if err != nil {
			continue
		}
		if name == "description" {
			hasDescription = true
			break
		}
	}

	// Обновляем кэш
	tagTableChecked = true
	tagTableHasDesc = hasDescription
	tagTableCheckTime = time.Now()

	return hasDescription, nil
}

// BeginTransaction начинает транзакцию
func BeginTransaction(ctx context.Context) (*sql.Tx, error) {
	return database.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
}

// CreateTag создает новый тег в транзакции
func CreateTag(ctx context.Context, tag *models.Tag) error {
	hasDesc, err := checkTagTableStructure(ctx, nil)
	if err != nil {
		return err
	}

	tx, err := BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Игнорируем ошибку отката, т.к. коммит уже мог состояться
	}()

	// Генерируем tag_uuid для нового тега
	tagUUID := generateTagUUID()

	var result sql.Result
	if hasDesc {
		result, err = tx.ExecContext(ctx,
			`INSERT INTO tags (name, description, color, tag_uuid, owner_peer_id) VALUES (?, ?, ?, ?, ?)`,
			tag.Name, tag.Description, tag.Color, tagUUID, "local",
		)
	} else {
		result, err = tx.ExecContext(ctx,
			`INSERT INTO tags (name, color, tag_uuid, owner_peer_id) VALUES (?, ?, ?, ?)`,
			tag.Name, tag.Color, tagUUID, "local",
		)
	}
	if err != nil {
		return fmt.Errorf("ошибка вставки тега: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("ошибка получения ID: %w", err)
	}
	tag.ID = int(id)
	tag.TagUUID = tagUUID
	tag.OwnerPeerID = "local"

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// GetTagByID возвращает тег по ID
func GetTagByID(ctx context.Context, id int) (*models.Tag, error) {
	hasDesc, err := checkTagTableStructure(ctx, nil)
	if err != nil {
		return nil, err
	}

	var tag models.Tag
	if hasDesc {
		err = database.DB.QueryRowContext(ctx,
			`SELECT id, name, description, color, tag_uuid, owner_peer_id FROM tags WHERE id = ?`,
			id,
		).Scan(&tag.ID, &tag.Name, &tag.Description, &tag.Color, &tag.TagUUID, &tag.OwnerPeerID)
	} else {
		err = database.DB.QueryRowContext(ctx,
			`SELECT id, name, color, tag_uuid, owner_peer_id FROM tags WHERE id = ?`,
			id,
		).Scan(&tag.ID, &tag.Name, &tag.Color, &tag.TagUUID, &tag.OwnerPeerID)
		tag.Description = ""
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("тег с ID %d не найден", id)
		}
		return nil, fmt.Errorf("ошибка получения тега: %w", err)
	}

	return &tag, nil
}

// GetTagByName возвращает тег по имени
func GetTagByName(ctx context.Context, name string) (*models.Tag, error) {
	hasDesc, err := checkTagTableStructure(ctx, nil)
	if err != nil {
		return nil, err
	}

	var tag models.Tag
	if hasDesc {
		err = database.DB.QueryRowContext(ctx,
			`SELECT id, name, description, color, tag_uuid, owner_peer_id FROM tags WHERE name = ?`,
			name,
		).Scan(&tag.ID, &tag.Name, &tag.Description, &tag.Color, &tag.TagUUID, &tag.OwnerPeerID)
	} else {
		err = database.DB.QueryRowContext(ctx,
			`SELECT id, name, color, tag_uuid, owner_peer_id FROM tags WHERE name = ?`,
			name,
		).Scan(&tag.ID, &tag.Name, &tag.Color, &tag.TagUUID, &tag.OwnerPeerID)
		tag.Description = ""
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("тег '%s' не найден", name)
		}
		return nil, fmt.Errorf("ошибка получения тега: %w", err)
	}

	return &tag, nil
}

// GetOrCreateTag получает существующий тег или создает новый
func GetOrCreateTag(ctx context.Context, name string) (*models.Tag, error) {
	// Сначала пытаемся получить существующий тег
	tag, err := GetTagByName(ctx, name)
	if err == nil {
		return tag, nil
	}

	// Если тег не найден, создаем новый
	newTag := &models.Tag{
		Name:  name,
		Color: "#808080", // Серый цвет по умолчанию
	}

	if err := CreateTag(ctx, newTag); err != nil {
		// Проверяем, не создался ли тег параллельно
		tag, err2 := GetTagByName(ctx, name)
		if err2 == nil {
			return tag, nil
		}
		return nil, fmt.Errorf("ошибка создания тега '%s': %w", name, err)
	}

	return newTag, nil
}

// GetOrCreateTags получает или создает несколько тегов
func GetOrCreateTags(ctx context.Context, tagNames []string) ([]int, error) {
	if len(tagNames) == 0 {
		return []int{}, nil
	}

	// Убираем дубликаты
	uniqueNames := make(map[string]bool)
	var cleanNames []string
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !uniqueNames[name] {
			uniqueNames[name] = true
			cleanNames = append(cleanNames, name)
		}
	}

	if len(cleanNames) == 0 {
		return []int{}, nil
	}

	// Начинаем транзакцию
	tx, err := BeginTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Игнорируем ошибку отката, т.к. коммит уже мог состояться
	}()

	// Получаем существующие теги
	placeholders := make([]string, len(cleanNames))
	args := make([]interface{}, len(cleanNames))
	for i, name := range cleanNames {
		placeholders[i] = "?"
		args[i] = name
	}

	query := fmt.Sprintf(
		`SELECT id, name FROM tags WHERE name IN (%s)`,
		strings.Join(placeholders, ","),
	)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса существующих тегов: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Собираем существующие теги
	existingTags := make(map[string]int)
	var tagIDs []int

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("ошибка сканирования тега: %w", err)
		}
		existingTags[name] = id
		tagIDs = append(tagIDs, id)
	}

	// Создаем отсутствующие теги
	hasDesc, err := checkTagTableStructure(ctx, tx)
	if err != nil {
		return nil, err
	}

	for _, name := range cleanNames {
		if _, exists := existingTags[name]; !exists {
			// Генерируем tag_uuid для нового тега
			tagUUID := generateTagUUID()
			var result sql.Result
			if hasDesc {
				result, err = tx.ExecContext(ctx,
					`INSERT INTO tags (name, color, tag_uuid, owner_peer_id) VALUES (?, ?, ?, ?)`,
					name, "#808080", tagUUID, "local",
				)
			} else {
				result, err = tx.ExecContext(ctx,
					`INSERT INTO tags (name, color, tag_uuid, owner_peer_id) VALUES (?, ?, ?, ?)`,
					name, "#808080", tagUUID, "local",
				)
			}
			if err != nil {
				return nil, fmt.Errorf("ошибка создания тега '%s': %w", name, err)
			}

			id, err := result.LastInsertId()
			if err != nil {
				return nil, fmt.Errorf("ошибка получения ID тега '%s': %w", name, err)
			}
			tagIDs = append(tagIDs, int(id))
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return tagIDs, nil
}

// GetAllTags возвращает все теги с подсчетом элементов
func GetAllTags(ctx context.Context) ([]*models.Tag, error) {
	hasDesc, err := checkTagTableStructure(ctx, nil)
	if err != nil {
		return nil, err
	}

	var query string
	if hasDesc {
		query = `
			SELECT t.id, t.name, t.color, t.description, t.tag_uuid, t.owner_peer_id,
			       COUNT(DISTINCT it.item_id) as item_count
			FROM tags t
			LEFT JOIN item_tags it ON t.id = it.tag_id
			GROUP BY t.id, t.name, t.color, t.description, t.tag_uuid, t.owner_peer_id
			ORDER BY t.name
		`
	} else {
		query = `
			SELECT t.id, t.name, t.color, t.tag_uuid, t.owner_peer_id,
			       COUNT(DISTINCT it.item_id) as item_count
			FROM tags t
			LEFT JOIN item_tags it ON t.id = it.tag_id
			GROUP BY t.id, t.name, t.color, t.tag_uuid, t.owner_peer_id
			ORDER BY t.name
		`
	}

	rows, err := database.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса тегов: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		var itemCount int

		if hasDesc {
			if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description, &tag.TagUUID, &tag.OwnerPeerID, &itemCount); err != nil {
				return nil, fmt.Errorf("ошибка сканирования тега: %w", err)
			}
		} else {
			if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.TagUUID, &tag.OwnerPeerID, &itemCount); err != nil {
				return nil, fmt.Errorf("ошибка сканирования тега: %w", err)
			}
			tag.Description = ""
		}
		tag.ItemCount = itemCount
		tags = append(tags, &tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации результатов: %w", err)
	}

	return tags, nil
}

// SearchTagsByName ищет теги по имени
func SearchTagsByName(ctx context.Context, name string) ([]*models.Tag, error) {
	hasDesc, err := checkTagTableStructure(ctx, nil)
	if err != nil {
		return nil, err
	}

	searchTerm := "%" + strings.ToLower(name) + "%"
	var query string

	if hasDesc {
		query = `
			SELECT t.id, t.name, t.color, t.description, t.tag_uuid, t.owner_peer_id,
			       COUNT(DISTINCT it.item_id) as item_count
			FROM tags t
			LEFT JOIN item_tags it ON t.id = it.tag_id
			WHERE LOWER(t.name) LIKE ?
			GROUP BY t.id, t.name, t.color, t.description, t.tag_uuid, t.owner_peer_id
			ORDER BY t.name
			LIMIT 50
		`
	} else {
		query = `
			SELECT t.id, t.name, t.color, t.tag_uuid, t.owner_peer_id,
			       COUNT(DISTINCT it.item_id) as item_count
			FROM tags t
			LEFT JOIN item_tags it ON t.id = it.tag_id
			WHERE LOWER(t.name) LIKE ?
			GROUP BY t.id, t.name, t.color, t.tag_uuid, t.owner_peer_id
			ORDER BY t.name
			LIMIT 50
		`
	}

	rows, err := database.DB.QueryContext(ctx, query, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска тегов: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		var itemCount int

		if hasDesc {
			if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description, &tag.TagUUID, &tag.OwnerPeerID, &itemCount); err != nil {
				return nil, fmt.Errorf("ошибка сканирования тега: %w", err)
			}
		} else {
			if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.TagUUID, &tag.OwnerPeerID, &itemCount); err != nil {
				return nil, fmt.Errorf("ошибка сканирования тега: %w", err)
			}
			tag.Description = ""
		}
		tag.ItemCount = itemCount
		tags = append(tags, &tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации результатов: %w", err)
	}

	return tags, nil
}

// UpdateTag обновляет тег
func UpdateTag(ctx context.Context, tag *models.Tag) error {
	hasDesc, err := checkTagTableStructure(ctx, nil)
	if err != nil {
		return err
	}

	tx, err := BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Игнорируем ошибку отката, т.к. коммит уже мог состояться
	}()

	if hasDesc {
		_, err = tx.ExecContext(ctx,
			`UPDATE tags SET name = ?, description = ?, color = ? WHERE id = ?`,
			tag.Name, tag.Description, tag.Color, tag.ID,
		)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE tags SET name = ?, color = ? WHERE id = ?`,
			tag.Name, tag.Color, tag.ID,
		)
	}
	if err != nil {
		return fmt.Errorf("ошибка обновления тега: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// DeleteTag удаляет тег
func DeleteTag(ctx context.Context, id int) error {
	tx, err := BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Игнорируем ошибку отката, т.к. коммит уже мог состояться
	}()

	// Удаляем связи с элементами
	_, err = tx.ExecContext(ctx, `DELETE FROM item_tags WHERE tag_id = ?`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления связей тега: %w", err)
	}

	// Удаляем сам тег
	_, err = tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления тега: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// AddTagToItem добавляет связь тега с элементом
func AddTagToItem(ctx context.Context, itemID, tagID int) error {
	// Получаем element_uuid и tag_uuid по локальным ID
	var elementUUID, tagUUID string

	err := database.DB.QueryRowContext(ctx, `SELECT element_uuid FROM items WHERE id = ?`, itemID).Scan(&elementUUID)
	if err != nil {
		return fmt.Errorf("ошибка получения element_uuid: %w", err)
	}

	err = database.DB.QueryRowContext(ctx, `SELECT tag_uuid FROM tags WHERE id = ?`, tagID).Scan(&tagUUID)
	if err != nil {
		return fmt.Errorf("ошибка получения tag_uuid: %w", err)
	}

	_, err = database.DB.ExecContext(ctx,
		`INSERT OR IGNORE INTO item_tags (item_id, item_element_uuid, tag_id, tag_uuid) VALUES (?, ?, ?, ?)`,
		itemID, elementUUID, tagID, tagUUID,
	)
	if err != nil {
		return fmt.Errorf("ошибка добавления связи тега: %w", err)
	}
	return nil
}

// RemoveTagFromItem удаляет связь тега с элементом
func RemoveTagFromItem(ctx context.Context, itemID, tagID int) error {
	// Получаем element_uuid и tag_uuid по локальным ID
	var elementUUID, tagUUID string

	err := database.DB.QueryRowContext(ctx, `SELECT element_uuid FROM items WHERE id = ?`, itemID).Scan(&elementUUID)
	if err != nil {
		return fmt.Errorf("ошибка получения element_uuid: %w", err)
	}

	err = database.DB.QueryRowContext(ctx, `SELECT tag_uuid FROM tags WHERE id = ?`, tagID).Scan(&tagUUID)
	if err != nil {
		return fmt.Errorf("ошибка получения tag_uuid: %w", err)
	}

	_, err = database.DB.ExecContext(ctx,
		`DELETE FROM item_tags WHERE item_element_uuid = ? AND tag_id = ?`,
		elementUUID, tagID,
	)
	if err != nil {
		return fmt.Errorf("ошибка удаления связи тега: %w", err)
	}
	return nil
}

// ReplaceItemTags заменяет все теги элемента на новые
func ReplaceItemTags(ctx context.Context, itemID int, tagIDs []int) error {
	tx, err := BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Игнорируем ошибку отката, т.к. коммит уже мог состояться
	}()

	// Получаем element_uuid по локальному ID
	var elementUUID string
	err = tx.QueryRowContext(ctx, `SELECT element_uuid FROM items WHERE id = ?`, itemID).Scan(&elementUUID)
	if err != nil {
		return fmt.Errorf("ошибка получения element_uuid: %w", err)
	}

	// Удаляем старые теги
	_, err = tx.ExecContext(ctx, `DELETE FROM item_tags WHERE item_element_uuid = ?`, elementUUID)
	if err != nil {
		return fmt.Errorf("ошибка удаления старых тегов: %w", err)
	}

	// Добавляем новые теги
	if len(tagIDs) > 0 {
		for _, tagID := range tagIDs {
			var tagUUID string
			err = tx.QueryRowContext(ctx, `SELECT tag_uuid FROM tags WHERE id = ?`, tagID).Scan(&tagUUID)
			if err != nil {
				return fmt.Errorf("ошибка получения tag_uuid: %w", err)
			}

			_, err = tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO item_tags (item_element_uuid, tag_id, tag_uuid) VALUES (?, ?, ?)`,
				elementUUID, tagID, tagUUID,
			)
			if err != nil {
				return fmt.Errorf("ошибка добавления тега: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// GetTagsForItem возвращает все теги элемента
func GetTagsForItem(ctx context.Context, itemID int) ([]*models.Tag, error) {
	hasDesc, err := checkTagTableStructure(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Используем item_element_uuid и tag_uuid для связи
	var query string
	if hasDesc {
		query = `
			SELECT t.id, t.tag_uuid, t.owner_peer_id, t.name, t.color, t.description
			FROM tags t
			INNER JOIN item_tags it ON t.tag_uuid = it.tag_uuid
			WHERE it.item_element_uuid = (SELECT element_uuid FROM items WHERE id = ?)
			ORDER BY t.name
		`
	} else {
		query = `
			SELECT t.id, t.tag_uuid, t.owner_peer_id, t.name, t.color
			FROM tags t
			INNER JOIN item_tags it ON t.tag_uuid = it.tag_uuid
			WHERE it.item_element_uuid = (SELECT element_uuid FROM items WHERE id = ?)
			ORDER BY t.name
		`
	}

	rows, err := database.DB.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса тегов элемента: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		if hasDesc {
			if err := rows.Scan(&tag.ID, &tag.TagUUID, &tag.OwnerPeerID, &tag.Name, &tag.Color, &tag.Description); err != nil {
				return nil, fmt.Errorf("ошибка сканирования тега: %w", err)
			}
		} else {
			if err := rows.Scan(&tag.ID, &tag.TagUUID, &tag.OwnerPeerID, &tag.Name, &tag.Color); err != nil {
				return nil, fmt.Errorf("ошибка сканирования тега: %w", err)
			}
			tag.Description = ""
		}
		tags = append(tags, &tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации результатов: %w", err)
	}

	return tags, nil
}

// GetItemsForTag возвращает все элементы тега
func GetItemsForTag(ctx context.Context, tagID int) ([]*models.Item, error) {
	query := `
		SELECT i.id, i.type, i.title, i.description, i.content_meta,
		       i.parent_id, i.created_at, i.updated_at
		FROM items i
		INNER JOIN item_tags it ON i.id = it.item_id
		WHERE it.tag_id = ?
		ORDER BY i.updated_at DESC
	`

	rows, err := database.DB.QueryContext(ctx, query, tagID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса элементов тега: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID *int
		if err := rows.Scan(
			&item.ID, &item.Type, &item.Title, &item.Description,
			&item.ContentMeta, &parentID, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования элемента: %w", err)
		}
		item.ParentID = parentID
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации результатов: %w", err)
	}

	return items, nil
}

// GetTagsUsageCount возвращает количество использований каждого тега
func GetTagsUsageCount(ctx context.Context) (map[int]int, error) {
	query := `
		SELECT tag_id, COUNT(*) as usage_count
		FROM item_tags
		GROUP BY tag_id
	`

	rows, err := database.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса статистики тегов: %w", err)
	}
	defer func() { _ = rows.Close() }()

	usageCount := make(map[int]int)
	for rows.Next() {
		var tagID, count int
		if err := rows.Scan(&tagID, &count); err != nil {
			return nil, fmt.Errorf("ошибка сканирования статистики: %w", err)
		}
		usageCount[tagID] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации результатов: %w", err)
	}

	return usageCount, nil
}

// BulkUpdateTags обновляет несколько тегов в одной транзакции
func BulkUpdateTags(ctx context.Context, tags []*models.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	tx, err := BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Игнорируем ошибку отката, т.к. коммит уже мог состояться
	}()

	hasDesc, err := checkTagTableStructure(ctx, tx)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		if hasDesc {
			_, err = tx.ExecContext(ctx,
				`UPDATE tags SET name = ?, description = ?, color = ? WHERE id = ?`,
				tag.Name, tag.Description, tag.Color, tag.ID,
			)
		} else {
			_, err = tx.ExecContext(ctx,
				`UPDATE tags SET name = ?, color = ? WHERE id = ?`,
				tag.Name, tag.Color, tag.ID,
			)
		}
		if err != nil {
			return fmt.Errorf("ошибка обновления тега %d: %w", tag.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// generateTagUUID генерирует уникальный UUID для тега
// Формат: T-xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx где x - случайные hex-символы
func generateTagUUID() string {
	uuid := make([]byte, 16)
	// Генерируем случайные байты
	for i := 0; i < 16; i++ {
		uuid[i] = byte(database.Rand.Intn(256))
	}
	// Устанавливаем версию (4) и вариант (10xx)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		uuid[0], uuid[1], uuid[2], uuid[3],
		uuid[4], uuid[5],
		uuid[6], uuid[7],
		uuid[8], uuid[9],
		uuid[10], uuid[11], uuid[12], uuid[13], uuid[14], uuid[15],
	)
}
