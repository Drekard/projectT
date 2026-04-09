// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetChatMessage получает сообщение по ID
func GetChatMessage(id int) (*models.ChatMessage, error) {
	row := database.DB.QueryRow(`
		SELECT id, chat_id, from_peer_id, content, content_type, metadata, is_read, sent_at, COALESCE(updated_at, sent_at)
		FROM chat_messages
		WHERE id = ?
	`, id)

	message := &models.ChatMessage{}
	var metadata sql.NullString
	var sentAt, updatedAt string

	err := row.Scan(
		&message.ID,
		&message.ChatID,
		&message.FromPeerID,
		&message.Content,
		&message.ContentType,
		&metadata,
		&message.IsRead,
		&sentAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("сообщение не найдено")
		}
		return nil, err
	}

	if metadata.Valid {
		message.Metadata = metadata.String
	}
	message.SentAt, _ = parseTime(sentAt)
	message.UpdatedAt, _ = parseTime(updatedAt)

	return message, nil
}

// GetMessagesForChat получает все сообщения для чата
func GetMessagesForChat(chatID int, limit, offset int) ([]*models.ChatMessage, error) {
	rows, err := database.DB.Query(`
		SELECT id, chat_id, from_peer_id, content, content_type, metadata, is_read, sent_at, COALESCE(updated_at, sent_at)
		FROM chat_messages
		WHERE chat_id = ?
		ORDER BY sent_at DESC
		LIMIT ? OFFSET ?
	`, chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var messages []*models.ChatMessage
	for rows.Next() {
		message := &models.ChatMessage{}
		var metadata sql.NullString
		var sentAt, updatedAt string

		err := rows.Scan(
			&message.ID,
			&message.ChatID,
			&message.FromPeerID,
			&message.Content,
			&message.ContentType,
			&metadata,
			&message.IsRead,
			&sentAt,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if metadata.Valid {
			message.Metadata = metadata.String
		}

		// Пробуем распарсить в формате RFC3339, затем в SQL формате
		message.SentAt, _ = parseTime(sentAt)
		message.UpdatedAt, _ = parseTime(updatedAt)

		messages = append(messages, message)
	}

	// Реверсируем порядок, чтобы новые сообщения были в конце
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, rows.Err()
}

// GetMessagesForContact устаревший метод, использует GetMessagesForChat
// Для обратной совместимости
func GetMessagesForContact(contactID int, limit, offset int) ([]*models.ChatMessage, error) {
	// Получаем чат по contact_id
	chat, err := GetChatByContactID(contactID)
	if err != nil {
		return nil, err
	}
	if chat == nil {
		return []*models.ChatMessage{}, nil
	}
	return GetMessagesForChat(chat.ID, limit, offset)
}

// parseTime парсит время из строки в формате RFC3339 или SQL
func parseTime(timeStr string) (time.Time, error) {
	// Пробуем RFC3339 (ISO8601)
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}
	// Пробуем SQL формат
	if t, err := time.Parse("2006-01-02 15:04:05", timeStr); err == nil {
		return t, nil
	}
	// Пробуем SQL формат с T
	if t, err := time.Parse("2006-01-02T15:04:05", timeStr); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("не удалось распарсить время: %s", timeStr)
}

// GetUnreadMessagesCount получает количество непрочитанных сообщений для чата
func GetUnreadMessagesCount(chatID int) (int, error) {
	var count int
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM chat_messages
		WHERE chat_id = ? AND is_read = 0
	`, chatID).Scan(&count)
	return count, err
}

// GetUnreadMessagesCountForContact устаревший метод
func GetUnreadMessagesCountForContact(contactID int) (int, error) {
	chat, err := GetChatByContactID(contactID)
	if err != nil {
		return 0, err
	}
	if chat == nil {
		return 0, nil
	}
	return GetUnreadMessagesCount(chat.ID)
}

// CreateChatMessage создаёт новое сообщение
func CreateChatMessage(message *models.ChatMessage) error {
	result, err := database.DB.Exec(`
		INSERT INTO chat_messages (chat_id, from_peer_id, content, content_type, metadata, is_read, sent_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, message.ChatID, message.FromPeerID, message.Content, message.ContentType, message.Metadata, message.IsRead)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	message.ID = int(id)
	return nil
}

// MarkMessageAsRead помечает сообщение как прочитанное
func MarkMessageAsRead(id int) error {
	_, err := database.DB.Exec(`
		UPDATE chat_messages
		SET is_read = 1
		WHERE id = ?
	`, id)
	return err
}

// MarkAllMessagesAsRead помечает все сообщения для чата как прочитанные
func MarkAllMessagesAsRead(chatID int) error {
	_, err := database.DB.Exec(`
		UPDATE chat_messages
		SET is_read = 1
		WHERE chat_id = ? AND is_read = 0
	`, chatID)
	return err
}

// MarkAllMessagesAsReadForContact устаревший метод
func MarkAllMessagesAsReadForContact(contactID int) error {
	chat, err := GetChatByContactID(contactID)
	if err != nil {
		return err
	}
	if chat == nil {
		return nil
	}
	return MarkAllMessagesAsRead(chat.ID)
}

// UpdateChatMessage обновляет сообщение
func UpdateChatMessage(message *models.ChatMessage) error {
	_, err := database.DB.Exec(`
		UPDATE chat_messages
		SET content = ?, content_type = ?, metadata = ?, is_read = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, message.Content, message.ContentType, message.Metadata, message.IsRead, message.ID)
	return err
}

// DeleteChatMessage удаляет сообщение по ID
func DeleteChatMessage(id int) error {
	_, err := database.DB.Exec(`DELETE FROM chat_messages WHERE id = ?`, id)
	return err
}

// DeleteMessagesForChat удаляет все сообщения для чата
func DeleteMessagesForChat(chatID int) error {
	_, err := database.DB.Exec(`DELETE FROM chat_messages WHERE chat_id = ?`, chatID)
	return err
}

// DeleteMessagesForContact устаревший метод
func DeleteMessagesForContact(contactID int) error {
	chat, err := GetChatByContactID(contactID)
	if err != nil {
		return err
	}
	if chat == nil {
		return nil
	}
	return DeleteMessagesForChat(chat.ID)
}

// GetLastMessageForChat получает последнее сообщение для чата
func GetLastMessageForChat(chatID int) (*models.ChatMessage, error) {
	row := database.DB.QueryRow(`
		SELECT id, chat_id, from_peer_id, content, content_type, metadata, is_read, sent_at, COALESCE(updated_at, sent_at)
		FROM chat_messages
		WHERE chat_id = ?
		ORDER BY sent_at DESC
		LIMIT 1
	`, chatID)

	message := &models.ChatMessage{}
	var metadata sql.NullString
	var sentAt, updatedAt string

	err := row.Scan(
		&message.ID,
		&message.ChatID,
		&message.FromPeerID,
		&message.Content,
		&message.ContentType,
		&metadata,
		&message.IsRead,
		&sentAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("сообщения не найдены")
		}
		return nil, err
	}

	if metadata.Valid {
		message.Metadata = metadata.String
	}
	message.SentAt, _ = time.Parse("2006-01-02 15:04:05", sentAt)
	message.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return message, nil
}

// GetLastMessageForContact устаревший метод
func GetLastMessageForContact(contactID int) (*models.ChatMessage, error) {
	chat, err := GetChatByContactID(contactID)
	if err != nil {
		return nil, err
	}
	if chat == nil {
		return nil, errors.New("чат не найден")
	}
	return GetLastMessageForChat(chat.ID)
}

// IsDuplicateMessage проверяет, есть ли сообщение с тем же содержимым в чате за последние N секунд
func IsDuplicateMessage(chatID int, fromPeerID, content string, window time.Duration) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM chat_messages 
		WHERE chat_id = ? 
		  AND from_peer_id = ? 
		  AND content = ? 
		  AND sent_at > datetime('now', ?)
	`

	var count int
	windowArg := fmt.Sprintf("-%.0f seconds", window.Seconds())

	row := database.DB.QueryRow(query, chatID, fromPeerID, content, windowArg)
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
