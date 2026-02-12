package services

import (
	"sync"
)

// BackgroundEventManager управляет событиями, связанными с фоновыми изображениями
type BackgroundEventManager struct {
	mu          sync.RWMutex
	subscribers []chan<- string
}

var backgroundInstance *BackgroundEventManager
var backgroundOnce sync.Once

// GetBackgroundEventManager возвращает глобальный экземпляр менеджера событий для фона
func GetBackgroundEventManager() *BackgroundEventManager {
	backgroundOnce.Do(func() {
		backgroundInstance = &BackgroundEventManager{
			subscribers: make([]chan<- string, 0),
		}
	})
	return backgroundInstance
}

// Subscribe регистрирует нового подписчика на события фона
func (bm *BackgroundEventManager) Subscribe() <-chan string {
	ch := make(chan string, 10) // Буферизированный канал для предотвращения блокировки
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.subscribers = append(bm.subscribers, ch)
	return ch
}

// Notify отправляет уведомление всем подписчикам
func (bm *BackgroundEventManager) Notify(eventType string) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	for _, subscriber := range bm.subscribers {
		select {
		case subscriber <- eventType:
		default:
			// Если канал заблокирован, пропускаем уведомление
		}
	}
}