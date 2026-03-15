// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// GetLocalProfile возвращает локальный профиль пользователя
func GetLocalProfile() (*models.Profile, error) {
	query := `
		SELECT id, owner_type, peer_id, username, title,
		       COALESCE(avatar_path, ''),
		       COALESCE(background_path, ''),
		       COALESCE(content_char, ''),
		       COALESCE(demo_elements, ''),
		       cached_at, created_at, updated_at
		FROM profiles
		WHERE owner_type = 'local'
		LIMIT 1
	`
	var profile models.Profile
	var cachedAt sql.NullString
	var createdAt, updatedAt string

	err := database.DB.QueryRow(query).Scan(
		&profile.ID, &profile.OwnerType, &profile.PeerID, &profile.Username,
		&profile.Title, &profile.AvatarPath, &profile.BackgroundPath,
		&profile.ContentChar, &profile.DemoElements, &cachedAt,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("локальный профиль не найден")
		}
		return nil, err
	}

	profile.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	profile.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &profile, nil
}

// GetRemoteProfile возвращает чужой профиль по PeerID
func GetRemoteProfile(peerID string) (*models.Profile, error) {
	query := `
		SELECT id, owner_type, peer_id, username, title,
		       COALESCE(avatar_path, ''),
		       COALESCE(background_path, ''),
		       COALESCE(content_char, ''),
		       COALESCE(demo_elements, ''),
		       cached_at, created_at, updated_at
		FROM profiles
		WHERE peer_id = ? AND owner_type = 'remote'
		LIMIT 1
	`
	var profile models.Profile
	var cachedAt sql.NullString
	var createdAt, updatedAt string

	err := database.DB.QueryRow(query).Scan(
		&profile.ID, &profile.OwnerType, &profile.PeerID, &profile.Username,
		&profile.Title, &profile.AvatarPath, &profile.BackgroundPath,
		&profile.ContentChar, &profile.DemoElements, &cachedAt,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("профиль пира не найден")
		}
		return nil, err
	}

	if cachedAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", cachedAt.String)
		profile.CachedAt = &t
	}
	profile.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	profile.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &profile, nil
}

// EnsureProfileForContact создаёт профиль для контакта если он ещё не существует
// Используется при добавлении нового контакта в contacts
func EnsureProfileForContact(peerID, username, avatarPath string) error {
	// Проверяем, существует ли уже профиль
	exists := false
	err := database.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM profiles WHERE peer_id = ?)
	`, peerID).Scan(&exists)
	if err != nil {
		return err
	}

	// Если профиль уже существует - ничего не делаем
	if exists {
		return nil
	}

	// Создаём новый remote профиль
	_, err = database.DB.Exec(`
		INSERT INTO profiles (owner_type, peer_id, username, title, avatar_path,
		                      background_path, content_char, demo_elements,
		                      created_at, updated_at)
		VALUES ('remote', ?, ?, ?, ?, '', '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, peerID, username, avatarPath)

	return err
}

// GetAllRemoteProfiles возвращает все чужие профили
func GetAllRemoteProfiles() ([]*models.Profile, error) {
	query := `
		SELECT id, owner_type, peer_id, username, title, avatar_path, background_path,
		       content_char, demo_elements, cached_at, created_at, updated_at
		FROM profiles
		WHERE owner_type = 'remote'
		ORDER BY username
	`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*models.Profile
	for rows.Next() {
		var profile models.Profile
		var cachedAt sql.NullString
		var createdAt, updatedAt string

		err := rows.Scan(
			&profile.ID, &profile.OwnerType, &profile.PeerID, &profile.Username,
			&profile.Title, &profile.AvatarPath, &profile.BackgroundPath,
			&profile.ContentChar, &profile.DemoElements, &cachedAt,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if cachedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", cachedAt.String)
			profile.CachedAt = &t
		}
		profile.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		profile.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		profiles = append(profiles, &profile)
	}

	return profiles, rows.Err()
}

// CreateRemoteProfile создаёт чужой профиль
func CreateRemoteProfile(profile *models.Profile) error {
	query := `
		INSERT INTO profiles (owner_type, peer_id, username, title, avatar_path, background_path,
		                      content_char, demo_elements, cached_at, created_at, updated_at)
		VALUES ('remote', ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	result, err := database.DB.Exec(query,
		profile.PeerID, profile.Username, profile.Title, profile.AvatarPath,
		profile.BackgroundPath, profile.ContentChar, profile.DemoElements,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	profile.ID = int(id)
	return nil
}

// UpdateRemoteProfile обновляет чужой профиль
func UpdateRemoteProfile(profile *models.Profile) error {
	query := `
		UPDATE profiles
		SET username = ?, title = ?, avatar_path = ?, background_path = ?,
		    content_char = ?, demo_elements = ?, cached_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE peer_id = ? AND owner_type = 'remote'
	`
	_, err := database.DB.Exec(query,
		profile.Username, profile.Title, profile.AvatarPath, profile.BackgroundPath,
		profile.ContentChar, profile.DemoElements, profile.PeerID,
	)
	return err
}

// UpdateLocalProfile обновляет локальный профиль
func UpdateLocalProfile(profile *models.Profile) error {
	query := `
		UPDATE profiles
		SET username = ?, title = ?, avatar_path = ?, background_path = ?,
		    content_char = ?, demo_elements = ?, updated_at = CURRENT_TIMESTAMP
		WHERE owner_type = 'local'
	`
	_, err := database.DB.Exec(query,
		profile.Username, profile.Title, profile.AvatarPath, profile.BackgroundPath,
		profile.ContentChar, profile.DemoElements,
	)
	return err
}

// UpdateLocalProfileField обновляет конкретное поле локального профиля
func UpdateLocalProfileField(field string, value interface{}) error {
	var query string
	switch field {
	case "username":
		query = `UPDATE profiles SET username = ?, updated_at = CURRENT_TIMESTAMP WHERE owner_type = 'local'`
	case "title":
		query = `UPDATE profiles SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE owner_type = 'local'`
	case "avatar_path":
		query = `UPDATE profiles SET avatar_path = ?, updated_at = CURRENT_TIMESTAMP WHERE owner_type = 'local'`
	case "background_path":
		query = `UPDATE profiles SET background_path = ?, updated_at = CURRENT_TIMESTAMP WHERE owner_type = 'local'`
	case "content_char":
		query = `UPDATE profiles SET content_char = ?, updated_at = CURRENT_TIMESTAMP WHERE owner_type = 'local'`
	case "demo_elements":
		query = `UPDATE profiles SET demo_elements = ?, updated_at = CURRENT_TIMESTAMP WHERE owner_type = 'local'`
	default:
		return errors.New("неподдерживаемое поле: " + field)
	}
	_, err := database.DB.Exec(query, value)
	return err
}

// DeleteRemoteProfile удаляет чужой профиль по PeerID
func DeleteRemoteProfile(peerID string) error {
	_, err := database.DB.Exec(`DELETE FROM profiles WHERE peer_id = ? AND owner_type = 'remote'`, peerID)
	return err
}

// ProfileExists проверяет, существует ли профиль с указанным PeerID
func ProfileExists(peerID string) (bool, error) {
	var exists bool
	err := database.DB.QueryRow(`SELECT COUNT(*) > 0 FROM profiles WHERE peer_id = ?`, peerID).Scan(&exists)
	return exists, err
}

// GetProfileByPeerID возвращает профиль (локальный или чужой) по PeerID
func GetProfileByPeerID(peerID string) (*models.Profile, error) {
	query := `
		SELECT id, owner_type, peer_id, username, title,
		       COALESCE(avatar_path, ''),
		       COALESCE(background_path, ''),
		       COALESCE(content_char, ''),
		       COALESCE(demo_elements, ''),
		       cached_at, created_at, updated_at
		FROM profiles
		WHERE peer_id = ?
		LIMIT 1
	`
	var profile models.Profile
	var cachedAt sql.NullString
	var createdAt, updatedAt string

	err := database.DB.QueryRow(query).Scan(
		&profile.ID, &profile.OwnerType, &profile.PeerID, &profile.Username,
		&profile.Title, &profile.AvatarPath, &profile.BackgroundPath,
		&profile.ContentChar, &profile.DemoElements, &cachedAt,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("профиль не найден")
		}
		return nil, err
	}

	if cachedAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", cachedAt.String)
		profile.CachedAt = &t
	}
	profile.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	profile.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &profile, nil
}
