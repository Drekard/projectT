package loading

import (
	"projectT/internal/storage/database/models"
)

// ItemLoaderInterface интерфейс для загрузки элементов
type ItemLoaderInterface interface {
	LoadItemsByParent(parentID int) ([]*models.Item, error)
	LoadItemsBySearch(query string) ([]*models.Item, error)
	GetCreateItem() *models.Item
	GetCurrentParentID() int
	SetCurrentParentID(parentID int)
}

// DataLoader интерфейс для загрузки данных
type DataLoader interface {
	FetchData(parentID int) ([]*models.Item, error)
	SearchData(query string) ([]*models.Item, error)
}
