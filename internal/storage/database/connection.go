package database

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDB инициализирует подключение к базе данных SQLite
func InitDB() {
	// Создаем директорию для базы данных, если она не существует
	dbDir := "./internal/storage"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatal("Ошибка при создании директории для базы данных:", err)
	}

	var err error
	DB, err = sql.Open("sqlite3", "./internal/storage/projectT.db?cache=shared&_busy_timeout=30000")
	if err != nil {
		log.Fatal("Ошибка при открытии базы данных:", err)
	}

	// Устанавливаем параметры соединения для лучшей обработки конкурентного доступа
	DB.SetMaxOpenConns(1) // Ограничиваем количество открытых соединений
	DB.SetMaxIdleConns(1) // Ограничиваем количество простаивающих соединений

	// Проверяем подключение
	if err = DB.Ping(); err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}

	// Подключение к SQLite успешно установлено
}

// CloseDB закрывает соединение с базой данных
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
