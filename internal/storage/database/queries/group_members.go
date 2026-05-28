package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

func AddGroupMember(member *models.GroupMember) error {
	result, err := database.DB.Exec(`
		INSERT INTO group_members (group_uuid, peer_id, role, invited_by, invite_depth, joined_at, left_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, NULL)
	`, member.GroupUUID, member.PeerID, member.Role, member.InvitedBy, member.InviteDepth)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	member.ID = int(id)
	return nil
}

func GetGroupMember(groupUUID, peerID string) (*models.GroupMember, error) {
	row := database.DB.QueryRow(`
		SELECT id, group_uuid, peer_id, role, invited_by, invite_depth, joined_at, left_at
		FROM group_members
		WHERE group_uuid = ? AND peer_id = ?
	`, groupUUID, peerID)

	member := &models.GroupMember{}
	var joinedAt sql.NullString
	var leftAt sql.NullString
	err := row.Scan(&member.ID, &member.GroupUUID, &member.PeerID, &member.Role, &member.InvitedBy, &member.InviteDepth, &joinedAt, &leftAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if joinedAt.Valid {
		member.JoinedAt, _ = time.Parse("2006-01-02 15:04:05", joinedAt.String)
	}
	if leftAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", leftAt.String)
		member.LeftAt = &t
	}
	return member, nil
}

func GetGroupMembers(groupUUID string) ([]*models.GroupMember, error) {
	rows, err := database.DB.Query(`
		SELECT id, group_uuid, peer_id, role, invited_by, invite_depth, joined_at, left_at
		FROM group_members
		WHERE group_uuid = ?
		ORDER BY joined_at ASC
	`, groupUUID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var members []*models.GroupMember
	for rows.Next() {
		member := &models.GroupMember{}
		var joinedAt sql.NullString
		var leftAt sql.NullString
		if err := rows.Scan(&member.ID, &member.GroupUUID, &member.PeerID, &member.Role, &member.InvitedBy, &member.InviteDepth, &joinedAt, &leftAt); err != nil {
			return nil, err
		}
		if joinedAt.Valid {
			member.JoinedAt, _ = time.Parse("2006-01-02 15:04:05", joinedAt.String)
		}
		if leftAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", leftAt.String)
			member.LeftAt = &t
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func GetActiveGroupMembers(groupUUID string) ([]*models.GroupMember, error) {
	rows, err := database.DB.Query(`
		SELECT id, group_uuid, peer_id, role, invited_by, invite_depth, joined_at, left_at
		FROM group_members
		WHERE group_uuid = ? AND left_at IS NULL
		ORDER BY joined_at ASC
	`, groupUUID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var members []*models.GroupMember
	for rows.Next() {
		member := &models.GroupMember{}
		var joinedAt sql.NullString
		if err := rows.Scan(&member.ID, &member.GroupUUID, &member.PeerID, &member.Role, &member.InvitedBy, &member.InviteDepth, &joinedAt, &member.LeftAt); err != nil {
			return nil, err
		}
		if joinedAt.Valid {
			member.JoinedAt, _ = time.Parse("2006-01-02 15:04:05", joinedAt.String)
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func UpdateGroupMemberRole(groupUUID, peerID, newRole string) error {
	_, err := database.DB.Exec(`
		UPDATE group_members SET role = ?
		WHERE group_uuid = ? AND peer_id = ?
	`, newRole, groupUUID, peerID)
	return err
}

func RemoveGroupMember(groupUUID, peerID string) error {
	_, err := database.DB.Exec(`
		UPDATE group_members SET left_at = CURRENT_TIMESTAMP
		WHERE group_uuid = ? AND peer_id = ? AND left_at IS NULL
	`, groupUUID, peerID)
	return err
}

func IsGroupMember(groupUUID, peerID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM group_members
		WHERE group_uuid = ? AND peer_id = ? AND left_at IS NULL
	`, groupUUID, peerID).Scan(&count)
	return count > 0, err
}

func GetMemberCount(groupUUID string) (int, error) {
	var count int
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM group_members
		WHERE group_uuid = ? AND left_at IS NULL
	`, groupUUID).Scan(&count)
	return count, err
}
