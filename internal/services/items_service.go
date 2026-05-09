package services

import (
	"projectT/internal/metrics"
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
	err := queries.CreateItem(item)
	if err == nil && metrics.IsInitialized() {
		m := metrics.Get().Metrics
		m.ItemsCreated.Inc()
		m.ItemsTotal.Inc()
	}
	return err
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
	err := queries.DeleteItem(id)
	if err == nil && metrics.IsInitialized() {
		m := metrics.Get().Metrics
		m.ItemsDeleted.Inc()
		m.ItemsTotal.Dec()
	}
	return err
}

// SearchItems выполняет поиск элементов по запросу
func (is *ItemsService) SearchItems(query string) ([]*models.Item, error) {
	return queries.SearchItems(query)
}

// GetAllItemsWithoutParentFilter возвращает все элементы без фильтрации по родительскому ID
func (is *ItemsService) GetAllItemsWithoutParentFilter() ([]*models.Item, error) {
	return queries.GetAllItems()
}

// GetSavedItemsWithoutParentFilter возвращает только сохранённые элементы (status='saved') без фильтрации по родительскому ID
func (is *ItemsService) GetSavedItemsWithoutParentFilter() ([]*models.Item, error) {
	return queries.GetSavedItems()
}

// GetPreviewItemsWithoutParentFilter возвращает элементы со статусом 'preview' без фильтрации по родительскому ID
func (is *ItemsService) GetPreviewItemsWithoutParentFilter() ([]*models.Item, error) {
	return queries.GetPreviewItems()
}

// GetPreviewItemsByParent возвращает элементы со статусом 'preview' по родительскому ID
func (is *ItemsService) GetPreviewItemsByParent(parentID int) ([]*models.Item, error) {
	return queries.GetPreviewItemsByParent(parentID)
}

// GetSavedItemsByParent возвращает элементы со статусом 'saved' по родительскому ID
func (is *ItemsService) GetSavedItemsByParent(parentID int) ([]*models.Item, error) {
	return queries.GetSavedItemsByParent(parentID)
}

// SavePreviewItem изменяет статус элемента с 'preview' на 'saved'
func (is *ItemsService) SavePreviewItem(itemID int) error {
	return queries.SetItemStatus(itemID, models.ItemStatusSaved)
}

// GetItemsByParentUUID возвращает элементы по parent_uuid
func (is *ItemsService) GetItemsByParentUUID(parentUUID string) ([]*models.Item, error) {
	return queries.GetItemsByParentUUID(parentUUID)
}

// GetSavedItemsByParentUUID возвращает сохранённые элементы по parent_uuid
func (is *ItemsService) GetSavedItemsByParentUUID(parentUUID string) ([]*models.Item, error) {
	return queries.GetSavedItemsByParentUUID(parentUUID)
}

// GetPreviewItemsByParentUUID возвращает preview элементы по parent_uuid
func (is *ItemsService) GetPreviewItemsByParentUUID(parentUUID string) ([]*models.Item, error) {
	return queries.GetPreviewItemsByParentUUID(parentUUID)
}

// GetElementUUIDByID возвращает element_uuid по ID
func (is *ItemsService) GetElementUUIDByID(id int) (string, error) {
	return queries.GetElementUUIDByID(id)
}

// GetElementUUIDsByIDs возвращает список element_uuid по списку ID
func (is *ItemsService) GetElementUUIDsByIDs(ids []int) ([]string, error) {
	return queries.GetElementUUIDsByIDs(ids)
}
