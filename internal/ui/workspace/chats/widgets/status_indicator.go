package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// StatusType определяет тип статуса
type StatusType int

const (
	StatusOnline StatusType = iota
	StatusOffline
	StatusAway
)

// StatusIndicator отображает индикатор статуса
type StatusIndicator struct {
	*fyne.Container
	status StatusType
}

// NewStatusIndicator создает новый индикатор статуса
func NewStatusIndicator(status StatusType) *StatusIndicator {
	si := &StatusIndicator{
		status: status,
	}
	si.Container = si.createIndicator()
	return si
}

// createIndicator создает визуальное представление индикатора
func (si *StatusIndicator) createIndicator() *fyne.Container {
	circle := canvas.NewCircle(si.getStatusColor())

	// Невидимый прямоугольник для задания минимального размера
	placeholder := canvas.NewRectangle(color.Transparent)
	placeholder.SetMinSize(fyne.NewSize(12, 12))

	return container.NewStack(placeholder, circle)
}

// getStatusColor возвращает цвет статуса
func (si *StatusIndicator) getStatusColor() color.Color {
	switch si.status {
	case StatusOnline:
		return color.RGBA{R: 76, G: 175, B: 80, A: 255} // Зеленый
	case StatusOffline:
		return color.RGBA{R: 158, G: 158, B: 158, A: 255} // Серый
	case StatusAway:
		return color.RGBA{R: 255, G: 193, B: 7, A: 255} // Желтый
	default:
		return color.RGBA{R: 158, G: 158, B: 158, A: 255}
	}
}

// SetStatus устанавливает новый статус и обновляет индикатор
func (si *StatusIndicator) SetStatus(status StatusType) {
	si.status = status
	if len(si.Objects) > 0 {
		if circle, ok := si.Objects[0].(*canvas.Circle); ok {
			circle.FillColor = si.getStatusColor()
			circle.Refresh()
		}
	}
}
