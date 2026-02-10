package rendering

import (
	db_models "projectT/internal/storage/database/models"
	ui_models "projectT/internal/ui/workspace/saved/models"
)

// CardCache кэширует карточки для улучшения производительности
type CardCache struct {
	cache         map[int]*ui_models.CardInfo // Кэш по ID элемента
	rendererCache map[int]interface{}         // Кэш рендереров
	sizeCache     map[db_models.ItemType]ui_models.CardSize
}

// NewCardCache создает новый кэш карточек
func NewCardCache() *CardCache {
	return &CardCache{
		cache:         make(map[int]*ui_models.CardInfo),
		rendererCache: make(map[int]interface{}),
		sizeCache:     make(map[db_models.ItemType]ui_models.CardSize),
	}
}

// GetOrCreateCard получает карточку из кэша или создает новую
func (cc *CardCache) GetOrCreateCard(item *db_models.Item, factoryFn func(*db_models.Item) *ui_models.CardInfo) *ui_models.CardInfo {
	if card, exists := cc.cache[item.ID]; exists {
		return card
	}

	card := factoryFn(item)
	cc.cache[item.ID] = card
	return card
}

// GetCardSize возвращает размер карточки (кэшированный)
func (cc *CardCache) GetCardSize(itemType db_models.ItemType) (int, int) {
	if size, exists := cc.sizeCache[itemType]; exists {
		return size.Width, size.Height
	}

	// Все элементы, кроме папок, теперь имеют одинаковый размер
	var size ui_models.CardSize
	if itemType == db_models.ItemTypeFolder {
		size = ui_models.CardSize{Width: 2, Height: 1} // Папки могут иметь другой размер
	} else {
		size = ui_models.CardSize{Width: 1, Height: 1} // Все остальные элементы (включая element) имеют стандартный размер
	}

	cc.sizeCache[itemType] = size
	return size.Width, size.Height
}

// Clear очищает кэш
func (cc *CardCache) Clear() {
	cc.cache = make(map[int]*ui_models.CardInfo)
	cc.rendererCache = make(map[int]interface{})
}

// Remove удаляет элемент из кэша
func (cc *CardCache) Remove(itemID int) {
	delete(cc.cache, itemID)
	delete(cc.rendererCache, itemID)
}
