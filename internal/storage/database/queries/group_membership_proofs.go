package queries

import (
	"database/sql"
	"errors"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

func UpsertGroupMembershipProof(proof *models.GroupMembershipProof) error {
	_, err := database.DB.Exec(`
		INSERT INTO group_membership_proofs (group_uuid, peer_id, role, granted_by, timestamp, admin_signature)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(group_uuid, peer_id) DO UPDATE SET
			role = excluded.role,
			granted_by = excluded.granted_by,
			timestamp = excluded.timestamp,
			admin_signature = excluded.admin_signature
	`, proof.GroupUUID, proof.PeerID, proof.Role, proof.GrantedBy, proof.Timestamp, proof.AdminSig)
	return err
}

func GetGroupMembershipProof(groupUUID, peerID string) (*models.GroupMembershipProof, error) {
	row := database.DB.QueryRow(`
		SELECT id, group_uuid, peer_id, role, granted_by, timestamp, admin_signature
		FROM group_membership_proofs
		WHERE group_uuid = ? AND peer_id = ?
	`, groupUUID, peerID)

	proof := &models.GroupMembershipProof{}
	err := row.Scan(&proof.ID, &proof.GroupUUID, &proof.PeerID, &proof.Role, &proof.GrantedBy, &proof.Timestamp, &proof.AdminSig)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return proof, nil
}

func DeleteGroupMembershipProof(groupUUID, peerID string) error {
	_, err := database.DB.Exec(`
		DELETE FROM group_membership_proofs WHERE group_uuid = ? AND peer_id = ?
	`, groupUUID, peerID)
	return err
}
