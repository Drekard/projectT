package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config основная структура конфигурации приложения
type Config struct {
	// Database настройки базы данных
	Database DatabaseConfig `yaml:"database" json:"database"`
	// Storage настройки файлового хранилища
	Storage StorageConfig `yaml:"storage" json:"storage"`
	// P2P настройки P2P сети
	P2P P2PConfig `yaml:"p2p" json:"p2p"`
}

// DatabaseConfig настройки базы данных
type DatabaseConfig struct {
	// Path путь к файлу базы данных SQLite
	Path string `yaml:"path" json:"path"`
	// BusyTimeout таймаут ожидания при блокировке БД (мс)
	BusyTimeout int `yaml:"busy_timeout" json:"busy_timeout"`
	// MaxOpenConns максимальное количество открытых соединений
	MaxOpenConns int `yaml:"max_open_conns" json:"max_open_conns"`
	// MaxIdleConns максимальное количество простаивающих соединений
	MaxIdleConns int `yaml:"max_idle_conns" json:"max_idle_conns"`
}

// GetPath возвращает путь к базе данных
func (c DatabaseConfig) GetPath() string {
	return c.Path
}

// GetBusyTimeout возвращает таймаут ожидания БД
func (c DatabaseConfig) GetBusyTimeout() int {
	return c.BusyTimeout
}

// GetMaxOpenConns возвращает максимальное количество открытых соединений
func (c DatabaseConfig) GetMaxOpenConns() int {
	return c.MaxOpenConns
}

// GetMaxIdleConns возвращает максимальное количество простаивающих соединений
func (c DatabaseConfig) GetMaxIdleConns() int {
	return c.MaxIdleConns
}

// StorageConfig настройки файлового хранилища
type StorageConfig struct {
	// Path корневая директория для хранения файлов
	Path string `yaml:"path" json:"path"`
	// FilesDir поддиректория для файлов контента
	FilesDir string `yaml:"files_dir" json:"files_dir"`
}

// GetPath возвращает путь к хранилищу
func (c StorageConfig) GetPath() string {
	return c.Path
}

// GetFilesDir возвращает директорию для файлов
func (c StorageConfig) GetFilesDir() string {
	return c.FilesDir
}

// P2PConfig настройки P2P сети
type P2PConfig struct {
	// Enabled включён ли P2P режим
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Port порт для P2P соединений
	Port int `yaml:"port" json:"port"`
	// EnableRelay использовать ли relay для обхода NAT
	EnableRelay bool `yaml:"enable_relay" json:"enable_relay"`
	// EnableRelayDiscovery использовать ли автообнаружение relay
	EnableRelayDiscovery bool `yaml:"enable_relay_discovery" json:"enable_relay_discovery"`
}

// DefaultConfig возвращает конфигурацию со значениями по умолчанию
func DefaultConfig() *Config {
	// Пути по умолчанию относительно текущей рабочей директории
	// При запуске go run это будет директория проекта
	// При запуске exe это будет директория, откуда запустили
	// Для standalone exe лучше использовать config.yaml с абсолютными путями
	cwd, _ := os.Getwd()
	if cwd == "" {
		cwd = "."
	}

	return &Config{
		Database: DatabaseConfig{
			Path:         filepath.Join(cwd, "storage", "projectT.db"),
			BusyTimeout:  30000,
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
		Storage: StorageConfig{
			Path:     filepath.Join(cwd, "storage"),
			FilesDir: "files",
		},
		P2P: P2PConfig{
			Enabled:              true,
			Port:                 4000,
			EnableRelay:          true,
			EnableRelayDiscovery: true,
		},
	}
}

// Loader загрузчик конфигурации из различных источников
type Loader struct {
	config *Config
}

// NewLoader создаёт новый загрузчик конфигурации
func NewLoader() *Loader {
	return &Loader{
		config: DefaultConfig(),
	}
}

// Load загружает конфигурацию из всех источников в порядке приоритета:
// 1. Flags (флаги командной строки) - высший приоритет
// 2. Env variables (переменные окружения)
// 3. Config file (файл config.yaml)
// 4. Default values (значения по умолчанию) - низший приоритет
func (l *Loader) Load() (*Config, error) {
	// Парсим все флаги сразу, но используем их в правильном порядке
	flags := l.parseAllFlags()

	// 1. Сначала загружаем из файла конфигурации (если указан)
	if flags.configFile != "" {
		if err := l.loadFromYAML(flags.configFile); err != nil {
			return nil, fmt.Errorf("ошибка загрузки файла конфигурации %s: %w", flags.configFile, err)
		}
	}

	// 2. Переопределяем из переменных окружения
	l.loadFromEnv()

	// 3. Переопределяем из флагов командной строки
	l.applyFlags(flags)

	// 4. Нормализуем пути (делаем абсолютными если относительные)
	l.normalizePaths()

	return l.config, nil
}

// parsedFlags хранит распарсенные флаги
type parsedFlags struct {
	configFile      string
	dbPath          string
	dbBusyTimeout   int
	storagePath     string
	storageFilesDir string
	p2pEnabled      bool
	p2pPort         int
	p2pRelay        bool
	p2pRelayDisc    bool
}

// parseAllFlags парсит все флаги командной строки
func (l *Loader) parseAllFlags() *parsedFlags {
	flags := &parsedFlags{}
	flagSet := flag.NewFlagSet("projectT", flag.ContinueOnError)

	flagSet.StringVar(&flags.configFile, "config", "", "Путь к файлу конфигурации (YAML)")
	flagSet.StringVar(&flags.dbPath, "db-path", "", "Путь к файлу базы данных SQLite")
	flagSet.IntVar(&flags.dbBusyTimeout, "db-timeout", 0, "Таймаут ожидания БД (мс)")
	flagSet.StringVar(&flags.storagePath, "storage-path", "", "Путь к корневой директории хранилища")
	flagSet.StringVar(&flags.storageFilesDir, "storage-files-dir", "", "Поддиректория для файлов")
	flagSet.BoolVar(&flags.p2pEnabled, "p2p-enabled", false, "Включить P2P режим")
	flagSet.IntVar(&flags.p2pPort, "p2p-port", 0, "Порт для P2P соединений")
	flagSet.BoolVar(&flags.p2pRelay, "p2p-relay", false, "Использовать relay для обхода NAT")
	flagSet.BoolVar(&flags.p2pRelayDisc, "p2p-relay-discovery", false, "Автообнаружение relay")

	// Игнорируем ошибку парсинга - флаги могут быть не переданы
	_ = flagSet.Parse(os.Args[1:])

	return flags
}

// applyFlags применяет распарсенные флаги к конфигурации
func (l *Loader) applyFlags(flags *parsedFlags) {
	if flags.dbPath != "" {
		l.config.Database.Path = filepath.ToSlash(flags.dbPath)
	}
	if flags.dbBusyTimeout > 0 {
		l.config.Database.BusyTimeout = flags.dbBusyTimeout
	}
	if flags.storagePath != "" {
		l.config.Storage.Path = filepath.ToSlash(flags.storagePath)
	}
	if flags.storageFilesDir != "" {
		l.config.Storage.FilesDir = flags.storageFilesDir
	}
	if flags.p2pEnabled {
		l.config.P2P.Enabled = flags.p2pEnabled
	}
	if flags.p2pPort > 0 {
		l.config.P2P.Port = flags.p2pPort
	}
	if flags.p2pRelay {
		l.config.P2P.EnableRelay = flags.p2pRelay
	}
	if flags.p2pRelayDisc {
		l.config.P2P.EnableRelayDiscovery = flags.p2pRelayDisc
	}
}

// loadFromYAML загружает конфигурацию из указанного YAML файла
func (l *Loader) loadFromYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Сначала загружаем в map для частичного обновления
	var yamlConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return err
	}

	// Маршалим обратно в структуру Config
	yamlData, err := yaml.Marshal(yamlConfig)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(yamlData, l.config); err != nil {
		return err
	}

	// Нормализуем пути из YAML в Unix-стиль
	l.config.Database.Path = filepath.ToSlash(l.config.Database.Path)
	l.config.Storage.Path = filepath.ToSlash(l.config.Storage.Path)

	return nil
}

// loadFromEnv загружает конфигурацию из переменных окружения
func (l *Loader) loadFromEnv() {
	// Database
	if val := os.Getenv("PROJECTT_DB_PATH"); val != "" {
		l.config.Database.Path = filepath.ToSlash(val)
	}
	if val := os.Getenv("PROJECTT_DB_BUSY_TIMEOUT"); val != "" {
		if timeout := parseInt(val); timeout > 0 {
			l.config.Database.BusyTimeout = timeout
		}
	}

	// Storage
	if val := os.Getenv("PROJECTT_STORAGE_PATH"); val != "" {
		l.config.Storage.Path = filepath.ToSlash(val)
	}
	if val := os.Getenv("PROJECTT_STORAGE_FILES_DIR"); val != "" {
		l.config.Storage.FilesDir = val
	}

	// P2P
	if val := os.Getenv("PROJECTT_P2P_ENABLED"); val != "" {
		l.config.P2P.Enabled = parseBool(val)
	}
	if val := os.Getenv("PROJECTT_P2P_PORT"); val != "" {
		if port := parseInt(val); port > 0 {
			l.config.P2P.Port = port
		}
	}
	if val := os.Getenv("PROJECTT_P2P_RELAY"); val != "" {
		l.config.P2P.EnableRelay = parseBool(val)
	}
	if val := os.Getenv("PROJECTT_P2P_RELAY_DISCOVERY"); val != "" {
		l.config.P2P.EnableRelayDiscovery = parseBool(val)
	}
}

// normalizePaths нормализует пути (конвертирует относительные в абсолютные)
// Относительные пути разрешаются относительно текущей рабочей директории
// Это обеспечивает корректную работу как при go run, так и при запуске exe
// Пути всегда приводятся к Unix-стилю (с forward slashes) для кроссплатформенности
func (l *Loader) normalizePaths() {
	// Нормализуем путь к базе данных
	if !filepath.IsAbs(l.config.Database.Path) {
		if abs, err := filepath.Abs(l.config.Database.Path); err == nil {
			l.config.Database.Path = filepath.ToSlash(abs)
		}
	}

	// Нормализуем путь к хранилищу
	if !filepath.IsAbs(l.config.Storage.Path) {
		if abs, err := filepath.Abs(l.config.Storage.Path); err == nil {
			l.config.Storage.Path = filepath.ToSlash(abs)
		}
	}
}

// parseInt парсит строку в int
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result) //nolint:errcheck
	return result
}

// parseBool парсит строку в bool
func parseBool(s string) bool {
	switch s {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return false
	}
}

// Save сохраняет конфигурацию в YAML файл
func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	// Создаём директорию если не существует
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SaveAsJSON сохраняет конфигурацию в JSON файл (для отладки)
func SaveAsJSON(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Get возвращает текущую конфигурацию
func (l *Loader) Get() *Config {
	return l.config
}
