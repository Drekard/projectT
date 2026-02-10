package queries

import (
	"fmt"
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
	"time"
)

// GetProfile возвращает профиль пользователя
func GetProfile() (*models.Profile, error) {
	query := `SELECT id, username, status, avatar_path, background_path, created_at, updated_at FROM profile LIMIT 1`
	var profile models.Profile
	err := database.DB.QueryRow(query).Scan(
		&profile.ID, &profile.Username, &profile.Status, &profile.AvatarPath, &profile.BackgroundPath, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		// Если нет профиля в базе, создаем профиль по умолчанию
		if err.Error() == "sql: no rows in result set" {
			defaultProfile := &models.Profile{
				Username:       "Аноним",
				Status:         "Доступен",
				AvatarPath:     "",
				BackgroundPath: "",
			}
			err = CreateProfile(defaultProfile)
			if err != nil {
				return nil, err
			}
			// Возвращаем созданный профиль
			return defaultProfile, nil
		}
		return nil, err
	}
	return &profile, nil
}

// CreateProfile создает новый профиль пользователя
func CreateProfile(profile *models.Profile) error {
	query := `INSERT INTO profile (username, status, avatar_path, background_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := database.DB.Exec(query, profile.Username, profile.Status, profile.AvatarPath, profile.BackgroundPath, time.Now(), time.Now())
	if err != nil {
		return err
	}

	// Получаем ID вставленной записи
	var id int64
	id, err = result.LastInsertId()
	if err != nil {
		return err
	}
	profile.ID = int(id)
	return nil
}

// UpdateProfile обновляет профиль пользователя
func UpdateProfile(profile *models.Profile) error {
	query := `UPDATE profile SET username = ?, status = ?, avatar_path = ?, background_path = ?, updated_at = ? WHERE id = ?`
	_, err := database.DB.Exec(query, profile.Username, profile.Status, profile.AvatarPath, profile.BackgroundPath, time.Now(), profile.ID)
	return err
}

// UpdateProfileField обновляет конкретное поле профиля
func UpdateProfileField(field string, value interface{}, profileID int) error {
	var query string
	switch field {
	case "username":
		query = `UPDATE profile SET username = ?, updated_at = ? WHERE id = ?`
	case "status":
		query = `UPDATE profile SET status = ?, updated_at = ? WHERE id = ?`
	case "avatar_path":
		query = `UPDATE profile SET avatar_path = ?, updated_at = ? WHERE id = ?`
	case "background_path":
		query = `UPDATE profile SET background_path = ?, updated_at = ? WHERE id = ?`
	default:
		return fmt.Errorf("неподдерживаемое поле: %s", field)
	}
	_, err := database.DB.Exec(query, value, time.Now(), profileID)
	return err
}

// InitializeDefaultProfile инициализирует профиль по умолчанию, если он не существует
func InitializeDefaultProfile() error {
	// Проверяем, существует ли уже профиль
	query := `SELECT COUNT(*) FROM profile`
	var count int
	err := database.DB.QueryRow(query).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Создаем профиль по умолчанию
		defaultProfile := &models.Profile{
			Username:       "Аноним",
			Status:         "Доступен",
			AvatarPath:     "",
			BackgroundPath: "",
		}
		return CreateProfile(defaultProfile)
	}

	return nil
}
