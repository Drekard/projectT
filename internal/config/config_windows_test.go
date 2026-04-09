//go:build windows

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormalizePaths_Absolute проверяет что абсолютные пути не меняются (Windows)
func TestNormalizePaths_Absolute(t *testing.T) {
	loader := NewLoader()

	// Устанавливаем абсолютные пути
	expectedDbPath := `C:\test\db.sqlite`
	expectedStoragePath := `D:\test\storage`

	loader.config.Database.Path = expectedDbPath
	loader.config.Storage.Path = expectedStoragePath

	// Нормализуем
	loader.normalizePaths()

	// На Windows пути должны остаться неизменными
	assert.Equal(t, expectedDbPath, loader.config.Database.Path)
	assert.Equal(t, expectedStoragePath, loader.config.Storage.Path)
}
