package center

import (
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/chats/dialogs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// MessagesList список сообщений
type MessagesList struct {
	container    *fyne.Container
	scroll       *container.Scroll
	menuManager  *dialogs.MessageMenuManager
	localPeerID  string
	onRefresh    func()
	onOpenFolder func(folderUUID, peerID string)
}

// NewMessagesList создаёт новый список сообщений
func NewMessagesList(menuManager *dialogs.MessageMenuManager, localPeerID string, onRefresh func(), onOpenFolder func(folderUUID, peerID string)) *MessagesList {
	ml := &MessagesList{
		menuManager:  menuManager,
		localPeerID:  localPeerID,
		onRefresh:    onRefresh,
		onOpenFolder: onOpenFolder,
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
	var bubbleContainer fyne.CanvasObject
	bubble := NewMessageBubble(
		message,
		isOutgoing,
		func() {
			if ml.menuManager != nil {
				ml.menuManager.ShowMessageMenu(message, bubbleContainer, isOutgoing)
			}
		},
		ml.onOpenFolder,
	)
	bubbleContainer = bubble.Container()
	ml.container.Add(bubbleContainer)

	ml.container.Refresh()
	if ml.scroll != nil {
		ml.scroll.Refresh()
	}

	ml.scrollToBottom()
}

// AddMessages добавляет несколько сообщений
func (ml *MessagesList) AddMessages(messages []*models.ChatMessage, localPeerID string) {
	for _, msg := range messages {
		isOutgoing := localPeerID != "" && msg.FromPeerID != "" && msg.FromPeerID == localPeerID
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

	ml.scroll.Refresh()

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
