// Package autodial предоставляет тесты для очереди переподключения
package autodial

import (
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReconnectQueue_Creation тестирует создание очереди
func TestReconnectQueue_Creation(t *testing.T) {
	t.Run("создание очереди", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 5)

		require.NotNil(t, queue)
		assert.Equal(t, 30*time.Second, queue.interval)
		assert.Equal(t, 5, queue.maxRetry)
		assert.Equal(t, 0, queue.Length())
	})
}

// TestReconnectQueue_Add тестирует добавление в очередь
func TestReconnectQueue_Add(t *testing.T) {
	t.Run("добавление пира в очередь", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 5)
		peerID := peer.ID("test-peer-1")

		added := queue.Add(peerID)

		assert.True(t, added)
		assert.Equal(t, 1, queue.Length())
	})

	t.Run("дубликат не добавляется", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 5)
		peerID := peer.ID("test-peer-1")

		added1 := queue.Add(peerID)
		added2 := queue.Add(peerID)

		assert.True(t, added1)
		assert.False(t, added2)
		assert.Equal(t, 1, queue.Length())
	})

	t.Run("превышение лимита попыток", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 2)
		peerID := peer.ID("test-peer-1")

		// Имитируем превышение лимита
		queue.mu.Lock()
		queue.attempts[peerID] = 2
		queue.mu.Unlock()

		added := queue.Add(peerID)

		assert.False(t, added)
	})
}

// TestReconnectQueue_GetAttempts тестирует счётчик попыток
func TestReconnectQueue_GetAttempts(t *testing.T) {
	t.Run("получение количества попыток", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 5)
		peerID := peer.ID("test-peer-1")

		// Имитируем попытки
		queue.mu.Lock()
		queue.attempts[peerID] = 3
		queue.mu.Unlock()

		attempts := queue.GetAttempts(peerID)

		assert.Equal(t, 3, attempts)
	})

	t.Run("получение попыток для неизвестного пира", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 5)
		peerID := peer.ID("unknown-peer")

		attempts := queue.GetAttempts(peerID)

		assert.Equal(t, 0, attempts)
	})
}

// TestReconnectQueue_Clear тестирует очистку очереди
func TestReconnectQueue_Clear(t *testing.T) {
	t.Run("очистка очереди", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 5)

		queue.Add(peer.ID("peer-1"))
		queue.Add(peer.ID("peer-2"))
		queue.Add(peer.ID("peer-3"))

		assert.Equal(t, 3, queue.Length())

		queue.Clear()

		assert.Equal(t, 0, queue.Length())
		assert.Equal(t, 0, queue.GetAttempts(peer.ID("peer-1")))
	})
}

// TestReconnectQueue_Length тестирует получение длины очереди
func TestReconnectQueue_Length(t *testing.T) {
	t.Run("длина пустой очереди", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 5)
		assert.Equal(t, 0, queue.Length())
	})

	t.Run("длина очереди с элементами", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 5)

		queue.Add(peer.ID("peer-1"))
		queue.Add(peer.ID("peer-2"))

		assert.Equal(t, 2, queue.Length())
	})
}

// TestReconnectQueue_Concurrent тестирует потокобезопасность
func TestReconnectQueue_Concurrent(t *testing.T) {
	t.Run("одновременное добавление из разных горутин", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 100)
		var wg sync.WaitGroup

		// Запускаем 10 горутин
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				peerID := peer.ID("peer-" + string(rune(id)))
				queue.Add(peerID)
			}(i)
		}

		wg.Wait()

		assert.Equal(t, 10, queue.Length())
	})

	t.Run("одновременное чтение и запись", func(t *testing.T) {
		queue := NewReconnectQueue(30*time.Second, 100)
		var wg sync.WaitGroup

		// Добавляем пиры
		for i := 0; i < 5; i++ {
			queue.Add(peer.ID("peer-" + string(rune(i))))
		}

		// Запускаем горутины на чтение и запись
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_ = queue.Length()
				if id%2 == 0 {
					queue.Add(peer.ID("new-peer-" + string(rune(id))))
				}
			}(i)
		}

		wg.Wait()

		// Не должно быть паники
		assert.GreaterOrEqual(t, queue.Length(), 5)
	})
}

// TestReconnectQueue_StartStop тестирует запуск и остановку
func TestReconnectQueue_StartStop(t *testing.T) {
	t.Run("запуск и остановка без паники", func(t *testing.T) {
		queue := NewReconnectQueue(100*time.Millisecond, 5)

		called := false
		queue.Start(func(p peer.ID) {
			called = true
		})

		// Ждём немного
		time.Sleep(50 * time.Millisecond)

		// Останавливаем
		queue.Stop()

		// Не должно паниковать
		assert.False(t, called) // Очередь пустая, callback не вызван
	})
}
