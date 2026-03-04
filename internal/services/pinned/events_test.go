package pinned

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEventManager(t *testing.T) {
	manager := GetEventManager()
	assert.NotNil(t, manager)
}

func TestGetEventManager_Singleton(t *testing.T) {
	manager1 := GetEventManager()
	manager2 := GetEventManager()

	// Должен возвращаться один и тот же экземпляр
	assert.Same(t, manager1, manager2)
}

func TestEventManager_Subscribe(t *testing.T) {
	// Создаём новый менеджер для теста (не глобальный)
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	ch := em.Subscribe()
	assert.NotNil(t, ch)
	assert.Len(t, em.subscribers, 1)
}

func TestEventManager_Subscribe_Multiple(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	ch1 := em.Subscribe()
	ch2 := em.Subscribe()
	ch3 := em.Subscribe()

	assert.NotNil(t, ch1)
	assert.NotNil(t, ch2)
	assert.NotNil(t, ch3)
	assert.Len(t, em.subscribers, 3)
}

func TestEventManager_Notify(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	// Подписываемся
	ch := em.Subscribe()

	// Уведомляем
	em.Notify("test_event")

	// Проверяем что получили уведомление
	select {
	case event := <-ch:
		assert.Equal(t, "test_event", event)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Уведомление не получено")
	}
}

func TestEventManager_Notify_MultipleSubscribers(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	ch1 := em.Subscribe()
	ch2 := em.Subscribe()
	ch3 := em.Subscribe()

	em.Notify("broadcast_event")

	// Все подписчики должны получить уведомление
	events := []string{<-ch1, <-ch2, <-ch3}
	assert.Contains(t, events, "broadcast_event")
}

func TestEventManager_Notify_NoSubscribers(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	// Не должно паниковать
	assert.NotPanics(t, func() {
		em.Notify("no_subscribers")
	})
}

func TestEventManager_Notify_BufferedChannel(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	ch := em.Subscribe()

	// Отправляем несколько уведомлений
	em.Notify("event1")
	em.Notify("event2")
	em.Notify("event3")

	// Проверяем что все уведомления доставлены
	events := []string{<-ch, <-ch, <-ch}
	assert.Contains(t, events, "event1")
	assert.Contains(t, events, "event2")
	assert.Contains(t, events, "event3")
}

func TestEventManager_Notify_ChannelFull(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	// Создаём подписчика с маленьким буфером
	smallCh := make(chan string, 1)
	em.subscribers = append(em.subscribers, smallCh)

	// Заполняем канал
	smallCh <- "first"

	// Следующее уведомление должно быть пропущено (неблокирующая отправка)
	em.Notify("second")

	// Первое уведомление должно остаться в канале
	assert.Equal(t, "first", <-smallCh)
}

func TestEventManager_ConcurrentSubscribe(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	var wg sync.WaitGroup
	done := make(chan bool)

	// Запускаем множество горутин на подписку
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = em.Subscribe()
		}()
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		assert.GreaterOrEqual(t, len(em.subscribers), 1)
	case <-time.After(5 * time.Second):
		t.Fatal("Тест завершен по таймауту")
	}
}

func TestEventManager_ConcurrentNotify(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	// Добавляем подписчиков
	for i := 0; i < 10; i++ {
		em.Subscribe()
	}

	var wg sync.WaitGroup
	done := make(chan bool)

	// Запускаем множество горутин на уведомление
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			em.Notify("concurrent_event")
		}(i)
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		assert.True(t, true)
	case <-time.After(5 * time.Second):
		t.Fatal("Тест завершен по таймауту")
	}
}

func TestEventManager_ConcurrentSubscribeAndNotify(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	var wg sync.WaitGroup
	done := make(chan bool)

	// Запускаем горутин на подписку
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = em.Subscribe()
		}()
	}

	// Запускаем горутин на уведомление
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			em.Notify("mixed_event")
		}()
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		assert.True(t, true)
	case <-time.After(5 * time.Second):
		t.Fatal("Тест завершен по таймауту")
	}
}

func TestEventManager_Notify_DifferentEventTypes(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	ch := em.Subscribe()

	eventTypes := []string{
		"pinned_items_changed",
		"item_pinned",
		"item_unpinned",
		"clear_all",
	}

	for _, eventType := range eventTypes {
		em.Notify(eventType)
	}

	// Проверяем что все события доставлены
	for i := 0; i < len(eventTypes); i++ {
		select {
		case event := <-ch:
			assert.Contains(t, eventTypes, event)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Уведомление %d не получено", i)
		}
	}
}

func TestEventManager_Subscribe_ChannelBuffer(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	ch := em.Subscribe()

	// Проверяем что канал буферизированный (ёмкость 10)
	assert.Equal(t, 10, cap(ch))
}

func TestEventManager_Notify_EmptyEventType(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	ch := em.Subscribe()

	em.Notify("")

	select {
	case event := <-ch:
		assert.Equal(t, "", event)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Уведомление не получено")
	}
}

func TestEventManager_Notify_SpecialCharacters(t *testing.T) {
	em := &EventManager{
		subscribers: make([]chan<- string, 0),
	}

	ch := em.Subscribe()

	specialEvents := []string{
		"event_with_spaces   ",
		"event_with_special_!@#$%",
		"event_with_unicode_привет",
	}

	for _, event := range specialEvents {
		em.Notify(event)
	}

	for i := 0; i < len(specialEvents); i++ {
		select {
		case event := <-ch:
			assert.Contains(t, specialEvents, event)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Уведомление не получено")
		}
	}
}
