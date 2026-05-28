package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

func CreateGroupInvitation(invite *models.GroupInvitation) error {
	result, err := database.DB.Exec(`
		INSERT INTO group_invitations (group_uuid, invite_token, invited_by, target_peer_id, depth, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'pending', CURRENT_TIMESTAMP)
	`, invite.GroupUUID, invite.InviteToken, invite.InvitedBy, nullIfEmpty(invite.TargetPeerID), invite.Depth)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	invite.ID = int(id)
	return nil
}

func GetGroupInvitationByToken(token string) (*models.GroupInvitation, error) {
	row := database.DB.QueryRow(`
		SELECT id, group_uuid, invite_token, invited_by, target_peer_id, depth, status, created_at
		FROM group_invitations
		WHERE invite_token = ?
	`, token)

	invite := &models.GroupInvitation{}
	var targetPeerID sql.NullString
	var createdAt string
	err := row.Scan(&invite.ID, &invite.GroupUUID, &invite.InviteToken, &invite.InvitedBy, &targetPeerID, &invite.Depth, &invite.Status, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if targetPeerID.Valid {
		invite.TargetPeerID = targetPeerID.String
	}
	invite.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return invite, nil
}

func UpdateInvitationStatus(token, status, targetPeerID string) error {
	_, err := database.DB.Exec(`
		UPDATE group_invitations SET status = ?, target_peer_id = ?
		WHERE invite_token = ?
	`, status, targetPeerID, token)
	return err
}

func GetPendingInvitations(groupUUID string) ([]*models.GroupInvitation, error) {
	rows, err := database.DB.Query(`
		SELECT id, group_uuid, invite_token, invited_by, target_peer_id, depth, status, created_at
		FROM group_invitations
		WHERE group_uuid = ? AND status = 'pending'
	`, groupUUID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var invites []*models.GroupInvitation
	for rows.Next() {
		invite := &models.GroupInvitation{}
		var targetPeerID sql.NullString
		var createdAt string
		if err := rows.Scan(&invite.ID, &invite.GroupUUID, &invite.InviteToken, &invite.InvitedBy, &targetPeerID, &invite.Depth, &invite.Status, &createdAt); err != nil {
			return nil, err
		}
		if targetPeerID.Valid {
			invite.TargetPeerID = targetPeerID.String
		}
		invite.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
