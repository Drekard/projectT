package services

import (
	"sync"
)

// PinnedEventManager управляет событиями, связанными с закрепленными элементами
type PinnedEventManager struct {
	mu          sync.RWMutex
	subscribers []chan<- string
}

var pinnedInstance *PinnedEventManager
var pinnedOnce sync.Once

// GetPinnedEventManager возвращает глобальный экземпляр менеджера событий для закрепленных элементов
func GetPinnedEventManager() *PinnedEventManager {
	pinnedOnce.Do(func() {
		pinnedInstance = &PinnedEventManager{
			subscribers: make([]chan<- string, 0),
		}
	})
	return pinnedInstance
}

// Subscribe регистрирует нового подписчика на события закрепленных элементов
func (pm *PinnedEventManager) Subscribe() <-chan string {
	ch := make(chan string, 10) // Буферизированный канал для предотвращения блокировки
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.subscribers = append(pm.subscribers, ch)
	return ch
}

// Notify отправляет уведомление всем подписчикам
func (pm *PinnedEventManager) Notify(eventType string) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, subscriber := range pm.subscribers {
		select {
		case subscriber <- eventType:
		default:
			// Если канал заблокирован, пропускаем уведомление
		}
	}
}