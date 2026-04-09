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
	dbPath := "./storage/projectT.db"

	// Проверяем существование файла
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Пробуем альтернативный путь
		absPath, _ := filepath.Abs(dbPath)
		log.Printf("Файл %s не найден, пробуем %s", dbPath, absPath)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			log.Fatalf("База данных не найдена ни по одному из путей")
		}
		dbPath = absPath
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer func() { _ = db.Close() }()

	fmt.Println("=== БАЗА ДАННЫХ:", dbPath, "===")

	// === PROFILES ===
	fmt.Println("=== TABLE: profiles ===")
	rows, err := db.Query(`SELECT id, owner_type, peer_id, username, title, avatar_path, background_path, content_char, pinned_uuids, cached_at FROM profiles`)
	if err != nil {
		log.Printf("Ошибка чтения profiles: %v", err)
	} else {
		printRows(rows, []string{"id", "owner_type", "peer_id", "username", "title", "avatar_path", "background_path", "content_char", "pinned_uuids", "cached_at"})
	}

	// === CONTACTS ===
	fmt.Println("\n=== TABLE: contacts ===")
	rows, err = db.Query(`SELECT id, peer_id, multiaddr, notes, is_blocked, is_favorite, last_seen, added_at, updated_at FROM contacts ORDER BY id`)
	if err != nil {
		log.Printf("Ошибка чтения contacts: %v", err)
	} else {
		printRows(rows, []string{"id", "peer_id", "multiaddr", "notes", "is_blocked", "is_favorite", "last_seen", "added_at", "updated_at"})
	}

	// === CHATS ===
	fmt.Println("\n=== TABLE: chats ===")
	rows, err = db.Query(`SELECT id, contact_id, peer_id, is_temporary, last_message_at, created_at, updated_at FROM chats ORDER BY id`)
	if err != nil {
		log.Printf("Ошибка чтения chats: %v", err)
	} else {
		printRows(rows, []string{"id", "contact_id", "peer_id", "is_temporary", "last_message_at", "created_at", "updated_at"})
	}

	// === CHAT_MESSAGES ===
	fmt.Println("\n=== TABLE: chat_messages ===")
	rows, err = db.Query(`SELECT id, chat_id, from_peer_id, content, content_type, sent_at, is_read FROM chat_messages ORDER BY sent_at DESC LIMIT 10`)
	if err != nil {
		log.Printf("Ошибка чтения chat_messages: %v", err)
	} else {
		printRows(rows, []string{"id", "chat_id", "from_peer_id", "content", "content_type", "sent_at", "is_read"})
	}
}

func printRows(rows *sql.Rows, columns []string) {
	defer func() { _ = rows.Close() }()

	// Печатаем заголовок
	fmt.Print("|")
	for _, colName := range columns {
		fmt.Printf(" %15s |", colName)
	}
	fmt.Println()
	fmt.Print("|")
	for range columns {
		fmt.Printf(" %15s |", "---------------")
	}
	fmt.Println()

	// Печатаем строки
	count := 0
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("Ошибка сканирования: %v", err)
			continue
		}

		fmt.Print("|")
		for _, val := range values {
			str := fmt.Sprintf("%v", val)
			if val == nil {
				str = "NULL"
			}
			if len(str) > 15 {
				str = str[:12] + "..."
			}
			fmt.Printf(" %15s |", str)
		}
		fmt.Println()
		count++
	}

	if count == 0 {
		fmt.Println("(пусто)")
	} else {
		fmt.Printf("\nВсего строк: %d\n", count)
	}
}
