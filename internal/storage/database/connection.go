package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// DatabaseConfig конфигурация базы данных
type DatabaseConfig interface {
	GetPath() string
	GetBusyTimeout() int
	GetMaxOpenConns() int
	GetMaxIdleConns() int
}

// InitDB инициализирует подключение к базе данных SQLite с настройками по умолчанию
func InitDB() {
	// Для обратной совместимости используем значения по умолчанию
	InitDBWithConfig(defaultDBConfig{})
}

// defaultDBConfig конфигурация по умолчанию для обратной совместимости
type defaultDBConfig struct{}

func (defaultDBConfig) GetPath() string      { return "./storage/projectT.db" }
func (defaultDBConfig) GetBusyTimeout() int  { return 30000 }
func (defaultDBConfig) GetMaxOpenConns() int { return 1 }
func (defaultDBConfig) GetMaxIdleConns() int { return 1 }

// InitDBWithConfig инициализирует подключение к базе данных SQLite с заданной конфигурацией
func InitDBWithConfig(cfg DatabaseConfig) {
	dbPath := cfg.GetPath()
	busyTimeout := cfg.GetBusyTimeout()
	maxOpenConns := cfg.GetMaxOpenConns()
	maxIdleConns := cfg.GetMaxIdleConns()

	// Создаем директорию для базы данных, если она не существует
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatal("Ошибка при создании директории для базы данных:", err)
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?cache=shared&_busy_timeout="+itoa(busyTimeout))
	if err != nil {
		log.Fatal("Ошибка при открытии базы данных:", err)
	}

	// Устанавливаем параметры соединения для лучшей обработки конкурентного доступа
	DB.SetMaxOpenConns(maxOpenConns)
	DB.SetMaxIdleConns(maxIdleConns)

	// Проверяем подключение
	if err = DB.Ping(); err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}

	// Подключение к SQLite успешно установлено
}

// itoa конвертирует int в string
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// Open открывает подключение к указанному файлу базы данных
func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?cache=shared&_busy_timeout=30000")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// CloseDB закрывает соединение с базой данных
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
