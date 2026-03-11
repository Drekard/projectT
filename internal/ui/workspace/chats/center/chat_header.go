// Package center содержит компоненты центральной панели чата
package center

import (
	"image/color"

	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ChatHeader заголовок чата
type ChatHeader struct {
	container *fyne.Container
	title     *widget.Label
	status    *widget.Label
}

// NewChatHeader создаёт новый заголовок чата
func NewChatHeader(contact *models.Contact) *ChatHeader {
	h := &ChatHeader{}
	h.container = h.createHeader(contact)
	return h
}

// createHeader создаёт заголовок чата
func (h *ChatHeader) createHeader(contact *models.Contact) *fyne.Container {
	// Индикатор статуса
	statusColor := color.RGBA{R: 158, G: 158, B: 158, A: 255}
	statusText := "оффлайн"
	if contact.Status == "online" || contact.Status == "connected" {
		statusColor = color.RGBA{R: 76, G: 175, B: 80, A: 255}
		statusText = "онлайн"
	}

	statusInd := canvas.NewCircle(statusColor)

	// Имя контакта
	h.title = widget.NewLabel(contact.Username)
	h.title.TextStyle = fyne.TextStyle{Bold: true}

	// Статус
	h.status = widget.NewLabel(statusText)
	h.status.TextStyle = fyne.TextStyle{Italic: true}

	infoContainer := container.NewVBox(h.title, h.status)
	headerContent := container.NewHBox(statusInd, infoContainer)

	// Кнопка закрытия чата
	closeBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		// Обработчик устанавливается извне
	})

	header := container.NewBorder(nil, nil, headerContent, closeBtn)

	// Разделитель
	separator := canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255})
	separator.SetMinSize(fyne.NewSize(0, 1))

	return container.NewVBox(header, separator)
}

// Container возвращает контейнер заголовка
func (h *ChatHeader) Container() *fyne.Container {
	return h.container
}

// SetOnClose устанавливает обработчик закрытия
func (h *ChatHeader) SetOnClose(onClose func()) {
	if h.container != nil && len(h.container.Objects) > 0 {
		if vbox, ok := h.container.Objects[0].(*fyne.Container); ok {
			if border, ok := vbox.Objects[0].(*fyne.Container); ok {
				if len(border.Objects) > 0 {
					if closeBtn, ok := border.Objects[0].(*widget.Button); ok {
						closeBtn.OnTapped = onClose
					}
				}
			}
		}
	}
}

// UpdateStatus обновляет статус контакта
func (h *ChatHeader) UpdateStatus(status string) {
	if h.status == nil {
		return
	}

	statusColor := color.RGBA{R: 158, G: 158, B: 158, A: 255}
	statusText := "оффлайн"
	if status == "online" || status == "connected" {
		statusColor = color.RGBA{R: 76, G: 175, B: 80, A: 255}
		statusText = "онлайн"
	}

	h.status.SetText(statusText)

	// Обновляем цвет индикатора
	if h.container != nil && len(h.container.Objects) > 0 {
		if vbox, ok := h.container.Objects[0].(*fyne.Container); ok {
			if border, ok := vbox.Objects[0].(*fyne.Container); ok {
				if len(border.Objects) > 0 {
					if hBox, ok := border.Objects[0].(*fyne.Container); ok {
						if len(hBox.Objects) > 0 {
							if circle, ok := hBox.Objects[0].(*canvas.Circle); ok {
								circle.FillColor = statusColor
								circle.Refresh()
							}
						}
					}
				}
			}
		}
	}

	h.container.Refresh()
}

// UpdateTitle обновляет заголовок
func (h *ChatHeader) UpdateTitle(title string) {
	if h.title != nil {
		h.title.SetText(title)
		h.container.Refresh()
	}
}
