package chats

import (
	"image/color"

	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// createProfileArea создает правую панель с профилем
func (ui *UI) createProfileArea() *fyne.Container {
	// Заголовок
	header := widget.NewLabel("Профиль")
	header.TextStyle = fyne.TextStyle{Bold: true}

	// Аватар - круг с цветным фоном
	ui.profileAvatar = canvas.NewCircle(color.RGBA{R: 100, G: 100, B: 100, A: 255})

	// Имя
	ui.profileName = widget.NewLabel("Имя собеседника")
	ui.profileName.Alignment = fyne.TextAlignCenter

	// Статус
	ui.profileStatus = widget.NewLabel("онлайн")
	ui.profileStatus.Alignment = fyne.TextAlignCenter

	// Адрес
	ui.profileAddress = widget.NewLabel("address@example.com")
	ui.profileAddress.Alignment = fyne.TextAlignCenter

	// Информация
	info := container.NewVBox(
		ui.profileAvatar,
		layout.NewSpacer(),
		ui.profileName,
		ui.profileStatus,
		ui.profileAddress,
		layout.NewSpacer(),
	)

	// Разделитель
	separator := canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255})

	// Настройки
	settingsLabel := widget.NewLabel("Настройки чата")

	muteButton := widget.NewButton("🔕 Уведомления", func() {
		// TODO: реализовать отключение уведомлений
	})
	muteButton.Importance = widget.LowImportance

	clearButton := widget.NewButton("🗑 Очистить историю", func() {
		// TODO: реализовать очистку истории
	})
	clearButton.Importance = widget.LowImportance

	settings := container.NewVBox(
		separator,
		settingsLabel,
		muteButton,
		clearButton,
	)

	content := container.NewVBox(
		container.NewPadded(info),
		container.NewPadded(settings),
	)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 35, G: 35, B: 35, A: 255})

	return container.NewStack(bg, content)
}

// updateProfile обновляет профиль собеседника
func (ui *UI) updateProfile(contact *models.Contact) {
	if ui.profileAvatar != nil {
		// Пробуем загрузить аватар из файла если указан путь
		if contact.AvatarPath != "" {
			// Загружаем изображение аватара в горутине
			go func() {
				avatarImg, err := fyne.LoadResourceFromPath(contact.AvatarPath)
				if err == nil && avatarImg != nil {
					// Создаём изображение
					img := canvas.NewImageFromResource(avatarImg)
					img.FillMode = canvas.ImageFillContain

					// Обновляем UI в главном потоке
					if ui.window != nil {
						ui.window.Canvas().Refresh(ui.profileAvatar)
					}
				} else {
					// Если не удалось загрузить, используем цвет
					ui.profileAvatar.FillColor = ui.getAvatarColorForContact(contact)
					ui.profileAvatar.Refresh()
				}
			}()
		} else {
			// Используем цветной аватар
			ui.profileAvatar.FillColor = ui.getAvatarColorForContact(contact)
			ui.profileAvatar.Refresh()
		}
	}
	if ui.profileName != nil {
		ui.profileName.SetText(contact.Username)
		ui.profileName.Refresh()
	}
	if ui.profileStatus != nil {
		// Обновляем статус из контакта
		statusText := "оффлайн"
		if contact.Status == "online" || contact.Status == "connected" {
			statusText = "онлайн"
		}
		ui.profileStatus.SetText(statusText)
		ui.profileStatus.Refresh()
	}
	if ui.profileAddress != nil {
		// Показываем сокращённый PeerID
		peerID := contact.PeerID
		if len(peerID) > 16 {
			peerID = peerID[:8] + "..." + peerID[len(peerID)-8:]
		}
		ui.profileAddress.SetText(peerID)
		ui.profileAddress.Refresh()
	}
}

func (ui *UI) getAvatarColorForContact(contact *models.Contact) color.Color {
	if contact == nil || contact.Username == "" {
		return color.RGBA{R: 100, G: 100, B: 100, A: 255}
	}
	hash := 0
	for _, c := range contact.Username {
		hash = int(c) + (hash * 31)
	}
	colors := []color.RGBA{
		{R: 144, G: 238, B: 144, A: 255},
		{R: 173, G: 216, B: 230, A: 255},
		{R: 255, G: 182, B: 193, A: 255},
		{R: 255, G: 218, B: 185, A: 255},
		{R: 221, G: 160, B: 221, A: 255},
		{R: 175, G: 238, B: 238, A: 255},
		{R: 255, G: 255, B: 153, A: 255},
		{R: 255, G: 224, B: 189, A: 255},
	}
	return colors[hash%len(colors)]
}
