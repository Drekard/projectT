package widgets

import (
	"image/color"

	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ContactItem представляет элемент контакта в списке
type ContactItem struct {
	*fyne.Container
	contact     *models.Contact
	unreadCount int
	status      StatusType
	onTap       func()
	avatar      *canvas.Circle
	statusInd   *StatusIndicator
	unreadBadge *canvas.Circle
	nameLabel   *widget.Label
}

// NewContactItem создает новый элемент контакта
func NewContactItem(contact *models.Contact, unreadCount int, status StatusType, onTap func()) *ContactItem {
	ci := &ContactItem{
		contact:     contact,
		unreadCount: unreadCount,
		status:      status,
		onTap:       onTap,
	}
	ci.Container = ci.createItem()
	return ci
}

// createItem создает визуальное представление элемента контакта
func (ci *ContactItem) createItem() *fyne.Container {
	// Создаем аватар (круглая иконка)
	avatarColor := ci.getColorForContact()
	ci.avatar = canvas.NewCircle(avatarColor)

	// Создаем индикатор статуса (в правом верхнем углу)
	ci.statusInd = NewStatusIndicator(ci.status)

	// Создаем бейдж непрочитанных (в правом нижнем углу)
	if ci.unreadCount > 0 {
		badgeColor := color.RGBA{R: 244, G: 67, B: 54, A: 255}
		ci.unreadBadge = canvas.NewCircle(badgeColor)
	}

	// Создаем имя контакта
	name := "Неизвестный"
	if ci.contact != nil && ci.contact.Username != "" {
		name = ci.contact.Username
	}
	ci.nameLabel = widget.NewLabel(name)
	ci.nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Вертикальная компоновка: аватар с индикаторами + имя
	avatarWithStatus := container.NewStack(ci.avatar, ci.statusInd)
	if ci.unreadBadge != nil {
		avatarWithStatus = container.NewStack(ci.avatar, ci.statusInd, ci.unreadBadge)
	}

	content := container.NewVBox(avatarWithStatus, ci.nameLabel)

	// Оборачиваем в контейнер с отступами
	wrapper := container.NewPadded(content)

	return wrapper
}

// getColorForContact генерирует цвет на основе username
func (ci *ContactItem) getColorForContact() color.Color {
	if ci.contact == nil || ci.contact.Username == "" {
		return color.RGBA{R: 100, G: 100, B: 100, A: 255}
	}
	// Простая хеш-функция для генерации цвета
	hash := 0
	for _, c := range ci.contact.Username {
		hash = int(c) + (hash * 31)
	}
	// Генерируем приятные приглушенные цвета
	colors := []color.RGBA{
		{R: 144, G: 238, B: 144, A: 255}, // Светло-зеленый
		{R: 173, G: 216, B: 230, A: 255}, // Светло-синий
		{R: 255, G: 182, B: 193, A: 255}, // Светло-розовый
		{R: 255, G: 218, B: 185, A: 255}, // Светло-оранжевый
		{R: 221, G: 160, B: 221, A: 255}, // Светло-фиолетовый
		{R: 175, G: 238, B: 238, A: 255}, // Бирюзовый
		{R: 255, G: 255, B: 153, A: 255}, // Светло-желтый
		{R: 255, G: 224, B: 189, A: 255}, // Персиковый
	}
	return colors[hash%len(colors)]
}

// Update обновляет данные контакта
func (ci *ContactItem) Update(contact *models.Contact, unreadCount int, status StatusType) {
	ci.contact = contact
	ci.unreadCount = unreadCount
	ci.status = status

	// Обновляем имя
	name := "Неизвестный"
	if contact != nil && contact.Username != "" {
		name = contact.Username
	}
	ci.nameLabel.SetText(name)

	// Обновляем статус
	ci.statusInd.SetStatus(status)

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
func (ci *ContactItem) SetUnreadCount(count int) {
	ci.unreadCount = count
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

// SetStatus устанавливает статус контакта
func (ci *ContactItem) SetStatus(status StatusType) {
	ci.status = status
	ci.statusInd.SetStatus(status)
	ci.Refresh()
}
