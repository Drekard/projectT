package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

func CreateGroupChat(chat *models.GroupChat) error {
	result, err := database.DB.Exec(`
		INSERT INTO group_chats (group_uuid, name, description, creator_peer_id, avatar_hash, chat_type, max_invite_depth, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, chat.GroupUUID, chat.Name, chat.Description, chat.CreatorPeerID, chat.AvatarHash, chat.ChatType, chat.MaxInviteDepth)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	chat.ID = int(id)
	return nil
}

func GetGroupChatByUUID(groupUUID string) (*models.GroupChat, error) {
	row := database.DB.QueryRow(`
		SELECT id, group_uuid, name, description, creator_peer_id, avatar_hash, chat_type, max_invite_depth, created_at, updated_at
		FROM group_chats
		WHERE group_uuid = ?
	`, groupUUID)

	chat := &models.GroupChat{}
	var createdAt, updatedAt string
	err := row.Scan(&chat.ID, &chat.GroupUUID, &chat.Name, &chat.Description, &chat.CreatorPeerID, &chat.AvatarHash, &chat.ChatType, &chat.MaxInviteDepth, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	chat.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	chat.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return chat, nil
}

func GetGroupChatsByPeerID(peerID string) ([]*models.GroupChat, error) {
	rows, err := database.DB.Query(`
		SELECT DISTINCT gc.id, gc.group_uuid, gc.name, gc.description, gc.creator_peer_id, gc.avatar_hash, gc.chat_type, gc.max_invite_depth, gc.created_at, gc.updated_at
		FROM group_chats gc
		INNER JOIN group_members gm ON gc.group_uuid = gm.group_uuid
		WHERE gm.peer_id = ? AND gm.left_at IS NULL
		ORDER BY gc.updated_at DESC
	`, peerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var chats []*models.GroupChat
	for rows.Next() {
		chat := &models.GroupChat{}
		var createdAt, updatedAt string
		if err := rows.Scan(&chat.ID, &chat.GroupUUID, &chat.Name, &chat.Description, &chat.CreatorPeerID, &chat.AvatarHash, &chat.ChatType, &chat.MaxInviteDepth, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		chat.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		chat.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func UpdateGroupChat(groupUUID, name, description string) error {
	_, err := database.DB.Exec(`
		UPDATE group_chats SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE group_uuid = ?
	`, name, description, groupUUID)
	return err
}

func DeleteGroupChat(groupUUID string) error {
	_, err := database.DB.Exec(`DELETE FROM group_chats WHERE group_uuid = ?`, groupUUID)
	return err
}
