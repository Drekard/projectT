package queries

import (
	"database/sql"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// FavoritesServiceImpl реализует интерфейс FavoritesServiceInterface
type FavoritesServiceImpl struct{}

// NewFavoritesServiceImpl создает новый экземпляр реализации сервиса избранного
func NewFavoritesServiceImpl() *FavoritesServiceImpl {
	return &FavoritesServiceImpl{}
}

// AddToFavorites добавляет элемент в избранное
func (f *FavoritesServiceImpl) AddToFavorites(entityType string, entityID int) error {
	query := `INSERT INTO favorites (entity_type, entity_id) VALUES (?, ?)`
	result, err := database.DB.Exec(query, entityType, entityID)
	if err != nil {
		return err
	}
	_, _ = result.RowsAffected()
	return nil
}

// RemoveFromFavorites удаляет элемент из избранного
func (f *FavoritesServiceImpl) RemoveFromFavorites(entityType string, entityID int) error {
	query := `DELETE FROM favorites WHERE entity_type = ? AND entity_id = ?`
	result, err := database.DB.Exec(query, entityType, entityID)
	if err != nil {
		return err
	}
	_, _ = result.RowsAffected()
	return nil
}

// IsFavorite проверяет, является ли элемент избранным
func (f *FavoritesServiceImpl) IsFavorite(entityType string, entityID int) (bool, error) {
	query := `SELECT 1 FROM favorites WHERE entity_type = ? AND entity_id = ?`
	var exists int
	err := database.DB.QueryRow(query, entityType, entityID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetFavoriteFolders возвращает все избранные папки
func (f *FavoritesServiceImpl) GetFavoriteFolders() ([]*models.Item, error) {
	query := `
		SELECT i.id, i.type, i.title, i.description, i.content_meta, i.parent_id, i.created_at, i.updated_at
		FROM items i
		INNER JOIN favorites f ON i.id = f.entity_id
		WHERE f.entity_type = 'folder'
	`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID *int
		err := rows.Scan(
			&item.ID, &item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		item.ParentID = parentID
		items = append(items, &item)
	}

	return items, nil
}

// GetFavoriteTags возвращает все избранные теги
func (f *FavoritesServiceImpl) GetFavoriteTags() ([]*models.Tag, error) {
	query := `
		SELECT t.id, t.name, t.color, t.description
		FROM tags t
		INNER JOIN favorites f ON t.id = f.entity_id
		WHERE f.entity_type = 'tag'
	`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Color, &tag.Description,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}

	return tags, nil
}

// GetAllFavorites возвращает все избранные элементы (и теги, и папки)
func (f *FavoritesServiceImpl) GetAllFavorites() ([]*models.Favorite, error) {
	query := `SELECT id, entity_type, entity_id FROM favorites ORDER BY id`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favorites []*models.Favorite
	for rows.Next() {
		var favorite models.Favorite
		err := rows.Scan(
			&favorite.ID, &favorite.EntityType, &favorite.EntityID,
		)
		if err != nil {
			return nil, err
		}
		favorites = append(favorites, &favorite)
	}

	return favorites, nil
}
