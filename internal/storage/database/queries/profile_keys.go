// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetProfileKeys возвращает ключи профиля по ID
func GetProfileKeys(profileID int) (*models.ProfileKey, error) {
	query := `
		SELECT profile_id, private_key, public_key, signature, is_key_encrypted
		FROM profile_keys
		WHERE profile_id = ?
		LIMIT 1
	`
	var keys models.ProfileKey

	err := database.DB.QueryRow(query, profileID).Scan(
		&keys.ProfileID,
		&keys.PrivateKey,
		&keys.PublicKey,
		&keys.Signature,
		&keys.IsKeyEncrypted,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("ключи профиля не найдены")
		}
		return nil, err
	}

	return &keys, nil
}

// CreateProfileKeys создаёт ключи профиля
func CreateProfileKeys(keys *models.ProfileKey) error {
	query := `
		INSERT INTO profile_keys (profile_id, private_key, public_key, signature, is_key_encrypted)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := database.DB.Exec(query,
		keys.ProfileID, keys.PrivateKey, keys.PublicKey, keys.Signature, keys.IsKeyEncrypted,
	)
	return err
}

// UpdateProfileKeys обновляет ключи профиля
func UpdateProfileKeys(keys *models.ProfileKey) error {
	query := `
		UPDATE profile_keys
		SET private_key = ?, public_key = ?, signature = ?, is_key_encrypted = ?
		WHERE profile_id = ?
	`
	_, err := database.DB.Exec(query,
		keys.PrivateKey, keys.PublicKey, keys.Signature, keys.IsKeyEncrypted, keys.ProfileID,
	)
	return err
}

// UpdateProfileKeyField обновляет отдельное поле ключей профиля
func UpdateProfileKeyField(profileID int, field string, value interface{}) error {
	validFields := map[string]bool{
		"private_key":      true,
		"public_key":       true,
		"signature":        true,
		"is_key_encrypted": true,
	}

	if !validFields[field] {
		return errors.New("недопустимое поле для обновления: " + field)
	}

	query := `UPDATE profile_keys SET ` + field + ` = ? WHERE profile_id = ?`
	_, err := database.DB.Exec(query, value, profileID)
	return err
}

// DeleteProfileKeys удаляет ключи профиля
func DeleteProfileKeys(profileID int) error {
	_, err := database.DB.Exec(`DELETE FROM profile_keys WHERE profile_id = ?`, profileID)
	return err
}

// ProfileKeysExists проверяет, существуют ли ключи у профиля
func ProfileKeysExists(profileID int) (bool, error) {
	var exists bool
	err := database.DB.QueryRow(`SELECT COUNT(*) > 0 FROM profile_keys WHERE profile_id = ?`, profileID).Scan(&exists)
	return exists, err
}

// IsProfileKeyEncrypted проверяет, зашифрован ли приватный ключ профиля
func IsProfileKeyEncrypted(profileID int) (bool, error) {
	var isEncrypted bool
	err := database.DB.QueryRow(`SELECT is_key_encrypted FROM profile_keys WHERE profile_id = ?`, profileID).Scan(&isEncrypted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("ключи профиля не найдены")
		}
		return false, err
	}
	return isEncrypted, nil
}

// UpdateProfileEncryptionStatus обновляет статус шифрования приватного ключа
func UpdateProfileEncryptionStatus(profileID int, isEncrypted bool) error {
	_, err := database.DB.Exec(`
		UPDATE profile_keys
		SET is_key_encrypted = ?
		WHERE profile_id = ?
	`, isEncrypted, profileID)
	return err
}
