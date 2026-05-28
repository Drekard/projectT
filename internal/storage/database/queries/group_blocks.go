package queries

import (
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

func BlockGroupMember(block *models.GroupBlock) error {
	_, err := database.DB.Exec(`
		INSERT INTO group_blocks (group_uuid, peer_id, blocked_by, reason, created_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(group_uuid, peer_id) DO UPDATE SET
			blocked_by = excluded.blocked_by,
			reason = excluded.reason,
			created_at = CURRENT_TIMESTAMP
	`, block.GroupUUID, block.PeerID, block.BlockedBy, block.Reason)
	return err
}

func IsGroupBlocked(groupUUID, peerID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM group_blocks WHERE group_uuid = ? AND peer_id = ?
	`, groupUUID, peerID).Scan(&count)
	return count > 0, err
}

func GetGroupBlocks(groupUUID string) ([]*models.GroupBlock, error) {
	rows, err := database.DB.Query(`
		SELECT id, group_uuid, peer_id, blocked_by, reason, created_at
		FROM group_blocks
		WHERE group_uuid = ?
		ORDER BY created_at DESC
	`, groupUUID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var blocks []*models.GroupBlock
	for rows.Next() {
		block := &models.GroupBlock{}
		var createdAt string
		if err := rows.Scan(&block.ID, &block.GroupUUID, &block.PeerID, &block.BlockedBy, &block.Reason, &createdAt); err != nil {
			return nil, err
		}
		block.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}

func UnblockGroupMember(groupUUID, peerID string) error {
	_, err := database.DB.Exec(`
		DELETE FROM group_blocks WHERE group_uuid = ? AND peer_id = ?
	`, groupUUID, peerID)
	return err
}

func GetBlocksByPeer(peerID string) ([]*models.GroupBlock, error) {
	rows, err := database.DB.Query(`
		SELECT id, group_uuid, peer_id, blocked_by, reason, created_at
		FROM group_blocks
		WHERE peer_id = ?
		ORDER BY created_at DESC
	`, peerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var blocks []*models.GroupBlock
	for rows.Next() {
		block := &models.GroupBlock{}
		var createdAt string
		if err := rows.Scan(&block.ID, &block.GroupUUID, &block.PeerID, &block.BlockedBy, &block.Reason, &createdAt); err != nil {
			return nil, err
		}
		block.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}
