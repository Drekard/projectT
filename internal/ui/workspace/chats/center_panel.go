package chats

import (
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/chats/center"

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
	// Создаём панель чата с использованием нового компонента
	ui.chatPanel = center.NewChatPanel(
		contact,
		ui.sendMessage,
		ui.closeChat,
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

		// Добавляем сообщение в UI (исходящее)
		ui.chatPanel.AddMessage(&models.ChatMessage{
			ContactID:   ui.currentContact.ID,
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
