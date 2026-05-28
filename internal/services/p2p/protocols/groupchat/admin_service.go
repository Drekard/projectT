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

const AdminProtocolID = "/projectt/groupchat-admin/1.0.0"

type AdminAction string

const (
	ActionKickMember AdminAction = "kick"
	ActionBanMember  AdminAction = "ban"
	ActionChangeRole AdminAction = "change_role"
	ActionUpdateMeta AdminAction = "update_meta"
	ActionGrantAdmin AdminAction = "grant_admin"
)

type AdminRequest struct {
	GroupUUID    string      `json:"group_uuid"`
	Action       AdminAction `json:"action"`
	TargetPeerID string      `json:"target_peer_id"`
	NewRole      string      `json:"new_role,omitempty"`
	NewName      string      `json:"new_name,omitempty"`
	Description  string      `json:"description,omitempty"`
	AdminPeerID  string      `json:"admin_peer_id"`
	AdminSig     []byte      `json:"admin_sig"`
}

type AdminResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type AdminService struct {
	host        host.Host
	ctx         context.Context
	localPeerID string
	mu          sync.RWMutex
	onAdmin     func(req *AdminRequest, from peer.ID) *AdminResponse
}

func NewAdminService(ctx context.Context, h host.Host, localPeerID string) *AdminService {
	svc := &AdminService{
		host:        h,
		ctx:         ctx,
		localPeerID: localPeerID,
	}
	h.SetStreamHandler(AdminProtocolID, svc.handleAdminStream)
	return svc
}

func (s *AdminService) SetOnAdmin(handler func(req *AdminRequest, from peer.ID) *AdminResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onAdmin = handler
}

func (s *AdminService) SendAdminAction(ctx context.Context, peerID peer.ID, req *AdminRequest) (*AdminResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации: %w", err)
	}

	stream, err := s.host.NewStream(ctx, peerID, AdminProtocolID)
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

	var resp AdminResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("ошибка десериализации ответа: %w", err)
	}

	return &resp, nil
}

func (s *AdminService) handleAdminStream(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	reader := bufio.NewReader(stream)
	data, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	var req AdminRequest
	if err := json.Unmarshal(data, &req); err != nil {
		s.sendAdminResponse(stream, &AdminResponse{Success: false, Error: "invalid request"})
		return
	}

	s.mu.RLock()
	handler := s.onAdmin
	s.mu.RUnlock()

	var resp *AdminResponse
	if handler != nil {
		resp = handler(&req, stream.Conn().RemotePeer())
	} else {
		resp = &AdminResponse{Success: false, Error: "no handler"}
	}

	s.sendAdminResponse(stream, resp)
}

func (s *AdminService) sendAdminResponse(stream network.Stream, resp *AdminResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = stream.Write(data)
}
