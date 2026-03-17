package queries

import (
	"projectT/internal/storage/database/models"
)

// AddToFavorites добавляет элемент в избранное
func AddToFavorites(entityType string, entityUUID string) error {
	// Создаем реализацию сервиса избранного
	favoritesService := NewFavoritesServiceImpl()

	// Вызываем метод добавления в избранное
	err := favoritesService.AddToFavorites(entityType, entityUUID)
	if err != nil {
		return err
	}

	return nil
}

// RemoveFromFavorites удаляет элемент из избранного
func RemoveFromFavorites(entityType string, entityUUID string) error {
	// Создаем реализацию сервиса избранного
	favoritesService := NewFavoritesServiceImpl()

	// Вызываем метод удаления из избранного
	err := favoritesService.RemoveFromFavorites(entityType, entityUUID)
	if err != nil {
		return err
	}

	return nil
}

// IsFavorite проверяет, является ли элемент избранным
func IsFavorite(entityType string, entityUUID string) (bool, error) {
	// Создаем реализацию сервиса избранного
	favoritesService := NewFavoritesServiceImpl()

	// Вызываем метод проверки избранного
	return favoritesService.IsFavorite(entityType, entityUUID)
}

// GetFavoriteFolders возвращает все избранные папки
func GetFavoriteFolders() ([]*models.Item, error) {
	// Создаем реализацию сервиса избранного
	favoritesService := NewFavoritesServiceImpl()

	// Вызываем метод получения избранных папок
	return favoritesService.GetFavoriteFolders()
}

// GetFavoriteTags возвращает все избранные теги
func GetFavoriteTags() ([]*models.Tag, error) {
	// Создаем реализацию сервиса избранного
	favoritesService := NewFavoritesServiceImpl()

	// Вызываем метод получения избранных тегов
	return favoritesService.GetFavoriteTags()
}

// GetAllFavorites возвращает все избранные элементы (и теги, и папки)
func GetAllFavorites() ([]*models.Favorite, error) {
	// Создаем реализацию сервиса избранного
	favoritesService := NewFavoritesServiceImpl()

	// Вызываем метод получения всех избранных элементов
	return favoritesService.GetAllFavorites()
}
