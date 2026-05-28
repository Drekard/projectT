package groupchat

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type MembershipProof struct {
	GroupUUID string `json:"group_uuid"`
	PeerID    string `json:"peer_id"`
	Role      string `json:"role"`
	GrantedBy string `json:"granted_by"`
	Timestamp int64  `json:"timestamp"`
	AdminSig  []byte `json:"admin_sig"`
}

type MembershipVerifier struct {
	mu       sync.RWMutex
	cache    map[string]*proofEntry
	cacheTTL time.Duration
}

type proofEntry struct {
	proof     *MembershipProof
	expiresAt time.Time
}

func NewMembershipVerifier(cacheTTL time.Duration) *MembershipVerifier {
	if cacheTTL == 0 {
		cacheTTL = 30 * time.Minute
	}
	return &MembershipVerifier{
		cache:    make(map[string]*proofEntry),
		cacheTTL: cacheTTL,
	}
}

func (v *MembershipVerifier) VerifyProof(proof *MembershipProof, adminPubKey ed25519.PublicKey) error {
	if proof == nil {
		return fmt.Errorf("proof is nil")
	}
	if proof.GroupUUID == "" || proof.PeerID == "" || proof.Role == "" {
		return fmt.Errorf("invalid proof: missing required fields")
	}
	if len(proof.AdminSig) == 0 {
		return fmt.Errorf("invalid proof: empty signature")
	}

	data, err := json.Marshal(map[string]interface{}{
		"group_uuid": proof.GroupUUID,
		"peer_id":    proof.PeerID,
		"role":       proof.Role,
		"granted_by": proof.GrantedBy,
		"timestamp":  proof.Timestamp,
	})
	if err != nil {
		return fmt.Errorf("error serializing proof data: %w", err)
	}

	if !ed25519.Verify(adminPubKey, data, proof.AdminSig) {
		return fmt.Errorf("invalid signature")
	}

	cacheKey := proof.GroupUUID + ":" + proof.PeerID
	v.mu.Lock()
	v.cache[cacheKey] = &proofEntry{
		proof:     proof,
		expiresAt: time.Now().Add(v.cacheTTL),
	}
	v.mu.Unlock()

	return nil
}

func (v *MembershipVerifier) GetCachedProof(groupUUID, peerID string) *MembershipProof {
	v.mu.RLock()
	defer v.mu.RUnlock()

	cacheKey := groupUUID + ":" + peerID
	entry, exists := v.cache[cacheKey]
	if !exists {
		return nil
	}
	if time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.proof
}

func (v *MembershipVerifier) InvalidateProof(groupUUID, peerID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	cacheKey := groupUUID + ":" + peerID
	delete(v.cache, cacheKey)
}

func (v *MembershipVerifier) CreateProof(groupUUID, peerID, role, grantedBy string, adminPrivKey ed25519.PrivateKey) (*MembershipProof, error) {
	proof := &MembershipProof{
		GroupUUID: groupUUID,
		PeerID:    peerID,
		Role:      role,
		GrantedBy: grantedBy,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(map[string]interface{}{
		"group_uuid": proof.GroupUUID,
		"peer_id":    proof.PeerID,
		"role":       proof.Role,
		"granted_by": proof.GrantedBy,
		"timestamp":  proof.Timestamp,
	})
	if err != nil {
		return nil, fmt.Errorf("error serializing proof data: %w", err)
	}

	sig := ed25519.Sign(adminPrivKey, data)
	proof.AdminSig = sig

	return proof, nil
}

func PeerIDToPubKey(peerIDStr string) (ed25519.PublicKey, error) {
	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		return nil, fmt.Errorf("error decoding peer ID: %w", err)
	}

	pubKey, err := pid.ExtractPublicKey()
	if err != nil {
		return nil, fmt.Errorf("error extracting public key: %w", err)
	}

	rawBytes, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("error getting raw bytes: %w", err)
	}

	return ed25519.PublicKey(rawBytes), nil
}
