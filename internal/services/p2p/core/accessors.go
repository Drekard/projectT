// Package core предоставляет функции доступа к сервисам P2P
package core

import (
	"context"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"projectT/internal/services/p2p/address"
	"projectT/internal/services/p2p/autodial"
	"projectT/internal/services/p2p/connection"
	"projectT/internal/services/p2p/discovery"
	"projectT/internal/services/p2p/helper"
	"projectT/internal/services/p2p/protocols/chat"
	"projectT/internal/services/p2p/protocols/itemsync"
	"projectT/internal/services/p2p/protocols/profile"
	"projectT/internal/services/p2p/protocols/profilesync"
	"projectT/internal/services/p2p/protocols/transfer"
	"projectT/internal/storage/database/models"
)

// Host возвращает libp2p хост
func (n *P2PNetwork) Host() host.Host {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.host
}

// DHT возвращает DHT таблицу
func (n *P2PNetwork) DHT() *dht.IpfsDHT {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.dht
}

// PubSub возвращает PubSub систему
func (n *P2PNetwork) PubSub() *pubsub.PubSub {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.pubsub
}

// PeerID возвращает идентификатор текущего пира
func (n *P2PNetwork) PeerID() peer.ID {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.host == nil {
		return ""
	}
	return n.host.ID()
}

// GetPeerAddress возвращает адрес текущего пира для экспорта
func (n *P2PNetwork) GetPeerAddress() (*address.PeerAddress, error) {
	n.mu.RLock()
	host := n.host
	n.mu.RUnlock()
	return address.GetPeerAddress(host)
}

// ImportPeerAddress импортирует адрес пира и добавляет в контакты
func (n *P2PNetwork) ImportPeerAddress(addrStr string) (*address.PeerAddress, error) {
	n.mu.Lock()
	host := n.host
	n.mu.Unlock()
	return address.ImportPeerAddress(host, addrStr)
}

// ConnectToPeer подключается к пиру по адресу
func (n *P2PNetwork) ConnectToPeer(ctx context.Context, addrStr string) error {
	n.mu.RLock()
	host := n.host
	n.mu.RUnlock()
	return address.ConnectToPeer(ctx, host, addrStr)
}

// GetConnectedPeers возвращает список подключённых пиров
func (n *P2PNetwork) GetConnectedPeers() []peer.ID {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.host == nil {
		return []peer.ID{}
	}
	return n.host.Network().Peers()
}

// Discovery возвращает сервис обнаружения
func (n *P2PNetwork) Discovery() *discovery.DiscoveryService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.discovery
}

// DiscoveryService возвращает сервис обнаружения (алиас для Discovery)
func (n *P2PNetwork) DiscoveryService() *discovery.DiscoveryService {
	return n.Discovery()
}

// AddBootstrapPeer добавляет bootstrap-узел
func (n *P2PNetwork) AddBootstrapPeer(multiaddr string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.discovery == nil {
		return errors.New("сервис обнаружения не инициализирован")
	}
	return n.discovery.AddBootstrapPeer(multiaddr)
}

// RemoveBootstrapPeer удаляет bootstrap-узел
func (n *P2PNetwork) RemoveBootstrapPeer(multiaddr string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.discovery == nil {
		return errors.New("сервис обнаружения не инициализирован")
	}
	return n.discovery.RemoveBootstrapPeer(multiaddr)
}

// GetBootstrapPeers возвращает список bootstrap-узлов
func (n *P2PNetwork) GetBootstrapPeers() ([]*models.PeerAddress, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.discovery == nil {
		return nil, errors.New("сервис обнаружения не инициализирован")
	}
	return n.discovery.GetBootstrapPeers()
}

// GetDiscoveredPeers возвращает список обнаруженных пиров
func (n *P2PNetwork) GetDiscoveredPeers() map[string]time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.discovery == nil {
		return make(map[string]time.Time)
	}
	return n.discovery.GetDiscoveredPeers()
}

// Connections возвращает сервис соединений
func (n *P2PNetwork) Connections() *connection.Service {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connections
}

// GetConnectionStatus возвращает статус подключения к пиру
func (n *P2PNetwork) GetConnectionStatus(peerID peer.ID) connection.ConnectionStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.connections == nil {
		return connection.StatusUnknown
	}
	return n.connections.GetConnectionStatus(peerID)
}

// GetConnectedPeersCount возвращает количество подключённых пиров
func (n *P2PNetwork) GetConnectedPeersCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.connections == nil {
		return 0
	}
	return n.connections.GetConnectedPeersCount()
}

// GetPeerInfo возвращает информацию о подключении к пиру
func (n *P2PNetwork) GetPeerInfo(peerID peer.ID) *connection.PeerConnectionInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.connections == nil {
		return nil
	}
	return n.connections.GetPeerInfo(peerID)
}

// Chat возвращает сервис чата
func (n *P2PNetwork) Chat() *chat.Service {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.chat
}

// SendMessage отправляет сообщение пиру
func (n *P2PNetwork) SendMessage(ctx context.Context, peerID peer.ID, content, contentType, metadata string) error {
	n.mu.RLock()
	chatSvc := n.chat
	n.mu.RUnlock()

	if chatSvc == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chatSvc.SendMessage(ctx, peerID, content, contentType, metadata)
}

// SendTextMessage отправляет текстовое сообщение
func (n *P2PNetwork) SendTextMessage(ctx context.Context, peerID peer.ID, content string) error {
	err := n.SendMessage(ctx, peerID, content, "text", "")
	return err
}

// SendFileMessage отправляет сообщение с файлом
func (n *P2PNetwork) SendFileMessage(ctx context.Context, peerID peer.ID, filePath, fileName, mimeType string) error {
	n.mu.RLock()
	chatSvc := n.chat
	n.mu.RUnlock()

	if chatSvc == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chatSvc.SendFileMessage(ctx, peerID, filePath, fileName, mimeType)
}

// SendImageMessage отправляет сообщение с изображением
func (n *P2PNetwork) SendImageMessage(ctx context.Context, peerID peer.ID, imagePath, imageName string) error {
	n.mu.RLock()
	chatSvc := n.chat
	n.mu.RUnlock()

	if chatSvc == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chatSvc.SendImageMessage(ctx, peerID, imagePath, imageName)
}

// GetMessagesForContact получает сообщения для контакта
func (n *P2PNetwork) GetMessagesForContact(contactID int, limit, offset int) ([]*models.ChatMessage, error) {
	n.mu.RLock()
	chatSvc := n.chat
	n.mu.RUnlock()

	if chatSvc == nil {
		return nil, errors.New("ChatService не инициализирован")
	}
	return chatSvc.GetMessagesForContact(contactID, limit, offset)
}

// GetUnreadMessagesCount получает количество непрочитанных сообщений
func (n *P2PNetwork) GetUnreadMessagesCount(contactID int) (int, error) {
	n.mu.RLock()
	chatSvc := n.chat
	n.mu.RUnlock()

	if chatSvc == nil {
		return 0, errors.New("ChatService не инициализирован")
	}
	return chatSvc.GetUnreadMessagesCount(contactID)
}

// MarkMessageAsRead помечает сообщение как прочитанное
func (n *P2PNetwork) MarkMessageAsRead(id int) error {
	n.mu.RLock()
	chatSvc := n.chat
	n.mu.RUnlock()

	if chatSvc == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chatSvc.MarkMessageAsRead(id)
}

// MarkAllMessagesAsRead помечает все сообщения для контакта как прочитанные
func (n *P2PNetwork) MarkAllMessagesAsRead(contactID int) error {
	n.mu.RLock()
	chatSvc := n.chat
	n.mu.RUnlock()

	if chatSvc == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chatSvc.MarkAllMessagesAsRead(contactID)
}

// GetQueuedMessagesCount возвращает количество сообщений в очереди для пира
func (n *P2PNetwork) GetQueuedMessagesCount(peerID peer.ID) int {
	n.mu.RLock()
	chatSvc := n.chat
	n.mu.RUnlock()

	if chatSvc == nil {
		return 0
	}
	return chatSvc.GetQueuedMessagesCount(peerID)
}

// Helper возвращает сервис режима помощника
func (n *P2PNetwork) Helper() *HelperService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.helper
}

// HelperService возвращает внутренний сервис helper
func (n *P2PNetwork) HelperService() *helper.Helper {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.helper == nil {
		return nil
	}
	return n.helper.helper
}

// HelperRegister регистрирует адрес пира в хранилище помощника
func (n *P2PNetwork) HelperRegister(peerID, address string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.helper == nil || n.helper.helper == nil {
		return errors.New("режим помощника не инициализирован")
	}
	return n.helper.helper.Register(peerID, address)
}

// HelperAsk запрашивает адрес пира из хранилища помощника
func (n *P2PNetwork) HelperAsk(peerID string) (*helper.PeerAddressData, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.helper == nil || n.helper.helper == nil {
		return nil, false
	}
	return n.helper.helper.Ask(peerID)
}

// HelperList возвращает список всех зарегистрированных пиров
func (n *P2PNetwork) HelperList() []helper.PeerEntry {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.helper == nil || n.helper.helper == nil {
		return nil
	}
	return n.helper.helper.List()
}

// HelperGetPeerCount возвращает количество зарегистрированных пиров
func (n *P2PNetwork) HelperGetPeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.helper == nil || n.helper.helper == nil {
		return 0
	}
	return n.helper.helper.GetPeerCount()
}

// ProfileExchange возвращает сервис обмена профилями
func (n *P2PNetwork) ProfileExchange() *profile.ExchangeService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.profileExchange
}

// ItemSync возвращает сервис синхронизации элементов
func (n *P2PNetwork) ItemSync() *itemsync.Service {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.itemSync
}

// Transfer возвращает сервис передачи файлов
func (n *P2PNetwork) Transfer() *transfer.Service {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.transfer
}

// ProfileSync возвращает сервис синхронизации профилей
func (n *P2PNetwork) ProfileSync() *profilesync.SyncService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.profileSync
}

// Autodial возвращает менеджер автоподключения
func (n *P2PNetwork) Autodial() *autodial.DialerManager {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.autodial
}

// SendBatch отправляет пакет элементов пиру
func (n *P2PNetwork) SendBatch(ctx context.Context, peerID peer.ID, elementUUIDs []string, transferType transfer.TransferType) (string, error) {
	n.mu.RLock()
	transferSvc := n.transfer
	n.mu.RUnlock()

	if transferSvc == nil {
		return "", errors.New("TransferService не инициализирован")
	}
	return transferSvc.SendBatch(ctx, peerID, elementUUIDs, transferType)
}

// SendFolder отправляет папку пиру
func (n *P2PNetwork) SendFolder(ctx context.Context, peerID peer.ID, parentUUID string) (string, error) {
	n.mu.RLock()
	transferSvc := n.transfer
	n.mu.RUnlock()

	if transferSvc == nil {
		return "", errors.New("TransferService не инициализирован")
	}
	return transferSvc.SendFolder(ctx, peerID, parentUUID)
}

// SendPinnedItems отправляет закреплённые элементы пиру
func (n *P2PNetwork) SendPinnedItems(ctx context.Context, peerID peer.ID) (string, error) {
	n.mu.RLock()
	transferSvc := n.transfer
	n.mu.RUnlock()

	if transferSvc == nil {
		return "", errors.New("TransferService не инициализирован")
	}
	return transferSvc.SendPinnedItems(ctx, peerID)
}

// SendSelection отправляет выбранные элементы пиру
func (n *P2PNetwork) SendSelection(ctx context.Context, peerID peer.ID, elementUUIDs []string) (string, error) {
	n.mu.RLock()
	transferSvc := n.transfer
	n.mu.RUnlock()

	if transferSvc == nil {
		return "", errors.New("TransferService не инициализирован")
	}
	return transferSvc.SendSelection(ctx, peerID, elementUUIDs)
}

// GetBatchProgress возвращает прогресс батча
func (n *P2PNetwork) GetBatchProgress(batchID string) *transfer.BatchProgress {
	n.mu.RLock()
	transferSvc := n.transfer
	n.mu.RUnlock()

	if transferSvc == nil {
		return &transfer.BatchProgress{BatchID: batchID, Status: transfer.TransferStatusPending}
	}
	return transferSvc.GetBatchProgress(batchID)
}

// RequestBatchByUUIDs запрашивает батч элементов у пира
func (n *P2PNetwork) RequestBatchByUUIDs(ctx context.Context, peerID peer.ID, elementUUIDs []string) ([]*models.Item, error) {
	n.mu.RLock()
	itemSyncSvc := n.itemSync
	n.mu.RUnlock()

	if itemSyncSvc == nil {
		return nil, errors.New("ItemSyncService не инициализирован")
	}
	return itemSyncSvc.RequestBatchByUUIDs(ctx, peerID, elementUUIDs)
}

// RequestBatchByUUIDsAsync запрашивает батч элементов асинхронно с коллбэками
func (n *P2PNetwork) RequestBatchByUUIDsAsync(ctx context.Context, peerID peer.ID, elementUUIDs []string, callbacks itemsync.BatchRequestCallbacks) {
	n.mu.RLock()
	itemSyncSvc := n.itemSync
	n.mu.RUnlock()

	if itemSyncSvc == nil {
		if callbacks.OnDone != nil {
			callbacks.OnDone(nil, errors.New("ItemSyncService не инициализирован"))
		}
		return
	}
	itemSyncSvc.RequestBatchByUUIDsAsync(ctx, peerID, elementUUIDs, callbacks)
}

// RequestFolder запрашивает папку у пира
func (n *P2PNetwork) RequestFolder(ctx context.Context, peerID peer.ID, parentUUID string) ([]*models.Item, error) {
	n.mu.RLock()
	itemSyncSvc := n.itemSync
	n.mu.RUnlock()

	if itemSyncSvc == nil {
		return nil, errors.New("ItemSyncService не инициализирован")
	}
	return itemSyncSvc.RequestFolder(ctx, peerID, parentUUID)
}

// RequestRandomItemsAsync запрашивает случайные элементы у пира асинхронно
func (n *P2PNetwork) RequestRandomItemsAsync(ctx context.Context, peerID peer.ID, count int, callbacks itemsync.BatchRequestCallbacks) {
	n.mu.RLock()
	itemSyncSvc := n.itemSync
	n.mu.RUnlock()

	if itemSyncSvc == nil {
		if callbacks.OnDone != nil {
			callbacks.OnDone(nil, errors.New("ItemSyncService не инициализирован"))
		}
		return
	}
	itemSyncSvc.RequestRandomItemsAsync(ctx, peerID, count, callbacks)
}
