package chats

import (
	"fmt"
	"log"
	"time"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/chats/center"

	"github.com/libp2p/go-libp2p/core/peer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createChatArea создает центральную область чата
func (ui *UI) createChatArea() *fyne.Container {
	// По умолчанию показываем пустую панель
	emptyPanel := ui.createEmptyPanel()
	ui.chatArea = container.NewStack(emptyPanel)
	return ui.chatArea
}

// createEmptyPanel создает пустую панель с подсказкой
func (ui *UI) createEmptyPanel() *fyne.Container {
	// Иконка
	icon := widget.NewIcon(theme.MailComposeIcon())

	// Заголовок
	title := widget.NewLabel("Чаты")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Подзаголовок
	subtitle := widget.NewLabel("Выберите чат в левой панели")
	subtitle.Alignment = fyne.TextAlignCenter
	subtitle.TextStyle = fyne.TextStyle{Italic: true}

	content := container.NewVBox(
		icon,
		title,
		subtitle,
	)

	centered := container.NewCenter(content)

	return centered
}

// createChatPanel создает панель чата с сообщениями и полем ввода
func (ui *UI) createChatPanel(contact *models.Contact) fyne.CanvasObject {
	// Получаем локальный PeerID
	localPeerID := ""
	if ui.p2pUI != nil {
		status := ui.p2pUI.GetStatus()
		if status != nil {
			localPeerID = status.PeerID
		}
	} else {
		// Если P2P не инициализирован, используем PeerID из локального профиля
		localProfile, err := queries.GetLocalProfile()
		if err == nil {
			localPeerID = localProfile.PeerID
		}
	}

	// Создаём панель чата с использованием нового компонента
	ui.chatPanel = center.NewChatPanel(
		contact,
		ui.sendMessage,
		ui.closeChat,
		localPeerID,
	)

	return ui.chatPanel.Container()
}

// sendMessage отправляет сообщение
func (ui *UI) sendMessage() {
	if ui.chatPanel == nil {
		return
	}

	text := ui.chatPanel.MessageInput().Text()
	if text == "" {
		return
	}

	// Очищаем поле ввода
	ui.chatPanel.MessageInput().Clear()

	// Проверяем, локальный ли это чат
	if ui.currentContact != nil && ui.currentContact.IsLocalChat() {
		// Локальный чат - сохраняем сообщение только в БД
		localProfile, err := queries.GetLocalProfile()
		if err != nil {
			ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось получить профиль: %v", err))
			ui.chatPanel.MessageInput().SetText(text)
			return
		}

		// Получаем или создаём чат для локального профиля
		chat, err := queries.GetOrCreateChat(localProfile.PeerID, nil)
		if err != nil {
			ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось получить чат: %v", err))
			ui.chatPanel.MessageInput().SetText(text)
			return
		}

		// Сохраняем сообщение в БД
		message := &models.ChatMessage{
			ChatID:      chat.ID,
			FromPeerID:  localProfile.PeerID,
			Content:     text,
			ContentType: "text",
			SentAt:      getTimeNow(),
		}

		if err := queries.CreateChatMessage(message); err != nil {
			ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось сохранить сообщение: %v", err))
			ui.chatPanel.MessageInput().SetText(text)
			return
		}

		// Обновляем время чата
		if err := queries.UpdateChatLastMessage(chat.ID, message.SentAt); err != nil {
			log.Printf("Предупреждение: не удалось обновить время чата: %v", err)
		}

		// Добавляем сообщение в UI
		ui.chatPanel.AddMessage(message, true)
		return
	}

	// Отправляем через P2P сервис если он инициализирован
	if ui.p2pUI != nil && ui.currentContact != nil {
		// Получаем PeerID контакта
		peerID, err := peer.Decode(ui.currentContact.PeerID)
		if err != nil {
			// Показываем сообщение об ошибке
			ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось отправить сообщение: %v", err))
			// Возвращаем текст в поле ввода
			ui.chatPanel.MessageInput().SetText(text)
			return
		}

		// Отправляем сообщение
		err = ui.p2pUI.SendMessage(peerID, text)
		if err != nil {
			ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось отправить сообщение: %v", err))
			ui.chatPanel.MessageInput().SetText(text)
			return
		}

		// Получаем наш локальный PeerID для правильного определения направления
		localPeerID := ""
		status := ui.p2pUI.GetStatus()
		if status != nil {
			localPeerID = status.PeerID
		}

		// Получаем или создаём чат
		chat, err := queries.GetOrCreateChat(ui.currentContact.PeerID, &ui.currentContact.ID)
		if err != nil {
			ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось получить чат: %v", err))
			ui.chatPanel.MessageInput().SetText(text)
			return
		}

		// Добавляем сообщение в UI (исходящее)
		ui.chatPanel.AddMessage(&models.ChatMessage{
			ChatID:      chat.ID,
			FromPeerID:  localPeerID,
			Content:     text,
			ContentType: "text",
			SentAt:      getTimeNow(),
		}, true)
	}
}

// loadMessagesForContact загружает сообщения для контакта
func (ui *UI) loadMessagesForContact(contactID int) {
	if ui.chatPanel == nil {
		return
	}

	// Очищаем текущие сообщения
	ui.chatPanel.Clear()

	// Обычный чат с контактом
	messages, err := queries.GetMessagesForContact(contactID, 100, 0)
	if err != nil {
		log.Printf("Ошибка загрузки сообщений: %v", err)
		return
	}

	// Получаем наш локальный PeerID для определения направления
	localPeerID := ""
	if ui.p2pUI != nil {
		status := ui.p2pUI.GetStatus()
		if status != nil {
			localPeerID = status.PeerID
		}
	}

	// Загружаем сообщения
	ui.chatPanel.LoadMessages(messages, localPeerID)
}

// loadMessagesForChat загружает сообщения для чата по ID
func (ui *UI) loadMessagesForChat(chatID int) {
	if ui.chatPanel == nil {
		return
	}

	// Очищаем текущие сообщения
	ui.chatPanel.Clear()

	// Загружаем сообщения чата
	messages, err := queries.GetMessagesForChat(chatID, 100, 0)
	if err != nil {
		log.Printf("Ошибка загрузки сообщений чата: %v", err)
		return
	}

	// Получаем наш локальный PeerID для определения направления
	localPeerID := ""
	if ui.p2pUI != nil {
		status := ui.p2pUI.GetStatus()
		if status != nil {
			localPeerID = status.PeerID
		}
	} else {
		// Если P2P не инициализирован, используем PeerID из локального профиля
		localProfile, err := queries.GetLocalProfile()
		if err == nil {
			localPeerID = localProfile.PeerID
		}
	}

	// Загружаем сообщения (все сообщения в локальном чате - исходящие)
	ui.chatPanel.LoadMessages(messages, localPeerID)
}

// closeChat закрывает текущий чат
func (ui *UI) closeChat() {
	ui.currentContact = nil
	ui.currentChatID = 0
	ui.chatPanel = nil

	// Показываем пустую панель
	emptyPanel := ui.createEmptyPanel()
	ui.chatArea.Objects = []fyne.CanvasObject{emptyPanel}
	ui.chatArea.Refresh()
}

// showErrorDialog показывает диалог ошибки
func (ui *UI) showErrorDialog(title, message string) {
	if ui.window == nil {
		fmt.Printf("[%s] %s\n", title, message)
		return
	}
	dialog.ShowError(fmt.Errorf("%s", message), ui.window)
}

// getTimeNow возвращает текущее время
func getTimeNow() time.Time {
	return time.Now()
}
