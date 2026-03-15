package database

import (
	"log"
	"strings"
)

// RunMigrations выполняет миграции базы данных
// Порядок миграций важен! Сначала создаются основные таблицы, затем новые, затем перенос данных
func RunMigrations() {
	// 1. ТАБЛИЦА ЭЛЕМЕНТОВ (основная)
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id          INTEGER PRIMARY KEY,
			type        TEXT NOT NULL CHECK (type IN ('folder', 'element')),
			title       TEXT,
			description TEXT,
			content_meta TEXT,
			parent_id   INTEGER,
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

	// 6. ЗАКРЕПЛЁННЫЕ ЭЛЕМЕНТЫ В ПРОФИЛЕ
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pinned_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id INTEGER NOT NULL,
			order_num INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (item_id) REFERENCES items (id) ON DELETE CASCADE
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
			_ = err //nolint:staticcheck // Логируем ошибку, но не выводим в пользовательский интерфейс
		}
	}

	// Добавляем поле description, если оно не существует
	_, err = DB.Exec(`ALTER TABLE tags ADD COLUMN description TEXT DEFAULT ''`)
	// Игнорируем ошибку, если столбец уже существует
	if err != nil {
		_ = err //nolint:staticcheck // Логируем ошибку, но не выводим в пользовательский интерфейс
	}

	// Создаём новые таблицы для профилей и элементов
	createNewProfileTables()

	seedBootstrapPeers()
}

// createNewProfileTables создаёт новые таблицы для профилей и элементов
// Это новая схема с поддержкой множественных профилей (локальный + чужие)
func createNewProfileTables() {
	// 1. TABLE profiles - универсальная таблица для всех профилей
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
			demo_elements   TEXT,
			cached_at       DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("Ошибка при создании таблицы profiles: %v", err)
	}

	// Индексы для profiles
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_profiles_peer_id ON profiles(peer_id)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_profiles_peer_id: %v", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_profiles_owner_type ON profiles(owner_type)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_profiles_owner_type: %v", err)
	}

	// 2. TABLE profile_keys - криптографические ключи
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS profile_keys (
			profile_id      INTEGER PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
			private_key     BLOB,
			public_key      BLOB NOT NULL,
			signature       BLOB,
			is_key_encrypted BOOLEAN DEFAULT 0
		)
	`)
	if err != nil {
		log.Printf("Ошибка при создании таблицы profile_keys: %v", err)
	}

	// 3. TABLE remote_items - кэшированные чужие элементы
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS remote_items (
			id              INTEGER PRIMARY KEY,
			source_peer_id  TEXT NOT NULL REFERENCES profiles(peer_id),
			original_id     INTEGER NOT NULL,
			original_hash   TEXT NOT NULL,
			title           TEXT,
			description     TEXT,
			content_meta    TEXT,
			signature       BLOB,
			version         INTEGER DEFAULT 1,
			cached_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_peer_id, original_hash)
		)
	`)
	if err != nil {
		log.Printf("Ошибка при создании таблицы remote_items: %v", err)
	}

	// Индексы для remote_items
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_remote_items_source_peer ON remote_items(source_peer_id)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_remote_items_source_peer: %v", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_remote_items_hash ON remote_items(original_hash)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_remote_items_hash: %v", err)
	}

	// 4. TABLE item_files - файлы элементов
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS item_files (
			item_id         INTEGER NOT NULL,
			hash            TEXT NOT NULL,
			file_path       TEXT NOT NULL,
			size            INTEGER,
			mime_type       TEXT,
			is_remote       BOOLEAN DEFAULT 0,
			source_peer_id  TEXT,
			PRIMARY KEY (item_id, hash)
		)
	`)
	if err != nil {
		log.Printf("Ошибка при создании таблицы item_files: %v", err)
	}

	// Индекс для item_files
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_item_files_item_id ON item_files(item_id)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_item_files_item_id: %v", err)
	}

	// 5. Добавляем content_hash в items (если ещё нет)
	_, err = DB.Exec(`ALTER TABLE items ADD COLUMN content_hash TEXT`)
	if err != nil {
		// Игнорируем ошибку, если столбец уже существует
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении content_hash в items: %v", err)
		}
	}

	// Индекс для content_hash
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_items_content_hash ON items(content_hash)`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_items_content_hash: %v", err)
	}

	// 6. Таблица contacts - адресная книга (избранные пользователи)
	// Хранит только уникальные данные: адрес для подключения, заметки, настройки
	// Профиль пользователя (username, avatar, title) берётся из таблицы profiles
	_, err = DB.Exec(`
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
		log.Printf("Ошибка при создании таблицы contacts: %v", err)
	}

	// 7. Таблица chat_messages - история сообщений
	_, err = DB.Exec(`
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
		log.Printf("Ошибка при создании таблицы chat_messages: %v", err)
	}

	// Добавляем колонку updated_at если она не существует (для существующих БД)
	_, err = DB.Exec(`ALTER TABLE chat_messages ADD COLUMN updated_at DATETIME`)
	if err != nil {
		// Игнорируем ошибку, если столбец уже существует
		if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "column already exists") {
			log.Printf("Ошибка при добавлении updated_at в chat_messages: %v", err)
		}
	}

	// 8. Таблица bootstrap_peers - узлы для входа в сеть
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS bootstrap_peers (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			multiaddr     TEXT UNIQUE NOT NULL,
			peer_id       TEXT,
			is_active     BOOLEAN DEFAULT 1,
			last_connected DATETIME,
			added_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Printf("Ошибка при создании таблицы bootstrap_peers: %v", err)
	}

	// Индексы для производительности
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_contacts_peer_id ON contacts(peer_id);`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_contacts_peer_id: %v", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_contact_id ON chat_messages(contact_id);`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_chat_messages_contact_id: %v", err)
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_bootstrap_peers_multiaddr ON bootstrap_peers(multiaddr);`)
	if err != nil {
		log.Printf("Ошибка при создании индекса idx_bootstrap_peers_multiaddr: %v", err)
	}

	log.Println("Новые таблицы профилей и элементов созданы")
}

// seedBootstrapPeers добавляет предопределённые bootstrap-узлы
// Отключено - пользователь добавляет bootstrap пиры самостоятельно
func seedBootstrapPeers() {
	// Bootstrap пиры не добавляются по умолчанию
	// Пользователь может добавить их через настройки P2P в приложении
	log.Println("Bootstrap-узлы не добавлены (добавьте вручную через настройки)")
}
