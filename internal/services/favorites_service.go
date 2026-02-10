package services

import (
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// FavoritesService предоставляет сервис для работы с избранным
type FavoritesService struct {
	favoritesImpl *queries.FavoritesServiceImpl
}

// NewFavoritesService создает новый экземпляр сервиса избранного
func NewFavoritesService() *FavoritesService {
	return &FavoritesService{
		favoritesImpl: queries.NewFavoritesServiceImpl(),
	}
}

// AddToFavorites добавляет элемент в избранное
func (fs *FavoritesService) AddToFavorites(entityType string, entityID int) error {
	err := fs.favoritesImpl.AddToFavorites(entityType, entityID)
	if err != nil {
		return err
	}

	// Уведомляем об изменении избранного
	eventManager := GetFavoritesEventManager()
	eventManager.Notify("favorites_changed")

	return nil
}

// RemoveFromFavorites удаляет элемент избранного
func (fs *FavoritesService) RemoveFromFavorites(entityType string, entityID int) error {
	err := fs.favoritesImpl.RemoveFromFavorites(entityType, entityID)
	if err != nil {
		return err
	}

	// Уведомляем об изменении избранного
	eventManager := GetFavoritesEventManager()
	eventManager.Notify("favorites_changed")

	return nil
}

// IsFavorite проверяет, является ли элемент избранным
func (fs *FavoritesService) IsFavorite(entityType string, entityID int) (bool, error) {
	return fs.favoritesImpl.IsFavorite(entityType, entityID)
}

// GetFavoriteFolders возвращает все избранные папки
func (fs *FavoritesService) GetFavoriteFolders() ([]*models.Item, error) {
	return fs.favoritesImpl.GetFavoriteFolders()
}

// GetFavoriteTags возвращает все избранные теги
func (fs *FavoritesService) GetFavoriteTags() ([]*models.Tag, error) {
	return fs.favoritesImpl.GetFavoriteTags()
}

// GetAllFavorites возвращает все избранные элементы
func (fs *FavoritesService) GetAllFavorites() ([]*models.Favorite, error) {
	return fs.favoritesImpl.GetAllFavorites()
}
