// Package p2p предоставляет агрегированный доступ к P2P сервисам проекта.
//
// Подпакеты:
//   - protocols/profile: обмен профилями между пирами
//   - protocols/chat: обмен сообщениями
//   - protocols/itemsync: синхронизация элементов
//   - protocols/transfer: передача файлов
//   - core: инфраструктура P2P сети
package p2p

// Экспорт констант протоколов из подпакетов
import (
	"projectT/internal/services/p2p/protocols/chat"
	"projectT/internal/services/p2p/protocols/itemsync"
	"projectT/internal/services/p2p/protocols/profile"
	"projectT/internal/services/p2p/protocols/transfer"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

// ProtocolID идентификаторы протоколов (импортированы из подпакетов)
const (
	// ProfileProtocolID идентификатор протокола обмена профилями
	ProfileProtocolID = profile.ProtocolID
	// ItemSyncProtocolID идентификатор протокола синхронизации элементов
	ItemSyncProtocolID = itemsync.ProtocolID
	// TransferProtocolID идентификатор протокола передачи файлов
	TransferProtocolID = transfer.ProtocolID
)

// Экспорт типов из подпакетов для обратной совместимости

// Profile типы из package profile
type (
	ProfileRequest       = profile.ProfileRequest
	ProfileResponse      = profile.ProfileResponse
	ProfileWithSignature = profile.ProfileWithSignature
	ExchangeService      = profile.ExchangeService
)

// Chat типы из package chat
type (
	Message       = chat.Message
	QueuedMessage = chat.QueuedMessage
	ChatService   = chat.Service
	MessageType   = chat.MessageType
)

// ItemSync типы из package itemsync
type (
	ItemRequest     = itemsync.ItemRequest
	ItemResponse    = itemsync.ItemResponse
	ItemFileData    = itemsync.ItemFileData
	ItemSyncService = itemsync.Service
)

// Transfer типы из package transfer
type (
	FileTransferRequest = transfer.FileTransferRequest
	FileTransferChunk   = transfer.FileTransferChunk
	TransferAck         = transfer.TransferAck
	TransferProgress    = transfer.TransferProgress
	TransferType        = transfer.TransferType
	TransferStatus      = transfer.TransferStatus
	TransferService     = transfer.Service
)

// Типы передачи
const (
	TransferTypeFile   = transfer.TransferTypeFile
	TransferTypeAvatar = transfer.TransferTypeAvatar
	TransferTypeImage  = transfer.TransferTypeImage
)

// Статусы передачи
const (
	TransferStatusPending    = transfer.TransferStatusPending
	TransferStatusInProgress = transfer.TransferStatusInProgress
	TransferStatusCompleted  = transfer.TransferStatusCompleted
	TransferStatusFailed     = transfer.TransferStatusFailed
	TransferStatusCancelled  = transfer.TransferStatusCancelled
)

// Фабричные функции для создания сервисов

// NewProfileExchangeService создаёт сервис обмена профилями
func NewProfileExchangeService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey) *ExchangeService {
	return profile.NewExchangeService(host, privKey, pubKey)
}

// NewChatService создаёт сервис чата
func NewChatService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey, profileSvc *profile.ExchangeService) *ChatService {
	return chat.NewService(host, privKey, pubKey, profileSvc)
}

// NewItemSyncService создаёт сервис синхронизации элементов
func NewItemSyncService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey) *ItemSyncService {
	return itemsync.NewService(host, privKey, pubKey)
}

// NewTransferService создаёт сервис передачи файлов
func NewTransferService(host host.Host) *TransferService {
	return transfer.NewService(host)
}
