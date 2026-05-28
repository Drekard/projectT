package queries

import (
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

func CreateGroupMessage(msg *models.GroupMessage) error {
	result, err := database.DB.Exec(`
		INSERT INTO group_messages (group_uuid, message_uuid, from_peer_id, content, content_type, metadata, timestamp, lamport_clock, signature)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.GroupUUID, msg.MessageUUID, msg.FromPeerID, msg.Content, msg.ContentType, msg.Metadata, msg.Timestamp, msg.LamportClock, msg.Signature)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	msg.ID = int(id)
	return nil
}

func GetGroupMessages(groupUUID string, limit, offset int) ([]*models.GroupMessage, error) {
	rows, err := database.DB.Query(`
		SELECT id, group_uuid, message_uuid, from_peer_id, content, content_type, metadata, timestamp, lamport_clock, signature
		FROM group_messages
		WHERE group_uuid = ?
		ORDER BY lamport_clock ASC, timestamp ASC
		LIMIT ? OFFSET ?
	`, groupUUID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var messages []*models.GroupMessage
	for rows.Next() {
		msg := &models.GroupMessage{}
		if err := rows.Scan(&msg.ID, &msg.GroupUUID, &msg.MessageUUID, &msg.FromPeerID, &msg.Content, &msg.ContentType, &msg.Metadata, &msg.Timestamp, &msg.LamportClock, &msg.Signature); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func GetGroupMessagesAfterLamport(groupUUID string, afterLamport uint64, count int) ([]*models.GroupMessage, error) {
	rows, err := database.DB.Query(`
		SELECT id, group_uuid, message_uuid, from_peer_id, content, content_type, metadata, timestamp, lamport_clock, signature
		FROM group_messages
		WHERE group_uuid = ? AND lamport_clock > ?
		ORDER BY lamport_clock ASC, timestamp ASC
		LIMIT ?
	`, groupUUID, afterLamport, count)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var messages []*models.GroupMessage
	for rows.Next() {
		msg := &models.GroupMessage{}
		if err := rows.Scan(&msg.ID, &msg.GroupUUID, &msg.MessageUUID, &msg.FromPeerID, &msg.Content, &msg.ContentType, &msg.Metadata, &msg.Timestamp, &msg.LamportClock, &msg.Signature); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func GetGroupMessagesAfterUUID(groupUUID, afterUUID string, count int) ([]*models.GroupMessage, error) {
	rows, err := database.DB.Query(`
		SELECT id, group_uuid, message_uuid, from_peer_id, content, content_type, metadata, timestamp, lamport_clock, signature
		FROM group_messages
		WHERE group_uuid = ? AND timestamp > (
			SELECT timestamp FROM group_messages WHERE message_uuid = ?
		)
		ORDER BY lamport_clock ASC, timestamp ASC
		LIMIT ?
	`, groupUUID, afterUUID, count)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var messages []*models.GroupMessage
	for rows.Next() {
		msg := &models.GroupMessage{}
		if err := rows.Scan(&msg.ID, &msg.GroupUUID, &msg.MessageUUID, &msg.FromPeerID, &msg.Content, &msg.ContentType, &msg.Metadata, &msg.Timestamp, &msg.LamportClock, &msg.Signature); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func IsDuplicateGroupMessage(messageUUID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM group_messages WHERE message_uuid = ?`, messageUUID).Scan(&count)
	return count > 0, err
}

func GetLatestLamportClock(groupUUID string) (uint64, error) {
	var lamport uint64
	err := database.DB.QueryRow(`
		SELECT COALESCE(MAX(lamport_clock), 0) FROM group_messages WHERE group_uuid = ?
	`, groupUUID).Scan(&lamport)
	return lamport, err
}
