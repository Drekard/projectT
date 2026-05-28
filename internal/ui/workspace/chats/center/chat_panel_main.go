package center

import (
	"image/color"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/chats/dialogs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// ChatPanel панель чата
type ChatPanel struct {
	container    *fyne.Container
	contact      *models.Contact
	contactID    int
	peerID       string
	messagesList *MessagesList
	messageInput *MessageInput
	menuManager  *dialogs.MessageMenuManager
	localPeerID  string
	onOpenFolder func(folderUUID, peerID string)
}

// NewChatPanel создаёт новую панель чата
func NewChatPanel(contact *models.Contact, onSend func(), onClose func(), localPeerID string, onOpenFolder func(folderUUID, peerID string)) *ChatPanel {
	cp := &ChatPanel{
		contact:      contact,
		contactID:    contact.ID,
		peerID:       contact.PeerID,
		localPeerID:  localPeerID,
		onOpenFolder: onOpenFolder,
	}

	cp.menuManager = dialogs.NewMessageMenuManager(
		func(message *models.ChatMessage) {
			cp.LoadMessagesForCurrentContact()
		},
		func(messageID int) {
			cp.LoadMessagesForCurrentContact()
		},
	)

	cp.messagesList = NewMessagesList(cp.menuManager, cp.localPeerID, func() {
		cp.LoadMessagesForCurrentContact()
	}, cp.onOpenFolder)

	cp.messageInput = NewMessageInput(onSend)

	inputRow := container.NewBorder(nil, nil, nil, cp.messageInput.button, cp.messageInput.Container())

	content := container.NewBorder(
		nil,
		inputRow,
		nil,
		nil,
		cp.messagesList.Container(),
	)

	bg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 0})

	cp.container = container.NewStack(bg, content)

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
	cp.Clear()

	var messages []*models.ChatMessage
	var err error

	if cp.contactID == 0 {
		chat, chatErr := queries.GetChatByPeerID(cp.peerID)
		if chatErr != nil {
			return
		}
		if chat == nil {
			return
		}
		messages, err = queries.GetMessagesForChat(chat.ID, 100, 0)
	} else {
		messages, err = queries.GetMessagesForContact(cp.contactID, 100, 0)
	}

	if err != nil {
		return
	}

	cp.messagesList.AddMessages(messages, cp.localPeerID)
}

// Clear очищает панель
func (cp *ChatPanel) Clear() {
	cp.messagesList.Clear()
}
