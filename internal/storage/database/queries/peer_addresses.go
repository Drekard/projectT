// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetActivePeerAddresses возвращает все адреса для подключения
// Сортирует по приоритету (bootstrap/contact сначала) и времени последнего подключения
func GetActivePeerAddresses() ([]*models.PeerAddress, error) {
	rows, err := database.DB.Query(`
		SELECT 
			pa.id,
			pa.profile_id,
			pa.multiaddr,
			pa.address_type,
			pa.is_active,
			pa.last_connected,
			pa.last_seen,
			pa.priority,
			pa.source,
			pa.created_at,
			pa.updated_at,
			p.peer_id,
			p.username
		FROM peer_addresses pa
		JOIN profiles p ON pa.profile_id = p.id
		WHERE pa.is_active = 1
		ORDER BY 
			pa.priority DESC,
			pa.last_connected DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var addresses []*models.PeerAddress
	for rows.Next() {
		addr := &models.PeerAddress{}
		var lastConnected, lastSeen sql.NullString
		var createdAt, updatedAt sql.NullString

		err := rows.Scan(
			&addr.ID,
			&addr.ProfileID,
			&addr.Multiaddr,
			&addr.AddressType,
			&addr.IsActive,
			&lastConnected,
			&lastSeen,
			&addr.Priority,
			&addr.Source,
			&createdAt,
			&updatedAt,
			&addr.PeerID,
			&addr.Username,
		)
		if err != nil {
			return nil, err
		}

		if lastConnected.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastConnected.String)
			addr.LastConnected = &t
		}
		if lastSeen.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
			addr.LastSeen = &t
		}
		if createdAt.Valid {
			addr.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
		}
		if updatedAt.Valid {
			addr.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
		}

		addresses = append(addresses, addr)
	}

	return addresses, rows.Err()
}

// GetPeerAddressesByType возвращает адреса по типу
func GetPeerAddressesByType(addressType string) ([]*models.PeerAddress, error) {
	rows, err := database.DB.Query(`
		SELECT 
			pa.id,
			pa.profile_id,
			pa.multiaddr,
			pa.address_type,
			pa.is_active,
			pa.last_connected,
			pa.last_seen,
			pa.priority,
			pa.source,
			pa.created_at,
			pa.updated_at,
			p.peer_id,
			p.username
		FROM peer_addresses pa
		JOIN profiles p ON pa.profile_id = p.id
		WHERE pa.is_active = 1 AND pa.address_type = ?
		ORDER BY pa.priority DESC, pa.last_connected DESC
	`, addressType)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var addresses []*models.PeerAddress
	for rows.Next() {
		addr := &models.PeerAddress{}
		var lastConnected, lastSeen sql.NullString
		var createdAt, updatedAt sql.NullString

		err := rows.Scan(
			&addr.ID,
			&addr.ProfileID,
			&addr.Multiaddr,
			&addr.AddressType,
			&addr.IsActive,
			&lastConnected,
			&lastSeen,
			&addr.Priority,
			&addr.Source,
			&createdAt,
			&updatedAt,
			&addr.PeerID,
			&addr.Username,
		)
		if err != nil {
			return nil, err
		}

		if lastConnected.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastConnected.String)
			addr.LastConnected = &t
		}
		if lastSeen.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
			addr.LastSeen = &t
		}
		if createdAt.Valid {
			addr.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
		}
		if updatedAt.Valid {
			addr.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
		}

		addresses = append(addresses, addr)
	}

	return addresses, rows.Err()
}

// GetBootstrapAddresses возвращает bootstrap-адреса
func GetBootstrapAddresses() ([]*models.PeerAddress, error) {
	return GetPeerAddressesByType("bootstrap")
}

// GetContactAddresses возвращает адреса контактов
func GetContactAddresses() ([]*models.PeerAddress, error) {
	return GetPeerAddressesByType("contact")
}

// GetDiscoveredAddresses возвращает адреса обнаруженных пиров
func GetDiscoveredAddresses() ([]*models.PeerAddress, error) {
	return GetPeerAddressesByType("discovered")
}

// AddPeerAddress добавляет адрес пира
func AddPeerAddress(profileID int, multiaddr, addressType, source string) error {
	priority := getPriorityForType(addressType)

	_, err := database.DB.Exec(`
		INSERT INTO peer_addresses (profile_id, multiaddr, address_type, source, priority, is_active)
		VALUES (?, ?, ?, ?, ?, 1)
		ON CONFLICT(profile_id, multiaddr) DO UPDATE SET
			address_type = excluded.address_type,
			last_seen = CURRENT_TIMESTAMP,
			is_active = 1
	`, profileID, multiaddr, addressType, source, priority)

	return err
}

// getPriorityForType возвращает приоритет для типа адреса
func getPriorityForType(addressType string) int {
	switch addressType {
	case "bootstrap":
		return 10
	case "contact":
		return 10
	default: // discovered
		return 0
	}
}

// AddPeerAddressWithProfile создаёт профиль и добавляет адрес
func AddPeerAddressWithProfile(peerID, multiaddr, addressType, source, username string) error {
	// Сначала создаём или получаем профиль
	var profileID int64
	err := database.DB.QueryRow(`
		INSERT INTO profiles (owner_type, peer_id, username, cached_at)
		VALUES ('remote', ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(peer_id) DO UPDATE SET
			username = excluded.username,
			cached_at = CURRENT_TIMESTAMP
		RETURNING id
	`, peerID, username).Scan(&profileID)

	if err != nil {
		return err
	}

	// Добавляем адрес
	return AddPeerAddress(int(profileID), multiaddr, addressType, source)
}

// UpdatePeerAddressLastConnected обновляет время последнего подключения
func UpdatePeerAddressLastConnected(multiaddr string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := database.DB.Exec(`
		UPDATE peer_addresses
		SET last_connected = ?
		WHERE multiaddr = ?
	`, now, multiaddr)
	return err
}

// UpdateProfileLastConnected обновляет время последнего подключения профиля
func UpdateProfileLastConnected(peerID string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := database.DB.Exec(`
		UPDATE profiles
		SET last_connected = ?,
			connection_count = connection_count + 1
		WHERE peer_id = ?
	`, now, peerID)
	return err
}

// DeletePeerAddress удаляет адрес пира
func DeletePeerAddress(multiaddr string) error {
	_, err := database.DB.Exec(`DELETE FROM peer_addresses WHERE multiaddr = ?`, multiaddr)
	return err
}

// DeletePeerAddressByProfile удаляет все адреса профиля
func DeletePeerAddressByProfile(profileID int) error {
	_, err := database.DB.Exec(`DELETE FROM peer_addresses WHERE profile_id = ?`, profileID)
	return err
}

// PeerAddressExists проверяет, существует ли адрес
func PeerAddressExists(multiaddr string) (bool, error) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM peer_addresses WHERE multiaddr = ?`, multiaddr).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SetPeerAddressActive устанавливает активность адреса
func SetPeerAddressActive(multiaddr string, active bool) error {
	_, err := database.DB.Exec(`
		UPDATE peer_addresses
		SET is_active = ?
		WHERE multiaddr = ?
	`, active, multiaddr)
	return err
}

// GetPeerAddressByMultiaddr получает адрес по multiaddr
func GetPeerAddressByMultiaddr(multiaddr string) (*models.PeerAddress, error) {
	row := database.DB.QueryRow(`
		SELECT 
			pa.id,
			pa.profile_id,
			pa.multiaddr,
			pa.address_type,
			pa.is_active,
			pa.last_connected,
			pa.last_seen,
			pa.priority,
			pa.source,
			pa.created_at,
			pa.updated_at,
			p.peer_id,
			p.username
		FROM peer_addresses pa
		JOIN profiles p ON pa.profile_id = p.id
		WHERE pa.multiaddr = ?
	`, multiaddr)

	addr := &models.PeerAddress{}
	var lastConnected, lastSeen sql.NullString
	var createdAt, updatedAt sql.NullString

	err := row.Scan(
		&addr.ID,
		&addr.ProfileID,
		&addr.Multiaddr,
		&addr.AddressType,
		&addr.IsActive,
		&lastConnected,
		&lastSeen,
		&addr.Priority,
		&addr.Source,
		&createdAt,
		&updatedAt,
		&addr.PeerID,
		&addr.Username,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("адрес пира не найден")
		}
		return nil, err
	}

	if lastConnected.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastConnected.String)
		addr.LastConnected = &t
	}
	if lastSeen.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
		addr.LastSeen = &t
	}
	if createdAt.Valid {
		addr.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
	}
	if updatedAt.Valid {
		addr.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
	}

	return addr, nil
}

// GetKnownPeersForExchange возвращает пиров для обмена (кроме контактов)
// Используется при peer exchange для передачи адресов другим пирам
func GetKnownPeersForExchange(excludePeerID string, limit int) ([]*models.PeerAddress, error) {
	rows, err := database.DB.Query(`
		SELECT 
			pa.id,
			pa.profile_id,
			pa.multiaddr,
			pa.address_type,
			pa.is_active,
			pa.last_connected,
			pa.last_seen,
			pa.priority,
			pa.source,
			pa.created_at,
			pa.updated_at,
			p.peer_id,
			p.username
		FROM peer_addresses pa
		JOIN profiles p ON pa.profile_id = p.id
		WHERE pa.is_active = 1
		  AND pa.address_type != 'contact'  -- Контакты НЕ передаём
		  AND p.peer_id != ?
		ORDER BY pa.last_connected DESC
		LIMIT ?
	`, excludePeerID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var addresses []*models.PeerAddress
	for rows.Next() {
		addr := &models.PeerAddress{}
		var lastConnected, lastSeen sql.NullString
		var createdAt, updatedAt sql.NullString

		err := rows.Scan(
			&addr.ID,
			&addr.ProfileID,
			&addr.Multiaddr,
			&addr.AddressType,
			&addr.IsActive,
			&lastConnected,
			&lastSeen,
			&addr.Priority,
			&addr.Source,
			&createdAt,
			&updatedAt,
			&addr.PeerID,
			&addr.Username,
		)
		if err != nil {
			return nil, err
		}

		if lastConnected.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastConnected.String)
			addr.LastConnected = &t
		}
		if lastSeen.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastSeen.String)
			addr.LastSeen = &t
		}
		if createdAt.Valid {
			addr.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt.String)
		}
		if updatedAt.Valid {
			addr.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt.String)
		}

		addresses = append(addresses, addr)
	}

	return addresses, rows.Err()
}
