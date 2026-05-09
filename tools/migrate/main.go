// Скрипт миграции существующей БД: parent_id → parent_uuid
//
// Использование:
//
//	go run tools/migrate_parent_uuid.go [path/to/database.db]
//
// Если путь к БД не указан, используется storage/projectt.db
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Определяем путь к корню проекта
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filename))
	if err := os.Chdir(projectRoot); err != nil {
		log.Printf("Предупреждение: не удалось перейти в %s: %v", projectRoot, err)
	}

	dbPath := "storage\\projectT.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	// Проверяем существование файла
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("Файл БД не найден: %s", dbPath)
	}

	log.Printf("Открытие БД: %s", dbPath)

	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000")
	if err != nil {
		log.Fatalf("Ошибка открытия БД: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// 1. Добавляем колонку parent_uuid если её нет
	log.Println("Шаг 1: Добавление колонки parent_uuid...")
	_, err = db.Exec(`ALTER TABLE items ADD COLUMN parent_uuid TEXT`)
	if err != nil {
		log.Printf("  Колонка parent_uuid уже существует: %v", err)
	} else {
		log.Println("  Колонка parent_uuid добавлена")
	}

	// 2. Заполняем parent_uuid из element_uuid родителя
	log.Println("Шаг 2: Заполнение parent_uuid из element_uuid родителя...")
	result, err := db.Exec(`
		UPDATE items
		SET parent_uuid = (
			SELECT p.element_uuid
			FROM items p
			WHERE p.id = items.parent_id
		)
		WHERE parent_id IS NOT NULL
		  AND parent_id != 0
		  AND parent_uuid IS NULL
	`)
	if err != nil {
		log.Fatalf("Ошибка миграции данных: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("  Обновлено %d записей", rowsAffected)

	// 3. Создаём индекс
	log.Println("Шаг 3: Создание индекса idx_items_parent_uuid...")
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_items_parent_uuid ON items(parent_uuid)`)
	if err != nil {
		log.Printf("  Ошибка создания индекса (возможно уже существует): %v", err)
	} else {
		log.Println("  Индекс создан")
	}

	// 4. Заполняем item_element_uuid в pinned_items
	log.Println("Шаг 4: Заполнение item_element_uuid в pinned_items...")
	result, err = db.Exec(`
		UPDATE pinned_items
		SET item_element_uuid = (
			SELECT i.element_uuid
			FROM items i
			WHERE i.id = pinned_items.item_id
		)
		WHERE item_element_uuid IS NULL OR item_element_uuid = ''
	`)
	if err != nil {
		log.Printf("  Ошибка обновления pinned_items: %v", err)
	} else {
		rowsAffected, _ = result.RowsAffected()
		log.Printf("  Обновлено %d записей в pinned_items", rowsAffected)
	}

	// 5. Статистика
	log.Println("Шаг 5: Статистика...")
	var totalItems int
	_ = db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&totalItems)
	log.Printf("  Всего элементов: %d", totalItems)

	var withParentUUID int
	_ = db.QueryRow(`SELECT COUNT(*) FROM items WHERE parent_uuid IS NOT NULL AND parent_uuid != ''`).Scan(&withParentUUID)
	log.Printf("  С parent_uuid: %d", withParentUUID)

	var withParentID int
	_ = db.QueryRow(`SELECT COUNT(*) FROM items WHERE parent_id IS NOT NULL AND parent_id != 0`).Scan(&withParentID)
	log.Printf("  С parent_id (legacy): %d", withParentID)

	var pinnedWithUUID int
	_ = db.QueryRow(`SELECT COUNT(*) FROM pinned_items WHERE item_element_uuid IS NOT NULL AND item_element_uuid != ''`).Scan(&pinnedWithUUID)
	log.Printf("  Pinned с item_element_uuid: %d", pinnedWithUUID)

	log.Println("")
	log.Println("Миграция завершена успешно!")
	fmt.Println("")
	fmt.Println("Для запуска основного приложения:")
	fmt.Println("  go run cmd/main.go")
}
