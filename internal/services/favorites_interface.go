package services

import "projectT/internal/storage/database/models"

// FavoritesServiceInterface определяет интерфейс для работы с избранным
type FavoritesServiceInterface interface {
	AddToFavorites(entityType string, entityID int) error
	RemoveFromFavorites(entityType string, entityID int) error
	IsFavorite(entityType string, entityID int) (bool, error)
	GetFavoriteFolders() ([]*models.Item, error)
	GetFavoriteTags() ([]*models.Tag, error)
	GetAllFavorites() ([]*models.Favorite, error)
}
