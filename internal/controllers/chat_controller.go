package controllers

import (
	"projectT/internal/services"
	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"
)

// ChatMessageEvent представляет событие нового сообщения
type ChatMessageEvent = services.ChatMessageEvent

// ChatController контролирует бизнес-логику чатов
type ChatController struct {
	p2pUI          *network.UIP2P
	messageChannel <-chan *ChatMessageEvent

	onMessageSent          func(message *models.ChatMessage)
	onMessageReceived      func(event *services.ChatMessageEvent)
	onChatOpened           func(contact *models.Contact)
	onChatClosed           func()
	onContactsRefreshed    func()
	onPinnedElementsLoaded func(peerID string)

	currentContact *models.Contact
	currentChatID  int
	localPeerID    string
}

// NewChatController создаёт новый контроллер чатов
func NewChatController() *ChatController {
	cc := &ChatController{}

	chatSvc := services.GetChatService()
	if chatSvc != nil {
		cc.messageChannel = chatSvc.Subscribe()
		go cc.handleMessageEvents()
	}

	return cc
}

// SetP2PService устанавливает P2P сервис
func (cc *ChatController) SetP2PService(p2pUI *network.UIP2P) {
	cc.p2pUI = p2pUI

	if p2pUI != nil {
		status := p2pUI.GetStatus()
		if status != nil {
			cc.localPeerID = status.PeerID
		}
	}
}

// SetOnMessageSent устанавливает callback при отправке сообщения
func (cc *ChatController) SetOnMessageSent(handler func(message *models.ChatMessage)) {
	cc.onMessageSent = handler
}

// SetOnMessageReceived устанавливает callback при получении сообщения
func (cc *ChatController) SetOnMessageReceived(handler func(event *services.ChatMessageEvent)) {
	cc.onMessageReceived = handler
}

// SetOnChatOpened устанавливает callback при открытии чата
func (cc *ChatController) SetOnChatOpened(handler func(contact *models.Contact)) {
	cc.onChatOpened = handler
}

// SetOnChatClosed устанавливает callback при закрытии чата
func (cc *ChatController) SetOnChatClosed(handler func()) {
	cc.onChatClosed = handler
}

// SetOnContactsRefreshed устанавливает callback при обновлении списка контактов
func (cc *ChatController) SetOnContactsRefreshed(handler func()) {
	cc.onContactsRefreshed = handler
}

// SetOnPinnedElementsLoaded устанавливает callback при загрузке закреплённых элементов
func (cc *ChatController) SetOnPinnedElementsLoaded(handler func(peerID string)) {
	cc.onPinnedElementsLoaded = handler
}

// handleMessageEvents обрабатывает события новых сообщений
func (cc *ChatController) handleMessageEvents() {
	for event := range cc.messageChannel {
		if cc.onMessageReceived != nil {
			cc.onMessageReceived(event)
		}

		if cc.onContactsRefreshed != nil {
			cc.onContactsRefreshed()
		}
	}
}
