package controllers

import (
	"fmt"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// GetCurrentContact возвращает текущий контакт
func (cc *ChatController) GetCurrentContact() *models.Contact {
	return cc.currentContact
}

// GetCurrentChatID возвращает текущий ID чата
func (cc *ChatController) GetCurrentChatID() int {
	return cc.currentChatID
}

// GetLocalPeerID возвращает локальный PeerID
func (cc *ChatController) GetLocalPeerID() string {
	return cc.localPeerID
}

// GetContactByPeerID получает контакт по PeerID
func (cc *ChatController) GetContactByPeerID(peerID string) (*models.Contact, error) {
	return queries.GetContactByPeerID(peerID)
}

// RefreshContacts обновляет список контактов
func (cc *ChatController) RefreshContacts() {
	if cc.onContactsRefreshed != nil {
		cc.onContactsRefreshed()
	}
}

// DeleteContact удаляет контакт
func (cc *ChatController) DeleteContact(chatID int, peerID string) error {
	if err := queries.DeleteContact(chatID); err != nil {
		return fmt.Errorf("ошибка удаления контакта: %w", err)
	}

	if cc.currentChatID == chatID {
		_ = cc.CloseChat()
	}

	cc.RefreshContacts()

	return nil
}
