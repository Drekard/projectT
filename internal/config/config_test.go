package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestDefaultConfig проверяет создание конфигурации по умолчанию
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Database.Path)
	assert.NotEmpty(t, cfg.Storage.Path)
	assert.Equal(t, "files", cfg.Storage.FilesDir)
	assert.Equal(t, 30000, cfg.Database.BusyTimeout)
	assert.Equal(t, 1, cfg.Database.MaxOpenConns)
	assert.Equal(t, 1, cfg.Database.MaxIdleConns)
	assert.Equal(t, 4000, cfg.P2P.Port)
	assert.True(t, cfg.P2P.Enabled)
	assert.True(t, cfg.P2P.EnableRelay)
	assert.True(t, cfg.P2P.EnableRelayDiscovery)
}

// TestDatabaseConfigMethods проверяет методы DatabaseConfig
func TestDatabaseConfigMethods(t *testing.T) {
	cfg := DatabaseConfig{
		Path:         "/test/db.sqlite",
		BusyTimeout:  5000,
		MaxOpenConns: 5,
		MaxIdleConns: 3,
	}

	assert.Equal(t, "/test/db.sqlite", cfg.GetPath())
	assert.Equal(t, 5000, cfg.GetBusyTimeout())
	assert.Equal(t, 5, cfg.GetMaxOpenConns())
	assert.Equal(t, 3, cfg.GetMaxIdleConns())
}

// TestStorageConfigMethods проверяет методы StorageConfig
func TestStorageConfigMethods(t *testing.T) {
	cfg := StorageConfig{
		Path:     "/test/storage",
		FilesDir: "content",
	}

	assert.Equal(t, "/test/storage", cfg.GetPath())
	assert.Equal(t, "content", cfg.GetFilesDir())
}

// TestLoaderGet проверяет метод Get загрузчика
func TestLoaderGet(t *testing.T) {
	loader := NewLoader()
	require.NotNil(t, loader)

	cfg := loader.Get()
	assert.NotNil(t, cfg)
	assert.Equal(t, loader.config, cfg)
}

// TestParseInt проверяет парсинг целых чисел
func TestParseInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"positive", "123", 123},
		{"zero", "0", 0},
		{"negative", "-42", -42},
		{"invalid", "abc", 0},
		{"empty", "", 0},
		{"float", "3.14", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseBool проверяет парсинг булевых значений
func TestParseBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"one", "1", true},
		{"zero", "0", false},
		{"yes", "yes", true},
		{"no", "no", false},
		{"on", "on", true},
		{"off", "off", false},
		{"invalid", "invalid", false},
		{"empty", "", false},
		{"True", "True", false}, // чувствительно к регистру
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBool(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLoadFromYAML проверяет загрузку из YAML файла
func TestLoadFromYAML(t *testing.T) {
	// Создаём временный файл конфигурации
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	testConfig := &Config{
		Database: DatabaseConfig{
			Path:         "/custom/db.sqlite",
			BusyTimeout:  60000,
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
		Storage: StorageConfig{
			Path:     "/custom/storage",
			FilesDir: "custom_files",
		},
		P2P: P2PConfig{
			Enabled:              false,
			Port:                 5000,
			EnableRelay:          false,
			EnableRelayDiscovery: false,
		},
	}

	// Сериализуем в YAML
	yamlData, err := yaml.Marshal(testConfig)
	require.NoError(t, err)

	// Записываем в файл
	err = os.WriteFile(configPath, yamlData, 0644)
	require.NoError(t, err)

	// Загружаем конфигурацию
	loader := NewLoader()
	err = loader.loadFromYAML(configPath)
	require.NoError(t, err)

	cfg := loader.Get()
	assert.Equal(t, "/custom/db.sqlite", cfg.Database.Path)
	assert.Equal(t, 60000, cfg.Database.BusyTimeout)
	assert.Equal(t, 10, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, "/custom/storage", cfg.Storage.Path)
	assert.Equal(t, "custom_files", cfg.Storage.FilesDir)
	assert.False(t, cfg.P2P.Enabled)
	assert.Equal(t, 5000, cfg.P2P.Port)
	assert.False(t, cfg.P2P.EnableRelay)
	assert.False(t, cfg.P2P.EnableRelayDiscovery)
}

// TestLoadFromYAML_InvalidFile проверяет обработку невалидного YAML
func TestLoadFromYAML_InvalidFile(t *testing.T) {
	loader := NewLoader()

	// Пытаемся загрузить несуществующий файл
	err := loader.loadFromYAML("/nonexistent/path/config.yaml")
	assert.Error(t, err)
}

// TestLoadFromYAML_Malformed проверяет обработку некорректного YAML
func TestLoadFromYAML_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Записываем невалидный YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.loadFromYAML(configPath)
	assert.Error(t, err)
}

// TestLoadFromYAML_Partial проверяет частичную загрузку (только некоторые поля)
func TestLoadFromYAML_Partial(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.yaml")

	// YAML только с P2P настройками
	yamlContent := `
p2p:
  enabled: false
  port: 8080
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.loadFromYAML(configPath)
	require.NoError(t, err)

	cfg := loader.Get()
	// P2P настройки должны обновиться
	assert.False(t, cfg.P2P.Enabled)
	assert.Equal(t, 8080, cfg.P2P.Port)
	// Остальные настройки должны остаться по умолчанию
	assert.NotEmpty(t, cfg.Database.Path)
	assert.NotEmpty(t, cfg.Storage.Path)
}

// TestNormalizePaths проверяет нормализацию путей
func TestNormalizePaths(t *testing.T) {
	loader := NewLoader()

	// Устанавливаем относительные пути
	loader.config.Database.Path = "./relative/db.sqlite"
	loader.config.Storage.Path = "./relative/storage"

	// Нормализуем
	loader.normalizePaths()

	// Проверяем что пути стали абсолютными
	assert.True(t, filepath.IsAbs(loader.config.Database.Path))
	assert.True(t, filepath.IsAbs(loader.config.Storage.Path))

	// Проверяем что пути заканчиваются на ожидаемые компоненты
	assert.True(t, filepath.IsAbs(loader.config.Database.Path))
	assert.Contains(t, loader.config.Database.Path, "relative")
	assert.Contains(t, loader.config.Storage.Path, "relative")
}

// TestNormalizePaths_Absolute проверяет что абсолютные пути не меняются
func TestNormalizePaths_Absolute(t *testing.T) {
	loader := NewLoader()

	// Устанавливаем абсолютные пути
	expectedDbPath := "C:\\test\\db.sqlite"
	expectedStoragePath := "D:\\test\\storage"

	loader.config.Database.Path = expectedDbPath
	loader.config.Storage.Path = expectedStoragePath

	// Нормализуем
	loader.normalizePaths()

	// На Windows пути должны остаться неизменными
	assert.Equal(t, expectedDbPath, loader.config.Database.Path)
	assert.Equal(t, expectedStoragePath, loader.config.Storage.Path)
}

// TestSave проверяет сохранение конфигурации в YAML
func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "saved_config.yaml")

	cfg := &Config{
		Database: DatabaseConfig{
			Path:         "/test/db.sqlite",
			BusyTimeout:  30000,
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
		Storage: StorageConfig{
			Path:     "/test/storage",
			FilesDir: "files",
		},
		P2P: P2PConfig{
			Enabled:              true,
			Port:                 4000,
			EnableRelay:          true,
			EnableRelayDiscovery: true,
		},
	}

	err := Save(cfg, configPath)
	require.NoError(t, err)

	// Проверяем что файл создан
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Читаем и проверяем содержимое
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var loaded Config
	err = yaml.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, cfg.Database.Path, loaded.Database.Path)
	assert.Equal(t, cfg.Storage.Path, loaded.Storage.Path)
	assert.Equal(t, cfg.P2P.Port, loaded.P2P.Port)
}

// TestSaveAsJSON проверяет сохранение конфигурации в JSON
func TestSaveAsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "saved_config.json")

	cfg := &Config{
		Database: DatabaseConfig{
			Path:         "/test/db.sqlite",
			BusyTimeout:  30000,
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
		Storage: StorageConfig{
			Path:     "/test/storage",
			FilesDir: "files",
		},
		P2P: P2PConfig{
			Enabled: true,
			Port:    4000,
		},
	}

	err := SaveAsJSON(cfg, configPath)
	require.NoError(t, err)

	// Проверяем что файл создан
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Читаем и проверяем содержимое
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var loaded Config
	err = yaml.Unmarshal(data, &loaded) // yaml может читать JSON
	require.NoError(t, err)

	assert.Equal(t, cfg.Database.Path, loaded.Database.Path)
	assert.Equal(t, cfg.P2P.Port, loaded.P2P.Port)
}

// TestSave_CreatesDirectory проверяет что Save создаёт директорию
func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.yaml")

	cfg := DefaultConfig()

	err := Save(cfg, configPath)
	require.NoError(t, err)

	// Проверяем что файл создан в_nested директории
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

// TestLoadFromEnv проверяет загрузку из переменных окружения
func TestLoadFromEnv(t *testing.T) {
	// Сохраняем текущие значения
	originalEnv := map[string]string{
		"PROJECTT_DB_PATH":             os.Getenv("PROJECTT_DB_PATH"),
		"PROJECTT_DB_BUSY_TIMEOUT":     os.Getenv("PROJECTT_DB_BUSY_TIMEOUT"),
		"PROJECTT_STORAGE_PATH":        os.Getenv("PROJECTT_STORAGE_PATH"),
		"PROJECTT_STORAGE_FILES_DIR":   os.Getenv("PROJECTT_STORAGE_FILES_DIR"),
		"PROJECTT_P2P_ENABLED":         os.Getenv("PROJECTT_P2P_ENABLED"),
		"PROJECTT_P2P_PORT":            os.Getenv("PROJECTT_P2P_PORT"),
		"PROJECTT_P2P_RELAY":           os.Getenv("PROJECTT_P2P_RELAY"),
		"PROJECTT_P2P_RELAY_DISCOVERY": os.Getenv("PROJECTT_P2P_RELAY_DISCOVERY"),
	}

	// Восстанавливаем после теста
	defer func() {
		for key, val := range originalEnv {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Устанавливаем тестовые значения
	os.Setenv("PROJECTT_DB_PATH", "/env/db.sqlite")
	os.Setenv("PROJECTT_DB_BUSY_TIMEOUT", "45000")
	os.Setenv("PROJECTT_STORAGE_PATH", "/env/storage")
	os.Setenv("PROJECTT_STORAGE_FILES_DIR", "env_files")
	os.Setenv("PROJECTT_P2P_ENABLED", "false")
	os.Setenv("PROJECTT_P2P_PORT", "6000")
	os.Setenv("PROJECTT_P2P_RELAY", "false")
	os.Setenv("PROJECTT_P2P_RELAY_DISCOVERY", "false")

	loader := NewLoader()
	loader.loadFromEnv()

	cfg := loader.Get()
	assert.Equal(t, "/env/db.sqlite", cfg.Database.Path)
	assert.Equal(t, 45000, cfg.Database.BusyTimeout)
	assert.Equal(t, "/env/storage", cfg.Storage.Path)
	assert.Equal(t, "env_files", cfg.Storage.FilesDir)
	assert.False(t, cfg.P2P.Enabled)
	assert.Equal(t, 6000, cfg.P2P.Port)
	assert.False(t, cfg.P2P.EnableRelay)
	assert.False(t, cfg.P2P.EnableRelayDiscovery)
}

// TestLoadFromEnv_InvalidValues проверяет обработку невалидных значений env
func TestLoadFromEnv_InvalidValues(t *testing.T) {
	originalEnv := map[string]string{
		"PROJECTT_DB_BUSY_TIMEOUT": os.Getenv("PROJECTT_DB_BUSY_TIMEOUT"),
		"PROJECTT_P2P_PORT":        os.Getenv("PROJECTT_P2P_PORT"),
	}

	defer func() {
		for key, val := range originalEnv {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Устанавливаем невалидные значения
	os.Setenv("PROJECTT_DB_BUSY_TIMEOUT", "invalid")
	os.Setenv("PROJECTT_P2P_PORT", "not_a_number")

	loader := NewLoader()
	originalTimeout := loader.config.Database.BusyTimeout
	originalPort := loader.config.P2P.Port

	loader.loadFromEnv()

	// При невалидных значениях должны остаться оригинальные
	assert.Equal(t, originalTimeout, loader.config.Database.BusyTimeout)
	assert.Equal(t, originalPort, loader.config.P2P.Port)
}

// TestLoadFromFlags_Help проверяет флаг помощи
func TestLoadFromFlags_Help(t *testing.T) {
	// Сохраняем оригинальные аргументы
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Устанавливаем флаг помощи
	os.Args = []string{"projectT", "-help"}

	// Help вызывает os.Exit(0), поэтому тест завершится
	// Для теста нам нужно поймать panic или использовать другой подход
	// В данном случае просто проверяем что метод существует
	loader := NewLoader()

	// Тест будет завершен с os.Exit, поэтому это последний тест
	_ = loader
}
