package services

import (
	"sync"
)

// FavoritesEventManager управляет событиями, связанными с избранными элементами
type FavoritesEventManager struct {
	mu          sync.RWMutex
	subscribers []chan<- string
}

var instance *FavoritesEventManager
var once sync.Once

// GetFavoritesEventManager возвращает глобальный экземпляр менеджера событий
func GetFavoritesEventManager() *FavoritesEventManager {
	once.Do(func() {
		instance = &FavoritesEventManager{
			subscribers: make([]chan<- string, 0),
		}
	})
	return instance
}

// Subscribe регистрирует нового подписчика на события избранного
func (fm *FavoritesEventManager) Subscribe() <-chan string {
	ch := make(chan string, 10) // Буферизированный канал для предотвращения блокировки
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.subscribers = append(fm.subscribers, ch)
	return ch
}

// Notify отправляет уведомление всем подписчикам
func (fm *FavoritesEventManager) Notify(eventType string) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	for _, subscriber := range fm.subscribers {
		select {
		case subscriber <- eventType:
		default:
			// Если канал заблокирован, пропускаем уведомление
		}
	}
}
