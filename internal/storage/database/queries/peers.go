// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"errors"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// P2PProfileQueries содержит методы для работы с P2PProfile
type P2PProfileQueries struct{}

// GetP2PProfile получает профиль P2P узла из таблиц profiles + profile_keys
func GetP2PProfile() (*models.P2PProfile, error) {
	// Загружаем профиль из profiles (локальный)
	localProfile, err := GetLocalProfile()
	if err != nil {
		return nil, errors.New("P2P профиль не найден: " + err.Error())
	}

	// Загружаем ключи из profile_keys
	keys, err := GetProfileKeys(localProfile.ID)
	if err != nil {
		return nil, errors.New("ключи P2P профиля не найдены: " + err.Error())
	}

	// Возвращаем объединённый P2PProfile
	return &models.P2PProfile{
		ID:             localProfile.ID,
		PeerID:         localProfile.PeerID,
		PrivateKey:     keys.PrivateKey,
		PublicKey:      keys.PublicKey,
		IsKeyEncrypted: keys.IsKeyEncrypted,
		Username:       localProfile.Username,
		Title:          localProfile.Title,
		ListenAddrs:    "", // TODO: добавить listen_addrs в profiles или отдельную таблицу
		CreatedAt:      localProfile.CreatedAt,
		UpdatedAt:      localProfile.UpdatedAt,
	}, nil
}

// CreateP2PProfile создаёт новый профиль P2P узла в profiles + profile_keys
func CreateP2PProfile(profile *models.P2PProfile) error {
	// Создаём локальный профиль в profiles
	localProfile := &models.Profile{
		OwnerType: models.OwnerTypeLocal,
		PeerID:    profile.PeerID,
		Username:  profile.Username,
		Title:     profile.Title,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}

	// Вставляем профиль и получаем ID
	query := `
		INSERT INTO profiles (owner_type, peer_id, username, title, created_at, updated_at)
		VALUES ('local', ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	result, err := database.DB.Exec(query, localProfile.PeerID, localProfile.Username, localProfile.Title)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	localProfile.ID = int(id)

	// Создаём ключи в profile_keys
	keys := &models.ProfileKey{
		ProfileID:      localProfile.ID,
		PrivateKey:     profile.PrivateKey,
		PublicKey:      profile.PublicKey,
		IsKeyEncrypted: profile.IsKeyEncrypted,
	}

	return CreateProfileKeys(keys)
}

// UpdateP2PProfile обновляет профиль P2P узла в profiles + profile_keys
func UpdateP2PProfile(profile *models.P2PProfile) error {
	// Обновляем профиль в profiles
	err := UpdateLocalProfile(&models.Profile{
		ID:        profile.ID,
		PeerID:    profile.PeerID,
		Username:  profile.Username,
		Title:     profile.Title,
		UpdatedAt: profile.UpdatedAt,
	})
	if err != nil {
		return err
	}

	// Обновляем ключи в profile_keys
	keys := &models.ProfileKey{
		ProfileID:      profile.ID,
		PrivateKey:     profile.PrivateKey,
		PublicKey:      profile.PublicKey,
		IsKeyEncrypted: profile.IsKeyEncrypted,
	}

	return UpdateProfileKeys(keys)
}

// UpdateP2PProfileField обновляет отдельное поле профиля P2P узла
func UpdateP2PProfileField(field string, value interface{}) error {
	// Поля профиля (profiles)
	profileFields := map[string]bool{
		"peer_id":      true,
		"username":     true,
		"title":        true,
		"listen_addrs": true, // TODO: реализовать после добавления колонки
	}

	// Поля ключей (profile_keys)
	keyFields := map[string]bool{
		"private_key":      true,
		"public_key":       true,
		"is_key_encrypted": true,
	}

	// Получаем ID локального профиля
	localProfile, err := GetLocalProfile()
	if err != nil {
		return errors.New("не удалось получить локальный профиль: " + err.Error())
	}

	if profileFields[field] {
		// Особая обработка для listen_addrs (пока не реализовано)
		if field == "listen_addrs" {
			// TODO: добавить колонку listen_addrs в profiles
			return errors.New("поле listen_addrs временно недоступно")
		}
		return UpdateLocalProfileField(field, value)
	}

	if keyFields[field] {
		return UpdateProfileKeyField(localProfile.ID, field, value)
	}

	return errors.New("недопустимое поле для обновления: " + field)
}

// P2PProfileExists проверяет, существует ли профиль P2P узла
func P2PProfileExists() (bool, error) {
	// Проверяем наличие локального профиля
	localProfile, err := GetLocalProfile()
	if err != nil {
		return false, nil
	}

	// Проверяем наличие ключей
	return ProfileKeysExists(localProfile.ID)
}

// IsP2PKeyEncrypted проверяет, зашифрован ли приватный ключ в профиле
func IsP2PKeyEncrypted() (bool, error) {
	localProfile, err := GetLocalProfile()
	if err != nil {
		return false, errors.New("P2P профиль не найден")
	}

	return IsProfileKeyEncrypted(localProfile.ID)
}

// UpdateP2PEncryptionStatus обновляет статус шифрования приватного ключа
func UpdateP2PEncryptionStatus(isEncrypted bool) error {
	localProfile, err := GetLocalProfile()
	if err != nil {
		return errors.New("не удалось получить локальный профиль: " + err.Error())
	}

	return UpdateProfileEncryptionStatus(localProfile.ID, isEncrypted)
}

// ChangeP2PKeyPassword меняет пароль шифрования приватного ключа
// Принимает новые зашифрованные данные и обновляет их в БД
func ChangeP2PKeyPassword(encryptedKey []byte) error {
	localProfile, err := GetLocalProfile()
	if err != nil {
		return errors.New("не удалось получить локальный профиль: " + err.Error())
	}

	return UpdateProfileKeyField(localProfile.ID, "private_key", encryptedKey)
}
