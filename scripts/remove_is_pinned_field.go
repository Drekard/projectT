package main

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./internal/storage/projectT.db?cache=shared&_busy_timeout=30000")
	if err != nil {
		log.Fatal("Ошибка при открытии базы данных:", err)
	}
	defer db.Close()

	// Проверяем, существует ли поле is_pinned в таблице items
	query := `SELECT sql FROM sqlite_master WHERE type='table' AND name='items';`
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Ошибка при проверке структуры таблицы items: %v", err)
		return
	}
	defer rows.Close()

	var tableSQL string
	if rows.Next() {
		err = rows.Scan(&tableSQL)
		if err != nil {
			log.Printf("Ошибка при чтении SQL-запроса таблицы items: %v", err)
			return
		}
	}

	// Проверяем, содержит ли таблица поле is_pinned
	if tableSQL != "" && strings.Contains(strings.ToUpper(tableSQL), "IS_PINNED") {
		log.Println("Обнаружено поле is_pinned в таблице items, начинаем процесс удаления...")

		// Создаем новую таблицу без поля is_pinned
		newTableSQL := `
			CREATE TABLE items_new (
				id          INTEGER PRIMARY KEY,
				type        TEXT NOT NULL CHECK (type IN ('text', 'link', 'folder', 'composite', 'image', 'file')),
				title       TEXT,
				description TEXT,
				content_meta TEXT,
				parent_id   INTEGER,
				created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (parent_id) REFERENCES items (id) ON DELETE CASCADE
			);
		`

		// Отключаем внешние ключи на время операции
		_, err = db.Exec(`PRAGMA foreign_keys = off;`)
		if err != nil {
			log.Printf("Ошибка отключения внешних ключей: %v", err)
			return
		}

		// Создаем новую таблицу
		_, err = db.Exec(newTableSQL)
		if err != nil {
			log.Printf("Ошибка создания новой таблицы items: %v", err)
			// Восстанавливаем внешние ключи
			db.Exec(`PRAGMA foreign_keys = on;`)
			return
		}

		// Копируем данные из старой таблицы в новую (без поля is_pinned)
		copyQuery := `
			INSERT INTO items_new (id, type, title, description, content_meta, parent_id, created_at, updated_at)
			SELECT id, type, title, description, content_meta, parent_id, created_at, updated_at
			FROM items;
		`

		log.Println("Начинаем копирование данных...")
		_, err = db.Exec(copyQuery)
		if err != nil {
			log.Printf("Ошибка копирования данных: %v", err)
			// Удаляем новую таблицу, чтобы не оставить мусора
			db.Exec(`DROP TABLE IF EXISTS items_new;`)
			// Восстанавливаем внешние ключи
			db.Exec(`PRAGMA foreign_keys = on;`)
			return
		}
		log.Println("Копирование данных завершено")

		// Удаляем старую таблицу
		log.Println("Удаляем старую таблицу...")
		_, err = db.Exec(`DROP TABLE items;`)
		if err != nil {
			log.Printf("Ошибка удаления старой таблицы: %v", err)
			// Удаляем новую таблицу, чтобы не оставить мусора
			db.Exec(`DROP TABLE IF EXISTS items_new;`)
			// Восстанавливаем внешние ключи
			db.Exec(`PRAGMA foreign_keys = on;`)
			return
		}

		// Переименовываем новую таблицу
		log.Println("Переименовываем таблицу...")
		_, err = db.Exec(`ALTER TABLE items_new RENAME TO items;`)
		if err != nil {
			log.Printf("Ошибка переименования таблицы: %v", err)
			// Восстанавливаем внешние ключи
			db.Exec(`PRAGMA foreign_keys = on;`)
			return
		}

		// Восстанавливаем внешние ключи
		_, err = db.Exec(`PRAGMA foreign_keys = on;`)
		if err != nil {
			log.Printf("Ошибка включения внешних ключей: %v", err)
			return
		}

		log.Println("Успешно удалено поле is_pinned из таблицы items")
	} else {
		log.Println("Поле is_pinned не найдено в таблице items, пропускаем удаление")
	}
}