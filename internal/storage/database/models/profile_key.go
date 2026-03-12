// Package models содержит модели данных для работы с базой данных.
package models

// ProfileKey представляет криптографические ключи профиля
type ProfileKey struct {
	ProfileID      int    `json:"profile_id"`
	PrivateKey     []byte `json:"-"` // Приватный ключ (только для local)
	PublicKey      []byte `json:"public_key"`
	Signature      []byte `json:"signature,omitempty"` // Подпись профиля
	IsKeyEncrypted bool   `json:"is_key_encrypted"`    // Зашифрован ли приватный ключ
}
