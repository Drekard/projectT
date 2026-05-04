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

// ShowSendToChatDialog показывает диалог выбора чата для отправки элемента
func ShowSendToChatDialog(item *models.Item, onSend func(chat *models.ChatWithLastMessage)) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]
	if window == nil {
		return
	}

	// Загружаем все чаты с последними сообщениями
	chats, err := queries.GetChatsWithLastMessages()
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	// Создаём список чатов
	chatsList := container.NewVBox()

	// Добавляем локальный чат
	localChatRow := createLocalChatRow(item, onSend, window)
	chatsList.Add(localChatRow)

	// Добавляем остальные чаты
	for _, chat := range chats {
		// Пропускаем локальный чат (он уже добавлен отдельно)
		if chat.PeerID == models.LocalChatPeerID {
			continue
		}

		// Создаём строку чата
		chatRow := createChatRow(chat, item, onSend, window)
		chatsList.Add(chatRow)
	}

	// Добавляем прокрутку если чатов много
	scrollContainer := container.NewVScroll(chatsList)
	scrollContainer.SetMinSize(fyne.NewSize(300, 200))

	// Создаём контент диалога
	content := container.NewVBox(
		widget.NewLabel("Select a chat to send:"),
		scrollContainer,
	)

	// Показываем диалог
	dialog.ShowCustom("Send item", "Cancel", content, window)
}

// createChatRow создаёт строку чата для диалога отправки
func createChatRow(
	chat *models.ChatWithLastMessage,
	item *models.Item,
	onSend func(chat *models.ChatWithLastMessage),
	window fyne.Window,
) *fyne.Container {
	// Аватар чата (если есть)
	var avatar fyne.CanvasObject
	if chat.AvatarPath != "" {
		// Загружаем изображение как ресурс
		avatarRes, err := fyne.LoadResourceFromPath(chat.AvatarPath)
		if err == nil {
			avatarImg := widget.NewIcon(avatarRes)
			avatar = container.NewStack(avatarImg)
		} else {
			// Если ошибка загрузки - используем заглушку
			initial := "?"
			if len(chat.Username) > 0 {
				initial = string(chat.Username[0])
			}
			avatar = widget.NewLabel(initial)
			avatar.Resize(fyne.NewSize(30, 30))
		}
	} else {
		// Заглушка вместо аватара
		initial := "?"
		if len(chat.Username) > 0 {
			initial = string(chat.Username[0])
		}
		avatar = widget.NewLabel(initial)
		avatar.Resize(fyne.NewSize(30, 30))
	}

	// Имя чата
	nameLabel := widget.NewLabel(chat.Username)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Название элемента
	itemLabel := widget.NewLabel("📤 " + item.Title)
	itemLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка отправки
	sendButton := widget.NewButton("Send", func() {
		if onSend != nil {
			onSend(chat)
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

// createLocalChatRow создаёт строку локального чата для диалога отправки
func createLocalChatRow(
	item *models.Item,
	onSend func(chat *models.ChatWithLastMessage),
	window fyne.Window,
) *fyne.Container {
	// Получаем локальный профиль для аватара и имени
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		// Если ошибка - используем заглушку
		localProfile = &models.Profile{
			Username:   "Local chat",
			AvatarPath: "",
		}
	}

	// Аватар локального чата
	var avatar fyne.CanvasObject
	if localProfile.AvatarPath != "" {
		avatarRes, err := fyne.LoadResourceFromPath(localProfile.AvatarPath)
		if err == nil {
			avatarImg := widget.NewIcon(avatarRes)
			avatar = container.NewStack(avatarImg)
		} else {
			initial := "?"
			if len(localProfile.Username) > 0 {
				initial = string(localProfile.Username[0])
			}
			avatar = widget.NewLabel(initial)
			avatar.Resize(fyne.NewSize(30, 30))
		}
	} else {
		initial := "?"
		if len(localProfile.Username) > 0 {
			initial = string(localProfile.Username[0])
		}
		avatar = widget.NewLabel(initial)
		avatar.Resize(fyne.NewSize(30, 30))
	}

	// Имя локального чата
	nameLabel := widget.NewLabel("📝 Local chat")
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Название элемента
	itemLabel := widget.NewLabel("📤 " + item.Title)
	itemLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка отправки
	sendButton := widget.NewButton("Send", func() {
		if onSend != nil {
			// Создаём заглушку локального чата для отправки
			localChat := &models.ChatWithLastMessage{
				PeerID:     models.LocalChatPeerID,
				Username:   localProfile.Username,
				AvatarPath: localProfile.AvatarPath,
			}
			onSend(localChat)
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
