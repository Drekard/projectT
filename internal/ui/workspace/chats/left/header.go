// Package left содержит компоненты левой панели чатов
package left

import (
	"image/color"
	"log"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createLeftPanelHeader создает заголовок левой панели с иконками
func (p *Panel) createLeftPanelHeader() *fyne.Container {
	// Иконка контактов
	contactsIcon := p.createContactsIcon()

	// Иконка чата с собой
	favoriteIcon := p.createFavoriteIcon()

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		p.chatsUI.RefreshContactsList()
	})

	// Вертикальная компоновка иконок
	icons := container.NewVBox(
		contactsIcon,
		favoriteIcon,
	)

	return container.NewBorder(nil, refreshBtn, nil, nil,
		container.NewPadded(icons),
	)
}

// createContactsIcon создает иконку для панели контактов
func (p *Panel) createContactsIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	avatar := canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем кнопку с иконкой поверх графики
	btn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		p.chatsUI.ShowContactsPanel()
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	// Оборачиваем в контейнер
	return container.NewStack(avatar, btnContainer)
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
		// Загружаем локальный профиль
		localProfile, err := queries.GetLocalProfile()
		if err != nil {
			log.Printf("Ошибка загрузки локального профиля: %v", err)
			return
		}

		// Получаем или создаём локальный чат (с contact_id = NULL)
		_, err = queries.GetOrCreateLocalChat(localProfile.PeerID)
		if err != nil {
			log.Printf("Ошибка получения локального чата: %v", err)
			return
		}

		// Создаём специальный контакт для локального чата (виртуальный, не из БД)
		localContact := models.NewLocalContact(
			localProfile.Username,
			localProfile.Title,
			localProfile.AvatarPath,
		)

		// Открываем чат через публичный метод UI
		p.chatsUI.OpenPeerChat(localContact.PeerID, localContact.Username)
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	// Оборачиваем в контейнер
	return container.NewStack(avatar, btnContainer)
}
