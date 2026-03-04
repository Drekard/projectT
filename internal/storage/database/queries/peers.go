// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// P2PProfileQueries содержит методы для работы с P2PProfile
type P2PProfileQueries struct{}

// GetP2PProfile получает профиль P2P узла
func GetP2PProfile() (*models.P2PProfile, error) {
	row := database.DB.QueryRow(`
		SELECT id, peer_id, private_key, public_key, is_key_encrypted, username, status, listen_addrs, created_at, updated_at
		FROM p2p_profile
		WHERE id = 1
	`)

	profile := &models.P2PProfile{}
	var privateKey, publicKey []byte
	var listenAddrs sql.NullString
	var createdAt, updatedAt string
	var isKeyEncrypted bool

	err := row.Scan(
		&profile.ID,
		&profile.PeerID,
		&privateKey,
		&publicKey,
		&isKeyEncrypted,
		&profile.Username,
		&profile.Status,
		&listenAddrs,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("P2P профиль не найден")
		}
		return nil, err
	}

	profile.PrivateKey = privateKey
	profile.PublicKey = publicKey
	profile.IsKeyEncrypted = isKeyEncrypted
	if listenAddrs.Valid {
		profile.ListenAddrs = listenAddrs.String
	}
	profile.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	profile.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return profile, nil
}

// CreateP2PProfile создаёт новый профиль P2P узла
func CreateP2PProfile(profile *models.P2PProfile) error {
	_, err := database.DB.Exec(`
		INSERT INTO p2p_profile (id, peer_id, private_key, public_key, is_key_encrypted, username, status, listen_addrs, created_at, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, profile.PeerID, profile.PrivateKey, profile.PublicKey, profile.IsKeyEncrypted, profile.Username, profile.Status, profile.ListenAddrs)
	return err
}

// UpdateP2PProfile обновляет профиль P2P узла
func UpdateP2PProfile(profile *models.P2PProfile) error {
	_, err := database.DB.Exec(`
		UPDATE p2p_profile
		SET peer_id = ?, private_key = ?, public_key = ?, is_key_encrypted = ?, username = ?, status = ?, listen_addrs = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, profile.PeerID, profile.PrivateKey, profile.PublicKey, profile.IsKeyEncrypted, profile.Username, profile.Status, profile.ListenAddrs)
	return err
}

// UpdateP2PProfileField обновляет отдельное поле профиля P2P узла
func UpdateP2PProfileField(field string, value interface{}) error {
	validFields := map[string]bool{
		"peer_id":          true,
		"private_key":      true,
		"public_key":       true,
		"is_key_encrypted": true,
		"username":         true,
		"status":           true,
		"listen_addrs":     true,
	}

	if !validFields[field] {
		return errors.New("недопустимое поле для обновления")
	}

	query := `UPDATE p2p_profile SET ` + field + ` = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1`
	_, err := database.DB.Exec(query, value)
	return err
}

// P2PProfileExists проверяет, существует ли профиль P2P узла
func P2PProfileExists() (bool, error) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM p2p_profile WHERE id = 1`).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsP2PKeyEncrypted проверяет, зашифрован ли приватный ключ в профиле
func IsP2PKeyEncrypted() (bool, error) {
	var isEncrypted bool
	err := database.DB.QueryRow(`SELECT is_key_encrypted FROM p2p_profile WHERE id = 1`).Scan(&isEncrypted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("P2P профиль не найден")
		}
		return false, err
	}
	return isEncrypted, nil
}

// UpdateP2PEncryptionStatus обновляет статус шифрования приватного ключа
func UpdateP2PEncryptionStatus(isEncrypted bool) error {
	_, err := database.DB.Exec(`
		UPDATE p2p_profile
		SET is_key_encrypted = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, isEncrypted)
	return err
}

// ChangeP2PKeyPassword меняет пароль шифрования приватного ключа
// Принимает новые зашифрованные данные и обновляет их в БД
func ChangeP2PKeyPassword(encryptedKey []byte) error {
	_, err := database.DB.Exec(`
		UPDATE p2p_profile
		SET private_key = ?, is_key_encrypted = 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, encryptedKey)
	return err
}
