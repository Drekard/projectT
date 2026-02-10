package utils

import (
	"sync"
	"time"
)

// Debouncer реализует функциональность дебаунсинга
type Debouncer struct {
	mu       sync.Mutex
	timer    *time.Timer
	delay    time.Duration
	callback func()
}

// NewDebouncer создает новый дебаунсер
func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{
		delay: delay,
	}
}

// Call вызывает функцию с задержкой
func (d *Debouncer) Call(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.callback = fn
	d.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		if d.callback != nil {
			d.callback()
		}
		d.timer = nil
		d.mu.Unlock()
	})
}

// Throttler реализует функциональность троттлинга
type Throttler struct {
	mu       sync.Mutex
	lastCall time.Time
	interval time.Duration
	callback func()
	running  bool
}

// NewThrottler создает новый троттлер
func NewThrottler(interval time.Duration) *Throttler {
	return &Throttler{
		interval: interval,
	}
}

// Call вызывает функцию с ограничением частоты
func (t *Throttler) Call(fn func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if now.Sub(t.lastCall) >= t.interval {
		// Вызов разрешен
		t.lastCall = now
		fn()
	} else if !t.running {
		// Запуск таймера для отложенного вызова
		t.running = true
		time.Sleep(t.interval - now.Sub(t.lastCall))

		// Проверяем, не был ли сделан прямой вызов за это время
		t.mu.Unlock()
		fn()
		t.mu.Lock()
		t.lastCall = time.Now()
		t.running = false
	}
}

// SafeThrottler реализует более безопасную версию троттлинга
type SafeThrottler struct {
	mu       sync.Mutex
	lastCall time.Time
	interval time.Duration
	pending  bool
	callback func()
}

// NewSafeThrottler создает новый безопасный троттлер
func NewSafeThrottler(interval time.Duration) *SafeThrottler {
	return &SafeThrottler{
		interval: interval,
	}
}

// Call вызывает функцию с ограничением частоты
func (st *SafeThrottler) Call(fn func()) {
	st.mu.Lock()
	defer st.mu.Unlock()

	now := time.Now()

	if now.Sub(st.lastCall) >= st.interval {
		// Вызов разрешен, выполним немедленно
		st.lastCall = now
		st.pending = false

		// Выполняем в отдельной горутине, чтобы не блокировать
		go fn()
	} else if !st.pending {
		// Установим флаг ожидания и запланируем вызов
		st.pending = true
		timeToWait := st.interval - now.Sub(st.lastCall)

		go func() {
			time.Sleep(timeToWait)

			st.mu.Lock()
			if st.pending {
				st.lastCall = time.Now()
				st.pending = false
				callback := st.callback
				st.callback = nil
				st.mu.Unlock()

				if callback != nil {
					callback()
				}
			} else {
				st.mu.Unlock()
			}
		}()
	}

	// Обновляем callback для возможного последнего вызова
	st.callback = fn
}
