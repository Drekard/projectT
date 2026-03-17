package filesystem

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
)

// CalculateHash вычисляет SHA-256 хэш для переданных байтов и возвращает его в виде hex-строки
func CalculateHash(fileBytes []byte) string {
	hash := sha256.Sum256(fileBytes)
	return hex.EncodeToString(hash[:])
}

// GenerateContentHash генерирует хэш содержимого элемента на основе title, description и content_meta
// Используется для дедупликации и идентификации элементов при обмене между пирами
func GenerateContentHash(title, description, contentMeta string) string {
	// Формируем строку для хэширования: title|description|content_meta
	data := fmt.Sprintf("%s|%s|%s", title, description, contentMeta)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GenerateElementUUID генерирует уникальный UUID v4 для элемента
// Используется как основной идентификатор для P2P синхронизации
func GenerateElementUUID() string {
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		// Fallback на детерминированный UUID в случае ошибки
		panic(fmt.Sprintf("failed to generate UUID: %v", err))
	}

	// Устанавливаем версию (4) и вариант (10xx)
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 10xx

	// Форматируем как строку: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// IsValidUUID проверяет, что строка похожа на валидный UUID v4
func IsValidUUID(uuid string) bool {
	// UUID v4 имеет формат: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 символов)
	if len(uuid) != 36 {
		return false
	}

	// Проверяем формат: 8-4-4-4-12 hex символов с дефисами
	matched, err := regexp.MatchString("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$", uuid)
	if err != nil {
		return false
	}

	return matched
}

// IsValidHash проверяет, что строка похожа на валидный SHA-256 хэш (64 символа в hex формате)
func IsValidHash(hash string) bool {
	// SHA-256 хэш в hex формате всегда имеет длину 64 символа
	if len(hash) != 64 {
		return false
	}

	// Проверяем, что строка содержит только шестнадцатеричные символы
	matched, err := regexp.MatchString("^[a-fA-F0-9]+$", hash)
	if err != nil {
		return false
	}

	return matched
}

// GetHashPrefix возвращает первые 2 символа хэша (для организации подпапок)
func GetHashPrefix(hash string) string {
	if len(hash) < 2 {
		return hash
	}
	return hash[:2]
}
