package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// normalizePathForOS нормализует путь для текущей ОС
// На Windows добавляет букву диска и конвертирует в forward slashes
// На Unix-системах оставляет как есть
func normalizePathForOS(path string) string {
	if runtime.GOOS == "windows" {
		// Конвертируем в абсолютный путь с буквой диска
		abs := filepath.Join("c:", path)
		return filepath.ToSlash(abs)
	}
	return path
}

// TestLoader_Load_FullIntegration проверяет полную загрузку конфигурации
// с приоритетом: флаги > env > file > default
func TestLoader_Load_FullIntegration(t *testing.T) {
	// Сохраняем оригинальные аргументы и env
	originalArgs := os.Args
	originalEnv := map[string]string{
		"PROJECTT_DB_PATH":           os.Getenv("PROJECTT_DB_PATH"),
		"PROJECTT_STORAGE_PATH":      os.Getenv("PROJECTT_STORAGE_PATH"),
		"PROJECTT_P2P_PORT":          os.Getenv("PROJECTT_P2P_PORT"),
		"PROJECTT_P2P_ENABLED":       os.Getenv("PROJECTT_P2P_ENABLED"),
		"PROJECTT_DB_BUSY_TIMEOUT":   os.Getenv("PROJECTT_DB_BUSY_TIMEOUT"),
		"PROJECTT_STORAGE_FILES_DIR": os.Getenv("PROJECTT_STORAGE_FILES_DIR"),
	}

	// Восстанавливаем после теста
	defer func() {
		os.Args = originalArgs
		for key, val := range originalEnv {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Очищаем env переменные для чистоты теста
	for key := range originalEnv {
		os.Unsetenv(key)
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	// Создаём конфиг файл с базовыми значениями
	fileConfig := &Config{
		Database: DatabaseConfig{
			Path:        normalizePathForOS("/file/db.sqlite"),
			BusyTimeout: 30000,
		},
		Storage: StorageConfig{
			Path:     normalizePathForOS("/file/storage"),
			FilesDir: "file_files",
		},
		P2P: P2PConfig{
			Enabled: true,
			Port:    4000,
		},
	}

	yamlData, err := yaml.Marshal(fileConfig)
	require.NoError(t, err)
	err = os.WriteFile(configPath, yamlData, 0644)
	require.NoError(t, err)

	// Тест 1: Загрузка только из файла
	os.Args = []string{"projectT", "-config=" + configPath}
	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	assert.Equal(t, normalizePathForOS("/file/db.sqlite"), cfg.Database.Path)
	assert.Equal(t, normalizePathForOS("/file/storage"), cfg.Storage.Path)
	assert.Equal(t, 4000, cfg.P2P.Port)
}

// TestLoader_Load_EnvOverridesFile проверяет что env переопределяет файл
func TestLoader_Load_EnvOverridesFile(t *testing.T) {
	originalArgs := os.Args
	originalEnv := map[string]string{
		"PROJECTT_DB_PATH":      os.Getenv("PROJECTT_DB_PATH"),
		"PROJECTT_STORAGE_PATH": os.Getenv("PROJECTT_STORAGE_PATH"),
		"PROJECTT_P2P_PORT":     os.Getenv("PROJECTT_P2P_PORT"),
	}

	defer func() {
		os.Args = originalArgs
		for key, val := range originalEnv {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	// Создаём конфиг файл
	fileConfig := &Config{
		Database: DatabaseConfig{
			Path: normalizePathForOS("/file/db.sqlite"),
		},
		Storage: StorageConfig{
			Path: normalizePathForOS("/file/storage"),
		},
		P2P: P2PConfig{
			Port: 4000,
		},
	}

	yamlData, err := yaml.Marshal(fileConfig)
	require.NoError(t, err)
	err = os.WriteFile(configPath, yamlData, 0644)
	require.NoError(t, err)

	// Устанавливаем env переменные
	os.Setenv("PROJECTT_DB_PATH", normalizePathForOS("/env/db.sqlite"))
	os.Setenv("PROJECTT_STORAGE_PATH", normalizePathForOS("/env/storage"))
	os.Setenv("PROJECTT_P2P_PORT", "5000")

	// Загружаем конфигурацию
	os.Args = []string{"projectT", "-config=" + configPath}
	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Env должны переопределить файл
	assert.Equal(t, normalizePathForOS("/env/db.sqlite"), cfg.Database.Path)
	assert.Equal(t, normalizePathForOS("/env/storage"), cfg.Storage.Path)
	assert.Equal(t, 5000, cfg.P2P.Port)
}

// TestLoader_Load_FlagsOverrideEnv проверяет что флаги переопределяют env
func TestLoader_Load_FlagsOverrideEnv(t *testing.T) {
	originalArgs := os.Args
	originalEnv := map[string]string{
		"PROJECTT_DB_PATH":      os.Getenv("PROJECTT_DB_PATH"),
		"PROJECTT_STORAGE_PATH": os.Getenv("PROJECTT_STORAGE_PATH"),
		"PROJECTT_P2P_PORT":     os.Getenv("PROJECTT_P2P_PORT"),
	}

	defer func() {
		os.Args = originalArgs
		for key, val := range originalEnv {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Устанавливаем env переменные
	os.Setenv("PROJECTT_DB_PATH", normalizePathForOS("/env/db.sqlite"))
	os.Setenv("PROJECTT_STORAGE_PATH", normalizePathForOS("/env/storage"))
	os.Setenv("PROJECTT_P2P_PORT", "5000")

	// Загружаем конфигурацию с флагами
	os.Args = []string{
		"projectT",
		"--db-path=" + normalizePathForOS("/flag/db.sqlite"),
		"--storage-path=" + normalizePathForOS("/flag/storage"),
		"--p2p-port=6000",
	}

	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Флаги должны переопределить env
	assert.Equal(t, normalizePathForOS("/flag/db.sqlite"), cfg.Database.Path)
	assert.Equal(t, normalizePathForOS("/flag/storage"), cfg.Storage.Path)
	assert.Equal(t, 6000, cfg.P2P.Port)
}

// TestLoader_Load_FullPriorityChain проверяет полную цепочку приоритетов
func TestLoader_Load_FullPriorityChain(t *testing.T) {
	originalArgs := os.Args
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

	defer func() {
		os.Args = originalArgs
		for key, val := range originalEnv {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	// Создаём конфиг файл со всеми значениями
	fileConfig := &Config{
		Database: DatabaseConfig{
			Path:         normalizePathForOS("/file/db.sqlite"),
			BusyTimeout:  30000,
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
		Storage: StorageConfig{
			Path:     normalizePathForOS("/file/storage"),
			FilesDir: "file_files",
		},
		P2P: P2PConfig{
			Enabled:              true,
			Port:                 4000,
			EnableRelay:          true,
			EnableRelayDiscovery: true,
		},
	}

	yamlData, err := yaml.Marshal(fileConfig)
	require.NoError(t, err)
	err = os.WriteFile(configPath, yamlData, 0644)
	require.NoError(t, err)

	// Устанавливаем env переменные (частичное переопределение)
	os.Setenv("PROJECTT_DB_PATH", normalizePathForOS("/env/db.sqlite"))
	os.Setenv("PROJECTT_P2P_PORT", "5000")

	// Загружаем конфигурацию с флагом (точечное переопределение)
	os.Args = []string{
		"projectT",
		"-config=" + configPath,
		"--p2p-port=6000",
	}

	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Проверяем приоритеты:
	// db-path: env > file → /env/db.sqlite
	assert.Equal(t, normalizePathForOS("/env/db.sqlite"), cfg.Database.Path)

	// storage-path: file (не переопределено) → /file/storage
	assert.Equal(t, normalizePathForOS("/file/storage"), cfg.Storage.Path)

	// storage-files-dir: file (не переопределено) → file_files
	assert.Equal(t, "file_files", cfg.Storage.FilesDir)

	// p2p-port: flag > env > file → 6000
	assert.Equal(t, 6000, cfg.P2P.Port)

	// busy-timeout: file (не переопределено) → 30000
	assert.Equal(t, 30000, cfg.Database.BusyTimeout)
}

// TestLoader_Load_NoConfigFile проверяет загрузку без файла конфигурации
func TestLoader_Load_NoConfigFile(t *testing.T) {
	originalArgs := os.Args
	originalEnv := map[string]string{
		"PROJECTT_DB_PATH": os.Getenv("PROJECTT_DB_PATH"),
	}

	defer func() {
		os.Args = originalArgs
		for key, val := range originalEnv {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Очищаем env
	os.Unsetenv("PROJECTT_DB_PATH")

	// Загружаем без файла и флагов
	os.Args = []string{"projectT"}

	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Должны использоваться значения по умолчанию
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Database.Path)
	assert.NotEmpty(t, cfg.Storage.Path)
}

// TestLoader_Load_WithConfigFlag проверяет загрузку с явным указанием файла
func TestLoader_Load_WithConfigFlag(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom_config.yaml")

	customConfig := &Config{
		Database: DatabaseConfig{
			Path:         normalizePathForOS("/custom/path/db.sqlite"),
			BusyTimeout:  99999,
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
		Storage: StorageConfig{
			Path:     normalizePathForOS("/custom/storage"),
			FilesDir: "custom_dir",
		},
		P2P: P2PConfig{
			Enabled:              false,
			Port:                 9999,
			EnableRelay:          false,
			EnableRelayDiscovery: false,
		},
	}

	yamlData, err := yaml.Marshal(customConfig)
	require.NoError(t, err)
	err = os.WriteFile(configPath, yamlData, 0644)
	require.NoError(t, err)

	os.Args = []string{"projectT", "-config=" + configPath}

	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	assert.Equal(t, normalizePathForOS("/custom/path/db.sqlite"), cfg.Database.Path)
	assert.Equal(t, 99999, cfg.Database.BusyTimeout)
	assert.Equal(t, 10, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, normalizePathForOS("/custom/storage"), cfg.Storage.Path)
	assert.Equal(t, "custom_dir", cfg.Storage.FilesDir)
	assert.False(t, cfg.P2P.Enabled)
	assert.Equal(t, 9999, cfg.P2P.Port)
	assert.False(t, cfg.P2P.EnableRelay)
	assert.False(t, cfg.P2P.EnableRelayDiscovery)
}

// TestLoader_Load_NormalizesPaths проверяет что загрузка нормализует пути
func TestLoader_Load_NormalizesPaths(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	// Создаём конфиг с относительными путями
	fileConfig := &Config{
		Database: DatabaseConfig{
			Path: "./relative/db.sqlite",
		},
		Storage: StorageConfig{
			Path: "./relative/storage",
		},
	}

	yamlData, err := yaml.Marshal(fileConfig)
	require.NoError(t, err)
	err = os.WriteFile(configPath, yamlData, 0644)
	require.NoError(t, err)

	os.Args = []string{"projectT", "-config=" + configPath}

	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Пути должны быть нормализованы (стать абсолютными)
	assert.True(t, filepath.IsAbs(cfg.Database.Path))
	assert.True(t, filepath.IsAbs(cfg.Storage.Path))
}

// TestConfig_RoundTrip проверяет полный цикл сохранения и загрузки
func TestConfig_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "roundtrip.yaml")

	original := &Config{
		Database: DatabaseConfig{
			Path:         normalizePathForOS("/test/db.sqlite"),
			BusyTimeout:  45000,
			MaxOpenConns: 5,
			MaxIdleConns: 3,
		},
		Storage: StorageConfig{
			Path:     normalizePathForOS("/test/storage"),
			FilesDir: "my_files",
		},
		P2P: P2PConfig{
			Enabled:              false,
			Port:                 8080,
			EnableRelay:          true,
			EnableRelayDiscovery: false,
		},
	}

	// Сохраняем
	err := Save(original, configPath)
	require.NoError(t, err)

	// Загружаем через Loader
	os.Args = []string{"projectT", "-config=" + configPath}
	loader := NewLoader()
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Сравниваем значения
	assert.Equal(t, original.Database.Path, cfg.Database.Path)
	assert.Equal(t, original.Database.BusyTimeout, cfg.Database.BusyTimeout)
	assert.Equal(t, original.Database.MaxOpenConns, cfg.Database.MaxOpenConns)
	assert.Equal(t, original.Database.MaxIdleConns, cfg.Database.MaxIdleConns)
	assert.Equal(t, original.Storage.Path, cfg.Storage.Path)
	assert.Equal(t, original.Storage.FilesDir, cfg.Storage.FilesDir)
	assert.Equal(t, original.P2P.Enabled, cfg.P2P.Enabled)
	assert.Equal(t, original.P2P.Port, cfg.P2P.Port)
	assert.Equal(t, original.P2P.EnableRelay, cfg.P2P.EnableRelay)
	assert.Equal(t, original.P2P.EnableRelayDiscovery, cfg.P2P.EnableRelayDiscovery)
}
