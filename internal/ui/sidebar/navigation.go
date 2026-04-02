package sidebar

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NavigationHandler определяет интерфейс для обработки навигации
type NavigationHandler interface {
	OnNavigationChanged(contentType string, param ...interface{})
	NavigateToFolder(folderID int) error
	SearchByTag(tagName string) error
	SetSearchQuery(query string) error
}

// CreateNavigation создает навигационные кнопки
func CreateNavigation(handler NavigationHandler) *fyne.Container {
	var profileButton, savedButton, previewButton, tagsButton, chatsButton, contactsButton, p2pButton *widget.Button

	updateButtonState := func(clickedButton *widget.Button, contentType string) {
		buttons := []*widget.Button{profileButton, savedButton, previewButton, tagsButton, chatsButton, contactsButton, p2pButton}
		for _, btn := range buttons {
			btn.Importance = widget.LowImportance
			btn.Refresh()
		}
		clickedButton.Importance = widget.MediumImportance
		clickedButton.Refresh()

		// Уведомляем обработчик о смене контента
		if handler != nil {
			handler.OnNavigationChanged(contentType)
		}
	}

	profileButton = createCustomNavButton("Профиль", theme.AccountIcon(), func() {
		updateButtonState(profileButton, "profile")
	})

	savedButton = createCustomNavButton("Сохраненное", theme.HomeIcon(), func() {
		updateButtonState(savedButton, "saved")
	})

	previewButton = createCustomNavButton("Загруженное", theme.DownloadIcon(), func() {
		updateButtonState(previewButton, "preview")
	})

	tagsButton = createCustomNavButton("Мои теги", theme.SettingsIcon(), func() {
		updateButtonState(tagsButton, "tags")
	})

	chatsButton = createCustomNavButton("Чаты", theme.MailComposeIcon(), func() {
		updateButtonState(chatsButton, "chats")
	})

	contactsButton = createCustomNavButton("Контакты", theme.AccountIcon(), func() {
		updateButtonState(contactsButton, "contacts")
	})

	p2pButton = createCustomNavButton("P2P", theme.ComputerIcon(), func() {
		updateButtonState(p2pButton, "p2p")
	})

	// Устанавливаем начальное состояние
	updateButtonState(savedButton, "saved")

	separator := widget.NewSeparator()

	return container.NewVBox(
		profileButton,
		savedButton,
		previewButton,
		tagsButton,
		chatsButton,
		contactsButton,
		p2pButton,
		separator,
	)
}

func createCustomNavButton(label string, icon fyne.Resource, onClick func()) *widget.Button {
	button := widget.NewButtonWithIcon("", icon, onClick)
	button.Alignment = widget.ButtonAlignLeading
	button.Importance = widget.LowImportance
	button.SetText(label)
	return button
}
