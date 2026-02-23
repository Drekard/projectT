// Package background предоставляет сервис для работы с фоновыми изображениями.
package background

import (
	"sync"
)

// EventManager управляет событиями, связанными с фоновыми изображениями
type EventManager struct {
	mu          sync.RWMutex
	subscribers []chan<- string
}

var instance *EventManager
var once sync.Once

// GetEventManager возвращает глобальный экземпляр менеджера событий для фона
func GetEventManager() *EventManager {
	once.Do(func() {
		instance = &EventManager{
			subscribers: make([]chan<- string, 0),
		}
	})
	return instance
}

// Subscribe регистрирует нового подписчика на события фона
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
