//go:build ignore
// +build ignore

// Скрипт для очистки таблиц чатов и сообщений
// Использование: go run clear_chats.go
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
	fmt.Println("=== Очистка чатов и сообщений ===")
	fmt.Println()

	// Определяем путь к базе данных
	dbPath := filepath.Join("storage", "projectT.db")

	// Проверяем существование файла БД
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Printf("❌ Файл базы данных не найден: %s\n", dbPath)
		fmt.Println("Убедитесь, что скрипт запущен из корневой директории проекта.")
		os.Exit(1)
	}

	fmt.Printf("📁 Путь к базе данных: %s\n", dbPath)
	fmt.Println()

	// Открываем базу данных
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("❌ Ошибка открытия базы данных: %v", err)
	}
	defer db.Close()

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Ошибка подключения к базе данных: %v", err)
	}

	fmt.Println("✅ Подключение к базе данных успешно")
	fmt.Println()

	// Получаем статистику до очистки
	var chatCount, messageCount int

	err = db.QueryRow("SELECT COUNT(*) FROM chats").Scan(&chatCount)
	if err != nil {
		log.Printf("⚠️  Ошибка подсчёта чатов: %v", err)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM chat_messages").Scan(&messageCount)
	if err != nil {
		log.Printf("⚠️  Ошибка подсчёта сообщений: %v", err)
	}

	fmt.Println("📊 Статистика до очистки:")
	fmt.Printf("   Чатов: %d\n", chatCount)
	fmt.Printf("   Сообщений: %d\n", messageCount)
	fmt.Println()

	// Флаг для автоматического подтверждения (можно добавить через аргумент)
	autoConfirm := len(os.Args) > 1 && (os.Args[1] == "-y" || os.Args[1] == "--yes")

	// Подтверждение
	if !autoConfirm {
		fmt.Print("⚠️  Вы уверены, что хотите удалить ВСЕ чаты и сообщения? (y/n): ")
		var confirm string
		fmt.Scanln(&confirm)

		if confirm != "y" && confirm != "Y" && confirm != "yes" && confirm != "YES" {
			fmt.Println("❌ Очистка отменена")
			os.Exit(0)
		}
	}

	fmt.Println()
	fmt.Println("🗑️  Начало очистки...")
	fmt.Println()

	// Включаем внешние ключи
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Printf("⚠️  Ошибка включения внешних ключей: %v", err)
	}

	// Очищаем таблицу сообщений
	result, err := db.Exec("DELETE FROM chat_messages")
	if err != nil {
		log.Fatalf("❌ Ошибка очистки таблицы chat_messages: %v", err)
	}
	deletedMessages, _ := result.RowsAffected()
	fmt.Printf("✅ Удалено сообщений: %d\n", deletedMessages)

	// Очищаем таблицу чатов
	result, err = db.Exec("DELETE FROM chats")
	if err != nil {
		log.Fatalf("❌ Ошибка очистки таблицы chats: %v", err)
	}
	deletedChats, _ := result.RowsAffected()
	fmt.Printf("✅ Удалено чатов: %d\n", deletedChats)

	// Сбрасываем автоинкремент
	_, err = db.Exec("DELETE FROM sqlite_sequence WHERE name='chats'")
	if err != nil {
		log.Printf("⚠️  Ошибка сброса автоинкремента chats: %v", err)
	}

	_, err = db.Exec("DELETE FROM sqlite_sequence WHERE name='chat_messages'")
	if err != nil {
		log.Printf("⚠️  Ошибка сброса автоинкремента chat_messages: %v", err)
	}

	fmt.Println()
	fmt.Println("📊 Статистика после очистки:")

	err = db.QueryRow("SELECT COUNT(*) FROM chats").Scan(&chatCount)
	if err != nil {
		log.Printf("⚠️  Ошибка подсчёта чатов: %v", err)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM chat_messages").Scan(&messageCount)
	if err != nil {
		log.Printf("⚠️  Ошибка подсчёта сообщений: %v", err)
	}

	fmt.Printf("   Чатов: %d\n", chatCount)
	fmt.Printf("   Сообщений: %d\n", messageCount)
	fmt.Println()

	// VACUUM для оптимизации БД
	fmt.Println("🔧 Оптимизация базы данных (VACUUM)...")
	_, err = db.Exec("VACUUM")
	if err != nil {
		log.Printf("⚠️  Ошибка оптимизации: %v", err)
	}

	fmt.Println()
	fmt.Println("✅ Очистка завершена успешно!")
	fmt.Println()
	fmt.Println("📝 Примечание:")
	fmt.Println("   - Локальный чат (ID=1) также был удалён")
	fmt.Println("   - Профили пиров не были затронуты")
	fmt.Println("   - Контакты не были затронуты")
}
