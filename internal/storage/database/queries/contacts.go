// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetContact получает контакт по ID с данными профиля из profiles
func GetContact(id int) (*models.Contact, error) {
	row := database.DB.QueryRow(`
		SELECT 
			c.id, c.peer_id, c.multiaddr, c.notes, c.is_blocked, c.last_seen, c.added_at, c.updated_at,
			p.username, p.title, p.avatar_path
		FROM contacts c
		LEFT JOIN profiles p ON c.peer_id = p.peer_id
		WHERE c.id = ?
	`, id)

	contact := &models.Contact{}
	var lastSeen sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(
		&contact.ID,
		&contact.PeerID,
		&contact.Multiaddr,
		&contact.Notes,
		&contact.IsBlocked,
		&lastSeen,
		&createdAt,
		&updatedAt,
		&contact.Username,
		&contact.Title,
		&contact.AvatarPath,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("контакт не найден")
		}
		return nil, err
	}

	if lastSeen.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
		contact.LastSeen = &t
	}
	contact.AddedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	contact.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return contact, nil
}

// GetContactByPeerID получает контакт по PeerID с данными профиля
func GetContactByPeerID(peerID string) (*models.Contact, error) {
	row := database.DB.QueryRow(`
		SELECT 
			c.id, c.peer_id, c.multiaddr, c.notes, c.is_blocked, c.last_seen, c.added_at, c.updated_at,
			p.username, p.title, p.avatar_path
		FROM contacts c
		LEFT JOIN profiles p ON c.peer_id = p.peer_id
		WHERE c.peer_id = ?
	`, peerID)

	contact := &models.Contact{}
	var lastSeen sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(
		&contact.ID,
		&contact.PeerID,
		&contact.Multiaddr,
		&contact.Notes,
		&contact.IsBlocked,
		&lastSeen,
		&createdAt,
		&updatedAt,
		&contact.Username,
		&contact.Title,
		&contact.AvatarPath,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("контакт не найден")
		}
		return nil, err
	}

	if lastSeen.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
		contact.LastSeen = &t
	}
	contact.AddedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	contact.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return contact, nil
}

// GetAllContacts получает все контакты с данными профилей
func GetAllContacts() ([]*models.Contact, error) {
	rows, err := database.DB.Query(`
		SELECT 
			c.id, c.peer_id, c.multiaddr, c.notes, c.is_blocked, c.last_seen, c.added_at, c.updated_at,
			p.username, p.title, p.avatar_path
		FROM contacts c
		LEFT JOIN profiles p ON c.peer_id = p.peer_id
		ORDER BY p.username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact := &models.Contact{}
		var lastSeen sql.NullString
		var createdAt, updatedAt string

		err := rows.Scan(
			&contact.ID,
			&contact.PeerID,
			&contact.Multiaddr,
			&contact.Notes,
			&contact.IsBlocked,
			&lastSeen,
			&createdAt,
			&updatedAt,
			&contact.Username,
			&contact.Title,
			&contact.AvatarPath,
		)
		if err != nil {
			return nil, err
		}

		if lastSeen.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
			contact.LastSeen = &t
		}
		contact.AddedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		contact.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}

// CreateContact создаёт новый контакт
func CreateContact(contact *models.Contact) error {
	result, err := database.DB.Exec(`
		INSERT INTO contacts (peer_id, multiaddr, notes, is_blocked, last_seen, added_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, contact.PeerID, contact.Multiaddr, contact.Notes, contact.IsBlocked, contact.LastSeen)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	contact.ID = int(id)
	return nil
}

// UpdateContact обновляет контакт
func UpdateContact(contact *models.Contact) error {
	_, err := database.DB.Exec(`
		UPDATE contacts
		SET multiaddr = ?, notes = ?, is_blocked = ?, last_seen = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, contact.Multiaddr, contact.Notes, contact.IsBlocked, contact.LastSeen, contact.ID)
	return err
}

// UpdateContactLastSeen обновляет время последней активности контакта
func UpdateContactLastSeen(id int, lastSeen *time.Time) error {
	var lastSeenStr interface{}
	if lastSeen != nil {
		lastSeenStr = lastSeen.Format("2006-01-02 15:04:05")
	} else {
		lastSeenStr = nil
	}

	_, err := database.DB.Exec(`
		UPDATE contacts
		SET last_seen = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, lastSeenStr, id)
	return err
}

// UpdateContactNotes обновляет заметки контакта
func UpdateContactNotes(id int, notes string) error {
	_, err := database.DB.Exec(`
		UPDATE contacts
		SET notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, notes, id)
	return err
}

// UpdateContactMultiaddr обновляет адрес для подключения
func UpdateContactMultiaddr(id int, multiaddr string) error {
	_, err := database.DB.Exec(`
		UPDATE contacts
		SET multiaddr = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, multiaddr, id)
	return err
}

// UpdateContactByPeerID обновляет multiaddr контакта по PeerID
func UpdateContactByPeerID(peerID, multiaddr string) error {
	_, err := database.DB.Exec(`
		UPDATE contacts
		SET multiaddr = ?, updated_at = CURRENT_TIMESTAMP
		WHERE peer_id = ?
	`, multiaddr, peerID)
	return err
}

// DeleteContact удаляет контакт по ID
func DeleteContact(id int) error {
	_, err := database.DB.Exec(`DELETE FROM contacts WHERE id = ?`, id)
	return err
}

// DeleteContactByPeerID удаляет контакт по PeerID
func DeleteContactByPeerID(peerID string) error {
	_, err := database.DB.Exec(`DELETE FROM contacts WHERE peer_id = ?`, peerID)
	return err
}

// IsContactBlocked проверяет, заблокирован ли контакт
func IsContactBlocked(peerID string) (bool, error) {
	var isBlocked bool
	err := database.DB.QueryRow(`SELECT is_blocked FROM contacts WHERE peer_id = ?`, peerID).Scan(&isBlocked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("контакт не найден")
		}
		return false, err
	}
	return isBlocked, nil
}

// SearchContacts ищет контакты по имени профиля
func SearchContacts(query string) ([]*models.Contact, error) {
	rows, err := database.DB.Query(`
		SELECT 
			c.id, c.peer_id, c.multiaddr, c.notes, c.is_blocked, c.last_seen, c.added_at, c.updated_at,
			p.username, p.title, p.avatar_path
		FROM contacts c
		LEFT JOIN profiles p ON c.peer_id = p.peer_id
		WHERE p.username LIKE ?
		ORDER BY p.username
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact := &models.Contact{}
		var lastSeen sql.NullString
		var createdAt, updatedAt string

		err := rows.Scan(
			&contact.ID,
			&contact.PeerID,
			&contact.Multiaddr,
			&contact.Notes,
			&contact.IsBlocked,
			&lastSeen,
			&createdAt,
			&updatedAt,
			&contact.Username,
			&contact.Title,
			&contact.AvatarPath,
		)
		if err != nil {
			return nil, err
		}

		if lastSeen.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
			contact.LastSeen = &t
		}
		contact.AddedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		contact.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}

// BlockContact блокирует контакт
func BlockContact(id int) error {
	_, err := database.DB.Exec(`
		UPDATE contacts
		SET is_blocked = 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// UnblockContact разблокирует контакт
func UnblockContact(id int) error {
	_, err := database.DB.Exec(`
		UPDATE contacts
		SET is_blocked = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}
