//go:build linux || darwin

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormalizePaths_Absolute проверяет что абсолютные пути не меняются (Linux/macOS)
func TestNormalizePaths_Absolute(t *testing.T) {
	loader := NewLoader()

	// Устанавливаем абсолютные пути
	expectedDbPath := "/test/db.sqlite"
	expectedStoragePath := "/test/storage"

	loader.config.Database.Path = expectedDbPath
	loader.config.Storage.Path = expectedStoragePath

	// Нормализуем
	loader.normalizePaths()

	// На Linux/macOS пути должны остаться неизменными
	assert.Equal(t, expectedDbPath, loader.config.Database.Path)
	assert.Equal(t, expectedStoragePath, loader.config.Storage.Path)
}
