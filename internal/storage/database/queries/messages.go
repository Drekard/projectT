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
		SELECT id, contact_id, from_peer_id, content, content_type, metadata, is_read, sent_at, COALESCE(updated_at, sent_at)
		FROM chat_messages
		WHERE id = ?
	`, id)

	message := &models.ChatMessage{}
	var metadata sql.NullString
	var sentAt, updatedAt string

	err := row.Scan(
		&message.ID,
		&message.ContactID,
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

// GetMessagesForContact получает все сообщения для контакта
func GetMessagesForContact(contactID int, limit, offset int) ([]*models.ChatMessage, error) {
	rows, err := database.DB.Query(`
		SELECT id, contact_id, from_peer_id, content, content_type, metadata, is_read, sent_at, COALESCE(updated_at, sent_at)
		FROM chat_messages
		WHERE contact_id = ?
		ORDER BY sent_at DESC
		LIMIT ? OFFSET ?
	`, contactID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.ChatMessage
	for rows.Next() {
		message := &models.ChatMessage{}
		var metadata sql.NullString
		var sentAt, updatedAt string

		err := rows.Scan(
			&message.ID,
			&message.ContactID,
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

	fmt.Printf("[DEBUG] Загружено %d сообщений для контакта (contact_id=%d)\n", len(messages), contactID)

	return messages, rows.Err()
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

// GetUnreadMessagesCount получает количество непрочитанных сообщений для контакта
func GetUnreadMessagesCount(contactID int) (int, error) {
	var count int
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM chat_messages
		WHERE contact_id = ? AND is_read = 0
	`, contactID).Scan(&count)
	return count, err
}

// CreateChatMessage создаёт новое сообщение
func CreateChatMessage(message *models.ChatMessage) error {
	result, err := database.DB.Exec(`
		INSERT INTO chat_messages (contact_id, from_peer_id, content, content_type, metadata, is_read, sent_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, message.ContactID, message.FromPeerID, message.Content, message.ContentType, message.Metadata, message.IsRead)
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

// MarkAllMessagesAsRead помечает все сообщения для контакта как прочитанные
func MarkAllMessagesAsRead(contactID int) error {
	_, err := database.DB.Exec(`
		UPDATE chat_messages
		SET is_read = 1
		WHERE contact_id = ? AND is_read = 0
	`, contactID)
	return err
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

// DeleteMessagesForContact удаляет все сообщения для контакта
func DeleteMessagesForContact(contactID int) error {
	_, err := database.DB.Exec(`DELETE FROM chat_messages WHERE contact_id = ?`, contactID)
	return err
}

// GetLastMessageForContact получает последнее сообщение для контакта
func GetLastMessageForContact(contactID int) (*models.ChatMessage, error) {
	row := database.DB.QueryRow(`
		SELECT id, contact_id, from_peer_id, content, content_type, metadata, is_read, sent_at, COALESCE(updated_at, sent_at)
		FROM chat_messages
		WHERE contact_id = ?
		ORDER BY sent_at DESC
		LIMIT 1
	`, contactID)

	message := &models.ChatMessage{}
	var metadata sql.NullString
	var sentAt, updatedAt string

	err := row.Scan(
		&message.ID,
		&message.ContactID,
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
