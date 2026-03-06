package widgets

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// MessagePosition определяет положение сообщения
type MessagePosition int

const (
	MessageLeft MessagePosition = iota
	MessageRight
)

// MessageBubble представляет пузырь сообщения в чате
type MessageBubble struct {
	*fyne.Container
	text     string
	time     time.Time
	position MessagePosition
}

// NewMessageBubble создает новый пузырь сообщения
func NewMessageBubble(text string, timestamp time.Time, position MessagePosition) *MessageBubble {
	mb := &MessageBubble{
		text:     text,
		time:     timestamp,
		position: position,
	}
	mb.Container = mb.createBubble()
	return mb
}

// createBubble создает визуальное представление пузыря сообщения
func (mb *MessageBubble) createBubble() *fyne.Container {
	// Определяем цвета в зависимости от положения
	var bubbleColor color.Color
	if mb.position == MessageRight {
		// Исходящее сообщение - синий фон
		bubbleColor = color.RGBA{R: 64, G: 110, B: 215, A: 255}
	} else {
		// Входящее сообщение - серый фон
		bubbleColor = color.RGBA{R: 50, G: 50, B: 50, A: 255}
	}

	// Создаем фон пузыря с закруглением
	bubble := canvas.NewRectangle(bubbleColor)
	bubble.CornerRadius = 12

	// Создаем текст сообщения
	messageText := widget.NewLabel(mb.text)
	messageText.Wrapping = fyne.TextWrapWord

	// Создаем время сообщения
	timeStr := mb.time.Format("15:04")
	timeLabel := widget.NewLabel(timeStr)
	timeLabel.TextStyle = fyne.TextStyle{}

	// Вертикальный контейнер для текста и времени
	content := container.NewVBox(messageText, timeLabel)

	// Контейнер пузыря с отступами
	bubbleContainer := container.NewPadded(content)

	// Оборачиваем в контейнер с фоном
	stack := container.NewStack(bubble, bubbleContainer)

	// Устанавливаем выравнивание в зависимости от положения
	if mb.position == MessageRight {
		return container.NewHBox(canvas.NewRectangle(color.Transparent), stack)
	}
	return container.NewHBox(stack, canvas.NewRectangle(color.Transparent))
}

// Update обновляет содержимое пузыря сообщения
func (mb *MessageBubble) Update(text string, timestamp time.Time, position MessagePosition) {
	mb.text = text
	mb.time = timestamp
	mb.position = position
	mb.Container = mb.createBubble()
	mb.Refresh()
}
