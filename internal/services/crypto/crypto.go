// Package crypto предоставляет функции шифрования для защиты приватных ключей.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize   = 16
	nonceSize  = 12
	iterations = 100000
)

// EncryptedKeyMarker маркер для определения зашифрованных данных
// Первые 4 байта зашифрованного ключа содержат соль, но мы добавим префикс
// для надёжного определения формата
var EncryptedKeyMarker = []byte{0x54, 0x50, 0x4B, 0x45} // "TPKE" - T: ProjectT, PKE: Private Key Encrypted

// EncryptPrivateKey шифрует приватный ключ с использованием пароля.
// Возвращает зашифрованные данные в формате: marker (4) + salt (16) + ciphertext.
func EncryptPrivateKey(privateKey []byte, password string) ([]byte, error) {
	// Генерируем случайную соль
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("ошибка генерации соли: %w", err)
	}

	// Деривируем ключ из пароля
	key := deriveKey(password, salt)

	// Создаём AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания шифра: %w", err)
	}

	// Создаём GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания GCM: %w", err)
	}

	// Генерируем nonce
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("ошибка генерации nonce: %w", err)
	}

	// Шифруем
	ciphertext := gcm.Seal(nonce, nonce, privateKey, nil)

	// Возвращаем marker + соль + зашифрованные данные
	result := append(EncryptedKeyMarker, salt...)
	result = append(result, ciphertext...)
	return result, nil
}

// DecryptPrivateKey расшифровывает приватный ключ с использованием пароля.
// Ожидает формат: marker (4) + salt (16) + ciphertext (nonce + encrypted).
func DecryptPrivateKey(encryptedData []byte, password string) ([]byte, error) {
	// Проверяем наличие маркера
	if !IsEncryptedKey(encryptedData) {
		return nil, errors.New("данные не зашифрованы или имеют неверный формат")
	}

	// Пропускаем маркер (4 байта)
	data := encryptedData[len(EncryptedKeyMarker):]

	// Извлекаем соль
	salt := data[:saltSize]
	ciphertext := data[saltSize:]

	// Деривируем ключ из пароля
	key := deriveKey(password, salt)

	// Создаём AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания шифра: %w", err)
	}

	// Создаём GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания GCM: %w", err)
	}

	// Извлекаем nonce и ciphertext
	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	// Расшифровываем
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка расшифровки: неверный пароль или повреждённые данные")
	}

	return plaintext, nil
}

// IsEncryptedKey проверяет, является ли ключ зашифрованным (по маркеру).
func IsEncryptedKey(data []byte) bool {
	if len(data) < len(EncryptedKeyMarker) {
		return false
	}
	for i := 0; i < len(EncryptedKeyMarker); i++ {
		if data[i] != EncryptedKeyMarker[i] {
			return false
		}
	}
	return true
}

// VerifyPassword проверяет правильность пароля без расшифровки ключа.
// Возвращает true, если пароль верный.
func VerifyPassword(encryptedData []byte, password string) bool {
	if !IsEncryptedKey(encryptedData) {
		return false
	}

	// Пробуем расшифровать — если успешно, пароль верный
	_, err := DecryptPrivateKey(encryptedData, password)
	return err == nil
}

// ChangePassword меняет пароль шифрования приватного ключа.
// Принимает зашифрованные данные со старым паролем и новый пароль.
// Возвращает новые зашифрованные данные.
func ChangePassword(encryptedData []byte, oldPassword, newPassword string) ([]byte, error) {
	// Расшифровываем старым паролем
	privateKey, err := DecryptPrivateKey(encryptedData, oldPassword)
	if err != nil {
		return nil, fmt.Errorf("ошибка расшифровки старым паролем: %w", err)
	}

	// Шифруем новым паролем
	return EncryptPrivateKey(privateKey, newPassword)
}

// deriveKey деривирует 256-битный ключ из пароля с использованием PBKDF2.
func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)
}
