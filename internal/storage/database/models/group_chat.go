package models

import "time"

// GroupChat представляет групповой чат или канал
type GroupChat struct {
	ID             int       `json:"id"`
	GroupUUID      string    `json:"group_uuid"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CreatorPeerID  string    `json:"creator_peer_id"`
	AvatarHash     string    `json:"avatar_hash"`
	ChatType       string    `json:"chat_type"` // "group" или "channel"
	MaxInviteDepth int       `json:"max_invite_depth"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// GroupMember представляет участника группового чата
type GroupMember struct {
	ID          int        `json:"id"`
	GroupUUID   string     `json:"group_uuid"`
	PeerID      string     `json:"peer_id"`
	Role        string     `json:"role"` // "creator", "admin", "moderator", "member", "subscriber"
	InvitedBy   string     `json:"invited_by"`
	InviteDepth int        `json:"invite_depth"`
	JoinedAt    time.Time  `json:"joined_at"`
	LeftAt      *time.Time `json:"left_at"` // NULL если активен
}

// GroupMessage представляет сообщение в групповом чате
type GroupMessage struct {
	ID           int    `json:"id"`
	GroupUUID    string `json:"group_uuid"`
	MessageUUID  string `json:"message_uuid"`
	FromPeerID   string `json:"from_peer_id"`
	Content      string `json:"content"`
	ContentType  string `json:"content_type"` // "text" или "element"
	Metadata     string `json:"metadata"`
	Timestamp    int64  `json:"timestamp"` // Unix nanoseconds
	LamportClock uint64 `json:"lamport_clock"`
	Signature    []byte `json:"signature"`
}

// GroupMessageRead отслеживает прочитанные сообщения в групповых чатах
type GroupMessageRead struct {
	ID          int       `json:"id"`
	GroupUUID   string    `json:"group_uuid"`
	MessageUUID string    `json:"message_uuid"`
	PeerID      string    `json:"peer_id"`
	ReadAt      time.Time `json:"read_at"`
}

// GroupMembershipProof представляет подписанное доказательство членства
type GroupMembershipProof struct {
	ID        int    `json:"id"`
	GroupUUID string `json:"group_uuid"`
	PeerID    string `json:"peer_id"`
	Role      string `json:"role"`
	GrantedBy string `json:"granted_by"`
	Timestamp int64  `json:"timestamp"`
	AdminSig  []byte `json:"admin_sig"`
}

// GroupInvitation представляет приглашение в группу
type GroupInvitation struct {
	ID           int       `json:"id"`
	GroupUUID    string    `json:"group_uuid"`
	InviteToken  string    `json:"invite_token"`
	InvitedBy    string    `json:"invited_by"`
	TargetPeerID string    `json:"target_peer_id"` // NULL если ссылка ещё не использована
	Depth        int       `json:"depth"`
	Status       string    `json:"status"` // "pending", "accepted", "rejected", "expired"
	CreatedAt    time.Time `json:"created_at"`
}

// GroupBlock представляет заблокированного участника
type GroupBlock struct {
	ID        int       `json:"id"`
	GroupUUID string    `json:"group_uuid"`
	PeerID    string    `json:"peer_id"`
	BlockedBy string    `json:"blocked_by"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}
