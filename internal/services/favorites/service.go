// Package favorites предоставляет сервис для работы с избранным.
package favorites

import (
	"log"
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
func (s *Service) AddToFavorites(entityType string, entityUUID string) error {
	log.Printf("[Favorites] AddToFavorites: entityType=%s, entityUUID=%s", entityType, entityUUID)
	err := s.favoritesImpl.AddToFavorites(entityType, entityUUID)
	if err != nil {
		log.Printf("[Favorites] AddToFavorites error: %v", err)
		return err
	}
	log.Printf("[Favorites] AddToFavorites success")

	// Уведомляем об изменении избранного
	eventManager := GetEventManager()
	eventManager.Notify("favorites_changed")

	return nil
}

// RemoveFromFavorites удаляет элемент избранного
func (s *Service) RemoveFromFavorites(entityType string, entityUUID string) error {
	err := s.favoritesImpl.RemoveFromFavorites(entityType, entityUUID)
	if err != nil {
		return err
	}

	// Уведомляем об изменении избранного
	eventManager := GetEventManager()
	eventManager.Notify("favorites_changed")

	return nil
}

// IsFavorite проверяет, является ли элемент избранным
func (s *Service) IsFavorite(entityType string, entityUUID string) (bool, error) {
	return s.favoritesImpl.IsFavorite(entityType, entityUUID)
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
