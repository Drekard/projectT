// Package center содержит компоненты центральной панели чата
package center

import (
	"image/color"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/cards/concrete"
	"projectT/internal/ui/cards/hover_preview"
	"projectT/internal/ui/workspace/chats/dialogs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MessageBubble пузырёк сообщения
type MessageBubble struct {
	container fyne.CanvasObject
}

// NewMessageBubble создаёт новый пузырёк сообщения
func NewMessageBubble(message *models.ChatMessage, isOutgoing bool, onRightClick func()) *MessageBubble {
	mb := &MessageBubble{}

	// Проверяем тип сообщения
	if message.ContentType == "element" {
		mb.container = mb.createBubbleForElement(message, isOutgoing, onRightClick)
	} else {
		mb.container = mb.createBubble(message, isOutgoing, onRightClick)
	}

	return mb
}

// createBubble создаёт пузырёк сообщения
func (mb *MessageBubble) createBubble(message *models.ChatMessage, isOutgoing bool, onRightClick func()) fyne.CanvasObject {
	// Текст сообщения
	msgLabel := widget.NewLabel(message.Content)
	msgLabel.Wrapping = fyne.TextWrapBreak

	// Выравнивание текста в зависимости от направления
	if isOutgoing {
		msgLabel.Alignment = fyne.TextAlignTrailing
	}

	// Время отправки
	timeStr := message.SentAt.Format("15:04")
	timeLabel := widget.NewLabel(timeStr)
	timeLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Выравнивание времени в зависимости от направления
	if isOutgoing {
		timeLabel.Alignment = fyne.TextAlignTrailing
	}

	// Компонуем сообщение и время
	content := container.NewVBox(msgLabel, timeLabel)

	// Цвет фона в зависимости от направления
	bgColor := color.RGBA{R: 144, G: 55, B: 255, A: 200} // Синий для исходящих
	if !isOutgoing {
		bgColor = color.RGBA{R: 80, G: 80, B: 80, A: 200} // Серый для входящих
	}

	// Расчёт ширины на основе количества символов (максимум 300)
	const (
		maxWidth     = 300
		minWidth     = 100
		charsPerUnit = 10 // символов на единицу ширины
	)
	calculatedWidth := float32(minWidth) + float32(len(message.Content))/charsPerUnit*float32(maxWidth-minWidth)/10
	if calculatedWidth > maxWidth {
		calculatedWidth = maxWidth
	}

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(calculatedWidth, 20))

	messageContainer := container.NewStack(bg, container.NewPadded(content))

	// Выравнивание по правому/левому краю
	var bubbleContent fyne.CanvasObject
	if isOutgoing {
		bubbleContent = container.NewHBox(layout.NewSpacer(), messageContainer)
	} else {
		bubbleContent = container.NewHBox(messageContainer, layout.NewSpacer())
	}

	// Оборачиваем в кликабельный виджет для обработки правого клика
	clickableBubble := hover_preview.NewClickableCard(bubbleContent, onRightClick)

	return clickableBubble
}

// createBubbleForElement создаёт пузырёк для сообщения типа element
func (mb *MessageBubble) createBubbleForElement(message *models.ChatMessage, isOutgoing bool, onRightClick func()) fyne.CanvasObject {
	// Извлекаем element_uuid из Content
	elementUUID := message.Content
	if elementUUID == "" {
		// Если UUID пустой, показываем ошибку
		return mb.createErrorBubble("Неверный формат элемента")
	}

	// Загружаем элемент из базы данных по element_uuid
	item, err := queries.GetItemByElementUUID(elementUUID)
	if err != nil {
		// Если элемент не найден, показываем сообщение об ошибке
		return mb.createErrorBubble("Элемент не найден")
	}

	// Создаём полноценную карточку элемента используя функционал concrete
	var cardRenderer fyne.CanvasObject
	switch item.Type {
	case "folder":
		cardRenderer = concrete.NewFolderCard(item).GetContainer()
	case "element":
		// Для элементов используем композитную карточку
		cardRenderer = concrete.NewCompositeCard(item).GetContainer()
	default:
		// Для неизвестных типов используем композитную карточку
		cardRenderer = concrete.NewCompositeCard(item).GetContainer()
	}

	// Время отправки
	timeStr := message.SentAt.Format("15:04")
	timeLabel := widget.NewLabel(timeStr)
	timeLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Выравнивание времени в зависимости от направления
	if isOutgoing {
		timeLabel.Alignment = fyne.TextAlignTrailing
	}

	// Компонуем карточку и время
	content := container.NewVBox(
		cardRenderer,
		timeLabel,
	)

	// Цвет фона в зависимости от направления
	bgColor := color.RGBA{R: 144, G: 55, B: 255, A: 200} // Синий для исходящих
	if !isOutgoing {
		bgColor = color.RGBA{R: 80, G: 80, B: 80, A: 200} // Серый для входящих
	}

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(200, 150))

	messageContainer := container.NewStack(bg, container.NewPadded(content))

	// Выравнивание по правому/левому краю
	var bubbleContent fyne.CanvasObject
	if isOutgoing {
		bubbleContent = container.NewHBox(layout.NewSpacer(), messageContainer)
	} else {
		bubbleContent = container.NewHBox(messageContainer, layout.NewSpacer())
	}

	// Оборачиваем в кликабельный виджет для обработки правого клика
	clickableBubble := hover_preview.NewClickableCard(bubbleContent, onRightClick)

	return clickableBubble
}

// createErrorBubble создаёт пузырёк с сообщением об ошибке
func (mb *MessageBubble) createErrorBubble(errorMsg string) fyne.CanvasObject {
	msgLabel := widget.NewLabel(errorMsg)
	msgLabel.Wrapping = fyne.TextWrapBreak

	// Используем RichText для установки цвета
	timeLabel := widget.NewRichTextFromMarkdown("*ошибка*")
	if len(timeLabel.Segments) > 0 {
		timeLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyleInline
		timeLabel.Segments[0].(*widget.TextSegment).Style.ColorName = theme.ColorNameError
	}

	content := container.NewVBox(msgLabel, timeLabel)

	bgColor := color.RGBA{R: 200, G: 50, B: 50, A: 200} // Красный для ошибок
	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(200, 50))

	messageContainer := container.NewStack(bg, container.NewPadded(content))
	bubbleContent := container.NewHBox(messageContainer, layout.NewSpacer())

	clickableBubble := hover_preview.NewClickableCard(bubbleContent, func() {})

	return clickableBubble
}

// Container возвращает контейнер пузырька
func (mb *MessageBubble) Container() fyne.CanvasObject {
	return mb.container
}

// MessageInput поле ввода сообщения
type MessageInput struct {
	entry          *widget.Entry
	entryContainer *fyne.Container
	button         *widget.Button
}

// NewMessageInput создаёт новое поле ввода сообщения
func NewMessageInput(onSend func()) *MessageInput {
	mi := &MessageInput{}

	// Создаём поле ввода с минимальной шириной
	entryWidget := widget.NewMultiLineEntry()
	entryWidget.SetPlaceHolder("Введите сообщение...")
	entryWidget.Wrapping = fyne.TextWrapBreak

	// Ограничиваем минимальную ширину
	bg := canvas.NewRectangle(color.Transparent)
	bg.SetMinSize(fyne.NewSize(600, 0))
	mi.entryContainer = container.NewStack(bg, entryWidget)
	mi.entry = entryWidget

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
	return mi.entryContainer
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
	container   *fyne.Container
	scroll      *container.Scroll
	menuManager *dialogs.MessageMenuManager
	localPeerID string
	onRefresh   func()
}

// NewMessagesList создаёт новый список сообщений
func NewMessagesList(menuManager *dialogs.MessageMenuManager, localPeerID string, onRefresh func()) *MessagesList {
	ml := &MessagesList{
		menuManager: menuManager,
		localPeerID: localPeerID,
		onRefresh:   onRefresh,
	}
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
	// Создаём пузырёк сообщения с обработчиком правого клика
	var bubbleContainer fyne.CanvasObject
	bubble := NewMessageBubble(
		message,
		isOutgoing,
		func() {
			// Показываем контекстное меню при правом клике
			if ml.menuManager != nil {
				ml.menuManager.ShowMessageMenu(message, bubbleContainer, isOutgoing)
			}
		},
	)
	bubbleContainer = bubble.Container()
	ml.container.Add(bubbleContainer)
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
	contactID    int
	messagesList *MessagesList
	messageInput *MessageInput
	menuManager  *dialogs.MessageMenuManager
	localPeerID  string
}

// NewChatPanel создаёт новую панель чата
func NewChatPanel(contact *models.Contact, onSend func(), onClose func(), localPeerID string) *ChatPanel {
	cp := &ChatPanel{
		contactID:   contact.ID,
		localPeerID: localPeerID,
	}

	// Создаём менеджер меню для сообщений
	cp.menuManager = dialogs.NewMessageMenuManager(
		func(message *models.ChatMessage) {
			// Обновляем сообщение в UI
			cp.LoadMessagesForCurrentContact()
		},
		func(messageID int) {
			// Удаляем сообщение из UI
			cp.LoadMessagesForCurrentContact()
		},
	)

	// Создаём список сообщений с менеджером меню
	cp.messagesList = NewMessagesList(cp.menuManager, cp.localPeerID, func() {
		// Функция обновления - перезагружаем сообщения
		cp.LoadMessagesForCurrentContact()
	})

	// Создаём поле ввода
	cp.messageInput = NewMessageInput(onSend)

	// Компонуем поле ввода и кнопку
	inputRow := container.NewHBox(
		cp.messageInput.Container(),
		cp.messageInput.button,
	)

	// Собираем панель
	content := container.NewBorder(
		nil,
		inputRow,
		nil,
		nil,
		cp.messagesList.Container(),
	)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 0})

	cp.container = container.NewStack(bg, content)

	// Автоматически загружаем сообщения для контакта
	cp.LoadMessagesForCurrentContact()

	return cp
}

// Container возвращает контейнер панели
func (cp *ChatPanel) Container() fyne.CanvasObject {
	return cp.container
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

// LoadMessagesForCurrentContact загружает сообщения для текущего контакта
func (cp *ChatPanel) LoadMessagesForCurrentContact() {
	// Очищаем текущие сообщения
	cp.Clear()

	// Получаем сообщения из БД
	messages, err := queries.GetMessagesForContact(cp.contactID, 100, 0)
	if err != nil {
		// Если ошибка, просто не загружаем ничего
		return
	}

	// Загружаем сообщения в список
	cp.messagesList.AddMessages(messages, cp.localPeerID)
}

// Clear очищает панель
func (cp *ChatPanel) Clear() {
	cp.messagesList.Clear()
}
