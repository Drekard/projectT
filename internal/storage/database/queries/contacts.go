// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetContact получает контакт по ID
func GetContact(id int) (*models.Contact, error) {
	row := database.DB.QueryRow(`
		SELECT id, peer_id, username, public_key, multiaddr, status, last_seen, notes, is_blocked, added_at, updated_at
		FROM contacts
		WHERE id = ?
	`, id)

	contact := &models.Contact{}
	var publicKey []byte
	var lastSeen sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(
		&contact.ID,
		&contact.PeerID,
		&contact.Username,
		&publicKey,
		&contact.Multiaddr,
		&contact.Status,
		&lastSeen,
		&contact.Notes,
		&contact.IsBlocked,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("контакт не найден")
		}
		return nil, err
	}

	contact.PublicKey = publicKey
	if lastSeen.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
		contact.LastSeen = &t
	}
	contact.AddedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	contact.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return contact, nil
}

// GetContactByPeerID получает контакт по PeerID
func GetContactByPeerID(peerID string) (*models.Contact, error) {
	row := database.DB.QueryRow(`
		SELECT id, peer_id, username, public_key, multiaddr, status, last_seen, notes, is_blocked, added_at, updated_at
		FROM contacts
		WHERE peer_id = ?
	`, peerID)

	contact := &models.Contact{}
	var publicKey []byte
	var lastSeen sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(
		&contact.ID,
		&contact.PeerID,
		&contact.Username,
		&publicKey,
		&contact.Multiaddr,
		&contact.Status,
		&lastSeen,
		&contact.Notes,
		&contact.IsBlocked,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("контакт не найден")
		}
		return nil, err
	}

	contact.PublicKey = publicKey
	if lastSeen.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
		contact.LastSeen = &t
	}
	contact.AddedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	contact.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return contact, nil
}

// GetAllContacts получает все контакты
func GetAllContacts() ([]*models.Contact, error) {
	rows, err := database.DB.Query(`
		SELECT id, peer_id, username, public_key, multiaddr, status, last_seen, notes, is_blocked, added_at, updated_at
		FROM contacts
		ORDER BY username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact := &models.Contact{}
		var publicKey []byte
		var lastSeen sql.NullString
		var createdAt, updatedAt string

		err := rows.Scan(
			&contact.ID,
			&contact.PeerID,
			&contact.Username,
			&publicKey,
			&contact.Multiaddr,
			&contact.Status,
			&lastSeen,
			&contact.Notes,
			&contact.IsBlocked,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}

		contact.PublicKey = publicKey
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
		INSERT INTO contacts (peer_id, username, public_key, multiaddr, status, last_seen, notes, is_blocked, added_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, contact.PeerID, contact.Username, contact.PublicKey, contact.Multiaddr, contact.Status, contact.LastSeen, contact.Notes, contact.IsBlocked)
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
		SET peer_id = ?, username = ?, public_key = ?, multiaddr = ?, status = ?, last_seen = ?, notes = ?, is_blocked = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, contact.PeerID, contact.Username, contact.PublicKey, contact.Multiaddr, contact.Status, contact.LastSeen, contact.Notes, contact.IsBlocked, contact.ID)
	return err
}

// UpdateContactStatus обновляет статус контакта
func UpdateContactStatus(id int, status string, lastSeen *time.Time) error {
	var lastSeenStr interface{}
	if lastSeen != nil {
		lastSeenStr = lastSeen.Format("2006-01-02 15:04:05")
	} else {
		lastSeenStr = nil
	}

	_, err := database.DB.Exec(`
		UPDATE contacts
		SET status = ?, last_seen = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, lastSeenStr, id)
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

// SearchContacts ищет контакты по имени
func SearchContacts(query string) ([]*models.Contact, error) {
	rows, err := database.DB.Query(`
		SELECT id, peer_id, username, public_key, multiaddr, status, last_seen, notes, is_blocked, added_at, updated_at
		FROM contacts
		WHERE username LIKE ?
		ORDER BY username
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact := &models.Contact{}
		var publicKey []byte
		var lastSeen sql.NullString
		var createdAt, updatedAt string

		err := rows.Scan(
			&contact.ID,
			&contact.PeerID,
			&contact.Username,
			&publicKey,
			&contact.Multiaddr,
			&contact.Status,
			&lastSeen,
			&contact.Notes,
			&contact.IsBlocked,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}

		contact.PublicKey = publicKey
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
