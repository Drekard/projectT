// Package pinned предоставляет сервис для работы с закрепленными элементами.
package pinned

import (
	"projectT/internal/storage/database/queries"
)

// Service предоставляет сервис для работы с закрепленными элементами
type Service struct {
	eventsManager *EventManager
}

// NewService создает новый экземпляр сервиса закрепленных элементов
func NewService() *Service {
	return &Service{
		eventsManager: GetEventManager(),
	}
}

// PinItem закрепляет элемент
func (s *Service) PinItem(itemID int) error {
	err := queries.PinItem(itemID)
	if err != nil {
		return err
	}

	// Уведомляем об изменении закрепленных элементов
	s.eventsManager.Notify("pinned_items_changed")

	return nil
}

// UnpinItem открепляет элемент
func (s *Service) UnpinItem(itemID int) error {
	err := queries.UnpinItem(itemID)
	if err != nil {
		return err
	}

	// Уведомляем об изменении закрепленных элементов
	s.eventsManager.Notify("pinned_items_changed")

	return nil
}

// IsItemPinned проверяет, закреплен ли элемент
func (s *Service) IsItemPinned(itemID int) (bool, error) {
	return queries.IsItemPinned(itemID)
}
