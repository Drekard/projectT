package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// ChatIconType определяет тип иконки чата
type ChatIconType int

const (
	// ContactsIcon - Круг с widget.NewIcon(theme.AccountIcon()) в центре. Открывает панель контактов
	ContactsIcon ChatIconType = iota
	// SelfChatIcon - Круг с widget.NewIcon(theme.MailAttachmentIcon()) в центре. Открывает пустой чат с самим собой
	SelfChatIcon
	// PeerChatIcon - Картинка аватарки пользователя 50x50. Открывает чат с пиром
	PeerChatIcon
)

// ChatIcon представляет круглую иконку чата
type ChatIcon struct {
	*fyne.Container
	iconType    ChatIconType
	avatar      *canvas.Rectangle
	statusInd   *StatusIndicator
	unreadBadge *canvas.Circle
	onTap       func()
}

// NewChatIcon создает новую круглую иконку чата
func NewChatIcon(iconType ChatIconType, avatarColor color.Color, unreadCount int, status StatusType, onTap func()) *ChatIcon {
	ci := &ChatIcon{
		iconType: iconType,
		onTap:    onTap,
	}
	ci.Container = ci.createIcon(avatarColor, unreadCount, status)
	return ci
}

// createIcon создает визуальное представление иконки
func (ci *ChatIcon) createIcon(avatarColor color.Color, unreadCount int, status StatusType) *fyne.Container {
	switch ci.iconType {
	case ContactsIcon:
		return ci.createContactsIcon()
	case SelfChatIcon:
		return ci.createSelfChatIcon()
	case PeerChatIcon:
		return ci.createPeerChatIcon(avatarColor, unreadCount, status)
	default:
		return ci.createContactsIcon()
	}
}

// createContactsIcon создает иконку для панели контактов
func (ci *ChatIcon) createContactsIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	ci.avatar = canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 255})
	ci.avatar.CornerRadius = 15
	ci.avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	ci.avatar.StrokeWidth = 1
	ci.avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем иконку аккаунта в центре
	accountIcon := canvas.NewImageFromResource(theme.AccountIcon())
	accountIcon.FillMode = canvas.ImageFillContain
	accountIcon.SetMinSize(fyne.NewSize(30, 30))

	// Центрируем иконку внутри прямоугольника
	iconContainer := container.NewCenter(accountIcon)

	// Создаем индикатор статуса (в правом нижнем углу для контактов - всегда offline)
	ci.statusInd = NewStatusIndicator(StatusOffline)

	// Собираем все элементы в стек
	stack := container.NewStack(ci.avatar, iconContainer, ci.statusInd)

	return container.NewPadded(stack)
}

// createSelfChatIcon создает иконку для чата с самим собой
func (ci *ChatIcon) createSelfChatIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	ci.avatar = canvas.NewRectangle(color.RGBA{R: 100, G: 100, B: 100, A: 255})
	ci.avatar.CornerRadius = 15
	ci.avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	ci.avatar.StrokeWidth = 1
	ci.avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем иконку вложения в центре
	attachmentIcon := canvas.NewImageFromResource(theme.MailAttachmentIcon())
	attachmentIcon.FillMode = canvas.ImageFillContain
	attachmentIcon.SetMinSize(fyne.NewSize(28, 28))

	// Центрируем иконку внутри прямоугольника
	iconContainer := container.NewCenter(attachmentIcon)

	// Для self-chat статус не нужен
	ci.statusInd = NewStatusIndicator(StatusOffline)
	ci.statusInd.Hide()

	// Собираем все элементы в стек
	stack := container.NewStack(ci.avatar, iconContainer)

	return container.NewPadded(stack)
}

// createPeerChatIcon создает иконку для чата с пиром (аватарка 50x50)
func (ci *ChatIcon) createPeerChatIcon(avatarColor color.Color, unreadCount int, status StatusType) *fyne.Container {
	// Создаем аватар (прямоугольник 50x50 с закругленными углами)
	ci.avatar = canvas.NewRectangle(avatarColor)
	ci.avatar.CornerRadius = 15
	ci.avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	ci.avatar.StrokeWidth = 1
	ci.avatar.SetMinSize(fyne.NewSize(50, 50))

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

	return container.NewPadded(stack)
}

// Update обновляет данные иконки
func (ci *ChatIcon) Update(avatarColor color.Color, unreadCount int, status StatusType) {
	// Обновляем только для PeerChatIcon
	if ci.iconType != PeerChatIcon {
		return
	}

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
			ci.Objects[0].(*fyne.Container).Add(ci.unreadBadge)
		} else {
			ci.unreadBadge.Show()
		}
	} else if ci.unreadBadge != nil {
		ci.unreadBadge.Hide()
	}

	ci.Refresh()
}

// SetUnreadCount устанавливает количество непрочитанных сообщений
func (ci *ChatIcon) SetUnreadCount(count int) {
	if ci.iconType != PeerChatIcon {
		return
	}

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
	if ci.iconType != PeerChatIcon {
		return
	}

	if ci.statusInd != nil {
		ci.statusInd.SetStatus(status)
	}
	ci.Refresh()
}

// GetType возвращает тип иконки
func (ci *ChatIcon) GetType() ChatIconType {
	return ci.iconType
}
