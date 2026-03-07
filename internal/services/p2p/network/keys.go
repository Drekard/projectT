// Package network предоставляет функции для работы с криптографическими ключами
package network

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"

	p2pcrypto "projectT/internal/services/crypto"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// KeyManager управляет ключами шифрования
type KeyManager struct{}

// NewKeyManager создаёт менеджер ключей
func NewKeyManager() *KeyManager {
	return &KeyManager{}
}

// GenerateKeyPair генерирует пару ключей Ed25519
func GenerateKeyPair() (crypto.PrivKey, crypto.PubKey, error) {
	privKey, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privKey, pubKey, nil
}

// VerifyPassword проверяет правильность пароля для расшифровки приватного ключа
func VerifyPassword(password string) (bool, error) {
	profile, err := queries.GetP2PProfile()
	if err != nil {
		return false, fmt.Errorf("ошибка загрузки профиля: %w", err)
	}

	if !profile.IsKeyEncrypted {
		// Ключ не зашифрован — пароль не требуется
		return true, nil
	}

	// Проверяем пароль
	if !p2pcrypto.VerifyPassword(profile.PrivateKey, password) {
		return false, errors.New("неверный пароль")
	}

	return true, nil
}

// ChangePassword меняет пароль шифрования приватного ключа
func ChangePassword(oldPassword, newPassword string, privKey crypto.PrivKey) error {
	profile, err := queries.GetP2PProfile()
	if err != nil {
		return fmt.Errorf("ошибка загрузки профиля: %w", err)
	}

	if !profile.IsKeyEncrypted {
		// Если ключ не зашифрован, просто шифруем новым паролем
		privKeyRaw, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			return fmt.Errorf("ошибка сериализации ключа: %w", err)
		}
		encryptedKey, err := p2pcrypto.EncryptPrivateKey(privKeyRaw, newPassword)
		if err != nil {
			return fmt.Errorf("ошибка шифрования ключа: %w", err)
		}
		return queries.ChangeP2PKeyPassword(encryptedKey)
	}

	// Расшифровываем старым паролем и шифруем новым
	newEncryptedKey, err := p2pcrypto.ChangePassword(profile.PrivateKey, oldPassword, newPassword)
	if err != nil {
		return fmt.Errorf("ошибка смены пароля: %w", err)
	}

	return queries.ChangeP2PKeyPassword(newEncryptedKey)
}

// IsKeyEncrypted возвращает true, если приватный ключ зашифрован
func IsKeyEncrypted() (bool, error) {
	return queries.IsP2PKeyEncrypted()
}

// EnableEncryption включает шифрование приватного ключа с заданным паролем
func EnableEncryption(profile *models.P2PProfile, password string) error {
	if profile.IsKeyEncrypted {
		return errors.New("ключ уже зашифрован")
	}

	// Шифруем приватный ключ
	encryptedKey, err := p2pcrypto.EncryptPrivateKey(profile.PrivateKey, password)
	if err != nil {
		return fmt.Errorf("ошибка шифрования ключа: %w", err)
	}

	return queries.ChangeP2PKeyPassword(encryptedKey)
}
