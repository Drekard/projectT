// Скрипт для миграции таблицы contacts
// Удаляет дублирующиеся поля и создаёт FK на profiles
// Использование: go run scripts/migrate_contacts_table.go

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
	// Определяем путь к БД
	dbPath := filepath.Join("storage", "projectT.db")

	// Проверяем, существует ли файл БД
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Printf("База данных не найдена по пути: %s", dbPath)
		log.Println("Пробуем альтернативный путь: ./projectT.db")
		dbPath = "projectT.db"
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			log.Fatalf("База данных не найдена")
		}
	}

	fmt.Printf("Открываем базу данных: %s\n", dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Ошибка открытия БД: %v", err)
	}
	defer db.Close()

	// Проверяем существование таблицы contacts
	var exists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='contacts'").Scan(&exists)
	if err != nil || exists == 0 {
		log.Println("Таблица contacts не существует. Миграция не требуется.")
		return
	}

	// Проверяем, есть ли старые колонки в contacts
	var hasOldColumns bool
	rows, err := db.Query("PRAGMA table_info(contacts)")
	if err != nil {
		log.Fatalf("Ошибка получения информации о таблице: %v", err)
	}
	defer rows.Close()

	oldCols := []string{"username", "public_key", "status", "avatar_path"}
	hasOldColumns = false
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ, dflt_value string
		err := rows.Scan(&cid, &name, &typ, &notnull, &dflt_value, &pk)
		if err != nil {
			continue
		}
		for _, col := range oldCols {
			if name == col {
				hasOldColumns = true
				fmt.Printf("   Найдена старая колонка: %s\n", name)
			}
		}
	}

	if !hasOldColumns {
		fmt.Println("✅ Таблица contacts уже имеет новую структуру. Миграция не требуется.")
		return
	}

	fmt.Println("\nНачинаем миграцию таблицы contacts...")

	// 0. Проверяем существование таблицы profiles
	var profilesExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='profiles'").Scan(&profilesExists)
	if err != nil || profilesExists == 0 {
		log.Fatalf("Таблица profiles не существует. Сначала создайте профили.")
	}

	// 1. Создаём профили для всех контактов у которых их ещё нет
	fmt.Println("1. Создаём профили для контактов...")
	result, err := db.Exec(`
		INSERT OR IGNORE INTO profiles (owner_type, peer_id, username, title, avatar_path, created_at, updated_at)
		SELECT 
			'remote' as owner_type,
			peer_id,
			COALESCE(username, substr(peer_id, 1, 8)) as username,
			COALESCE(status, '') as title,
			COALESCE(avatar_path, '') as avatar_path,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		FROM contacts
		WHERE peer_id NOT IN (SELECT peer_id FROM profiles WHERE peer_id = contacts.peer_id)
	`)
	if err != nil {
		log.Printf("Предупреждение при создании профилей: %v", err)
	} else {
		count, _ := result.RowsAffected()
		fmt.Printf("   Создано профилей: %d\n", count)
	}

	// 2. Создаём новую таблицу contacts_new с правильной схемой
	fmt.Println("2. Создаём временную таблицу contacts_new...")
	_, err = db.Exec(`
		CREATE TABLE contacts_new (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			peer_id     TEXT UNIQUE NOT NULL REFERENCES profiles(peer_id),
			multiaddr   TEXT,
			notes       TEXT,
			is_blocked  BOOLEAN DEFAULT 0,
			is_favorite BOOLEAN DEFAULT 1,
			last_seen   DATETIME,
			added_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Ошибка создания contacts_new: %v", err)
	}

	// 3. Копируем данные из contacts в contacts_new
	fmt.Println("3. Копируем данные из contacts в contacts_new...")
	_, err = db.Exec(`
		INSERT INTO contacts_new (id, peer_id, multiaddr, notes, is_blocked, last_seen, added_at, updated_at)
		SELECT 
			id,
			peer_id,
			multiaddr,
			notes,
			is_blocked,
			last_seen,
			added_at,
			updated_at
		FROM contacts
	`)
	if err != nil {
		log.Fatalf("Ошибка копирования данных: %v", err)
	}

	// 4. Удаляем старую таблицу
	fmt.Println("4. Удаляем старую таблицу contacts...")
	_, err = db.Exec("DROP TABLE IF EXISTS contacts")
	if err != nil {
		log.Fatalf("Ошибка удаления старой таблицы: %v", err)
	}

	// 5. Переименовываем contacts_new в contacts
	fmt.Println("5. Переименовываем contacts_new в contacts...")
	_, err = db.Exec("ALTER TABLE contacts_new RENAME TO contacts")
	if err != nil {
		log.Fatalf("Ошибка переименования: %v", err)
	}

	// 6. Создаём индексы
	fmt.Println("6. Создаём индексы...")
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id)")
	if err != nil {
		log.Printf("Предупреждение при создании индекса: %v", err)
	}

	// 7. Проверяем результат
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM contacts").Scan(&count)
	if err != nil {
		log.Fatalf("Ошибка проверки результата: %v", err)
	}

	fmt.Printf("\n✅ Миграция завершена успешно!\n")
	fmt.Printf("   Количество контактов: %d\n", count)

	// Проверяем схему
	fmt.Println("\nНовая схема таблицы contacts:")
	rows, err = db.Query("PRAGMA table_info(contacts)")
	if err != nil {
		log.Fatalf("Ошибка получения информации о таблице: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var dflt_value sql.NullString
		err := rows.Scan(&cid, &name, &typ, &notnull, &dflt_value, &pk)
		if err != nil {
			log.Printf("Ошибка чтения строки: %v", err)
			continue
		}
		pkMark := ""
		if pk > 0 {
			pkMark = " [PK]"
		}
		dflt := ""
		if dflt_value.Valid {
			dflt = " DEFAULT " + dflt_value.String
		}
		fmt.Printf("   %-15s %s%s%s\n", name, typ, pkMark, dflt)
	}

	fmt.Println("\n✅ Миграция завершена!")
	fmt.Println("\n⚠️  ВНИМАНИЕ: Старые поля (username, public_key, status, avatar_path) удалены из contacts.")
	fmt.Println("   Эти данные теперь хранятся в таблице profiles.")
}
