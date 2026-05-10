package itemsync

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

// Service сервис для синхронизации элементов между пирами
type Service struct {
	host         host.Host
	localPrivKey crypto.PrivKey
	localPubKey  crypto.PubKey
}

// NewService создаёт сервис синхронизации элементов
func NewService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey) *Service {
	return &Service{
		host:         host,
		localPrivKey: privKey,
		localPubKey:  pubKey,
	}
}

// Start запускает сервис синхронизации элементов
func (iss *Service) Start() error {
	iss.host.SetStreamHandler(ProtocolID, iss.handleItemRequest)
	return nil
}

// Stop останавливает сервис
func (iss *Service) Stop() error {
	return nil
}
