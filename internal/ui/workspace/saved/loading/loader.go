package loading

import (
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/saved/sorting"
)

// ItemLoader загружает элементы из базы данных
type ItemLoader struct {
	currentParentID int
	itemsService    *services.ItemsService
	sortingManager  *sorting.SortingManager
}

// NewItemLoader создает новый загрузчик элементов
func NewItemLoader() *ItemLoader {
	return &ItemLoader{
		itemsService:   services.NewItemsService(),
		sortingManager: sorting.NewSortingManager(),
	}
}

// LoadItemsByParent загружает элементы по родителю
func (il *ItemLoader) LoadItemsByParent(parentID int) ([]*models.Item, error) {
	items, err := il.itemsService.GetItemsByParent(parentID)
	if err == nil {
		il.currentParentID = parentID
	}
	return items, err
}

// LoadAndSortItemsByParent загружает и сортирует элементы по родителю с учетом настроек фильтрации
func (il *ItemLoader) LoadAndSortItemsByParent(parentID int, options *services.FilterOptions) ([]*models.Item, error) {
	items, err := il.itemsService.GetItemsByParent(parentID)
	if err == nil {
		il.currentParentID = parentID
		// Сортируем элементы по настройкам
		sortedItems := il.sortingManager.GetSortedItems(items, options)
		return sortedItems, err
	}
	return items, err
}

// LoadItemsBySearch загружает элементы по поиску
func (il *ItemLoader) LoadItemsBySearch(query string) ([]*models.Item, error) {
	return il.itemsService.SearchItems(query)
}

// LoadAndSortItemsBySearch загружает и сортирует элементы по поиску с учетом настроек фильтрации
func (il *ItemLoader) LoadAndSortItemsBySearch(query string, options *services.FilterOptions) ([]*models.Item, error) {
	items, err := il.itemsService.SearchItems(query)
	if err != nil {
		return items, err
	}

	// Сортируем элементы по настройкам
	sortedItems := il.sortingManager.GetSortedItems(items, options)
	return sortedItems, nil
}

// GetCurrentParentID возвращает текущий parent ID
func (il *ItemLoader) GetCurrentParentID() int {
	return il.currentParentID
}

// SetCurrentParentID устанавливает parent ID
func (il *ItemLoader) SetCurrentParentID(parentID int) {
	il.currentParentID = parentID
}
