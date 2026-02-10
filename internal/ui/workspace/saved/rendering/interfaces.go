package rendering

import (
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards/interfaces"
)

// CardInfo структура информации о карточке (временное определение до рефакторинга)
type CardInfo struct {
	// Поля будут определены после рефакторинга
}

// RenderFactoryInterface интерфейс для создания карточек
type RenderFactoryInterface interface {
	CreateCard(item *models.Item, options ...interface{}) interfaces.CardRenderer
}

// CardCacheInterface интерфейс для кэширования карточек
type CardCacheInterface interface {
	GetOrCreateCard(item *models.Item, factoryFn func(*models.Item) *CardInfo) *CardInfo
	GetCardSize(itemType models.ItemType) (int, int)
	Clear()
	Remove(itemID int)
}

// CardRenderer интерфейс для отображения карточек
type CardRenderer interface {
	GetWidget() interfaces.CardRenderer
}
