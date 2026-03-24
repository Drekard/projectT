// Package dialogs содержит диалоги для чатов
package dialogs

import (
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// ShowSendToContactDialog показывает диалог выбора контакта для отправки элемента
func ShowSendToContactDialog(item *models.Item, onSend func(contact *models.Contact)) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]
	if window == nil {
		return
	}

	// Загружаем все контакты
	contacts, err := queries.GetAllContacts()
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	// Если контактов нет, показываем сообщение
	if len(contacts) == 0 {
		dialog.ShowInformation(
			"Нет контактов",
			"Сначала добавьте контакты для отправки элементов",
			window,
		)
		return
	}

	// Создаём список контактов
	contactsList := container.NewVBox()

	for _, contact := range contacts {
		// Пропускаем локальный чат (отправка самому себе)
		if contact.IsLocalChat() {
			continue
		}

		// Создаём строку контакта
		contactRow := createContactRow(contact, item, onSend, window)
		contactsList.Add(contactRow)
	}

	// Добавляем прокрутку если контактов много
	scrollContainer := container.NewVScroll(contactsList)
	scrollContainer.SetMinSize(fyne.NewSize(300, 200))

	// Создаём контент диалога
	content := container.NewVBox(
		widget.NewLabel("Выберите контакт для отправки:"),
		scrollContainer,
	)

	// Показываем диалог
	dialog.ShowCustom("Отправить элемент", "Отмена", content, window)
}

// createContactRow создаёт строку контакта для диалога отправки
func createContactRow(
	contact *models.Contact,
	item *models.Item,
	onSend func(contact *models.Contact),
	window fyne.Window,
) *fyne.Container {
	// Аватар контакта (если есть)
	var avatar fyne.CanvasObject
	if contact.AvatarPath != "" {
		// Загружаем изображение как ресурс
		avatarRes, err := fyne.LoadResourceFromPath(contact.AvatarPath)
		if err == nil {
			avatarImg := widget.NewIcon(avatarRes)
			avatar = container.NewStack(avatarImg)
		} else {
			// Если ошибка загрузки - используем заглушку
			initial := "?"
			if len(contact.Username) > 0 {
				initial = string(contact.Username[0])
			}
			avatar = widget.NewLabel(initial)
			avatar.Resize(fyne.NewSize(30, 30))
		}
	} else {
		// Заглушка вместо аватара
		initial := "?"
		if len(contact.Username) > 0 {
			initial = string(contact.Username[0])
		}
		avatar = widget.NewLabel(initial)
		avatar.Resize(fyne.NewSize(30, 30))
	}

	// Имя контакта
	nameLabel := widget.NewLabel(contact.Username)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Название элемента
	itemLabel := widget.NewLabel("📤 " + item.Title)
	itemLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка отправки
	sendButton := widget.NewButton("Отправить", func() {
		if onSend != nil {
			onSend(contact)
		}
	})
	sendButton.Importance = widget.HighImportance

	// Собираем строку
	row := container.NewHBox(
		avatar,
		container.NewVBox(nameLabel, itemLabel),
		layout.NewSpacer(),
		sendButton,
	)

	return container.NewPadded(row)
}
