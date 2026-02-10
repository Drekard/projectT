package events

// EventType тип события
type EventType string

const (
	// ItemAdded событие добавления элемента
	ItemAdded EventType = "item_added"

	// ItemRemoved событие удаления элемента
	ItemRemoved EventType = "item_removed"

	// LayoutChanged событие изменения раскладки
	LayoutChanged EventType = "layout_changed"

	// SizeChanged событие изменения размера
	SizeChanged EventType = "size_changed"

	// NavigationRequested событие запроса навигации
	NavigationRequested EventType = "navigation_requested"
)

// Event структура события
type Event struct {
	Type    EventType
	Payload interface{}
	Source  interface{}
}
