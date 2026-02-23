// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetBootstrapPeer получает bootstrap-узел по ID
func GetBootstrapPeer(id int) (*models.BootstrapPeer, error) {
	row := database.DB.QueryRow(`
		SELECT id, multiaddr, peer_id, is_active, last_connected, added_at
		FROM bootstrap_peers
		WHERE id = ?
	`, id)

	peer := &models.BootstrapPeer{}
	var lastConnected, addedAt sql.NullString

	err := row.Scan(
		&peer.ID,
		&peer.Multiaddr,
		&peer.PeerID,
		&peer.IsActive,
		&lastConnected,
		&addedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("bootstrap-узел не найден")
		}
		return nil, err
	}

	if lastConnected.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastConnected.String)
		peer.LastConnected = &t
	}
	if addedAt.Valid {
		peer.AddedAt, _ = time.Parse("2006-01-02 15:04:05", addedAt.String)
	}

	return peer, nil
}

// GetAllBootstrapPeers получает все bootstrap-узлы
func GetAllBootstrapPeers() ([]*models.BootstrapPeer, error) {
	rows, err := database.DB.Query(`
		SELECT id, multiaddr, peer_id, is_active, last_connected, added_at
		FROM bootstrap_peers
		ORDER BY added_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*models.BootstrapPeer
	for rows.Next() {
		peer := &models.BootstrapPeer{}
		var lastConnected, addedAt sql.NullString

		err := rows.Scan(
			&peer.ID,
			&peer.Multiaddr,
			&peer.PeerID,
			&peer.IsActive,
			&lastConnected,
			&addedAt,
		)
		if err != nil {
			return nil, err
		}

		if lastConnected.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastConnected.String)
			peer.LastConnected = &t
		}
		if addedAt.Valid {
			peer.AddedAt, _ = time.Parse("2006-01-02 15:04:05", addedAt.String)
		}

		peers = append(peers, peer)
	}

	return peers, rows.Err()
}

// GetActiveBootstrapPeers получает активные bootstrap-узлы
func GetActiveBootstrapPeers() ([]*models.BootstrapPeer, error) {
	rows, err := database.DB.Query(`
		SELECT id, multiaddr, peer_id, is_active, last_connected, added_at
		FROM bootstrap_peers
		WHERE is_active = 1
		ORDER BY added_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*models.BootstrapPeer
	for rows.Next() {
		peer := &models.BootstrapPeer{}
		var lastConnected, addedAt sql.NullString

		err := rows.Scan(
			&peer.ID,
			&peer.Multiaddr,
			&peer.PeerID,
			&peer.IsActive,
			&lastConnected,
			&addedAt,
		)
		if err != nil {
			return nil, err
		}

		if lastConnected.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastConnected.String)
			peer.LastConnected = &t
		}
		if addedAt.Valid {
			peer.AddedAt, _ = time.Parse("2006-01-02 15:04:05", addedAt.String)
		}

		peers = append(peers, peer)
	}

	return peers, rows.Err()
}

// CreateBootstrapPeer создаёт новый bootstrap-узел
func CreateBootstrapPeer(peer *models.BootstrapPeer) error {
	var peerID interface{}
	if peer.PeerID.Valid {
		peerID = peer.PeerID.String
	} else {
		peerID = nil
	}

	result, err := database.DB.Exec(`
		INSERT INTO bootstrap_peers (multiaddr, peer_id, is_active, last_connected, added_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, peer.Multiaddr, peerID, peer.IsActive, peer.LastConnected)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	peer.ID = int(id)
	return nil
}

// UpdateBootstrapPeer обновляет bootstrap-узел
func UpdateBootstrapPeer(peer *models.BootstrapPeer) error {
	var peerID interface{}
	if peer.PeerID.Valid {
		peerID = peer.PeerID.String
	} else {
		peerID = nil
	}

	_, err := database.DB.Exec(`
		UPDATE bootstrap_peers
		SET multiaddr = ?, peer_id = ?, is_active = ?, last_connected = ?
		WHERE id = ?
	`, peer.Multiaddr, peerID, peer.IsActive, peer.LastConnected, peer.ID)
	return err
}

// UpdateBootstrapPeerLastConnected обновляет время последнего подключения
func UpdateBootstrapPeerLastConnected(multiaddr string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := database.DB.Exec(`
		UPDATE bootstrap_peers
		SET last_connected = ?
		WHERE multiaddr = ?
	`, now, multiaddr)
	return err
}

// DeleteBootstrapPeer удаляет bootstrap-узел по ID
func DeleteBootstrapPeer(id int) error {
	_, err := database.DB.Exec(`DELETE FROM bootstrap_peers WHERE id = ?`, id)
	return err
}

// DeleteBootstrapPeerByMultiaddr удаляет bootstrap-узел по Multiaddr
func DeleteBootstrapPeerByMultiaddr(multiaddr string) error {
	_, err := database.DB.Exec(`DELETE FROM bootstrap_peers WHERE multiaddr = ?`, multiaddr)
	return err
}

// BootstrapPeerExists проверяет, существует ли bootstrap-узел
func BootstrapPeerExists(multiaddr string) (bool, error) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM bootstrap_peers WHERE multiaddr = ?`, multiaddr).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SetBootstrapPeerActive устанавливает активность bootstrap-узла
func SetBootstrapPeerActive(id int, active bool) error {
	_, err := database.DB.Exec(`
		UPDATE bootstrap_peers
		SET is_active = ?
		WHERE id = ?
	`, active, id)
	return err
}
