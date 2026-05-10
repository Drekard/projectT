package controllers

import (
	"fmt"
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"github.com/libp2p/go-libp2p/core/peer"
)

// SendMessage отправляет текстовое сообщение
func (cc *ChatController) SendMessage(text string) error {
	if cc.p2pUI == nil {
		return fmt.Errorf("P2P сервис не инициализирован")
	}

	if cc.currentContact == nil || cc.currentContact.PeerID == "" {
		return fmt.Errorf("текущий контакт не выбран или не имеет PeerID")
	}

	if cc.currentContact.IsLocalChat() {
		chatSvc := services.GetChatService()
		if chatSvc != nil {
			_, err := chatSvc.SendTextMessage(0, cc.currentContact.PeerID, cc.localPeerID, text)
			return err
		}
		return nil
	}

	peerID, err := peer.Decode(cc.currentContact.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	if err := cc.p2pUI.SendMessage(peerID, text); err != nil {
		return fmt.Errorf("ошибка отправки сообщения: %w", err)
	}

	return nil
}

// SendElementMessage отправляет элемент в чат
func (cc *ChatController) SendElementMessage(item *models.Item) error {
	if cc.p2pUI == nil {
		return fmt.Errorf("P2P сервис не инициализирован")
	}

	if cc.currentContact == nil || cc.currentContact.PeerID == "" {
		return fmt.Errorf("текущий контакт не выбран или не имеет PeerID")
	}

	if cc.currentContact.IsLocalChat() {
		return nil
	}

	peerID, err := peer.Decode(cc.currentContact.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	if err := cc.p2pUI.SendElementMessage(peerID, item); err != nil {
		return fmt.Errorf("ошибка отправки элемента: %w", err)
	}

	return nil
}

// OpenChat открывает чат с контактом
func (cc *ChatController) OpenChat(contact *models.Contact) error {
	cc.currentContact = contact
	cc.currentChatID = contact.ID

	if cc.onChatOpened != nil {
		cc.onChatOpened(contact)
	}

	_, _ = cc.LoadMessages()

	if cc.p2pUI != nil && !contact.IsLocalChat() {
		go func() {
			err := cc.p2pUI.RequestProfile(contact.PeerID)
			if err != nil {
				return
			}

			cc.downloadPinnedElements(contact.PeerID)
		}()
	}

	return nil
}

// OpenPeerChat открывает чат с пиром (создаёт временный контакт)
func (cc *ChatController) OpenPeerChat(peerID, username string) error {
	profile, _ := queries.GetProfileByPeerID(peerID)

	contactUsername := username
	contactTitle := ""
	contactAvatarPath := ""
	if profile != nil {
		contactUsername = profile.Username
		contactTitle = profile.Title
		contactAvatarPath = profile.AvatarPath
	}

	tempContact := &models.Contact{
		PeerID:     peerID,
		Username:   contactUsername,
		Title:      contactTitle,
		AvatarPath: contactAvatarPath,
		ID:         0,
	}

	return cc.OpenChat(tempContact)
}

// OpenLocalChat открывает локальный чат с самим собой
func (cc *ChatController) OpenLocalChat() error {
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		return fmt.Errorf("ошибка загрузки локального профиля: %w", err)
	}

	_, err = queries.GetOrCreateLocalChat(localProfile.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка получения локального чата: %w", err)
	}

	localContact := models.NewLocalContact(
		localProfile.Username,
		localProfile.Title,
		localProfile.AvatarPath,
	)

	return cc.OpenChat(localContact)
}

// CloseChat закрывает текущий чат
func (cc *ChatController) CloseChat() error {
	cc.currentContact = nil
	cc.currentChatID = 0

	if cc.onChatClosed != nil {
		cc.onChatClosed()
	}

	return nil
}

// LoadMessages загружает сообщения для текущего чата
func (cc *ChatController) LoadMessages() ([]*models.ChatMessage, error) {
	if cc.currentContact == nil {
		return nil, fmt.Errorf("текущий контакт не выбран")
	}

	chat, err := queries.GetChatByPeerID(cc.currentContact.PeerID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения чата: %w", err)
	}
	if chat == nil {
		return []*models.ChatMessage{}, nil
	}

	messages, err := queries.GetMessagesForChat(chat.ID, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки сообщений: %w", err)
	}

	return messages, nil
}

// LoadMessagesForChat загружает сообщения для чата по ID
func (cc *ChatController) LoadMessagesForChat(chatID int) ([]*models.ChatMessage, error) {
	messages, err := queries.GetMessagesForChat(chatID, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки сообщений: %w", err)
	}

	return messages, nil
}
