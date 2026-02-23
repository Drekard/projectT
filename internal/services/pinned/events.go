// Package pinned предоставляет сервис для работы с закрепленными элементами.
package pinned

import (
	"sync"
)

// EventManager управляет событиями, связанными с закрепленными элементами
type EventManager struct {
	mu          sync.RWMutex
	subscribers []chan<- string
}

var instance *EventManager
var once sync.Once

// GetEventManager возвращает глобальный экземпляр менеджера событий для закрепленных элементов
func GetEventManager() *EventManager {
	once.Do(func() {
		instance = &EventManager{
			subscribers: make([]chan<- string, 0),
		}
	})
	return instance
}

// Subscribe регистрирует нового подписчика на события закрепленных элементов
func (em *EventManager) Subscribe() <-chan string {
	ch := make(chan string, 10) // Буферизированный канал для предотвращения блокировки
	em.mu.Lock()
	defer em.mu.Unlock()
	em.subscribers = append(em.subscribers, ch)
	return ch
}

// Notify отправляет уведомление всем подписчикам
func (em *EventManager) Notify(eventType string) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	for _, subscriber := range em.subscribers {
		select {
		case subscriber <- eventType:
		default:
			// Если канал заблокирован, пропускаем уведомление
		}
	}
}
