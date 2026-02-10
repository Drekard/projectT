package events

// EventHandler функция обработки события
type EventHandler func(Event)

// EventManagerInterface интерфейс для управления событиями
type EventManagerInterface interface {
	Emit(event Event)
	On(eventType EventType, handler EventHandler)
	Off(eventType EventType, handler EventHandler)
}
