package models

import "time"

// Profile представляет профиль пользователя
type Profile struct {
	ID                    int       `json:"id"`
	Username              string    `json:"username"`
	Status                string    `json:"status"`
	AvatarPath            string    `json:"avatar_path"`
	BackgroundPath        string    `json:"background_path"`
	ContentCharacteristic string    `json:"content_characteristic"`
	DemoElements          string    `json:"demo_elements"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
