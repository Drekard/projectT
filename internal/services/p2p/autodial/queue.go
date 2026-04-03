// Package autodial предоставляет сервисы для автоматического подключения к пирам
package autodial

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ReconnectQueue очередь переподключения с таймерами
type ReconnectQueue struct {
	mu       sync.RWMutex
	queue    []peer.ID
	attempts map[peer.ID]int
	interval time.Duration
	maxRetry int
	onRetry  func(peer.ID) // Callback для попытки переподключения
	stopChan chan struct{}
}

// NewReconnectQueue создаёт очередь переподключения
func NewReconnectQueue(interval time.Duration, maxRetry int) *ReconnectQueue {
	return &ReconnectQueue{
		queue:    make([]peer.ID, 0),
		attempts: make(map[peer.ID]int),
		interval: interval,
		maxRetry: maxRetry,
		stopChan: make(chan struct{}),
	}
}

// Start запускает обработку очереди
func (q *ReconnectQueue) Start(onRetry func(peer.ID)) {
	q.onRetry = onRetry
	go q.run()
}

// Stop останавливает обработку очереди
func (q *ReconnectQueue) Stop() {
	close(q.stopChan)
}

// Add добавляет пира в очередь
func (q *ReconnectQueue) Add(peerID peer.ID) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Проверяем, есть ли уже в очереди
	for _, p := range q.queue {
		if p == peerID {
			return false
		}
	}

	// Проверяем лимит попыток
	if q.attempts[peerID] >= q.maxRetry {
		return false
	}

	q.queue = append(q.queue, peerID)
	return true
}

// run основной цикл обработки очереди
func (q *ReconnectQueue) run() {
	ticker := time.NewTicker(q.interval)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopChan:
			return
		case <-ticker.C:
			q.processNext()
		}
	}
}

// processNext обрабатывает следующий элемент очереди
func (q *ReconnectQueue) processNext() {
	q.mu.Lock()
	if len(q.queue) == 0 {
		q.mu.Unlock()
		return
	}

	peerID := q.queue[0]
	q.queue = q.queue[1:]
	q.attempts[peerID]++
	attempts := q.attempts[peerID]
	q.mu.Unlock()

	if attempts > q.maxRetry {
		return
	}

	if q.onRetry != nil {
		q.onRetry(peerID)
	}
}

// GetAttempts возвращает количество попыток для пира
func (q *ReconnectQueue) GetAttempts(peerID peer.ID) int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.attempts[peerID]
}

// Length возвращает длину очереди
func (q *ReconnectQueue) Length() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.queue)
}

// Clear очищает очередь
func (q *ReconnectQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue = make([]peer.ID, 0)
	q.attempts = make(map[peer.ID]int)
}
