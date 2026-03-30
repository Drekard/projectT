package database

import (
	"log"
)

// RunMigrations выполняет миграции базы данных
// Структура:
// 1. Создание всех таблиц
// 2. Создание всех индексов
// 3. Триггеры и ограничения
// 4. Seed данные
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
	createChatsTable()
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
	createChatsIndexes()
	createChatMessagesIndexes()
	createBootstrapPeersIndexes()

	// ============================================================
	// ЧАСТЬ 3: ТРИГГЕРЫ И ОГРАНИЧЕНИЯ
	// ============================================================

	createElementUUIDTrigger()

	// ============================================================
	// ЧАСТЬ 4: Миграции существующих таблиц
	// ============================================================

	migrateItemsAddStatusColumn()
	createRemoteItemUniqueIndex()

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
			status          TEXT DEFAULT 'saved' CHECK (status IN ('saved', 'preview', 'archived')),
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
			chat_id      INTEGER NOT NULL,
			from_peer_id TEXT NOT NULL,
			content      TEXT NOT NULL,
			content_type TEXT DEFAULT 'text',
			metadata     TEXT,
			is_read      BOOLEAN DEFAULT 0,
			sent_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at   DATETIME,
			FOREIGN KEY (chat_id) REFERENCES chats (id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы chat_messages:", err)
	}
}

func createChatsTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS chats (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			contact_id      INTEGER,
			peer_id         TEXT NOT NULL,
			is_temporary    BOOLEAN DEFAULT 0,
			last_message_at DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (contact_id) REFERENCES contacts (id) ON DELETE SET NULL,
			UNIQUE (peer_id)
		);
	`)
	if err != nil {
		log.Fatal("Ошибка при создании таблицы chats:", err)
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
		// UNIQUE индекс для предотвращения дубликатов remote элементов
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_items_remote_unique ON items(source_peer_id, element_uuid)`,
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
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_chat_id ON chat_messages(chat_id)`)
	if err != nil {
		// Индекс не создаётся, если колонка chat_id не существует (старые БД)
		log.Printf("Предупреждение: не удалось создать индекс chat_messages.chat_id: %v", err)
	}
}

func createChatsIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_chats_contact_id ON chats(contact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chats_peer_id ON chats(peer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chats_last_message ON chats(last_message_at DESC)`,
	}
	for _, sql := range indexes {
		_, err := DB.Exec(sql)
		if err != nil {
			log.Printf("Ошибка при создании индекса: %v", err)
		}
	}
}

func createBootstrapPeersIndexes() {
	_, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_bootstrap_peers_multiaddr ON bootstrap_peers(multiaddr)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_bootstrap_peers_multiaddr: %v", err)
	}
}

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
// ЧАСТЬ 4.5: МИГРАЦИИ СУЩЕСТВУЮЩИХ ТАБЛИЦ
// ============================================================

// migrateItemsAddStatusColumn добавляет колонку status в таблицу items
// Для существующих элементов устанавливается status = 'saved'
func migrateItemsAddStatusColumn() {
	// Проверяем, существует ли уже колонка
	var count int
	err := DB.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('items') WHERE name = 'status'
	`).Scan(&count)

	if err != nil {
		log.Printf("Ошибка проверки колонки status: %v", err)
		return
	}

	if count > 0 {
		log.Println("[Миграция] Колонка status уже существует в таблице items")
		return
	}

	// Добавляем колонку
	_, err = DB.Exec(`
		ALTER TABLE items ADD COLUMN status TEXT DEFAULT 'saved' CHECK (status IN ('saved', 'preview', 'archived'))
	`)
	if err != nil {
		log.Printf("Ошибка добавления колонки status: %v", err)
		return
	}

	log.Println("[Миграция] Колонка status добавлена в таблицу items")
}

// createRemoteItemUniqueIndex создаёт UNIQUE индекс для remote элементов
// Это необходимо для работы ON CONFLICT в CreateRemoteItem
func createRemoteItemUniqueIndex() {
	_, err := DB.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_items_remote_unique 
		ON items(source_peer_id, element_uuid)
	`)
	if err != nil {
		log.Printf("[Миграция] Ошибка создания UNIQUE индекса: %v", err)
	} else {
		log.Println("[Миграция] UNIQUE индекс для remote элементов создан")
	}
}

// ============================================================
// ЧАСТЬ 5: ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ============================================================

// seedBootstrapPeers добавляет предопределённые bootstrap-узлы
// Отключено - пользователь добавляет bootstrap пиры самостоятельно
func seedBootstrapPeers() {
}
