package groupchat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const InviteProtocolID = "/projectt/groupchat-invite/1.0.0"

type InviteRequest struct {
	GroupUUID     string `json:"group_uuid"`
	GroupName     string `json:"group_name"`
	ChatType      string `json:"chat_type"`
	InviteToken   string `json:"invite_token"`
	InviterPeerID string `json:"inviter_peer_id"`
	Depth         int    `json:"depth"`
	InviterSig    []byte `json:"inviter_sig"`
}

type InviteResponse struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

type InviteService struct {
	host        host.Host
	ctx         context.Context
	localPeerID string
	mu          sync.RWMutex
	onInvite    func(req *InviteRequest, from peer.ID) *InviteResponse
}

func NewInviteService(ctx context.Context, h host.Host, localPeerID string) *InviteService {
	svc := &InviteService{
		host:        h,
		ctx:         ctx,
		localPeerID: localPeerID,
	}
	h.SetStreamHandler(InviteProtocolID, svc.handleInviteStream)
	return svc
}

func (s *InviteService) SetOnInvite(handler func(req *InviteRequest, from peer.ID) *InviteResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onInvite = handler
}

func (s *InviteService) SendInvite(ctx context.Context, peerID peer.ID, req *InviteRequest) (*InviteResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации: %w", err)
	}

	stream, err := s.host.NewStream(ctx, peerID, InviteProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("ошибка отправки: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}
	_ = stream.CloseWrite()

	reader := bufio.NewReader(stream)
	respData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var resp InviteResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("ошибка десериализации ответа: %w", err)
	}

	return &resp, nil
}

func (s *InviteService) handleInviteStream(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	remotePeer := stream.Conn().RemotePeer()

	reader := bufio.NewReader(stream)
	data, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	var req InviteRequest
	if err := json.Unmarshal(data, &req); err != nil {
		s.sendInviteResponse(stream, &InviteResponse{Accepted: false, Reason: "invalid request"})
		return
	}

	s.mu.RLock()
	handler := s.onInvite
	s.mu.RUnlock()

	var resp *InviteResponse
	if handler != nil {
		resp = handler(&req, remotePeer)
	} else {
		resp = &InviteResponse{Accepted: false, Reason: "no handler"}
	}

	s.sendInviteResponse(stream, resp)
}

func (s *InviteService) sendInviteResponse(stream network.Stream, resp *InviteResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = stream.Write(data)
}
