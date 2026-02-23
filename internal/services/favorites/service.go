// Package favorites предоставляет сервис для работы с избранным.
package favorites

import (
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// Service предоставляет сервис для работы с избранным
type Service struct {
	favoritesImpl *queries.FavoritesServiceImpl
}

// NewService создает новый экземпляр сервиса избранного
func NewService() *Service {
	return &Service{
		favoritesImpl: queries.NewFavoritesServiceImpl(),
	}
}

// AddToFavorites добавляет элемент в избранное
func (s *Service) AddToFavorites(entityType string, entityID int) error {
	err := s.favoritesImpl.AddToFavorites(entityType, entityID)
	if err != nil {
		return err
	}

	// Уведомляем об изменении избранного
	eventManager := GetEventManager()
	eventManager.Notify("favorites_changed")

	return nil
}

// RemoveFromFavorites удаляет элемент избранного
func (s *Service) RemoveFromFavorites(entityType string, entityID int) error {
	err := s.favoritesImpl.RemoveFromFavorites(entityType, entityID)
	if err != nil {
		return err
	}

	// Уведомляем об изменении избранного
	eventManager := GetEventManager()
	eventManager.Notify("favorites_changed")

	return nil
}

// IsFavorite проверяет, является ли элемент избранным
func (s *Service) IsFavorite(entityType string, entityID int) (bool, error) {
	return s.favoritesImpl.IsFavorite(entityType, entityID)
}

// GetFavoriteFolders возвращает все избранные папки
func (s *Service) GetFavoriteFolders() ([]*models.Item, error) {
	return s.favoritesImpl.GetFavoriteFolders()
}

// GetFavoriteTags возвращает все избранные теги
func (s *Service) GetFavoriteTags() ([]*models.Tag, error) {
	return s.favoritesImpl.GetFavoriteTags()
}

// GetAllFavorites возвращает все избранные элементы
func (s *Service) GetAllFavorites() ([]*models.Favorite, error) {
	return s.favoritesImpl.GetAllFavorites()
}
