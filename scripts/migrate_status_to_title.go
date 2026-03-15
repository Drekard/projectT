//go:build ignore

// Скрипт для миграции поля status в title в таблице profiles
// Использование: go run scripts/migrate_status_to_title.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Определяем путь к базе данных
	dbPath := filepath.Join("internal", "storage", "database", "..", "..", "..", "storage", "projectT.db")

	// Проверяем существование файла БД
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Пробуем альтернативный путь
		dbPath = filepath.Join("storage", "projectT.db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			log.Fatalf("База данных не найдена по путям: %s или %s",
				filepath.Join("internal", "storage", "database", "..", "..", "..", "storage", "projectT.db"),
				filepath.Join("storage", "projectT.db"))
		}
	}

	fmt.Printf("Используемая база данных: %s\n", dbPath)

	// Открываем подключение
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Ошибка открытия БД: %v", err)
	}
	defer db.Close()

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	fmt.Println("Подключение к базе данных успешно")

	// Проверяем существование таблицы profiles
	var tableExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master 
		WHERE type='table' AND name='profiles'
	`).Scan(&tableExists)
	if err != nil {
		log.Fatalf("Ошибка проверки таблицы profiles: %v", err)
	}

	if tableExists == 0 {
		fmt.Println("Таблица profiles не найдена. Миграция не требуется.")
		return
	}

	fmt.Println("Таблица profiles найдена")

	// Проверяем существование колонки status
	var statusColumnExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('profiles') 
		WHERE name='status'
	`).Scan(&statusColumnExists)
	if err != nil {
		log.Fatalf("Ошибка проверки колонки status: %v", err)
	}

	// Проверяем существование колонки title
	var titleColumnExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('profiles') 
		WHERE name='title'
	`).Scan(&titleColumnExists)
	if err != nil {
		log.Fatalf("Ошибка проверки колонки title: %v", err)
	}

	fmt.Printf("Колонка status существует: %v\n", statusColumnExists > 0)
	fmt.Printf("Колонка title существует: %v\n", titleColumnExists > 0)

	// Если колонка title уже существует, а status нет - миграция уже выполнена
	if titleColumnExists > 0 && statusColumnExists == 0 {
		fmt.Println("Миграция уже выполнена (колонка title существует, status отсутствует)")
		return
	}

	// Если обе колонки существуют - переименовываем
	if statusColumnExists > 0 && titleColumnExists > 0 {
		fmt.Println("Обе колонки существуют. Копируем данные из status в title...")

		// Копируем данные
		result, err := db.Exec(`
			UPDATE profiles 
			SET title = status 
			WHERE title IS NULL OR title = ''
		`)
		if err != nil {
			log.Fatalf("Ошибка копирования данных: %v", err)
		}

		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("Обновлено строк: %d\n", rowsAffected)

		// Удаляем колонку status (в SQLite это сложно, поэтому просто оставляем)
		fmt.Println("Колонка status оставлена для обратной совместимости")
		fmt.Println("Миграция завершена")
		return
	}

	// Если только status существует - создаём title и копируем данные
	if statusColumnExists > 0 && titleColumnExists == 0 {
		fmt.Println("Копируем данные из status в новую колонку title...")

		// Добавляем колонку title
		_, err := db.Exec(`
			ALTER TABLE profiles ADD COLUMN title TEXT
		`)
		if err != nil {
			log.Fatalf("Ошибка добавления колонки title: %v", err)
		}
		fmt.Println("Колонка title добавлена")

		// Копируем данные
		result, err := db.Exec(`
			UPDATE profiles 
			SET title = status 
			WHERE status IS NOT NULL
		`)
		if err != nil {
			log.Fatalf("Ошибка копирования данных: %v", err)
		}

		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("Скопировано строк: %d\n", rowsAffected)

		fmt.Println("Миграция завершена успешно")
		return
	}

	// Если ни одной колонки нет - ошибка
	fmt.Println("Ошибка: ни колонка status, ни колонка title не найдены")
	fmt.Println("Возможно, схема базы данных не соответствует ожидаемой")
}
