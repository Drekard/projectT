package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := "./storage/projectT.db"

	// Проверяем существование файла
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("База данных не найдена: %s", dbPath)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()

	fmt.Println("=== ОЧИСТКА БАЗЫ ДАННЫХ ===")
	fmt.Println("Путь:", dbPath)
	fmt.Println()

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Ошибка начала транзакции: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Ошибка отката транзакции: %v", rollbackErr)
			}
			log.Fatalf("Паника: %v", r)
		}
	}()

	// 1. Удаляем все чаты кроме локального (local_chat_0)
	fmt.Println("1. Удаление чатов (кроме локального)...")
	result, err := tx.Exec(`DELETE FROM chats WHERE peer_id != 'local_chat_0'`)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("Ошибка отката транзакции: %v", rollbackErr)
		}
		log.Fatalf("Ошибка удаления чатов: %v", err)
	}
	count, _ := result.RowsAffected()
	fmt.Printf("   Удалено чатов: %d\n", count)

	// 2. Удаляем все контакты кроме локального (local_chat_0)
	fmt.Println("2. Удаление контактов (кроме локального)...")
	result, err = tx.Exec(`DELETE FROM contacts WHERE peer_id != 'local_chat_0'`)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("Ошибка отката транзакции: %v", rollbackErr)
		}
		log.Fatalf("Ошибка удаления контактов: %v", err)
	}
	count, _ = result.RowsAffected()
	fmt.Printf("   Удалено контактов: %d\n", count)

	// 3. Удаляем все профили кроме локального (owner_type = 'local')
	fmt.Println("3. Удаление удалённых профилей...")
	result, err = tx.Exec(`DELETE FROM profiles WHERE owner_type != 'local'`)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("Ошибка отката транзакции: %v", rollbackErr)
		}
		log.Fatalf("Ошибка удаления профилей: %v", err)
	}
	count, _ = result.RowsAffected()
	fmt.Printf("   Удалено профилей: %d\n", count)

	// 4. Удаляем сообщения из удалённых чатов (каскадное удаление должно сработать, но на всякий случай)
	fmt.Println("4. Очистка сообщений от удалённых чатов...")
	result, err = tx.Exec(`DELETE FROM chat_messages WHERE chat_id NOT IN (SELECT id FROM chats)`)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("Ошибка отката транзакции: %v", rollbackErr)
		}
		log.Fatalf("Ошибка удаления сообщений: %v", err)
	}
	count, _ = result.RowsAffected()
	fmt.Printf("   Удалено сообщений: %d\n", count)

	// 5. Сбрасываем счётчики автоинкремента
	fmt.Println("5. Сброс счётчиков автоинкремента...")
	_, err = tx.Exec(`DELETE FROM sqlite_sequence WHERE name IN ('chats', 'contacts', 'profiles', 'chat_messages')`)
	if err != nil {
		log.Printf("Предупреждение: не удалось сбросить счётчики: %v", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("Ошибка отката транзакции: %v", rollbackErr)
		}
		log.Fatalf("Ошибка фиксации транзакции: %v", err)
	}

	fmt.Println()
	fmt.Println("=== ОЧИСТКА ЗАВЕРШЕНА ===")
	fmt.Println("Сохранено:")
	fmt.Println("  - Локальный профиль (owner_type = 'local')")
	fmt.Println("  - Локальный чат (peer_id = 'local_chat_0')")
}
