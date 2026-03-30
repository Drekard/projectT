// Package dialogs содержит диалоги для чатов
package dialogs

import (
	"log"

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

	log.Printf("[SendDialog] 📤 Открытие диалога отправки элемента: ID=%d, UUID=%s, title=%q, type=%s",
		item.ID, item.ElementUUID, item.Title, item.Type)

	// Загружаем все чаты с последними сообщениями
	chats, err := queries.GetChatsWithLastMessages()
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	log.Printf("[SendDialog] 📋 Загружено %d чатов для выбора", len(chats))

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
		widget.NewLabel("Выберите чат для отправки:"),
		scrollContainer,
	)

	// Показываем диалог
	dialog.ShowCustom("Отправить элемент", "Отмена", content, window)
}

// createChatRow создаёт строку чата для диалога отправки
func createChatRow(
	chat *models.ChatWithLastMessage,
	item *models.Item,
	onSend func(chat *models.ChatWithLastMessage),
	window fyne.Window,
) *fyne.Container {
	log.Printf("[SendDialog] 🔍 Создание строки чата: peer_id=%s, username=%q, chat_id=%d, avatar_path=%q",
		chat.PeerID[:min(10, len(chat.PeerID))], chat.Username, chat.ID, chat.AvatarPath)

	// Аватар чата (если есть)
	var avatar fyne.CanvasObject
	if chat.AvatarPath != "" {
		// Загружаем изображение как ресурс
		avatarRes, err := fyne.LoadResourceFromPath(chat.AvatarPath)
		if err == nil {
			log.Printf("[SendDialog] ✅ Аватар загружен из %q (%d байт)", chat.AvatarPath, len(avatarRes.Content()))
			avatarImg := widget.NewIcon(avatarRes)
			avatar = container.NewStack(avatarImg)
		} else {
			log.Printf("[SendDialog] ❌ Ошибка загрузки аватара из %q: %v", chat.AvatarPath, err)
			// Если ошибка загрузки - используем заглушку
			initial := "?"
			if len(chat.Username) > 0 {
				initial = string(chat.Username[0])
			}
			avatar = widget.NewLabel(initial)
			avatar.Resize(fyne.NewSize(30, 30))
		}
	} else {
		log.Printf("[SendDialog] ℹ️ AvatarPath пустой, используем заглушку")
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
	sendButton := widget.NewButton("Отправить", func() {
		log.Printf("[SendDialog] 🚀 Нажата кнопка отправки в чат: peer_id=%s, username=%q, element_uuid=%s",
			chat.PeerID[:min(10, len(chat.PeerID))], chat.Username, item.ElementUUID)
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
		log.Printf("[SendDialog] ❌ Ошибка получения локального профиля: %v", err)
		// Если ошибка - используем заглушку
		localProfile = &models.Profile{
			Username:   "Локальный чат",
			AvatarPath: "",
		}
	} else {
		log.Printf("[SendDialog] 📋 Локальный профиль загружен: username=%q, avatar_path=%q",
			localProfile.Username, localProfile.AvatarPath)
	}

	// Аватар локального чата
	var avatar fyne.CanvasObject
	if localProfile.AvatarPath != "" {
		avatarRes, err := fyne.LoadResourceFromPath(localProfile.AvatarPath)
		if err == nil {
			log.Printf("[SendDialog] ✅ Локальный аватар загружен из %q (%d байт)", localProfile.AvatarPath, len(avatarRes.Content()))
			avatarImg := widget.NewIcon(avatarRes)
			avatar = container.NewStack(avatarImg)
		} else {
			log.Printf("[SendDialog] ❌ Ошибка загрузки локального аватара из %q: %v", localProfile.AvatarPath, err)
			initial := "?"
			if len(localProfile.Username) > 0 {
				initial = string(localProfile.Username[0])
			}
			avatar = widget.NewLabel(initial)
			avatar.Resize(fyne.NewSize(30, 30))
		}
	} else {
		log.Printf("[SendDialog] ℹ️ Локальный AvatarPath пустой, используем заглушку")
		initial := "?"
		if len(localProfile.Username) > 0 {
			initial = string(localProfile.Username[0])
		}
		avatar = widget.NewLabel(initial)
		avatar.Resize(fyne.NewSize(30, 30))
	}

	// Имя локального чата
	nameLabel := widget.NewLabel("📝 Локальный чат")
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Название элемента
	itemLabel := widget.NewLabel("📤 " + item.Title)
	itemLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка отправки
	sendButton := widget.NewButton("Отправить", func() {
		log.Printf("[SendDialog] 🚀 Нажата кнопка отправки в локальный чат: element_uuid=%s", item.ElementUUID)
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

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
