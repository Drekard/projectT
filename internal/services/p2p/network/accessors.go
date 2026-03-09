// Package network предоставляет функции доступа к сервисам P2P
package network

import (
	"context"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	p2p "projectT/internal/services/p2p"
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
func (n *P2PNetwork) GetPeerAddress() (*p2p.PeerAddress, error) {
	n.mu.RLock()
	host := n.host
	n.mu.RUnlock()
	return p2p.GetPeerAddress(host)
}

// ImportPeerAddress импортирует адрес пира и добавляет в контакты
func (n *P2PNetwork) ImportPeerAddress(addrStr string) (*p2p.PeerAddress, error) {
	n.mu.Lock()
	host := n.host
	n.mu.Unlock()
	return p2p.ImportPeerAddress(host, addrStr)
}

// ConnectToPeer подключается к пиру по адресу
func (n *P2PNetwork) ConnectToPeer(ctx context.Context, addrStr string) error {
	n.mu.RLock()
	host := n.host
	n.mu.RUnlock()
	return p2p.ConnectToPeer(ctx, host, addrStr)
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
func (n *P2PNetwork) Discovery() *p2p.DiscoveryService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.discovery
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
func (n *P2PNetwork) GetBootstrapPeers() ([]*models.BootstrapPeer, error) {
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
func (n *P2PNetwork) Connections() *p2p.ConnectionService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connections
}

// GetConnectionStatus возвращает статус подключения к пиру
func (n *P2PNetwork) GetConnectionStatus(peerID peer.ID) p2p.ConnectionStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.connections == nil {
		return p2p.StatusUnknown
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
func (n *P2PNetwork) GetPeerInfo(peerID peer.ID) *p2p.PeerConnectionInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.connections == nil {
		return nil
	}
	return n.connections.GetPeerInfo(peerID)
}

// Chat возвращает сервис чата
func (n *P2PNetwork) Chat() *p2p.ChatService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.chat
}

// SendMessage отправляет сообщение пиру
func (n *P2PNetwork) SendMessage(ctx context.Context, peerID peer.ID, content, contentType, metadata string) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.SendMessage(ctx, peerID, content, contentType, metadata)
}

// SendTextMessage отправляет текстовое сообщение
func (n *P2PNetwork) SendTextMessage(ctx context.Context, peerID peer.ID, content string) error {
	return n.SendMessage(ctx, peerID, content, "text", "")
}

// SendFileMessage отправляет сообщение с файлом
func (n *P2PNetwork) SendFileMessage(ctx context.Context, peerID peer.ID, filePath, fileName, mimeType string) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.SendFileMessage(ctx, peerID, filePath, fileName, mimeType)
}

// SendImageMessage отправляет сообщение с изображением
func (n *P2PNetwork) SendImageMessage(ctx context.Context, peerID peer.ID, imagePath, imageName string) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.SendImageMessage(ctx, peerID, imagePath, imageName)
}

// GetMessagesForContact получает сообщения для контакта
func (n *P2PNetwork) GetMessagesForContact(contactID int, limit, offset int) ([]*models.ChatMessage, error) {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return nil, errors.New("ChatService не инициализирован")
	}
	return chat.GetMessagesForContact(contactID, limit, offset)
}

// GetUnreadMessagesCount получает количество непрочитанных сообщений
func (n *P2PNetwork) GetUnreadMessagesCount(contactID int) (int, error) {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return 0, errors.New("ChatService не инициализирован")
	}
	return chat.GetUnreadMessagesCount(contactID)
}

// MarkMessageAsRead помечает сообщение как прочитанное
func (n *P2PNetwork) MarkMessageAsRead(id int) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.MarkMessageAsRead(id)
}

// MarkAllMessagesAsRead помечает все сообщения для контакта как прочитанные
func (n *P2PNetwork) MarkAllMessagesAsRead(contactID int) error {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return errors.New("ChatService не инициализирован")
	}
	return chat.MarkAllMessagesAsRead(contactID)
}

// GetQueuedMessagesCount возвращает количество сообщений в очереди для пира
func (n *P2PNetwork) GetQueuedMessagesCount(peerID peer.ID) int {
	n.mu.RLock()
	chat := n.chat
	n.mu.RUnlock()

	if chat == nil {
		return 0
	}
	return chat.GetQueuedMessagesCount(peerID)
}

// Helper возвращает сервис режима помощника
func (n *P2PNetwork) Helper() *HelperService {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.helper
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
func (n *P2PNetwork) HelperAsk(peerID string) (*p2p.PeerAddressData, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.helper == nil || n.helper.helper == nil {
		return nil, false
	}
	return n.helper.helper.Ask(peerID)
}

// HelperList возвращает список всех зарегистрированных пиров
func (n *P2PNetwork) HelperList() []p2p.PeerEntry {
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
