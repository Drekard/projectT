package services

import (
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ItemsService предоставляет сервис для работы с элементами
type ItemsService struct{}

// NewItemsService создает новый экземпляр сервиса элементов
func NewItemsService() *ItemsService {
	return &ItemsService{}
}

// CreateItem создает новый элемент
func (is *ItemsService) CreateItem(item *models.Item) error {
	return queries.CreateItem(item)
}

// GetItemByID возвращает элемент по ID
func (is *ItemsService) GetItemByID(id int) (*models.Item, error) {
	return queries.GetItemByID(id)
}

// GetItemsByParent возвращает элементы по родительскому ID
func (is *ItemsService) GetItemsByParent(parentID int) ([]*models.Item, error) {
	return queries.GetItemsByParent(parentID)
}

// UpdateItem обновляет элемент
func (is *ItemsService) UpdateItem(item *models.Item) error {
	return queries.UpdateItem(item)
}

// DeleteItem удаляет элемент по ID
func (is *ItemsService) DeleteItem(id int) error {
	return queries.DeleteItem(id)
}

// SearchItems выполняет поиск элементов по запросу
func (is *ItemsService) SearchItems(query string) ([]*models.Item, error) {
	return queries.SearchItems(query)
}

// GetAllItemsWithoutParentFilter возвращает все элементы без фильтрации по родительскому ID
func (is *ItemsService) GetAllItemsWithoutParentFilter() ([]*models.Item, error) {
	return queries.GetAllItems()
}
