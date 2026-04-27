package metrics

import (
	"sync"
)

var (
	globalManager *Manager
	once          sync.Once
)

// Init инициализирует глобальный менеджер метрик
func Init() *Manager {
	once.Do(func() {
		globalManager = NewManager()
	})
	return globalManager
}

// Get возвращает глобальный менеджер метрик
func Get() *Manager {
	if globalManager == nil {
		return Init()
	}
	return globalManager
}

// IsInitialized проверяет, инициализирован ли менеджер
func IsInitialized() bool {
	return globalManager != nil
}
