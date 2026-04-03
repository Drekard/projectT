// Package peerexchange предоставляет сервис для обмена списками пиров между пирами
package peerexchange

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ProtocolID идентификатор протокола обмена пирами
const ProtocolID = "/projectt/peer-exchange/1.0.0"

// PeerData данные о пире для обмена
type PeerData struct {
	PeerID      string `json:"peer_id"`
	Multiaddr   string `json:"multiaddr"`
	AddressType string `json:"address_type"` // bootstrap, discovered (contact НЕ передаётся)
	Username    string `json:"username,omitempty"`
}

// ExchangeService сервис для обмена списками пиров
type ExchangeService struct {
	mu       sync.RWMutex
	host     host.Host
	provider PeerProvider
	maxPeers int
}

// PeerProvider интерфейс для получения и сохранения пиров
type PeerProvider interface {
	GetKnownPeersForExchange(excludePeerID string, limit int) ([]*models.PeerAddress, error)
	AddPeerAddressWithProfile(peerID, multiaddr, addressType, source, username string) error
	IsKnownPeer(peerID string) bool
}

// DefaultProvider провайдер по умолчанию (работает с БД)
type DefaultProvider struct{}

// GetKnownPeersForExchange возвращает пиров для обмена
func (p *DefaultProvider) GetKnownPeersForExchange(excludePeerID string, limit int) ([]*models.PeerAddress, error) {
	return queries.GetKnownPeersForExchange(excludePeerID, limit)
}

// AddPeerAddressWithProfile добавляет адрес пира
func (p *DefaultProvider) AddPeerAddressWithProfile(peerID, multiaddr, addressType, source, username string) error {
	return queries.AddPeerAddressWithProfile(peerID, multiaddr, addressType, source, username)
}

// IsKnownPeer проверяет, известен ли пир
func (p *DefaultProvider) IsKnownPeer(peerID string) bool {
	// Проверяем, есть ли пир в БД
	addresses, err := queries.GetActivePeerAddresses()
	if err != nil {
		return false
	}

	for _, addr := range addresses {
		if addr.PeerID == peerID {
			return true
		}
	}
	return false
}

// NewExchangeService создаёт сервис обмена пирами
func NewExchangeService(h host.Host, provider PeerProvider) *ExchangeService {
	if provider == nil {
		provider = &DefaultProvider{}
	}

	return &ExchangeService{
		host:     h,
		provider: provider,
		maxPeers: 20, // Максимум 20 пиров за обмен
	}
}

// Start запускает сервис
func (s *ExchangeService) Start() error {
	s.host.SetStreamHandler(ProtocolID, s.handleExchange)
	log.Println("[PeerExchange] Сервис обмена пирами запущен")
	return nil
}

// Stop останавливает сервис
func (s *ExchangeService) Stop() error {
	log.Println("[PeerExchange] Сервис обмена пирами остановлен")
	return nil
}

// handleExchange обрабатывает входящий запрос обмена
func (s *ExchangeService) handleExchange(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("[PeerExchange] 📥 Запрос обмена от %s", remotePeer.String()[:8])

	// Читаем наши пиры для обмена (кроме контактов!)
	ourPeers, err := s.provider.GetKnownPeersForExchange(remotePeer.String(), s.maxPeers)
	if err != nil {
		log.Printf("[PeerExchange] ❌ Ошибка получения пиров: %v", err)
		return
	}

	log.Printf("[PeerExchange] 📤 Отправка %d пиров", len(ourPeers))

	// Отправляем наши пиры
	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	for _, p := range ourPeers {
		peerData := PeerData{
			PeerID:      p.PeerID,
			Multiaddr:   p.Multiaddr,
			AddressType: p.AddressType,
			Username:    p.Username,
		}

		if err := encoder.Encode(peerData); err != nil {
			log.Printf("[PeerExchange] ❌ Ошибка отправки пира %s: %v", p.PeerID[:8], err)
			break
		}
	}

	if err := writer.Flush(); err != nil {
		log.Printf("[PeerExchange] ❌ Ошибка flush: %v", err)
		return
	}

	// Читаем пиры от удалённого пира
	log.Printf("[PeerExchange] 📥 Чтение пиров от %s", remotePeer.String()[:8])
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	receivedCount := 0
	for {
		var peerData PeerData
		if err := decoder.Decode(&peerData); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("[PeerExchange] ⚠️ Ошибка чтения пира: %v", err)
			break
		}

		// Пропускаем контакты (не должны приходить, но на всякий случай)
		if peerData.AddressType == "contact" {
			log.Printf("[PeerExchange] ⚠️ Пропускаем контакт (не должен передаваться)")
			continue
		}

		// Проверяем, не знаем ли уже этого пира
		if s.provider.IsKnownPeer(peerData.PeerID) {
			log.Printf("[PeerExchange] ℹ️ Пир %s уже известен", peerData.PeerID[:8])
			continue
		}

		// Добавляем пира в БД
		err := s.provider.AddPeerAddressWithProfile(
			peerData.PeerID,
			peerData.Multiaddr,
			"discovered", // Новые пиры всегда discovered
			"peer_exchange",
			peerData.Username,
		)

		if err != nil {
			log.Printf("[PeerExchange] ❌ Ошибка добавления пира %s: %v", peerData.PeerID[:8], err)
			continue
		}

		receivedCount++
		log.Printf("[PeerExchange] ✅ Добавлен пир %s (%s)", peerData.PeerID[:8], peerData.AddressType)
	}

	log.Printf("[PeerExchange] 📊 Получено %d новых пиров от %s", receivedCount, remotePeer.String()[:8])
}

// ExchangeWithPeer обменивается пирами с удалённым пиром
func (s *ExchangeService) ExchangeWithPeer(ctx context.Context, peerID peer.ID) error {
	log.Printf("[PeerExchange] 🔌 Обмен с пиром %s", peerID.String()[:8])

	stream, err := s.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

	// Читаем пиры от удалённого пира
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	receivedCount := 0
	for {
		var peerData PeerData
		if err := decoder.Decode(&peerData); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("[PeerExchange] ⚠️ Ошибка чтения пира: %v", err)
			break
		}

		// Пропускаем контакты
		if peerData.AddressType == "contact" {
			continue
		}

		// Проверяем, не знаем ли уже этого пира
		if s.provider.IsKnownPeer(peerData.PeerID) {
			continue
		}

		// Добавляем пира в БД
		err := s.provider.AddPeerAddressWithProfile(
			peerData.PeerID,
			peerData.Multiaddr,
			"discovered",
			"peer_exchange",
			peerData.Username,
		)

		if err != nil {
			log.Printf("[PeerExchange] ❌ Ошибка добавления пира %s: %v", peerData.PeerID[:8], err)
			continue
		}

		receivedCount++
	}

	log.Printf("[PeerExchange] 📊 Получено %d новых пиров", receivedCount)
	return nil
}

// SetMaxPeers устанавливает максимальное количество пиров для обмена
func (s *ExchangeService) SetMaxPeers(max int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxPeers = max
}

// GetMaxPeers возвращает максимальное количество пиров
func (s *ExchangeService) GetMaxPeers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxPeers
}
