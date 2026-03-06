package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// ChatIcon представляет круглую иконку чата
type ChatIcon struct {
	*fyne.Container
	avatar      *canvas.Circle
	statusInd   *StatusIndicator
	unreadBadge *canvas.Circle
	onTap       func()
}

// NewChatIcon создает новую круглую иконку чата
func NewChatIcon(avatarColor color.Color, unreadCount int, status StatusType, onTap func()) *ChatIcon {
	ci := &ChatIcon{
		onTap: onTap,
	}
	ci.Container = ci.createIcon(avatarColor, unreadCount, status)
	return ci
}

// createIcon создает визуальное представление иконки
func (ci *ChatIcon) createIcon(avatarColor color.Color, unreadCount int, status StatusType) *fyne.Container {
	// Создаем аватар (круглая иконка)
	ci.avatar = canvas.NewCircle(avatarColor)
	ci.avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	ci.avatar.StrokeWidth = 1

	// Создаем индикатор статуса (в правом верхнем углу)
	ci.statusInd = NewStatusIndicator(status)

	// Создаем бейдж непрочитанных (в правом нижнем углу)
	if unreadCount > 0 {
		badgeColor := color.RGBA{R: 244, G: 67, B: 54, A: 255}
		ci.unreadBadge = canvas.NewCircle(badgeColor)
	}

	// Собираем все элементы в стек
	var stack *fyne.Container
	if ci.unreadBadge != nil {
		stack = container.NewStack(ci.avatar, ci.statusInd, ci.unreadBadge)
	} else {
		stack = container.NewStack(ci.avatar, ci.statusInd)
	}

	// Создаем невидимый прямоугольник для установки минимального размера
	placeholder := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 0})
	placeholder.SetMinSize(fyne.NewSize(60, 60))

	// Оборачиваем в контейнер с фиксированным размером
	wrapper := container.NewStack(placeholder, stack)

	return container.NewPadded(wrapper)
}

// Update обновляет данные иконки
func (ci *ChatIcon) Update(avatarColor color.Color, unreadCount int, status StatusType) {
	if ci.avatar != nil {
		ci.avatar.FillColor = avatarColor
		ci.avatar.Refresh()
	}

	// Обновляем статус
	if ci.statusInd != nil {
		ci.statusInd.SetStatus(status)
	}

	// Обновляем бейдж непрочитанных
	if unreadCount > 0 {
		if ci.unreadBadge == nil {
			badgeColor := color.RGBA{R: 244, G: 67, B: 54, A: 255}
			ci.unreadBadge = canvas.NewCircle(badgeColor)
		}
		ci.unreadBadge.Show()
	} else if ci.unreadBadge != nil {
		ci.unreadBadge.Hide()
	}

	ci.Refresh()
}

// SetUnreadCount устанавливает количество непрочитанных сообщений
func (ci *ChatIcon) SetUnreadCount(count int) {
	if count > 0 {
		if ci.unreadBadge == nil {
			badgeColor := color.RGBA{R: 244, G: 67, B: 54, A: 255}
			ci.unreadBadge = canvas.NewCircle(badgeColor)
		}
		ci.unreadBadge.Show()
	} else if ci.unreadBadge != nil {
		ci.unreadBadge.Hide()
	}
	ci.Refresh()
}

// SetStatus устанавливает статус чата
func (ci *ChatIcon) SetStatus(status StatusType) {
	if ci.statusInd != nil {
		ci.statusInd.SetStatus(status)
	}
	ci.Refresh()
}
