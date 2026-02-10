package events

import (
	"sync"
)

// EventManager реализует менеджер событий
type EventManager struct {
	handlers map[EventType][]EventHandler
	mutex    sync.RWMutex
}

// NewEventManager создает новый менеджер событий
func NewEventManager() *EventManager {
	return &EventManager{
		handlers: make(map[EventType][]EventHandler),
	}
}

// Emit генерирует событие
func (em *EventManager) Emit(event Event) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	if handlers, exists := em.handlers[event.Type]; exists {
		for _, handler := range handlers {
			handler(event)
		}
	}
}

// On регистрирует обработчик события
func (em *EventManager) On(eventType EventType, handler EventHandler) {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.handlers[eventType] = append(em.handlers[eventType], handler)
}

// Off удаляет обработчик события
func (em *EventManager) Off(eventType EventType, handler EventHandler) {
	// В простой реализации мы не будем удалять конкретный обработчик
	// потому что в Go нельзя сравнить функции напрямую
	// В реальном приложении можно использовать идентификаторы обработчиков
}
