package database

import (
	"log"
	"strings"
)

// RunMigrations выполняет миграции базы данных
func RunMigrations() {
	// 1. ТАБЛИЦА ЭЛЕМЕНТОВ (основная)
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id          INTEGER PRIMARY KEY,
			type        TEXT NOT NULL CHECK (type IN ('text', 'link', 'folder', 'composite', 'image', 'file')),
			title       TEXT,
			description TEXT,
			content_meta TEXT,
			parent_id   INTEGER,
			is_pinned   BOOLEAN DEFAULT 0,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES items (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы items:", err)
	}

	// 2. ТАБЛИЦА ФАЙЛОВ (для дедупликации)
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id          INTEGER PRIMARY KEY,
			hash        TEXT UNIQUE NOT NULL,
			size        INTEGER NOT NULL,
			mime_type   TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы files:", err)
	}

	// 3. ТЕГИ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS tags (
			id          INTEGER PRIMARY KEY,
			name        TEXT UNIQUE NOT NULL,
			color       TEXT DEFAULT '#FFBB00',
			description TEXT DEFAULT ''
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы tags:", err)
	}

	// 4. СВЯЗЬ ЭЛЕМЕНТОВ С ТЕГАМИ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS item_tags (
			item_id INTEGER,
			tag_id  INTEGER,
			PRIMARY KEY (item_id, tag_id),
			FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE CASCADE,
			FOREIGN KEY (tag_id)  REFERENCES tags (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы item_tags:", err)
	}

	// 5. ИЗБРАННОЕ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS favorites (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_type TEXT NOT NULL CHECK (entity_type IN ('tag', 'folder')),
			entity_id   INTEGER NOT NULL
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы favorites:", err)
	}

	// 6. ЗАКРЕПЛЁННЫЕ В ПАПКАХ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pinned_items (
			folder_id   INTEGER,
			item_id     INTEGER,
			order_index INTEGER DEFAULT 0,
			PRIMARY KEY (folder_id, item_id),
			FOREIGN KEY (folder_id) REFERENCES items (id) ON DELETE CASCADE,
			FOREIGN KEY (item_id)   REFERENCES items (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы pinned_items:", err)
	}

	// ИНДЕКСЫ для производительности
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_items_parent ON items(parent_id);`)
	if err != nil {
		log.Fatal("Ошибка при создании индекса idx_items_parent:", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_items_type ON items(type);`)
	if err != nil {
		log.Fatal("Ошибка при создании индекса idx_items_type:", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_items_updated ON items(updated_at DESC);`)
	if err != nil {
		log.Fatal("Ошибка при создании индекса idx_items_updated:", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);`)
	if err != nil {
		log.Fatal("Ошибка при создании индекса idx_files_hash:", err)
	}

	// Если таблица tags уже существует, добавляем поле color
	_, err = DB.Exec(`ALTER TABLE tags ADD COLUMN color TEXT DEFAULT '#FFBB00'`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Проверяем, возможно столбец уже существует
		if err.Error() != "duplicate column name: color" {
			// Логируем ошибку, но не выводим в пользовательский интерфейс
		}
	}

	// Добавляем поле description, если оно не существует
	_, err = DB.Exec(`ALTER TABLE tags ADD COLUMN description TEXT DEFAULT ''`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Логируем ошибку, но не выводим в пользовательский интерфейс
	}

	// 7. ТАБЛИЦА ПРОФИЛЯ ПОЛЬЗОВАТЕЛЯ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS profile (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL DEFAULT 'Аноним',
			status TEXT NOT NULL DEFAULT 'Доступен',
			avatar_path TEXT,
			background_path TEXT DEFAULT '',
			content_characteristic TEXT,
			demo_elements TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы profile:", err)
	}

	// Добавляем поле background_path, если оно не существует
	_, err = DB.Exec(`ALTER TABLE profile ADD COLUMN background_path TEXT DEFAULT ''`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Проверяем, возможно столбец уже существует
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении столбца background_path: %v", err)
		}
	}

	// Добавляем поле content_characteristic, если оно не существует
	_, err = DB.Exec(`ALTER TABLE profile ADD COLUMN content_characteristic TEXT`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Проверяем, возможно столбец уже существует
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении столбца content_characteristic: %v", err)
		}
	}

	// Добавляем поле demo_elements, если оно не существует
	_, err = DB.Exec(`ALTER TABLE profile ADD COLUMN demo_elements TEXT`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Проверяем, возможно столбец уже существует
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении столбца demo_elements: %v", err)
		}
	}

	// Добавляем индекс для производительности
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_profile_username ON profile(username);`)
	if err != nil {
		// Логируем ошибку, но не выводим в пользовательский интерфейс
	}

	// Добавляем поле is_pinned, если оно не существует
	_, err = DB.Exec(`ALTER TABLE items ADD COLUMN is_pinned BOOLEAN DEFAULT 0`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		// Проверяем, возможно столбец уже существует
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении столбца is_pinned: %v", err)
		}
	}

	// Логируем успешное выполнение миграций
	fixFolderConstraint()

	// Обновляем структуру таблицы favorites
	updateFavoritesTable()
}

// updateFavoritesTable обновляет структуру таблицы favorites
func updateFavoritesTable() {
	// Проверяем, существует ли старая структура таблицы favorites
	rows, err := DB.Query(`SELECT sql FROM sqlite_master WHERE type='table' AND name='favorites';`)
	if err != nil {
		log.Printf("Ошибка при проверке структуры таблицы favorites: %v", err)
		return
	}
	defer rows.Close()

	var tableSQL string
	if rows.Next() {
		err = rows.Scan(&tableSQL)
		if err != nil {
			log.Printf("Ошибка при чтении SQL-запроса таблицы favorites: %v", err)
			return
		}
	}

	// Если таблица имеет старую структуру (содержит item_id, category, added_at), пересоздаем её
	if strings.Contains(tableSQL, "item_id") && strings.Contains(tableSQL, "category") && strings.Contains(tableSQL, "added_at") {
		// Сохраняем старые данные избранного (только для папок)
		var oldFavorites []int
		rows, err := DB.Query(`SELECT item_id FROM favorites`)
		if err != nil {
			log.Printf("Ошибка при чтении старых данных избранного: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var itemID int
				err := rows.Scan(&itemID)
				if err != nil {
					continue
				}
				oldFavorites = append(oldFavorites, itemID)
			}
		}

		// Пересоздаем таблицу с новой структурой
		_, err = DB.Exec(`
			PRAGMA foreign_keys = off;

			DROP TABLE favorites;

			CREATE TABLE IF NOT EXISTS favorites (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				entity_type TEXT NOT NULL CHECK (entity_type IN ('tag', 'folder')),
				entity_id   INTEGER NOT NULL
			);

			PRAGMA foreign_keys = on;
		`)
		if err != nil {
			log.Printf("Ошибка при обновлении структуры таблицы favorites: %v", err)
		} else {
			log.Println("Успешно обновлена структура таблицы favorites")

			// Восстанавливаем старые данные избранного как избранное папок
			for _, itemID := range oldFavorites {
				// Проверяем, является ли элемент папкой
				var itemType string
				err := DB.QueryRow(`SELECT type FROM items WHERE id = ?`, itemID).Scan(&itemType)
				if err != nil || itemType != "folder" {
					continue // Пропускаем, если элемент не найден или не является папкой
				}

				// Добавляем папку в избранное
				_, err = DB.Exec(`INSERT INTO favorites (entity_type, entity_id) VALUES ('folder', ?)`, itemID)
				if err != nil {
					log.Printf("Ошибка при восстановлении избранного: %v", err)
				}
			}
		}
	}
}

// fixFolderConstraint обновляет таблицу items для разрешения вложенных папок
func fixFolderConstraint() {
	// Проверяем, существует ли старое ограничение, и пересоздаем таблицу
	rows, err := DB.Query(`SELECT sql FROM sqlite_master WHERE type='table' AND name='items';`)
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

	// Проверяем, содержит ли таблица старое ограничение
	if strings.Contains(tableSQL, "CHECK (NOT (type = 'folder' AND parent_id IS NOT NULL))") {
		// Пересоздаем таблицу без ограничения
		_, err = DB.Exec(`
			PRAGMA foreign_keys = off;

			CREATE TABLE items_new (
				id          INTEGER PRIMARY KEY,
				type        TEXT NOT NULL CHECK (type IN ('text', 'link', 'folder', 'composite', 'image', 'file')),
				title       TEXT NOT NULL,
				description TEXT,
				content_meta TEXT,
				parent_id   INTEGER,
				created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (parent_id) REFERENCES items (id) ON DELETE CASCADE
			);

			INSERT INTO items_new SELECT * FROM items;

			DROP TABLE items;
			ALTER TABLE items_new RENAME TO items;

			PRAGMA foreign_keys = on;
		`)
		if err != nil {
			log.Printf("Ошибка при обновлении структуры таблицы items: %v", err)
		} else {
			log.Println("Успешно обновлена структура таблицы items для разрешения вложенных папок")
		}
	}
}
