package groupchat

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
)

const SyncProtocolID = "/projectt/groupchat-sync/1.0.0"

type SyncRequest struct {
	GroupUUID            string `json:"group_uuid"`
	LastKnownMessageUUID string `json:"last_known_message_uuid"`
	LastLamportClock     uint64 `json:"last_lamport_clock"`
	Count                int    `json:"count"`
}

type SyncService struct {
	host        host.Host
	ctx         context.Context
	localPeerID string
	mu          sync.RWMutex
	onSync      func(req *SyncRequest, from peer.ID) []*GroupMessage
}

func NewSyncService(ctx context.Context, h host.Host, localPeerID string) *SyncService {
	svc := &SyncService{
		host:        h,
		ctx:         ctx,
		localPeerID: localPeerID,
	}
	h.SetStreamHandler(SyncProtocolID, svc.handleSyncStream)
	return svc
}

func (s *SyncService) SetOnSync(handler func(req *SyncRequest, from peer.ID) []*GroupMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onSync = handler
}

func (s *SyncService) RequestSync(ctx context.Context, peerID peer.ID, req *SyncRequest) ([]*GroupMessage, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации: %w", err)
	}

	stream, err := s.host.NewStream(ctx, peerID, SyncProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}
	_ = stream.CloseWrite()

	var messages []*GroupMessage
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	for {
		var msg GroupMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return messages, fmt.Errorf("ошибка чтения сообщения: %w", err)
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

func (s *SyncService) handleSyncStream(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	reader := bufio.NewReader(stream)
	data, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	var req SyncRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return
	}

	s.mu.RLock()
	handler := s.onSync
	s.mu.RUnlock()

	var messages []*GroupMessage
	if handler != nil {
		messages = handler(&req, stream.Conn().RemotePeer())
	}

	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)
	for _, msg := range messages {
		if err := encoder.Encode(msg); err != nil {
			log.Printf("[GroupChat/Sync] ⚠️ Ошибка отправки сообщения: %v", err)
			break
		}
	}
	_ = writer.Flush()
}
