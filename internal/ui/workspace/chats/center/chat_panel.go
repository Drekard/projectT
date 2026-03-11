// Package center содержит компоненты центральной панели чата
package center

import (
	"image/color"

	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MessageBubble пузырёк сообщения
type MessageBubble struct {
	container *fyne.Container
}

// NewMessageBubble создаёт новый пузырёк сообщения
func NewMessageBubble(message *models.ChatMessage, isOutgoing bool) *MessageBubble {
	mb := &MessageBubble{}
	mb.container = mb.createBubble(message, isOutgoing)
	return mb
}

// createBubble создаёт пузырёк сообщения
func (mb *MessageBubble) createBubble(message *models.ChatMessage, isOutgoing bool) *fyne.Container {
	// Текст сообщения
	msgLabel := widget.NewLabel(message.Content)
	msgLabel.Wrapping = fyne.TextWrapBreak

	// Время отправки
	timeStr := message.SentAt.Format("15:04")
	timeLabel := widget.NewLabel(timeStr)
	timeLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Компонуем сообщение и время
	content := container.NewVBox(msgLabel, timeLabel)

	// Цвет фона в зависимости от направления
	bgColor := color.RGBA{R: 70, G: 130, B: 180, A: 200} // Синий для исходящих
	if !isOutgoing {
		bgColor = color.RGBA{R: 80, G: 80, B: 80, A: 200} // Серый для входящих
	}

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10

	messageContainer := container.NewStack(bg, container.NewPadded(content))

	// Выравнивание по правому/левому краю
	if isOutgoing {
		return container.NewHBox(layout.NewSpacer(), messageContainer)
	}
	return container.NewHBox(messageContainer, layout.NewSpacer())
}

// Container возвращает контейнер пузырька
func (mb *MessageBubble) Container() *fyne.Container {
	return mb.container
}

// MessageInput поле ввода сообщения
type MessageInput struct {
	entry  *widget.Entry
	button *widget.Button
}

// NewMessageInput создаёт новое поле ввода сообщения
func NewMessageInput(onSend func()) *MessageInput {
	mi := &MessageInput{}
	mi.entry = widget.NewMultiLineEntry()
	mi.entry.SetPlaceHolder("Введите сообщение...")
	mi.entry.Wrapping = fyne.TextWrapBreak

	mi.button = widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		if onSend != nil {
			onSend()
		}
	})
	mi.button.Importance = widget.HighImportance

	// Отправка по Enter
	mi.entry.OnSubmitted = func(s string) {
		if onSend != nil {
			onSend()
		}
	}

	return mi
}

// Container возвращает контейнер поля ввода
func (mi *MessageInput) Container() *fyne.Container {
	return container.NewHBox(mi.entry, mi.button)
}

// Text возвращает текст сообщения
func (mi *MessageInput) Text() string {
	return mi.entry.Text
}

// SetText устанавливает текст сообщения
func (mi *MessageInput) SetText(text string) {
	mi.entry.SetText(text)
}

// Clear очищает поле ввода
func (mi *MessageInput) Clear() {
	mi.entry.SetText("")
}

// SetEnabled устанавливает доступность поля ввода
func (mi *MessageInput) SetEnabled(enabled bool) {
	mi.entry.Disable()
	if enabled {
		mi.entry.Enable()
	}
	// Кнопка не имеет метода SetEnabled в Fyne v2
}

// MessagesList список сообщений
type MessagesList struct {
	container *fyne.Container
	scroll    *container.Scroll
}

// NewMessagesList создаёт новый список сообщений
func NewMessagesList() *MessagesList {
	ml := &MessagesList{}
	ml.container = container.NewVBox()
	ml.scroll = container.NewScroll(ml.container)
	return ml
}

// Container возвращает контейнер списка
func (ml *MessagesList) Container() fyne.CanvasObject {
	return ml.scroll
}

// AddMessage добавляет сообщение в список
func (ml *MessagesList) AddMessage(message *models.ChatMessage, isOutgoing bool) {
	bubble := NewMessageBubble(message, isOutgoing)
	ml.container.Add(bubble.Container())
	ml.scrollToBottom()
}

// AddMessages добавляет несколько сообщений
func (ml *MessagesList) AddMessages(messages []*models.ChatMessage, localPeerID string) {
	for _, msg := range messages {
		isOutgoing := msg.FromPeerID == localPeerID
		ml.AddMessage(msg, isOutgoing)
	}
}

// Clear очищает список сообщений
func (ml *MessagesList) Clear() {
	ml.container.Objects = nil
	ml.container.Refresh()
}

// scrollToBottom прокручивает к последнему сообщению
func (ml *MessagesList) scrollToBottom() {
	if ml.scroll == nil {
		return
	}

	contentHeight := ml.container.MinSize().Height
	scrollHeight := ml.scroll.Size().Height

	if contentHeight > scrollHeight {
		ml.scroll.Offset.Y = contentHeight - scrollHeight
		if ml.scroll.Offset.Y < 0 {
			ml.scroll.Offset.Y = 0
		}
	}

	ml.scroll.Refresh()
}

// ChatPanel панель чата
type ChatPanel struct {
	container    *fyne.Container
	header       *ChatHeader
	messagesList *MessagesList
	messageInput *MessageInput
}

// NewChatPanel создаёт новую панель чата
func NewChatPanel(contact *models.Contact, onSend func(), onClose func()) *ChatPanel {
	cp := &ChatPanel{}

	// Создаём заголовок
	cp.header = NewChatHeader(contact)
	cp.header.SetOnClose(onClose)

	// Создаём список сообщений
	cp.messagesList = NewMessagesList()

	// Создаём поле ввода
	cp.messageInput = NewMessageInput(onSend)

	// Собираем панель
	content := container.NewBorder(
		cp.header.Container(),
		cp.messageInput.Container(),
		nil,
		nil,
		cp.messagesList.Container(),
	)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 40, G: 40, B: 40, A: 255})

	cp.container = container.NewStack(bg, content)

	return cp
}

// Container возвращает контейнер панели
func (cp *ChatPanel) Container() fyne.CanvasObject {
	return cp.container
}

// Header возвращает заголовок
func (cp *ChatPanel) Header() *ChatHeader {
	return cp.header
}

// MessagesList возвращает список сообщений
func (cp *ChatPanel) MessagesList() *MessagesList {
	return cp.messagesList
}

// MessageInput возвращает поле ввода
func (cp *ChatPanel) MessageInput() *MessageInput {
	return cp.messageInput
}

// AddMessage добавляет сообщение
func (cp *ChatPanel) AddMessage(message *models.ChatMessage, isOutgoing bool) {
	cp.messagesList.AddMessage(message, isOutgoing)
}

// LoadMessages загружает сообщения
func (cp *ChatPanel) LoadMessages(messages []*models.ChatMessage, localPeerID string) {
	cp.messagesList.AddMessages(messages, localPeerID)
}

// Clear очищает панель
func (cp *ChatPanel) Clear() {
	cp.messagesList.Clear()
}

// UpdateStatus обновляет статус
func (cp *ChatPanel) UpdateStatus(status string) {
	cp.header.UpdateStatus(status)
}
