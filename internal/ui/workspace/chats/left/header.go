// Package left содержит компоненты левой панели чатов
package left

import (
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createLeftPanelHeader создает заголовок левой панели с иконками
func (p *Panel) createLeftPanelHeader() *fyne.Container {
	// Иконка чата с собой
	favoriteIcon := p.createFavoriteIcon()

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		p.chatsUI.RefreshContactsList()
	})

	// Вертикальная компоновка иконок
	icons := container.NewVBox(
		favoriteIcon,
	)

	return container.NewBorder(nil, refreshBtn, nil, nil,
		container.NewPadded(icons),
	)
}

// createFavoriteIcon создает иконку для чата Избранного (локальный чат с самим собой)
func (p *Panel) createFavoriteIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	avatar := canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем кнопку с иконкой поверх графики
	btn := widget.NewButtonWithIcon("", theme.ContentRedoIcon(), func() {
		log.Printf("[Chat] 🗨️ Нажата кнопка 'Избранное' - открытие локального чата")

		// Открываем локальный чат через специальный метод
		p.chatsUI.OpenLocalChat()
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	// Оборачиваем в контейнер
	return container.NewStack(avatar, btnContainer)
}
