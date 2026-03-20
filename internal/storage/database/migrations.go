package database

import (
	"fmt"
	"log"
	"strings"
)

// RunMigrations выполняет миграции базы данных
// Структура:
// 1. Создание всех таблиц
// 2. Создание всех индексов
// 3. Миграции для существующих БД (ALTER TABLE)
// 4. Триггеры и ограничения
// 5. Seed данные
func RunMigrations() {
	// ============================================================
	// ЧАСТЬ 1: СОЗДАНИЕ ТАБЛИЦ
	// ============================================================

	createItemsTable()
	createFilesTable()
	createTagsTable()
	createItemTagsTable()
	createFavoritesTable()
	createPinnedItemsTable()
	createItemFilesTable()
	createProfilesTable()
	createProfileKeysTable()
	createContactsTable()
	createChatMessagesTable()
	createBootstrapPeersTable()

	// ============================================================
	// ЧАСТЬ 2: СОЗДАНИЕ ИНДЕКСОВ
	// ============================================================

	createItemsIndexes()
	createFilesIndexes()
	createTagsIndexes()
	createItemTagsIndexes()
	createItemFilesIndexes()
	createProfilesIndexes()
	createContactsIndexes()
	createChatMessagesIndexes()
	createBootstrapPeersIndexes()

	// ============================================================
	// ЧАСТЬ 3: МИГРАЦИИ ДЛЯ СУЩЕСТВУЮЩИХ БД (ALTER TABLE)
	// ============================================================

	migrateItemsTable()        // Добавление element_uuid и hash
	migrateTagsTable()         // Добавление P2P полей в tags
	migrateItemRelations()     // Добавление item_element_uuid в связи
	migrateChatMessagesTable() // Добавление updated_at
	// migratePinnedUUIDs() больше не нужна — поля уже созданы в createProfilesTable()

	// ============================================================
	// ЧАСТЬ 4: ТРИГГЕРЫ И ОГРАНИЧЕНИЯ
	// ============================================================

	createElementUUIDTrigger()

	// ============================================================
	// ЧАСТЬ 5: SEED ДАННЫЕ
	// ============================================================

	seedBootstrapPeers()

	log.Println("Все миграции базы данных выполнены успешно")
}

// ============================================================
// ЧАСТЬ 1: ФУНКЦИИ СОЗДАНИЯ ТАБЛИЦ
// ============================================================

func createItemsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id              INTEGER PRIMARY KEY,
			element_uuid    TEXT,
			hash            TEXT,
			owner_type      TEXT DEFAULT 'local' CHECK (owner_type IN ('local', 'remote')),
			source_peer_id  TEXT,
			type            TEXT NOT NULL CHECK (type IN ('folder', 'element')),
			title           TEXT,
			description     TEXT,
			content_meta    TEXT,
			parent_id       INTEGER,
			signature       BLOB,
			version         INTEGER DEFAULT 1,
			cached_at       DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES items (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы items:", err)
	}
}

func createFilesTable() {
	_, err := DB.Exec(`
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
}

func createTagsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS tags (
			id              INTEGER PRIMARY KEY,
			tag_uuid        TEXT,
			owner_peer_id   TEXT DEFAULT 'local',
			name            TEXT UNIQUE NOT NULL,
			color           TEXT DEFAULT '#FFBB00',
			description     TEXT DEFAULT '',
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы tags:", err)
	}
}

func createItemTagsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS item_tags (
			item_id             INTEGER,
			item_element_uuid   TEXT,
			tag_id              INTEGER,
			tag_uuid            TEXT,
			PRIMARY KEY (item_id, tag_id),
			FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE CASCADE,
			FOREIGN KEY (tag_id)  REFERENCES tags (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы item_tags:", err)
	}
}

func createFavoritesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS favorites (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_type TEXT NOT NULL CHECK (entity_type IN ('tag', 'folder')),
			entity_uuid TEXT NOT NULL
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы favorites:", err)
	}
}

func createPinnedItemsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS pinned_items (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id           INTEGER NOT NULL,
			item_element_uuid TEXT,
			order_num         INTEGER DEFAULT 0,
			created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы pinned_items:", err)
	}
}

func createItemFilesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS item_files (
			item_id             INTEGER NOT NULL,
			item_element_uuid   TEXT,
			hash                TEXT NOT NULL,
			file_path           TEXT NOT NULL,
			size                INTEGER,
			mime_type           TEXT,
			is_remote           BOOLEAN DEFAULT 0,
			source_peer_id      TEXT,
			PRIMARY KEY (item_id, hash)
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы item_files:", err)
	}
}

func createProfilesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS profiles (
			id              INTEGER PRIMARY KEY,
			owner_type      TEXT NOT NULL CHECK (owner_type IN ('local', 'remote')),
			peer_id         TEXT UNIQUE NOT NULL,
			username        TEXT NOT NULL,
			title           TEXT,
			avatar_path     TEXT,
			background_path TEXT DEFAULT '',
			content_char    TEXT,
			pinned_uuids    TEXT DEFAULT '[]',
			cached_at       DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы profiles:", err)
	}
}

func createProfileKeysTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS profile_keys (
			profile_id       INTEGER PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
			private_key      BLOB,
			public_key       BLOB NOT NULL,
			signature        BLOB,
			is_key_encrypted BOOLEAN DEFAULT 0
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы profile_keys:", err)
	}
}

func createContactsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS contacts (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			peer_id     TEXT UNIQUE NOT NULL REFERENCES profiles(peer_id),
			multiaddr   TEXT,
			notes       TEXT,
			is_blocked  BOOLEAN DEFAULT 0,
			is_favorite BOOLEAN DEFAULT 1,
			last_seen   DATETIME,
			added_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы contacts:", err)
	}
}

func createChatMessagesTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS chat_messages (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			contact_id   INTEGER NOT NULL,
			from_peer_id TEXT NOT NULL,
			content      TEXT NOT NULL,
			content_type TEXT DEFAULT 'text',
			metadata     TEXT,
			is_read      BOOLEAN DEFAULT 0,
			sent_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at   DATETIME,
			FOREIGN KEY (contact_id) REFERENCES contacts (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы chat_messages:", err)
	}
}

func createBootstrapPeersTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS bootstrap_peers (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			multiaddr      TEXT UNIQUE NOT NULL,
			peer_id        TEXT,
			is_active      BOOLEAN DEFAULT 1,
			last_connected DATETIME,
			added_at       DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы bootstrap_peers:", err)
	}
}

// ============================================================
// ЧАСТЬ 2: ФУНКЦИИ СОЗДАНИЯ ИНДЕКСОВ
// ============================================================

func createItemsIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_items_parent ON items(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_type ON items(type)`,
		`CREATE INDEX IF NOT EXISTS idx_items_updated ON items(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_items_owner_type ON items(owner_type)`,
		`CREATE INDEX IF NOT EXISTS idx_items_source_peer ON items(source_peer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_element_uuid ON items(element_uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_items_hash ON items(hash)`,
	}

	for _, sql := range indexes {
		if _, err := DB.Exec(sql); err != nil {
			log.Printf("Ошибка при создании индекса: %v", err)
		}
	}
}

func createFilesIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_files_hash: %v", err)
	}
}

func createTagsIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_tags_tag_uuid ON tags(tag_uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_owner_peer_id ON tags(owner_peer_id)`,
	}

	for _, sql := range indexes {
		if _, err := DB.Exec(sql); err != nil {
			log.Printf("Ошибка при создании индекса: %v", err)
		}
	}
}

func createItemTagsIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_tags_element_uuid ON item_tags(item_element_uuid)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_item_tags_element_uuid: %v", err)
	}
}

func createItemFilesIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_files_item_id ON item_files(item_id)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_item_files_item_id: %v", err)
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_files_element_uuid ON item_files(item_element_uuid)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_item_files_element_uuid: %v", err)
	}
}

func createProfilesIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_profiles_peer_id ON profiles(peer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_profiles_owner_type ON profiles(owner_type)`,
	}

	for _, sql := range indexes {
		if _, err := DB.Exec(sql); err != nil {
			log.Printf("Ошибка при создании индекса: %v", err)
		}
	}
}

func createContactsIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_contacts_peer_id: %v", err)
	}
}

func createChatMessagesIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_contact_id ON chat_messages(contact_id)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_chat_messages_contact_id: %v", err)
	}
}

func createBootstrapPeersIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_bootstrap_peers_multiaddr ON bootstrap_peers(multiaddr)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_bootstrap_peers_multiaddr: %v", err)
	}
}

// ============================================================
// ЧАСТЬ 3: ФУНКЦИИ МИГРАЦИИ (ALTER TABLE)
// ============================================================

// migrateItemsTable добавляет новые колонки element_uuid и hash в таблицу items
func migrateItemsTable() {
	// Добавляем колонку element_uuid если не существует
	_, err := DB.Exec(`ALTER TABLE items ADD COLUMN element_uuid TEXT`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении element_uuid в items: %v", err)
		}
	}

	// Добавляем колонку hash если не существует (новое имя для content_hash)
	_, err = DB.Exec(`ALTER TABLE items ADD COLUMN hash TEXT`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении hash в items: %v", err)
		}
	}

	// Копируем данные из content_hash в hash для существующих записей
	_, err = DB.Exec(`UPDATE items SET hash = content_hash WHERE hash IS NULL AND content_hash IS NOT NULL`)
	if err != nil && !strings.Contains(err.Error(), "no such column") {
		log.Printf("Ошибка при копировании content_hash в hash: %v", err)
	}

	// Генерируем element_uuid для существующих записей без UUID
	rows, err := DB.Query(`SELECT id FROM items WHERE element_uuid IS NULL`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err == nil {
				uuid := generateCompatibilityUUID(id)
				_, _ = DB.Exec(`UPDATE items SET element_uuid = ? WHERE id = ?`, uuid, id)
			}
		}
	}

	log.Println("Миграция items table: добавлены element_uuid и hash")
}

// migrateTagsTable добавляет поддержку P2P для тегов
func migrateTagsTable() {
	// Добавляем колонку tag_uuid если не существует
	_, err := DB.Exec(`ALTER TABLE tags ADD COLUMN tag_uuid TEXT`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении tag_uuid в tags: %v", err)
		}
	}

	// Добавляем колонку owner_peer_id если не существует
	_, err = DB.Exec(`ALTER TABLE tags ADD COLUMN owner_peer_id TEXT`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении owner_peer_id в tags: %v", err)
		}
	}

	// Генерируем tag_uuid для существующих тегов
	rows, err := DB.Query(`SELECT id FROM tags WHERE tag_uuid IS NULL`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err == nil {
				uuid := generateCompatibilityUUID(id)
				_, _ = DB.Exec(`UPDATE tags SET tag_uuid = ?, owner_peer_id = 'local' WHERE id = ?`, uuid, id)
			}
		}
	}

	log.Println("Миграция tags: добавлена поддержка P2P")
}

// migrateItemRelations добавляет item_element_uuid в таблицы связей
func migrateItemRelations() {
	// Добавляем item_element_uuid в item_tags
	_, err := DB.Exec(`ALTER TABLE item_tags ADD COLUMN item_element_uuid TEXT`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении item_element_uuid в item_tags: %v", err)
		}
	}

	// Копируем данные из item_id в item_element_uuid
	_, err = DB.Exec(`
		UPDATE item_tags 
		SET item_element_uuid = (SELECT element_uuid FROM items WHERE items.id = item_tags.item_id)
		WHERE item_element_uuid IS NULL
	`)
	if err != nil {
		log.Printf("Ошибка при копировании item_id в item_element_uuid: %v", err)
	}

	// Добавляем item_element_uuid в item_files
	_, err = DB.Exec(`ALTER TABLE item_files ADD COLUMN item_element_uuid TEXT`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении item_element_uuid в item_files: %v", err)
		}
	}

	// Копируем данные из item_id в item_element_uuid в item_files
	_, err = DB.Exec(`
		UPDATE item_files 
		SET item_element_uuid = (SELECT element_uuid FROM items WHERE items.id = item_files.item_id)
		WHERE item_element_uuid IS NULL
	`)
	if err != nil {
		log.Printf("Ошибка при копировании item_id в item_element_uuid в item_files: %v", err)
	}

	// Добавляем item_element_uuid в pinned_items
	_, err = DB.Exec(`ALTER TABLE pinned_items ADD COLUMN item_element_uuid TEXT`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении item_element_uuid в pinned_items: %v", err)
		}
	}

	// Копируем данные из item_id в item_element_uuid в pinned_items
	_, err = DB.Exec(`
		UPDATE pinned_items
		SET item_element_uuid = (SELECT element_uuid FROM items WHERE items.id = pinned_items.item_id)
		WHERE item_element_uuid IS NULL
	`)
	if err != nil {
		log.Printf("Ошибка при копировании item_id в item_element_uuid в pinned_items: %v", err)
	}

	// ============================================================
	// Миграция favorites: замена entity_id на entity_uuid
	// ============================================================

	// Добавляем колонку entity_uuid
	_, err = DB.Exec(`ALTER TABLE favorites ADD COLUMN entity_uuid TEXT`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении entity_uuid в favorites: %v", err)
		}
	}

	// Копируем данные из entity_id в entity_uuid
	// Для tag: берём tag_uuid из tags
	// Для folder: берём element_uuid из items (type='folder')
	_, err = DB.Exec(`
		UPDATE favorites
		SET entity_uuid = (
			SELECT tag_uuid FROM tags WHERE tags.id = favorites.entity_id
		)
		WHERE entity_type = 'tag' AND entity_uuid IS NULL
	`)
	if err != nil && !strings.Contains(err.Error(), "no such column") {
		log.Printf("Ошибка при копировании entity_id в entity_uuid для tags: %v", err)
	}

	_, err = DB.Exec(`
		UPDATE favorites
		SET entity_uuid = (
			SELECT element_uuid FROM items WHERE items.id = favorites.entity_id AND items.type = 'folder'
		)
		WHERE entity_type = 'folder' AND entity_uuid IS NULL
	`)
	if err != nil && !strings.Contains(err.Error(), "no such column") {
		log.Printf("Ошибка при копировании entity_id в entity_uuid для folders: %v", err)
	}

	// Делаем entity_uuid NOT NULL после заполнения
	// Создаём триггер для валидации
	_, err = DB.Exec(`
		CREATE TRIGGER IF NOT EXISTS validate_favorites_entity_uuid_insert
		BEFORE INSERT ON favorites
		FOR EACH ROW
		WHEN NEW.entity_uuid IS NULL
		BEGIN
			SELECT RAISE(ABORT, 'entity_uuid cannot be NULL');
		END
	`)
	if err != nil {
		log.Printf("Ошибка при создании триггера validate_favorites_entity_uuid_insert: %v", err)
	}

	// Создаём индекс для entity_uuid
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_favorites_entity_uuid ON favorites(entity_uuid)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_favorites_entity_uuid: %v", err)
	}

	log.Println("Миграция favorites: entity_id заменён на entity_uuid")
}

// migrateChatMessagesTable добавляет updated_at в chat_messages
func migrateChatMessagesTable() {
	_, err := DB.Exec(`ALTER TABLE chat_messages ADD COLUMN updated_at DATETIME`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении updated_at в chat_messages: %v", err)
		}
	}
}

// migratePinnedUUIDs и removeDemoElements больше не вызываются автоматически
// Поля создаются в createProfilesTable() при инициализации БД
// Заполнение pinned_uuids выполняется отдельным скриптом при необходимости
/*
func migratePinnedUUIDs() {
	log.Println("Миграция pinned_uuids: начало...")

	// Проверяем, существует ли поле pinned_uuids
	var hasPinnedUUIDsField bool
	err := DB.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('profiles')
		WHERE name = 'pinned_uuids'
	`).Scan(&hasPinnedUUIDsField)

	if err != nil {
		log.Printf("Ошибка проверки поля pinned_uuids: %v", err)
		return
	}

	if !hasPinnedUUIDsField {
		// Добавляем поле pinned_uuids
		_, err = DB.Exec(`ALTER TABLE profiles ADD COLUMN pinned_uuids TEXT DEFAULT '[]'`)
		if err != nil {
			log.Printf("Ошибка добавления поля pinned_uuids: %v", err)
			return
		}
		log.Println("Поле pinned_uuids добавлено")
	} else {
		log.Println("Поле pinned_uuids уже существует")
	}

	// Удаляем поле demo_elements если существует
	removeDemoElements()

	log.Println("Миграция pinned_uuids завершена")
}

// removeDemoElements удаляет поле demo_elements из таблицы profiles
func removeDemoElements() {
	// Проверяем, существует ли поле demo_elements
	var hasDemoElementsField bool
	err := DB.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('profiles')
		WHERE name = 'demo_elements'
	`).Scan(&hasDemoElementsField)

	if err != nil {
		log.Printf("Ошибка проверки поля demo_elements: %v", err)
		return
	}

	if !hasDemoElementsField {
		log.Println("Поле demo_elements уже удалено")
		return
	}

	log.Println("Миграция pinned_uuids: удаление поля demo_elements...")

	// SQLite не поддерживает DROP COLUMN напрямую до версии 3.35.0
	// Используем пересоздание таблицы без этого поля

	// 1. Переименовываем таблицу
	_, err = DB.Exec(`ALTER TABLE profiles RENAME TO profiles_old`)
	if err != nil {
		log.Printf("Ошибка переименования таблицы: %v", err)
		return
	}
	log.Println("Таблица profiles переименована в profiles_old")

	// 2. Создаём новую таблицу без demo_elements
	_, err = DB.Exec(`
		CREATE TABLE profiles (
			id              INTEGER PRIMARY KEY,
			owner_type      TEXT NOT NULL CHECK (owner_type IN ('local', 'remote')),
			peer_id         TEXT UNIQUE NOT NULL,
			username        TEXT NOT NULL,
			title           TEXT,
			avatar_path     TEXT,
			background_path TEXT DEFAULT '',
			content_char    TEXT,
			pinned_uuids    TEXT DEFAULT '[]',
			cached_at       DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Printf("Ошибка создания новой таблицы profiles: %v", err)
		return
	}
	log.Println("Создана новая таблица profiles без demo_elements")

	// 3. Копируем данные из старой таблицы в новую
	_, err = DB.Exec(`
		INSERT INTO profiles (
			id, owner_type, peer_id, username, title, avatar_path,
			background_path, content_char, pinned_uuids, cached_at,
			created_at, updated_at
		)
		SELECT
			id, owner_type, peer_id, username, title, avatar_path,
			background_path, content_char,
			COALESCE(pinned_uuids, '[]'), cached_at,
			created_at, updated_at
		FROM profiles_old
	`)
	if err != nil {
		log.Printf("Ошибка копирования данных: %v", err)
		return
	}
	log.Println("Данные скопированы из profiles_old в profiles")

	// 4. Удаляем старую таблицу
	_, err = DB.Exec(`DROP TABLE profiles_old`)
	if err != nil {
		log.Printf("Ошибка удаления старой таблицы: %v", err)
		return
	}
	log.Println("Старая таблица profiles_old удалена")
}
*/

// ============================================================
// ЧАСТЬ 4: ТРИГГЕРЫ И ОГРАНИЧЕНИЯ
// ============================================================

func createElementUUIDTrigger() {
	_, err := DB.Exec(`
		CREATE TRIGGER IF NOT EXISTS validate_element_uuid_insert
		BEFORE INSERT ON items
		FOR EACH ROW
		WHEN NEW.element_uuid IS NULL
		BEGIN
			SELECT RAISE(ABORT, 'element_uuid cannot be NULL');
		END
	`)
	if err != nil {
		log.Printf("Ошибка при создании триггера validate_element_uuid_insert: %v", err)
	}
}

// ============================================================
// ЧАСТЬ 5: ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ============================================================

// generateCompatibilityUUID генерирует детерминированный UUID на основе ID
// Используется только для миграции существующих записей
// Формат: 00000001-0000-0000-0000-000000000001 где число = id
func generateCompatibilityUUID(id int) string {
	return fmt.Sprintf("%08d-0000-0000-0000-%012d", id, id)
}

// seedBootstrapPeers добавляет предопределённые bootstrap-узлы
// Отключено - пользователь добавляет bootstrap пиры самостоятельно
func seedBootstrapPeers() {
	log.Println("Bootstrap-узлы не добавлены (добавьте вручную через настройки P2P)")
}
