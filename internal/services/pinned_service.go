package services

import (
	"projectT/internal/storage/database/queries"
)

// PinnedService предоставляет сервис для работы с закрепленными элементами
type PinnedService struct {
	pinnedEventsManager *PinnedEventManager
}

// NewPinnedService создает новый экземпляр сервиса закрепленных элементов
func NewPinnedService() *PinnedService {
	return &PinnedService{
		pinnedEventsManager: GetPinnedEventManager(),
	}
}

// PinItem закрепляет элемент
func (ps *PinnedService) PinItem(itemID int) error {
	err := queries.PinItem(itemID)
	if err != nil {
		return err
	}

	// Уведомляем об изменении закрепленных элементов
	ps.pinnedEventsManager.Notify("pinned_items_changed")

	return nil
}

// UnpinItem открепляет элемент
func (ps *PinnedService) UnpinItem(itemID int) error {
	err := queries.UnpinItem(itemID)
	if err != nil {
		return err
	}

	// Уведомляем об изменении закрепленных элементов
	ps.pinnedEventsManager.Notify("pinned_items_changed")

	return nil
}

// IsItemPinned проверяет, закреплен ли элемент
func (ps *PinnedService) IsItemPinned(itemID int) (bool, error) {
	return queries.IsItemPinned(itemID)
}