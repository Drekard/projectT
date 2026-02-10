package sorting

import (
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
)

// ItemSortingInterface интерфейс для сортировки элементов
type ItemSortingInterface interface {
	SortItems(items []*models.Item, options *services.FilterOptions) []*models.Item
}

// SortingManager предоставляет методы для управления сортировкой
type SortingManager struct {
	sorter ItemSortingInterface
}

// NewSortingManager создает новый менеджер сортировки
func NewSortingManager() *SortingManager {
	return &SortingManager{
		sorter: NewItemSorter(),
	}
}

// GetSortedItems возвращает отсортированные элементы по заданным настройкам
func (sm *SortingManager) GetSortedItems(items []*models.Item, options *services.FilterOptions) []*models.Item {
	return sm.sorter.SortItems(items, options)
}
