package groupchat

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

type Event struct {
	Type       string
	GroupUUID  string
	Message    *models.GroupMessage
	Member     *models.GroupMember
	Invitation *models.GroupInvitation
	Error      error
}

const (
	EventNewMessage    = "new_message"
	EventMemberJoined  = "member_joined"
	EventMemberLeft    = "member_left"
	EventMemberKicked  = "member_kicked"
	EventMemberBanned  = "member_banned"
	EventRoleChanged   = "role_changed"
	EventMetaUpdated   = "meta_updated"
	EventInviteCreated = "invite_created"
	EventError         = "error"
)

type Service struct {
	host        host.Host
	pubsub      *pubsub.PubSub
	ctx         context.Context
	localPeerID string
	privKey     ed25519.PrivateKey
	pubKey      ed25519.PublicKey

	pubsubMgr *PubSubManager
	inviteSvc *InviteService
	syncSvc   *SyncService
	adminSvc  *AdminService
	verifier  *MembershipVerifier

	mu          sync.RWMutex
	subscribers []chan *Event

	lamportMu     sync.Mutex
	lamportClocks map[string]uint64
}

func NewService(ctx context.Context, h host.Host, ps *pubsub.PubSub, localPeerID string, privKey ed25519.PrivateKey) *Service {
	svc := &Service{
		ctx:           ctx,
		host:          h,
		pubsub:        ps,
		localPeerID:   localPeerID,
		privKey:       privKey,
		pubKey:        privKey.Public().(ed25519.PublicKey),
		verifier:      NewMembershipVerifier(30 * time.Minute),
		lamportClocks: make(map[string]uint64),
	}

	svc.pubsubMgr = NewPubSubManager(ctx, h, ps, localPeerID)
	svc.inviteSvc = NewInviteService(ctx, h, localPeerID)
	svc.syncSvc = NewSyncService(ctx, h, localPeerID)
	svc.adminSvc = NewAdminService(ctx, h, localPeerID)

	svc.setupHandlers()

	return svc
}

func (s *Service) setupHandlers() {
	s.pubsubMgr.JoinGroupCallback = func(groupUUID string, msg *GroupMessage) {
		s.handleIncomingMessage(groupUUID, msg)
	}

	s.inviteSvc.SetOnInvite(func(req *InviteRequest, from peer.ID) *InviteResponse {
		return s.handleIncomingInvite(req, from)
	})

	s.syncSvc.SetOnSync(func(req *SyncRequest, from peer.ID) []*GroupMessage {
		return s.handleSyncRequest(req, from)
	})

	s.adminSvc.SetOnAdmin(func(req *AdminRequest, from peer.ID) *AdminResponse {
		return s.handleAdminAction(req, from)
	})
}

func (s *Service) CreateGroup(name, description, chatType string, maxInviteDepth int) (*models.GroupChat, error) {
	groupUUID := uuid.New().String()

	chat := &models.GroupChat{
		GroupUUID:      groupUUID,
		Name:           name,
		Description:    description,
		CreatorPeerID:  s.localPeerID,
		ChatType:       chatType,
		MaxInviteDepth: maxInviteDepth,
	}

	if err := queries.CreateGroupChat(chat); err != nil {
		return nil, fmt.Errorf("ошибка создания группы: %w", err)
	}

	member := &models.GroupMember{
		GroupUUID:   groupUUID,
		PeerID:      s.localPeerID,
		Role:        "creator",
		InvitedBy:   s.localPeerID,
		InviteDepth: maxInviteDepth,
	}
	if err := queries.AddGroupMember(member); err != nil {
		return nil, fmt.Errorf("ошибка добавления создателя: %w", err)
	}

	proof, err := s.verifier.CreateProof(groupUUID, s.localPeerID, "creator", s.localPeerID, s.privKey)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания proof: %w", err)
	}
	dbProof := &models.GroupMembershipProof{
		GroupUUID: proof.GroupUUID,
		PeerID:    proof.PeerID,
		Role:      proof.Role,
		GrantedBy: proof.GrantedBy,
		Timestamp: proof.Timestamp,
		AdminSig:  proof.AdminSig,
	}
	if err := queries.UpsertGroupMembershipProof(dbProof); err != nil {
		return nil, fmt.Errorf("ошибка сохранения proof: %w", err)
	}

	if err := s.pubsubMgr.JoinGroup(groupUUID, func(msg *GroupMessage) {
		s.handleIncomingMessage(groupUUID, msg)
	}); err != nil {
		log.Printf("[GroupChat] ⚠️ Ошибка подписки на группу: %v", err)
	}

	s.lamportMu.Lock()
	s.lamportClocks[groupUUID] = 0
	s.lamportMu.Unlock()

	s.emit(&Event{Type: EventMemberJoined, GroupUUID: groupUUID, Member: member})

	log.Printf("[GroupChat] ✅ Группа создана: %s (%s)", name, groupUUID[:8])
	return chat, nil
}

func (s *Service) JoinGroupViaInvite(inviteToken string) (*models.GroupChat, error) {
	invite, err := queries.GetGroupInvitationByToken(inviteToken)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения инвайта: %w", err)
	}
	if invite == nil {
		return nil, fmt.Errorf("инвайт не найден")
	}
	if invite.Status != "pending" {
		return nil, fmt.Errorf("инвайт уже использован (статус: %s)", invite.Status)
	}

	blocked, err := queries.IsGroupBlocked(invite.GroupUUID, s.localPeerID)
	if err != nil || blocked {
		return nil, fmt.Errorf("вы заблокированы в этой группе")
	}

	alreadyMember, err := queries.IsGroupMember(invite.GroupUUID, s.localPeerID)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки членства: %w", err)
	}
	if alreadyMember {
		return nil, fmt.Errorf("вы уже участник этой группы")
	}

	groupChat, err := queries.GetGroupChatByUUID(invite.GroupUUID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения группы: %w", err)
	}
	if groupChat == nil {
		return nil, fmt.Errorf("группа не найдена")
	}

	newDepth := invite.Depth - 1
	if newDepth < 0 {
		newDepth = 0
	}

	member := &models.GroupMember{
		GroupUUID:   invite.GroupUUID,
		PeerID:      s.localPeerID,
		Role:        "member",
		InvitedBy:   invite.InvitedBy,
		InviteDepth: newDepth,
	}
	if err := queries.AddGroupMember(member); err != nil {
		return nil, fmt.Errorf("ошибка добавления участника: %w", err)
	}

	if err := queries.UpdateInvitationStatus(inviteToken, "accepted", s.localPeerID); err != nil {
		log.Printf("[GroupChat] ⚠️ Ошибка обновления статуса инвайта: %v", err)
	}

	if err := s.pubsubMgr.JoinGroup(invite.GroupUUID, func(msg *GroupMessage) {
		s.handleIncomingMessage(invite.GroupUUID, msg)
	}); err != nil {
		log.Printf("[GroupChat] ⚠️ Ошибка подписки на группу: %v", err)
	}

	s.syncHistory(invite.GroupUUID)

	s.emit(&Event{Type: EventMemberJoined, GroupUUID: invite.GroupUUID, Member: member})

	log.Printf("[GroupChat] ✅ Присоединился к группе: %s", groupChat.Name)
	return groupChat, nil
}

func (s *Service) SendMessage(groupUUID, content, contentType string, metadata map[string]string) error {
	isMember, err := queries.IsGroupMember(groupUUID, s.localPeerID)
	if err != nil || !isMember {
		return fmt.Errorf("вы не участник этой группы")
	}

	groupChat, err := queries.GetGroupChatByUUID(groupUUID)
	if err != nil {
		return fmt.Errorf("ошибка получения группы: %w", err)
	}
	if groupChat == nil {
		return fmt.Errorf("группа не найдена")
	}

	if groupChat.ChatType == "channel" {
		member, _ := queries.GetGroupMember(groupUUID, s.localPeerID)
		if member == nil || (member.Role != "creator" && member.Role != "admin") {
			return fmt.Errorf("только администраторы могут публиковать сообщения в канале")
		}
	}

	s.lamportMu.Lock()
	s.lamportClocks[groupUUID]++
	lamport := s.lamportClocks[groupUUID]
	s.lamportMu.Unlock()

	metaJSON := ""
	if metadata != nil {
		data, _ := json.Marshal(metadata)
		metaJSON = string(data)
	}

	msgUUID := uuid.New().String()
	msg := &models.GroupMessage{
		GroupUUID:    groupUUID,
		MessageUUID:  msgUUID,
		FromPeerID:   s.localPeerID,
		Content:      content,
		ContentType:  contentType,
		Metadata:     metaJSON,
		Timestamp:    time.Now().UnixNano(),
		LamportClock: lamport,
	}

	dataToSign := fmt.Sprintf("%s:%s:%s:%d", msg.GroupUUID, msg.MessageUUID, msg.Content, msg.Timestamp)
	sig := ed25519.Sign(s.privKey, []byte(dataToSign))
	msg.Signature = sig

	if err := queries.CreateGroupMessage(msg); err != nil {
		isDuplicate, _ := queries.IsDuplicateGroupMessage(msgUUID)
		if isDuplicate {
			return nil
		}
		return fmt.Errorf("ошибка сохранения сообщения: %w", err)
	}

	pubsubMsg := &GroupMessage{
		MessageUUID:  msg.MessageUUID,
		GroupUUID:    msg.GroupUUID,
		FromPeerID:   msg.FromPeerID,
		Content:      msg.Content,
		ContentType:  msg.ContentType,
		Metadata:     msg.Metadata,
		Timestamp:    msg.Timestamp,
		LamportClock: msg.LamportClock,
		Signature:    msg.Signature,
	}

	if err := s.pubsubMgr.PublishMessage(groupUUID, pubsubMsg); err != nil {
		log.Printf("[GroupChat] ⚠️ Ошибка публикации сообщения: %v", err)
	}

	s.emit(&Event{Type: EventNewMessage, GroupUUID: groupUUID, Message: msg})

	return nil
}

func (s *Service) CreateInvite(groupUUID string, depth int) (*models.GroupInvitation, error) {
	member, err := queries.GetGroupMember(groupUUID, s.localPeerID)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки членства: %w", err)
	}
	if member == nil {
		return nil, fmt.Errorf("вы не участник этой группы")
	}
	if member.InviteDepth <= 0 {
		return nil, fmt.Errorf("лимит приглашений исчерпан")
	}

	inviteDepth := depth
	if inviteDepth > member.InviteDepth-1 {
		inviteDepth = member.InviteDepth - 1
	}
	if inviteDepth < 0 {
		inviteDepth = 0
	}

	token := uuid.New().String()
	invite := &models.GroupInvitation{
		GroupUUID:   groupUUID,
		InviteToken: token,
		InvitedBy:   s.localPeerID,
		Depth:       inviteDepth,
	}

	if err := queries.CreateGroupInvitation(invite); err != nil {
		return nil, fmt.Errorf("ошибка создания инвайта: %w", err)
	}

	s.emit(&Event{Type: EventInviteCreated, GroupUUID: groupUUID, Invitation: invite})

	return invite, nil
}

func (s *Service) KickMember(groupUUID, targetPeerID string) error {
	return s.performAdminAction(groupUUID, targetPeerID, "kick", "")
}

func (s *Service) BanMember(groupUUID, targetPeerID, reason string) error {
	return s.performAdminAction(groupUUID, targetPeerID, "ban", reason)
}

func (s *Service) ChangeRole(groupUUID, targetPeerID, newRole string) error {
	return s.performAdminAction(groupUUID, targetPeerID, "change_role", newRole)
}

func (s *Service) performAdminAction(groupUUID, targetPeerID, action, extra string) error {
	member, err := queries.GetGroupMember(groupUUID, s.localPeerID)
	if err != nil {
		return fmt.Errorf("ошибка проверки членства: %w", err)
	}
	if member == nil {
		return fmt.Errorf("вы не участник этой группы")
	}
	if member.Role != "creator" && member.Role != "admin" {
		return fmt.Errorf("недостаточно прав")
	}

	proof, err := s.verifier.CreateProof(groupUUID, targetPeerID, extra, s.localPeerID, s.privKey)
	if err != nil {
		return fmt.Errorf("ошибка создания proof: %w", err)
	}

	adminReq := &AdminRequest{
		GroupUUID:    groupUUID,
		Action:       AdminAction(action),
		TargetPeerID: targetPeerID,
		NewRole:      extra,
		AdminPeerID:  s.localPeerID,
		AdminSig:     proof.AdminSig,
	}

	members, _ := queries.GetActiveGroupMembers(groupUUID)
	for _, m := range members {
		if m.PeerID == s.localPeerID {
			continue
		}
		peerID, err := peer.Decode(m.PeerID)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		_, _ = s.adminSvc.SendAdminAction(ctx, peerID, adminReq)
		cancel()
	}

	s.executeAdminAction(groupUUID, targetPeerID, action, extra)

	return nil
}

func (s *Service) executeAdminAction(groupUUID, targetPeerID, action, extra string) {
	switch action {
	case "kick":
		_ = queries.RemoveGroupMember(groupUUID, targetPeerID)
		s.verifier.InvalidateProof(groupUUID, targetPeerID)
		s.emit(&Event{Type: EventMemberKicked, GroupUUID: groupUUID})
	case "ban":
		_ = queries.RemoveGroupMember(groupUUID, targetPeerID)
		block := &models.GroupBlock{
			GroupUUID: groupUUID,
			PeerID:    targetPeerID,
			BlockedBy: s.localPeerID,
			Reason:    extra,
		}
		_ = queries.BlockGroupMember(block)
		s.verifier.InvalidateProof(groupUUID, targetPeerID)
		s.emit(&Event{Type: EventMemberBanned, GroupUUID: groupUUID})
	case "change_role":
		_ = queries.UpdateGroupMemberRole(groupUUID, targetPeerID, extra)
		proof, _ := s.verifier.CreateProof(groupUUID, targetPeerID, extra, s.localPeerID, s.privKey)
		dbProof := &models.GroupMembershipProof{
			GroupUUID: proof.GroupUUID,
			PeerID:    proof.PeerID,
			Role:      proof.Role,
			GrantedBy: proof.GrantedBy,
			Timestamp: proof.Timestamp,
			AdminSig:  proof.AdminSig,
		}
		_ = queries.UpsertGroupMembershipProof(dbProof)
		s.emit(&Event{Type: EventRoleChanged, GroupUUID: groupUUID})
	}
}

func (s *Service) GetGroupChats() ([]*models.GroupChat, error) {
	return queries.GetGroupChatsByPeerID(s.localPeerID)
}

func (s *Service) GetGroupMessages(groupUUID string, limit, offset int) ([]*models.GroupMessage, error) {
	return queries.GetGroupMessages(groupUUID, limit, offset)
}

func (s *Service) GetGroupMembers(groupUUID string) ([]*models.GroupMember, error) {
	return queries.GetActiveGroupMembers(groupUUID)
}

func (s *Service) LeaveGroup(groupUUID string) error {
	member, err := queries.GetGroupMember(groupUUID, s.localPeerID)
	if err != nil {
		return fmt.Errorf("ошибка проверки членства: %w", err)
	}
	if member == nil {
		return fmt.Errorf("вы не участник этой группы")
	}
	if member.Role == "creator" {
		return fmt.Errorf("создатель не может покинуть группу")
	}

	if err := queries.RemoveGroupMember(groupUUID, s.localPeerID); err != nil {
		return fmt.Errorf("ошибка выхода из группы: %w", err)
	}

	_ = s.pubsubMgr.LeaveGroup(groupUUID)

	s.emit(&Event{Type: EventMemberLeft, GroupUUID: groupUUID})

	return nil
}

func (s *Service) Subscribe() <-chan *Event {
	ch := make(chan *Event, 100)
	s.mu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.mu.Unlock()
	return ch
}

func (s *Service) emit(event *Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Service) handleIncomingMessage(groupUUID string, msg *GroupMessage) {
	isDuplicate, _ := queries.IsDuplicateGroupMessage(msg.MessageUUID)
	if isDuplicate {
		return
	}

	blocked, _ := queries.IsGroupBlocked(groupUUID, msg.FromPeerID)
	if blocked {
		return
	}

	isMember, _ := queries.IsGroupMember(groupUUID, msg.FromPeerID)
	if !isMember {
		return
	}

	groupChat, _ := queries.GetGroupChatByUUID(groupUUID)
	if groupChat == nil {
		return
	}

	if groupChat.ChatType == "channel" {
		member, _ := queries.GetGroupMember(groupUUID, msg.FromPeerID)
		if member != nil && member.Role == "subscriber" {
			return
		}
	}

	dbMsg := &models.GroupMessage{
		GroupUUID:    msg.GroupUUID,
		MessageUUID:  msg.MessageUUID,
		FromPeerID:   msg.FromPeerID,
		Content:      msg.Content,
		ContentType:  msg.ContentType,
		Metadata:     msg.Metadata,
		Timestamp:    msg.Timestamp,
		LamportClock: msg.LamportClock,
		Signature:    msg.Signature,
	}

	if err := queries.CreateGroupMessage(dbMsg); err != nil {
		return
	}

	s.lamportMu.Lock()
	if msg.LamportClock > s.lamportClocks[groupUUID] {
		s.lamportClocks[groupUUID] = msg.LamportClock
	}
	s.lamportMu.Unlock()

	s.emit(&Event{Type: EventNewMessage, GroupUUID: groupUUID, Message: dbMsg})
}

func (s *Service) handleIncomingInvite(req *InviteRequest, from peer.ID) *InviteResponse {
	blocked, _ := queries.IsGroupBlocked(req.GroupUUID, s.localPeerID)
	if blocked {
		return &InviteResponse{Accepted: false, Reason: "you are blocked"}
	}

	alreadyMember, _ := queries.IsGroupMember(req.GroupUUID, s.localPeerID)
	if alreadyMember {
		return &InviteResponse{Accepted: false, Reason: "already a member"}
	}

	return &InviteResponse{Accepted: true}
}

func (s *Service) handleSyncRequest(req *SyncRequest, from peer.ID) []*GroupMessage {
	isMember, _ := queries.IsGroupMember(req.GroupUUID, s.localPeerID)
	if !isMember {
		return nil
	}

	var dbMessages []*models.GroupMessage
	var err error

	if req.LastKnownMessageUUID != "" {
		dbMessages, err = queries.GetGroupMessagesAfterUUID(req.GroupUUID, req.LastKnownMessageUUID, req.Count)
	} else if req.LastLamportClock > 0 {
		dbMessages, err = queries.GetGroupMessagesAfterLamport(req.GroupUUID, req.LastLamportClock, req.Count)
	} else {
		dbMessages, err = queries.GetGroupMessages(req.GroupUUID, req.Count, 0)
	}

	if err != nil {
		return nil
	}

	var messages []*GroupMessage
	for _, dbMsg := range dbMessages {
		messages = append(messages, &GroupMessage{
			MessageUUID:  dbMsg.MessageUUID,
			GroupUUID:    dbMsg.GroupUUID,
			FromPeerID:   dbMsg.FromPeerID,
			Content:      dbMsg.Content,
			ContentType:  dbMsg.ContentType,
			Metadata:     dbMsg.Metadata,
			Timestamp:    dbMsg.Timestamp,
			LamportClock: dbMsg.LamportClock,
			Signature:    dbMsg.Signature,
		})
	}

	return messages
}

func (s *Service) handleAdminAction(req *AdminRequest, from peer.ID) *AdminResponse {
	proof := &MembershipProof{
		GroupUUID: req.GroupUUID,
		PeerID:    req.AdminPeerID,
		Role:      "admin",
		GrantedBy: req.AdminPeerID,
		Timestamp: time.Now().Unix(),
		AdminSig:  req.AdminSig,
	}

	adminPubKey, err := PeerIDToPubKey(req.AdminPeerID)
	if err != nil {
		return &AdminResponse{Success: false, Error: "invalid admin peer ID"}
	}

	if err := s.verifier.VerifyProof(proof, adminPubKey); err != nil {
		return &AdminResponse{Success: false, Error: "invalid admin proof"}
	}

	s.executeAdminAction(req.GroupUUID, req.TargetPeerID, string(req.Action), req.NewRole)

	return &AdminResponse{Success: true}
}

func (s *Service) syncHistory(groupUUID string) {
	members, err := queries.GetActiveGroupMembers(groupUUID)
	if err != nil {
		return
	}

	for _, m := range members {
		if m.PeerID == s.localPeerID {
			continue
		}

		peerID, err := peer.Decode(m.PeerID)
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
		messages, err := s.syncSvc.RequestSync(ctx, peerID, &SyncRequest{
			GroupUUID: groupUUID,
			Count:     1000,
		})
		cancel()

		if err != nil {
			continue
		}

		for _, msg := range messages {
			isDuplicate, _ := queries.IsDuplicateGroupMessage(msg.MessageUUID)
			if isDuplicate {
				continue
			}

			dbMsg := &models.GroupMessage{
				GroupUUID:    msg.GroupUUID,
				MessageUUID:  msg.MessageUUID,
				FromPeerID:   msg.FromPeerID,
				Content:      msg.Content,
				ContentType:  msg.ContentType,
				Metadata:     msg.Metadata,
				Timestamp:    msg.Timestamp,
				LamportClock: msg.LamportClock,
				Signature:    msg.Signature,
			}
			_ = queries.CreateGroupMessage(dbMsg)
		}

		if len(messages) > 0 {
			break
		}
	}
}
