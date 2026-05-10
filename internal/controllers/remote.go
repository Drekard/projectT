package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// SendElementMetadata отправляет метаданные элемента через P2P
func (cc *ChatController) SendElementMetadata(ctx context.Context, item *models.Item) (string, error) {
	if cc.p2pUI == nil || cc.currentContact == nil {
		return "", fmt.Errorf("P2P сервис или контакт не инициализированы")
	}

	peerID, err := peer.Decode(cc.currentContact.PeerID)
	if err != nil {
		return "", fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	p2pNet := cc.p2pUI.GetNetwork()
	if p2pNet == nil {
		return "", fmt.Errorf("P2P сеть не инициализирована")
	}

	transferSvc := p2pNet.Transfer()
	if transferSvc == nil {
		return "", fmt.Errorf("TransferService не инициализирован")
	}

	transferID, err := transferSvc.SendElementMetadata(ctx, peerID, item.ElementUUID, item.Title, item.Description, item.ContentMeta)
	if err != nil {
		return "", fmt.Errorf("ошибка отправки метаданных: %w", err)
	}

	return transferID, nil
}

// RequestItem запрашивает элемент у пира
func (cc *ChatController) RequestItem(ctx context.Context, peerIDStr, elementUUID string) (*models.Item, error) {
	if cc.p2pUI == nil {
		return nil, fmt.Errorf("P2P сервис не инициализирован")
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	p2pNet := cc.p2pUI.GetNetwork()
	if p2pNet == nil {
		return nil, fmt.Errorf("P2P сеть не инициализирована")
	}

	itemSyncSvc := p2pNet.ItemSync()
	if itemSyncSvc == nil {
		return nil, fmt.Errorf("ItemSyncService не инициализирован")
	}

	item, err := itemSyncSvc.RequestItemByElementUUID(ctx, peerID, elementUUID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса элемента: %w", err)
	}

	return item, nil
}

// downloadPinnedElements загружает pinned элементы из профиля пира
func (cc *ChatController) downloadPinnedElements(peerIDStr string) {
	if cc.p2pUI == nil {
		return
	}

	profile, err := queries.GetProfileByPeerID(peerIDStr)
	if err != nil || profile == nil {
		return
	}

	var pinnedUUIDs []string
	if err := json.Unmarshal([]byte(profile.PinnedUUIDs), &pinnedUUIDs); err != nil {
		return
	}

	if len(pinnedUUIDs) == 0 {
		return
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return
	}

	p2pNet := cc.p2pUI.GetNetwork()
	if p2pNet == nil {
		return
	}

	itemSyncSvc := p2pNet.ItemSync()
	if itemSyncSvc == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	loadedCount := 0
	for _, uuid := range pinnedUUIDs {
		existing, err := queries.GetItemByElementUUID(uuid)
		if err == nil && existing != nil {
			loadedCount++
			continue
		}

		item, err := itemSyncSvc.RequestItemByElementUUID(ctx, peerID, uuid)
		if err != nil {
			continue
		}

		if item != nil {
			loadedCount++
		}
	}

	if cc.onPinnedElementsLoaded != nil && loadedCount > 0 {
		cc.onPinnedElementsLoaded(peerIDStr)
	}
}

// SendFolderToChat отправляет папку в чат через batch transfer
func (cc *ChatController) SendFolderToChat(contactID int, parentUUID string) error {
	if cc.p2pUI == nil {
		return fmt.Errorf("P2P сервис не инициализирован")
	}

	contact, err := queries.GetContact(contactID)
	if err != nil {
		return fmt.Errorf("ошибка получения контакта: %w", err)
	}
	if contact == nil || contact.PeerID == "" {
		return fmt.Errorf("контакт не найден или не имеет PeerID")
	}

	if contact.IsLocalChat() {
		return fmt.Errorf("нельзя отправить папку в локальный чат")
	}

	peerID, err := peer.Decode(contact.PeerID)
	if err != nil {
		return fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	folder, err := queries.GetItemByElementUUID(parentUUID)
	if err != nil {
		return fmt.Errorf("ошибка получения папки: %w", err)
	}
	if folder == nil {
		return fmt.Errorf("папка не найдена: %s", parentUUID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	p2pNet := cc.p2pUI.GetNetwork()
	if p2pNet == nil {
		return fmt.Errorf("P2P сеть не инициализирована")
	}

	transferSvc := p2pNet.Transfer()
	if transferSvc == nil {
		return fmt.Errorf("TransferService не инициализирован")
	}

	_, err = transferSvc.SendFolder(ctx, peerID, folder.ElementUUID)
	if err != nil {
		return fmt.Errorf("ошибка отправки папки: %w", err)
	}

	chatSvc := services.GetChatService()
	if chatSvc != nil {
		_, _ = chatSvc.SendFolderMessage(contactID, contact.PeerID, cc.localPeerID, folder)
	}

	return nil
}

// LoadRemoteFolder загружает содержимое папки удалённого пира
func (cc *ChatController) LoadRemoteFolder(peerIDStr, folderUUID string) ([]*models.Item, error) {
	if cc.p2pUI == nil {
		return nil, fmt.Errorf("P2P сервис не инициализирован")
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	p2pNet := cc.p2pUI.GetNetwork()
	if p2pNet == nil {
		return nil, fmt.Errorf("P2P сеть не инициализирована")
	}

	itemSyncSvc := p2pNet.ItemSync()
	if itemSyncSvc == nil {
		return nil, fmt.Errorf("ItemSyncService не инициализирован")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	items, err := itemSyncSvc.RequestFolder(ctx, peerID, folderUUID)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки папки: %w", err)
	}

	return items, nil
}

// LoadRemoteProfileItems загружает pinned элементы профиля удалённого пира
func (cc *ChatController) LoadRemoteProfileItems(peerIDStr string) ([]*models.Item, error) {
	if cc.p2pUI == nil {
		return nil, fmt.Errorf("P2P сервис не инициализирован")
	}

	profile, err := queries.GetProfileByPeerID(peerIDStr)
	if err != nil || profile == nil {
		return nil, fmt.Errorf("профиль пира не найден: %w", err)
	}

	var pinnedUUIDs []string
	if err := json.Unmarshal([]byte(profile.PinnedUUIDs), &pinnedUUIDs); err != nil {
		return nil, fmt.Errorf("ошибка парсинга pinned UUIDs: %w", err)
	}

	if len(pinnedUUIDs) == 0 {
		return []*models.Item{}, nil
	}

	items, err := queries.GetRemoteItemsByElementUUIDs(peerIDStr, pinnedUUIDs)
	if err == nil && len(items) == len(pinnedUUIDs) {
		return items, nil
	}

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования PeerID: %w", err)
	}

	p2pNet := cc.p2pUI.GetNetwork()
	if p2pNet == nil {
		return nil, fmt.Errorf("P2P сеть не инициализирована")
	}

	itemSyncSvc := p2pNet.ItemSync()
	if itemSyncSvc == nil {
		return nil, fmt.Errorf("ItemSyncService не инициализирован")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	items, err = itemSyncSvc.RequestBatchByUUIDs(ctx, peerID, pinnedUUIDs)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса pinned элементов: %w", err)
	}

	return items, nil
}
