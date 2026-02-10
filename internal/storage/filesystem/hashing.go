package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

// CalculateHash вычисляет SHA-256 хэш для переданных байтов и возвращает его в виде hex-строки
func CalculateHash(fileBytes []byte) string {
	hash := sha256.Sum256(fileBytes)
	return hex.EncodeToString(hash[:])
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
