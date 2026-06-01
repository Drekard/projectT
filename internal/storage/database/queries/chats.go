// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"
	"log"
	"sort"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetChat получает чат по ID с данными профиля
func GetChat(chatID int) (*models.Chat, error) {
	row := database.DB.QueryRow(`
		SELECT
			c.id, c.contact_id, c.peer_id, c.is_temporary, c.last_message_at, c.created_at, c.updated_at,
			COALESCE(p.username, '') as username,
			COALESCE(p.avatar_path, '') as avatar_path
		FROM chats c
		LEFT JOIN profiles p ON c.peer_id = p.peer_id
		WHERE c.id = ?
	`, chatID)

	chat := &models.Chat{}
	var lastMessageAt, createdAt, updatedAt sql.NullString

	err := row.Scan(
		&chat.ID,
		&chat.ContactID,
		&chat.PeerID,
		&chat.IsTemporary,
		&lastMessageAt,
		&createdAt,
		&updatedAt,
		&chat.Username,
		&chat.AvatarPath,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("чат не найден")
		}
		return nil, err
	}

	if lastMessageAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastMessageAt.String)
		chat.LastMessageAt = &t
	}
	if createdAt.Valid {
		chat.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
	}
	if updatedAt.Valid {
		chat.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
	}

	return chat, nil
}

// GetChatByPeerID получает чат по PeerID
func GetChatByPeerID(peerID string) (*models.Chat, error) {
	row := database.DB.QueryRow(`
		SELECT
			c.id, c.contact_id, c.peer_id, c.is_temporary, c.last_message_at, c.created_at, c.updated_at,
			COALESCE(p.username, '') as username,
			COALESCE(p.avatar_path, '') as avatar_path
		FROM chats c
		LEFT JOIN profiles p ON c.peer_id = p.peer_id
		WHERE c.peer_id = ?
	`, peerID)

	chat := &models.Chat{}
	var lastMessageAt, createdAt, updatedAt sql.NullString

	err := row.Scan(
		&chat.ID,
		&chat.ContactID,
		&chat.PeerID,
		&chat.IsTemporary,
		&lastMessageAt,
		&createdAt,
		&updatedAt,
		&chat.Username,
		&chat.AvatarPath,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Чат не найден - это не ошибка
		}
		return nil, err
	}

	if lastMessageAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastMessageAt.String)
		chat.LastMessageAt = &t
	}
	if createdAt.Valid {
		chat.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
	}
	if updatedAt.Valid {
		chat.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
	}

	return chat, nil
}

// GetUnifiedChatList возвращает объединённый список чатов, групп и каналов
// Сортировка по времени последнего сообщения (новые сверху)
func GetUnifiedChatList(localPeerID string) ([]*models.UnifiedChatItem, error) {
	var items []*models.UnifiedChatItem

	// 1. Получаем обычные чаты (direct)
	directChats, err := GetChatsWithLastMessages()
	if err == nil {
		for _, chat := range directChats {
			item := &models.UnifiedChatItem{
				ID:              chat.ID,
				ChatType:        "direct",
				PeerID:          chat.PeerID,
				Username:        chat.Username,
				AvatarPath:      chat.AvatarPath,
				LastMessageID:   chat.LastMessageID,
				LastMessage:     chat.LastMessage,
				LastMessageType: chat.LastMessageType,
				LastMessageAt:   chat.LastMessageAt,
				IsOutgoing:      chat.IsOutgoing,
				UnreadCount:     chat.UnreadCount,
			}
			items = append(items, item)
		}
	}

	// 2. Получаем группы и каналы
	groupChats, err := GetGroupChatsByPeerID(localPeerID)
	if err == nil {
		for _, gc := range groupChats {
			item := &models.UnifiedChatItem{
				ID:            gc.ID,
				ChatType:      gc.ChatType, // "group" or "channel"
				GroupUUID:     gc.GroupUUID,
				PeerID:        gc.CreatorPeerID,
				Username:      gc.Name,
				LastMessageAt: &gc.UpdatedAt,
			}
			items = append(items, item)
		}
	}

	// Сортировка по LastMessageAt (новые сверху)
	sort.Slice(items, func(i, j int) bool {
		if items[i].LastMessageAt == nil && items[j].LastMessageAt == nil {
			return false
		}
		if items[i].LastMessageAt == nil {
			return false
		}
		if items[j].LastMessageAt == nil {
			return true
		}
		return items[i].LastMessageAt.After(*items[j].LastMessageAt)
	})

	return items, nil
}

// GetChatsWithLastMessages возвращает все чаты с последними сообщениями
// Сортировка по времени последнего сообщения (новые сверху)
func GetChatsWithLastMessages() ([]*models.ChatWithLastMessage, error) {
	query := `
		SELECT
			c.id,
			c.contact_id,
			c.peer_id,
			COALESCE(p.username, c.peer_id) as username,
			COALESCE(p.avatar_path, '') as avatar_path,
			c.is_temporary,
			COALESCE(lm.id, 0) as last_message_id,
			COALESCE(lm.content, '') as last_message,
			COALESCE(lm.content_type, 'text') as last_message_type,
			lm.sent_at as last_message_at,
			COALESCE(lm.from_peer_id, '') != c.peer_id as is_outgoing,
			COALESCE(uc.unread_count, 0) as unread_count
		FROM chats c
		LEFT JOIN profiles p ON c.peer_id = p.peer_id AND p.owner_type = 'remote'
		LEFT JOIN (
			SELECT
				cm1.chat_id,
				cm1.id,
				cm1.content,
				cm1.content_type,
				cm1.from_peer_id,
				cm1.sent_at
			FROM chat_messages cm1
			INNER JOIN (
				SELECT chat_id, MAX(id) as max_id
				FROM chat_messages
				WHERE sent_at IN (
					SELECT MAX(sent_at) FROM chat_messages GROUP BY chat_id
				)
				GROUP BY chat_id
			) cm2 ON cm1.chat_id = cm2.chat_id AND cm1.id = cm2.max_id
		) lm ON c.id = lm.chat_id
		LEFT JOIN (
			SELECT chat_id, COUNT(*) as unread_count
			FROM chat_messages
			WHERE is_read = 0
			GROUP BY chat_id
		) uc ON c.id = uc.chat_id
		WHERE c.peer_id IS NOT NULL AND c.peer_id != ''
		AND c.peer_id != ?
		AND c.peer_id NOT IN (
			SELECT peer_id FROM profiles WHERE owner_type = 'local'
		)
		GROUP BY c.id
	`

	rows, err := database.DB.Query(query, models.LocalChatPeerID)
	if err != nil {
		log.Printf("[Chat] ❌ SQL ошибка GetChatsWithLastMessages: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var chats []*models.ChatWithLastMessage
	rowCount := 0
	for rows.Next() {
		rowCount++
		chat := &models.ChatWithLastMessage{}
		var lastMessageAt sql.NullString
		var contactID sql.NullInt64

		err := rows.Scan(
			&chat.ID,
			&contactID,
			&chat.PeerID,
			&chat.Username,
			&chat.AvatarPath,
			&chat.IsTemporary,
			&chat.LastMessageID,
			&chat.LastMessage,
			&chat.LastMessageType,
			&lastMessageAt,
			&chat.IsOutgoing,
			&chat.UnreadCount,
		)
		if err != nil {
			log.Printf("[Chat] ❌ Ошибка Scan: %v", err)
			return nil, err
		}

		if contactID.Valid {
			id := int(contactID.Int64)
			chat.ContactID = &id
		}

		if lastMessageAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastMessageAt.String)
			chat.LastMessageAt = &t
		}

		chats = append(chats, chat)
	}

	// Сортируем чаты по времени последнего сообщения (в памяти)
	sort.Slice(chats, func(i, j int) bool {
		if chats[i].LastMessageAt == nil {
			return false
		}
		if chats[j].LastMessageAt == nil {
			return true
		}
		return chats[i].LastMessageAt.After(*chats[j].LastMessageAt)
	})

	return chats, rows.Err()
}

// CreateChat создаёт новый чат
func CreateChat(chat *models.Chat) error {
	result, err := database.DB.Exec(`
		INSERT INTO chats (contact_id, peer_id, is_temporary, last_message_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, chat.ContactID, chat.PeerID, chat.IsTemporary, chat.LastMessageAt)
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

// GetOrCreateChat получает существующий чат или создаёт новый
// Для временных чатов contact_id может быть nil
func GetOrCreateChat(peerID string, contactID *int) (*models.Chat, error) {
	// Пробуем получить существующий
	chat, err := GetChatByPeerID(peerID)
	if err != nil {
		return nil, err
	}

	if chat != nil {
		return chat, nil
	}

	// Создаём новый чат
	chat = &models.Chat{
		ContactID:   contactID,
		PeerID:      peerID,
		IsTemporary: contactID == nil,
	}

	if err := CreateChat(chat); err != nil {
		return nil, err
	}

	return chat, nil
}

// UpdateChatLastMessage обновляет время последнего сообщения в чате
func UpdateChatLastMessage(chatID int, sentAt time.Time) error {
	_, err := database.DB.Exec(`
		UPDATE chats
		SET last_message_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, sentAt, chatID)
	return err
}

// DeleteChat удаляет чат по ID (сообщения удалятся каскадом)
func DeleteChat(chatID int) error {
	_, err := database.DB.Exec(`DELETE FROM chats WHERE id = ?`, chatID)
	return err
}

// DeleteChatByPeerID удаляет чат по PeerID
func DeleteChatByPeerID(peerID string) error {
	_, err := database.DB.Exec(`DELETE FROM chats WHERE peer_id = ?`, peerID)
	return err
}

// LinkChatToContact привязывает контакт к чату (превращает временный чат в постоянный)
func LinkChatToContact(chatID int, contactID int) error {
	_, err := database.DB.Exec(`
		UPDATE chats
		SET contact_id = ?, is_temporary = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, contactID, chatID)
	return err
}

// GetChatByContactID получает чат по ID контакта
func GetChatByContactID(contactID int) (*models.Chat, error) {
	row := database.DB.QueryRow(`
		SELECT
			c.id, c.contact_id, c.peer_id, c.is_temporary, c.last_message_at, c.created_at, c.updated_at,
			COALESCE(p.username, '') as username,
			COALESCE(p.avatar_path, '') as avatar_path
		FROM chats c
		LEFT JOIN profiles p ON c.peer_id = p.peer_id
		WHERE c.contact_id = ?
	`, contactID)

	chat := &models.Chat{}
	var lastMessageAt, createdAt, updatedAt sql.NullString

	err := row.Scan(
		&chat.ID,
		&chat.ContactID,
		&chat.PeerID,
		&chat.IsTemporary,
		&lastMessageAt,
		&createdAt,
		&updatedAt,
		&chat.Username,
		&chat.AvatarPath,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Чат не найден
		}
		return nil, err
	}

	if lastMessageAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastMessageAt.String)
		chat.LastMessageAt = &t
	}
	if createdAt.Valid {
		chat.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
	}
	if updatedAt.Valid {
		chat.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
	}

	return chat, nil
}

// GetOrCreateLocalChat получает или создаёт чат для локального профиля
// Локальный чат имеет contact_id = NULL и peer_id = локальный PeerID
// Такой чат не отображается в общем списке чатов, но может хранить сообщения
func GetOrCreateLocalChat(localPeerID string) (*models.Chat, error) {
	// Пробуем получить существующий локальный чат
	chat, err := GetChatByPeerID(localPeerID)
	if err != nil {
		return nil, err
	}

	if chat != nil {
		return chat, nil
	}

	// Создаём новый локальный чат с contact_id = NULL
	chat = &models.Chat{
		ContactID:   nil, // NULL означает "нет контакта" (локальный чат)
		PeerID:      localPeerID,
		IsTemporary: false, // Это не временный чат, а постоянный локальный
	}

	if err := CreateChat(chat); err != nil {
		return nil, err
	}

	return chat, nil
}

// GetLocalChat получает локальный чат по PeerID
func GetLocalChat(localPeerID string) (*models.Chat, error) {
	return GetChatByPeerID(localPeerID)
}
