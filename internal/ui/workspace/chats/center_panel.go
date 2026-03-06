package chats

import (
	"image/color"
	"time"

	"projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/chats/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// createChatArea создает центральную область чата
func (ui *UI) createChatArea() *fyne.Container {
	// Заголовок чата
	header := ui.createChatHeader()

	// Область сообщений
	ui.messagesList = container.NewVBox()
	ui.messageScroll = container.NewScroll(ui.messagesList)

	// Пустое сообщение по умолчанию
	emptyLabel := widget.NewLabel("Выберите чат для начала общения")
	emptyLabel.Alignment = fyne.TextAlignCenter
	ui.messagesList.Add(emptyLabel)

	// Поле ввода сообщения
	ui.messageEntry = widget.NewEntry()
	ui.messageEntry.SetPlaceHolder("Введите сообщение...")
	ui.messageEntry.MultiLine = false
	ui.messageEntry.OnSubmitted = func(text string) {
		ui.sendMessage()
	}

	// Кнопка отправки
	ui.sendButton = widget.NewButton("➤", func() {
		ui.sendMessage()
	})
	ui.sendButton.Importance = widget.HighImportance

	// Панель ввода
	inputPanel := container.NewHBox(ui.messageEntry, ui.sendButton)

	// Вертикальная компоновка
	content := container.NewBorder(header, inputPanel, nil, nil, ui.messageScroll)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 40, G: 40, B: 40, A: 255})

	return container.NewStack(bg, content)
}

// createChatHeader создает заголовок чата
func (ui *UI) createChatHeader() *fyne.Container {
	ui.chatTitle = widget.NewLabel("Выберите чат")
	ui.chatTitle.TextStyle = fyne.TextStyle{Bold: true}

	ui.chatStatus = widget.NewLabel("")

	content := container.NewVBox(ui.chatTitle, ui.chatStatus)

	return container.NewPadded(content)
}

// selectChat выбирает чат
func (ui *UI) selectChat(contact *models.Contact) {
	ui.currentChatID = 0 // TODO: использовать реальный ID
	ui.updateChatHeader(contact.Username, contact.PeerID)
	ui.loadMessages(contact)
	ui.updateProfile(contact)
}

// updateChatHeader обновляет заголовок чата
func (ui *UI) updateChatHeader(name, address string) {
	if ui.chatTitle != nil {
		ui.chatTitle.SetText(name)
	}
	if ui.chatStatus != nil {
		ui.chatStatus.SetText("онлайн • " + address)
	}
	if ui.chatArea != nil {
		ui.chatArea.Refresh()
	}
}

// loadMessages загружает сообщения чата
func (ui *UI) loadMessages(contact *models.Contact) {
	ui.messagesList.Objects = nil

	// Тестовые сообщения
	messages := []struct {
		text     string
		time     time.Time
		position widgets.MessagePosition
	}{
		{"Привет! Как дела?", time.Now().Add(-2 * time.Hour), widgets.MessageLeft},
		{"Привет! Все хорошо, спасибо!", time.Now().Add(-2*time.Hour + 5*time.Minute), widgets.MessageRight},
		{"Работаешь над новым проектом?", time.Now().Add(-1 * time.Hour), widgets.MessageLeft},
		{"Да, делаю мессенджер на Go", time.Now().Add(-55 * time.Minute), widgets.MessageRight},
		{"Звучит круто! Покажешь потом?", time.Now().Add(-30 * time.Minute), widgets.MessageLeft},
		{"Конечно! Скоро будет готово", time.Now().Add(-25 * time.Minute), widgets.MessageRight},
	}

	for _, msg := range messages {
		bubble := widgets.NewMessageBubble(msg.text, msg.time, msg.position)
		ui.messagesList.Add(bubble)
	}

	ui.messagesList.Refresh()

	// Прокрутка вниз
	ui.messageScroll.ScrollToBottom()
}

// sendMessage отправляет сообщение
func (ui *UI) sendMessage() {
	text := ui.messageEntry.Text
	if text == "" {
		return
	}

	// Добавляем сообщение в чат
	bubble := widgets.NewMessageBubble(text, time.Now(), widgets.MessageRight)
	ui.messagesList.Add(bubble)
	ui.messagesList.Refresh()

	// Прокрутка вниз
	ui.messageScroll.ScrollToBottom()

	// Очищаем поле ввода
	ui.messageEntry.SetText("")

	// Вызываем обработчик отправки
	if ui.onSendMessage != nil {
		ui.onSendMessage(text)
	}
}

// showMessage показывает сообщение в чате
func (ui *UI) showMessage(title, message string) {
	// TODO: реализовать отображение сообщения
}
