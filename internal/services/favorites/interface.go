// Package favorites предоставляет сервис для работы с избранным.
package favorites

import "projectT/internal/storage/database/models"

// ServiceInterface определяет интерфейс для работы с избранным
type ServiceInterface interface {
	AddToFavorites(entityType string, entityID int) error
	RemoveFromFavorites(entityType string, entityID int) error
	IsFavorite(entityType string, entityID int) (bool, error)
	GetFavoriteFolders() ([]*models.Item, error)
	GetFavoriteTags() ([]*models.Tag, error)
	GetAllFavorites() ([]*models.Favorite, error)
}
